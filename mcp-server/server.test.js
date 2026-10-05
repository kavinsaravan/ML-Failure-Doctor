import { describe, it, afterEach } from 'node:test';
import assert from 'node:assert';
import { Client } from '@modelcontextprotocol/sdk/client/index.js';
import { InMemoryTransport } from '@modelcontextprotocol/sdk/inMemory.js';
import { createServer } from './server.js';

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

// Setup function
async function setupTestServer(env = {}) {
  const url = env.url || 'https://test-backend.com';
  const apiKey = env.apiKey;

  const server = createServer({ url, apiKey });

  const [clientTransport, serverTransport] = InMemoryTransport.createLinkedPair();

  const client = new Client({
    name: 'test-client',
    version: '1.0.0'
  }, {
    capabilities: {}
  });

  await Promise.all([
    server.connect(serverTransport),
    client.connect(clientTransport)
  ]);

  return { client, server };
}

// Cleanup helper
let activeClients = [];
let activeServers = [];

afterEach(async () => {
  for (const client of activeClients) {
    try {
      await client.close();
    } catch {}
  }
  for (const server of activeServers) {
    try {
      await server.close();
    } catch {}
  }
  activeClients = [];
  activeServers = [];
});

describe('MCP Server Tests', () => {
  it('tools/list returns 7 tools with input schemas', async () => {
    setMockResponses([]);
    const { client, server } = await setupTestServer();
    activeClients.push(client);
    activeServers.push(server);

    const tools = await client.listTools();

    assert.strictEqual(tools.tools.length, 7);
    assert.ok(tools.tools.every(t => t.name && t.description && t.inputSchema));
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

    const { client, server } = await setupTestServer();
    activeClients.push(client);
    activeServers.push(server);

    const result = await client.callTool({
      name: 'get_gpu_metrics',
      arguments: { workload_id: 1 }
    });

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

    const { client, server } = await setupTestServer();
    activeClients.push(client);
    activeServers.push(server);

    const result = await client.callTool({
      name: 'get_gpu_metrics',
      arguments: { workload_id: 1 }
    });

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

    const { client, server } = await setupTestServer();
    activeClients.push(client);
    activeServers.push(server);

    const result = await client.callTool({
      name: 'get_workload_summary',
      arguments: { workload_id: 999 }
    });

    assert.ok(result.isError);
    const data = JSON.parse(result.content[0].text);
    assert.ok(data.error.toLowerCase().includes('not found'));
  });

  it('invalid workload_id returns isError true', async () => {
    setMockResponses([]);
    const { client, server } = await setupTestServer();
    activeClients.push(client);
    activeServers.push(server);

    // Test with string instead of number
    const r1 = await client.callTool({
      name: 'get_workload_summary',
      arguments: { workload_id: "abc" }
    });
    assert.strictEqual(r1.isError, true);

    // Test with negative number
    const r2 = await client.callTool({
      name: 'get_workload_summary',
      arguments: { workload_id: -1 }
    });
    assert.strictEqual(r2.isError, true);

    // Test with missing workload_id
    const r3 = await client.callTool({
      name: 'get_workload_summary',
      arguments: {}
    });
    assert.strictEqual(r3.isError, true);
  });

  it('backend 401 returns isError true with auth message mentioning CRASHLENS_API_KEY', async () => {
    setMockResponses([
      {
        status: 401,
        body: { error: 'Unauthorized' }
      }
    ]);

    const { client, server } = await setupTestServer();
    activeClients.push(client);
    activeServers.push(server);

    const result = await client.callTool({
      name: 'get_workload_summary',
      arguments: { workload_id: 1 }
    });

    assert.ok(result.isError);
    const data = JSON.parse(result.content[0].text);
    assert.ok(data.error.includes('CRASHLENS_API_KEY'));
  });

  it('network failure/timeout returns isError true and server keeps running', async () => {
    setMockResponses([
      { error: 'timeout' },
      { status: 200, body: { id: 1, name: 'test', job_logs: 'success' } }
    ]);

    const { client, server } = await setupTestServer();
    activeClients.push(client);
    activeServers.push(server);

    // First call times out
    const result1 = await client.callTool({
      name: 'get_workload_summary',
      arguments: { workload_id: 1 }
    });

    assert.ok(result1.isError);

    // Second call succeeds (server still running)
    const result2 = await client.callTool({
      name: 'get_workload_summary',
      arguments: { workload_id: 1 }
    });

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

    const { client, server } = await setupTestServer();
    activeClients.push(client);
    activeServers.push(server);

    const result = await client.callTool({
      name: 'get_workload_logs',
      arguments: { workload_id: 1 }
    });

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
    const { client: client1, server: server1 } = await setupTestServer({ apiKey: 'test-key-123' });
    activeClients.push(client1);
    activeServers.push(server1);

    await client1.callTool({
      name: 'get_workload_summary',
      arguments: { workload_id: 1 }
    });

    assert.strictEqual(capturedHeaders.Authorization, 'Bearer test-key-123');

    // Test without API key
    capturedHeaders = null;
    const { client: client2, server: server2 } = await setupTestServer({ apiKey: undefined });
    activeClients.push(client2);
    activeServers.push(server2);

    await client2.callTool({
      name: 'get_workload_summary',
      arguments: { workload_id: 1 }
    });

    assert.ok(!capturedHeaders.Authorization);

    globalThis.fetch = originalFetch;
  });

  it('list_failed_workloads with limit 500 returns isError true', async () => {
    setMockResponses([]);
    const { client, server } = await setupTestServer();
    activeClients.push(client);
    activeServers.push(server);

    const r = await client.callTool({
      name: 'list_failed_workloads',
      arguments: { limit: 500 }
    });
    assert.strictEqual(r.isError, true);
  });
});
