"""Validate an isolated Docker stack; never use the application's existing volume."""
import json
import os
from pathlib import Path
import secrets
import select
import shutil
import socket
import subprocess
import tempfile
import time
import urllib.error
import urllib.request

ROOT = Path(__file__).resolve().parents[1]

def free_port():
    with socket.socket() as sock:
        sock.bind(('127.0.0.1', 0))
        return sock.getsockname()[1]

def main():
    docker = shutil.which('docker')
    if not docker:
        raise RuntimeError('docker must be on PATH and its daemon must be running')
    project = f'crashlens-smoke-{os.getpid()}'
    api_port, frontend_port = free_port(), free_port()
    while frontend_port == api_port:
        frontend_port = free_port()
    base = f'http://localhost:{api_port}'
    frontend = f'http://localhost:{frontend_port}'
    key = secrets.token_hex(32)
    with tempfile.TemporaryDirectory(prefix='crashlens-smoke-') as temp:
        envfile = Path(temp) / '.env'
        envfile.write_text('\n'.join([
            f'CRASHLENS_API_KEY={key}', 'ACCESS_MODE=private', 'APP_ENV=production',
            'FIREWORKS_API_KEY=', f'BACKEND_PORT={api_port}', f'FRONTEND_PORT={frontend_port}',
            f'NEXT_PUBLIC_API_URL={base}', f'ALLOWED_ORIGINS={frontend}',
            'JOB_CONCURRENCY=1', 'JOB_QUEUE_SIZE=2', 'JOB_TIMEOUT_SECONDS=5',
        ]) + '\n')
        command = [docker, 'compose', '--project-directory', str(ROOT), '--env-file', str(envfile), '-p', project]
        def compose(*args):
            result = subprocess.run(command + list(args), stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
            if result.returncode:
                raise RuntimeError(f'Compose {args} failed:\n{result.stdout[-3000:]}')
        def request(path, method='GET', data=None, authenticated=True):
            headers = {'Content-Type': 'application/json'}
            if authenticated:
                headers['Authorization'] = f'Bearer {key}'
            req = urllib.request.Request(base + path, method=method, headers=headers,
                                         data=json.dumps(data).encode() if data is not None else None)
            try:
                response = urllib.request.urlopen(req, timeout=10)
            except urllib.error.HTTPError as error:
                response = error
            with response:
                raw = response.read()
                try:
                    body = json.loads(raw) if raw else None
                except ValueError:
                    body = raw.decode()
                return response.status, body
        def wait_for(predicate, seconds=20):
            deadline = time.monotonic() + seconds
            while time.monotonic() < deadline:
                if predicate():
                    return
                time.sleep(.1)
            raise AssertionError('Timed out waiting for expected state')
        def workload(ident):
            status, body = request(f'/workloads/{ident}')
            assert status == 200, (status, body)
            return body
        def run(template):
            status, body = request('/workloads/run', 'POST', {'template': template})
            assert status == 202, (status, body)
            return body['workload_id']
        try:
            print('Building isolated backend, frontend, and MCP images...', flush=True)
            compose('--profile', 'mcp', 'build')
            compose('up', '-d', '--wait')
            assert request('/health', authenticated=False)[0] == 200
            for path in ['/workloads', '/summary', '/workloads/1/logs', '/workloads/1/metrics']:
                assert request(path, authenticated=False)[0] == 401, path
            assert request('/workloads/run', 'POST', {'template': 'gpu_oom'}, authenticated=False)[0] == 401
            assert request('/session')[0] == 200
            with urllib.request.urlopen(frontend + '/dashboard', timeout=10) as response:
                assert response.status == 200
            print('PASS: healthy containers and protected reads/writes', flush=True)
            first = run('timeout')
            wait_for(lambda: workload(first)['status'] == 'running')
            second, third = run('dependency_error'), run('dependency_error')
            assert request('/workloads/run', 'POST', {'template': 'successful'})[0] == 503
            assert workload(second)['status'] == 'pending'
            wait_for(lambda: workload(first)['status'] == 'running' and
                     workload(first).get('job_logs') and json.loads(workload(first).get('gpu_metrics') or '[]'), 4)
            print('PASS: live logs/metrics before completion and bounded queue', flush=True)
            wait_for(lambda: workload(first)['status'] == 'failed')
            assert workload(first)['failure_type'] == 'TIMEOUT'
            assert workload(first)['runtime_seconds'] < 9
            wait_for(lambda: workload(third)['status'] == 'failed')
            assert workload(second)['failure_type'] == 'DEPENDENCY_ERROR'
            status, report = request(f'/workloads/{first}/diagnose', 'POST')
            assert status == 200 and report['source'] == 'rules', (status, report)
            print('PASS: execution deadline and persisted diagnosis', flush=True)
            # The stdio container must work with a real MCP initialization and tool request.
            stderr = open(Path(temp) / 'mcp.log', 'w+')
            proc = subprocess.Popen(command + ['run', '--rm', '--no-deps', '-T', 'mcp-server'],
                                    stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=stderr, text=True, bufsize=1)
            def rpc(ident, method, params):
                proc.stdin.write(json.dumps({'jsonrpc': '2.0', 'id': ident, 'method': method, 'params': params}) + '\n')
                proc.stdin.flush()
                deadline = time.monotonic() + 15
                while time.monotonic() < deadline:
                    if not select.select([proc.stdout], [], [], 1)[0]:
                        continue
                    line = proc.stdout.readline()
                    if not line:
                        raise RuntimeError('MCP container exited before replying')
                    result = json.loads(line)
                    if result.get('id') == ident:
                        assert 'error' not in result, result
                        return result['result']
                raise RuntimeError('MCP response timeout')
            try:
                rpc(1, 'initialize', {'protocolVersion': '2024-11-05', 'capabilities': {}, 'clientInfo': {'name': 'crashlens-smoke', 'version': '1'}})
                proc.stdin.write(json.dumps({'jsonrpc': '2.0', 'method': 'notifications/initialized'}) + '\n'); proc.stdin.flush()
                assert len(rpc(2, 'tools/list', {})['tools']) == 7
                summary = rpc(3, 'tools/call', {'name': 'get_workload_summary', 'arguments': {'workload_id': first}})
                assert not summary.get('isError'), summary
            finally:
                proc.stdin.close()
                try:
                    proc.wait(timeout=5)
                except subprocess.TimeoutExpired:
                    proc.terminate(); proc.wait(timeout=5)
                stderr.close()
            print('PASS: MCP container handshake and authenticated tool call', flush=True)
            # Abrupt restart loses local processes but must not fail an external SDK job.
            external_status, external = request('/workloads', 'POST', {'name': 'external SDK', 'status': 'running'})
            assert external_status == 201
            active = run('timeout')
            wait_for(lambda: workload(active)['status'] == 'running')
            pending = run('dependency_error')
            compose('kill', '--signal', 'SIGKILL', 'backend')
            compose('up', '-d', '--wait', 'backend')
            assert workload(active)['status'] == 'failed'
            assert workload(pending)['status'] == 'failed'
            assert 'backend restarted' in workload(active)['job_logs']
            assert workload(external['id'])['status'] == 'running'
            assert workload(first)['failure_report']
            compose('up', '-d', '--force-recreate', '--wait', 'backend')
            assert workload(first)['failure_report']
            print('PASS: restart recovery, external SDK preservation, and volume persistence', flush=True)
            assert request('/workloads/clear', 'DELETE', authenticated=False)[0] == 401
            assert request('/workloads/clear', 'DELETE')[0] == 200
            assert request('/workloads')[1] == []
            print('PASS: authenticated clear operation', flush=True)
        finally:
            compose('--profile', 'mcp', 'down', '--volumes', '--remove-orphans')
    print('PASS: isolated Docker validation complete; test containers and volume removed')

if __name__ == '__main__':
    main()
