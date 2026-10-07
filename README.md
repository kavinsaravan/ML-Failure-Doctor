# CrashLens

CrashLens tracks workload logs, runtime, and GPU metrics, then produces
rule-based or optional Fireworks AI failure diagnoses. Workloads run on the user's
machine; the hosted API and dashboard store and display the results.

## What is implemented

- Private workspaces accessed with individual, revocable API keys. Visitors can
  create a workspace without email or password and save their key once.
- SDK telemetry uploads every two seconds; the dashboard polls every three seconds.
- NVIDIA telemetry through NVML or PyTorch CUDA; AMD memory through PyTorch ROCm;
  Apple process Metal memory through PyTorch MPS.
- Failure classification, evidence, recommended fixes, and persisted reports.
- Request-scoped user Fireworks credentials for AI diagnosis. User workloads never
  fall back to the operator's Fireworks account.
- Jupyter context tracking and IPython magic commands.
- Seven read-only MCP tools for querying the connected workspace.
- Docker deployment with persistent SQLite, bounded template execution, deadlines,
  and recovery of interrupted backend-managed jobs.

This is a prototype. The diagnosis score is a heuristic log-match score, not
measured AI accuracy. Failed-job GPU seconds are a runtime estimate, not a bill.
NVIDIA and Apple MPS have been exercised on real hardware; AMD hardware validation
remains pending. Supported metrics vary by source. SDK device targeting is currently
CUDA/ROCm device 0 or the process's MPS allocations.

## Run locally or deploy

```bash
cp .env.example .env
# Configure .env; set an operator CRASHLENS_API_KEY for private hosting.
docker compose up --build -d
```

Dashboard: `http://localhost:3000/dashboard`. Backend: `http://localhost:8080`.
For production set `APP_ENV=production`, `ACCESS_MODE=private`, a random operator
key, the public HTTPS `NEXT_PUBLIC_API_URL`, and exact `ALLOWED_ORIGINS`.
The frontend API URL is embedded at build time. Never put private keys in
`NEXT_PUBLIC_*` variables. SQLite lives in `crashlens-data`; preserve that volume
and run one backend instance per database. See [deployment](PRODUCTION_SETUP.md).

The dashboard's **Simulated Example Jobs** print predefined logs and errors on
the backend. They do not train models or validate the visitor's GPU. If backend
hardware collection is unavailable, demo metrics are explicitly tagged `Simulated`.
The SDK never invents GPU readings.

## Connect a real workload

1. Open the dashboard, select **Create my workspace**, and save the generated key.
   Labels are descriptive; identical labels still create separate workspaces.
2. Install the SDK on the machine that runs the workload:

```bash
python -m pip install "git+https://github.com/kavinsaravan/ML-Failure-Doctor.git#subdirectory=crashlens-sdk"
# Or, from a checkout:
python -m pip install -e ./crashlens-sdk
```

For NVML telemetry install the `gpu` extra. For notebooks use `jupyter`; `all`
includes both. Install a GPU-compatible PyTorch separately for CUDA, ROCm, or MPS.
The SDK monitors workloads; training code must explicitly select its device.
Run Apple GPU workloads natively on macOS, not in Docker Desktop's Linux VM.

```python
import os
from crashlens import WorkloadTracker

tracker = WorkloadTracker(
    os.environ["CRASHLENS_URL"],
    api_key=os.environ["CRASHLENS_API_KEY"],
    fireworks_api_key=os.environ.get("FIREWORKS_API_KEY"),
    fireworks_model=os.environ.get("FIREWORKS_MODEL"),
)
with tracker.track("My training run") as workload_id:
    train_model()
# After a failure, explicitly request a report:
# report = tracker.diagnose(workload_id)
```

Use the same workspace key in the SDK and dashboard. SDK captures Python
exceptions, interrupts, and nonzero SystemExit; it cannot report SIGKILL, kernel
crashes, or machine loss. Reporting errors are logged without hiding training errors.
See [SDK](crashlens-sdk/README.md), [Jupyter](docs/JUPYTER_INTEGRATION.md),
[examples](examples/README.md), and [MCP](mcp-server/README.md).

## Fireworks diagnosis and billing

Users enter their own Fireworks key and exact tool-capable model identifier in the
dashboard, or provide them to the SDK. Serverless models need no dedicated GPU
deployment. The selected model must be available to that user's account.
New diagnoses send bounded logs and metrics to Fireworks using the user's key;
`X-Fireworks-API-Key` and `X-Fireworks-Model` are sent only on diagnosis requests.
Use HTTPS and a backend you trust. Keys remain in browser/process memory, not
SQLite or localStorage, and are not included in reports or server logs.

Missing credentials or unusable AI results produce a rules report. User requests
never fall back to operator credits. Saved AI reports are reused; **Re-run
Diagnosis** requests a fresh report and can incur charges. Your operator key may
use backend `FIREWORKS_API_KEY` and `FIREWORKS_MODEL` for the legacy workspace;
there is no hardcoded model or private deployment default.

## Access and operations

The operator `CRASHLENS_API_KEY` manages user keys and retains preexisting
workloads in the legacy workspace. It has no cross-owner workload access.
Public creation returns a random key once; only its SHA-256 hash is stored.
There is no self-service lost-key recovery. Sharing a key shares the workspace.
Ownership checks cover lists, summaries, logs, metrics, updates, diagnoses,
backend template runs, and deletion. Users cannot select ownership in payloads.

```bash
python scripts/manage_keys.py --api-url https://YOUR-BACKEND issue Alice
python scripts/manage_keys.py --api-url https://YOUR-BACKEND list
python scripts/manage_keys.py --api-url https://YOUR-BACKEND revoke KEY_ID
# Preserve ownership when replacing a key:
python scripts/manage_keys.py --api-url https://YOUR-BACKEND issue Alice --owner-id OWNER_ID
```

Set `ALLOW_WORKSPACE_CREATION=false` to disable public creation. Public creation
is limited per backend process to 3 requests/minute per IP (burst 1) and 10/minute
globally (burst 5). Once individual keys exist, anonymous demo reads are disabled.
Use explicit `TRUSTED_PROXY_CIDRS` only for known proxies; forwarded headers are
otherwise ignored. Template jobs default to 2 workers, 16 queued jobs, and a
300-second deadline through `JOB_CONCURRENCY`, `JOB_QUEUE_SIZE`, and
`JOB_TIMEOUT_SECONDS`. Diagnosis spending quotas are not implemented.

## Validation

```bash
(cd backend && go test -race ./... && go vet ./...)
(cd frontend && npm run lint && npm test && npx tsc --noEmit --incremental false)
(cd mcp-server && npm test)
python -m pip install -e './crashlens-sdk[jupyter]'
PYTHONPATH=crashlens-sdk python -m unittest discover -s crashlens-sdk/tests -v
python scripts/docker_smoke.py
```

On a real GPU machine, set `CRASHLENS_URL` and `CRASHLENS_API_KEY`, then run:

```bash
python scripts/validate_gpu.py             # CUDA/ROCm/MPS auto-detection
python scripts/validate_gpu.py --device mps
```

This trains for ten seconds and requires real telemetry at the API while running;
it refuses CPU/simulation. MPS memory is process Metal driver allocation, including
caches, relative to the recommended working set—not physical VRAM. Ratios may
exceed 100%. Utilization, temperature, and an exact MPS allocation peak are unavailable.
Sampling may miss brief allocations. Synthetic training data is generated locally;
it is not a real-world diagnosis evaluation dataset.
