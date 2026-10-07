# Jupyter tracking

From the repository root install `python -m pip install -e './crashlens-sdk[jupyter]'`.
Create a workspace in the dashboard and save its key. Configure credentials via
secure prompts or environment variables; do not paste them into notebook source.

```python
import os
from crashlens.jupyter import JupyterWorkloadTracker
tracker = JupyterWorkloadTracker(
    os.environ["CRASHLENS_URL"], api_key=os.environ["CRASHLENS_API_KEY"],
    fireworks_api_key=os.environ.get("FIREWORKS_API_KEY"),
    fireworks_model=os.environ.get("FIREWORKS_MODEL"),
)
with tracker.track("Training", display_metrics=True, auto_diagnose=False) as workload_id:
    train_model()
```

`auto_diagnose=True` requests a report after a failure. With valid Fireworks
credentials this may consume your credits; without them user workloads get rules
reports. To request one explicitly, call `tracker.diagnose(workload_id)`.
Use `tracker.show_workload(workload_id)` for stored workload details.

## IPython magic

Set CRASHLENS_URL, CRASHLENS_API_KEY, and optional FIREWORKS_API_KEY/FIREWORKS_MODEL
in the notebook process environment first, then:

```python
%load_ext crashlens.jupyter
%crashlens_init https://YOUR-BACKEND
```

```python
%%crashlens_track "Training"
train_model()
```

The extension reads keys from environment variables, not magic arguments. Cell
syntax and runtime failures are recorded. Use `%crashlens_diagnose WORKLOAD_ID`
and `%crashlens_show WORKLOAD_ID` with the actual ID returned by your run, not a
hardcoded ID from another workspace. Kernel kills cannot be reported by the SDK.

Metrics come from the shared SDK collector; available fields depend on CUDA,
ROCm, or MPS. No GPU means no invented samples. CPU computations are real CPU
workloads and do not validate a GPU. See [SDK](../crashlens-sdk/README.md).
