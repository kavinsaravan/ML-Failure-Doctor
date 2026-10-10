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
  torch=SimpleNamespace(cuda=SimpleNamespace(is_available=lambda:False),backends=SimpleNamespace(mps=SimpleNamespace(is_available=lambda:False)))
  with patch.dict(sys.modules,{'torch':torch}),patch.object(collector,'request') as request:
   for device in ('cuda','mps'):
    with self.assertRaisesRegex(RuntimeError,'fallback refused'):collector.child('checkpoint',device,'unused')
   request.assert_not_called()
 def test_mps_cpu_operation_fallback_is_refused(self):
  torch=SimpleNamespace(backends=SimpleNamespace(mps=SimpleNamespace(is_available=lambda:True)))
  with patch.dict(sys.modules,{'torch':torch}),patch.dict('os.environ',{'PYTORCH_ENABLE_MPS_FALLBACK':'1'}),patch.object(collector,'request') as request:
   with self.assertRaisesRegex(RuntimeError,'Unset'):collector.child('checkpoint','mps','unused')
   request.assert_not_called()
if __name__=='__main__':unittest.main()
