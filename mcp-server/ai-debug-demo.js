#!/usr/bin/env node

/**
 * AI Debug Demo - Shows how an AI assistant would use MCP tools
 * to diagnose failures in CrashLens
 *
 * This simulates what Claude, Gemma, or another AI would do when
 * asked "Why did my training job fail?"
 */

import { spawn } from 'child_process';

class MCPClient {
  constructor() {
    this.server = spawn('node', ['server.js'], {
      cwd: process.cwd(),
      stdio: ['pipe', 'pipe', 'pipe'],
    });

    this.responseBuffer = '';
    this.pendingRequests = new Map();
    this.requestId = 1;

    this.server.stdout.on('data', (data) => {
      this.handleResponse(data.toString());
    });

    this.server.stderr.on('data', (data) => {
      // Ignore stderr for now
    });
  }

  handleResponse(data) {
    this.responseBuffer += data;
    const lines = this.responseBuffer.split('\n');
    this.responseBuffer = lines.pop() || '';

    lines.forEach(line => {
      if (line.trim()) {
        try {
          const response = JSON.parse(line);
          if (response.id && this.pendingRequests.has(response.id)) {
            const { resolve } = this.pendingRequests.get(response.id);
            this.pendingRequests.delete(response.id);
            resolve(response);
          }
        } catch (e) {
          // Ignore parse errors
        }
      }
    });
  }

  async callTool(toolName, args) {
    const id = this.requestId++;
    const request = {
      jsonrpc: '2.0',
      id,
      method: 'tools/call',
      params: {
        name: toolName,
        arguments: args,
      },
    };

    return new Promise((resolve, reject) => {
      this.pendingRequests.set(id, { resolve, reject });
      this.server.stdin.write(JSON.stringify(request) + '\n');

      // Timeout after 5 seconds
      setTimeout(() => {
        if (this.pendingRequests.has(id)) {
          this.pendingRequests.delete(id);
          reject(new Error('Request timeout'));
        }
      }, 5000);
    });
  }

  close() {
    this.server.kill();
  }
}

// AI Assistant Simulation
async function simulateAIDebugging() {
  console.log('> AI Debug Assistant Starting...\n');
  console.log('P'.repeat(60));
  console.log('Simulating: "Why did my recent ML training job fail?"');
  console.log('P'.repeat(60));
  console.log();

  const client = new MCPClient();

  // Wait for server to initialize
  await new Promise(resolve => setTimeout(resolve, 500));

  try {
    // Step 1: List failed workloads
    console.log('=� Step 1: Finding recent failed workloads...');
    const failedWorkloads = await client.callTool('list_failed_workloads', { limit: 3 });

    if (!failedWorkloads.result || failedWorkloads.result.content.length === 0) {
      console.log(' No failed workloads found!');
      client.close();
      return;
    }

    const response = JSON.parse(failedWorkloads.result.content[0].text);
    const workloads = response.workloads || [];

    if (workloads.length === 0) {
      console.log('No failed workloads found!');
      client.close();
      return;
    }

    console.log(`Found ${workloads.length} failed workload(s):`);
    workloads.forEach(w => {
      console.log(`  - Workload #${w.id}: ${w.name} (${w.failure_type || 'unknown'})`);
    });
    console.log();

    // Focus on the most recent failure
    const targetWorkload = workloads[0];
    console.log(`= Investigating Workload #${targetWorkload.id}: ${targetWorkload.name}`);
    console.log();

    // Step 2: Get workload summary
    console.log('=� Step 2: Getting workload summary...');
    const summary = await client.callTool('get_workload_summary', {
      workload_id: targetWorkload.id
    });
    const summaryData = JSON.parse(summary.result.content[0].text);
    console.log(`  Type: ${summaryData.type}`);
    console.log(`  Status: ${summaryData.status}`);
    console.log(`  Runtime: ${summaryData.runtime_seconds}s`);
    console.log(`  Exit Code: ${summaryData.exit_code}`);
    console.log();

    // Step 3: Analyze GPU metrics
    console.log('=� Step 3: Analyzing GPU metrics...');
    const metrics = await client.callTool('get_gpu_metrics', {
      workload_id: targetWorkload.id
    });
    const metricsData = JSON.parse(metrics.result.content[0].text);

    if (metricsData.metrics && metricsData.metrics.length > 0) {
      console.log(`  Snapshots collected: ${metricsData.metrics.length}`);
      console.log(`  Peak GPU Memory: ${metricsData.summary.peak_memory_percent.toFixed(1)}%`);
      console.log(`  Avg GPU Utilization: ${metricsData.summary.avg_utilization_percent.toFixed(1)}%`);

      // Detect patterns
      if (metricsData.summary.peak_memory_percent > 95) {
        console.log('  �  WARNING: GPU memory usage exceeded 95%!');
      }
    }
    console.log();

    // Step 4: Review logs
    console.log('=� Step 4: Examining error logs...');
    const logs = await client.callTool('get_workload_logs', {
      workload_id: targetWorkload.id
    });
    const logsData = JSON.parse(logs.result.content[0].text);

    if (logsData.logs) {
      const logLines = logsData.logs.split('\n').filter(l => l.trim());
      console.log(`  Total log lines: ${logLines.length}`);

      // Show key error lines
      const errorLines = logLines.filter(l =>
        l.toLowerCase().includes('error') ||
        l.toLowerCase().includes('failed') ||
        l.toLowerCase().includes('exception')
      ).slice(0, 3);

      if (errorLines.length > 0) {
        console.log('  Key error lines:');
        errorLines.forEach(line => {
          console.log(`    � ${line.substring(0, 80)}...`);
        });
      }
    }
    console.log();

    // Step 5: Calculate cost impact
    console.log('=� Step 5: Calculating wasted GPU time...');
    const wastedTime = await client.callTool('get_wasted_gpu_time', {
      workload_id: targetWorkload.id
    });
    const timeData = JSON.parse(wastedTime.result.content[0].text);
    console.log(`  Wasted GPU Time: ${timeData.wasted_seconds}s (${timeData.wasted_minutes} minutes)`);
    console.log();

    // Step 6: Get AI diagnosis
    console.log('>� Step 6: Retrieving AI diagnosis...');
    const diagnosis = await client.callTool('get_failure_report', {
      workload_id: targetWorkload.id
    });
    const diagnosisData = JSON.parse(diagnosis.result.content[0].text);

    console.log('\n' + 'P'.repeat(60));
    console.log('<� DIAGNOSIS REPORT');
    console.log('P'.repeat(60));
    console.log();
    console.log(`Failure Type: ${diagnosisData.failure_type}`);
    console.log(`Confidence: ${(diagnosisData.confidence * 100).toFixed(0)}%`);
    console.log();
    console.log('Root Cause:');
    console.log(`  ${diagnosisData.root_cause}`);
    console.log();

    if (diagnosisData.evidence && diagnosisData.evidence.length > 0) {
      console.log('Evidence:');
      diagnosisData.evidence.slice(0, 3).forEach((e, i) => {
        console.log(`  ${i + 1}. ${e}`);
      });
      console.log();
    }

    console.log('Recommended Repairs:');
    const fixes = diagnosisData.recommended_fix.split('\n').filter(f => f.trim());
    fixes.forEach(fix => {
      console.log(`  ${fix}`);
    });
    console.log();

    console.log(`Safe to Retry: ${diagnosisData.safe_to_retry ? ' Yes' : 'L No'}`);
    console.log();

    console.log('P'.repeat(60));
    console.log('( Debugging session complete!');
    console.log('P'.repeat(60));

  } catch (error) {
    console.error('Error during AI debugging:', error.message);
  } finally {
    client.close();
  }
}

// Run the simulation
simulateAIDebugging();
