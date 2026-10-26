"""
WorkloadTracker - Track real ML training workloads
"""

import requests
import time
import traceback
import sys
import logging
import os
import uuid
from datetime import datetime, timezone
from .outbox import PendingUploads
from functools import wraps
from threading import Thread, Event, Lock
from typing import Optional, Dict, Any
from contextlib import contextmanager

try:
    from .metrics import GPUMetricsSampler
    METRICS_AVAILABLE = True
except ImportError:
    METRICS_AVAILABLE = False


_tracking_lock = Lock()
_REQUEST_TIMEOUT = 3
_FINAL_ATTEMPTS = 3
DEFAULT_API_URL = "https://ml-failure-doctor-production.up.railway.app"


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

    def __init__(self, api_url: Optional[str] = None, api_key: Optional[str] = None, upload_interval: float = 2.0, fireworks_api_key: Optional[str] = None, fireworks_model: Optional[str] = None, device=None, outbox_dir=None):
        if upload_interval <= 0:
            raise ValueError("upload_interval must be positive")
        self.device = device
        self.fireworks_api_key = fireworks_api_key
        self.fireworks_model = (fireworks_model or "auto").strip() or "auto"
        self.upload_interval = upload_interval
        self.api_url = ((api_url or "").strip() or os.getenv("CRASHLENS_URL", "").strip() or DEFAULT_API_URL).rstrip('/')
        self.api_key = api_key
        self.last_workload_id = None
        self._last_client_run_id = None
        self._outbox = PendingUploads(self.api_url, api_key, outbox_dir)
        self.headers = {'Content-Type': 'application/json'}
        if api_key:
            self.headers['Authorization'] = f'Bearer {api_key}'
        
    def _create_workload(self, name: str, workload_type: str = "ML_JOB", client_run_id=None) -> int:
        """Create a workload entry"""
        response = requests.post(
            f"{self.api_url}/workloads",
            json={
                "name": name,
                "type": workload_type,
                "status": "running",
                "client_run_id": client_run_id,
            },
            headers=self.headers, timeout=_REQUEST_TIMEOUT
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
        timeout: float = _REQUEST_TIMEOUT,
        started_at=None,
        finished_at=None
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

        if started_at is not None:
            data["started_at"] = started_at
        if finished_at is not None:
            data["finished_at"] = finished_at

        response = requests.put(
            f"{self.api_url}/workloads/{workload_id}",
            json=data,
            headers=self.headers, timeout=timeout
        )
    
        response.raise_for_status()

    @contextmanager
    def track(self, name: str):
        """Track one workload per process; overlapping/nested contexts are rejected."""
        if not _tracking_lock.acquire(blocking=False):
            raise RuntimeError("CrashLens tracking is already active in this process; use separate processes for concurrent workloads")
        try:
            with self._track(name) as workload_id:
                yield workload_id
        finally:
            _tracking_lock.release()

    @contextmanager
    def _track(self, name: str):
        """
        Context manager for tracking a workload

        Example:
            with tracker.track("Training ResNet"):
                model.train()
        """
        record = {"schema_version": 1, "client_run_id": uuid.uuid4().hex,
                  "name": name, "workload_id": None}
        self.last_workload_id = None
        self._last_client_run_id = record["client_run_id"]
        try:
            record["workload_id"] = self._create_workload(name, client_run_id=record["client_run_id"])
            self.last_workload_id = record["workload_id"]
        except requests.RequestException:
            logging.getLogger(__name__).warning(
                "CrashLens registration unavailable; training will continue and delivery will be retried"
            )
        workload_id = record["workload_id"]
        start_time = time.monotonic()
        started_at = datetime.now(timezone.utc).isoformat()
        logs = LogBuffer()

        # Start GPU metrics collection
        sampler = None
        if METRICS_AVAILABLE:
            try:
                sampler = GPUMetricsSampler(device=self.device) if self.device is not None else GPUMetricsSampler()
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
                    payload = dict(status="running", logs=logs.snapshot(),
                                   runtime_seconds=time.monotonic() - start_time,
                                   gpu_metrics=GPUMetricsSampler.to_json(samples) if sampler else "",
                                   started_at=started_at)
                    self._deliver(record, payload, stop_live)
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
            finished_at = datetime.now(timezone.utc).isoformat()
            # Stop uploads before the terminal status update.
            stop_live.set()
            publisher.join(timeout=2 * _REQUEST_TIMEOUT + 1)

            # Stop GPU metrics collection
            gpu_metrics = None
            if sampler:
                try:
                    samples = sampler.stop()
                    gpu_metrics = sampler.to_json(samples)
                except Exception:
                    logging.getLogger(__name__).warning("GPU sampling cleanup failed", exc_info=True)

            record["payload"] = dict(
                status=status, logs=logs.snapshot(), runtime_seconds=runtime,
                exit_code=exit_code, gpu_metrics=gpu_metrics, started_at=started_at,
                finished_at=finished_at,
            )
            # Persist before retrying, so interruption during delivery can be replayed.
            pending_path = None
            try:
                pending_path = self._outbox.save(record)
            except Exception:
                logging.getLogger(__name__).warning(
                    "CrashLens could not save recovery telemetry; attempting delivery", exc_info=True
                )
            # A stalled publisher must not race a terminal upload. Its stop event
            # also prevents a late registration from starting another live update.
            delivered = False
            if not publisher.is_alive():
                delivered = self._deliver_final(record)
            if delivered and pending_path is not None:
                try:
                    self._outbox.remove(pending_path)
                except Exception:
                    logging.getLogger(__name__).warning(
                        "CrashLens delivered telemetry but could not remove its recovery file"
                    )
            elif not delivered:
                logging.getLogger(__name__).warning(
                    "CrashLens upload remains pending%s; call tracker.flush_pending() after restoring access",
                    " in " + str(self._outbox.directory) if pending_path is not None else " without a saved recovery file",
                )

    def _deliver(self, record, payload, stop=None):
        if record["workload_id"] is None:
            record["workload_id"] = self._create_workload(
                record["name"], client_run_id=record["client_run_id"]
            )
            if record["client_run_id"] == self._last_client_run_id:
                self.last_workload_id = record["workload_id"]
        if stop is not None and stop.is_set():
            return
        self._update_workload(record["workload_id"], **payload)

    @staticmethod
    def _retryable(error):
        if isinstance(error, requests.HTTPError):
            response = error.response
            return response is not None and (response.status_code in (408, 429) or response.status_code >= 500)
        return isinstance(error, (requests.ConnectionError, requests.Timeout))

    def _deliver_final(self, record):
        for attempt in range(_FINAL_ATTEMPTS):
            try:
                self._deliver(record, record["payload"])
                return True
            except Exception as error:
                if not self._retryable(error) or attempt == _FINAL_ATTEMPTS - 1:
                    logging.getLogger(__name__).warning("CrashLens terminal upload is pending")
                    return False
                time.sleep(.25 * 2 ** attempt)
        return False

    def flush_pending(self):
        """Retry saved terminal uploads for this backend/key; return local run ID → workload ID."""
        delivered = {}
        for path in self._outbox.paths():
            try:
                record = self._outbox.load(path)
                if self._deliver_final(record):
                    self._outbox.remove(path)
                    delivered[record["client_run_id"]] = record["workload_id"]
                else:
                    break  # Avoid a long queue of network timeouts during an outage.
            except Exception:
                logging.getLogger(__name__).warning("CrashLens could not replay pending upload %s", path.name)
        return delivered

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
        if workload_id is None:
            workload_id = self.last_workload_id
        if workload_id is None:
            raise ValueError("Workload is not registered yet; call flush_pending() when connected and use the returned workload ID")
        headers = dict(self.headers)
        if self.fireworks_api_key:
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


def init(api_url: Optional[str] = None, api_key: Optional[str] = None, fireworks_api_key: Optional[str] = None, fireworks_model: Optional[str] = None, device=None, outbox_dir=None):
    """Initialize global tracker"""
    global _global_tracker
    _global_tracker = WorkloadTracker(api_url, api_key, fireworks_api_key=fireworks_api_key, fireworks_model=fireworks_model, device=device, outbox_dir=outbox_dir)
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
