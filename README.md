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

## AMD ROCm Support

**CrashLens is designed for ROCm-based workloads** and provides first-class support for AMD GPU infrastructure:

- **HIP-style OOM Errors**: Detects and diagnoses `HIP out of memory` errors specific to AMD ROCm runtime
- **AMD GPU Metrics**: Collects memory, utilization, and temperature data via `rocm-smi` compatible interface
- **ROCm 5.7+ Compatible**: Tested with AMD ROCm 5.7+ runtime environment
- **AMD Developer Cloud Ready**: Architecture supports deployment on AMD Developer Cloud infrastructure
- **Pluggable Metric Collector**:
  ```go
  // Backend automatically detects ROCm availability
  if ROCmAvailable() {
      collector = NewROCmSMICollector()  // Real rocm-smi metrics
  } else {
      collector = NewSimulatedCollector()  // Demo metrics for development
  }
  ```

The platform seamlessly switches between real `rocm-smi` metrics (when ROCm is available) and simulated metrics (for local development), making it easy to demo on any hardware while being production-ready for AMD GPU clusters.

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

- **Docker** and **Docker Compose** (recommended)
  - OR -
- **Go** 1.22+
- **Node.js** 18+
- **Python** 3.9+ with PyTorch (ROCm version for AMD GPUs)
- **AMD ROCm** (optional, for GPU workloads)
- **Fireworks AI API Key** (for AI diagnosis)

### Option 1: Docker (Recommended)

The easiest way to run CrashLens is using Docker Compose:

```bash
# Clone the repository
git clone https://github.com/kavinsaravan/ML-Failure-Doctor.git
cd ML-Failure-Doctor

# Create .env file with your API key
echo "FIREWORKS_API_KEY=your_api_key_here" > .env

# Start all services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop all services
docker-compose down
```

The services will be available at:
- Frontend: http://localhost:3000
- Backend API: http://localhost:8080
- Health Check: http://localhost:8080/health

### Option 2: Local Development

#### 1. Backend Setup

```bash
cd backend
go mod download
export FIREWORKS_API_KEY="your_api_key_here"
go run .
```

The backend will start on `http://localhost:8080`

#### 2. Frontend Setup

```bash
cd frontend
npm install
npm run dev
```

The frontend will start on `http://localhost:3000`

#### 3. MCP Server Setup

```bash
cd mcp-server
npm install
npm start
```

### Create Your First Workload

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

## Docker Configuration

### Architecture

The Docker setup uses multi-stage builds for optimal image size and security:

- **Backend**: Go binary built in Alpine Linux (~15MB final image)
- **Frontend**: Next.js standalone output with Node.js runtime
- **MCP Server**: Node.js application with production dependencies only

### Docker Compose Services

| Service | Port | Description |
|---------|------|-------------|
| `backend` | 8080 | Go REST API with SQLite database |
| `frontend` | 3000 | Next.js production build |
| `mcp-server` | - | MCP tool server (stdio mode) |

### Environment Variables

Create a `.env` file in the project root:

```env
FIREWORKS_API_KEY=your_fireworks_api_key_here
```

### Development vs Production

**Development Mode** (with hot reload):
```bash
# Use docker-compose.dev.yml for development
docker-compose -f docker-compose.dev.yml up
```

**Production Mode**:
```bash
# Use default docker-compose.yml
docker-compose up -d
```

### Volume Management

- `crashlens-data`: Persistent SQLite database storage
- `./jobs`: Shared directory for ML job scripts
- `./agents`: Shared directory for agent configurations

### Health Checks

The backend includes a health check endpoint that Docker uses to verify service availability:
```bash
curl http://localhost:8080/health
```

### Troubleshooting

**View logs**:
```bash
docker-compose logs -f [service_name]
```

**Rebuild containers**:
```bash
docker-compose build --no-cache
docker-compose up -d
```

**Reset database**:
```bash
docker-compose down -v  # Removes volumes
docker-compose up -d
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
