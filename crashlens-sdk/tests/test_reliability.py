import json
from pathlib import Path
import tempfile
import threading
import unittest
from unittest.mock import Mock, patch

import requests
from crashlens.workload_tracker import WorkloadTracker
from crashlens.jupyter import JupyterWorkloadTracker


class ReliabilityTests(unittest.TestCase):
    def setUp(self):
        directory = tempfile.TemporaryDirectory()
        self.directory = Path(directory.name)
        self.addCleanup(directory.cleanup)
        self.post = patch('crashlens.workload_tracker.requests.post').start()
        self.post.return_value.json.return_value = {'id': 7}
        self.put = patch('crashlens.workload_tracker.requests.put').start()
        patch('crashlens.workload_tracker.METRICS_AVAILABLE', False).start()
        patch('crashlens.workload_tracker.time.sleep').start()
        self.addCleanup(patch.stopall)

    def tracker(self, **kwargs):
        return WorkloadTracker('http://test', api_key='workspace-secret',
                               fireworks_api_key='fireworks-secret',
                               outbox_dir=self.directory, **kwargs)

    def pending(self):
        return list(self.directory.rglob('*.json'))

    def test_startup_outage_does_not_prevent_training_and_final_registration_recovers(self):
        self.post.side_effect = [requests.ConnectionError('offline'), self.post.return_value]
        tracker = self.tracker()
        with self.assertLogs('crashlens.workload_tracker', level='WARNING'):
            with tracker.track('Offline startup') as ident:
                self.assertIsNone(ident)
                print('Training still runs')
        self.assertEqual(tracker.last_workload_id, 7)
        first, second = self.post.call_args_list
        self.assertEqual(first.kwargs['json']['client_run_id'], second.kwargs['json']['client_run_id'])
        payload = self.put.call_args.kwargs['json']
        self.assertEqual(payload['status'], 'succeeded')
        self.assertIn('Training still runs', payload['job_logs'])
        self.assertIn('started_at', payload)
        self.assertIn('finished_at', payload)
        self.assertFalse(self.pending())
        self.post.side_effect = None
        tracker.diagnose(ident)
        self.assertTrue(self.post.call_args.args[0].endswith('/workloads/7/diagnose'))

    def test_registration_recovers_while_training_is_running(self):
        tracker = self.tracker(upload_interval=.01)
        self.post.side_effect = [requests.Timeout('response lost'), self.post.return_value]
        live = threading.Event()

        def put(*args, **kwargs):
            if kwargs['json']['status'] == 'running':
                live.set()
            return Mock()

        self.put.side_effect = put
        with self.assertLogs('crashlens.workload_tracker', level='WARNING'):
            with tracker.track('Live recovery'):
                self.assertTrue(live.wait(2))
                self.assertEqual(tracker.last_workload_id, 7)
        self.assertEqual(self.put.call_args.kwargs['json']['status'], 'succeeded')

    def test_terminal_upload_retries_transient_errors(self):
        response = requests.Response()
        response.status_code = 503
        self.put.return_value.raise_for_status.side_effect = [
            requests.Timeout(), requests.HTTPError(response=response), None]
        with self.tracker().track('Retry final'):
            pass
        self.assertEqual(self.put.call_count, 3)
        self.assertFalse(self.pending())
        ids = [call.args[0] for call in self.put.call_args_list]
        self.assertEqual(len(set(ids)), 1)

    def test_auth_errors_are_not_retried_and_pending_data_contains_no_credentials(self):
        response = requests.Response()
        response.status_code = 401
        self.put.return_value.raise_for_status.side_effect = requests.HTTPError(response=response)
        with self.assertLogs('crashlens.workload_tracker', level='WARNING'):
            with self.tracker().track('Unauthorized upload'):
                print('Captured output')
        self.assertEqual(self.put.call_count, 1)
        files = self.pending()
        self.assertEqual(len(files), 1)
        text = files[0].read_text()
        self.assertNotIn('workspace-secret', text)
        self.assertNotIn('fireworks-secret', text)
        self.assertIn('Captured output', text)
        self.assertEqual(files[0].stat().st_mode & 0o777, 0o600)

    def test_long_outage_preserves_exception_and_replays_after_process_restart(self):
        self.post.side_effect = requests.ConnectionError('offline')
        tracker = self.tracker()
        with self.assertLogs('crashlens.workload_tracker', level='WARNING'):
            with self.assertRaisesRegex(ValueError, 'training failure'):
                with tracker.track('Offline failure'):
                    raise ValueError('training failure')
        self.assertIsNone(tracker.last_workload_id)
        self.assertEqual(self.post.call_count, 4)  # Initial attempt plus three final attempts.
        stored = json.loads(self.pending()[0].read_text())
        self.post.side_effect = None
        replay = self.tracker().flush_pending()
        self.assertEqual(replay, {stored['client_run_id']: 7})
        self.assertEqual(self.put.call_args.kwargs['json']['status'], 'failed')
        self.assertIn('ValueError: training failure', self.put.call_args.kwargs['json']['job_logs'])
        self.assertEqual(self.put.call_args.kwargs['json']['finished_at'], stored['payload']['finished_at'])
        self.assertFalse(self.pending())

    def test_replay_is_scoped_to_same_backend_and_workspace_key(self):
        self.put.side_effect = requests.ConnectionError('offline')
        with self.assertLogs('crashlens.workload_tracker', level='WARNING'):
            with self.tracker().track('Scoped replay'):
                pass
        self.put.reset_mock(side_effect=True)
        other = WorkloadTracker('http://test', api_key='other-key', outbox_dir=self.directory)
        self.assertEqual(other.flush_pending(), {})
        other = WorkloadTracker('http://another-backend', api_key='workspace-secret', outbox_dir=self.directory)
        self.assertEqual(other.flush_pending(), {})
        self.put.assert_not_called()
        self.assertEqual(len(self.pending()), 1)
        self.assertEqual(len(self.tracker().flush_pending()), 1)
        self.assertFalse(self.pending())

    def test_diagnosis_does_not_send_an_unregistered_workload_id(self):
        with self.assertRaisesRegex(ValueError, 'not registered'):
            self.tracker().diagnose(None)
        self.post.assert_not_called()

    def test_disk_failure_does_not_replace_training_exception(self):
        tracker = self.tracker()
        self.put.side_effect = requests.ConnectionError('offline')
        with patch.object(tracker._outbox, 'save', side_effect=OSError('disk full')):
            with self.assertLogs('crashlens.workload_tracker', level='WARNING'):
                with self.assertRaisesRegex(RuntimeError, 'original error'):
                    with tracker.track('Disk failure'):
                        raise RuntimeError('original error')

    def test_interruption_during_delivery_leaves_a_terminal_recovery_file(self):
        self.put.side_effect = KeyboardInterrupt()
        with self.assertRaises(KeyboardInterrupt):
            with self.tracker().track('Interrupted delivery'):
                pass
        record = json.loads(self.pending()[0].read_text())
        self.assertEqual(record['payload']['status'], 'succeeded')

    def test_stalled_publisher_defers_terminal_delivery_to_outbox(self):
        tracker = self.tracker()
        with patch('crashlens.workload_tracker.Thread') as thread, \
             patch.object(tracker, '_deliver_final') as deliver:
            thread.return_value.is_alive.return_value = True
            with self.assertLogs('crashlens.workload_tracker', level='WARNING'):
                with tracker.track('Stalled publisher'):
                    pass
            deliver.assert_not_called()
        self.assertEqual(len(self.pending()), 1)

    def test_stop_after_registration_prevents_a_late_running_update(self):
        stop = threading.Event()
        tracker = self.tracker()
        record = {'client_run_id': 'test-run', 'workload_id': None, 'name': 'late registration'}

        def register(*args, **kwargs):
            stop.set()
            return 7

        with patch.object(tracker, '_create_workload', side_effect=register):
            tracker._deliver(record, {'status': 'running'}, stop)
        self.put.assert_not_called()

    def test_notebook_offline_failure_does_not_diagnose_a_missing_workload(self):
        self.post.side_effect = requests.ConnectionError('offline')
        tracker = JupyterWorkloadTracker('http://test', outbox_dir=self.directory)
        tracker.in_jupyter = False
        with patch.object(tracker, 'diagnose') as diagnose:
            with self.assertLogs('crashlens.workload_tracker', level='WARNING'):
                with self.assertRaisesRegex(RuntimeError, 'cell failure'):
                    with tracker.track('Notebook offline', auto_diagnose=True):
                        raise RuntimeError('cell failure')
            diagnose.assert_not_called()
        self.assertEqual(len(self.pending()), 1)


if __name__ == '__main__':
    unittest.main()
