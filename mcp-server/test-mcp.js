#!/usr/bin/env node

/**
 * Test script for CrashLens MCP Server
 * Usage: node test-mcp.js
 */

import { spawn } from 'child_process';

// Test the MCP server by sending requests via stdio
const mcpServer = spawn('node', ['server.js'], {
  cwd: process.cwd(),
  stdio: ['pipe', 'pipe', 'inherit'],
});

let responseBuffer = '';

mcpServer.stdout.on('data', (data) => {
  responseBuffer += data.toString();

  // Try to parse complete JSON-RPC messages
  const lines = responseBuffer.split('\n');
  responseBuffer = lines.pop() || ''; // Keep incomplete line in buffer

  lines.forEach(line => {
    if (line.trim()) {
      try {
        const response = JSON.parse(line);
        console.log('\n=== MCP Response ===');
        console.log(JSON.stringify(response, null, 2));
      } catch (e) {
        // Not valid JSON, might be partial
      }
    }
  });
});

mcpServer.on('error', (error) => {
  console.error('Error starting MCP server:', error);
  process.exit(1);
});

// Wait for server to initialize
setTimeout(() => {
  console.log('=== Testing CrashLens MCP Server ===\n');

  // Test 1: List available tools
  console.log('Test 1: Listing available tools...');
  const listToolsRequest = {
    jsonrpc: '2.0',
    id: 1,
    method: 'tools/list',
    params: {},
  };
  mcpServer.stdin.write(JSON.stringify(listToolsRequest) + '\n');

  // Test 2: Get workload summary
  setTimeout(() => {
    console.log('\nTest 2: Getting workload summary for workload 1...');
    const getWorkloadRequest = {
      jsonrpc: '2.0',
      id: 2,
      method: 'tools/call',
      params: {
        name: 'get_workload_summary',
        arguments: { workload_id: 1 },
      },
    };
    mcpServer.stdin.write(JSON.stringify(getWorkloadRequest) + '\n');
  }, 1000);

  // Test 3: Get GPU metrics
  setTimeout(() => {
    console.log('\nTest 3: Getting GPU metrics for workload 1...');
    const getMetricsRequest = {
      jsonrpc: '2.0',
      id: 3,
      method: 'tools/call',
      params: {
        name: 'get_gpu_metrics',
        arguments: { workload_id: 1 },
      },
    };
    mcpServer.stdin.write(JSON.stringify(getMetricsRequest) + '\n');
  }, 2000);

  // Test 4: List failed workloads
  setTimeout(() => {
    console.log('\nTest 4: Listing failed workloads...');
    const listFailedRequest = {
      jsonrpc: '2.0',
      id: 4,
      method: 'tools/call',
      params: {
        name: 'list_failed_workloads',
        arguments: { limit: 5 },
      },
    };
    mcpServer.stdin.write(JSON.stringify(listFailedRequest) + '\n');

    // Exit after all tests
    setTimeout(() => {
      console.log('\n=== All tests completed ===');
      mcpServer.kill();
      process.exit(0);
    }, 1000);
  }, 3000);
}, 500);
