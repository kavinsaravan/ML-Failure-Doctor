# CrashLens MCP Server

Model Context Protocol (MCP) server providing AI debugging tools for CrashLens workload analysis.

## Overview

The CrashLens MCP Server exposes debugging tools that give AI assistants (like Gemma) controlled access to:
- Workload logs and execution traces
- GPU memory and utilization metrics
- Failure diagnosis reports
- Checkpoint states
- Wasted GPU time calculations

## Available Tools

### 1. `get_workload_logs`
Retrieve job execution logs for a specific workload.

**Arguments:**
- `workload_id` (number): The ID of the workload

**Returns:** Job stdout/stderr logs with line count

### 2. `get_gpu_metrics`
Get GPU memory usage, utilization, and temperature metrics.

**Arguments:**
- `workload_id` (number): The ID of the workload

**Returns:** Time-series metrics data with peak memory and average utilization summary

### 3. `get_failure_report`
Retrieve the AI-generated failure diagnosis report.

**Arguments:**
- `workload_id` (number): The ID of the workload

**Returns:** Diagnosis including root cause, evidence, and recommended fixes

### 4. `get_checkpoint_state`
Get checkpoint information for a workload.

**Arguments:**
- `workload_id` (number): The ID of the workload

**Returns:** Checkpoint paths and availability status

### 5. `get_wasted_gpu_time`
Calculate wasted GPU-seconds for a failed workload.

**Arguments:**
- `workload_id` (number): The ID of the workload

**Returns:** Wasted time in seconds, minutes, and hours

### 6. `list_failed_workloads`
List all failed workloads with their failure types.

**Arguments:**
- `limit` (number, optional): Maximum number of workloads to return (default: 10)

**Returns:** Array of failed workloads with metadata

### 7. `get_workload_summary`
Get complete summary of a workload including all available metadata.

**Arguments:**
- `workload_id` (number): The ID of the workload

**Returns:** Comprehensive workload information

## Installation

```bash
cd mcp-server
npm install
```

## Usage

### Running the Server

```bash
npm start
```

The server runs on stdio transport and communicates via JSON-RPC 2.0 protocol.

### Testing

Run the test script to verify all tools work correctly:

```bash
node test-mcp.js
```

### Integrating with Claude Desktop

Add to your Claude Desktop MCP settings (`~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "crashlens": {
      "command": "node",
      "args": ["/path/to/ML-Failure-Doctor/mcp-server/server.js"]
    }
  }
}
```

### Using with Gemma

The MCP tools can be integrated with Gemma via Fireworks AI function-calling:

1. The AI Doctor receives a workload ID
2. It calls MCP tools to gather context (logs, metrics, etc.)
3. It analyzes the data and generates a diagnosis
4. Results are stored back in the database

## Example Tool Call

```javascript
{
  "method": "tools/call",
  "params": {
    "name": "get_workload_summary",
    "arguments": {
      "workload_id": 1
    }
  }
}
```

## Architecture

```
┌─────────────────┐
│  AI Assistant   │
│   (Gemma)       │
└────────┬────────┘
         │
         │ MCP Protocol (stdio)
         │
┌────────▼────────┐
│  MCP Server     │
│  (Node.js)      │
└────────┬────────┘
         │
         │ SQLite
         │
┌────────▼────────┐
│  CrashLens DB   │
│  (crashlens.db) │
└─────────────────┘
```

## Benefits

- **Controlled Access**: AI gets structured access to debugging data
- **Standardized Interface**: MCP protocol works across different AI clients
- **Tool Documentation**: Each tool is self-describing via JSON schemas
- **Extensible**: Easy to add new debugging tools
- **Safe**: Read-only access to database, no modification capabilities

## Resume-Ready Line

*Built an MCP tool layer exposing workload logs, GPU metrics, checkpoint state, and failure reports to a Gemma-powered AI debugging assistant.*
