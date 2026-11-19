# CrashLens Python SDK

Wrap Python training code to capture logs, runtime, failures, and available GPU
metrics. See the [main documentation](../README.md) for backend setup, dashboard
usage and authentication.

## Installation

From the repository root:

```bash
python -m pip install -e ./crashlens-sdk
```

Or install directly from GitHub:

```bash
python -m pip install "git+https://github.com/kavinsaravan/ML-Failure-Doctor.git#subdirectory=crashlens-sdk"
```

Optional extras are `gpu` (NVIDIA NVML), `jupyter` (notebook integration), and
`all` (both). For example, use `python -m pip install -e "./crashlens-sdk[all]"`.
Install GPU-compatible PyTorch separately for CUDA or ROCm telemetry.

## Track a workload

Create a workspace in the dashboard and save its key. Set `CRASHLENS_API_KEY`
to that workspace key, then wrap your training. The SDK connects to the hosted
CrashLens backend automatically:

```python
import os
from crashlens import WorkloadTracker

tracker = WorkloadTracker(
    api_key=os.environ["CRASHLENS_API_KEY"],
    fireworks_api_key=os.environ.get("FIREWORKS_API_KEY"),
    fireworks_model=os.environ.get("FIREWORKS_MODEL"),
)

with tracker.track("Training run") as workload_id:
    train_model()  # Replace with your training code.
```

For self-hosted deployments, set `CRASHLENS_URL` or pass `api_url=` explicitly.
URL selection is: explicit argument → `CRASHLENS_URL` →
`https://ml-failure-doctor-production.up.railway.app`.
`JupyterWorkloadTracker`, `crashlens.init()`, and `%crashlens_init` use the same default.

Use `@tracker.track_function("Training run")` to wrap a synchronous function.
After a failed run, call `tracker.diagnose(workload_id)` or diagnose it in the
dashboard. Fireworks model selection defaults to the `auto` serverless router;
set `fireworks_model` to override it. Reports retain the requested router/model
and the serving model returned by Fireworks when available. See
[Backend Diagnosis](../README.md#backend-diagnosis) for credential, fallback,
and caching behavior.

## Tracking behavior and limits

- Logs and runtime upload every two seconds by default; change `upload_interval`
  on the tracker to adjust uploads. GPU sampling remains every two seconds.
- The SDK retains the last 256 KiB of captured output and 300 GPU samples.
- Only one tracking context can run per Python process. Use separate processes
  for concurrent runs; stdout/stderr capture includes other threads' output.
- If backend registration fails, training continues and the live publisher
  retries. Registration and telemetry requests use three-second timeouts. The
  context yields `None` until a remote ID is available; after the run, inspect
  `tracker.last_workload_id` for an ID recovered during delivery.
- Terminal uploads retry transient connection errors, timeouts, HTTP 408/429,
  and server errors up to three times, with short backoff. Other HTTP errors
  are retained for recovery without immediate retries. Training exceptions
  still propagate, and telemetry failures do not fail successful training.
- Without supported GPU telemetry, logs and runtime still work; the SDK does
  not generate simulated metrics.
- Set `device="cuda:1"` on the tracker to select telemetry. PyTorch uses the
  `cuda` namespace on both NVIDIA CUDA and AMD ROCm builds.
  Otherwise, it detects the GPU and captures PyTorch's current CUDA device when
  available. This does not move training to that device. CUDA visibility is
  respected through UUID mapping, with PyTorch telemetry as a fallback.
- One GPU is monitored per run; distributed multi-GPU aggregation is unsupported.
  NVIDIA NVML can report utilization and temperature; the PyTorch CUDA/ROCm
  fallback reports memory and leaves unavailable utilization/temperature empty.

## Recover pending uploads

Before terminal delivery, the SDK saves logs, metrics, timestamps, and status
in a local outbox; successful delivery removes the file. The default is
`~/.cache/crashlens/outbox` (or beneath
`XDG_CACHE_HOME`). Set `CRASHLENS_OUTBOX_DIR` or the tracker's `outbox_dir` argument
to choose a persistent directory, especially in temporary notebook environments.
Files have private permissions and do not include SDK or Fireworks credentials.
They contain captured workload output, so retain them on an appropriate local disk.

Once connected, use the same backend URL, workspace key, and outbox directory:

```python
delivered = tracker.flush_pending()  # Local run IDs mapped to backend workload IDs.
```

Delivered files are removed; unsent files remain for another attempt, including
after a Python restart. Replay sends telemetry only, without requesting an AI
diagnosis. A persistent `client_run_id` prevents duplicate records if a creation
response was lost; use the updated backend with this SDK. During an outage the
dashboard may remain stale until delivery or replay succeeds. This terminal
outbox does not recover telemetry from a process killed before cleanup executes.

## Jupyter

Install the `jupyter` extra and use `JupyterWorkloadTracker` from
`crashlens.jupyter` for rich displays. Its `track()` method accepts
`auto_diagnose=True`; automatic diagnosis is off by default.

Alternatively, load `%load_ext crashlens.jupyter`, initialize with
`%crashlens_init <backend-url>`, and track cells with
`%%crashlens_track "Training run"`. The extension reads credentials from the
environment variables above and automatically diagnoses failed tracked cells.

See the [example notebooks](examples/notebooks) for complete workflows.

## Tests

After installing the `jupyter` extra, run from the repository root:

```bash
PYTHONPATH=crashlens-sdk python -m unittest discover -s crashlens-sdk/tests
```

GPU collectors are mocked in unit tests. To check actual hardware and live
uploads, run `python scripts/validate_gpu.py` on the machine running your training.
