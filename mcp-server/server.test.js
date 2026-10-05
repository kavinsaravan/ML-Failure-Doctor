import { describe, it, before } from 'node:test';
import assert from 'node:assert';
import { Client } from '@modelcontextprotocol/sdk/client/index.js';
import { InMemoryTransport } from '@modelcontextprotocol/sdk/inMemory.js';
import { McpServer } from '@modelcontextprotocol/sdk/server/mcp.js';
import { z } from 'zod';

// Mock fetch globally
let mockResponses = [];
let mockResponseIndex = 0;

function setMockResponses(responses) {
  mockResponses = responses;
  mockResponseIndex = 0;
}

globalThis.fetch = async (url, options) => {
  const response = mockResponses[mockResponseIndex++] || mockResponses[mockResponses.length - 1];

  if (!response) {
    throw new Error('No mock response configured');
  }

  if (response.error) {
    if (response.error === 'timeout') {
      const error = new Error('The operation was aborted');
      error.name = 'AbortError';
      throw error;
    }
    throw new Error(response.error);
  }

  return {
    ok: response.status >= 200 && response.status < 300,
    status: response.status,
    statusText: response.statusText || 'OK',
    json: async () => response.body
  };
};

// Helper functions (copied from server.js logic)
function parseJsonField(field) {
  if (!field) return null;
  try {
    return JSON.parse(field);
  } catch {
    return null;
  }
}

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

// Setup function
async function setupTestServer(env = {}) {
  const CRASHLENS_URL = (env.CRASHLENS_URL || 'https://test-backend.com').replace(/\/+$/, '');
  const CRASHLENS_API_KEY = env.CRASHLENS_API_KEY;

  async function apiGet(path) {
    const url = `${CRASHLENS_URL}${path}`;
    const headers = {};
    if (CRASHLENS_API_KEY) {
      headers['Authorization'] = `Bearer ${CRASHLENS_API_KEY}`;
    }

    const response = await fetch(url, {
      headers,
      signal: AbortSignal.timeout(10000)
    });

    if (response.status === 404) {
      throw new Error('Resource not found: ' + path);
    }
    if (response.status === 401 || response.status === 403) {
      throw new Error('Authentication failed. Check CRASHLENS_API_KEY environment variable.');
    }
    if (!response.ok) {
      throw new Error(`Backend error: ${response.status} ${response.statusText}`);
    }

    return await response.json();
  }

  const server = new McpServer({
    name: 'test-server',
    version: '1.0.0'
  });

  // Register tools
  server.registerTool(
    'get_workload_summary',
    {
      description: 'Get complete summary of a workload',
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
              has_logs: !!workload.job_logs
            })
          }]
        };
      } catch (error) {
        return {
          content: [{ type: 'text', text: JSON.stringify({ error: error.message }) }],
          isError: true
        };
      }
    }
  );

  server.registerTool(
    'get_gpu_metrics',
    {
      description: 'Get GPU metrics',
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

        const nonNullUtil = metrics.filter(m => m.gpu_utilization_percent != null);
        const avg_utilization_percent = nonNullUtil.length > 0
          ? nonNullUtil.reduce((sum, m) => sum + m.gpu_utilization_percent, 0) / nonNullUtil.length
          : null;

        const peakMemory = metrics.reduce((max, m) =>
          m.gpu_memory_percent > (max?.gpu_memory_percent ?? 0) ? m : max
        );

        const summary = {
          peak_memory_mb: peakMemory.gpu_memory_used_mb,
          peak_memory_percent: peakMemory.gpu_memory_percent,
          avg_utilization_percent,
          data_points: metrics.length
        };

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
          content: [{ type: 'text', text: JSON.stringify({ error: error.message }) }],
          isError: true
        };
      }
    }
  );

  server.registerTool(
    'get_workload_logs',
    {
      description: 'Get workload logs',
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
                truncated: false
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
          content: [{ type: 'text', text: JSON.stringify({ error: error.message }) }],
          isError: true
        };
      }
    }
  );

  server.registerTool(
    'list_failed_workloads',
    {
      description: 'List failed workloads',
      inputSchema: z.object({
        limit: z.number().int().min(1).max(100).optional().default(10)
      })
    },
    async ({ limit }) => {
      try {
        const workloads = await apiGet('/workloads');
        const failed = workloads
          .filter(w => w.status === 'failed')
          .slice(0, limit);
        return {
          content: [{
            type: 'text',
            text: JSON.stringify({ total: failed.length, workloads: failed })
          }]
        };
      } catch (error) {
        return {
          content: [{ type: 'text', text: JSON.stringify({ error: error.message }) }],
          isError: true
        };
      }
    }
  );

  const [clientTransport, serverTransport] = InMemoryTransport.createLinkedPair();

  const client = new Client({
    name: 'test-client',
    version: '1.0.0'
  }, {
    capabilities: {}
  });

  await client.connect(clientTransport);
  await server.connect(serverTransport);

  return { client, server };
}

describe('MCP Server Tests', () => {
  it('tools/list returns tools with input schemas', async () => {
    setMockResponses([]);
    const { client } = await setupTestServer();

    const result = await client.request({ method: 'tools/list' }, {});

    assert.ok(result.tools.length >= 4);
    assert.ok(result.tools.every(t => t.name && t.description && t.inputSchema));
  });

  it('get_gpu_metrics with empty metrics returns no error, summary null', async () => {
    setMockResponses([
      {
        status: 200,
        body: {
          id: 1,
          name: 'test',
          gpu_metrics: '[]'
        }
      }
    ]);

    const { client } = await setupTestServer();

    const result = await client.request({
      method: 'tools/call',
      params: {
        name: 'get_gpu_metrics',
        arguments: { workload_id: 1 }
      }
    }, {});

    const data = JSON.parse(result.content[0].text);
    assert.strictEqual(data.metrics.length, 0);
    assert.strictEqual(data.summary, null);
    assert.strictEqual(data.message, 'No GPU metrics recorded');
    assert.ok(!result.isError);
  });

  it('get_gpu_metrics where all utilization values are null returns avg_utilization_percent null', async () => {
    const metrics = [
      { gpu_memory_percent: 50, gpu_memory_used_mb: 1000, gpu_utilization_percent: null },
      { gpu_memory_percent: 60, gpu_memory_used_mb: 1200, gpu_utilization_percent: null }
    ];

    setMockResponses([
      {
        status: 200,
        body: {
          id: 1,
          name: 'test',
          gpu_metrics: JSON.stringify(metrics)
        }
      }
    ]);

    const { client } = await setupTestServer();

    const result = await client.request({
      method: 'tools/call',
      params: {
        name: 'get_gpu_metrics',
        arguments: { workload_id: 1 }
      }
    }, {});

    const data = JSON.parse(result.content[0].text);
    assert.strictEqual(data.summary.avg_utilization_percent, null);
  });

  it('any tool with nonexistent ID returns isError true with "not found" message', async () => {
    setMockResponses([
      {
        status: 404,
        body: { error: 'Not found' }
      }
    ]);

    const { client } = await setupTestServer();

    const result = await client.request({
      method: 'tools/call',
      params: {
        name: 'get_workload_summary',
        arguments: { workload_id: 999 }
      }
    }, {});

    assert.ok(result.isError);
    const data = JSON.parse(result.content[0].text);
    assert.ok(data.error.toLowerCase().includes('not found'));
  });

  it('missing or invalid workload_id returns rejected with isError true', async () => {
    setMockResponses([]);
    const { client } = await setupTestServer();

    // Test with string instead of number
    await assert.rejects(async () => {
      await client.request({
        method: 'tools/call',
        params: {
          name: 'get_workload_summary',
          arguments: { workload_id: "invalid" }
        }
      }, {});
    });

    // Test with negative number
    await assert.rejects(async () => {
      await client.request({
        method: 'tools/call',
        params: {
          name: 'get_workload_summary',
          arguments: { workload_id: -1 }
        }
      }, {});
    });
  });

  it('backend 401 returns isError true with auth message mentioning CRASHLENS_API_KEY', async () => {
    setMockResponses([
      {
        status: 401,
        body: { error: 'Unauthorized' }
      }
    ]);

    const { client } = await setupTestServer();

    const result = await client.request({
      method: 'tools/call',
      params: {
        name: 'get_workload_summary',
        arguments: { workload_id: 1 }
      }
    }, {});

    assert.ok(result.isError);
    const data = JSON.parse(result.content[0].text);
    assert.ok(data.error.includes('CRASHLENS_API_KEY'));
  });

  it('network failure/timeout returns isError true and server keeps running', async () => {
    setMockResponses([
      { error: 'timeout' },
      { status: 200, body: { id: 1, name: 'test', job_logs: 'success' } }
    ]);

    const { client } = await setupTestServer();

    // First call times out
    const result1 = await client.request({
      method: 'tools/call',
      params: {
        name: 'get_workload_summary',
        arguments: { workload_id: 1 }
      }
    }, {});

    assert.ok(result1.isError);

    // Second call succeeds (server still running)
    const result2 = await client.request({
      method: 'tools/call',
      params: {
        name: 'get_workload_summary',
        arguments: { workload_id: 1 }
      }
    }, {});

    assert.ok(!result2.isError);
  });

  it('get_workload_logs on 5000 lines with default tail returns 200 lines, truncated true', async () => {
    const lines = Array(5000).fill('log line').join('\n');

    setMockResponses([
      {
        status: 200,
        body: {
          id: 1,
          name: 'test',
          job_logs: lines
        }
      }
    ]);

    const { client } = await setupTestServer();

    const result = await client.request({
      method: 'tools/call',
      params: {
        name: 'get_workload_logs',
        arguments: { workload_id: 1 }
      }
    }, {});

    const data = JSON.parse(result.content[0].text);
    assert.strictEqual(data.total_lines, 5000);
    assert.strictEqual(data.truncated, true);
    assert.strictEqual(data.logs.split('\n').length, 200);
  });

  it('CRASHLENS_API_KEY set sends Authorization header; unset sends no header', async () => {
    let capturedHeaders = null;

    const originalFetch = globalThis.fetch;
    globalThis.fetch = async (url, options) => {
      capturedHeaders = options.headers;
      return {
        ok: true,
        status: 200,
        json: async () => ({ id: 1, name: 'test', job_logs: 'test' })
      };
    };

    // Test with API key
    const { client: client1 } = await setupTestServer({ CRASHLENS_API_KEY: 'test-key-123' });

    await client1.request({
      method: 'tools/call',
      params: {
        name: 'get_workload_summary',
        arguments: { workload_id: 1 }
      }
    }, {});

    assert.strictEqual(capturedHeaders.Authorization, 'Bearer test-key-123');

    // Test without API key
    capturedHeaders = null;
    const { client: client2 } = await setupTestServer({ CRASHLENS_API_KEY: undefined });

    await client2.request({
      method: 'tools/call',
      params: {
        name: 'get_workload_summary',
        arguments: { workload_id: 1 }
      }
    }, {});

    assert.ok(!capturedHeaders.Authorization);

    globalThis.fetch = originalFetch;
  });

  it('list_failed_workloads with limit 500 is rejected', async () => {
    setMockResponses([]);
    const { client } = await setupTestServer();

    await assert.rejects(async () => {
      await client.request({
        method: 'tools/call',
        params: {
          name: 'list_failed_workloads',
          arguments: { limit: 500 }
        }
      }, {});
    });
  });
});
