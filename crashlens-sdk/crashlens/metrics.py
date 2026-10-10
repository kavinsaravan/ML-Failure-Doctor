"""
GPU metrics collection for real workloads

Collects GPU memory and utilization during training with:
- Try/except wrapping so collection never crashes the job
- Ring buffer to limit memory (last 300 samples = 10 minutes at 2s intervals)
- Final snapshot on exit to capture OOM peak
- Multi-vendor support (NVIDIA via pynvml, AMD/ROCm via torch.cuda, Apple via torch.mps)
"""

import os
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

    def __init__(self, interval: float = 2.0, max_samples: int = 300, device=None):
        self.interval = interval
        self.max_samples = max_samples
        self.samples = deque(maxlen=max_samples)
        self.samples_lock = Lock()
        self.thread: Optional[Thread] = None
        self.stop_event = Event()

        # Try to initialize GPU monitoring
        self.nvml_available = False
        self.torch_cuda_available = False
        self.mps_available = False

        self.device_index = None
        self.device_id = None
        self.device_name = None
        cuda_detected = False
        # Resolve the CUDA logical device on the calling thread, before sampling.
        try:
            import torch
            if device != "mps" and torch.cuda.is_available():
                cuda_detected = True
                self.torch = torch
                index = torch.cuda.current_device() if device is None else int(str(device).replace("cuda:", "", 1))
                props = torch.cuda.get_device_properties(index)
                self.device_index = index
                self.device_id = str(props.uuid) if getattr(props, "uuid", None) else None
                self.device_name = props.name
                self.torch_cuda_available = True
            elif device in (None, "mps") and hasattr(torch.backends, "mps") and torch.backends.mps.is_available():
                self.torch = torch
                self.mps_available = True
                self.device_id = "mps:0"
                self.device_name = "Apple MPS"
        except Exception:
            pass

        try:
            import pynvml
            if self.mps_available or device == "mps":
                return
            pynvml.nvmlInit()
            self.pynvml = pynvml
            if self.torch_cuda_available:
                # UUID mapping respects CUDA visibility and device ordering. If
                # unavailable, use torch telemetry rather than guessing an NVML index.
                if not self.device_id:
                    return
                handle = pynvml.nvmlDeviceGetHandleByUUID(self.device_id)
            elif cuda_detected and not (isinstance(device, str) and device.startswith(("GPU-", "MIG-"))):
                return
            elif isinstance(device, str) and device.startswith(("GPU-", "MIG-")):
                handle = pynvml.nvmlDeviceGetHandleByUUID(device)
            elif "CUDA_VISIBLE_DEVICES" not in os.environ and "CUDA_DEVICE_ORDER" not in os.environ:
                handle = pynvml.nvmlDeviceGetHandleByIndex(0 if device is None else int(device))
            else:
                return
            self.nvml_handle = handle
            def text(value):
                return value.decode() if isinstance(value, bytes) else str(value)
            self.device_id = text(pynvml.nvmlDeviceGetUUID(handle))
            self.device_name = text(pynvml.nvmlDeviceGetName(handle))
            self.nvml_available = True
        except Exception:
            pass

    def start(self):
        """Start background sampling thread"""
        if self.thread is not None:
            return  # Already started

        # Reset peak memory stats so we only track this run (important for notebooks)
        if self.torch_cuda_available:
            try:
                self.torch.cuda.reset_peak_memory_stats(self.device_index)
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
                try:
                    return self._collect_nvidia(timestamp)
                except Exception:
                    pass

            # Fall back to torch.cuda (works on AMD ROCm too)
            if self.torch_cuda_available:
                return self._collect_torch_cuda(timestamp)

            if self.mps_available:
                return self._collect_mps(timestamp)

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
            "device_id": self.device_id,
            "device_index": self.device_index,
            "device_name": self.device_name,
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
        free_bytes, total_bytes = self.torch.cuda.mem_get_info(self.device_index)
        used_bytes = total_bytes - free_bytes

        used_mb = used_bytes / (1024 ** 2)
        total_mb = total_bytes / (1024 ** 2)

        # Peak allocated since last reset - captures OOM spikes between samples
        peak_allocated_mb = self.torch.cuda.max_memory_allocated(self.device_index) / (1024 ** 2)

        return {
            "timestamp": timestamp,
            "device_id": self.device_id,
            "device_index": self.device_index,
            "device_name": self.device_name,
            "gpu_memory_used_mb": used_mb,
            "gpu_memory_total_mb": total_mb,
            "gpu_memory_percent": (used_mb / total_mb * 100) if total_mb > 0 else 0,
            "gpu_memory_peak_mb": peak_allocated_mb,
            "gpu_utilization_percent": None,  # Not available via torch.cuda
            "temperature_celsius": None,
            "source": "torch.cuda"
        }

    def _collect_mps(self, timestamp: str) -> Optional[Dict]:
        """Process Metal allocations relative to the recommended working set, not VRAM."""
        mps = self.torch.mps
        allocated = mps.current_allocated_memory()
        used = mps.driver_allocated_memory()
        recommended = mps.recommended_max_memory()
        if recommended <= 0:
            return None
        return {
            "timestamp": timestamp,
            "device_id": self.device_id,
            "device_index": self.device_index,
            "device_name": self.device_name,
            "gpu_memory_used_mb": used / (1024 ** 2),
            "gpu_memory_total_mb": recommended / (1024 ** 2),
            "gpu_memory_percent": used / recommended * 100,
            "gpu_tensor_memory_mb": allocated / (1024 ** 2),
            "memory_scope": "process",
            "memory_total_basis": "recommended_working_set",
            "gpu_utilization_percent": None,
            "temperature_celsius": None,
            "source": "torch.mps",
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
