# CrashLens MCP Server

Model Context Protocol (MCP) server that lets Claude Desktop and other MCP clients query CrashLens deployments for workload debugging.

## Overview

This MCP server exposes 7 tools that give AI assistants structured access to CrashLens workload data via the REST API:
- Workload logs and execution traces
- GPU memory and utilization metrics
- AI-generated failure diagnosis reports
- Checkpoint states
- Wasted GPU time calculations

## Installation

```bash
cd mcp-server
npm install
```

## Configuration

The server requires environment variables to connect to your CrashLens backend:

- **CRASHLENS_URL** (required): Backend base URL (e.g., `https://your-backend.up.railway.app`). Trailing slashes are automatically stripped.
- **CRASHLENS_API_KEY** (optional): If your backend requires authentication, set this to send `Authorization: Bearer <key>` on every request.

If `CRASHLENS_URL` is not set, the server will log a clear error to stderr and exit immediately.

## Available Tools

### 1. `get_workload_logs`
Retrieve job execution logs for a specific workload.

**Arguments:**
- `workload_id` (number, required): The ID of the workload
- `tail` (number, optional): Number of lines to return (1-2000, default: 200)

**Returns:** Last N lines of stdout/stderr, total line count, and truncation status

### 2. `get_gpu_metrics`
Get GPU memory usage, utilization, and temperature metrics.

**Arguments:**
- `workload_id` (number, required): The ID of the workload

**Returns:** Time-series metrics data (downsampled to 300 points if needed) with summary including peak memory, average utilization, and data point count. Metrics come from the CrashLens SDK (NVIDIA via `nvidia-ml-py`, memory via `torch.cuda` on CUDA/ROCm) or are simulated for demo jobs.

### 3. `get_failure_report`
Retrieve the AI-generated failure diagnosis report.

**Arguments:**
- `workload_id` (number, required): The ID of the workload

**Returns:** Diagnosis including root cause, evidence, and recommended fixes. If no report exists, suggests running `POST /workloads/{id}/diagnose`.

### 4. `get_checkpoint_state`
Get checkpoint information for a workload.

**Arguments:**
- `workload_id` (number, required): The ID of the workload

**Returns:** Checkpoint paths and availability status

### 5. `get_wasted_gpu_time`
Calculate wasted GPU-seconds for a failed workload.

**Arguments:**
- `workload_id` (number, required): The ID of the workload

**Returns:** Wasted time in seconds, minutes, and hours, plus runtime and status

### 6. `list_failed_workloads`
List all failed workloads with their failure types.

**Arguments:**
- `limit` (number, optional): Maximum number of workloads to return (1-100, default: 10)

**Returns:** Array of failed workloads with metadata

### 7. `get_workload_summary`
Get complete summary of a workload including all available metadata.

**Arguments:**
- `workload_id` (number, required): The ID of the workload

**Returns:** Comprehensive workload information including status, runtime, and flags for available data (logs, metrics, reports, checkpoints)

## Usage

### Running the Server

```bash
export CRASHLENS_URL=https://your-backend.up.railway.app
export CRASHLENS_API_KEY=your-api-key  # optional
npm start
```

The server runs on stdio transport and communicates via JSON-RPC 2.0 protocol.

### Integrating with Claude Desktop

Add to your Claude Desktop MCP settings (`~/Library/Application Support/Claude/claude_desktop_config.json` on macOS):

```json
{
  "mcpServers": {
    "crashlens": {
      "command": "node",
      "args": ["/absolute/path/to/ML-Failure-Doctor/mcp-server/server.js"],
      "env": {
        "CRASHLENS_URL": "https://your-backend.up.railway.app",
        "CRASHLENS_API_KEY": "your-api-key-if-required"
      }
    }
  }
}
```

Replace `/absolute/path/to/ML-Failure-Doctor` with the actual path on your system.

### Testing with MCP Inspector

```bash
export CRASHLENS_URL=https://your-backend.up.railway.app
npx @modelcontextprotocol/inspector node server.js
```

This opens a web UI where you can list tools and test tool calls interactively.

## Testing

Run the test suite:

```bash
npm test
```

All tests use mocked network calls and verify:
- Tool registration and schema validation
- Error handling (404, 401, network failures)
- Edge cases (empty metrics, null values, large log files)
- Authorization header handling

## Architecture

```
┌─────────────────┐
│  MCP Client     │
│ (Claude, etc.)  │
└────────┬────────┘
         │
         │ MCP Protocol (stdio)
         │
┌────────▼────────┐
│  MCP Server     │
│  (Node.js)      │
└────────┬────────┘
         │
         │ HTTPS (REST API)
         │
┌────────▼────────┐
│  CrashLens      │
│  Backend (Go)   │
└─────────────────┘
```

## Error Handling

All tools return structured errors via `isError: true` with specific messages:
- **404 errors**: "Workload 42 not found"
- **401/403 errors**: "Authentication failed. Check CRASHLENS_API_KEY environment variable."
- **Network errors**: "Request timeout after 10 seconds" or "Network error: ..."
- **Validation errors**: Automatically handled by Zod schemas

The server never crashes on individual tool failures and continues serving subsequent requests.

## Implementation Notes

- Uses Node 18+ built-in `fetch` with 10-second timeouts
- All logging goes to stderr; stdout is reserved for MCP protocol
- JSON-encoded fields (gpu_metrics, failure_report, checkpoint_state) are parsed safely
- GPU metrics are downsampled to 300 points if the response contains more
- Workload logs can be tailed to limit response size (default: last 200 lines)

## Docker

From the repository root, start the API and build the optional MCP image:

```bash
docker compose up --build -d
docker compose --profile mcp build mcp-server
```

Configure your MCP client to launch the stdio container (replace the project path):

```json
{
  "mcpServers": {
    "crashlens": {
      "command": "docker",
      "args": [
        "compose", "--project-directory", "/absolute/path/to/ML-Failure-Doctor",
        "run", "--rm", "--no-deps", "-T", "mcp-server"
      ]
    }
  }
}
```

The backend must already be running. Compose supplies `CRASHLENS_URL` using the
internal service hostname and passes `CRASHLENS_API_KEY` from `.env`. Do not use a
TTY: stdout is reserved for the MCP protocol.
