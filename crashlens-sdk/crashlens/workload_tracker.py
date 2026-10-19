"""
WorkloadTracker - Track real ML training workloads
"""

import requests
import time
import traceback
import sys
import logging
from functools import wraps
from threading import Thread, Event, Lock
from typing import Optional, Dict, Any
from contextlib import contextmanager

try:
    from .metrics import GPUMetricsSampler
    METRICS_AVAILABLE = True
except ImportError:
    METRICS_AVAILABLE = False


class LogBuffer:
    """Bounded, thread-safe tail of captured output."""
    def __init__(self, limit=256 * 1024):
        self.limit = limit
        self.text = ""
        self.lock = Lock()

    def append(self, text):
        with self.lock:
            self.text = (self.text + text).encode("utf-8")[-self.limit:].decode("utf-8", errors="ignore")

    def snapshot(self):
        with self.lock:
            return self.text


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

    def __init__(self, api_url: str, api_key: Optional[str] = None, upload_interval: float = 2.0, fireworks_api_key: Optional[str] = None, fireworks_model: Optional[str] = None):
        if upload_interval <= 0:
            raise ValueError("upload_interval must be positive")
        self.fireworks_api_key = fireworks_api_key
        self.fireworks_model = fireworks_model
        self.upload_interval = upload_interval
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
            headers=self.headers, timeout=15
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
        gpu_metrics: Optional[str] = None,
        timeout: float = 15
    ):
        """Update workload status"""
        data = {"status": status}
        if logs is not None:
            data["job_logs"] = logs
        if runtime_seconds is not None:
            data["runtime_seconds"] = runtime_seconds
        if exit_code is not None:
            data["exit_code"] = exit_code
        if failure_type:
            data["failure_type"] = failure_type
        if gpu_metrics is not None:
            data["gpu_metrics"] = gpu_metrics

        response = requests.put(
            f"{self.api_url}/workloads/{workload_id}",
            json=data,
            headers=self.headers, timeout=timeout
        )
    
        response.raise_for_status()

    @contextmanager
    def track(self, name: str):
        """
        Context manager for tracking a workload

        Example:
            with tracker.track("Training ResNet"):
                model.train()
        """
        workload_id = self._create_workload(name)
        start_time = time.monotonic()
        logs = LogBuffer()

        # Start GPU metrics collection
        sampler = None
        if METRICS_AVAILABLE:
            try:
                sampler = GPUMetricsSampler()
                sampler.start()
            except Exception:
                sampler = None
                logging.getLogger(__name__).warning("GPU sampling unavailable", exc_info=True)

        # Capture stdout/stderr
        class LogCapture:
            def __init__(self, original):
                self.original = original

            def write(self, text):
                logs.append(text)
                return self.original.write(text)

            def flush(self):
                self.original.flush()

            def __getattr__(self, name):
                return getattr(self.original, name)

        old_stdout = sys.stdout
        old_stderr = sys.stderr
        sys.stdout = LogCapture(old_stdout)
        sys.stderr = LogCapture(old_stderr)

        stop_live = Event()

        def publish_live():
            while not stop_live.wait(self.upload_interval):
                try:
                    samples = sampler.snapshot() if sampler else []
                    self._update_workload(
                        workload_id, status="running", logs=logs.snapshot(),
                        runtime_seconds=time.monotonic() - start_time,
                        gpu_metrics=GPUMetricsSampler.to_json(samples) if sampler else "",
                        timeout=5
                    )
                except Exception:
                    logging.getLogger(__name__).warning(
                        "CrashLens live upload failed for workload %s", workload_id, exc_info=True
                    )

        publisher = Thread(target=publish_live, daemon=True)
        publisher.start()

        status = "succeeded"
        exit_code = 0

        try:
            yield workload_id

        except BaseException as e:
            # Failure - set status but don't update yet
            status = "succeeded" if isinstance(e, SystemExit) and e.code in (None, 0) else "failed"
            exit_code = 0 if status == "succeeded" else 1
            logs.append("\n\n" + traceback.format_exc())
            raise

        finally:
            sys.stdout = old_stdout
            sys.stderr = old_stderr

            runtime = time.monotonic() - start_time
            # Stop uploads before the terminal status update.
            stop_live.set()
            publisher.join(timeout=12)

            # Stop GPU metrics collection
            gpu_metrics = None
            if sampler:
                try:
                    samples = sampler.stop()
                    gpu_metrics = sampler.to_json(samples)
                except Exception:
                    logging.getLogger(__name__).warning("GPU sampling cleanup failed", exc_info=True)

            # Update workload with logs and metrics; cleanup latency is not training time.
            try:
                self._update_workload(
                    workload_id, status=status, logs=logs.snapshot(),
                    runtime_seconds=runtime, exit_code=exit_code, gpu_metrics=gpu_metrics
                )
            except Exception:
                # Telemetry must not replace the training exception or fail completed training.
                logging.getLogger(__name__).warning(
                    "CrashLens could not report workload %s", workload_id, exc_info=True
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
            @wraps(func)
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
        headers = dict(self.headers)
        if self.fireworks_api_key:
            if not self.fireworks_model:
                raise ValueError("fireworks_model is required with fireworks_api_key")
            headers["X-Fireworks-API-Key"] = self.fireworks_api_key
            headers["X-Fireworks-Model"] = self.fireworks_model
        response = requests.post(
            f"{self.api_url}/workloads/{workload_id}/diagnose",
            headers=headers, timeout=75
        )
        response.raise_for_status()
        return response.json()


# Singleton instance
_global_tracker: Optional[WorkloadTracker] = None


def init(api_url: str, api_key: Optional[str] = None, fireworks_api_key: Optional[str] = None, fireworks_model: Optional[str] = None):
    """Initialize global tracker"""
    global _global_tracker
    _global_tracker = WorkloadTracker(api_url, api_key, fireworks_api_key=fireworks_api_key, fireworks_model=fireworks_model)
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
