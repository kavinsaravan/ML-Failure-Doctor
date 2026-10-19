import importlib.util
import sys
import unittest
from pathlib import Path
from types import SimpleNamespace
from unittest.mock import patch

path=Path(__file__).resolve().parents[1]/'collect_gpu_cases.py'
spec=importlib.util.spec_from_file_location('collector',path)
collector=importlib.util.module_from_spec(spec);spec.loader.exec_module(collector)
class CollectionGuards(unittest.TestCase):
 def test_cpu_fallback_is_refused_before_network_calls(self):
  torch=SimpleNamespace(cuda=SimpleNamespace(is_available=lambda:False))
  with patch.dict(sys.modules,{'torch':torch}),patch.object(collector,'request') as request:
   with self.assertRaisesRegex(RuntimeError,'fallback refused'):collector.child('checkpoint','cuda','unused')
   request.assert_not_called()
 def test_unsupported_device_is_refused_before_network_calls(self):
  with patch.object(collector,'request') as request:
   with self.assertRaisesRegex(ValueError,'CUDA or AMD ROCm'):collector.child('checkpoint','cpu','unused')
   request.assert_not_called()
if __name__=='__main__':unittest.main()
