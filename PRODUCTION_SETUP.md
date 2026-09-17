# CrashLens Production Setup Guide

## Getting Started with Real Workloads

### Step 1: Install the SDK

```bash
cd crashlens-sdk
pip install -e .
```

### Step 2: Wrap Your Training Code

**Before:**
```python
model.fit(X_train, y_train, epochs=10)
```

**After:**
```python
from crashlens import WorkloadTracker

tracker = WorkloadTracker("https://invigorating-empathy-production-dee5.up.railway.app")

with tracker.track("Model Training"):
    model.fit(X_train, y_train, epochs=10)
```

That's it! CrashLens now tracks your workload.

### Step 3: View Results

Go to your dashboard:
```
https://frontend-zeta-eight-92.vercel.app/dashboard
```

## Real-World Integration Examples

### 1. Local Development (No GPU Required)

```python
# test_crashlens.py
from crashlens import WorkloadTracker

tracker = WorkloadTracker("https://invigorating-empathy-production-dee5.up.railway.app")

with tracker.track("Test Job - Local"):
    print("Running computation...")
    result = sum(range(1000000))
    print(f"Result: {result}")
```

Run it:
```bash
python test_crashlens.py
```

Check the dashboard - you'll see it tracked!

### 2. Real PyTorch Training

If you have PyTorch installed:

```bash
cd crashlens-sdk/examples
pip install torch
python pytorch_training.py
```

### 3. With Your Existing Code

Just import and wrap:

```python
from crashlens import WorkloadTracker

tracker = WorkloadTracker("https://invigorating-empathy-production-dee5.up.railway.app")

# Wrap any function
with tracker.track("Data Processing"):
    df = pd.read_csv("large_file.csv")
    df_clean = preprocess(df)
    df_clean.to_csv("output.csv")

# It works with everything!
with tracker.track("Model Inference"):
    predictions = model.predict(test_data)
```

## Using with Cloud GPUs

### Google Colab

```python
# In your Colab notebook
!pip install requests

# Copy the SDK files or install from your repo
from crashlens import WorkloadTracker

tracker = WorkloadTracker("https://invigorating-empathy-production-dee5.up.railway.app")

with tracker.track("Colab Training"):
    # Your GPU training code
    model.fit(...)
```

### AWS SageMaker

```python
# In your training script
from crashlens import WorkloadTracker

tracker = WorkloadTracker(os.environ['CRASHLENS_URL'])

with tracker.track(f"SageMaker Job - {job_name}"):
    estimator.fit()
```

### Runpod / Vast.ai

Same approach - just add the tracker wrapper!

## What Gets Tracked?

- ✅ Start/end time
- ✅ Runtime duration
- ✅ Exit code (success/failure)
- ✅ Full stdout/stderr logs
- ✅ Exception tracebacks
- ✅ GPU metrics (nvidia-smi/rocm-smi or simulated)
- ✅ GPU memory usage, utilization, temperature
- ✅ Failure classification and diagnosis

## Next Steps

1. **Try the examples**: `cd crashlens-sdk/examples`
2. **Integrate with your code**: Add 3 lines of code
3. **Run a job**: Execute your script
4. **Check dashboard**: View results
5. **Simulate a failure**: Raise an exception
6. **Get AI diagnosis**: Click "Run AI Diagnosis"

## Cost Tracking

CrashLens automatically calculates:
- Wasted GPU-seconds on failures
- Total runtime per job
- Most common failure types
- Success rate percentage

Perfect for:
- Optimizing training costs
- Finding recurring issues
- Improving reliability

## Questions?

- Check the SDK README: `crashlens-sdk/README.md`
- View examples: `crashlens-sdk/examples/`
- Dashboard: https://frontend-zeta-eight-92.vercel.app

