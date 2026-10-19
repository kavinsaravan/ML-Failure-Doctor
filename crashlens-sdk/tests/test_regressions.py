import sys
import threading
import unittest
from types import SimpleNamespace as NS
from unittest.mock import Mock, patch
from crashlens.metrics import GPUMetricsSampler
from crashlens.workload_tracker import WorkloadTracker
from crashlens.jupyter import JupyterWorkloadTracker

class RegressionTests(unittest.TestCase):
    def test_jupyter_byok(self):
        t = JupyterWorkloadTracker('http://test', api_key='workspace', fireworks_api_key='fireworks', fireworks_model='model')
        with patch('requests.post') as post:
            post.return_value.json.return_value = {'source': 'ai'}
            self.assertEqual(t.diagnose(1, display=False), {'source': 'ai'})
            self.assertEqual(post.call_args.kwargs['headers']['X-Fireworks-API-Key'], 'fireworks')
            self.assertEqual(post.call_args.kwargs['headers']['X-Fireworks-Model'], 'model')
            self.assertNotIn('X-Fireworks-API-Key', t.headers)

    def test_overlap_and_setup_failure_release(self):
        t = WorkloadTracker('http://test')
        other = WorkloadTracker('http://test')
        with patch.object(t, '_create_workload', return_value=1), patch.object(t, '_update_workload'), patch('crashlens.workload_tracker.METRICS_AVAILABLE', False), patch.object(other, '_create_workload') as create:
            old = sys.stdout
            errors = []
            def overlap():
                try:
                    with other.track('overlap'): pass
                except RuntimeError as e: errors.append(str(e))
            with t.track('first'):
                thread = threading.Thread(target=overlap); thread.start(); thread.join()
                with self.assertRaises(RuntimeError):
                    with t.track('nested'): pass
            self.assertEqual(len(errors), 1)
            create.assert_not_called()
            self.assertIs(sys.stdout, old)
            with t.track('next'): pass
        with patch.object(t, '_create_workload', side_effect=ValueError('offline')):
            with self.assertRaises(ValueError):
                with t.track('offline'): pass
        with patch.object(t, '_create_workload', return_value=1), patch.object(t, '_update_workload'), patch('crashlens.workload_tracker.METRICS_AVAILABLE', False):
            with t.track('recovered'): pass

    def torch(self, uuid='GPU-selected'):
        cuda = Mock()
        cuda.is_available.return_value = True
        cuda.current_device.return_value = 1
        cuda.get_device_properties.return_value = NS(uuid=uuid, name='Selected GPU')
        cuda.mem_get_info.return_value = (1024, 2048)
        cuda.max_memory_allocated.return_value = 512
        return NS(cuda=cuda)

    def test_cuda_visibility_uses_uuid(self):
        torch = self.torch(); nvml = Mock()
        nvml.nvmlDeviceGetUUID.return_value = 'GPU-selected'
        nvml.nvmlDeviceGetName.return_value = 'Selected GPU'
        with patch.dict(sys.modules, {'torch': torch, 'pynvml': nvml}), patch.dict('os.environ', {'CUDA_VISIBLE_DEVICES': '3,2'}):
            sampler = GPUMetricsSampler()
        nvml.nvmlDeviceGetHandleByUUID.assert_called_once_with('GPU-selected')
        nvml.nvmlDeviceGetHandleByIndex.assert_not_called()
        self.assertEqual(sampler.device_index, 1)

    def test_unknown_uuid_falls_back_to_selected_torch_device(self):
        torch = self.torch(None); nvml = Mock()
        with patch.dict(sys.modules, {'torch': torch, 'pynvml': nvml}):
            sampler = GPUMetricsSampler(device='cuda:2')
        sample = sampler._collect_sample()
        torch.cuda.mem_get_info.assert_called_once_with(2)
        torch.cuda.max_memory_allocated.assert_called_once_with(2)
        nvml.nvmlDeviceGetHandleByIndex.assert_not_called()
        self.assertEqual(sample['device_index'], 2)
        self.assertEqual(sample['source'], 'torch.cuda')

    def test_visibility_without_torch_does_not_guess(self):
        nvml = Mock()
        with patch.dict(sys.modules, {'torch': None, 'pynvml': nvml}), patch.dict('os.environ', {'CUDA_VISIBLE_DEVICES': '2'}):
            sampler = GPUMetricsSampler()
        self.assertIsNone(sampler._collect_sample())
        nvml.nvmlDeviceGetHandleByIndex.assert_not_called()
