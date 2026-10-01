# CrashLens Jupyter Notebook Integration

Complete guide to using CrashLens in Jupyter notebooks.

---

## Overview

CrashLens provides first-class support for Jupyter notebooks:

- **Rich HTML Displays**: Color-coded status messages and formatted diagnosis reports
- **IPython Magic Commands**: Track cells with `%%crashlens_track "Job Name"`
- **Inline Metrics**: GPU metrics displayed as pandas DataFrames
- **Auto-Diagnosis**: Automatically diagnose failures in tracked cells
- **Auto-Detection**: Automatically detects Jupyter environment

---

## Installation

```bash
cd crashlens-sdk
pip install -e ".[jupyter]"
```

This installs CrashLens with IPython, Jupyter, and pandas support.

---

## Quick Start

### Option 1: Enhanced Tracker

```python
from crashlens.jupyter import JupyterWorkloadTracker

tracker = JupyterWorkloadTracker("https://your-backend.railway.app")

# Track with rich displays
with tracker.track("Training Model", display_metrics=True):
    model.fit(X_train, y_train)
    # Shows: ✓ Success (green), runtime, GPU metrics table

# Auto-diagnose failures
with tracker.track("Risky Job", auto_diagnose=True):
    risky_operation()
    # On failure: Shows ✗ Failed (red) + AI diagnosis automatically
```

### Option 2: Magic Commands

```python
# Load extension (once per notebook)
%load_ext crashlens.jupyter

# Initialize tracker
%crashlens_init https://your-backend.railway.app

# Track entire cells
%%crashlens_track "Data Processing"
data = load_and_process_data()
features = extract_features(data)
# Entire cell tracked + auto-diagnosed on failure
```

---

## Magic Commands Reference

| Command | Description | Example |
|---------|-------------|---------|
| `%crashlens_init <url>` | Initialize tracker | `%crashlens_init https://backend.url` |
| `%%crashlens_track "Name"` | Track entire cell | See above |
| `%crashlens_diagnose <id>` | Get AI diagnosis | `%crashlens_diagnose 123` |
| `%crashlens_show <id>` | Show workload details | `%crashlens_show 123` |

**When to use magic commands:**
- ✅ Quick cell tracking
- ✅ Interactive exploration
- ❌ Inside functions or loops (use context manager instead)

---

## Usage Examples

### Track Training Pipeline

```python
from crashlens.jupyter import JupyterWorkloadTracker

tracker = JupyterWorkloadTracker(api_url)

# Track each stage separately
with tracker.track("1. Data Loading"):
    X, y = load_data()

with tracker.track("2. Model Training"):
    model.fit(X, y)

with tracker.track("3. Evaluation"):
    score = model.score(X_test, y_test)
```

### Track Hyperparameter Experiments

```python
for lr in [0.001, 0.01, 0.1]:
    for bs in [16, 32, 64]:
        with tracker.track(f"lr={lr} bs={bs}", auto_diagnose=True):
            model = train_model(lr=lr, batch_size=bs)
            score = evaluate(model)
```

### PyTorch Training

```python
with tracker.track("PyTorch Training", display_metrics=True):
    model = SimpleNN().cuda()
    optimizer = optim.Adam(model.parameters())

    for epoch in range(10):
        for data, targets in dataloader:
            outputs = model(data.cuda())
            loss = criterion(outputs, targets.cuda())
            loss.backward()
            optimizer.step()
```

---

## Example Notebooks

Check out the example notebooks in `crashlens-sdk/examples/notebooks/`:

1. **01_quickstart.ipynb** - Basic features, success/failure tracking, rich displays
2. **02_magic_commands.ipynb** - IPython magic commands, cell tracking
3. **03_pytorch_training.ipynb** - PyTorch integration, GPU OOM handling

```bash
cd crashlens-sdk/examples/notebooks
jupyter notebook
```

---

## Best Practices

### 1. Initialize Once
```python
# ✅ Good: Initialize at top of notebook
tracker = JupyterWorkloadTracker(api_url)

# ❌ Bad: Re-initialize every time
JupyterWorkloadTracker(api_url).track("Job")
```

### 2. Use Descriptive Names
```python
# ✅ Good
with tracker.track("ResNet-50 Training - Batch 32 - LR 0.001"):

# ❌ Bad
with tracker.track("Training"):
```

### 3. Track Stages Separately
```python
# ✅ Good: Isolate failures to specific stages
with tracker.track("Data Loading"):
    data = load()
with tracker.track("Training"):
    model.fit(data)

# ❌ Bad: Track everything together
with tracker.track("Everything"):
    data = load()
    model.fit(data)
```

### 4. Use Auto-Diagnose During Development
```python
# ✅ Good: Auto-diagnose during experiments
with tracker.track("Experiment", auto_diagnose=True):
    risky_code()

# ❌ Bad: Manual diagnosis each time
with tracker.track("Experiment"):
    risky_code()
# Then have to remember: tracker.diagnose(workload_id)
```

### 5. Disable Metrics for Speed
```python
# ✅ For quick jobs, skip metrics collection
with tracker.track("Fast Job", display_metrics=False):
    quick_operation()
```

---

## Troubleshooting

### "IPython not available"

**Solution:**
```bash
pip install -e ".[jupyter]"
```

### Magic commands not working

**Solutions:**
1. Load extension: `%load_ext crashlens.jupyter`
2. Restart kernel and try again
3. Verify installation:
   ```python
   import crashlens.jupyter
   print("Jupyter integration available!")
   ```

### "Module 'crashlens' has no attribute 'jupyter'"

**Solution:**
```bash
pip uninstall crashlens
pip install -e ".[jupyter]"
```

### No rich HTML displays

**Expected behavior:** Falls back to plain text when not in Jupyter

**Solution:** Use Jupyter notebook or JupyterLab (not terminal Python)

### Metrics not displaying

**Cause:** SDK-tracked jobs use simulated metrics (not real GPU metrics)

**Note:** Only jobs run via backend API (`/workloads/run`) collect real GPU metrics

---

## Key Features

### Rich Displays

**Success (Green):**
```
✓ Workload Completed Successfully
Name: Training Model
Runtime: 45.2s
Workload ID: 123
```

**Failure (Red):**
```
✗ Workload Failed
Runtime: 12.5s
Run diagnosis: tracker.diagnose(123)
```

**AI Diagnosis:**
```
🔍 AI Diagnosis Report

Root Cause: GPU Out of Memory - batch size too large

✅ Recommended Fixes:
1. Reduce batch size from 32 to 16
2. Enable gradient checkpointing
3. Use mixed precision (FP16)

Evidence:
• CUDA out of memory
• Tried to allocate 2048 MiB
```

### Manual Diagnosis

```python
# Get diagnosis with rich formatting
diagnosis = tracker.diagnose(workload_id)

# View workload details
tracker.show_workload(workload_id)
```

---

## API Reference

### JupyterWorkloadTracker

```python
tracker = JupyterWorkloadTracker(api_url: str)

# Track with context manager
with tracker.track(
    name: str,
    display_metrics: bool = True,
    auto_diagnose: bool = False
) -> int:
    # Your code
    pass

# Manual diagnosis
diagnosis = tracker.diagnose(
    workload_id: int,
    display: bool = True
) -> Dict[str, Any]

# Show workload details
tracker.show_workload(workload_id: int)
```

---

## Additional Resources

- **Dashboard**: https://frontend-zeta-eight-92.vercel.app/dashboard
- **SDK README**: ../crashlens-sdk/README.md
- **Main README**: ../README.md
- **Example Notebooks**: ../crashlens-sdk/examples/notebooks/

---

## License

MIT License - Part of the CrashLens project
