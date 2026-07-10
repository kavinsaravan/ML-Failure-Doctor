# CrashLens - AI Reliability Platform for AMD GPU Workloads

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![AMD ROCm](https://img.shields.io/badge/AMD-ROCm-red.svg)
![Go](https://img.shields.io/badge/Go-1.22-00ADD8.svg)
![Next.js](https://img.shields.io/badge/Next.js-16-black.svg)

**CrashLens** is an AI-powered reliability platform designed for AMD GPU workloads. It provides intelligent failure diagnosis for ML training/inference jobs and extends into agent observability by tracking tool calls, model calls, latency, and failure traces.

Built for the [AMD Developer Hackathon Act II](https://lablab.ai/ai-hackathons/amd-developer-hackathon-act-ii) - Track 3: AI Agent Observability & Reliability.

## Core Features

### ML Job Failure Diagnosis
- **Automatic Failure Classification**: GPU OOM, missing checkpoint, dependency errors, data path errors, timeouts, ROCm errors
- **AI-Powered Doctor**: Gemma-powered diagnosis via Fireworks AI providing root cause analysis, evidence extraction, and recommended fixes
- **AMD ROCm Support**: Native support for HIP/ROCm-style GPU failure logs and rocm-smi metrics
- **GPU Cost Tracking**: Calculates wasted GPU-seconds on failed jobs

### Agent Observability (MVP)
- **Agent Run Traces**: Track tool calls, model calls, and execution steps
- **Failure Detection**: Identify tool-call failures and infinite loops
- **Unified Dashboard**: Same diagnosis format for both ML jobs and agent runs

### MCP Tool Layer
Exposes diagnostic capabilities through Model Context Protocol (MCP):
- `get_workload_logs` - Retrieve workload execution logs
- `get_gpu_metrics` - Access GPU memory, utilization, and temperature metrics
- `get_failure_report` - Get AI-generated diagnosis reports
- `get_checkpoint_state` - View checkpoint paths and availability
- `get_wasted_gpu_time` - Calculate cost impact of failures
- `list_failed_workloads` - Query failed workloads with filters
- `get_workload_summary` - Get comprehensive workload metadata

See [MCP Server Documentation](./mcp-server/README.md) for detailed tool specifications.

## Architecture

```
ML Job / Agent Run
        ↓
Log + Metric + Trace Collector
        ↓
Rule-Based Failure Classifier
        ↓
MCP Tool Server
        ↓
Gemma AI (via Fireworks AI)
        ↓
Next.js Dashboard
```

## Tech Stack

| Component | Technology |
|-----------|-----------|
| **Frontend** | Next.js 16, React, TypeScript, Tailwind CSS, Recharts |
| **Backend** | Go 1.22, Gorilla Mux (REST API) |
| **Database** | SQLite |
| **AI Model** | Gemma via Fireworks AI |
| **GPU Platform** | AMD ROCm / AMD Developer Cloud |
| **Tool Layer** | Model Context Protocol (MCP) |
| **ML Scripts** | Python with PyTorch (ROCm) |

## Project Structure

```
ML-Failure-Doctor/
├── backend/              # Go REST API server
│   ├── main.go          # Server setup, routes, database
│   ├── classifier.go    # Rule-based failure classification
│   ├── diagnosis.go     # AI diagnosis with Fireworks AI
│   └── go.mod           # Go dependencies
├── frontend/            # Next.js dashboard
│   ├── app/            # App router pages
│   ├── components/     # React components
│   └── package.json    # Node dependencies
├── mcp-server/         # MCP tool server
│   ├── server.js       # MCP tool implementations
│   └── package.json    # MCP SDK dependencies
├── scripts/            # Python ML job runners
│   └── [training scripts with ROCm support]
└── docs/              # Documentation
```

## Quick Start

### Prerequisites

- **Go** 1.22+
- **Node.js** 18+
- **Python** 3.9+ with PyTorch (ROCm version for AMD GPUs)
- **AMD ROCm** (optional, for GPU workloads)
- **Fireworks AI API Key** (for AI diagnosis)

### 1. Backend Setup

```bash
cd backend
go mod download
export FIREWORKS_API_KEY="your_api_key_here"
go run .
```

The backend will start on `http://localhost:8080`

### 2. Frontend Setup

```bash
cd frontend
npm install
npm run dev
```

The frontend will start on `http://localhost:3000`

### 3. MCP Server Setup

```bash
cd mcp-server
npm install
npm start
```

### 4. Create Your First Workload

**Option A: Via API**
```bash
curl -X POST http://localhost:8080/api/workloads \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Training ResNet-50",
    "type": "ML_JOB",
    "status": "running"
  }'
```

**Option B: Via Dashboard**
Navigate to `http://localhost:3000` and use the UI to create workloads.

## API Endpoints

### Workloads
- `GET /api/workloads` - List all workloads
- `POST /api/workloads` - Create new workload
- `GET /api/workloads/{id}` - Get workload details
- `PUT /api/workloads/{id}` - Update workload
- `POST /api/workloads/{id}/diagnose` - Run AI diagnosis

### Statistics
- `GET /api/stats` - Get platform statistics (total workloads, failures, wasted GPU-seconds)

### Health
- `GET /health` - Health check

## Failure Types Supported

| Failure Type | Description | Safe to Retry? |
|--------------|-------------|----------------|
| `gpu_oom` | GPU Out of Memory | No |
| `missing_checkpoint` | Checkpoint file not found | Yes |
| `dependency_error` | Python import/package errors | Yes |
| `data_path_error` | Training data not accessible | Yes |
| `timeout` | Job exceeded time limit | Yes |
| `rocm_error` | AMD ROCm/HIP runtime error | Yes |
| `gpu_driver_error` | GPU driver version mismatch | Yes |

## Database Schema

### Workloads Table
```sql
CREATE TABLE workloads (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT NOT NULL,              -- ML_JOB or AGENT_RUN
    status TEXT NOT NULL,            -- pending, running, failed, succeeded
    failure_type TEXT,
    created_at TIMESTAMP,
    started_at TIMESTAMP,
    finished_at TIMESTAMP,
    runtime_seconds REAL,
    exit_code INTEGER,
    wasted_gpu_seconds REAL,

    -- ML Job Data
    job_logs TEXT,
    gpu_metrics TEXT,
    checkpoint_state TEXT,
    failure_report TEXT,

    -- Agent Data
    agent_steps TEXT,
    tool_calls TEXT,
    model_calls TEXT,
    trace_events TEXT
);
```

## AI Diagnosis Format

```json
{
  "root_cause": "GPU Out of Memory - batch size exceeded GPU capacity",
  "evidence": [
    "RuntimeError: CUDA out of memory",
    "Allocated 15.2 GB / Available 16.0 GB"
  ],
  "recommended_fix": "1. Reduce batch size\n2. Enable gradient checkpointing\n3. Use FP16 mixed precision",
  "safe_to_retry": false,
  "diagnosed_at": "2026-07-09T02:30:00Z"
}
```

## Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `FIREWORKS_API_KEY` | Fireworks AI API key for Gemma model | No (falls back to rule-based) |
| `PORT` | Backend server port | No (default: 8080) |


## AMD ROCm Integration

CrashLens is designed to work seamlessly with AMD GPUs:

- **Log Parsing**: Recognizes HIP/ROCm error patterns
- **Metrics Collection**: Supports `rocm-smi` output format
- **Compatibility Detection**: Identifies ROCm version mismatches
- **GPU Discovery**: Works with AMD GPU worker nodes

Example ROCm metrics:
```bash
rocm-smi --showmeminfo vram
```

## Contributing

This project was built for the AMD Developer Hackathon. Contributions welcome!

1. Fork the repository
2. Create a feature branch
3. Commit your changes
4. Push to the branch
5. Open a Pull Request

## License

MIT License - see LICENSE file for details

## Acknowledgments

- **AMD** for ROCm platform and hackathon opportunity
- **Fireworks AI** for Gemma model hosting
- **Model Context Protocol** for standardized tool interfaces

## Support

For issues and questions:
- GitHub Issues: [ML-Failure-Doctor Issues](https://github.com/kavinsaravan/ML-Failure-Doctor/issues)
- Hackathon Discord: AMD Developer Hackathon Act II

---

Built with ❤️ for AMD GPU reliability
