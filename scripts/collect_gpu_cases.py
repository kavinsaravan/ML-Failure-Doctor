"""Collect actual tracked GPU workload failures, refusing CPU/simulated telemetry."""
import argparse,json,os,socket,subprocess,sys,tempfile,time,uuid
from pathlib import Path
import urllib.request
ROOT=Path(__file__).resolve().parents[1]
sys.path.insert(0,str(ROOT/'crashlens-sdk'))
SCENARIOS={
 'allocation-256': ('GPU_OUT_OF_MEMORY','executed_gpu_allocator_error','GPU allocation exceeds the configured per-process memory budget.','Reduce allocation or batch size to fit the budget; do not disable safety limits.'),
 'allocation-384': ('GPU_OUT_OF_MEMORY','executed_gpu_allocator_error','GPU allocation exceeds the configured per-process memory budget.','Reduce allocation or batch size to fit the budget; do not disable safety limits.'),
 'matmul-shape': ('UNKNOWN_ERROR','executed_gpu_operation_error','GPU tensor matrix multiplication has incompatible inner dimensions.','Correct tensor dimensions before matrix multiplication.'),
 'reshape-size': ('UNKNOWN_ERROR','executed_gpu_operation_error','Requested GPU tensor reshape has a different element count.','Preserve tensor element count when reshaping.'),
 'dtype-or-device': ('UNKNOWN_ERROR','executed_gpu_operation_error','MPS cannot represent float64 tensors, or CUDA operation mixes CPU and GPU tensors.','Use float32 on MPS or move both operands to the same CUDA device.'),
 'checkpoint': ('MISSING_CHECKPOINT','executed_application_error_after_gpu_work','Required resume checkpoint does not exist.','Provide the checkpoint or start a new run without resuming.'),
 'dataset': ('DATA_PATH_ERROR','executed_application_error_after_gpu_work','Required dataset file does not exist.','Correct the path or provide the dataset.'),
 'dependency': ('DEPENDENCY_ERROR','executed_application_error_after_gpu_work','A deliberately absent Python package cannot be imported.','Install/provide the required package in the workload environment.'),
}

def request(url,key,path,body=None):
 import requests
 response=requests.request('GET' if body is None else 'POST',url+path,headers={'Authorization':'Bearer '+key},json=body,timeout=15)
 response.raise_for_status();return response.json()

def child(case,device,out):
 import torch
 from crashlens import WorkloadTracker
 if device=='cuda' and not torch.cuda.is_available(): raise RuntimeError('CUDA/ROCm GPU required; CPU fallback refused')
 if device=='mps' and not torch.backends.mps.is_available(): raise RuntimeError('MPS GPU required; CPU fallback refused')
 if device=='mps' and os.getenv('PYTORCH_ENABLE_MPS_FALLBACK')=='1': raise RuntimeError('Unset PYTORCH_ENABLE_MPS_FALLBACK')
 url,key=os.environ['CRASHLENS_URL'],os.environ['CRASHLENS_API_KEY']
 tracker=WorkloadTracker(url,api_key=key,device=device if device=='mps' else 'cuda:0')
 sync=torch.mps.synchronize if device=='mps' else torch.cuda.synchronize
 ident=None;live=False;trained=False;error=None
 try:
  with tracker.track('Hardware evaluation '+case) as ident:
   model=torch.nn.Linear(64,32).to(device)
   x=torch.randn(16,64,device=device)
   optimizer=torch.optim.SGD(model.parameters(),lr=.01)
   for _ in range(4):
    optimizer.zero_grad();loss=model(x).square().mean();loss.backward();optimizer.step();sync()
   trained=all(p.device.type==device for p in model.parameters()) and x.device.type==device
   print('Verified GPU forward/backward/optimizer steps on',device,flush=True)
   deadline=time.monotonic()+12
   while time.monotonic()<deadline:
    time.sleep(.5)
    stored=request(url,key,f'/workloads/{ident}')
    samples=json.loads(stored.get('gpu_metrics') or '[]')
    sources={'torch.mps'} if device=='mps' else {'torch.cuda','nvidia-ml-py'}
    if stored['status']=='running' and any(s.get('source') in sources and s.get('gpu_memory_used_mb',0)>0 for s in samples):
     live=True;break
   if not live:raise RuntimeError('Real GPU telemetry did not reach API while running')
   if case.startswith('allocation-'):
    budget_mib=int(case.split('-')[1])
    if device=='mps':
     used=torch.mps.driver_allocated_memory()
     budget=max(budget_mib*1024**2,used+64*1024**2)
     recommended=torch.mps.recommended_max_memory()
     if budget/recommended>=.5:raise RuntimeError('Device memory budget too small for bounded test')
     torch.mps.set_per_process_memory_fraction(budget/recommended)
     requested=budget+16*1024**2
    else:
     total=torch.cuda.get_device_properties(0).total_memory
     requested=total+budget_mib*1024**2
    print('Requesting oversized allocation of',requested,'bytes',flush=True)
    tensor=torch.empty(requested,dtype=torch.uint8,device=device)
    sync()
   elif case=='matmul-shape': torch.matmul(x,torch.randn(65,32,device=device))
   elif case=='reshape-size': x.reshape(7,7)
   elif case=='dtype-or-device':
    if device=='mps':x.to(torch.float64)
    else:x+torch.randn(16,64,device='cpu')
   elif case=='checkpoint':open('absent_resume_checkpoint.pt','rb')
   elif case=='dataset':open('absent_training_dataset.csv','rb')
   elif case=='dependency':__import__('crashlens_benchmark_absent_dependency')
 except Exception as e:error=type(e).__name__
 if ident is None or not trained or not live:raise RuntimeError('Incomplete hardware execution; case not exported')
 if error is None:raise RuntimeError('Expected failure did not occur')
 stored=request(url,key,f'/workloads/{ident}')
 if stored['status']!='failed':raise RuntimeError('Failed terminal status not persisted')
 # Ensure the allocator test produced an actual OOM, not another Python exception.
 if case.startswith('allocation-') and 'out of memory' not in stored.get('job_logs','').lower():raise RuntimeError('Allocator test failed without OOM; case not exported')
 label,provenance,cause,fix=SCENARIOS[case]
 samples=json.loads(stored.get('gpu_metrics') or '[]')
 name=torch.cuda.get_device_name(0) if device=='cuda' else 'Apple MPS'
 Path(out).write_text(json.dumps(dict(id=f'{device}-{case}',provenance=provenance,expected_failure_type=label,
  logs=stored['job_logs'],gpu_metrics=stored.get('gpu_metrics',''),expected_root_cause=cause,acceptable_remediation=fix,
  hardware=dict(device=device,name=name,torch_version=torch.__version__,gpu_training_verified=trained,live_telemetry_verified=live,
   samples=len(samples),workload_id=ident,exception_class=error)),indent=2))
 print('Exported real tracked case:',case)

def main():
 parser=argparse.ArgumentParser(description=__doc__)
 parser.add_argument('--device',choices=['mps','cuda'],required=True)
 parser.add_argument('--api-url',default=os.getenv('CRASHLENS_URL'))
 parser.add_argument('--local-backend',action='store_true',help='Build/run a disposable isolated API; never touches the deployment database')
 parser.add_argument('--out',type=Path,default=ROOT/'evaluation/hardware-cases.json')
 parser.add_argument('--child',choices=list(SCENARIOS),help=argparse.SUPPRESS)
 args=parser.parse_args()
 args.out=args.out.resolve()
 if args.child:child(args.child,args.device,args.out);return
 if args.out.exists():parser.error('Output exists; choose a new dataset filename')
 if not args.local_backend and not args.api_url:parser.error('Choose --api-url or --local-backend')
 with tempfile.TemporaryDirectory(prefix='crashlens-hardware-') as directory:
  directory=Path(directory);process=None
  try:
   env=os.environ.copy()
   if args.local_backend:
    binary=directory/'backend'
    subprocess.run(['go','build','-o',str(binary),'.'],cwd=ROOT/'backend',check=True,env={**env,'GOCACHE':'/private/tmp/crashlens-review-go-cache'})
    with socket.socket() as sock:sock.bind(('127.0.0.1',0));port=sock.getsockname()[1]
    url=f'http://127.0.0.1:{port}'
    backendenv={**env,'PORT':str(port),'DATABASE_PATH':str(directory/'isolated.db'),'CRASHLENS_API_KEY':uuid.uuid4().hex,
     'FIREWORKS_API_KEY':'','FIREWORKS_MODEL':'','APP_ENV':'production','ACCESS_MODE':'private','ALLOW_WORKSPACE_CREATION':'true'}
    process=subprocess.Popen([str(binary)],cwd=ROOT,env=backendenv,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
    for _ in range(100):
     try:
      with urllib.request.urlopen(url+'/health',timeout=1):break
     except Exception:time.sleep(.1)
    else:raise RuntimeError('Isolated backend did not start')
    key=request(url,'','/workspaces',{'name':'Hardware evaluation'})['api_key']
   else:
    url=args.api_url.rstrip('/');key=env.get('CRASHLENS_API_KEY')
    if not key:raise RuntimeError('Set CRASHLENS_API_KEY to a personal workspace key')
   env.update(CRASHLENS_URL=url,CRASHLENS_API_KEY=key)
   cases=[]
   for case in SCENARIOS:
    result=directory/(case+'.json')
    subprocess.run([sys.executable,str(Path(__file__).resolve()),'--child',case,'--device',args.device,'--out',str(result)],cwd=directory,env=env,timeout=90,check=True)
    cases.append(json.loads(result.read_text()))
    # Save every completed case; a failed later test cannot erase collected evidence.
    args.out.parent.mkdir(parents=True,exist_ok=True)
    args.out.write_text(json.dumps(dict(schema_version=1,description='Controlled failures after verified GPU forward/backward/optimizer computation and live telemetry; GPU allocator/operation errors are distinguished from application failures.',cases=cases),indent=2)+'\n')
   print('Collected',len(cases),'actual GPU workload cases to',args.out)
  finally:
   if process:
    process.terminate()
    try:process.wait(timeout=15)
    except subprocess.TimeoutExpired:process.kill();process.wait()
if __name__=='__main__':main()
