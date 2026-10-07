# CrashLens Python SDK

Install from a checkout with `python -m pip install -e ./crashlens-sdk`.
Extras: `gpu` adds NVIDIA NVML, `jupyter` adds notebook dependencies, and `all`
includes both. Install GPU-compatible PyTorch separately for CUDA/ROCm/MPS.

Create a workspace in the dashboard and save its key. Use credentials from
secure prompts or environment variables, not notebook source:

```python
import os
from crashlens import WorkloadTracker
tracker = WorkloadTracker(
    os.environ["CRASHLENS_URL"],
    api_key=os.environ["CRASHLENS_API_KEY"],
    fireworks_api_key=os.environ.get("FIREWORKS_API_KEY"),
    fireworks_model=os.environ.get("FIREWORKS_MODEL"),
)
with tracker.track("Training") as workload_id:
    train_model()
```

Context tracking captures output and reports exceptions. The same tracker can
wrap functions using `@tracker.track_function("Training")`. It uploads live logs,
runtime, and GPU samples every two seconds (`upload_interval` is configurable),
then finalizes the run. Logs retain a bounded tail; metrics retain 300 samples.
Requests have timeouts; reporting failures do not mask training exceptions.
SIGKILL, kernel failure, and machine loss cannot be reported by the SDK itself.

For a failed run, use `tracker.diagnose(workload_id)` to request a report.
New AI reports charge the supplied Fireworks account; without credentials user
workloads receive rules reports. A tool-capable model identifier is required with
a Fireworks key. Credentials are sent only for diagnosis and never added to
workload uploads. Saved AI reports are reused by default.

NVIDIA NVML provides device-0 memory, utilization, and temperature; PyTorch CUDA
and ROCm provide device-0 memory. Apple MPS reports process Metal allocations and
tensor memory relative to the recommended working set, not physical VRAM. MPS
utilization and temperature are unavailable. Without a supported source there are
no GPU samples; the SDK does not simulate them. Move the model and tensors onto
the intended GPU in your training code.

For enhanced notebook displays use `JupyterWorkloadTracker` with the same
credentials. See [Jupyter guide](../docs/JUPYTER_INTEGRATION.md) and
[example notebooks](examples/notebooks/README.md).

Run `scripts/validate_gpu.py` from the repository root for a real hardware and
live-upload check. See the [main README](../README.md) for deployment, privacy,
key management, limits, and regression commands.
