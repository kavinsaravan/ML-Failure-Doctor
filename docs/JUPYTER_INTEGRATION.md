# Jupyter Notebook Integration

CrashLens provides enhanced Jupyter notebook support with rich HTML displays, inline error formatting, and magic commands for seamless workload tracking.

## What You Get

- **Rich HTML displays** - Color-coded status messages with visual feedback
- **Inline diagnostics** - Formatted diagnosis reports with syntax highlighting
- **GPU metrics tables** - Pandas DataFrames showing memory, utilization, and temperature
- **Magic commands** - Track cells with `%%crashlens_track` syntax
- **Auto-diagnosis** - Optional automatic failure analysis on exceptions
- **Live metrics display** - View GPU usage during training

## Installation

Install the SDK with Jupyter extras:

```bash
# From a cloned repository
pip install -e './crashlens-sdk[jupyter]'

# Or directly from GitHub
pip install 'git+https://github.com/kavinsaravan/ML-Failure-Doctor.git#subdirectory=crashlens-sdk[jupyter]'
```

## Setup

### 1. Get Your Credentials

1. Go to the CrashLens dashboard
2. Create a workspace or connect to an existing one
3. Save your API key securely

### 2. Configure Environment Variables

**Important**: Never paste credentials directly into notebook cells. Use environment variables or secure prompts.

```bash
# Set these before starting Jupyter
export CRASHLENS_URL="https://your-backend.railway.app"
export CRASHLENS_API_KEY="cl_your_api_key_here"

# Optional: For AI-powered diagnosis (BYOK)
export FIREWORKS_API_KEY="your_fireworks_key"
export FIREWORKS_MODEL="accounts/fireworks/models/llama-v3p3-70b-instruct"
```

**Alternative**: Use `getpass` for secure credential input inside notebooks:

```python
import getpass
import os

if "CRASHLENS_API_KEY" not in os.environ:
    os.environ["CRASHLENS_API_KEY"] = getpass.getpass("Enter CrashLens API Key: ")
```

## Usage

### Option 1: JupyterWorkloadTracker (Recommended)

Use the Jupyter-enhanced tracker for rich displays:

```python
import os
from crashlens.jupyter import JupyterWorkloadTracker

tracker = JupyterWorkloadTracker(
    os.environ["CRASHLENS_URL"],
    api_key=os.environ["CRASHLENS_API_KEY"],
    fireworks_api_key=os.environ.get("FIREWORKS_API_KEY"),
    fireworks_model=os.environ.get("FIREWORKS_MODEL"),
)

# Track a training run
with tracker.track("BERT Fine-tuning", display_metrics=True, auto_diagnose=False) as workload_id:
    model = train_model()
    evaluate(model)

print(f"Workload ID: {workload_id}")
```

**Parameters:**
- `display_metrics=True`: Show GPU metrics as a pandas DataFrame after completion
- `auto_diagnose=True`: Automatically request diagnosis if the job fails (requires Fireworks credentials)

### Option 2: IPython Magic Commands

Load the extension for cell-level tracking:

```python
%load_ext crashlens.jupyter
%crashlens_init https://your-backend.railway.app
```

Then track entire cells:

```python
%%crashlens_track "Training Run"
model = create_model()
train(model, epochs=10)
evaluate(model)
```

**What happens:**
- The cell is automatically wrapped in a workload tracker
- On success: displays ✓ with runtime
- On failure: displays ✗ with error details
- Returns the workload ID for later reference

## Features

### Rich Status Displays

The Jupyter tracker provides color-coded visual feedback:

- **✓ Success** (green): Job completed without errors
- **✗ Failure** (red): Job failed with exception details
- **Runtime**: Displayed in minutes and seconds

### GPU Metrics Display

When `display_metrics=True`, you'll see a formatted table:

```
GPU Metrics (last 10 samples):
┌─────────────┬──────────┬───────────┬─────────────┐
│ Memory (MB) │ Memory % │ Util (%)  │ Temp (°C)   │
├─────────────┼──────────┼───────────┼─────────────┤
│ 2048 / 8192 │   25.0   │    45.2   │    62       │
│ 2156 / 8192 │   26.3   │    48.1   │    63       │
└─────────────┴──────────┴───────────┴─────────────┘
```

**Note**: Available metrics depend on your GPU platform (NVIDIA CUDA, AMD ROCm, or Apple MPS). See [GPU Metrics Collection](../crashlens-sdk/README.md#gpu-metrics-collection).

### Auto-Diagnosis on Failure

Enable automatic diagnosis when jobs fail:

```python
with tracker.track("Training", auto_diagnose=True) as workload_id:
    train_model()
```

**With Fireworks credentials (BYOK):**
- Gets AI-powered root cause analysis
- Consumes your Fireworks credits
- Provides actionable recommendations

**Without Fireworks credentials:**
- Falls back to rule-based diagnosis
- Free heuristic pattern matching
- Still useful for common failure types

### Manual Diagnosis

Request diagnosis explicitly for any workload:

```python
# After a failure
report = tracker.diagnose(workload_id)
```

The report includes:
- Root cause explanation
- Recommended fix
- Evidence from logs
- Whether it's safe to retry
- Prevention tips

### View Workload Details

Retrieve stored workload information:

```python
workload = tracker.show_workload(workload_id)
```

Returns logs, metrics, status, and runtime details.

## Magic Command Reference

### `%load_ext crashlens.jupyter`

Loads the CrashLens IPython extension. Run this once per notebook session.

### `%crashlens_init <backend_url>`

Initializes the tracker with your backend URL. Reads API keys from environment variables.

```python
%crashlens_init https://your-backend.railway.app
```

### `%%crashlens_track "Job Name"`

Cell magic that wraps the entire cell in a workload tracker.

```python
%%crashlens_track "Data Preprocessing"
df = load_data()
df = clean_data(df)
save_processed(df)
```

**Returns**: The workload ID is displayed after execution.

### `%crashlens_diagnose <workload_id>`

Requests a diagnosis for a failed workload.

```python
%crashlens_diagnose wl_abc123def456
```

**Note**: Use the actual workload ID returned by your run, not a hardcoded ID from another workspace.

### `%crashlens_show <workload_id>`

Displays full details for a workload (logs, metrics, status).

```python
%crashlens_show wl_abc123def456
```

## Limitations

The SDK cannot capture failures in certain scenarios:

- **Kernel crashes**: If the Jupyter kernel dies, the context manager cannot execute cleanup code
- **Kernel interrupts**: Using the "Stop" button interrupts execution, which may prevent final uploads
- **Machine loss**: If the machine shuts down, the SDK cannot upload final state

In these cases, the workload will remain in `running` status. The last successfully uploaded telemetry snapshot will be available, but there will be no error traceback.

## Examples

### Basic Training with Auto-Diagnosis

```python
import os
from crashlens.jupyter import JupyterWorkloadTracker

tracker = JupyterWorkloadTracker(
    os.environ["CRASHLENS_URL"],
    api_key=os.environ["CRASHLENS_API_KEY"],
    fireworks_api_key=os.environ.get("FIREWORKS_API_KEY"),
    fireworks_model=os.environ.get("FIREWORKS_MODEL"),
)

with tracker.track("BERT Fine-tuning", auto_diagnose=True) as workload_id:
    model = create_bert_model()
    train(model, epochs=5)
    evaluate(model)
```

### Using Magic Commands

```python
%load_ext crashlens.jupyter
%crashlens_init https://your-backend.railway.app

%%crashlens_track "Hyperparameter Sweep"
for lr in [0.001, 0.01, 0.1]:
    model = create_model(learning_rate=lr)
    train(model)
    metrics = evaluate(model)
    print(f"LR {lr}: Accuracy {metrics['accuracy']}")
```

### Manual Diagnosis After Failure

```python
try:
    with tracker.track("Risky Training") as workload_id:
        train_unstable_model()
except Exception as e:
    print(f"Training failed: {e}")
    # Get diagnosis manually
    report = tracker.diagnose(workload_id)
    print(f"Root cause: {report['root_cause']}")
    print(f"Fix: {report['recommended_fix']}")
```

### Display Metrics During Development

```python
# Show GPU metrics to verify your setup
with tracker.track("Quick Test", display_metrics=True) as workload_id:
    quick_training_test()

# Metrics table appears after completion
```
