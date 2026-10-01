# CrashLens Jupyter Notebook Integration Guide

Complete guide to using CrashLens in Jupyter notebooks with rich displays, magic commands, and inline visualizations.

---

## Overview

CrashLens provides first-class support for Jupyter notebooks, including:

- **Rich HTML Displays**: Color-coded status messages, formatted diagnosis reports
- **IPython Magic Commands**: Track cells with `%%crashlens_track`
- **Inline Metrics**: GPU metrics displayed as pandas DataFrames
- **Auto-Detection**: Automatically detects Jupyter environment
- **Live Progress**: Real-time workload status indicators

**Why use CrashLens in Jupyter?**
- Track experiments without leaving your notebook
- Get instant AI diagnosis when cells fail
- Visualize GPU metrics inline
- Share reproducible failure analysis

---

## Installation

### Option 1: With Jupyter Support (Recommended)

```bash
cd crashlens-sdk
pip install -e ".[jupyter]"
```

Installs:
- CrashLens core SDK
- IPython (for magic commands)
- Jupyter notebook
- pandas (for metrics display)

### Option 2: All Features

```bash
pip install -e ".[all]"
```

### Verify Installation

```python
from crashlens.jupyter import JupyterWorkloadTracker
print("✓ Jupyter integration available!")
```

---

## Quick Start

### Basic Usage

```python
# 1. Import the Jupyter tracker
from crashlens.jupyter import JupyterWorkloadTracker

# 2. Initialize
tracker = JupyterWorkloadTracker("https://your-backend.railway.app")

# 3. Track your code
with tracker.track("My First Job", display_metrics=True):
    print("Hello from CrashLens!")
    result = 42 * 42
    print(f"Result: {result}")
```

**Output:**
- Rich HTML display showing success
- Workload ID and runtime
- Link to dashboard

### Using Magic Commands

```python
# 1. Load extension
%load_ext crashlens.jupyter

# 2. Initialize
%crashlens_init https://your-backend.railway.app

# 3. Track a cell
%%crashlens_track "Data Processing"
import pandas as pd
data = pd.read_csv("data.csv")
print(f"Loaded {len(data)} rows")
```

---

## Features

### 1. Rich HTML Displays

All outputs are beautifully formatted with color-coded status:

**Success (Green):**
```python
with tracker.track("Simple Job"):
    print("Success!")
# Displays: ✓ Workload Completed Successfully (green box)
```

**Failure (Red):**
```python
try:
    with tracker.track("Failing Job"):
        raise ValueError("Oops!")
except:
    pass
# Displays: ✗ Workload Failed (red box with error)
```

**Running (Blue):**
```python
with tracker.track("Long Job"):
    time.sleep(5)
# Shows: 🔄 Workload Running (blue box with timer)
```

### 2. Auto-Diagnosis

Automatically run AI diagnosis on failure:

```python
with tracker.track("GPU Training", auto_diagnose=True):
    huge_tensor = torch.randn(100000, 100000).cuda()
    # On OOM: Automatically shows diagnosis with fixes
```

**Diagnosis includes:**
- Root cause analysis
- Evidence from logs
- Recommended fixes (numbered list)
- Prevention strategies
- Safe to retry? (Yes/No)

### 3. Inline GPU Metrics

Display GPU metrics as pandas DataFrame:

```python
with tracker.track("Training", display_metrics=True):
    train_model()
# After completion, shows table:
# | timestamp | memory_used_mb | gpu_utilization | temperature |
```

### 4. Workload Details

View comprehensive workload info:

```python
tracker.show_workload(workload_id)
```

Displays:
- Status (with color)
- Type
- Runtime
- Exit code
- Failure type
- Wasted GPU-seconds

### 5. Manual Diagnosis

Get AI diagnosis with rich formatting:

```python
diagnosis = tracker.diagnose(workload_id)
# Returns dict AND displays rich HTML report
```

---

## Usage Patterns

### Pattern 1: Track Entire Training Pipeline

```python
from crashlens.jupyter import JupyterWorkloadTracker

tracker = JupyterWorkloadTracker(api_url)

# Track each stage separately
with tracker.track("1. Data Loading"):
    X, y = load_data()

with tracker.track("2. Preprocessing"):
    X = preprocess(X)

with tracker.track("3. Model Training"):
    model.fit(X, y)

with tracker.track("4. Evaluation"):
    score = model.score(X_test, y_test)
    print(f"Score: {score}")
```

**Benefits:**
- Isolate failures to specific stages
- Track progress through pipeline
- Separate diagnosis for each step

### Pattern 2: Track Hyperparameter Experiments

```python
import itertools

params = {
    'learning_rate': [0.001, 0.01, 0.1],
    'batch_size': [16, 32, 64]
}

for lr, bs in itertools.product(params['learning_rate'], params['batch_size']):
    name = f"Experiment lr={lr} bs={bs}"

    with tracker.track(name):
        model = train_model(lr=lr, batch_size=bs)
        score = evaluate(model)
        print(f"Score: {score}")
```

**Benefits:**
- Track all experiments automatically
- Compare success/failure rates
- Identify problematic hyperparameters

### Pattern 3: Track with Error Handling

```python
def train_with_tracking(name, **kwargs):
    """Train model with automatic failure tracking"""
    try:
        with tracker.track(name, auto_diagnose=True):
            model = create_model(**kwargs)
            model.fit(X_train, y_train)
            return model
    except Exception as e:
        print(f"Training failed: {e}")
        print("Check diagnosis above for fixes")
        return None

# Use it
model = train_with_tracking("ResNet Training", epochs=10, batch_size=32)
```

**Benefits:**
- Graceful error handling
- Automatic diagnosis
- Reusable pattern

### Pattern 4: Conditional Tracking (Dev vs Prod)

```python
import os

# Only track in development
if os.getenv("ENV") == "development":
    from crashlens.jupyter import JupyterWorkloadTracker
    tracker = JupyterWorkloadTracker(api_url)
    context = tracker.track
else:
    # No-op context manager for production
    from contextlib import nullcontext
    context = lambda name, **kwargs: nullcontext()

# Works in both environments
with context("Training"):
    train_model()
```

---

## Magic Commands

### Available Commands

| Command | Type | Description |
|---------|------|-------------|
| `%crashlens_init <url>` | Line | Initialize tracker |
| `%%crashlens_track "Name"` | Cell | Track entire cell |
| `%crashlens_diagnose <id>` | Line | Get AI diagnosis |
| `%crashlens_show <id>` | Line | Show workload details |

### Detailed Usage

#### 1. Initialize Tracker

```python
%crashlens_init https://your-backend.railway.app
```

**Output:**
- Success message (green box)
- Connected URL
- Usage instructions

**Only needed once per notebook session.**

#### 2. Track a Cell

```python
%%crashlens_track "Data Analysis"

import pandas as pd
import matplotlib.pyplot as plt

df = pd.read_csv("data.csv")
df.describe()
plt.hist(df['column'])
plt.show()
```

**Features:**
- Captures all cell output
- Auto-diagnoses on failure
- Shows rich HTML status

**Note:** Magic commands work at cell granularity, not line-by-line.

#### 3. Diagnose Workload

```python
%crashlens_diagnose 123
```

**Output:**
- Root cause (formatted)
- Evidence (code blocks)
- Recommended fixes (numbered list)
- Prevention tips
- Retry safety

#### 4. Show Workload

```python
%crashlens_show 123
```

**Output:**
- Workload metadata table
- Color-coded status
- Runtime, exit code, etc.

### Magic vs Context Manager

**When to use magic commands:**
- ✅ Quick cell tracking
- ✅ Interactive exploration
- ✅ One-off experiments

**When to use context managers:**
- ✅ Inside functions
- ✅ In loops
- ✅ Conditional tracking
- ✅ Production code

---

## Rich HTML Displays

### Success Display

```html
<div style="border: 2px solid #27ae60; ...">
  <h4>✓ Workload Completed Successfully</h4>
  <p>Name: Training Model</p>
  <p>Runtime: 12.5s</p>
  <p>Workload ID: 123</p>
  <a href="dashboard">View details</a>
</div>
```

### Failure Display

```html
<div style="border: 2px solid #e74c3c; ...">
  <h4>✗ Workload Failed</h4>
  <p>Name: GPU Training</p>
  <p>Runtime: 5.2s</p>
  <p>Run diagnosis: tracker.diagnose(123)</p>
</div>
```

### Diagnosis Display

```html
<div style="border: 2px solid #e74c3c; ...">
  <h3>🔍 AI Diagnosis Report</h3>

  <strong>Root Cause:</strong>
  <p>GPU Out of Memory - batch size too large</p>

  <strong>Evidence:</strong>
  <ul>
    <li><code>CUDA out of memory</code></li>
    <li><code>Tried to allocate 2048 MiB</code></li>
  </ul>

  <strong>Recommended Fixes:</strong>
  <ol>
    <li>Reduce batch size from 32 to 16</li>
    <li>Enable gradient checkpointing</li>
    ...
  </ol>
</div>
```

---

## Example Notebooks

### 01_quickstart.ipynb

**What it covers:**
- Basic tracking
- Success and failure scenarios
- Manual diagnosis
- Workload details
- Real ML example (scikit-learn)

**Best for:** First-time users

**Run it:**
```bash
cd crashlens-sdk/examples/notebooks
jupyter notebook 01_quickstart.ipynb
```

### 02_magic_commands.ipynb

**What it covers:**
- Loading extension
- All magic commands
- Cell tracking
- Loop tracking (workarounds)
- Magic vs context manager

**Best for:** Users who want minimal code

**Run it:**
```bash
jupyter notebook 02_magic_commands.ipynb
```

### 03_pytorch_training.ipynb

**What it covers:**
- PyTorch integration
- GPU OOM simulation
- Real training loops
- Memory management
- Common failure patterns

**Best for:** Deep learning practitioners

**Run it:**
```bash
jupyter notebook 03_pytorch_training.ipynb
```

---

## Best Practices

### 1. Initialize Once

```python
# ✅ Good: Initialize at top of notebook
from crashlens.jupyter import JupyterWorkloadTracker
tracker = JupyterWorkloadTracker(api_url)

# Use throughout notebook
with tracker.track("Job 1"): ...
with tracker.track("Job 2"): ...
```

```python
# ❌ Bad: Re-initialize every time
with JupyterWorkloadTracker(api_url).track("Job"): ...
```

### 2. Use Descriptive Names

```python
# ✅ Good: Descriptive names
with tracker.track("ResNet-50 Training - Batch 32 - LR 0.001"):
    train()

# ❌ Bad: Generic names
with tracker.track("Training"):
    train()
```

### 3. Track Granularly

```python
# ✅ Good: Track stages separately
with tracker.track("Data Loading"):
    data = load()

with tracker.track("Preprocessing"):
    data = preprocess(data)

with tracker.track("Training"):
    model.fit(data)
```

```python
# ❌ Bad: Track everything together
with tracker.track("Everything"):
    data = load()
    data = preprocess(data)
    model.fit(data)
```

### 4. Use Auto-Diagnose During Development

```python
# ✅ Good: Auto-diagnose during experiments
with tracker.track("Experiment", auto_diagnose=True):
    risky_code()
```

```python
# ❌ Bad: Manual diagnosis every time
try:
    with tracker.track("Experiment"):
        risky_code()
except:
    tracker.diagnose(workload_id)  # Have to remember ID
```

### 5. Disable Metrics for Speed

```python
# ✅ Good: Skip metrics for quick jobs
with tracker.track("Fast Job", display_metrics=False):
    quick_operation()
```

---
