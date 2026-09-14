# 🚀 Get Started with Real AMD GPU Workloads - Action Plan

## What You Have Now

✅ **CrashLens Platform - Fully Deployed**
- Backend: https://invigorating-empathy-production-dee5.up.railway.app
- Frontend: https://frontend-zeta-eight-92.vercel.app/dashboard
- Python SDK: `crashlens-sdk/` (ready to install)
- 6 demo workloads already tracked
- $1.86 in Fireworks AI credits (9,000+ diagnoses)

## What You Need: Real AMD GPU Access

### ⏱️ DO THIS NOW (5 minutes)

**Step 1: Apply for AMD Developer Cloud**

1. Go to: https://www.amd.com/en/technologies/infinity-hub
2. Click "Request Access" or "Get Started"
3. Copy the application template: `docs/AMD_CLOUD_APPLICATION_TEMPLATE.txt`
4. Fill out the form with your info
5. Submit!

**What happens next:**
- ✉️ Confirmation email (immediate)
- ⏳ Review period (1-3 business days)
- ✅ Approval email with portal access
- 💰 $100-500 in free GPU credits

---

## While You Wait (Do These Today)

### Option 1: Test Locally (No GPU Needed)

The CrashLens SDK works WITHOUT a GPU! Test it right now:

```bash
cd crashlens-sdk

# Simple test
python3 << 'EOF'
from crashlens import WorkloadTracker
tracker = WorkloadTracker("https://invigorating-empathy-production-dee5.up.railway.app")

with tracker.track("Local Test"):
    print("Processing...")
    result = sum(range(1000000))
    print(f"Done: {result}")
EOF

# Check dashboard - it's already there!
open https://frontend-zeta-eight-92.vercel.app/dashboard
```

### Option 2: Use Google Colab (Free GPU - Available Now!)

**Start in 2 minutes:**

1. Go to: https://colab.research.google.com
2. Create new notebook
3. Change runtime: Runtime → Change runtime type → GPU → Save
4. Copy this code:

```python
# Install CrashLens
!pip install requests

# Download SDK
!git clone https://github.com/kavinsaravan/ML-Failure-Doctor.git
!cd ML-Failure-Doctor/crashlens-sdk && pip install -e .

# Test it!
from crashlens import WorkloadTracker
import torch

tracker = WorkloadTracker("https://invigorating-empathy-production-dee5.up.railway.app")

print(f"GPU Available: {torch.cuda.is_available()}")
print(f"GPU Name: {torch.cuda.get_device_name(0) if torch.cuda.is_available() else 'None'}")

with tracker.track("Google Colab - GPU Test"):
    # Your GPU code here
    x = torch.randn(1000, 1000).cuda()
    y = torch.mm(x, x.T)
    print(f"✅ GPU computation successful!")

print("Check: https://frontend-zeta-eight-92.vercel.app/dashboard")
```

5. Run it!
6. Check your dashboard - you'll see it tracked!

### Option 3: Try RunPod (Pay-as-you-go, Easy Setup)

**If you want to start with real GPUs TODAY:**

1. Go to: https://www.runpod.io
2. Sign up (no waiting for approval)
3. Add $10 credit
4. Deploy a pod:
   - GPU: RTX 3090 ($0.34/hr) or A100 ($1.89/hr)
   - Template: "PyTorch" or "RunPod Pytorch"
5. SSH into your instance
6. Install CrashLens SDK (see Quick Start below)

---

## Once You Get AMD Cloud Access

### Quick Start (15 minutes total)

```bash
# 1. SSH into your AMD instance
ssh ubuntu@<AMD_INSTANCE_IP>

# 2. Verify GPU
rocm-smi
# Should show: AMD Instinct MI210 or MI250

# 3. Install CrashLens
git clone https://github.com/kavinsaravan/ML-Failure-Doctor.git
cd ML-Failure-Doctor/crashlens-sdk
pip3 install -e .

# 4. Install PyTorch with ROCm
pip3 install torch --index-url https://download.pytorch.org/whl/rocm5.7

# 5. Run test workload
python3 << 'EOF'
import torch
from crashlens import WorkloadTracker

tracker = WorkloadTracker("https://invigorating-empathy-production-dee5.up.railway.app")

print(f"GPU: {torch.cuda.get_device_name(0)}")

with tracker.track("AMD Instinct - First Test"):
    model = torch.nn.Linear(1000, 100).cuda()
    x = torch.randn(32, 1000).cuda()

    for i in range(10):
        y = model(x)
        print(f"Step {i+1} complete")

    print("✅ Success on real AMD GPU!")
EOF

# 6. View on dashboard
open https://frontend-zeta-eight-92.vercel.app/dashboard
```

---

## Your Next 7 Days

### Day 1 (Today) ✓
- [x] Applied for AMD Developer Cloud
- [ ] Test CrashLens SDK locally
- [ ] Try Google Colab if you want GPU now

### Day 2-3 (Waiting for AMD Approval)
- [ ] Read docs: `docs/AMD_DEVELOPER_CLOUD_QUICKSTART.md`
- [ ] Explore dashboard features
- [ ] Plan what workloads to test

### Day 4-5 (AMD Access Granted!)
- [ ] Launch AMD Instinct instance
- [ ] Setup CrashLens SDK
- [ ] Run first real workload
- [ ] Test failure scenarios

### Day 6-7 (Production Testing)
- [ ] Run real model training
- [ ] Capture GPU OOM failures
- [ ] Get AI diagnoses
- [ ] Document results

---

## Files Created for You

```
docs/
├── AMD_DEVELOPER_CLOUD_QUICKSTART.md    ← Full setup guide
└── AMD_CLOUD_APPLICATION_TEMPLATE.txt   ← Copy-paste for application

crashlens-sdk/
├── crashlens/
│   ├── workload_tracker.py              ← Track ML jobs
│   └── agent_tracker.py                 ← Track AI agents
├── examples/
│   ├── pytorch_training.py              ← Example scripts
│   └── langchain_agent.py
├── setup.py
└── README.md                            ← SDK documentation

PRODUCTION_SETUP.md                       ← Production deployment guide
```

---

## Quick Reference

### Dashboard
https://frontend-zeta-eight-92.vercel.app/dashboard

### Backend API
https://invigorating-empathy-production-dee5.up.railway.app

### Apply for AMD Cloud
https://www.amd.com/en/technologies/infinity-hub

### SDK Installation
```bash
cd crashlens-sdk && pip install -e .
```

### Track Any Code
```python
from crashlens import WorkloadTracker
tracker = WorkloadTracker("https://invigorating-empathy-production-dee5.up.railway.app")

with tracker.track("My Workload"):
    # Your code here
    pass
```

---

## 🎯 Your Goal This Week

**Get at least 3 real workloads tracked in CrashLens:**

1. ✅ One successful workload
2. ✅ One GPU OOM failure
3. ✅ One ROCm/dependency error

**Then:**
- Run AI diagnosis on each failure
- Document the recommendations
- Share results on GitHub!

---

## Questions?

- **SDK Issues:** Check `crashlens-sdk/README.md`
- **AMD Cloud:** Check `docs/AMD_DEVELOPER_CLOUD_QUICKSTART.md`
- **Dashboard:** https://frontend-zeta-eight-92.vercel.app
- **GitHub:** https://github.com/kavinsaravan/ML-Failure-Doctor

---

**🚀 Ready? Go apply for AMD Cloud access NOW!**

https://www.amd.com/en/technologies/infinity-hub
