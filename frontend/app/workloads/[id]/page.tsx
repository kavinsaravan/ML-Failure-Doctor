'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import { useParams } from 'next/navigation';
import Link from 'next/link';
import { api, Workload, DiagnosisReport, MetricsSnapshot, getMemoryPeaks, getWorkloadStatus } from '@/lib/api';
import ConnectionNotice from '@/components/connection-notice';
import {
  ArrowLeft,
  AlertTriangle,
  CheckCircle,
  Clock,
  Terminal,
  Activity,
  Zap,
  Database,
  Cpu,
  Loader2,
} from 'lucide-react';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend } from 'recharts';

export default function WorkloadDetail() {
  const params = useParams();
  const [workload, setWorkload] = useState<Workload | null>(null);
  const [diagnosis, setDiagnosis] = useState<DiagnosisReport | null>(null);
  const [metrics, setMetrics] = useState<MetricsSnapshot[]>([]);
  const [loading, setLoading] = useState(true);
  const [diagnosing, setDiagnosing] = useState(false);

  const [error, setError] = useState<string | null>(null);
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);
  const [retrying, setRetrying] = useState(false);
  const inFlight = useRef(false);

  const loadWorkload = useCallback(async () => {
    if (inFlight.current) return;
    inFlight.current = true;
    setRetrying(true);
    try {
      const data = await api.getWorkload(params.id as string);
      const report = data.failure_report ? JSON.parse(data.failure_report) : null;
      const samples = data.gpu_metrics ? JSON.parse(data.gpu_metrics) : [];
      if (!Array.isArray(samples)) throw new Error('Invalid GPU metrics received.');
      setWorkload(data);
      setDiagnosis(report);
      setMetrics(samples);
      setError(null);
      setLastUpdated(new Date());
    } catch (error) {
      setError('Could not refresh this workload. ' + (error instanceof Error ? error.message : 'Backend unavailable.'));
    } finally {
      setLoading(false);
      setRetrying(false);
      inFlight.current = false;
    }
  }, [params.id]);

  useEffect(() => {
    if (!params.id) return;
    const initial = setTimeout(() => void loadWorkload(), 0);
    const interval = setInterval(loadWorkload, 3000);
    return () => { clearTimeout(initial); clearInterval(interval); };
  }, [params.id, loadWorkload]);

  const handleDiagnose = async () => {
    setDiagnosing(true);
    try {
      const report = await api.diagnoseWorkload(params.id as string, true);
      setDiagnosis(report);
    } catch (error) {
      alert('Failed to run diagnosis: ' + (error as Error).message);
    } finally {
      setDiagnosing(false);
    }
  };

  const formatDuration = (seconds?: number) => {
    if (seconds == null) return 'N/A';
    const mins = Math.floor(seconds / 60);
    const secs = Math.floor(seconds % 60);
    return `${mins}m ${secs}s`;
  };

  const getChartData = () => {
    if (!metrics || metrics.length === 0) return [];
    return metrics.map((m, idx) => ({
      index: idx,
      memory: m.gpu_memory_percent,
      utilization: m.gpu_utilization_percent,
    }));
  };

  const getTimelineEvents = () => {
    const events = [];

    if (workload?.created_at) {
      events.push({
        time: new Date(workload.created_at),
        label: 'Job Created',
        icon: Activity,
        color: 'blue',
      });
    }

    if (workload?.started_at) {
      events.push({
        time: new Date(workload.started_at),
        label: 'Job Started',
        icon: Zap,
        color: 'green',
      });
    }

    if (workload?.finished_at) {
      events.push({
        time: new Date(workload.finished_at),
        label: workload.status === 'failed' ? 'Job Failed' : 'Job Completed',
        icon: workload.status === 'failed' ? AlertTriangle : CheckCircle,
        color: workload.status === 'failed' ? 'red' : 'green',
      });
    }

    return events.sort((a, b) => a.time.getTime() - b.time.getTime());
  };

  const extractCrashLogs = () => {
    if (!workload?.job_logs) return '';
    const lines = workload.job_logs.split('\n');
    // Get last 15 lines before crash
    return lines.slice(-15).join('\n');
  };

  if (loading) {
    return (
      <div className="min-h-screen bg-slate-900 flex items-center justify-center">
        <div className="text-white text-xl">Loading...</div>
      </div>
    );
  }

  if (!workload) {
    return (
      <div className="min-h-screen bg-slate-900 flex items-center justify-center">
        <div className="max-w-2xl p-8">
          <Link href="/dashboard" className="text-blue-400">Back to Dashboard</Link>
          <ConnectionNotice error={error || 'Workload not found.'} lastUpdated={lastUpdated} retrying={retrying} onRetry={() => void loadWorkload()} />
        </div>
      </div>
    );
  }

  const { sampledDevice, frameworkAllocatedMB } = getMemoryPeaks(metrics);
  const peakMemory = sampledDevice ? { used: sampledDevice.gpu_memory_used_mb, total: sampledDevice.gpu_memory_total_mb, percent: sampledDevice.gpu_memory_percent } : null;
  const displayStatus = getWorkloadStatus(workload);
  const chartData = getChartData();
  const timelineEvents = getTimelineEvents();
  const crashLogs = extractCrashLogs();

  return (
    <div className="min-h-screen bg-slate-900">
      <div className="container mx-auto px-4 py-8">
        {/* Header */}
        <div className="mb-6">
          <Link
            href="/dashboard"
            className="text-blue-400 hover:text-blue-300 flex items-center gap-2 mb-4"
          >
            <ArrowLeft className="w-4 h-4" />
            Back to Dashboard
          </Link>
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-4xl font-bold text-white mb-2">{workload.name}</h1>
              <p className="text-slate-400">
                {workload.type} • Created {new Date(workload.created_at).toLocaleString()}
              </p>
              {workload.status === 'failed' && workload.failure_type && (
                <div className="mt-2 flex items-center gap-2 text-sm">
                  <span className="text-slate-500">Failure Signal:</span>
                  <span className="text-red-400 font-medium">
                    {workload.failure_type === 'GPU_OUT_OF_MEMORY' ? 'GPU out of memory' : workload.failure_type.replace(/_/g, ' ')}
                  </span>
                </div>
              )}
            </div>
            {workload.status === 'failed' && (
              <button
                onClick={handleDiagnose}
                disabled={diagnosing}
                className="px-6 py-3 bg-blue-600 hover:bg-blue-700 text-white rounded-lg font-semibold disabled:opacity-50 disabled:cursor-not-allowed transition-colors flex items-center gap-2"
              >
                {diagnosing ? (
                  <>
                    <Loader2 className="w-4 h-4 animate-spin" />
                    Diagnosing...
                  </>
                ) : diagnosis ? 'Re-run Diagnosis' : 'Run Diagnosis'}
              </button>
            )}
          </div>
        </div>

        <ConnectionNotice error={error} lastUpdated={lastUpdated} retrying={retrying} onRetry={() => void loadWorkload()} />
        {workload.heartbeat_stale && (
          <div role="status" className="mb-6 rounded-lg border border-amber-600 bg-amber-950/40 p-4 text-amber-100">
            No heartbeat for over two minutes. This job may have stopped or lost its connection; a failure has not been confirmed.
            {workload.last_heartbeat_at && <p className="mt-1 text-sm">Last contact: {new Date(workload.last_heartbeat_at).toLocaleString()}</p>}
            <p className="mt-1 text-sm">Monitoring will resume when the SDK reconnects.</p>
          </div>
        )}

        {/* Status Overview */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-8">
          <div className="bg-slate-800 border border-slate-700 rounded-lg p-4">
            <div className="flex items-center gap-2 mb-2">
              {workload.status === 'succeeded' ? (
                <CheckCircle className="w-5 h-5 text-green-400" />
              ) : (workload.status === 'failed' || workload.heartbeat_stale) ? (
                <AlertTriangle className="w-5 h-5 text-red-400" />
              ) : (
                <Activity className="w-5 h-5 text-blue-400" />
              )}
              <span className="text-sm text-slate-400">Status</span>
            </div>
            <div className="text-xl font-bold text-white capitalize">{displayStatus}</div>
          </div>

          <div className="bg-slate-800 border border-slate-700 rounded-lg p-4">
            <div className="flex items-center gap-2 mb-2">
              <Clock className="w-5 h-5 text-purple-400" />
              <span className="text-sm text-slate-400">Runtime</span>
            </div>
            <div className="text-xl font-bold text-white">
              {formatDuration(workload.runtime_seconds)}
            </div>
          </div>

          <div className="bg-slate-800 border border-slate-700 rounded-lg p-4">
            <div className="flex items-center gap-2 mb-2">
              <Terminal className="w-5 h-5 text-orange-400" />
              <span className="text-sm text-slate-400">Exit Code</span>
            </div>
            <div className="text-xl font-bold text-white">{workload.exit_code ?? 'N/A'}</div>
          </div>

          <div className="bg-slate-800 border border-slate-700 rounded-lg p-4">
            <div className="flex items-center gap-2 mb-2">
              <Zap className="w-5 h-5 text-yellow-400" />
              <span className="text-sm text-slate-400">Estimated GPU Seconds Wasted</span>
            </div>
            <div className="text-xl font-bold text-white">
              {workload.wasted_gpu_seconds ? Math.floor(workload.wasted_gpu_seconds) : 0}s
            </div>
          </div>
        </div>

        {metrics.length > 0 && (
          <div className="bg-slate-800 border border-slate-700 rounded-lg p-4 mb-8 text-slate-300">
            <h2 className="font-semibold text-white mb-2">GPU memory peaks</h2>
            {peakMemory && <p>Peak sampled device usage: {peakMemory.used.toFixed(1)} MB / {peakMemory.total.toFixed(1)} MB ({peakMemory.percent.toFixed(1)}%)</p>}
            <p>Peak framework allocation: {frameworkAllocatedMB == null ? 'Unavailable' : `${frameworkAllocatedMB.toFixed(1)} MB`}</p>
            <p className="text-sm text-slate-400 mt-2">Device usage is sampled periodically. Framework allocation records PyTorch’s high-water mark during this run, including peaks between samples.</p>
          </div>
        )}

        {/* Diagnosis Report - CENTERPIECE */}
        {diagnosis && (
          <div className="bg-gradient-to-br from-slate-800 to-slate-900 border-2 border-blue-500/30 rounded-lg p-8 mb-8 shadow-2xl">
            <h2 className="text-3xl font-bold text-white mb-6 flex items-center gap-3">
              <Activity className="w-8 h-8 text-blue-400" />
              {diagnosis.source === 'ai' ? 'AI-assisted Diagnosis Report' : 'Rule-based Diagnosis Report'}
            </h2>

            <p className="text-slate-300 mb-6">
              {diagnosis.source === 'ai'
                ? 'Generated through Fireworks AI using workload logs and GPU metrics.'
                : (diagnosis.ai_unavailable_reason || 'Generated from log-matching rules. AI-assisted diagnosis was unavailable.')}
            </p>

            {diagnosis.source === 'ai' && diagnosis.requested_model && (
              <p className="text-sm text-slate-400 mb-6">
                Requested: {diagnosis.requested_model} · Served by: {diagnosis.served_model || 'Not reported by Fireworks'}
              </p>
            )}

            {/* Failure Type & Confidence - Prominent Display */}
            <div className="bg-slate-900 rounded-lg p-6 mb-6 border border-slate-700">
              <div className="grid md:grid-cols-2 gap-6">
                <div>
                  <div className="text-sm font-medium text-slate-400 mb-2">Failure Type</div>
                  <div className="text-2xl font-bold text-red-400">
                    {diagnosis.failure_type}
                  </div>
                </div>
                <div>
                  <div className="text-sm font-medium text-slate-400 mb-2">Heuristic log-match score</div>
                  <div className="text-2xl font-bold text-green-400">
                    {(diagnosis.confidence * 100).toFixed(0)}%
                    <p className="text-sm font-normal text-slate-400 mt-2">This score is not a measured probability of correctness.</p>
                  </div>
                </div>
              </div>
            </div>

            {/* Root Cause */}
            <div className="mb-6">
              <h3 className="text-xl font-semibold text-white mb-3">Root Cause</h3>
              <div className="text-slate-200 bg-slate-900 rounded-lg p-6 leading-relaxed border border-slate-700">
                {diagnosis.root_cause}
              </div>
            </div>

            {/* Evidence Section */}
            <div className="mb-6">
              <h3 className="text-xl font-semibold text-white mb-3">Evidence</h3>
              <div className="bg-slate-900 rounded-lg p-6 space-y-3 border border-slate-700">
                {diagnosis.evidence && diagnosis.evidence.length > 0 && (
                  <>
                    {diagnosis.evidence.map((line, idx) => (
                      <div key={idx} className="flex items-start gap-2">
                        <AlertTriangle className="w-4 h-4 text-red-400 mt-1 flex-shrink-0" />
                        <span className="text-red-300 font-mono text-sm">{line}</span>
                      </div>
                    ))}
                  </>
                )}
                {workload.exit_code && (
                  <div className="flex items-center gap-2">
                    <Terminal className="w-4 h-4 text-purple-400" />
                    <span className="text-slate-300">
                      Exit Code: <span className="font-semibold text-white">{workload.exit_code}</span>
                    </span>
                  </div>
                )}
              </div>
            </div>

            {/* Recommended Repair - Highlighted */}
            <div className="mb-6">
              <h3 className="text-xl font-semibold text-white mb-3">Recommended Repair</h3>
              <div className="bg-blue-900/20 border border-blue-500/30 rounded-lg p-6">
                <div className="text-slate-200 whitespace-pre-line leading-relaxed">
                  {diagnosis.recommended_fix}
                </div>
              </div>
            </div>

            {/* Safe to Retry */}
            <div className="bg-slate-900 rounded-lg p-6 border border-slate-700">
              <h3 className="text-lg font-semibold text-white mb-3">Safe to Retry</h3>
              <div className="flex items-start gap-3">
                {diagnosis.safe_to_retry ? (
                  <>
                    <CheckCircle className="w-6 h-6 text-green-400 flex-shrink-0 mt-1" />
                    <div>
                      <div className="text-green-400 font-semibold mb-1">
                        Yes, safe to retry without changes
                      </div>
                      <div className="text-slate-400 text-sm">
                        The diagnosis indicates that an unchanged retry may succeed.
                      </div>
                    </div>
                  </>
                ) : (
                  <>
                    <AlertTriangle className="w-6 h-6 text-orange-400 flex-shrink-0 mt-1" />
                    <div>
                      <div className="text-orange-400 font-semibold mb-1">
                        Configuration changes required before retry
                      </div>
                      <div className="text-slate-400 text-sm">
                        This issue requires careful configuration review before attempting to re-run.
                      </div>
                    </div>
                  </>
                )}
              </div>
            </div>
          </div>
        )}

        {/* GPU Metrics Charts */}
        {chartData.length > 0 && (
          <div className="grid md:grid-cols-2 gap-6 mb-8">
            {/* GPU Memory Chart */}
            <div className="bg-slate-800 border border-slate-700 rounded-lg p-6">
              <div className="flex items-center gap-2 mb-4">
                <Database className="w-5 h-5 text-blue-400" />
                <h3 className="text-xl font-semibold text-white">GPU Memory Usage{metrics.some(m => m.source === 'Simulated') ? ' (simulated demo data)' : ''}</h3>
              </div>
              <ResponsiveContainer width="100%" height={300}>
                <LineChart data={chartData}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#374151" />
                  <XAxis dataKey="index" stroke="#9CA3AF" />
                  <YAxis stroke="#9CA3AF" domain={[0, (dataMax: number) => Math.max(100, dataMax)]} />
                  <Tooltip
                    contentStyle={{
                      backgroundColor: '#1e293b',
                      border: '1px solid #475569',
                      borderRadius: '8px',
                    }}
                    labelStyle={{ color: '#e2e8f0' }}
                  />
                  <Legend />
                  <Line
                    type="monotone"
                    dataKey="memory"
                    stroke="#3b82f6"
                    strokeWidth={2}
                    name="Memory %"
                    dot={{ fill: '#3b82f6', r: 3 }}
                  />
                </LineChart>
              </ResponsiveContainer>
            </div>

            {/* GPU Utilization Chart */}
            <div className="bg-slate-800 border border-slate-700 rounded-lg p-6">
              <div className="flex items-center gap-2 mb-4">
                <Cpu className="w-5 h-5 text-green-400" />
                <h3 className="text-xl font-semibold text-white">GPU Utilization</h3>
              </div>
              {!metrics.some(m => m.gpu_utilization_percent != null) ? (
                <p className="text-slate-400">Utilization is unavailable from this telemetry source.</p>
              ) : (
              <ResponsiveContainer width="100%" height={300}>
                <LineChart data={chartData}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#374151" />
                  <XAxis dataKey="index" stroke="#9CA3AF" />
                  <YAxis stroke="#9CA3AF" domain={[0, 100]} />
                  <Tooltip
                    contentStyle={{
                      backgroundColor: '#1e293b',
                      border: '1px solid #475569',
                      borderRadius: '8px',
                    }}
                    labelStyle={{ color: '#e2e8f0' }}
                  />
                  <Legend />
                  <Line
                    type="monotone"
                    dataKey="utilization"
                    stroke="#10b981"
                    strokeWidth={2}
                    name="Utilization %"
                    dot={{ fill: '#10b981', r: 3 }}
                  />
                </LineChart>
              </ResponsiveContainer>
              )}
            </div>
          </div>
        )}

        {/* Timeline */}
        {timelineEvents.length > 0 && (
          <div className="bg-slate-800 border border-slate-700 rounded-lg p-6 mb-8">
            <h3 className="text-xl font-semibold text-white mb-6">Job Timeline</h3>
            <div className="space-y-4">
              {timelineEvents.map((event, idx) => {
                const Icon = event.icon;
                const colorClasses = {
                  blue: 'bg-blue-500',
                  green: 'bg-green-500',
                  red: 'bg-red-500',
                };
                return (
                  <div key={idx} className="flex items-center gap-4">
                    <div className={`w-10 h-10 ${colorClasses[event.color as keyof typeof colorClasses]} rounded-full flex items-center justify-center`}>
                      <Icon className="w-5 h-5 text-white" />
                    </div>
                    <div className="flex-1">
                      <div className="text-white font-medium">{event.label}</div>
                      <div className="text-slate-400 text-sm">
                        {event.time.toLocaleTimeString()} ({event.time.toLocaleDateString()})
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        )}

        {/* Logs Near Crash */}
        {crashLogs && (
          <div className="bg-slate-800 border border-slate-700 rounded-lg overflow-hidden">
            <div className="px-6 py-4 border-b border-slate-700">
              <h3 className="text-xl font-semibold text-white">Logs Near Crash</h3>
              <p className="text-slate-400 text-sm mt-1">Last 15 lines before job termination</p>
            </div>
            <div className="p-6 bg-black overflow-x-auto max-h-96">
              <pre className="text-sm text-green-400 font-mono">{crashLogs}</pre>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
