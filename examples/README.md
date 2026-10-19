# CrashLens Examples

This directory contains example notebooks and scripts demonstrating CrashLens functionality.

## End-to-End Test (Google Colab)

**File:** `CrashLens_E2E_Test.ipynb`

**Purpose:** Real-world test with actual CUDA/PyTorch GPU workloads

**Features:**
- ✅ Successful training test
- ❌ GPU Out of Memory test (intentional failure)
- ❌ Dependency error test (intentional failure)
- 🔍 AI diagnosis demonstration
- 📊 Dashboard verification

**How to Run:**

### Option 1: Google Colab (Recommended)
1. Upload `CrashLens_E2E_Test.ipynb` to Google Colab
2. Enable GPU runtime: Runtime → Change runtime type → GPU (T4)
3. Set your backend URL in the notebook config cell
4. Run all cells
5. Check your CrashLens dashboard for results

### Option 2: Local Jupyter (Requires GPU)
```bash
# Install dependencies
pip install jupyter torch torchvision requests

# Start Jupyter
jupyter notebook examples/CrashLens_E2E_Test.ipynb
```

**Expected Results:**
- Workload 1: ✅ Successful training
- Workload 2: ❌ Failed with `CUDA out of memory` (correctly diagnosed as `GPU_OUT_OF_MEMORY`)
- Workload 3: ❌ Failed with `ModuleNotFoundError` (correctly diagnosed as `DEPENDENCY_ERROR`)

## Requirements

- GPU runtime (CUDA or ROCm)
- CrashLens backend deployed and accessible
- Python 3.8+
- PyTorch with GPU support
- `requests` library

## Configuration

Before running, update these variables in the notebook:

```python
BACKEND_URL = "https://your-backend.railway.app"  # Your deployed backend
API_KEY = "your_api_key"  # If backend requires authentication
```

## Troubleshooting

**No GPU available:**
- In Colab: Runtime → Change runtime type → GPU
- Locally: Ensure CUDA/ROCm drivers are installed

**Connection refused:**
- Verify backend URL is correct
- Check backend is running: `curl https://your-backend.railway.app/health`

**401 Unauthorized:**
- Set `API_KEY` if your backend requires authentication
- Check API key is correct in backend environment variables

## Next Steps

After running the E2E test:
1. Check the CrashLens dashboard to see all workloads
2. Review AI diagnosis for failed workloads
3. Inspect GPU metrics collected during execution
4. Try tracking your own ML training jobs
