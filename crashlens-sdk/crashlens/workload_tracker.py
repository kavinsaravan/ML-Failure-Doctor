"""
WorkloadTracker - Track real ML training workloads
"""

import requests
import time
import traceback
import sys
from typing import Optional, Dict, Any
from contextlib import contextmanager

try:
    from .metrics import GPUMetricsSampler
    METRICS_AVAILABLE = True
except ImportError:
    METRICS_AVAILABLE = False


class WorkloadTracker:
    """
    Track ML workloads with automatic failure reporting

    Usage:
        tracker = WorkloadTracker(
            "https://your-backend.railway.app",
            api_key=os.getenv("CRASHLENS_API_KEY")  # Required in production
        )

        # Option 1: Context manager (recommended)
        with tracker.track("Training GPT-2"):
            # Your training code here
            train_model()

        # Option 2: Decorator
        @tracker.track_function("Fine-tuning BERT")
        def train():
            # Your training code
            pass
    """

    def __init__(self, api_url: str, api_key: Optional[str] = None):
        self.api_url = api_url.rstrip('/')
        self.api_key = api_key
        self.workload_id: Optional[int] = None
        self.headers = {'Content-Type': 'application/json'}
        if api_key:
            self.headers['Authorization'] = f'Bearer {api_key}'
        
    def _create_workload(self, name: str, workload_type: str = "ML_JOB") -> int:
        """Create a workload entry"""
        response = requests.post(
            f"{self.api_url}/workloads",
            json={
                "name": name,
                "type": workload_type,
                "status": "running"
            },
            headers=self.headers
        )
        response.raise_for_status()
        return response.json()["id"]
    
    def _update_workload(
        self,
        workload_id: int,
        status: str,
        logs: Optional[str] = None,
        runtime_seconds: Optional[float] = None,
        exit_code: Optional[int] = None,
        failure_type: Optional[str] = None,
        gpu_metrics: Optional[str] = None
    ):
        """Update workload status"""
        data = {"status": status}
        if logs:
            data["job_logs"] = logs
        if runtime_seconds is not None:
            data["runtime_seconds"] = runtime_seconds
        if exit_code is not None:
            data["exit_code"] = exit_code
        if failure_type:
            data["failure_type"] = failure_type
        if gpu_metrics:
            data["gpu_metrics"] = gpu_metrics

        requests.put(
            f"{self.api_url}/workloads/{workload_id}",
            json=data,
            headers=self.headers
        )
    
    @contextmanager
    def track(self, name: str):
        """
        Context manager for tracking a workload

        Example:
            with tracker.track("Training ResNet"):
                model.train()
        """
        workload_id = self._create_workload(name)
        start_time = time.time()
        logs = []

        # Start GPU metrics collection
        sampler = None
        if METRICS_AVAILABLE:
            sampler = GPUMetricsSampler()
            sampler.start()

        # Capture stdout/stderr
        class LogCapture:
            def __init__(self, original):
                self.original = original

            def write(self, text):
                logs.append(text)
                self.original.write(text)

            def flush(self):
                self.original.flush()

        old_stdout = sys.stdout
        old_stderr = sys.stderr
        sys.stdout = LogCapture(old_stdout)
        sys.stderr = LogCapture(old_stderr)

        status = "succeeded"
        exit_code = 0

        try:
            yield workload_id

        except Exception as e:
            # Failure - set status but don't update yet
            status = "failed"
            exit_code = 1
            error_logs = "".join(logs) + "\n\n" + traceback.format_exc()
            logs = [error_logs]
            raise

        finally:
            sys.stdout = old_stdout
            sys.stderr = old_stderr

            # Stop GPU metrics collection
            gpu_metrics = None
            if sampler:
                samples = sampler.stop()
                gpu_metrics = sampler.to_json(samples)

            # Update workload with logs and metrics
            runtime = time.time() - start_time
            self._update_workload(
                workload_id,
                status=status,
                logs="".join(logs),
                runtime_seconds=runtime,
                exit_code=exit_code,
                gpu_metrics=gpu_metrics
            )
    
    def track_function(self, name: str):
        """
        Decorator for tracking a function as a workload
        
        Example:
            @tracker.track_function("Training Model")
            def train():
                model.fit(X, y)
        """
        def decorator(func):
            def wrapper(*args, **kwargs):
                with self.track(name):
                    return func(*args, **kwargs)
            return wrapper
        return decorator
    
    def diagnose(self, workload_id: int) -> Dict[str, Any]:
        """
        Run AI diagnosis on a failed workload

        Returns:
            dict with root_cause, recommended_fixes, etc.
        """
        response = requests.post(
            f"{self.api_url}/workloads/{workload_id}/diagnose",
            headers=self.headers
        )
        response.raise_for_status()
        return response.json()


# Singleton instance
_global_tracker: Optional[WorkloadTracker] = None


def init(api_url: str):
    """Initialize global tracker"""
    global _global_tracker
    _global_tracker = WorkloadTracker(api_url)
    return _global_tracker


def track(name: str):
    """Use global tracker"""
    if _global_tracker is None:
        raise RuntimeError("Call crashlens.init() first")
    return _global_tracker.track(name)


def track_function(name: str):
    """Use global tracker as decorator"""
    if _global_tracker is None:
        raise RuntimeError("Call crashlens.init() first")
    return _global_tracker.track_function(name)
