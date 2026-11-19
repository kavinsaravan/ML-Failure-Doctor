# 🔍 CrashLens - AI-Powered GPU Workload Failure Diagnosis

![NVIDIA CUDA](https://img.shields.io/badge/NVIDIA-CUDA-76B900.svg)
![AMD ROCm](https://img.shields.io/badge/AMD-ROCm-red.svg)
![Go](https://img.shields.io/badge/Go-1.26-00ADD8.svg)
![Next.js](https://img.shields.io/badge/Next.js-16-black.svg)
![Docker](https://img.shields.io/badge/Docker-Ready-2496ED.svg)

**CrashLens** is an intelligent failure diagnosis platform designed for GPU workloads. It combines real-time GPU metrics collection (nvidia-smi/rocm-smi/), AI-powered root cause analysis, and comprehensive observability for ML training jobs across GPU platforms.

---

## Why CrashLens?

**The Problem:** ML engineers spend hours debugging GPU failures—deciphering cryptic CUDA/HIP errors, analyzing memory dumps, and manually correlating logs with metrics.

**The Solution:** CrashLens diagnoses GPU workload failures in **seconds**, not hours:
-  **AI-Powered Diagnosis** - Fireworks AI user-selected model analyzes logs and provides actionable fixes
-  **Universal GPU Support** - Works with NVIDIA CUDA and AMD ROCm GPUs, locally or in the cloud
-  **Real-time Metrics** - Live GPU memory, utilization, and temperature monitoring
-  **Cost Tracking** - Monitor wasted GPU-seconds on failed jobs
-  **Containerized Prototype** - Docker setup for reproducible deployments

---

##  Key Features

###  GPU Workload Diagnosis
- **Automatic Failure Classification**: GPU OOM, missing checkpoints, dependency errors, data path errors, timeouts, runtime errors
- **AI-Powered Doctor**: AI-powered diagnosis via Fireworks AI providing:
  - Root cause analysis
  - Evidence extraction from logs
  - Recommended fixes with retry safety assessment
  - Prevention strategies
- **Universal GPU Support**:
  - SDK supports NVIDIA CUDA and AMD ROCm on the client side
  - Select one GPU per tracked run, with CUDA_VISIBLE_DEVICES support
  - Device metadata (UUID, index, name) identifies the selected GPU when available
- **Real-time Metrics**: Live GPU memory, utilization, temperature monitoring, and automatic calculation of wasted GPU-seconds
  - SDK workloads record live metrics from the actual training environment; the backend stores and analyzes uploaded telemetry.

###  Jupyter Notebook Integration
CrashLens provides first-class support for Jupyter notebooks:
- **IPython Magic Commands**: Track cells with `%%crashlens_track "Job Name"`
- **Rich HTML Displays**: Color-coded status, formatted diagnosis reports
- **Inline Metrics**: GPU metrics displayed as pandas DataFrames
- **Auto-Diagnosis**: Automatically diagnose failures in tracked cells

### Backend Diagnosis

`POST /workloads/{id}/diagnose` analyzes a failed workload. Ordinary users supply their own Fireworks key through the dashboard, SDK, or request headers. Diagnosis defaults to Fireworks’ `auto` serverless router; an explicit tool-capable model can override it. All API diagnoses use the Fireworks key supplied with the request; backend environment variables do not fund diagnoses. Without a key, the backend returns a rule-based report. A saved AI report is returned without another inference call unless `?refresh=true` is supplied.

---

##  Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                       GPU Workload                          │
│              (PyTorch, TensorFlow, etc.)                    │
└─────────────────┬───────────────────────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────────────────────┐
│            SDK GPU Metric Collector                         │
│   (NVIDIA NVML/CUDA, AMD ROCm)                              │
│   Live metrics uploaded every 2 seconds                     │
└─────────────────┬───────────────────────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────────────────────┐
│               Backend API (Go)                              │
│   State validation, metric storage, private workspaces      │
└─────────────────┬───────────────────────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────────────────────┐
│          Rule-Based Failure Classifier                      │
│      (GPU OOM, CUDA/ROCm errors, Dependencies)              │
└─────────────────┬───────────────────────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────────────────────┐
│             Fireworks AI (BYOK)                             │
│        (Root cause + Fixes + Prevention)                    │
└─────────────────┬───────────────────────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────────────────────┐
│              Next.js Dashboard                              │
│         (Real-time monitoring + Diagnosis UI)               │
└─────────────────────────────────────────────────────────────┘
```

---

## 🛠️ Tech Stack

| Component | Technology | Purpose |
|-----------|-----------|---------|
| **Frontend** | Next.js 16, React, TypeScript, Tailwind CSS | Modern, responsive dashboard |
| **Backend** | Go 1.26, Gorilla Mux | High-performance REST API |
| **Database** | SQLite | Lightweight, embedded persistence |
| **AI Model** | Fireworks AI | Intelligent failure diagnosis |
| **GPU Platform** | **NVIDIA CUDA / AMD ROCm** | Universal GPU metrics collection |
| **Visualization** | Recharts | GPU metrics and performance charts |
| **Containerization** | Docker, Docker Compose | Containerized deployment |

---

##  Quick Start

### Try the hosted service

1. Open your CrashLens dashboard, enter a workspace name, and click **Create my workspace**. Copy and save the generated key; it is shown only once.
2. Open the [sample notebook in Colab](https://colab.research.google.com/github/kavinsaravan/ML-Failure-Doctor/blob/main/sample%20notebook/CrashLens_E2E_Test.ipynb), select a GPU runtime, and run the cells in order. Enter your workspace key when prompted; the hosted backend is selected automatically.
3. Keep the dashboard open with the same workspace key to see the notebook's training run and intentional failures.
4. To generate an AI diagnosis, add your Fireworks key in the dashboard, leave the model as `auto`, and click **Use my Fireworks account**. Open a failed workload and select **Run Diagnosis**. Without a Fireworks key, diagnosis uses rules.

### Track your own workload

Install the SDK on the machine running your code:

```bash
python -m pip install "crashlens[gpu] @ git+https://github.com/kavinsaravan/ML-Failure-Doctor.git#subdirectory=crashlens-sdk"
```

This minimal example sends logs and run status to your workspace. Run your
training code inside the block to collect available GPU telemetry:

```python
from getpass import getpass
from crashlens import WorkloadTracker

tracker = WorkloadTracker(api_key=getpass("CrashLens workspace key: "))
with tracker.track("My workload"):
    print("CrashLens is tracking this run.")
    # Run your training code here.
```

See the [SDK guide](crashlens-sdk/README.md) for GPU dependencies, device selection,
notebook integration, and tracking options.

### Self-hosting and local development

See the [deployment guide](PRODUCTION_SETUP.md) for Docker setup, local Go/Next.js
development, environment variables, and database persistence. When using your
own backend, set `CRASHLENS_URL` or pass `api_url=` to the SDK.

---

## Authentication & Multi-Tenancy

CrashLens supports private workspaces with API key-based isolation. Each key identifies an owner, and the API restricts all operations (list, view, update, delete, diagnose) to that owner's workloads.

For operator key listing, revocation, and security guidance, see
[Operator Key Revocation](PRODUCTION_SETUP.md#operator-key-revocation).

### Bring Your Own AI Key (BYOK)

Users provide their own Fireworks credentials for AI-powered diagnosis.

**Dashboard setup:**
1. Connect with your CrashLens API key
2. Enter your Fireworks key in the dashboard's Fireworks settings; leave the model as `auto` for automatic serverless selection
3. Credentials stay in browser memory only (not localStorage)
4. Re-enter after refresh or session expiration


### Self-Service Workspaces

Users can create their own workspaces directly from the dashboard without operator intervention.

**How it works:**
1. Open the dashboard and select **Create my workspace**
2. Enter a workspace label (descriptive only, not an identity)
3. Receive a randomly generated CrashLens key (shown once)
4. Use the key in the dashboard and SDK

---
