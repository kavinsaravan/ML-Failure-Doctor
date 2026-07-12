# CrashLens Jupyter Notebooks

This folder contains Jupyter notebooks for testing CrashLens with real AMD GPU workloads.

## 📓 Available Notebooks

### `CrashLens_AMD_GPU_Demo.ipynb`

Complete demonstration of CrashLens running on AMD GPUs with ROCm.

**Features:**
- ✅ Real AMD GPU metric collection via PyTorch
- ✅ Automatic workload creation and tracking
- ✅ Live synchronization with CrashLens backend
- ✅ AI-powered failure diagnosis
- ✅ Works with MI210, MI250X, and other AMD GPUs

**Test Scenarios Included:**
1. **Successful Training** - PyTorch MNIST training (5 epochs)
2. **GPU Out of Memory** - Intentional OOM failure
3. **Dependency Error** - Missing Python library
4. **Missing Checkpoint** - File not found error

---

## 🚀 How to Use

### Option 1: AMD Developer Cloud

1. **Login to AMD Developer Cloud**
   - Visit: https://radeon-global.anruicloud.com/
   - Use your AMD Developer Cloud credentials

2. **Upload Notebook**
   - Click "Upload" in Jupyter
   - Select `CrashLens_AMD_GPU_Demo.ipynb`

3. **Configure Backend Connection**
   - Open the notebook
   - Find the `CRASHLENS_API` variable in cell 4
   - Update with your backend URL:
     - If using ngrok: `https://your-subdomain.ngrok-free.dev`
     - If using deployed backend: Your backend URL

4. **Run the Notebook**
   - Execute cells sequentially (Cell → Run All)
   - Watch as workloads are created, executed, and diagnosed
   - View results in CrashLens dashboard

### Option 2: Local Jupyter with AMD GPU

```bash
# Install Jupyter
pip install jupyter

# Install PyTorch with ROCm
pip install torch torchvision torchaudio --index-url https://download.pytorch.org/whl/rocm5.7

# Start Jupyter
cd notebooks
jupyter notebook

# Open CrashLens_AMD_GPU_Demo.ipynb
# Update CRASHLENS_API to http://localhost:8080
# Run all cells
```

---

## 🔧 Setup Requirements

**Required:**
- Python 3.9+
- PyTorch with ROCm support
- `requests` library
- CrashLens backend running and accessible

**Optional (for real GPU metrics):**
- AMD GPU (MI210, MI250X, etc.)
- ROCm 5.7+
- `rocm-smi` utility

**For Development (CPU Mode):**
- The notebook works in CPU mode for testing
- GPU metrics will be simulated
- OOM errors are simulated with `RuntimeError`

---

## 📊 What You'll See

After running the notebook:

1. **In the Notebook:**
   - ✅ Workload creation confirmations
   - 📊 Training progress logs
   - 🤖 AI diagnosis reports
   - 📈 GPU metrics snapshots

2. **In CrashLens Dashboard:**
   - 1 successful workload (green)
   - 3 failed workloads (red)
   - Detailed logs for each job
   - GPU memory/utilization charts
   - AI-generated diagnosis with fixes

---

## 🌐 Backend Connection Options

### Using ngrok (Easiest for Remote Testing)

```bash
# Start CrashLens backend locally
cd backend
go run .

# In another terminal, expose with ngrok
ngrok http 8080

# Copy the ngrok URL (e.g., https://abc-xyz.ngrok-free.dev)
# Update CRASHLENS_API in notebook to this URL
```

### Using Deployed Backend

If you deployed the backend to Railway, Fly.io, or another service:

```python
# In notebook cell 4, update:
CRASHLENS_API = "https://your-backend.railway.app"
```

### Using Local Connection

If running Jupyter locally on the same machine as CrashLens:

```python
# In notebook cell 4, use:
CRASHLENS_API = "http://localhost:8080"
```

---

## 🐛 Troubleshooting

**"Cannot connect to CrashLens backend"**
- Verify backend is running: `curl http://localhost:8080/health`
- Check ngrok tunnel is active: `curl https://your-ngrok-url/health`
- Ensure firewall allows connections

**"No GPU detected"**
- Notebook will run in CPU mode with simulated errors
- Real GPU metrics require AMD GPU with ROCm installed
- Check PyTorch CUDA availability: `torch.cuda.is_available()`

**"Workload created but not visible in dashboard"**
- Refresh the dashboard page
- Check browser console for errors
- Verify API URL matches between notebook and dashboard

**"rocm-smi not found"**
- This is expected if ROCm isn't installed
- Notebook will use PyTorch metrics instead
- Real `rocm-smi` output requires ROCm installation

---

## 📝 Customizing Tests

You can modify the notebook to test your own workloads:

```python
# Create custom workload
workload = CrashLensWorkload("My Custom Training Job")
workload.create()

try:
    workload.log("Starting custom job...")
    # Your PyTorch code here

    workload.update_status("succeeded")
except Exception as e:
    workload.log(f"ERROR: {str(e)}")
    workload.update_status("failed", failure_type="CUSTOM_ERROR", exit_code=1)
    workload.diagnose()
```

---

## 🎥 Video Demo

This notebook is designed for:
- ✅ Live demonstrations
- ✅ AMD GPU testing validation
- ✅ Hackathon presentations
- ✅ Integration testing

Perfect for showing CrashLens working on real AMD hardware!

---

## 📚 Additional Resources

- [CrashLens Documentation](../README.md)
- [AMD ROCm Documentation](https://rocm.docs.amd.com/)
- [PyTorch ROCm Support](https://pytorch.org/get-started/locally/)
- [AMD Developer Cloud](https://radeon-global.anruicloud.com/)

---

**Questions?** Check the main [README](../README.md) or open an issue on GitHub.
