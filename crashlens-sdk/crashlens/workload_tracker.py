"""
WorkloadTracker - Track real ML training workloads
"""

import requests
import time
import traceback
import sys
from typing import Optional, Dict, Any
from contextlib import contextmanager


class WorkloadTracker:
    """
    Track ML workloads with automatic failure reporting
    
    Usage:
        tracker = WorkloadTracker("https://your-backend.railway.app")
        
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
    
    def __init__(self, api_url: str):
        self.api_url = api_url.rstrip('/')
        self.workload_id: Optional[int] = None
        
    def _create_workload(self, name: str, workload_type: str = "ML_JOB") -> int:
        """Create a workload entry"""
        response = requests.post(
            f"{self.api_url}/workloads",
            json={
                "name": name,
                "type": workload_type,
                "status": "running"
            }
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
        failure_type: Optional[str] = None
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
            
        requests.put(
            f"{self.api_url}/workloads/{workload_id}",
            json=data
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
        
        try:
            yield workload_id
            
            # Success
            runtime = time.time() - start_time
            self._update_workload(
                workload_id,
                status="succeeded",
                logs="".join(logs),
                runtime_seconds=runtime,
                exit_code=0
            )
            
        except Exception as e:
            # Failure
            runtime = time.time() - start_time
            error_logs = "".join(logs) + "\n\n" + traceback.format_exc()
            
            self._update_workload(
                workload_id,
                status="failed",
                logs=error_logs,
                runtime_seconds=runtime,
                exit_code=1
            )
            
            # Re-raise the exception
            raise
            
        finally:
            sys.stdout = old_stdout
            sys.stderr = old_stderr
    
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
            f"{self.api_url}/workloads/{workload_id}/diagnose"
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
