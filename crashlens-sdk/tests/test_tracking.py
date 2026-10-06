import io
import json
import sys
import unittest
from unittest.mock import Mock, patch
from IPython.core.interactiveshell import InteractiveShell
from crashlens.workload_tracker import WorkloadTracker
from crashlens import jupyter

class TrackingTests(unittest.TestCase):
    def setUp(self):
        self.post = patch("crashlens.workload_tracker.requests.post").start()
        self.put = patch("crashlens.workload_tracker.requests.put").start()
        patch("crashlens.workload_tracker.METRICS_AVAILABLE", False).start()
        self.post.return_value.json.return_value = {"id": 7}
        self.addCleanup(patch.stopall)

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
                self.assertEqual(self.put.call_args.kwargs["timeout"], 15)

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

if __name__ == "__main__":
    unittest.main()
