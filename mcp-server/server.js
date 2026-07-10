#!/usr/bin/env node

import { Server } from '@modelcontextprotocol/sdk/server/index.js';
import { StdioServerTransport } from '@modelcontextprotocol/sdk/server/stdio.js';
import {
  CallToolRequestSchema,
  ListToolsRequestSchema,
} from '@modelcontextprotocol/sdk/types.js';
import sqlite3 from 'sqlite3';
import { promisify } from 'util';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// Database connection
const dbPath = path.join(__dirname, '../backend/crashlens.db');
const db = new sqlite3.Database(dbPath);
const dbGet = promisify(db.get.bind(db));
const dbAll = promisify(db.all.bind(db));

// MCP Server for CrashLens AI Doctor
const server = new Server(
  {
    name: 'crashlens-mcp-server',
    version: '1.0.0',
  },
  {
    capabilities: {
      tools: {},
    },
  }
);

// Define tools available to AI Doctor
const TOOLS = [
  {
    name: 'get_workload_logs',
    description: 'Retrieve job execution logs for a specific workload. Returns stdout/stderr from the ML job or agent run.',
    inputSchema: {
      type: 'object',
      properties: {
        workload_id: {
          type: 'number',
          description: 'The ID of the workload to retrieve logs for',
        },
      },
      required: ['workload_id'],
    },
  },
  {
    name: 'get_gpu_metrics',
    description: 'Get GPU memory usage, utilization, and temperature metrics collected during workload execution. Returns time-series data from rocm-smi.',
    inputSchema: {
      type: 'object',
      properties: {
        workload_id: {
          type: 'number',
          description: 'The ID of the workload to retrieve GPU metrics for',
        },
      },
      required: ['workload_id'],
    },
  },
  {
    name: 'get_failure_report',
    description: 'Retrieve the AI-generated failure diagnosis report including root cause, evidence, and recommended fixes.',
    inputSchema: {
      type: 'object',
      properties: {
        workload_id: {
          type: 'number',
          description: 'The ID of the workload to retrieve failure report for',
        },
      },
      required: ['workload_id'],
    },
  },
  {
    name: 'get_checkpoint_state',
    description: 'Get checkpoint information for a workload, including paths and whether checkpoint was found.',
    inputSchema: {
      type: 'object',
      properties: {
        workload_id: {
          type: 'number',
          description: 'The ID of the workload to retrieve checkpoint state for',
        },
      },
      required: ['workload_id'],
    },
  },
  {
    name: 'get_wasted_gpu_time',
    description: 'Calculate wasted GPU-seconds for a failed workload. Helps quantify cost impact of failures.',
    inputSchema: {
      type: 'object',
      properties: {
        workload_id: {
          type: 'number',
          description: 'The ID of the workload to calculate wasted GPU time for',
        },
      },
      required: ['workload_id'],
    },
  },
  {
    name: 'list_failed_workloads',
    description: 'List all failed workloads with their failure types. Useful for identifying patterns.',
    inputSchema: {
      type: 'object',
      properties: {
        limit: {
          type: 'number',
          description: 'Maximum number of workloads to return (default: 10)',
          default: 10,
        },
      },
    },
  },
  {
    name: 'get_workload_summary',
    description: 'Get complete summary of a workload including status, runtime, failure type, and all available metadata.',
    inputSchema: {
      type: 'object',
      properties: {
        workload_id: {
          type: 'number',
          description: 'The ID of the workload to retrieve summary for',
        },
      },
      required: ['workload_id'],
    },
  },
];

// Tool handlers
async function getWorkloadLogs(workloadId) {
  const workload = await dbGet('SELECT job_logs FROM workloads WHERE id = ?', [workloadId]);

  if (!workload) {
    return { error: `Workload ${workloadId} not found` };
  }

  if (!workload.job_logs) {
    return { workload_id: workloadId, logs: null, message: 'No logs available' };
  }

  return {
    workload_id: workloadId,
    logs: workload.job_logs,
    line_count: workload.job_logs.split('\n').length,
  };
}

async function getGpuMetrics(workloadId) {
  const workload = await dbGet('SELECT gpu_metrics FROM workloads WHERE id = ?', [workloadId]);

  if (!workload) {
    return { error: `Workload ${workloadId} not found` };
  }

  if (!workload.gpu_metrics) {
    return { workload_id: workloadId, metrics: null, message: 'No GPU metrics available' };
  }

  const metrics = JSON.parse(workload.gpu_metrics);
  const peakMemory = metrics.reduce((max, m) =>
    m.gpu_memory_percent > max.gpu_memory_percent ? m : max,
    metrics[0]
  );

  return {
    workload_id: workloadId,
    metrics: metrics,
    summary: {
      peak_memory_mb: peakMemory.gpu_memory_used_mb,
      peak_memory_percent: peakMemory.gpu_memory_percent,
      avg_utilization: metrics.reduce((sum, m) => sum + m.gpu_utilization_percent, 0) / metrics.length,
      data_points: metrics.length,
    },
  };
}

async function getFailureReport(workloadId) {
  const workload = await dbGet('SELECT failure_report FROM workloads WHERE id = ?', [workloadId]);

  if (!workload) {
    return { error: `Workload ${workloadId} not found` };
  }

  if (!workload.failure_report) {
    return { workload_id: workloadId, report: null, message: 'No failure report available. Run diagnosis first.' };
  }

  return {
    workload_id: workloadId,
    report: JSON.parse(workload.failure_report),
  };
}

async function getCheckpointState(workloadId) {
  const workload = await dbGet('SELECT checkpoint_state FROM workloads WHERE id = ?', [workloadId]);

  if (!workload) {
    return { error: `Workload ${workloadId} not found` };
  }

  if (!workload.checkpoint_state) {
    return { workload_id: workloadId, checkpoint_state: null, message: 'No checkpoint state information' };
  }

  return {
    workload_id: workloadId,
    checkpoint_state: JSON.parse(workload.checkpoint_state),
  };
}

async function getWastedGpuTime(workloadId) {
  const workload = await dbGet(
    'SELECT wasted_gpu_seconds, runtime_seconds, status, failure_type FROM workloads WHERE id = ?',
    [workloadId]
  );

  if (!workload) {
    return { error: `Workload ${workloadId} not found` };
  }

  const wastedSeconds = workload.wasted_gpu_seconds || 0;
  const wastedMinutes = (wastedSeconds / 60).toFixed(2);
  const wastedHours = (wastedSeconds / 3600).toFixed(2);

  return {
    workload_id: workloadId,
    status: workload.status,
    failure_type: workload.failure_type,
    wasted_gpu_seconds: wastedSeconds,
    wasted_gpu_minutes: parseFloat(wastedMinutes),
    wasted_gpu_hours: parseFloat(wastedHours),
    runtime_seconds: workload.runtime_seconds,
  };
}

async function listFailedWorkloads(limit = 10) {
  const workloads = await dbAll(
    'SELECT id, name, type, status, failure_type, runtime_seconds, wasted_gpu_seconds, created_at FROM workloads WHERE status = ? ORDER BY created_at DESC LIMIT ?',
    ['failed', limit]
  );

  return {
    total: workloads.length,
    workloads: workloads.map(w => ({
      id: w.id,
      name: w.name,
      type: w.type,
      failure_type: w.failure_type,
      runtime_seconds: w.runtime_seconds,
      wasted_gpu_seconds: w.wasted_gpu_seconds,
      created_at: w.created_at,
    })),
  };
}

async function getWorkloadSummary(workloadId) {
  const workload = await dbGet('SELECT * FROM workloads WHERE id = ?', [workloadId]);

  if (!workload) {
    return { error: `Workload ${workloadId} not found` };
  }

  return {
    id: workload.id,
    name: workload.name,
    type: workload.type,
    status: workload.status,
    failure_type: workload.failure_type,
    created_at: workload.created_at,
    started_at: workload.started_at,
    finished_at: workload.finished_at,
    runtime_seconds: workload.runtime_seconds,
    exit_code: workload.exit_code,
    wasted_gpu_seconds: workload.wasted_gpu_seconds,
    has_logs: !!workload.job_logs,
    has_metrics: !!workload.gpu_metrics,
    has_failure_report: !!workload.failure_report,
    has_checkpoint_state: !!workload.checkpoint_state,
  };
}

// Register tool handlers
server.setRequestHandler(ListToolsRequestSchema, async () => {
  return { tools: TOOLS };
});

server.setRequestHandler(CallToolRequestSchema, async (request) => {
  const { name, arguments: args } = request.params;

  try {
    switch (name) {
      case 'get_workload_logs':
        return {
          content: [
            {
              type: 'text',
              text: JSON.stringify(await getWorkloadLogs(args.workload_id), null, 2),
            },
          ],
        };

      case 'get_gpu_metrics':
        return {
          content: [
            {
              type: 'text',
              text: JSON.stringify(await getGpuMetrics(args.workload_id), null, 2),
            },
          ],
        };

      case 'get_failure_report':
        return {
          content: [
            {
              type: 'text',
              text: JSON.stringify(await getFailureReport(args.workload_id), null, 2),
            },
          ],
        };

      case 'get_checkpoint_state':
        return {
          content: [
            {
              type: 'text',
              text: JSON.stringify(await getCheckpointState(args.workload_id), null, 2),
            },
          ],
        };

      case 'get_wasted_gpu_time':
        return {
          content: [
            {
              type: 'text',
              text: JSON.stringify(await getWastedGpuTime(args.workload_id), null, 2),
            },
          ],
        };

      case 'list_failed_workloads':
        return {
          content: [
            {
              type: 'text',
              text: JSON.stringify(await listFailedWorkloads(args.limit || 10), null, 2),
            },
          ],
        };

      case 'get_workload_summary':
        return {
          content: [
            {
              type: 'text',
              text: JSON.stringify(await getWorkloadSummary(args.workload_id), null, 2),
            },
          ],
        };

      default:
        return {
          content: [
            {
              type: 'text',
              text: JSON.stringify({ error: `Unknown tool: ${name}` }),
            },
          ],
          isError: true,
        };
    }
  } catch (error) {
    return {
      content: [
        {
          type: 'text',
          text: JSON.stringify({ error: error.message }),
        },
      ],
      isError: true,
    };
  }
});

// Start server
async function main() {
  const transport = new StdioServerTransport();
  await server.connect(transport);
  console.error('CrashLens MCP Server running on stdio');
}

main().catch((error) => {
  console.error('Server error:', error);
  process.exit(1);
});
