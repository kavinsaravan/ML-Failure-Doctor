"""
GPU metrics collection for real workloads

Collects GPU memory and utilization during training with:
- Try/except wrapping so collection never crashes the job
- Ring buffer to limit memory (last 300 samples = 10 minutes at 2s intervals)
- Final snapshot on exit to capture OOM peak
- Multi-vendor support (NVIDIA via pynvml, AMD/ROCm via torch.cuda)
"""

import time
import json
from typing import List, Dict, Optional
from threading import Thread, Event, Lock
from collections import deque
from datetime import datetime, timezone


class GPUMetricsSampler:
    """
    Background GPU metrics sampler with ring buffer

    Usage:
        sampler = GPUMetricsSampler()
        sampler.start()
        try:
            train_model()
        finally:
            samples = sampler.stop()
            metrics_json = sampler.to_json(samples)
    """

    def __init__(self, interval: float = 2.0, max_samples: int = 300):
        self.interval = interval
        self.max_samples = max_samples
        self.samples = deque(maxlen=max_samples)
        self.samples_lock = Lock()
        self.thread: Optional[Thread] = None
        self.stop_event = Event()

        # Try to initialize GPU monitoring
        self.nvml_available = False
        self.torch_cuda_available = False

        try:
            import pynvml
            pynvml.nvmlInit()
            self.pynvml = pynvml
            self.nvml_handle = pynvml.nvmlDeviceGetHandleByIndex(0)
            self.nvml_available = True
        except Exception:
            # pynvml not available or no NVIDIA GPU
            pass

        try:
            import torch
            if torch.cuda.is_available():
                self.torch = torch
                self.torch_cuda_available = True
        except Exception:
            # torch not available
            pass

    def start(self):
        """Start background sampling thread"""
        if self.thread is not None:
            return  # Already started

        # Reset peak memory stats so we only track this run (important for notebooks)
        if self.torch_cuda_available:
            try:
                self.torch.cuda.reset_peak_memory_stats()
            except Exception:
                pass

        self.stop_event.clear()
        self.thread = Thread(target=self._sample_loop, daemon=True)
        self.thread.start()

    def stop(self) -> List[Dict]:
        """
        Stop sampling and return all collected samples

        Returns:
            List of sample dicts with timestamp, memory, utilization, etc.
        """
        self.stop_event.set()
        if self.thread is not None:
            self.thread.join(timeout=5.0)
            self.thread = None

        # Take final snapshot to capture peak memory
        final_sample = self._collect_sample()
        if final_sample:
            with self.samples_lock:
                self.samples.append(final_sample)

        return self.snapshot()

    def snapshot(self) -> List[Dict]:
        """Thread-safe copy for live uploads."""
        with self.samples_lock:
            return list(self.samples)

    def _sample_loop(self):
        """Background thread that periodically collects metrics"""
        while not self.stop_event.is_set():
            sample = self._collect_sample()
            if sample:
                with self.samples_lock:
                    self.samples.append(sample)

            # Sleep with early exit if stop is signaled
            self.stop_event.wait(self.interval)

    def _collect_sample(self) -> Optional[Dict]:
        """
        Collect a single GPU metrics snapshot

        Never raises exceptions - returns None on failure
        """
        try:
            # ISO 8601 timestamp for JavaScript Date compatibility
            timestamp = datetime.now(timezone.utc).isoformat().replace("+00:00", "Z")

            # Try NVIDIA first (most detailed)
            if self.nvml_available:
                return self._collect_nvidia(timestamp)

            # Fall back to torch.cuda (works on AMD ROCm too)
            if self.torch_cuda_available:
                return self._collect_torch_cuda(timestamp)

            return None

        except Exception:
            # Never crash the user's job
            return None

    def _collect_nvidia(self, timestamp: str) -> Dict:
        """Collect NVIDIA GPU metrics via pynvml"""
        mem_info = self.pynvml.nvmlDeviceGetMemoryInfo(self.nvml_handle)
        utilization = self.pynvml.nvmlDeviceGetUtilizationRates(self.nvml_handle)

        try:
            temp = self.pynvml.nvmlDeviceGetTemperature(
                self.nvml_handle,
                self.pynvml.NVML_TEMPERATURE_GPU
            )
        except Exception:
            temp = None

        return {
            "timestamp": timestamp,
            "gpu_memory_used_mb": mem_info.used / (1024 ** 2),
            "gpu_memory_total_mb": mem_info.total / (1024 ** 2),
            "gpu_memory_percent": (mem_info.used / mem_info.total) * 100,
            "gpu_utilization_percent": utilization.gpu,
            "temperature_celsius": temp,
            "source": "nvidia-ml-py"
        }

    def _collect_torch_cuda(self, timestamp: str) -> Dict:
        """
        Collect GPU metrics via torch.cuda (works on NVIDIA and AMD ROCm)

        Uses mem_get_info() for device-wide stats (matches what nvidia-smi shows)
        """
        # Device-wide memory (includes cached blocks and CUDA context)
        free_bytes, total_bytes = self.torch.cuda.mem_get_info(0)
        used_bytes = total_bytes - free_bytes

        used_mb = used_bytes / (1024 ** 2)
        total_mb = total_bytes / (1024 ** 2)

        # Peak allocated since last reset - captures OOM spikes between samples
        peak_allocated_mb = self.torch.cuda.max_memory_allocated(0) / (1024 ** 2)

        return {
            "timestamp": timestamp,
            "gpu_memory_used_mb": used_mb,
            "gpu_memory_total_mb": total_mb,
            "gpu_memory_percent": (used_mb / total_mb * 100) if total_mb > 0 else 0,
            "gpu_memory_peak_mb": peak_allocated_mb,
            "gpu_utilization_percent": None,  # Not available via torch.cuda
            "temperature_celsius": None,
            "source": "torch.cuda"
        }

    @staticmethod
    def to_json(samples: List[Dict]) -> str:
        """
        Convert samples to JSON string for storage

        Args:
            samples: List of sample dicts from stop()

        Returns:
            JSON string ready to send to backend
        """
        if not samples:
            return ""

        return json.dumps(samples)
