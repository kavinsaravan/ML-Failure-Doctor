# 🔍 CrashLens - AI-Powered GPU Workload Failure Diagnosis

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![NVIDIA CUDA](https://img.shields.io/badge/NVIDIA-CUDA-76B900.svg)
![AMD ROCm](https://img.shields.io/badge/AMD-ROCm-red.svg)
![Go](https://img.shields.io/badge/Go-1.22-00ADD8.svg)
![Next.js](https://img.shields.io/badge/Next.js-16-black.svg)
![Docker](https://img.shields.io/badge/Docker-Ready-2496ED.svg)

**CrashLens** is an intelligent failure diagnosis platform designed for GPU workloads. It combines real-time GPU metrics collection (nvidia-smi/rocm-smi), AI-powered root cause analysis, and comprehensive observability for ML training jobs across all GPU platforms.

---

## Why CrashLens?

**The Problem:** ML engineers spend hours debugging GPU failures—deciphering cryptic CUDA/HIP errors, analyzing memory dumps, and manually correlating logs with metrics.

**The Solution:** CrashLens diagnoses GPU workload failures in **seconds**, not hours:
-  **AI-Powered Diagnosis** - Gemma model analyzes logs and provides actionable fixes
-  **Universal GPU Support** - Works with NVIDIA (nvidia-smi), AMD (rocm-smi), and cloud platforms
-  **Real-time Metrics** - Live GPU memory, utilization, and temperature monitoring
-  **Cost Tracking** - Monitor wasted GPU-seconds on failed jobs
-  **Production-Ready** - Fully containerized with Docker

---

##  Key Features

###  GPU Workload Diagnosis
- **Automatic Failure Classification**: GPU OOM, missing checkpoints, dependency errors, data path errors, timeouts, CUDA/ROCm runtime errors
- **AI-Powered Doctor**: Gemma-powered diagnosis via Fireworks AI providing:
  - Root cause analysis
  - Evidence extraction from logs
  - Recommended fixes with retry safety assessment
  - Prevention strategies
- **Universal GPU Support**: Auto-detects NVIDIA (nvidia-smi) or AMD (rocm-smi) GPUs
- **Real-time Metrics**: Live GPU memory, utilization, and temperature monitoring
  - *Note: Demo jobs use simulated metrics. Workloads tracked with the SDK record live GPU memory (NVIDIA and ROCm), plus utilization and temperature on NVIDIA when `nvidia-ml-py` is installed.*
- **Cost Intelligence**: Automatic calculation of wasted GPU-seconds and economic impact

###  Jupyter Notebook Integration
CrashLens provides first-class support for Jupyter notebooks:
- **IPython Magic Commands**: Track cells with `%%crashlens_track "Job Name"`
- **Rich HTML Displays**: Color-coded status, formatted diagnosis reports
- **Inline Metrics**: GPU metrics displayed as pandas DataFrames
- **Auto-Diagnosis**: Automatically diagnose failures in tracked cells
- **Live Progress**: Real-time workload status indicators

See [Jupyter Integration Guide](./docs/JUPYTER_INTEGRATION.md) for detailed usage.

###  Model Context Protocol (MCP) Integration
CrashLens provides an optional MCP server that lets Claude Desktop and other MCP clients query workload data via 7 standardized tools:
- `get_workload_logs` - Retrieve execution logs and error traces
- `get_gpu_metrics` - Access GPU memory, utilization, temperature data
- `get_failure_report` - Get AI-generated diagnosis reports (created by the backend's Gemma 2 integration)
- `get_checkpoint_state` - View checkpoint availability
- `get_wasted_gpu_time` - Calculate failure cost impact
- `list_failed_workloads` - Query failed workloads with filters
- `get_workload_summary` - Get complete workload metadata

**Two separate AI integration paths:**
1. **Backend AI Diagnosis (Gemma 2)**: When you call `POST /workloads/{id}/diagnose`, the Go backend sends logs and metrics directly to Gemma 2 via Fireworks AI (configured with `FIREWORKS_API_KEY` and `FIREWORKS_MODEL`). This produces the failure reports stored in the database.
2. **MCP Client Queries**: External tools like Claude Desktop can use the MCP server to retrieve these reports and other workload data. The MCP server calls the CrashLens REST API; it does not invoke Gemma directly.

See [MCP Server Documentation](./mcp-server/README.md) for setup and detailed tool specifications.

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
│            Log + GPU Metric Collector                       │
│   (nvidia-smi / rocm-smi with simulated fallback)           │
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

##  Project Structure

```
ML-Failure-Doctor/
├── backend/                    # Go backend API
│   ├── main.go                 
│   ├── api/                    
│   │   └── handlers.go
│   ├── db/                     
│   │   └── database.go
│   ├── fireworks/              
│   │   └── client.go
│   ├── metrics/                
│   │   └── collector.go        
│   ├── classifier/             
│   │   └── classifier.go
│   ├── diagnosis/             
│   │   └── diagnosis.go
│   ├── runner/                
│   │   └── runner.go
│   └── go.mod
│
├── frontend/                   # Next.js dashboard
│   ├── app/
│   │   ├── page.tsx
│   │   ├── dashboard/          
│   │   │   └── page.tsx
│   │   └── workloads/[id]/     
│   │       └── page.tsx
│   ├── components/            
│   ├── lib/
│   │   └── api.ts              
│   └── package.json
│
├── crashlens-sdk/              # Python SDK
│   ├── crashlens/
│   │   ├── __init__.py
│   │   └── workload_tracker.py 
│   └── setup.py
│
├── jobs/                       
│   ├── gpu_oom.py
│   ├── dependency_error.py
│   ├── missing_checkpoint.py
│   └── successful_training.py
│
├── docker-compose.yml
├── Dockerfile
└── README.md
```

## 🛠️ Tech Stack

| Component | Technology | Purpose |
|-----------|-----------|---------|
| **Frontend** | Next.js 16, React, TypeScript, Tailwind CSS | Modern, responsive dashboard |
| **Backend** | Go 1.22, Gorilla Mux | High-performance REST API |
| **Database** | SQLite | Lightweight, embedded persistence |
| **AI Model** | Gemma via Fireworks AI | Intelligent failure diagnosis |
| **GPU Platform** | **NVIDIA CUDA / AMD ROCm** | Universal GPU metrics collection |
| **Visualization** | Recharts | GPU metrics and performance charts |
| **Containerization** | Docker, Docker Compose | Production-ready deployment |

---

##  Quick Start

### Prerequisites

#### Option A: Docker (Recommended for Quick Testing)
- **Docker** and **Docker Compose** installed
- **Fireworks AI API Key** ([Get one free here](https://fireworks.ai))

#### Option B: Local Development
- **Go** 1.22+ ([Download](https://go.dev/dl/))
- **Node.js** 18+ ([Download](https://nodejs.org/))
- **Fireworks AI API Key** ([Get one free here](https://fireworks.ai))
- **AMD ROCm** (optional, for real GPU metrics on AMD hardware)

---

### Option 1: Docker (Recommended) 

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
-  **Frontend Dashboard**: http://localhost:3000
-  **Backend API**: http://localhost:8080
-  **Health Check**: http://localhost:8080/health
-  **Database**: SQLite (auto-created in Docker volume)

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

To enable Claude Desktop or other MCP clients to query your CrashLens deployment, add to `~/Library/Application Support/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "crashlens": {
      "command": "node",
      "args": ["/absolute/path/to/ML-Failure-Doctor/mcp-server/server.js"],
      "env": {
        "CRASHLENS_URL": "http://localhost:8080",
        "CRASHLENS_API_KEY": "your-api-key-if-required"
      }
    }
  }
}
```

See [MCP Server Documentation](./mcp-server/README.md) for more details.
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
DELETE /workloads/{id}          # Delete a workload
GET    /workloads/{id}/logs     # Get workload execution logs
GET    /workloads/{id}/metrics  # Get GPU metrics for workload
POST   /workloads/{id}/diagnose # Run AI diagnosis on failure
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
