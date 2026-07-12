# 🔍 CrashLens - AI-Powered Reliability Platform for AMD GPUs

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![AMD ROCm](https://img.shields.io/badge/AMD-ROCm%205.7%2B-red.svg)
![Go](https://img.shields.io/badge/Go-1.22-00ADD8.svg)
![Next.js](https://img.shields.io/badge/Next.js-16-black.svg)
![Docker](https://img.shields.io/badge/Docker-Ready-2496ED.svg)

**CrashLens** is an intelligent failure diagnosis and observability platform designed specifically for **AMD GPU workloads**. It combines real-time GPU metrics from `rocm-smi`, AI-powered root cause analysis, and comprehensive observability for both ML training jobs and AI agent executions.

> Built for the [AMD Developer Hackathon Act II](https://lablab.ai/ai-hackathons/amd-developer-hackathon-act-ii) - **Track 3: Unicorn Track**

---

## 🎯 Why CrashLens?

**The Problem:** ML engineers spend hours debugging GPU failures—deciphering cryptic HIP errors, analyzing memory dumps, and manually correlating logs with metrics.

**The Solution:** CrashLens diagnoses GPU workload failures in **seconds**, not hours:
- 🤖 **AI-Powered Diagnosis** - Gemma model analyzes logs and provides actionable fixes
- 📊 **Real-time AMD GPU Metrics** - Native `rocm-smi` integration for memory, utilization, and temperature
- 🔬 **Agent Observability** - Track tool calls, model interactions, and execution traces
- 💰 **Cost Tracking** - Monitor wasted GPU-seconds on failed jobs
- 🐳 **Production-Ready** - Fully containerized with Docker

---

## ✨ Key Features

### 🎯 GPU Workload Diagnosis
- **Automatic Failure Classification**: GPU OOM, missing checkpoints, dependency errors, data path errors, timeouts, ROCm runtime errors
- **AI-Powered Doctor**: Gemma-powered diagnosis via Fireworks AI providing:
  - Root cause analysis
  - Evidence extraction from logs
  - Recommended fixes with retry safety assessment
  - Prevention strategies
- **Real-time Metrics**: Live GPU memory, utilization, and temperature monitoring via `rocm-smi`
- **Cost Intelligence**: Automatic calculation of wasted GPU-seconds and economic impact

### 🤖 AI Agent Observability
- **Execution Traces**: Visual timeline of tool calls, model calls, and decision points
- **Performance Metrics**: Track latency, token usage, and model call patterns
- **Failure Detection**: Identify infinite loops, API errors, and reasoning failures
- **Unified Dashboard**: Same diagnostic interface for both GPU jobs and agent runs

### 🔧 Model Context Protocol (MCP) Integration
CrashLens exposes diagnostic capabilities through standardized MCP tools:
- `get_workload_logs` - Retrieve execution logs and error traces
- `get_gpu_metrics` - Access GPU memory, utilization, temperature data
- `get_failure_report` - Get AI-generated diagnosis reports
- `get_checkpoint_state` - View checkpoint availability
- `get_wasted_gpu_time` - Calculate failure cost impact
- `list_failed_workloads` - Query failed workloads with filters

See [MCP Server Documentation](./mcp-server/README.md) for detailed tool specifications.

---

## 🚀 AMD ROCm Platform Integration

**CrashLens is purpose-built for AMD GPU infrastructure** with deep ROCm ecosystem integration:

### Native AMD Support
- ✅ **HIP Error Detection** - Recognizes and diagnoses HIP out-of-memory and runtime errors
- ✅ **rocm-smi Metrics** - Real-time GPU metrics via AMD's `rocm-smi` utility
- ✅ **ROCm 5.7+ Compatible** - Tested with latest AMD ROCm runtime
- ✅ **AMD Developer Cloud Ready** - Designed for deployment on AMD infrastructure
- ✅ **Intelligent Fallback** - Seamless switch between real rocm-smi and simulated metrics for development

### AMD-Specific Features
```go
// Automatic AMD GPU detection and metric collection
if ROCmAvailable() {
    collector = NewROCmSMICollector()  // Native rocm-smi integration
} else {
    collector = NewSimulatedCollector()  // Development fallback
}
```

### ROCm Error Patterns Detected
- `HIP out of memory` errors
- ROCm runtime failures
- GPU driver version mismatches
- AMD GPU utilization bottlenecks
- Memory bandwidth saturation

---

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     GPU Workload / Agent Run                │
└─────────────────┬───────────────────────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────────────────────┐
│           Log + Metric + Trace Collector                    │
│           (rocm-smi + Python + Agent Traces)                │
└─────────────────┬───────────────────────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────────────────────┐
│          Rule-Based Failure Classifier                      │
│          (Pattern matching + Error detection)               │
└─────────────────┬───────────────────────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────────────────────┐
│               MCP Tool Server                               │
│          (Standardized diagnostic tools)                    │
└─────────────────┬───────────────────────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────────────────────┐
│            Gemma AI (via Fireworks AI)                      │
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
| **Backend** | Go 1.22, Gorilla Mux | High-performance REST API |
| **Database** | SQLite | Lightweight, embedded persistence |
| **AI Model** | Gemma via Fireworks AI | Intelligent failure diagnosis |
| **GPU Platform** | **AMD ROCm 5.7+** | GPU metrics and error detection |
| **Tool Protocol** | Model Context Protocol (MCP) | Standardized AI tool interface |
| **Visualization** | Recharts | GPU metrics and performance charts |
| **Containerization** | Docker, Docker Compose | Production-ready deployment |

---

## 🚀 Quick Start

### Prerequisites

**Choose one of these options:**

#### Option A: Docker (Recommended for Quick Testing)
- **Docker** and **Docker Compose** installed
- **Fireworks AI API Key** ([Get one free here](https://fireworks.ai))

#### Option B: Local Development
- **Go** 1.22+ ([Download](https://go.dev/dl/))
- **Node.js** 18+ ([Download](https://nodejs.org/))
- **Fireworks AI API Key** ([Get one free here](https://fireworks.ai))
- **AMD ROCm** (optional, for real GPU metrics on AMD hardware)

---

### Option 1: Docker (Recommended) 🐳

**Step-by-step setup:**

```bash
# 1. Clone the repository
git clone https://github.com/kavinsaravan/ML-Failure-Doctor.git
cd ML-Failure-Doctor

# 2. Create environment file with your Fireworks AI API key
echo "FIREWORKS_API_KEY=your_api_key_here" > .env

# 3. Start all services (backend, frontend, database)
docker-compose up -d

# 4. Wait for services to start (about 30 seconds)
# Check logs to verify everything is running:
docker-compose logs -f

# 5. Access the application
open http://localhost:3000
```

**What's Running:**
- 🎨 **Frontend Dashboard**: http://localhost:3000
- 🔧 **Backend API**: http://localhost:8080
- ✅ **Health Check**: http://localhost:8080/health
- 💾 **Database**: SQLite (auto-created in Docker volume)

**To Stop:**
```bash
docker-compose down
```

**To Restart:**
```bash
docker-compose restart
```

---

### Option 2: Local Development (For Developers)

**Step 1: Backend Setup**

```bash
# Navigate to backend directory
cd backend

# Download Go dependencies
go mod download

# Set your Fireworks AI API key (get free key at fireworks.ai)
export FIREWORKS_API_KEY="your_fireworks_api_key_here"

# Start the backend server (runs on port 8080)
go run .

# You should see: "CrashLens Backend starting on port 8080"
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

**Step 3: (Optional) MCP Server Setup**

For advanced Model Context Protocol features, open a **third terminal**:

```bash
# Navigate to MCP server directory (from project root)
cd mcp-server

# Install dependencies
npm install

# Start the MCP server
npm start
```

---

### 🎬 Create Your First Workload

**Via Dashboard (Easiest Method):**

1. **Open the Dashboard**
   - Navigate to http://localhost:3000
   - Click "Get Started" to enter the dashboard

2. **Generate Test Workloads**
   - Look for the "Quick Test Jobs" section
   - Click on any test button:
     - **GPU OOM Test** - Simulates out-of-memory failure
     - **Dependency Error** - Missing Python package
     - **Missing Checkpoint** - File not found error
     - **Successful Job** - Completes without errors

3. **View Results**
   - The workload appears in the table below
   - Click "View Details" to see logs and GPU metrics
   - For failed jobs, click "Run AI Diagnosis" to get AI-powered fixes

**Via API (Advanced):**

Test the backend directly with curl:

```bash
# Check backend is running
curl http://localhost:8080/health

# Create a test workload
curl -X POST http://localhost:8080/workloads/run \
  -H "Content-Type: application/json" \
  -d '{
    "template": "gpu_oom",
    "type": "ML_JOB"
  }'

# List all workloads
curl http://localhost:8080/workloads
```

---

## 🔌 API Reference

**Base URL**: `http://localhost:8080` (local) or your deployed backend URL

### Workloads
```http
GET    /workloads               # List all GPU workloads
POST   /workloads               # Create a new workload
POST   /workloads/run           # Create and run workload from template
GET    /workloads/{id}          # Get workload details with metrics
PUT    /workloads/{id}          # Update workload status/data
GET    /workloads/{id}/logs     # Get workload execution logs
GET    /workloads/{id}/metrics  # Get GPU metrics for workload
POST   /workloads/{id}/diagnose # Run AI diagnosis on failure
```

### Agent Runs
```http
GET    /agent-runs              # List all agent executions
POST   /agent-runs              # Create a new agent run
GET    /agent-runs/{id}         # Get agent run details
PUT    /agent-runs/{id}         # Update agent run status
GET    /agent-runs/{id}/steps   # Get execution trace steps
POST   /agent-runs/{id}/diagnose # Run AI diagnosis on failed agent
POST   /agent-steps             # Create agent step in trace
```

### Statistics
```http
GET    /summary                 # Platform-wide statistics and metrics
```

### Health
```http
GET    /health                  # Service health check (returns {"status":"ok"})
```

**Example API Calls:**

```bash
# Create a workload
curl -X POST http://localhost:8080/workloads \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Training Job",
    "type": "ML_JOB",
    "status": "running"
  }'

# Get AI diagnosis for a failed workload
curl -X POST http://localhost:8080/workloads/1/diagnose

# List all workloads
curl http://localhost:8080/workloads
```
---

## 🐳 Docker Configuration

### Multi-Stage Builds
Optimized for production with minimal image sizes:

- **Backend**: Go binary in Alpine Linux (~15MB)
- **Frontend**: Next.js standalone output (~100MB)
- **MCP Server**: Node.js production build (~50MB)

### Services Architecture

| Service | Port | Description | Health Check |
|---------|------|-------------|--------------|
| `backend` | 8080 | Go REST API + SQLite | ✅ /health endpoint |
| `frontend` | 3000 | Next.js production build | Depends on backend |
| `mcp-server` | - | MCP tool server (stdio) | Depends on backend |

### Volume Persistence
```yaml
volumes:
  crashlens-data:      # SQLite database
  ./jobs:              # ML job scripts (mounted)
  ./agents:            # Agent configurations (mounted)
```

### Environment Variables

Create `.env` file:
```env
FIREWORKS_API_KEY=your_fireworks_api_key_here
PORT=8080
NODE_ENV=production
```

---

## 🌐 Deployment

### Vercel Deployment (Frontend)

The frontend is deployed on Vercel for easy access:

**Live Demo**: https://frontend-zeta-eight-92.vercel.app

**Deploy Your Own:**

```bash
# Install Vercel CLI
npm install -g vercel

# Login to Vercel
vercel login

# Deploy from the frontend directory
cd frontend
vercel --prod

# Set environment variable for backend API
vercel env add NEXT_PUBLIC_API_URL production
# Enter your backend URL (e.g., https://your-backend.fly.io or ngrok URL)
```

### Backend Deployment Options

**Option 1: Railway** (Recommended for Go backends)
1. Create account at [railway.app](https://railway.app)
2. Connect your GitHub repository
3. Add `FIREWORKS_API_KEY` environment variable
4. Railway auto-detects Go and deploys

**Option 2: Fly.io**
```bash
# Install flyctl
curl -L https://fly.io/install.sh | sh

# Login and deploy
cd backend
fly launch
fly secrets set FIREWORKS_API_KEY=your_key_here
fly deploy
```

**Option 3: ngrok (Quick Testing)**
```bash
# Start backend locally
cd backend && go run .

# In another terminal, expose with ngrok
ngrok http 8080

# Use the ngrok URL (e.g., https://xyz.ngrok-free.dev) as NEXT_PUBLIC_API_URL
```

### Environment Variables

**Frontend** (`frontend/.env.local`):
```env
NEXT_PUBLIC_API_URL=http://localhost:8080  # Local development
# OR
NEXT_PUBLIC_API_URL=https://your-backend-url.com  # Production
```

**Backend** (set in shell or Docker):
```env
FIREWORKS_API_KEY=your_fireworks_api_key_here
PORT=8080  # Optional, defaults to 8080
```

---

## 🤝 Contributing

We welcome contributions! This project was built for the AMD Developer Hackathon.

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request


---
