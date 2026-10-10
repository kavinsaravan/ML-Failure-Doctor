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
-  **Universal GPU Support** - Works with NVIDIA CUDA, AMD (rocm-smi), MPS SDK, and cloud platforms
-  **Real-time Metrics** - Live GPU memory, utilization, and temperature monitoring
-  **Cost Tracking** - Monitor wasted GPU-seconds on failed jobs
-  **Containerized Prototype** - Docker setup for reproducible demos

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
  - Backend auto-detects NVIDIA (nvidia-smi), AMD (rocm-smi), or APPLE MPS for demo job metrics
  - SDK supports NVIDIA CUDA, AMD ROCm, and Apple MPS on client side
  - Select one GPU per tracked run, with CUDA_VISIBLE_DEVICES support
  - Device metadata (UUID, index, name) identifies the selected GPU when available
- **Real-time Metrics**: Live GPU memory, utilization, temperature monitoring, and automatic calculation of wasted GPU-seconds
  - *Note: Backend demo jobs use scenario-based simulation with some randomized values, or real GPU metrics when available. SDK workloads record live metrics from the actual training environment.*

###  Jupyter Notebook Integration
CrashLens provides first-class support for Jupyter notebooks:
- **IPython Magic Commands**: Track cells with `%%crashlens_track "Job Name"`
- **Rich HTML Displays**: Color-coded status, formatted diagnosis reports
- **Inline Metrics**: GPU metrics displayed as pandas DataFrames
- **Auto-Diagnosis**: Automatically diagnose failures in tracked cells

###  Model Context Protocol (MCP) Integration
CrashLens provides an optional MCP server that lets Claude Desktop and other MCP clients query workload data via 7 standardized tools:
- `get_workload_logs` - Retrieve execution logs and error traces
- `get_gpu_metrics` - Access GPU memory, utilization, temperature data
- `get_failure_report` - Get AI-generated diagnosis reports
- `get_checkpoint_state` - View checkpoint availability
- `get_wasted_gpu_time` - Calculate wasted GPU time
- `list_failed_workloads` - Query failed workloads with filters
- `get_workload_summary` - Get complete workload metadata

**Two separate AI integration paths:**
1. **Backend Diagnosis**: `POST /workloads/{id}/diagnose` analyzes a failed workload. Ordinary users supply their own Fireworks key and tool-capable model through the dashboard, SDK, or request headers. Operator requests may use backend `FIREWORKS_API_KEY` and `FIREWORKS_MODEL`, which must both be configured. A saved AI report is returned without another inference call unless `?refresh=true` is supplied.
2. **MCP Client Queries**: External tools like Claude Desktop can use the MCP server to retrieve these reports and other workload data. The MCP server calls the CrashLens REST API.

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
│   (NVIDIA NVML/CUDA, AMD ROCm, Apple MPS)                   │
│   Live metrics uploaded every 2 seconds                     │
└─────────────────┬───────────────────────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────────────────────┐
│               Backend API (Go)                              │
│   State validation, metric storage, demo job runner         │
│   Backend GPU: nvidia-smi/rocm-smi detection                │
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

##  Project Structure

```
ML-Failure-Doctor/
├── backend/
│   ├── main.go
│   ├── api/
│   │   ├── handlers.go
│   │   ├── handlers_test.go
│   │   ├── validation.go
│   │   └── validation_test.go
│   ├── db/
│   │   ├── database.go
│   │   └── database_test.go
│   ├── fireworks/
│   │   ├── client.go
│   │   └── client_test.go
│   ├── metrics/
│   │   ├── collector.go
│   │   └── collector_test.go
│   ├── classifier/
│   │   ├── classifier.go
│   │   └── classifier_test.go
│   ├── diagnosis/
│   │   ├── diagnosis.go
│   │   └── diagnosis_test.go
│   ├── runner/
│   │   ├── runner.go
│   │   └── runner_test.go
│   └── go.mod
│
├── frontend/
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
├── crashlens-sdk/
│   ├── crashlens/
│   │   ├── __init__.py
│   │   ├── workload_tracker.py
│   │   ├── metrics.py
│   │   └── jupyter.py
│   ├── tests/
│   │   ├── test_tracking.py
│   │   ├── test_mps.py
│   │   └── test_regressions.py
│   ├── setup.py
│   └── README.md
│
├── mcp-server/
│   ├── server.js
│   ├── package.json
│   └── README.md
│
├── scripts/
│   ├── validate_gpu.py
│   └── manage_keys.py
│
├── docker-compose.yml
├── Dockerfile
└── README.md
```

## 🛠️ Tech Stack

| Component | Technology | Purpose |
|-----------|-----------|---------|
| **Frontend** | Next.js 16, React, TypeScript, Tailwind CSS | Modern, responsive dashboard |
| **Backend** | Go 1.26, Gorilla Mux | High-performance REST API |
| **Database** | SQLite | Lightweight, embedded persistence |
| **AI Model** | Fireworks AI | Intelligent failure diagnosis |
| **GPU Platform** | **NVIDIA CUDA / AMD ROCm / APPLE MPS** | Universal GPU metrics collection |
| **Visualization** | Recharts | GPU metrics and performance charts |
| **Containerization** | Docker, Docker Compose | Containerized deployment |

---

##  Quick Start

### Prerequisites

#### Option A: Docker (Recommended for Quick Testing)
- **Docker** and **Docker Compose** installed
- **Fireworks credentials** (optional): Your API key and a tool-capable model identifier for AI-assisted diagnosis. Rule-based diagnosis requires neither.

#### Option B: Local Development
- **Go** 1.26+ ([Download](https://go.dev/dl/))
- **Node.js** 20.9+ ([Download](https://nodejs.org/))
- **Fireworks credentials** (optional): Your API key and a tool-capable model identifier for AI-assisted diagnosis. Rule-based diagnosis requires neither.
- **Python 3** to execute the backend’s demo scripts; Python and the SDK are also needed on the machine running your own workloads
- **C compiler** with CGO enabled for the Go SQLite dependency (Xcode Command Line Tools on macOS, or GCC/build tools on Linux)
- **GPU runtime** (optional): CUDA, ROCm, or a PyTorch installation supporting Apple MPS, installed on the machine running your workloads

---

### Option 1: Docker (Recommended) 

**Step-by-step setup:**

```bash
# 1. Clone the repository
git clone https://github.com/kavinsaravan/ML-Failure-Doctor.git
cd ML-Failure-Doctor

# 2. Create an environment file
cp .env.example .env
# Optional: set BOTH FIREWORKS_API_KEY and FIREWORKS_MODEL for operator AI diagnosis
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

Demo scripts are packaged in the Docker image built from the root `Dockerfile`.
SQLite is stored in the named `crashlens-data` volume, which survives container
recreation and `docker compose down`. Use `docker compose down -v` to delete
stored workloads.

`NEXT_PUBLIC_API_URL` must be reachable from the browser. For a remote host, set it
to that host's backend URL in `.env` and rebuild with `docker compose up --build -d`.
The default is `http://localhost:8080`.

The optional MCP service uses stdio and is launched by an MCP client, rather than
started with the dashboard. See [Docker MCP setup](./mcp-server/README.md#docker).

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

# Optional: operator-only AI diagnosis (both values are required)
export FIREWORKS_API_KEY="your_fireworks_api_key_here"
export FIREWORKS_MODEL="accounts/fireworks/models/YOUR_TOOL_CAPABLE_MODEL"

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

**Step 3: (Optional) MCP Server Setup**

To enable Claude Desktop or other MCP clients to query your CrashLens deployment:

```bash
cd mcp-server && npm install
```

Then add to `~/Library/Application Support/Claude/claude_desktop_config.json`:

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

**Test the setup** with MCP Inspector before configuring Claude Desktop:

```bash
CRASHLENS_URL=https://your-backend.up.railway.app npx @modelcontextprotocol/inspector node server.js
```

See [MCP Server Documentation](./mcp-server/README.md) for more details.

---

## 🔌 API Reference

**Base URL**: `http://localhost:8080` (local) or your deployed backend URL

Authenticated requests use `Authorization: Bearer <CrashLens workspace key>`.
Workload operations and statistics are restricted to the connected workspace.
`/health` and workspace creation are public; key management requires the operator key.

### Workloads
```http
GET    /workloads               # List workloads in the connected workspace
POST   /workloads               # Create a new workload
POST   /workloads/run           # Queue a predefined demo template
GET    /workloads/{id}          # Get workload details with metrics
PUT    /workloads/{id}          # Update workload status/data
DELETE /workloads/{id}          # Delete a workload
DELETE /workloads/clear         # Clear the connected workspace’s workloads
GET    /workloads/{id}/logs     # Get workload execution logs
GET    /workloads/{id}/metrics  # Get GPU metrics for workload
POST   /workloads/{id}/diagnose # Diagnose a failed workload using AI or rules
```

### Statistics
```http
GET    /summary                 # Workspace statistics; backend_gpu_platform identifies the backend host
```

### Health
```http
GET    /health                  # Status, auth_required, access_mode, workspace_creation_enabled
```

### Workspaces and Access
```http
POST   /workspaces              # Public creation of a new private workspace; returns a key once
GET    /session                 # Validate the connected key and retrieve its owner/admin status
```

### Operator Key Management
```http
POST   /api-keys                # Issue a user key: {"name":"Alice"}, optionally existing owner_id
GET    /api-keys                # List issued key metadata (never secrets)
DELETE /api-keys/{id}           # Revoke a user key
```

For new workspaces, send `{ "name": "My workspace" }` to `POST /workspaces`.
Creation can be disabled with `ALLOW_WORKSPACE_CREATION=false`.
The operator key manages keys and accesses legacy workloads; it does not grant
cross-workspace workload access. See [Authentication & Multi-Tenancy](#authentication--multi-tenancy).

### API Validation

The backend enforces comprehensive input validation:

**State Transitions:**
- Only valid status transitions are allowed (e.g., `pending` → `running` → `failed/succeeded`)
- Completed workloads (`failed` or `succeeded`) cannot revert to `running` or `pending`
- Invalid transitions return HTTP 409 Conflict

**GPU Metrics:**
- Maximum 300 samples per update request
- Required fields: `gpu_memory_used_mb`, `gpu_memory_total_mb`, `gpu_memory_percent`
- Numeric validation: utilization 0-100%, memory values non-negative
- Metrics must be valid JSON array of objects

**Workload Identity:**
- Workload `name` and `type` cannot be changed after creation
- Runtime and wasted GPU seconds must be non-negative
- Type must be `ML_JOB`

**Example API Calls:**

```bash
export CRASHLENS_URL="http://localhost:8080"
# Use the personal workspace key returned by the dashboard or POST /workspaces
export CRASHLENS_API_KEY="YOUR_WORKSPACE_KEY"

# Create a workload; save the numeric id returned in the response
curl -X POST "$CRASHLENS_URL/workloads" \
  -H "Authorization: Bearer $CRASHLENS_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"name":"Training Job","type":"ML_JOB","status":"running"}'

# Replace this with the actual id from the previous response
export WORKLOAD_ID="123"

# Record a failure before requesting diagnosis
curl -X PUT "$CRASHLENS_URL/workloads/$WORKLOAD_ID" \
  -H "Authorization: Bearer $CRASHLENS_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"status":"failed","exit_code":1,"runtime_seconds":4,"job_logs":"RuntimeError: CUDA out of memory"}'

# Optional AI diagnosis: set your own Fireworks credentials before this request
curl -X POST "$CRASHLENS_URL/workloads/$WORKLOAD_ID/diagnose" \
  -H "Authorization: Bearer $CRASHLENS_API_KEY" \
  -H "X-Fireworks-API-Key: $FIREWORKS_API_KEY" \
  -H "X-Fireworks-Model: $FIREWORKS_MODEL"

# List workloads in your workspace
curl "$CRASHLENS_URL/workloads" \
  -H "Authorization: Bearer $CRASHLENS_API_KEY"
```

AI diagnosis requires a tool-capable model that your Fireworks account can access.
If inference fails, the backend returns a rule-based report with an
`ai_unavailable_reason`. Add `?refresh=true` to explicitly regenerate a cached AI
report; a successful inference consumes the supplied account’s credits.

---

### Environment Variables

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

# Optional operator AI diagnosis; both are required to enable it
FIREWORKS_API_KEY=
FIREWORKS_MODEL=

PORT=8080
DATABASE_PATH=./crashlens.db
ACCESS_MODE=private
APP_ENV=development
ALLOWED_ORIGINS=http://localhost:3000,http://localhost:3001
ALLOW_WORKSPACE_CREATION=true
TRUSTED_PROXY_CIDRS=
JOB_CONCURRENCY=2
JOB_QUEUE_SIZE=16
JOB_TIMEOUT_SECONDS=300
```


---

## Apple GPU Tracking (Metal / MPS)

The SDK supports Apple Silicon GPUs via MPS (Metal Performance Shaders). Docker Desktop does not expose the Apple GPU to Linux containers, so run the SDK natively on your Mac.

**Setup:**
```bash
python -m pip install torch -e ./crashlens-sdk
export CRASHLENS_URL=http://localhost:8080
export CRASHLENS_API_KEY=your_key_here  # If authentication is enabled
python scripts/validate_gpu.py --device mps
```

**Usage:**
```python
# The tracker detects MPS automatically
with tracker.track("Training") as workload_id:
    model = model.to("mps")
    train(model)
```

**MPS Metrics:**
- **Driver Memory**: Metal driver allocations (including caches)
- **Tensor Memory**: Actual PyTorch tensor allocations
- **Working Set**: Apple's recommended maximum (**not physical VRAM**)
- **Percentage**: Relative to working set (can exceed 100%)

**Limitations**: Utilization, temperature, and exact peak allocation are unavailable from Apple's APIs and are not simulated. See [PyTorch MPS docs](https://docs.pytorch.org/docs/stable/mps.html) for details.


## Authentication & Multi-Tenancy

CrashLens supports private workspaces with API key-based isolation. Each key identifies an owner, and the API restricts all operations (list, view, update, delete, diagnose) to that owner's workloads.

For operator key issuance, revocation, rotation, and security guidance, see
[Operator Key Management](./PRODUCTION_SETUP.md#operator-key-management).

### Bring Your Own AI Key (BYOK)

Users can provide their own Fireworks credentials for AI-powered diagnosis instead of using shared credits.

**Dashboard setup:**
1. Connect with your CrashLens API key
2. Enter your Fireworks key and model in the dashboard's Fireworks settings
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

# Failure Diagnosis Evaluation

## Recorded Results

| Metric | Rule-based | Fireworks-assisted |
|---|---:|---:|
| Correct classifications | 90 / 100 | 90 / 100 |
| Classification accuracy | 90.0% | 90.0% |
| Macro F1 | 87.1% | 87.1% |

The Fireworks run used `accounts/fireworks/models/gpt-oss-120b`, available through
[serverless inference with function calling](https://fireworks.ai/models/fireworks/gpt-oss-120b).
Both runs used the same dataset hash. 

| Expected category | Correct / cases (both runs) |
|---|---:|
| GPU out of memory | 10 / 10 |
| Missing checkpoint | 13 / 13 |
| Dependency error | 14 / 14 |
| Data path error | 13 / 13 |
| Timeout | 10 / 10 |
| ROCm runtime error | 6 / 10 |
| CUDA runtime error | 10 / 10 |
| GPU driver error | 6 / 10 |
| Unknown / outside supported categories | 8 / 10 |

## Dataset and Limitations

The set contains **90 authored log fixtures and 10 executed GPU failures**
(missing imports, checkpoints, and data files).
