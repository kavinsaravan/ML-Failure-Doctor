"""Validate an isolated Docker stack; never use the application's existing volume."""
import json
import os
from pathlib import Path
import secrets
import shutil
import socket
import subprocess
import tempfile
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
        def workload(ident):
            status, body = request(f'/workloads/{ident}')
            assert status == 200, (status, body)
            return body
        try:
            print('Building isolated backend and frontend images...', flush=True)
            compose('build')
            compose('up', '-d', '--wait')
            assert request('/health', authenticated=False)[0] == 200
            for path in ['/workloads', '/summary', '/workloads/1/logs', '/workloads/1/metrics']:
                assert request(path, authenticated=False)[0] == 401, path
            assert request('/workloads', 'POST', {'name': 'unauthenticated'}, authenticated=False)[0] == 401
            assert request('/session')[0] == 200
            with urllib.request.urlopen(frontend + '/dashboard', timeout=10) as response:
                assert response.status == 200
            print('PASS: healthy containers and protected reads/writes', flush=True)
            status, created = request('/workloads', 'POST', {'name': 'SDK smoke test', 'status': 'running'})
            assert status == 201, (status, created)
            first = created['id']
            # Exercise the SDK upload contract; these fixtures are not GPU measurements.
            logs = 'ModuleNotFoundError: No module named missing_training_package'
            samples = json.dumps([{
                'source': 'smoke-test-fixture', 'gpu_memory_used_mb': 128,
                'gpu_memory_total_mb': 1024, 'gpu_memory_percent': 12.5,
            }])
            status, body = request(f'/workloads/{first}', 'PUT', {
                'job_logs': logs, 'gpu_metrics': samples, 'runtime_seconds': 2,
            })
            assert status == 200, (status, body)
            assert workload(first)['status'] == 'running'
            assert request(f'/workloads/{first}/logs')[1]['logs'] == logs
            assert json.loads(request(f'/workloads/{first}/metrics')[1]['metrics']) == json.loads(samples)
            print('PASS: SDK registration and live telemetry uploads', flush=True)
            status, body = request(f'/workloads/{first}', 'PUT', {
                'status': 'failed', 'exit_code': 1, 'runtime_seconds': 3,
            })
            assert status == 200 and body['failure_type'] == 'DEPENDENCY_ERROR', (status, body)
            status, report = request(f'/workloads/{first}/diagnose', 'POST')
            assert status == 200 and report['source'] == 'rules', (status, report)
            assert workload(first)['failure_report']
            print('PASS: failure classification and persisted diagnosis', flush=True)
            external_status, external = request('/workloads', 'POST', {'name': 'external SDK', 'status': 'running'})
            assert external_status == 201
            compose('kill', '--signal', 'SIGKILL', 'backend')
            compose('up', '-d', '--wait', 'backend')
            assert workload(external['id'])['status'] == 'running'
            assert workload(first)['failure_report']
            compose('up', '-d', '--force-recreate', '--wait', 'backend')
            assert workload(first)['failure_report']
            print('PASS: external SDK preservation and volume persistence', flush=True)
            assert request('/workloads/clear', 'DELETE', authenticated=False)[0] == 401
            assert request('/workloads/clear', 'DELETE')[0] == 200
            assert request('/workloads')[1] == []
            print('PASS: authenticated clear operation', flush=True)
        finally:
            compose('down', '--volumes', '--remove-orphans')
    print('PASS: isolated Docker validation complete; test containers and volume removed')

if __name__ == '__main__':
    main()
