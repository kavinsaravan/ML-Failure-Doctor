import sys
import unittest
from types import SimpleNamespace
from unittest.mock import Mock, patch

from crashlens.metrics import GPUMetricsSampler


class GPUMetricsTests(unittest.TestCase):
    def torch(self, hip=None, uuid=None):
        cuda = Mock()
        cuda.is_available.return_value = True
        cuda.current_device.return_value = 1
        cuda.get_device_properties.return_value = SimpleNamespace(
            uuid=uuid, name="AMD Instinct" if hip else "NVIDIA GPU")
        cuda.mem_get_info.return_value = (2 * 1024**3, 8 * 1024**3)
        cuda.max_memory_allocated.return_value = 5 * 1024**3
        return SimpleNamespace(cuda=cuda, version=SimpleNamespace(hip=hip))

    def sampler(self, torch, device=None, nvml=None):
        with patch.dict(sys.modules, {"torch": torch, "pynvml": nvml}):
            return GPUMetricsSampler(device=device)

    def test_cuda_memory_uses_the_selected_device(self):
        torch = self.torch()
        sampler = self.sampler(torch, "cuda:2")
        sample = sampler._collect_sample()
        torch.cuda.mem_get_info.assert_called_once_with(2)
        torch.cuda.max_memory_allocated.assert_called_once_with(2)
        self.assertEqual(sample["gpu_memory_used_mb"], 6144)
        self.assertEqual(sample["gpu_memory_total_mb"], 8192)
        self.assertEqual(sample["gpu_memory_percent"], 75)
        self.assertEqual(sample["gpu_memory_peak_mb"], 5120)
        self.assertEqual(sample["source"], "torch.cuda")
        self.assertIsNone(sample["gpu_utilization_percent"])
        self.assertIsNone(sample["temperature_celsius"])

    def test_rocm_collects_through_torch_cuda_without_nvml(self):
        torch = self.torch(hip="6.2.0")
        sampler = self.sampler(torch, "cuda:0")
        sample = sampler._collect_sample()
        self.assertEqual(sample["device_name"], "AMD Instinct")
        self.assertEqual(sample["device_index"], 0)
        self.assertEqual(sample["source"], "torch.cuda")
        self.assertEqual(sample["gpu_memory_percent"], 75)

    def test_bare_cuda_captures_the_current_device(self):
        torch = self.torch()
        sampler = self.sampler(torch, "cuda")
        torch.cuda.current_device.assert_called_once_with()
        self.assertEqual(sampler._collect_sample()["device_index"], 1)

    def test_nvml_sample_failure_falls_back_to_torch(self):
        torch = self.torch(uuid="GPU-selected")
        nvml = Mock()
        nvml.nvmlDeviceGetUUID.return_value = "GPU-selected"
        nvml.nvmlDeviceGetName.return_value = "NVIDIA GPU"
        nvml.nvmlDeviceGetMemoryInfo.side_effect = RuntimeError("unavailable")
        sampler = self.sampler(torch, "cuda:1", nvml)
        nvml.nvmlDeviceGetHandleByUUID.assert_called_once_with("GPU-selected")
        self.assertEqual(sampler._collect_sample()["source"], "torch.cuda")

    def test_collection_failure_does_not_break_training(self):
        torch = self.torch(hip="6.2.0")
        torch.cuda.mem_get_info.side_effect = RuntimeError("unavailable")
        self.assertIsNone(self.sampler(torch)._collect_sample())

    def test_no_gpu_does_not_simulate_metrics(self):
        torch = self.torch()
        torch.cuda.is_available.return_value = False
        self.assertIsNone(self.sampler(torch)._collect_sample())

    def test_unsupported_device_cannot_select_an_unrelated_gpu(self):
        for device in ("cpu", "vulkan", "cuda:-1"):
            with self.subTest(device=device), self.assertRaisesRegex(ValueError, "CUDA/ROCm"):
                GPUMetricsSampler(device=device)
