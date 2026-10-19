# AMD Developer Cloud Quick Start for CrashLens

Access real AMD Instinct MI210/MI250 GPUs for FREE!

## Step 1: Apply for Access (5 minutes)

### Go to AMD Infinity Hub
- URL: https://www.amd.com/en/technologies/infinity-hub
- Click "Request Access" or "Get Started"

### Fill Out Application

**Your Project Description:**
```
CrashLens - ML Failure Diagnosis Platform for AMD GPUs

I'm building an AI-powered platform that diagnoses failures in ML workloads
running on AMD GPUs. It provides real-time observability, captures ROCm errors,
and uses AI to suggest fixes.

I need AMD Instinct GPU access to test real ROCm workloads and capture
authentic AMD GPU failure scenarios.

GitHub: https://github.com/kavinsaravan/ML-Failure-Doctor
Live: https://frontend-zeta-eight-92.vercel.app
```

**What You'll Get:**
- AMD Instinct MI210 or MI250 GPU
- $100-500 in free compute credits
- ROCm 5.7+ pre-installed
- Ubuntu 22.04 environment

**Timeline:**
- Application review: 1-3 business days
- Approval email with portal access

## Step 2: Launch Your Instance (Once Approved)

### Login to AMD Portal
1. Check email for portal URL and credentials
2. Login and navigate to "Compute" → "Create Instance"

### Recommended Configuration
- **GPU:** AMD Instinct MI210 (8GB VRAM)
- **vCPUs:** 8-16
- **RAM:** 32GB
- **Storage:** 100GB SSD
- **OS:** Ubuntu 22.04 with ROCm 5.7+

### Get SSH Access
```bash
# From the portal, get your instance IP
ssh ubuntu@<INSTANCE_IP>
```

## Step 3: Setup CrashLens (10 minutes)

### Install Dependencies
```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install Python and tools
sudo apt install -y python3-pip git

# Verify ROCm is working
rocm-smi

# Should show your AMD Instinct GPU
```

### Install PyTorch with ROCm
```bash
pip3 install torch torchvision --index-url https://download.pytorch.org/whl/rocm5.7

# Verify
python3 -c "import torch; print(f'GPU Available: {torch.cuda.is_available()}')"
# Should print: GPU Available: True
```

### Install CrashLens SDK
```bash
# Clone repo
git clone https://github.com/kavinsaravan/ML-Failure-Doctor.git
cd ML-Failure-Doctor/crashlens-sdk

# Install
pip3 install -e .
```

## Step 4: Run Your First Workload (5 minutes)

### Create Test Script
```python
# Save as: amd_test.py
import torch
import torch.nn as nn
from crashlens import WorkloadTracker

tracker = WorkloadTracker("https://invigorating-empathy-production-dee5.up.railway.app")

print(f"GPU: {torch.cuda.get_device_name(0)}")

with tracker.track("AMD Instinct Test"):
    # Simple model
    model = nn.Linear(1000, 100).cuda()
    x = torch.randn(32, 1000).cuda()

    # Forward pass
    for i in range(10):
        y = model(x)
        loss = y.mean()
        print(f"Step {i+1}: Loss = {loss.item():.4f}")

    print("✅ Training complete!")

print("\nCheck dashboard: https://frontend-zeta-eight-92.vercel.app/dashboard")
```

### Run It
```bash
python3 amd_test.py
```

### Expected Output
```
GPU: AMD Instinct MI210
Step 1: Loss = 0.0234
Step 2: Loss = 0.0187
...
Step 10: Loss = 0.0098
✅ Training complete!

Check dashboard: https://frontend-zeta-eight-92.vercel.app/dashboard
```

## Step 5: Test a Failure Scenario

### GPU Out of Memory
```python
# Save as: test_oom.py
import torch
from crashlens import WorkloadTracker

tracker = WorkloadTracker("https://invigorating-empathy-production-dee5.up.railway.app")

with tracker.track("AMD GPU - OOM Test"):
    tensors = []
    for i in range(100):
        # Each tensor ~1GB
        tensor = torch.randn(128, 1024, 1024).cuda()
        tensors.append(tensor)
        print(f"Allocated {i+1} tensors")
```

### Run and Capture Failure
```bash
python3 test_oom.py
# Will fail with "HIP out of memory" - exactly what CrashLens tracks!
```

### Get AI Diagnosis
1. Go to: https://frontend-zeta-eight-92.vercel.app/dashboard
2. Find the failed workload
3. Click "Run AI Diagnosis"
4. See AI-powered recommendations!

## Quick Commands Reference

### Check GPU Status
```bash
rocm-smi                    # GPU info
rocm-smi --showmeminfo vram # Memory usage
rocm-smi --showtemp         # Temperature
```

### Monitor GPU in Real-time
```bash
watch -n 1 rocm-smi
```

### Check ROCm Version
```bash
rocm-smi --showdriverversion
```

## What Gets Tracked

✅ **Real AMD GPU metrics**
✅ **HIP/ROCm error messages**
✅ **Memory usage patterns**
✅ **Training logs and outputs**
✅ **Runtime statistics**

## Cost Management

- **Free credits:** Usually $100-500
- **MI210 cost:** ~$2-3/hour
- **Stop instance when not in use!**
- **Credits last:** 30-50+ hours

### Stop Your Instance
```bash
# From AMD portal:
# Compute → Your Instance → Stop

# Or keep running if actively testing
```

## Troubleshooting

### GPU Not Detected
```bash
# Check if GPU is visible
ls /dev/kfd /dev/dri

# Add user to groups
sudo usermod -a -G video $USER
sudo usermod -a -G render $USER

# Logout and back in
```

### PyTorch Not Using GPU
```bash
# Reinstall with correct ROCm version
pip3 uninstall torch
pip3 install torch --index-url https://download.pytorch.org/whl/rocm5.7
```

## Next Steps

1. **✅ Applied for AMD Cloud access**
2. **⏳ Wait for approval (1-3 days)**
3. **🚀 Launch instance and run tests**
4. **📊 View results on CrashLens dashboard**
5. **🤖 Get AI-powered failure diagnosis**

## Resources

- **Application:** https://www.amd.com/en/technologies/infinity-hub
- **ROCm Docs:** https://rocm.docs.amd.com
- **Dashboard:** https://frontend-zeta-eight-92.vercel.app
- **GitHub:** https://github.com/kavinsaravan/ML-Failure-Doctor

---

**You're all set! Apply now and you'll have real AMD GPU workloads in CrashLens within days!** 🚀
