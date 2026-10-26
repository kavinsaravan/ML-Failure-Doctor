import io
import json
import sys
import threading
import time
import tempfile
import unittest
from unittest.mock import Mock, patch
from IPython.core.interactiveshell import InteractiveShell
from crashlens.workload_tracker import WorkloadTracker, DEFAULT_API_URL, init
from crashlens import jupyter

class TrackingTests(unittest.TestCase):
    def setUp(self):
        directory = tempfile.TemporaryDirectory()
        self.addCleanup(directory.cleanup)
        patch.dict('os.environ', {'CRASHLENS_OUTBOX_DIR': directory.name}).start()
        self.post = patch("crashlens.workload_tracker.requests.post").start()
        self.put = patch("crashlens.workload_tracker.requests.put").start()
        patch("crashlens.workload_tracker.METRICS_AVAILABLE", False).start()
        self.post.return_value.json.return_value = {"id": 7}
        self.addCleanup(patch.stopall)

    def test_fireworks_credentials_only_sent_for_diagnosis(self):
        tracker = WorkloadTracker("http://test", api_key="crashlens", fireworks_api_key="user-fireworks", fireworks_model="user-model")
        with tracker.track("job"):
            pass
        self.assertNotIn("X-Fireworks-API-Key", self.post.call_args.kwargs["headers"])
        self.assertNotIn("X-Fireworks-API-Key", self.put.call_args.kwargs["headers"])
        tracker.diagnose(7)
        headers = self.post.call_args.kwargs["headers"]
        self.assertEqual(headers["X-Fireworks-API-Key"], "user-fireworks")
        self.assertEqual(headers["X-Fireworks-Model"], "user-model")
        self.assertEqual(headers["Authorization"], "Bearer crashlens")
        self.assertNotIn("X-Fireworks-API-Key", tracker.headers)
        WorkloadTracker("http://test", fireworks_api_key="secret").diagnose(7)
        self.assertEqual(self.post.call_args.kwargs["headers"]["X-Fireworks-Model"], "auto")

    def test_url_selection_routes_uploads_and_workspace_credentials(self):
        for environment, explicit, expected in [
            ("", None, DEFAULT_API_URL),
            ("   ", None, DEFAULT_API_URL),
            (" https://self-hosted.example/ ", None, "https://self-hosted.example"),
            ("https://environment.example", " https://explicit.example/ ", "https://explicit.example"),
        ]:
            with self.subTest(environment=environment, explicit=explicit), patch.dict('os.environ', {'CRASHLENS_URL': environment}):
                tracker = WorkloadTracker(explicit, api_key="workspace")
                with tracker.track("sample"):
                    pass
                self.assertEqual(self.post.call_args.args[0], expected + "/workloads")
                self.assertEqual(self.put.call_args.args[0], expected + "/workloads/7")
                self.assertEqual(self.post.call_args.kwargs["headers"]["Authorization"], "Bearer workspace")

    def test_global_and_jupyter_initialization_allow_key_without_url(self):
        with patch.dict('os.environ', {'CRASHLENS_URL': ''}), patch('crashlens.workload_tracker._global_tracker'):
            self.assertEqual(init(api_key="workspace").api_url, DEFAULT_API_URL)
            self.assertEqual(jupyter.JupyterWorkloadTracker(api_key="workspace").api_url, DEFAULT_API_URL)

    def test_notebook_magic_initializes_without_url(self):
        magics = jupyter.CrashLensMagics(shell=InteractiveShell.instance())
        with patch.dict('os.environ', {'CRASHLENS_URL': '', 'CRASHLENS_API_KEY': 'workspace'}), patch('crashlens.jupyter.display'):
            magics.crashlens_init('')
        self.assertEqual(magics.tracker.api_url, DEFAULT_API_URL)
        self.assertEqual(magics.tracker.headers['Authorization'], 'Bearer workspace')

    def test_interruptions_and_exceptions_report_failure(self):
        for error in [KeyboardInterrupt(), SystemExit(2), RuntimeError("broken")]:
            with self.subTest(error=type(error).__name__):
                with self.assertRaises(type(error)):
                    with WorkloadTracker("http://test").track("job"):
                        raise error
                data = self.put.call_args.kwargs["json"]
                self.assertEqual(data["status"], "failed")
                self.assertIn(type(error).__name__, data["job_logs"])
                self.assertGreaterEqual(data["runtime_seconds"], 0)
                self.assertEqual(self.put.call_args.kwargs["timeout"], 3)

    def test_reporting_failure_preserves_training_exception(self):
        self.put.return_value.raise_for_status.side_effect = RuntimeError("HTTP 500")
        with self.assertLogs("crashlens.workload_tracker", level="WARNING"):
            with self.assertRaisesRegex(ValueError, "training error"):
                with WorkloadTracker("http://test").track("job"):
                    raise ValueError("training error")

    def test_success_captures_logs_and_restores_streams(self):
        old_stdout, old_stderr = sys.stdout, sys.stderr
        with patch("sys.stdout", io.StringIO()):
            with WorkloadTracker("http://test").track("job"):
                print("epoch 1")
        self.assertIs(sys.stdout, old_stdout)
        self.assertIs(sys.stderr, old_stderr)
        data = self.put.call_args.kwargs["json"]
        self.assertEqual(data["status"], "succeeded")
        self.assertIn("epoch 1", data["job_logs"])

    def test_live_metrics_and_logs_are_sent_before_exit(self):
        live = threading.Event()
        sample = {"gpu_memory_used_mb": 1024, "gpu_memory_total_mb": 2048, "source": "test-fixture"}
        class Sampler:
            def start(self): pass
            def snapshot(self): return [sample]
            def stop(self): return [sample]
            @staticmethod
            def to_json(samples): return json.dumps(samples)
        payloads = []
        def put(*args, **kwargs):
            payloads.append(kwargs["json"])
            if kwargs["json"]["status"] == "running": live.set()
            return Mock()
        self.put.side_effect = put
        with patch("crashlens.workload_tracker.METRICS_AVAILABLE", True), patch("crashlens.workload_tracker.GPUMetricsSampler", Sampler), patch("sys.stdout", io.StringIO()):
            with WorkloadTracker("http://test", upload_interval=.02).track("live"):
                print("epoch progress")
                self.assertTrue(live.wait(2), "no live update before context exit")
                running = [data for data in payloads if data["status"] == "running"][-1]
                self.assertIn("epoch progress", running["job_logs"])
                self.assertEqual(json.loads(running["gpu_metrics"])[0]["gpu_memory_used_mb"], 1024)
        self.assertEqual(payloads[-1]["status"], "succeeded")
        count = len(payloads)
        time.sleep(.05)
        self.assertEqual(len(payloads), count, "publisher continued after finalization")

    def test_notebook_magic_marks_failed_cell_and_diagnoses(self):
        shell = InteractiveShell.instance()
        magics = jupyter.CrashLensMagics(shell=shell)
        magics.tracker = jupyter.JupyterWorkloadTracker("http://test")
        magics.tracker.in_jupyter = False
        magics.tracker.diagnose = Mock(return_value={"source": "rules", "recommended_fix": "Install torch"})
        with patch("sys.stdout", io.StringIO()), patch("sys.stderr", io.StringIO()):
            with self.assertRaisesRegex(RuntimeError, "cell failed"):
                magics.crashlens_track('"Training"', 'raise RuntimeError("cell failed")')
        self.assertEqual(self.put.call_args.kwargs["json"]["status"], "failed")
        magics.tracker.diagnose.assert_called_once_with(7, display=False)

    def test_notebook_syntax_error_is_recorded(self):
        shell = InteractiveShell.instance()
        magics = jupyter.CrashLensMagics(shell=shell)
        magics.tracker = jupyter.JupyterWorkloadTracker("http://test")
        magics.tracker.in_jupyter = False
        magics.tracker.diagnose = Mock(return_value={})
        with patch("sys.stdout", io.StringIO()), patch("sys.stderr", io.StringIO()):
            with self.assertRaises(SyntaxError):
                magics.crashlens_track("syntax", "if")
        self.assertEqual(self.put.call_args.kwargs["json"]["status"], "failed")

    def test_notebook_metrics_decodes_backend_envelope(self):
        tracker = jupyter.JupyterWorkloadTracker("http://test")
        tracker.in_jupyter = True
        samples = [{"gpu_memory_used_mb": 1024}]
        with patch.object(jupyter.requests, "get") as get, patch.object(jupyter, "display") as display:
            get.return_value.status_code = 200
            get.return_value.json.return_value = {"metrics": json.dumps(samples)}
            tracker._display_metrics(7)
        self.assertEqual(display.call_count, 2)
        self.assertEqual(display.call_args.args[0].iloc[0]["gpu_memory_used_mb"], 1024)

    def test_notebook_report_displays_fixes_source_and_escapes_html(self):
        tracker = jupyter.JupyterWorkloadTracker("http://test")
        tracker.in_jupyter = True
        with patch.object(jupyter, "display") as display:
            tracker._display_diagnosis({"source": "rules", "root_cause": "<script>bad</script>", "recommended_fix": "Install torch", "prevention": "Pin versions", "confidence": .5})
        html = display.call_args.args[0].data
        self.assertIn("Install torch", html)
        self.assertIn("Pin versions", html)
        self.assertIn("Rule-based", html)
        self.assertNotIn("<script>", html)

    def test_notebook_workload_escapes_external_failure_label(self):
        tracker = jupyter.JupyterWorkloadTracker("http://test")
        tracker.in_jupyter = True
        label = '<img src=x onerror="alert(1)">'
        with patch.object(tracker, "_get_workload", return_value={
            "id": 7, "name": "Training", "status": "failed", "failure_type": label,
        }), patch.object(jupyter, "display") as display:
            tracker.show_workload(7)
        html = display.call_args.args[0].data
        self.assertNotIn(label, html)
        self.assertIn("&lt;img", html)

    def test_notebook_initialization_escapes_backend_url(self):
        magics = jupyter.CrashLensMagics(shell=InteractiveShell.instance())
        with patch.object(jupyter, "display") as display:
            magics.crashlens_init('https://example.test/<img src=x onerror="alert(1)">')
        html = display.call_args.args[0].data
        self.assertNotIn("<img", html)
        self.assertIn("&lt;img", html)

if __name__ == "__main__":
    unittest.main()
