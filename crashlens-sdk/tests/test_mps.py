import sys
import unittest
from types import SimpleNamespace
from unittest.mock import Mock, patch
from crashlens.metrics import GPUMetricsSampler


class MPSTests(unittest.TestCase):
    def sampler(self, available=True):
        mps = SimpleNamespace(
            current_allocated_memory=Mock(return_value=2 * 1024**2),
            driver_allocated_memory=Mock(return_value=12 * 1024**2),
            recommended_max_memory=Mock(return_value=10 * 1024**2),
        )
        torch = SimpleNamespace(
            cuda=SimpleNamespace(is_available=lambda: False),
            backends=SimpleNamespace(mps=SimpleNamespace(is_available=lambda: available)),
            mps=mps,
        )
        with patch.dict(sys.modules, {"torch": torch, "pynvml": None}):
            sampler = GPUMetricsSampler()
        return sampler, mps

    def test_process_memory_semantics_and_unknown_measurements(self):
        sampler, _ = self.sampler()
        sample = sampler._collect_sample()
        self.assertEqual(sample["source"], "torch.mps")
        self.assertEqual(sample["gpu_memory_used_mb"], 12)
        self.assertEqual(sample["gpu_tensor_memory_mb"], 2)
        self.assertEqual(sample["gpu_memory_total_mb"], 10)
        self.assertEqual(sample["gpu_memory_percent"], 120)
        self.assertEqual(sample["memory_total_basis"], "recommended_working_set")
        self.assertIsNone(sample["gpu_utilization_percent"])
        self.assertIsNone(sample["temperature_celsius"])
        self.assertNotIn("gpu_memory_peak_mb", sample)

    def test_unavailable_mps_does_not_simulate(self):
        sampler, _ = self.sampler(False)
        self.assertIsNone(sampler._collect_sample())

    def test_metric_api_failure_does_not_break_training(self):
        sampler, mps = self.sampler()
        mps.driver_allocated_memory.side_effect = RuntimeError("unavailable")
        self.assertIsNone(sampler._collect_sample())

    def test_invalid_denominator_is_not_reported(self):
        sampler, mps = self.sampler()
        mps.recommended_max_memory.return_value = 0
        self.assertIsNone(sampler._collect_sample())
