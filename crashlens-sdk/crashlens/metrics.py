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
from threading import Thread, Event
from collections import deque


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
            self.samples.append(final_sample)

        return list(self.samples)

    def _sample_loop(self):
        """Background thread that periodically collects metrics"""
        while not self.stop_event.is_set():
            sample = self._collect_sample()
            if sample:
                self.samples.append(sample)

            # Sleep with early exit if stop is signaled
            self.stop_event.wait(self.interval)

    def _collect_sample(self) -> Optional[Dict]:
        """
        Collect a single GPU metrics snapshot

        Never raises exceptions - returns None on failure
        """
        try:
            timestamp = time.time()

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

    def _collect_nvidia(self, timestamp: float) -> Dict:
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

    def _collect_torch_cuda(self, timestamp: float) -> Dict:
        """
        Collect GPU metrics via torch.cuda (works on NVIDIA and AMD ROCm)

        Limited to memory stats - no utilization or temperature
        """
        allocated = self.torch.cuda.memory_allocated(0) / (1024 ** 2)
        reserved = self.torch.cuda.memory_reserved(0) / (1024 ** 2)

        # Peak since last reset - captures OOM spikes between samples
        peak_allocated = self.torch.cuda.max_memory_allocated(0) / (1024 ** 2)

        # Get total GPU memory (not always accurate on all platforms)
        try:
            props = self.torch.cuda.get_device_properties(0)
            total = props.total_memory / (1024 ** 2)
        except Exception:
            total = reserved  # Fallback: use reserved as proxy for total

        return {
            "timestamp": timestamp,
            "gpu_memory_used_mb": allocated,
            "gpu_memory_total_mb": total,
            "gpu_memory_percent": (allocated / total * 100) if total > 0 else 0,
            "gpu_memory_peak_mb": peak_allocated,
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

        return json.dumps(samples, indent=2)
