# 🔍 CrashLens - AI-Powered GPU Workload Failure Diagnosis

![NVIDIA CUDA](https://img.shields.io/badge/NVIDIA-CUDA-76B900.svg)
![AMD ROCm](https://img.shields.io/badge/AMD-ROCm-red.svg)
![Go](https://img.shields.io/badge/Go-1.26-00ADD8.svg)
![Next.js](https://img.shields.io/badge/Next.js-16-black.svg)
![Docker](https://img.shields.io/badge/Docker-Ready-2496ED.svg)

**CrashLens** is an intelligent failure diagnosis platform designed for GPU workloads. It combines real-time GPU metrics collection (nvidia-smi/rocm-smi/), AI-powered root cause analysis, and comprehensive observability for ML training jobs across all GPU platforms.

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

`POST /workloads/{id}/diagnose` analyzes a failed workload. Ordinary users supply their own Fireworks key through the dashboard, SDK, or request headers. Diagnosis defaults to Fireworks’ `auto` serverless router; an explicit tool-capable model can override it. Operator requests may use backend `FIREWORKS_API_KEY` and optional `FIREWORKS_MODEL`. A saved AI report is returned without another inference call unless `?refresh=true` is supplied.

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

### Use the hosted service

Create a workspace in the dashboard and use its key with
`WorkloadTracker(api_key="your-workspace-key")`. The SDK and example notebooks
select the hosted backend automatically. See the [SDK guide](crashlens-sdk/README.md)
for installation and tracking. Self-hosted users can set `CRASHLENS_URL` or pass
`api_url=` explicitly; the deployment instructions below are for running your own backend.

### Prerequisites

#### Option A: Docker (Recommended for Quick Testing)
- **Docker** and **Docker Compose** installed
- **Fireworks credentials** (optional): Your API key for AI-assisted diagnosis using automatic serverless model selection. Rule-based diagnosis requires no Fireworks credentials.

#### Option B: Local Development
- **Go** 1.26+ ([Download](https://go.dev/dl/))
- **Node.js** 20.9+ ([Download](https://nodejs.org/))
- **Fireworks credentials** (optional): Your API key for AI-assisted diagnosis using automatic serverless model selection. Rule-based diagnosis requires no Fireworks credentials.
- **Python 3** and the SDK on the machine running your workloads
- **C compiler** with CGO enabled for the Go SQLite dependency (Xcode Command Line Tools on macOS, or GCC/build tools on Linux)
- **GPU runtime** (optional): CUDA or ROCm with a compatible PyTorch installation, installed on the machine running your workloads

---

### Option 1: Docker (Recommended) 

**Step-by-step setup:**

```bash
# 1. Clone the repository
git clone https://github.com/kavinsaravan/ML-Failure-Doctor.git
cd ML-Failure-Doctor

# 2. Create an environment file
cp .env.example .env
# Optional: set FIREWORKS_API_KEY for operator AI diagnosis; model defaults to auto
# Workspace users supply their own Fireworks credentials in the dashboard or SDK

# 3. Start all services (backend, frontend, database)
docker compose up --build -d

# 4. Wait for services to start (about 30 seconds)
# Check logs to verify everything is running:
docker compose logs -f

# 5. Access the application
open http://localhost:3000
```

**What's Running:**
-  **Frontend Dashboard**: http://localhost:3000
-  **Backend API**: http://localhost:8080
-  **Health Check**: http://localhost:8080/health
-  **Database**: SQLite (auto-created in Docker volume)

The backend Docker image stores and diagnoses telemetry uploaded by SDK workloads running on your training machine.
SQLite is stored in the named `crashlens-data` volume, which survives container
recreation and `docker compose down`. Use `docker compose down -v` to delete
stored workloads.

`NEXT_PUBLIC_API_URL` must be reachable from the browser. For a remote host, set it
to that host's backend URL in `.env` and rebuild with `docker compose up --build -d`.
The default is `http://localhost:8080`.

**To Stop:**
```bash
docker compose down
```

**To Restart:**
```bash
docker compose restart
```

---

### Option 2: Local Development (For Developers)

**Step 1: Backend Setup**

```bash
# Navigate to backend directory
cd backend

# Download Go dependencies
go mod download

# Optional: operator-only AI diagnosis
export FIREWORKS_API_KEY="your_fireworks_api_key_here"
export FIREWORKS_MODEL="auto"  # Optional; a specific tool-capable model can override this

# Configure an operator key for key management; required in production
export CRASHLENS_API_KEY="your_secure_operator_key"
# Users create separate workspace keys through the dashboard

# Start the backend server (runs on port 8080)
go run .

# You should see: "CrashLens backend on port 8080 (private access)"
```

**Keep this terminal open and running.**

---

**Step 2: Frontend Setup**

Open a **new terminal window** and run:

```bash
# Navigate to frontend directory (from project root)
cd frontend

# Install Node.js dependencies
npm install

# Start the development server (runs on port 3000)
npm run dev

# You should see: "Ready on http://localhost:3000"
```

**Keep this terminal open and running.**

---

## Environment Variables

**Frontend** (`frontend/.env.local`):
```env
NEXT_PUBLIC_API_URL=http://localhost:8080  # Local development
# OR
NEXT_PUBLIC_API_URL=https://your-backend-url.com  # Production
```

**Backend** (set in the shell or your deployment; Compose reads the root `.env`):
```env
# Operator credential: required in production, never distribute it to users
CRASHLENS_API_KEY=your_secure_operator_key

# Optional operator AI diagnosis; model selection defaults to auto
FIREWORKS_API_KEY=
FIREWORKS_MODEL=auto

PORT=8080
DATABASE_PATH=./crashlens.db
ACCESS_MODE=private
APP_ENV=development
ALLOWED_ORIGINS=http://localhost:3000,http://localhost:3001
ALLOW_WORKSPACE_CREATION=true
TRUSTED_PROXY_CIDRS=
```


---

## Authentication & Multi-Tenancy

CrashLens supports private workspaces with API key-based isolation. Each key identifies an owner, and the API restricts all operations (list, view, update, delete, diagnose) to that owner's workloads.

For operator key issuance, revocation, rotation, and security guidance, see
[Operator Key Management](./PRODUCTION_SETUP.md#operator-key-management).

### Bring Your Own AI Key (BYOK)

Users can provide their own Fireworks credentials for AI-powered diagnosis instead of using shared credits.

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
