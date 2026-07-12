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

- **Docker** and **Docker Compose** (recommended)
  - OR -
- **Go** 1.22+, **Node.js** 18+, **Python** 3.9+
- **AMD ROCm** (optional, for real GPU metrics)
- **Fireworks AI API Key** ([Get one here](https://fireworks.ai))

### Option 1: Docker (Recommended) 🐳

The fastest way to run CrashLens:

```bash
# Clone the repository
git clone https://github.com/kavinsaravan/ML-Failure-Doctor.git
cd ML-Failure-Doctor

# Create environment file
echo "FIREWORKS_API_KEY=your_api_key_here" > .env

# Start all services
docker-compose up -d

# View logs
docker-compose logs -f

# Access the application
open http://localhost:3000
```

**Services:**
- 🎨 Frontend: http://localhost:3000
- 🔧 Backend API: http://localhost:8080
- ✅ Health Check: http://localhost:8080/health

### Option 2: Local Development

#### 1. Backend Setup
```bash
cd backend
go mod download
export FIREWORKS_API_KEY="your_api_key_here"
go run .
```

#### 2. Frontend Setup
```bash
cd frontend
npm install
npm run dev
```

#### 3. MCP Server Setup
```bash
cd mcp-server
npm install
npm start
```

### 🎬 Create Your First Workload

**Via Dashboard:**
1. Navigate to http://localhost:3000
2. Click "Quick Test Jobs"
3. Run a sample failure (GPU OOM, missing checkpoint, etc.)
4. Click "View Details" → "Run AI Diagnosis"

**Via API:**
```bash
curl -X POST http://localhost:8080/api/workloads/run \
  -H "Content-Type: application/json" \
  -d '{
    "template": "gpu_oom",
    "type": "ML_JOB"
  }'
```

---

## 📊 Dashboard Overview

### GPU Workloads Dashboard
- **Stats Cards**: Total workloads, failures, success rate, wasted GPU time
- **Quick Test Jobs**: One-click failure scenarios for testing
- **Workload Table**: Real-time status, failure types, runtime metrics
- **Detailed Views**: Per-job logs, GPU metrics charts, AI diagnosis

### Agent Runs Dashboard
- **Agent Metrics**: Total runs, success rate, tool/model calls, latency
- **Execution Traces**: Step-by-step agent decision timeline
- **Failure Analysis**: Tool call failures, infinite loop detection
- **Performance Insights**: Token usage, latency patterns

---

## 🔌 API Reference

### Workloads
```http
GET    /api/workloads           # List all workloads
POST   /api/workloads/run       # Create and run workload from template
GET    /api/workloads/{id}      # Get workload details with metrics
POST   /api/workloads/{id}/diagnose  # Run AI diagnosis
```

### Agent Runs
```http
GET    /agent-runs              # List all agent executions
GET    /agent-runs/{id}         # Get agent run details
GET    /agent-runs/{id}/steps   # Get execution trace steps
POST   /agent-runs/{id}/diagnose # Run AI diagnosis on failed agent
```

### Statistics
```http
GET    /api/stats               # Platform-wide statistics
```

### Health
```http
GET    /health                  # Service health check
```

---

## 🔍 Failure Types Supported

| Failure Type | Description | AMD-Specific | Safe to Retry? |
|--------------|-------------|--------------|----------------|
| `GPU_OUT_OF_MEMORY` | HIP out-of-memory error | ✅ Yes | ❌ No |
| `MISSING_CHECKPOINT` | Checkpoint file not found | ⬜ No | ✅ Yes |
| `DEPENDENCY_ERROR` | Python import/package errors | ⬜ No | ✅ Yes |
| `DATA_PATH_ERROR` | Training data not accessible | ⬜ No | ✅ Yes |
| `TIMEOUT` | Job exceeded time limit | ⬜ No | ✅ Yes |
| `ROCM_ERROR` | AMD ROCm/HIP runtime error | ✅ Yes | ⚠️ Maybe |
| `GPU_DRIVER_ERROR` | GPU driver version mismatch | ✅ Yes | ✅ Yes |

---

## 🗄️ Database Schema

### Workloads Table
```sql
CREATE TABLE workloads (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT NOT NULL,              -- ML_JOB or AGENT_RUN
    status TEXT NOT NULL,            -- pending, running, failed, succeeded
    failure_type TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP,
    finished_at TIMESTAMP,
    runtime_seconds REAL,
    exit_code INTEGER,
    wasted_gpu_seconds REAL,

    -- ML Job Data
    job_logs TEXT,                   -- Captured stdout/stderr
    gpu_metrics TEXT,                -- JSON array of rocm-smi snapshots
    checkpoint_state TEXT,
    failure_report TEXT,             -- AI diagnosis JSON

    -- Agent Data
    agent_steps TEXT,
    tool_calls TEXT,
    model_calls TEXT,
    trace_events TEXT
);
```

### Agent Runs Table
```sql
CREATE TABLE agent_runs (
    id TEXT PRIMARY KEY,
    workload_id TEXT NOT NULL,
    agent_name TEXT NOT NULL,
    task TEXT NOT NULL,
    status TEXT NOT NULL,
    failure_type TEXT,
    total_tool_calls INTEGER,
    total_model_calls INTEGER,
    total_tokens INTEGER,
    total_latency_ms INTEGER,
    started_at TIMESTAMP,
    finished_at TIMESTAMP,
    FOREIGN KEY (workload_id) REFERENCES workloads(id)
);
```

---

## 🤖 AI Diagnosis Output

CrashLens generates structured, actionable diagnosis reports:

```json
{
  "root_cause": "GPU Out of Memory - Batch size exceeded available VRAM",
  "evidence": [
    "HIP out of memory error detected in logs",
    "GPU memory peaked at 15.8 GB / 16.0 GB available",
    "Batch size: 128, Model parameters: 175M"
  ],
  "recommended_fixes": [
    "Reduce batch size from 128 to 64 or 32",
    "Enable gradient checkpointing to reduce memory footprint",
    "Use mixed precision training (FP16) with torch.cuda.amp",
    "Consider gradient accumulation for effective large batches"
  ],
  "prevention": "Implement automatic batch size tuning based on available GPU memory. Monitor memory usage during initial epochs and adjust dynamically.",
  "safe_to_retry": false,
  "confidence": 0.95,
  "diagnosed_at": "2026-07-12T03:24:00Z"
}
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

## 🔧 Development

### Local Development Setup
```bash
# Backend (with hot reload)
cd backend
air  # or: go run .

# Frontend (with hot reload)
cd frontend
npm run dev

# MCP Server
cd mcp-server
npm run dev
```

### Running Tests
```bash
# Backend tests
cd backend
go test ./...

# Frontend tests
cd frontend
npm test
```

### Building for Production
```bash
# Backend binary
cd backend
go build -ldflags="-w -s" -o crashlens

# Frontend static build
cd frontend
npm run build
```

---

## 📈 Monitoring & Observability

### GPU Metrics Collected
- **Memory**: Used MB, Total MB, Utilization %
- **Compute**: GPU utilization %
- **Thermal**: Temperature (°C)
- **Temporal**: Timestamp for each snapshot

### Performance Dashboards
- Real-time GPU utilization charts
- Memory usage timeline
- Temperature monitoring
- Wasted GPU-seconds tracking

---

## 🤝 Contributing

We welcome contributions! This project was built for the AMD Developer Hackathon.

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request


---

<div align="center">


[🌐 Live Demo](http://localhost:3000) • [📚 Documentation](./docs) • [🐛 Report Bug](https://github.com/kavinsaravan/ML-Failure-Doctor/issues)

</div>
