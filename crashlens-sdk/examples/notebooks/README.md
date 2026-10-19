# CrashLens Jupyter Notebooks

This directory contains example Jupyter notebooks demonstrating CrashLens integration.

## 📚 Available Notebooks

### 1. **01_quickstart.ipynb** - Getting Started
Learn the basics of CrashLens in Jupyter:
- Initialize the tracker
- Track successful and failed jobs
- Get AI-powered diagnoses
- View workload details
- See rich HTML displays

**Best for:** First-time users

---

### 2. **02_magic_commands.ipynb** - IPython Magic Commands
Master the magic commands for effortless tracking:
- `%crashlens_init <api_url>` - Initialize tracker
- `%%crashlens_track "Job Name"` - Track entire cells
- `%crashlens_diagnose <id>` - Get AI diagnosis
- `%crashlens_show <id>` - View workload details

**Best for:** Users who want minimal boilerplate code

---

### 3. **03_pytorch_training.ipynb** - PyTorch Integration
Track real PyTorch training jobs:
- Track training loops
- Handle GPU OOM errors
- Get framework-specific recommendations
- Memory management tips
- Common failure patterns

**Best for:** Deep learning practitioners using PyTorch

---

## 🚀 Quick Start

### 1. Install CrashLens with Jupyter support

```bash
cd ../../  # Navigate to crashlens-sdk directory
pip install -e ".[jupyter]"
```

This installs:
- CrashLens core SDK
- IPython (for magic commands)
- Jupyter notebook
- pandas (for metrics display)

### 2. Launch Jupyter

```bash
jupyter notebook
```

### 3. Open a notebook

Start with `01_quickstart.ipynb` to learn the basics.

---

## 📊 Features Demonstrated

### Rich HTML Displays
All notebooks show beautiful, color-coded HTML output:
- ✅ Success messages (green)
- ❌ Failure alerts (red)
- 🔄 Running indicators (blue)
- 📊 Metrics tables
- 🔍 Diagnosis reports with evidence and fixes

### Real-time Tracking
Track any code block:
```python
from crashlens.jupyter import JupyterWorkloadTracker

tracker = JupyterWorkloadTracker("https://your-backend.url")

with tracker.track("Training Model", display_metrics=True):
    model.fit(X_train, y_train)
```

### Auto-Diagnosis
Automatically diagnose failures:
```python
with tracker.track("Job Name", auto_diagnose=True):
    # Your code that might fail
    risky_operation()
```

### Magic Commands
Track entire cells with one line:
```python
%load_ext crashlens.jupyter
%crashlens_init https://your-backend.url

%%crashlens_track "Data Processing"
# Your entire cell is tracked!
data = process_large_dataset()
```

---

## 🎯 Use Cases

### Data Science Workflows
- Track data preprocessing steps
- Monitor feature engineering
- Debug pipeline failures

### ML Training
- Track training epochs
- Catch GPU OOM errors
- Diagnose CUDA issues
- Monitor memory usage

### Model Experimentation
- Track hyperparameter sweeps
- Compare different architectures
- Log successful vs failed runs

### Production Pipelines
- Monitor batch inference
- Track ETL jobs
- Debug deployment issues

---

## 🛠️ Installation Options

### Option 1: Jupyter + All Features
```bash
pip install -e ".[all]"
```

### Option 2: Jupyter Only
```bash
pip install -e ".[jupyter]"
```

### Option 3: Core SDK Only (no Jupyter)
```bash
pip install -e .
```

---

## 💡 Tips & Tricks

### 1. Use Auto-Diagnosis During Development
```python
# During development, auto-diagnose failures
with tracker.track("Experiment", auto_diagnose=True):
    experiment_code()
```

### 2. Track Loop Iterations Separately
```python
for epoch in range(10):
    with tracker.track(f"Epoch {epoch+1}"):
        train_one_epoch()
```

### 3. Separate Concerns
```python
# Track different stages separately
with tracker.track("Data Loading"):
    data = load_data()

with tracker.track("Preprocessing"):
    data = preprocess(data)

with tracker.track("Training"):
    model.fit(data)
```

### 4. Get Workload ID for Later
```python
with tracker.track("Job") as workload_id:
    do_work()
    print(f"Track this job: {workload_id}")
```

### 5. Disable Metrics for Faster Execution
```python
# Skip metrics collection for speed
with tracker.track("Quick Job", display_metrics=False):
    fast_operation()
```

---

## 🔗 Additional Resources

- **Dashboard**: https://frontend-zeta-eight-92.vercel.app/dashboard
- **SDK Documentation**: ../../README.md
- **Main README**: ../../../README.md
- **API Reference**: See backend API docs

---

## 🐛 Troubleshooting

### Issue: "IPython not available"
**Solution:**
```bash
pip install ipython jupyter
```

### Issue: "pandas not found"
**Solution:**
```bash
pip install pandas
```

### Issue: Magic commands not working
**Solution:**
1. Make sure you ran `%load_ext crashlens.jupyter`
2. Restart the kernel and try again
3. Check that Jupyter dependencies are installed

### Issue: "Module 'crashlens' has no attribute 'jupyter'"
**Solution:**
```bash
# Reinstall with Jupyter support
pip install -e ".[jupyter]"
```

---

## 📝 Contributing

Have a cool use case? Create a new notebook and submit a PR!

Example ideas:
- TensorFlow integration
- Hugging Face transformers
- Distributed training
- Cloud platform integrations (Colab, SageMaker)

---

## 📄 License

MIT License - See main project LICENSE file
