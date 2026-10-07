# 🔍 CrashLens - AI-Powered GPU Workload Failure Diagnosis

![NVIDIA CUDA](https://img.shields.io/badge/NVIDIA-CUDA-76B900.svg)
![AMD ROCm](https://img.shields.io/badge/AMD-ROCm-red.svg)
![Go](https://img.shields.io/badge/Go-1.26-00ADD8.svg)
![Next.js](https://img.shields.io/badge/Next.js-16-black.svg)
![Docker](https://img.shields.io/badge/Docker-Ready-2496ED.svg)

**CrashLens** is an intelligent failure diagnosis platform designed for GPU workloads. It combines real-time GPU metrics collection (nvidia-smi/rocm-smi), AI-powered root cause analysis, and comprehensive observability for ML training jobs across all GPU platforms.

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
- **Automatic Failure Classification**: GPU OOM, missing checkpoints, dependency errors, data path errors, timeouts, CUDA/ROCm runtime errors
- **AI-Powered Doctor**: AI-powered diagnosis via Fireworks AI providing:
  - Root cause analysis
  - Evidence extraction from logs
  - Recommended fixes with retry safety assessment
  - Prevention strategies
- **Universal GPU Support**: Auto-detects NVIDIA (nvidia-smi) or AMD (rocm-smi) GPUs
- **Real-time Metrics**: Live GPU memory, utilization, temperature monitoring, and automatic calculation of wasted GPU-seconds
  - *Note: Demo jobs use real NVIDIA/ROCm metrics when available, otherwise explicitly tagged simulated metrics. Workloads tracked with the SDK record live GPU memory (NVIDIA and ROCm), plus utilization and temperature on NVIDIA when `nvidia-ml-py` is installed.*

###  Jupyter Notebook Integration
CrashLens provides first-class support for Jupyter notebooks:
- **IPython Magic Commands**: Track cells with `%%crashlens_track "Job Name"`
- **Rich HTML Displays**: Color-coded status, formatted diagnosis reports
- **Inline Metrics**: GPU metrics displayed as pandas DataFrames
- **Auto-Diagnosis**: Automatically diagnose failures in tracked cells

See [Jupyter Notebook Support](./crashlens-sdk/README.md#jupyter-notebook-support) for usage.

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
1. **Backend AI Diagnosis**: When you call `POST /workloads/{id}/diagnose`, the Go backend sends logs and metrics directly to Fireworks AI (configured with `FIREWORKS_API_KEY` and `FIREWORKS_MODEL`). This produces the failure reports stored in the database.
2. **MCP Client Queries**: External tools like Claude Desktop can use the MCP server to retrieve these reports and other workload data. The MCP server calls the CrashLens REST API.

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
│             Fireworks AI model                              │
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
| **Backend** | Go 1.26, Gorilla Mux | High-performance REST API |
| **Database** | SQLite | Lightweight, embedded persistence |
| **AI Model** | Fireworks AI | Intelligent failure diagnosis |
| **GPU Platform** | **NVIDIA CUDA / AMD ROCm** | Universal GPU metrics collection |
| **Visualization** | Recharts | GPU metrics and performance charts |
| **Containerization** | Docker, Docker Compose | Containerized deployment |

---

##  Quick Start

### Prerequisites

#### Option A: Docker (Recommended for Quick Testing)
- **Docker** and **Docker Compose** installed
- **Fireworks AI API Key** ([Get one free here](https://fireworks.ai))

#### Option B: Local Development
- **Go** 1.26+ ([Download](https://go.dev/dl/))
- **Node.js** 20.9+ ([Download](https://nodejs.org/))
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
cp .env.example .env
# Optional: set FIREWORKS_API_KEY and FIREWORKS_MODEL in .env for AI diagnosis

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

Demo scripts are packaged in the backend image. SQLite is stored in the named
`crashlens-data` volume, which survives container recreation and `docker compose down`.
`docker compose down -v` deletes the stored workloads.

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

## Apple GPU tracking (Metal / MPS)

Run the SDK natively on a Mac with MPS-capable PyTorch; Docker Desktop does not
expose the Apple GPU to the Linux backend. The API can run locally or remotely.
In a Python virtual environment:

```bash
python -m pip install torch -e ./crashlens-sdk
export CRASHLENS_URL=http://localhost:8080
# Set CRASHLENS_API_KEY when the backend requires authentication.
python scripts/validate_gpu.py --device mps
```

Use `model.to("mps")` and move training tensors to `mps` inside your usual
`WorkloadTracker.track(...)` block. The tracker detects MPS automatically.
The validation command trains for ten seconds and requires live `torch.mps`
samples at the API. It refuses CPU fallback; it does not simulate GPU success.

MPS samples report process Metal driver allocations (including caches), tensor
allocations, and the recommended maximum working set. The legacy total-memory
field represents that working-set recommendation, **not physical VRAM**. Ratios
can exceed 100%. Utilization, temperature, and an exact allocation peak are not
available and are not fabricated. Sampling can miss short-lived allocations.
MPS OOM messages are classified as GPU_OUT_OF_MEMORY. GPU memory APIs follow
[PyTorch's MPS documentation](https://docs.pytorch.org/docs/stable/mps.html).

## Individual API keys and private workloads

Each issued CrashLens key identifies an owner. The API restricts lists, details,
logs, metrics, summaries, updates, diagnoses, template runs, and deletion to that
owner. Enter the user key in the dashboard and use the same key in the SDK or MCP
client. Users cannot select ownership through request payloads. No signup or
password login is required.

`CRASHLENS_API_KEY` is the operator key: keep it private. It can issue and revoke
user keys and retains access to preexisting workloads in the legacy workspace.
It does not provide cross-owner workload access. User secrets are generated with
cryptographic randomness and stored only as SHA-256 hashes; they are returned once
at issuance. Keep the SQLite database on persistent storage, since it stores both
workloads and key records. Existing data is migrated automatically.

Issue a separate key for each person (the command prompts for your operator key):

```bash
python scripts/manage_keys.py --api-url https://YOUR-BACKEND issue Alice
python scripts/manage_keys.py --api-url https://YOUR-BACKEND list
python scripts/manage_keys.py --api-url https://YOUR-BACKEND revoke KEY_ID
```

The returned `api_key` is what Alice enters in her SDK and dashboard. To replace
her key while retaining her workloads, issue with `--owner-id OWNER_ID`, then
revoke the old key. Revocation blocks subsequent requests and preserves the data.
The list endpoint never returns secrets or hashes. Operator endpoints are
`POST /api-keys`, `GET /api-keys`, and `DELETE /api-keys/{id}`.

Once any individual keys have been issued, anonymous access is disabled even in
`ACCESS_MODE=demo`; authenticated keys always see only their own workloads. For
public hosting use production mode and private access. This feature isolates
data; per-user spending quotas are not included. User AI diagnoses require their own
Fireworks credentials as described below.

## Use your own Fireworks credits

Connect with your individual CrashLens key, then enter your Fireworks key and an
exact tool-capable model identifier in the dashboard's Fireworks settings. Keys
stay in browser memory (not localStorage), are cleared on disconnect/session
expiration, and are sent only on diagnosis requests. Re-enter them after refresh.
Use HTTPS for deployed APIs. Choose a serverless model accessible to your account.
Setting credentials does not call Fireworks; a fresh diagnosis does.

SDK example (also supported by JupyterWorkloadTracker):

```python
tracker = WorkloadTracker(
    os.environ["CRASHLENS_URL"],
    api_key=os.environ["CRASHLENS_API_KEY"],
    fireworks_api_key=os.environ["FIREWORKS_API_KEY"],
    fireworks_model=os.environ["FIREWORKS_MODEL"],
)
report = tracker.diagnose(workload_id)
```

Diagnosis requests carry `X-Fireworks-API-Key` and `X-Fireworks-Model` headers.
The backend creates a request-scoped client to the fixed Fireworks endpoint;
credentials are never stored in SQLite or included in reports or server logs.
Your key is sent to CrashLens and Fireworks, so only use a backend you trust.
User workloads without a Fireworks key get rule-based reports, never the
operator's credits. Failed user requests also never fall back to the operator's
account. The operator key can still use backend FIREWORKS_API_KEY/FIREWORKS_MODEL
for the legacy workspace. Saved AI reports are reused; Re-run Diagnosis makes a
fresh request with the supplied credentials. Jupyter magic initialization reads
Fireworks credentials from environment variables, never magic-line arguments.

## Self-service workspaces

New visitors open the dashboard and select **Create my workspace**. They enter a
workspace label, receive a randomly generated CrashLens key once, and connect
immediately. The key identifies a new private owner; it cannot claim an existing
workspace or grant operator privileges. Save it to reconnect after a refresh and
use it in the SDK. Anyone with the key shares the workspace. There is no email,
password, or self-service recovery. Labels are descriptive and not identities.

Public `POST /workspaces` accepts only `{ "name": "My workspace" }`. Secrets
remain hashed in the database and are never listed later. Creation is limited to
3 requests/minute per IP (burst 1) and 10/minute globally (burst 5), per backend
process. Forwarded IPs require explicit trusted proxy configuration. Set
`ALLOW_WORKSPACE_CREATION=false` to disable public creation while retaining
operator issuance and existing keys. Run one backend process per SQLite database.
Users supply their own Fireworks credentials; creation makes no paid AI calls.
The database volume must persist keys and workload ownership across deployments.
