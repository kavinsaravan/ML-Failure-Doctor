# CrashLens Python SDK

Track your GPU workloads, capture failures automatically, and get AI-powered diagnosis when things go wrong.

## What This Does

The CrashLens SDK wraps your ML training code to:

- **Capture logs and errors** automatically when your job fails
- **Collect GPU metrics** (memory, utilization, temperature) during execution
- **Upload live telemetry** to the CrashLens backend every 2 seconds
- **Request AI diagnosis** for failures with root cause analysis and fixes
- **Support Jupyter notebooks** with rich HTML displays and magic commands

## Installation

### From a Cloned Repository

If you have the repository checked out locally:

```bash
pip install -e ./crashlens-sdk
```

### Direct from GitHub

Without cloning:

```bash
pip install git+https://github.com/kavinsaravan/ML-Failure-Doctor.git#subdirectory=crashlens-sdk
```

### With Extras

Install optional dependencies based on your needs:

```bash
# For NVIDIA GPU metrics (nvidia-ml-py)
pip install -e "./crashlens-sdk[gpu]"

# For Jupyter notebook support
pip install -e "./crashlens-sdk[jupyter]"

# For everything
pip install -e "./crashlens-sdk[all]"
```

### PyTorch Setup

Install PyTorch separately for your GPU platform. Use the [official installation selector](https://pytorch.org/get-started/locally/) to get the right command for your CUDA/ROCm environment.

## Quick Start

### 1. Get Your API Key

1. Go to the CrashLens dashboard
2. Create a workspace or connect to an existing one
3. Save your API key securely

### 2. Set Environment Variables

**Never hardcode credentials in your code!** Use environment variables:

```bash
export CRASHLENS_URL="https://your-backend.railway.app"
export CRASHLENS_API_KEY="cl_your_api_key_here"
```

### 3. Track Your Training Code

```python
import os
from crashlens import WorkloadTracker

# Initialize tracker with your credentials
tracker = WorkloadTracker(
    os.environ["CRASHLENS_URL"],
    api_key=os.environ["CRASHLENS_API_KEY"]
)

# Wrap your training code
with tracker.track("BERT Fine-tuning") as workload_id:
    model = train_bert_model()
    evaluate(model)

print(f"Workload ID: {workload_id}")
```

That's it! If your training fails, the SDK automatically:
- Captures the error traceback
- Uploads logs and GPU metrics
- Marks the workload as failed
- Makes it available for diagnosis in the dashboard

## How It Works

### Context Manager (`with tracker.track()`)

The SDK uses Python context managers to wrap your code:

```python
with tracker.track("Job Name") as workload_id:
    # Your code here
    train_model()
```

**What happens:**
1. Creates a new workload in the backend
2. Redirects stdout/stderr to capture logs
3. Starts GPU metrics collection (every 2 seconds)
4. Uploads live telemetry while your code runs
5. On success: marks workload as succeeded
6. On failure: captures exception, marks as failed
7. Always: restores stdout/stderr and uploads final state

### Function Decorator (`@tracker.track_function()`)

You can also track functions using a decorator:

```python
@tracker.track_function("Training Run")
def train_model():
    model = MyModel()
    train(model)
    return model

# Automatically tracked when called
model = train_model()
```

### Live Telemetry Streaming

By default, the SDK uploads logs and metrics every 2 seconds:

```python
tracker = WorkloadTracker(
    url,
    api_key=key,
    upload_interval=5  # Upload every 5 seconds instead
)
```

**What's uploaded:**
- **Logs**: Last 256KB of stdout/stderr (rolling buffer)
- **Metrics**: Last 300 GPU samples (10 minutes at 2-second intervals)
- **Runtime**: Elapsed seconds since job started

This allows you to monitor progress in the dashboard while the job is still running.

## GPU Metrics Collection

The SDK automatically detects your GPU platform and collects appropriate metrics:

### NVIDIA GPUs (via NVML)

If `nvidia-ml-py` is installed:
- **Memory**: Used MB, Total MB, Percentage
- **Utilization**: GPU compute utilization %
- **Temperature**: Celsius

### NVIDIA/AMD GPUs (via PyTorch)

Fallback if NVML unavailable:
- **Memory**: Device-wide used memory, Total memory, Percentage, and Peak tensor allocation
- Works with both `torch.cuda` (NVIDIA) and ROCm (AMD)

### Apple Silicon (via MPS)

For MPS-capable Macs with `torch.mps`:
- **Driver Memory**: Metal driver allocation
- **Tensor Memory**: Actual PyTorch tensor allocation
- **Working Set**: Apple's recommended maximum (not physical VRAM)
- **Percentage**: Relative to working set (can exceed 100%)

**Note**: MPS utilization and temperature are unavailable (API limitation)

### No GPU / Simulation

If no supported GPU is detected:
- **No metrics are collected**
- The SDK does NOT simulate fake metrics
- Workload tracking still works (logs, errors, runtime)

## AI-Powered Diagnosis

### Using Your Own Fireworks AI Account (BYOK)

Provide your Fireworks credentials to get AI diagnosis:

```python
tracker = WorkloadTracker(
    os.environ["CRASHLENS_URL"],
    api_key=os.environ["CRASHLENS_API_KEY"],
    fireworks_api_key=os.environ.get("FIREWORKS_API_KEY"),
    fireworks_model=os.environ.get("FIREWORKS_MODEL")
)

# After a failure
tracker.diagnose(workload_id)
```

**How it works:**
- Your Fireworks credentials are sent only during diagnosis requests
- Never stored in the backend database
- Never included in workload uploads
- You pay for AI diagnosis (not the operator)
- Requires a tool-capable Fireworks model (e.g., `accounts/fireworks/models/llama-v3p3-70b-instruct`)

### Without Fireworks Credentials

If you don't provide Fireworks credentials:
- You get **rule-based diagnosis** (heuristic pattern matching)
- No AI analysis, but still useful failure classification
- Free (no API costs)

### Diagnosis Report Format

```python
report = tracker.diagnose(workload_id)
print(report["root_cause"])
print(report["recommended_fix"])
print(report["evidence"])
print(report["safe_to_retry"])
print(report["confidence"])
print(report["source"])
```

**Report fields:**
- **root_cause**: What went wrong
- **recommended_fix**: How to fix it
- **evidence**: Log excerpts supporting the diagnosis
- **safe_to_retry**: Whether re-running *without changes* might succeed (e.g., transient network errors)
- **prevention**: How to avoid this in the future
- **confidence**: Heuristic score (0.0-1.0) indicating diagnosis confidence
- **source**: Diagnosis method used (`"ai"` or `"rule_based"`)
- **confidence_basis**: Explanation of why this confidence score was assigned
- **ai_unavailable_reason**: (Only if AI diagnosis failed) Why rule-based fallback was used

## Jupyter Notebook Support

For enhanced notebook displays, use `JupyterWorkloadTracker`:

```python
from crashlens.jupyter import JupyterWorkloadTracker

tracker = JupyterWorkloadTracker(
    os.environ["CRASHLENS_URL"],
    api_key=os.environ["CRASHLENS_API_KEY"]
)

with tracker.track("Training") as workload_id:
    train_model()
```

**Features:**
- Color-coded status displays (✓ success, ✗ failure)
- Rich HTML formatted diagnosis reports
- GPU metrics displayed as pandas DataFrames
- Inline error messages with styling
- Auto-diagnosis on failure

### Magic Commands

Load the extension for even easier tracking:

```python
%load_ext crashlens.jupyter
%crashlens_init https://your-backend.railway.app
```

Then use magic commands:

```python
%%crashlens_track "Training Job"
model = train_model()
evaluate(model)
```

See [Jupyter Integration Guide](../docs/JUPYTER_INTEGRATION.md) for detailed usage and examples.

## Error Handling

The SDK is designed to never break your training:

### Exception Handling

```python
with tracker.track("Training"):
    raise ValueError("My model broke!")
# Exception is captured, workload marked as failed, then re-raised
```

The original exception always propagates - the SDK never swallows errors.

### Network Failures

```python
# If backend is unreachable during upload:
# - Logs warning to stderr
# - Continues training anyway
# - Tries to upload at next interval
```

### GPU Collection Errors

```python
# If GPU metrics collection fails:
# - Returns None (no samples)
# - Logs warning
# - Continues without metrics
# - Training is unaffected
```

## Configuration Reference

### WorkloadTracker Parameters

```python
tracker = WorkloadTracker(
    api_url: str,                      # Backend URL (required)
    api_key: str = None,                # Your API key (required if backend has auth)
    fireworks_api_key: str = None,      # Fireworks AI key (optional, for BYOK)
    fireworks_model: str = None,        # Fireworks model ID (required with fireworks_api_key)
    upload_interval: float = 2.0,       # Seconds between live uploads
)
```

### Context Manager Methods

```python
# Track a code block
with tracker.track("Training") as workload_id:
    ...

# Track a function
@tracker.track_function(name: str)
def my_function():
    ...

# Diagnose a failure
report = tracker.diagnose(workload_id)
```

## Validation & Testing

### Validate Your GPU Setup

Run the validation script to test GPU detection and live uploads. The script performs ten seconds of actual training and requires real telemetry to reach the API while running - it doesn't just test detection/upload or print available metrics:

```bash
# From repository root
python scripts/validate_gpu.py
```

This will:
- Detect your GPU platform (NVIDIA/AMD/MPS)
- Show available metrics
- Perform a test upload to your backend
- Verify authentication works

### Run SDK Tests

The SDK tests can run if `pytest` is installed, but it isn't included. Run using the existing `unittest` command:

```bash
cd crashlens-sdk
python -m unittest discover tests/
```

Tests cover:
- Metric collection for all GPU platforms (with mocked collectors to distinguish from real hardware validation)
- Live telemetry streaming
- Error handling and exception propagation
- Jupyter magic commands
- BYOK credential forwarding


## SDK Limitations

The SDK cannot capture failures in certain scenarios:

- **SIGKILL (kill -9)**: Process is terminated immediately without cleanup - no error can be reported
- **Notebook kernel crashes**: If the Jupyter kernel dies, the SDK context manager cannot execute cleanup code
- **Machine loss**: If the entire machine shuts down or becomes unreachable, the SDK cannot upload final state

In these cases, the workload will remain in `running` status in the dashboard. The last successfully uploaded telemetry snapshot will be available, but there will be no error traceback.

## Architecture

```
Your Training Code
       ↓
WorkloadTracker Context Manager
       ↓
┌──────────────────────────────────────┐
│ 1. Capture stdout/stderr             │
│ 2. Start GPU metrics collection      │
│ 3. Upload every N seconds            │
│ 4. Handle exceptions                 │
│ 5. Mark success/failure              │
│ 6. Restore stdout/stderr             │
└──────────────────────────────────────┘
       ↓
CrashLens Backend (REST API)
       ↓
Dashboard (view workload, diagnose)
```    
