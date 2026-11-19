import test from 'node:test';
import assert from 'node:assert/strict';
import { getMemoryPeaks, getWorkloadStatus } from '../lib/api.ts';

test('stale contact changes monitoring status without inventing a failed lifecycle', () => {
  assert.equal(getWorkloadStatus({status: 'running', heartbeat_stale: true}), 'stale');
  assert.equal(getWorkloadStatus({status: 'running', heartbeat_stale: false}), 'running');
  assert.equal(getWorkloadStatus({status: 'succeeded', heartbeat_stale: true}), 'succeeded');
  assert.equal(getWorkloadStatus({status: 'failed'}), 'failed');
});

test('sampled device usage and framework high-water marks remain distinct', () => {
  const samples = [
    {gpu_memory_used_mb: 2000, gpu_memory_total_mb: 8000, gpu_memory_percent: 25, gpu_memory_peak_mb: 6000},
    {gpu_memory_used_mb: 4000, gpu_memory_total_mb: 8000, gpu_memory_percent: 50, gpu_memory_peak_mb: 6500},
    {gpu_memory_used_mb: 1000, gpu_memory_total_mb: 8000, gpu_memory_percent: 12.5, gpu_memory_peak_mb: 7000},
  ];
  const result = getMemoryPeaks(samples);
  assert.equal(result.sampledDevice.gpu_memory_used_mb, 4000);
  assert.equal(result.frameworkAllocatedMB, 7000);
});

test('unavailable framework peaks are not displayed as zero usage', () => {
  assert.deepEqual(getMemoryPeaks([]), {sampledDevice: null, frameworkAllocatedMB: null});
  const sample = {gpu_memory_used_mb: 2, gpu_memory_total_mb: 8, gpu_memory_percent: 25};
  for (const value of [undefined, null, NaN, -1]) {
    assert.equal(getMemoryPeaks([{...sample, gpu_memory_peak_mb: value}]).frameworkAllocatedMB, null);
  }
  assert.equal(getMemoryPeaks([{...sample, gpu_memory_peak_mb: 0}]).frameworkAllocatedMB, 0);
});
