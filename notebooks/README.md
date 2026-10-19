# CrashLens Jupyter Notebooks

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

