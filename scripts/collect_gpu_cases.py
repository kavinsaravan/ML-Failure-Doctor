"""Collect actual tracked GPU workload failures, refusing CPU/simulated telemetry."""
import argparse,json,os,socket,subprocess,sys,tempfile,time,uuid
from pathlib import Path
import urllib.request
ROOT=Path(__file__).resolve().parents[1]
sys.path.insert(0,str(ROOT/'crashlens-sdk'))
ORIGINAL_SCENARIOS={
 'allocation-256': ('GPU_OUT_OF_MEMORY','executed_gpu_allocator_error','A single allocation exceeds the device total memory capacity.','Reduce allocation or batch size to fit available GPU memory.'),
 'allocation-384': ('GPU_OUT_OF_MEMORY','executed_gpu_allocator_error','A single allocation exceeds the device total memory capacity.','Reduce allocation or batch size to fit available GPU memory.'),
 'matmul-shape': ('UNKNOWN_ERROR','executed_gpu_operation_error','GPU tensor matrix multiplication has incompatible inner dimensions.','Correct tensor dimensions before matrix multiplication.'),
 'reshape-size': ('UNKNOWN_ERROR','executed_gpu_operation_error','Requested GPU tensor reshape has a different element count.','Preserve tensor element count when reshaping.'),
 'dtype-or-device': ('UNKNOWN_ERROR','executed_gpu_operation_error','The operation mixes CPU and GPU tensors.','Move both operands to the same CUDA/ROCm device.'),
 'checkpoint': ('MISSING_CHECKPOINT','executed_application_error_after_gpu_work','Required resume checkpoint does not exist.','Provide the checkpoint or start a new run without resuming.'),
 'dataset': ('DATA_PATH_ERROR','executed_application_error_after_gpu_work','Required dataset file does not exist.','Correct the path or provide the dataset.'),
 'dependency': ('DEPENDENCY_ERROR','executed_application_error_after_gpu_work','A deliberately absent Python package cannot be imported.','Install/provide the required package in the workload environment.'),
}
EXTENDED_SCENARIOS={
 'gradient-disabled': ('UNKNOWN_ERROR','executed_gpu_autograd_error','The forward pass ran under torch.no_grad, so its loss has no autograd graph.','Recompute the forward pass with gradient tracking enabled before backward; merely marking the detached loss requires_grad does not reconnect model parameters.'),
 'non-scalar-gradient': ('UNKNOWN_ERROR','executed_gpu_autograd_error','Backward was called on a multi-element output without an explicit upstream gradient.','Reduce the output to a scalar loss or supply a correctly shaped upstream gradient.'),
 'inplace-leaf-update': ('UNKNOWN_ERROR','executed_gpu_autograd_error','A leaf tensor requiring gradients was modified in place while autograd was enabled.','Use an out-of-place operation, an optimizer update, or no_grad for an intentional non-differentiable parameter initialization/update.'),
 'empty-optimizer': ('UNKNOWN_ERROR','executed_application_error_after_gpu_work','A new optimizer was constructed with an empty parameter list.','Pass the intended model parameters to the optimizer; do not remove parameter validation.'),
 'invalid-learning-rate': ('UNKNOWN_ERROR','executed_application_error_after_gpu_work','The optimizer was configured with a negative learning rate.','Choose a valid non-negative learning rate; use a positive rate for actual learning.'),
 'corrupt-checkpoint': ('UNKNOWN_ERROR','executed_application_error_after_gpu_work','The checkpoint exists but contains deliberately invalid non-PyTorch bytes.','Restore or regenerate a valid checkpoint and validate it; keep safe loading enabled rather than bypassing it for untrusted files.'),
 'csv-column-missing': ('UNKNOWN_ERROR','executed_application_error_after_gpu_work','CSV data has feature and label columns, but the code requests an absent target column.','Use the actual label column or explicitly map/validate the required schema; do not silently fabricate labels.'),
 'nonfinite-loss': ('UNKNOWN_ERROR','executed_gpu_numerical_error','A zero divided by zero computation produced NaN, caught before an optimizer update.','Prevent zero-denominator division using a mathematically justified guard or stable formulation and validate finite losses; do not suppress the check or blindly continue.'),
}
SCENARIOS={**ORIGINAL_SCENARIOS,**EXTENDED_SCENARIOS}

def request(url,key,path,body=None):
 import requests
 response=requests.request('GET' if body is None else 'POST',url+path,headers={'Authorization':'Bearer '+key},json=body,timeout=15)
 response.raise_for_status();return response.json()

def child(case,device,out,trial=1):
 if device!='cuda':raise ValueError('Use device=cuda for NVIDIA CUDA or AMD ROCm')
 import torch
 from crashlens import WorkloadTracker
 if device=='cuda' and not torch.cuda.is_available(): raise RuntimeError('CUDA/ROCm GPU required; CPU fallback refused')
 url,key=os.environ['CRASHLENS_URL'],os.environ['CRASHLENS_API_KEY']
 tracker=WorkloadTracker(url,api_key=key,device='cuda:0',upload_interval=.5)
 sync=torch.cuda.synchronize
 ident=None;live=False;trained=False;error=None
 try:
  with tracker.track(f'Hardware evaluation {case}, trial {trial}') as ident:
   width,batch=64+trial,16+trial
   print(f'GPU execution: scenario={case}, trial={trial}, width={width}, batch={batch}',flush=True)
   model=torch.nn.Linear(width,32).to(device)
   x=torch.randn(batch,width,device=device)
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
    sources={'torch.cuda','nvidia-ml-py'}
    if stored['status']=='running' and any(s.get('source') in sources and s.get('gpu_memory_used_mb',0)>0 for s in samples):
     live=True;break
   if not live:raise RuntimeError('Real GPU telemetry did not reach API while running')
   if case.startswith('allocation-'):
    budget_mib=int(case.split('-')[1])
    total=torch.cuda.get_device_properties(0).total_memory
    requested=total+budget_mib*1024**2
    print('Requesting oversized allocation of',requested,'bytes',flush=True)
    tensor=torch.empty(requested,dtype=torch.uint8,device=device)
    sync()
   elif case=='matmul-shape': torch.matmul(x,torch.randn(width+1,32,device=device))
   elif case=='reshape-size': x.reshape(7,7)
   elif case=='dtype-or-device':x+torch.randn(batch,width,device='cpu')
   elif case=='checkpoint':open('absent_resume_checkpoint.pt','rb')
   elif case=='dataset':open('absent_training_dataset.csv','rb')
   elif case=='dependency':__import__('crashlens_benchmark_absent_dependency')
   elif case=='gradient-disabled':
    with torch.no_grad():detached_loss=model(x).sum()
    print('Loss forward pass ran under torch.no_grad; loss.requires_grad =',detached_loss.requires_grad,flush=True)
    detached_loss.backward()
   elif case=='non-scalar-gradient':model(x).backward()
   elif case=='inplace-leaf-update':next(model.parameters()).add_(1)
   elif case=='empty-optimizer':torch.optim.Adam([],lr=.001)
   elif case=='invalid-learning-rate':torch.optim.SGD(model.parameters(),lr=-.01)
   elif case=='corrupt-checkpoint':
    Path('corrupt_state.pt').write_bytes(b'not-a-valid-PyTorch-archive')
    print('Existing checkpoint contains invalid bytes, not a saved PyTorch object',flush=True)
    torch.load('corrupt_state.pt',map_location='cpu',weights_only=True)
   elif case=='csv-column-missing':
    import csv,io
    row=next(csv.DictReader(io.StringIO('feature,label\n1,0\n')))
    print('CSV columns:',list(row),'requested column: target',flush=True)
    row['target']
   elif case=='nonfinite-loss':
    zero=torch.tensor(0.,device=device);invalid_loss=zero/zero
    if not torch.isfinite(invalid_loss).item():raise FloatingPointError('Loss is non-finite (NaN) after 0/0 division; stopped before optimizer update')
 except Exception as e:error=type(e).__name__
 if ident is None or not trained or not live:raise RuntimeError('Incomplete hardware execution; case not exported')
 if error is None:raise RuntimeError('Expected failure did not occur')
 stored=request(url,key,f'/workloads/{ident}')
 if stored['status']!='failed':raise RuntimeError('Failed terminal status not persisted')
 # Ensure the allocator test produced an actual OOM, not another Python exception.
 if case.startswith('allocation-') and 'out of memory' not in stored.get('job_logs','').lower():raise RuntimeError('Allocator test failed without OOM; case not exported')
 label,provenance,cause,fix=SCENARIOS[case]
 samples=json.loads(stored.get('gpu_metrics') or '[]')
 name=torch.cuda.get_device_name(0)
 Path(out).write_text(json.dumps(dict(id=f'{device}-{case}-trial-{trial:03}',scenario_family=case,trial=trial,provenance=provenance,expected_failure_type=label,
  logs=stored['job_logs'],gpu_metrics=stored.get('gpu_metrics',''),expected_root_cause=cause,acceptable_remediation=fix,
  hardware=dict(device=device,platform='ROCm' if torch.version.hip else 'CUDA',name=name,torch_version=torch.__version__,gpu_training_verified=trained,live_telemetry_verified=live,
   samples=len(samples),workload_id=ident,exception_class=error,runtime_seconds=stored.get('runtime_seconds'),
   created_at=stored.get('created_at'),finished_at=stored.get('finished_at'),input_width=width,batch_size=batch)),indent=2))
 print('Exported real tracked case:',case)

def main():
 parser=argparse.ArgumentParser(description=__doc__)
 parser.add_argument('--device',choices=['cuda'],required=True,help='PyTorch uses cuda for both NVIDIA CUDA and AMD ROCm')
 parser.add_argument('--api-url',default=os.getenv('CRASHLENS_URL'))
 parser.add_argument('--local-backend',action='store_true',help='Build/run a disposable isolated API; never touches the deployment database')
 parser.add_argument('--out',type=Path,default=ROOT/'evaluation/hardware-cases.json')
 parser.add_argument('--repeats',type=int,default=1,help='Executions per scenario; repeated families are not independent diagnostic cases')
 parser.add_argument('--suite',choices=['original','extended','all'],default='original',help='Original eight families, eight new failure families, or both')
 parser.add_argument('--trial',type=int,default=1,help=argparse.SUPPRESS)
 parser.add_argument('--child',choices=list(SCENARIOS),help=argparse.SUPPRESS)
 args=parser.parse_args()
 args.out=args.out.resolve()
 if args.repeats<1 or args.trial<1:parser.error('repeats and trial must be positive')
 if args.child:child(args.child,args.device,args.out,args.trial);return
 selected=ORIGINAL_SCENARIOS if args.suite=='original' else EXTENDED_SCENARIOS if args.suite=='extended' else SCENARIOS
 if args.out.exists():parser.error('Output exists; choose a new dataset filename')
 if not args.local_backend and not args.api_url:parser.error('Choose --api-url or --local-backend')
 with tempfile.TemporaryDirectory(prefix='crashlens-hardware-') as directory:
  directory=Path(directory);process=None
  try:
   env=os.environ.copy()
   if args.local_backend:
    binary=directory/'backend'
    subprocess.run(['go','build','-o',str(binary),'.'],cwd=ROOT/'backend',check=True,env={**env,'GOCACHE':env.get('GOCACHE') or str(Path(tempfile.gettempdir())/'crashlens-review-go-cache')})
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
   for trial in range(1,args.repeats+1):
    for case in selected:
     result=directory/(case+'.json')
     subprocess.run([sys.executable,str(Path(__file__).resolve()),'--child',case,'--device',args.device,'--trial',str(trial),'--out',str(result)],cwd=directory,env=env,timeout=90,check=True)
     cases.append(json.loads(result.read_text()))
     # Checkpoint every execution, including its true provenance and repeated family.
     args.out.parent.mkdir(parents=True,exist_ok=True)
     args.out.write_text(json.dumps(dict(schema_version=1,description='Controlled failures after verified GPU forward/backward/optimizer computation and live telemetry. Repeated trials establish execution count, not independent diagnostic diversity.',suite=args.suite,scenario_families=len(selected),repeats=args.repeats,cases=cases),indent=2)+'\n')
     print(f'Progress: {len(cases)}/{len(selected)*args.repeats} real GPU executions',flush=True)
   print('Collected',len(cases),'actual GPU workload cases to',args.out)
  finally:
   if process:
    process.terminate()
    try:process.wait(timeout=15)
    except subprocess.TimeoutExpired:process.kill();process.wait()
if __name__=='__main__':main()
