#!/usr/bin/env node

import { McpServer } from '@modelcontextprotocol/sdk/server/mcp.js';
import { StdioServerTransport } from '@modelcontextprotocol/sdk/server/stdio.js';
import { z } from 'zod';
import { pathToFileURL } from 'url';

// Error types
class NotFoundError extends Error {
  constructor(message) {
    super(message);
    this.name = 'NotFoundError';
  }
}

class AuthError extends Error {
  constructor(message) {
    super(message);
    this.name = 'AuthError';
  }
}

class NetworkError extends Error {
  constructor(message) {
    super(message);
    this.name = 'NetworkError';
  }
}

class BackendError extends Error {
  constructor(message, status) {
    super(message);
    this.name = 'BackendError';
    this.status = status;
  }
}

// Helper to parse JSON fields safely
function parseJsonField(field) {
  if (!field) return null;
  try {
    return JSON.parse(field);
  } catch {
    return null;
  }
}

// Helper to downsample metrics to 300 points
function downsample(samples, targetCount) {
  if (samples.length <= targetCount) return samples;

  const step = samples.length / targetCount;
  const downsampled = [];

  for (let i = 0; i < targetCount; i++) {
    const index = Math.floor(i * step);
    downsampled.push(samples[index]);
  }

  return downsampled;
}

// Export createServer for testing
export function createServer({ url, apiKey }) {
  const CRASHLENS_URL = url.replace(/\/+$/, '');
  const CRASHLENS_API_KEY = apiKey;

  // API helper
  async function apiGet(path) {
    const requestUrl = `${CRASHLENS_URL}${path}`;
    const headers = {};
    if (CRASHLENS_API_KEY) {
      headers['Authorization'] = `Bearer ${CRASHLENS_API_KEY}`;
    }

    try {
      const response = await fetch(requestUrl, {
        headers,
        signal: AbortSignal.timeout(10000)
      });

      if (response.status === 404) {
        const workloadId = path.match(/\/workloads\/(\d+)/)?.[1];
        throw new NotFoundError(workloadId ? `Workload ${workloadId} not found` : 'Resource not found');
      }

      if (response.status === 401 || response.status === 403) {
        throw new AuthError('Authentication failed. Check CRASHLENS_API_KEY environment variable.');
      }

      if (!response.ok) {
        throw new BackendError(`Backend error: ${response.status} ${response.statusText}`, response.status);
      }

      return await response.json();
    } catch (error) {
      if (error instanceof NotFoundError || error instanceof AuthError || error instanceof BackendError) {
        throw error;
      }
      if (error.name === 'AbortError') {
        throw new NetworkError('Request timeout after 10 seconds');
      }
      throw new NetworkError(`Network error: ${error.message}`);
    }
  }

  // Create MCP server
  const server = new McpServer({
    name: 'crashlens-mcp-server',
    version: '1.0.0',
  });

  // Tool: get_workload_logs
  server.registerTool(
    'get_workload_logs',
    {
      description: 'Retrieve job execution logs for a specific workload. Returns the last N lines of stdout/stderr from the ML job.',
      inputSchema: z.object({
        workload_id: z.number().int().positive(),
        tail: z.number().int().min(1).max(2000).optional().default(200)
      })
    },
    async ({ workload_id, tail }) => {
      try {
        const workload = await apiGet(`/workloads/${workload_id}`);

        if (!workload.job_logs) {
          return {
            content: [{
              type: 'text',
              text: JSON.stringify({
                workload_id,
                logs: null,
                total_lines: 0,
                truncated: false,
                message: 'No logs available'
              })
            }]
          };
        }

        const lines = workload.job_logs.split('\n');
        const total_lines = lines.length;
        const truncated = total_lines > tail;
        const logs = truncated ? lines.slice(-tail).join('\n') : workload.job_logs;

        return {
          content: [{
            type: 'text',
            text: JSON.stringify({
              workload_id,
              logs,
              total_lines,
              truncated
            })
          }]
        };
      } catch (error) {
        return {
          content: [{
            type: 'text',
            text: JSON.stringify({ error: error.message })
          }],
          isError: true
        };
      }
    }
  );

  // Tool: get_gpu_metrics
  server.registerTool(
    'get_gpu_metrics',
    {
      description: 'Get GPU memory usage, utilization, and temperature metrics collected during workload execution. Metrics come from the CrashLens SDK (NVIDIA via nvidia-ml-py, memory via torch.cuda on CUDA/ROCm) or are simulated for demo jobs.',
      inputSchema: z.object({
        workload_id: z.number().int().positive()
      })
    },
    async ({ workload_id }) => {
      try {
        const workload = await apiGet(`/workloads/${workload_id}`);

        const metrics = parseJsonField(workload.gpu_metrics);

        if (!metrics || metrics.length === 0) {
          return {
            content: [{
              type: 'text',
              text: JSON.stringify({
                workload_id,
                metrics: [],
                summary: null,
                message: 'No GPU metrics recorded'
              })
            }]
          };
        }

        // Compute summary over all samples
        const nonNullUtil = metrics.filter(m => m.gpu_utilization_percent != null);
        const avg_utilization_percent = nonNullUtil.length > 0
          ? nonNullUtil.reduce((sum, m) => sum + m.gpu_utilization_percent, 0) / nonNullUtil.length
          : null;

        const peakMemory = metrics.reduce((max, m) =>
          m.gpu_memory_percent > (max?.gpu_memory_percent ?? 0) ? m : max
        );

        const peakAllocated = metrics.find(m => m.gpu_memory_peak_mb != null);

        const summary = {
          peak_memory_mb: peakMemory.gpu_memory_used_mb,
          peak_memory_percent: peakMemory.gpu_memory_percent,
          avg_utilization_percent,
          data_points: metrics.length,
          ...(peakAllocated && { peak_allocated_mb: Math.max(...metrics.map(m => m.gpu_memory_peak_mb || 0)) })
        };

        // Downsample for response
        const downsampledMetrics = downsample(metrics, 300);

        return {
          content: [{
            type: 'text',
            text: JSON.stringify({
              workload_id,
              metrics: downsampledMetrics,
              summary
            })
          }]
        };
      } catch (error) {
        return {
          content: [{
            type: 'text',
            text: JSON.stringify({ error: error.message })
          }],
          isError: true
        };
      }
    }
  );

  // Tool: get_failure_report
  server.registerTool(
    'get_failure_report',
    {
      description: 'Retrieve the AI-generated failure diagnosis report including root cause, evidence, and recommended fixes.',
      inputSchema: z.object({
        workload_id: z.number().int().positive()
      })
    },
    async ({ workload_id }) => {
      try {
        const workload = await apiGet(`/workloads/${workload_id}`);

        const report = parseJsonField(workload.failure_report);

        if (!report) {
          return {
            content: [{
              type: 'text',
              text: JSON.stringify({
                workload_id,
                report: null,
                message: `No failure report available. Run diagnosis with: POST /workloads/${workload_id}/diagnose`
              })
            }]
          };
        }

        return {
          content: [{
            type: 'text',
            text: JSON.stringify({
              workload_id,
              report
            })
          }]
        };
      } catch (error) {
        return {
          content: [{
            type: 'text',
            text: JSON.stringify({ error: error.message })
          }],
          isError: true
        };
      }
    }
  );

  // Tool: get_checkpoint_state
  server.registerTool(
    'get_checkpoint_state',
    {
      description: 'Get checkpoint information for a workload, including paths and whether checkpoint was found.',
      inputSchema: z.object({
        workload_id: z.number().int().positive()
      })
    },
    async ({ workload_id }) => {
      try {
        const workload = await apiGet(`/workloads/${workload_id}`);

        const checkpoint_state = parseJsonField(workload.checkpoint_state);

        if (!checkpoint_state) {
          return {
            content: [{
              type: 'text',
              text: JSON.stringify({
                workload_id,
                checkpoint_state: null,
                message: 'No checkpoint state information'
              })
            }]
          };
        }

        return {
          content: [{
            type: 'text',
            text: JSON.stringify({
              workload_id,
              checkpoint_state
            })
          }]
        };
      } catch (error) {
        return {
          content: [{
            type: 'text',
            text: JSON.stringify({ error: error.message })
          }],
          isError: true
        };
      }
    }
  );

  // Tool: get_wasted_gpu_time
  server.registerTool(
    'get_wasted_gpu_time',
    {
      description: 'Calculate wasted GPU-seconds for a failed workload. Helps quantify cost impact of failures.',
      inputSchema: z.object({
        workload_id: z.number().int().positive()
      })
    },
    async ({ workload_id }) => {
      try {
        const workload = await apiGet(`/workloads/${workload_id}`);

        const wasted_seconds = workload.wasted_gpu_seconds || 0;
        const wasted_minutes = wasted_seconds / 60;
        const wasted_hours = wasted_seconds / 3600;

        return {
          content: [{
            type: 'text',
            text: JSON.stringify({
              workload_id,
              status: workload.status,
              seconds: wasted_seconds,
              minutes: wasted_minutes,
              hours: wasted_hours,
              runtime: workload.runtime_seconds
            })
          }]
        };
      } catch (error) {
        return {
          content: [{
            type: 'text',
            text: JSON.stringify({ error: error.message })
          }],
          isError: true
        };
      }
    }
  );

  // Tool: list_failed_workloads
  server.registerTool(
    'list_failed_workloads',
    {
      description: 'List all failed workloads with their failure types. Useful for identifying patterns.',
      inputSchema: z.object({
        limit: z.number().int().min(1).max(100).optional().default(10)
      })
    },
    async ({ limit }) => {
      try {
        const workloads = await apiGet('/workloads');

        const failed = workloads
          .filter(w => w.status === 'failed')
          .sort((a, b) => new Date(b.created_at) - new Date(a.created_at))
          .slice(0, limit)
          .map(w => ({
            id: w.id,
            name: w.name,
            type: w.type,
            failure_type: w.failure_type,
            runtime_seconds: w.runtime_seconds,
            wasted_gpu_seconds: w.wasted_gpu_seconds,
            created_at: w.created_at
          }));

        return {
          content: [{
            type: 'text',
            text: JSON.stringify({
              total: failed.length,
              workloads: failed
            })
          }]
        };
      } catch (error) {
        return {
          content: [{
            type: 'text',
            text: JSON.stringify({ error: error.message })
          }],
          isError: true
        };
      }
    }
  );

  // Tool: get_workload_summary
  server.registerTool(
    'get_workload_summary',
    {
      description: 'Get complete summary of a workload including status, runtime, failure type, and all available metadata.',
      inputSchema: z.object({
        workload_id: z.number().int().positive()
      })
    },
    async ({ workload_id }) => {
      try {
        const workload = await apiGet(`/workloads/${workload_id}`);

        return {
          content: [{
            type: 'text',
            text: JSON.stringify({
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
              has_metrics: !!parseJsonField(workload.gpu_metrics),
              has_failure_report: !!parseJsonField(workload.failure_report),
              has_checkpoint_state: !!parseJsonField(workload.checkpoint_state)
            })
          }]
        };
      } catch (error) {
        return {
          content: [{
            type: 'text',
            text: JSON.stringify({ error: error.message })
          }],
          isError: true
        };
      }
    }
  );

  return server;
}

// Entry point guard - only run main() when executed directly
if (import.meta.url === pathToFileURL(process.argv[1]).href) {
  const CRASHLENS_URL = process.env.CRASHLENS_URL;
  const CRASHLENS_API_KEY = process.env.CRASHLENS_API_KEY;

  if (!CRASHLENS_URL) {
    console.error('Error: CRASHLENS_URL environment variable is required');
    console.error('Example: export CRASHLENS_URL=https://your-backend.up.railway.app');
    process.exit(1);
  }

  async function main() {
    const server = createServer({ url: CRASHLENS_URL, apiKey: CRASHLENS_API_KEY });
    const transport = new StdioServerTransport();
    await server.connect(transport);
    console.error('CrashLens MCP Server running on stdio');
    console.error(`Connected to: ${CRASHLENS_URL}`);
    console.error(`Authentication: ${CRASHLENS_API_KEY ? 'enabled' : 'disabled'}`);
  }

  main().catch((error) => {
    console.error('Server error:', error);
    process.exit(1);
  });
}
