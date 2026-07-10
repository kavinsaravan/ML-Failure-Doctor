'use client';

import { useEffect, useState } from 'react';
import { useParams } from 'next/navigation';
import Link from 'next/link';
import { api, Workload, DiagnosisReport } from '@/lib/api';
import {
  ArrowLeft,
  AlertTriangle,
  CheckCircle,
  Clock,
  Terminal,
  Activity,
  Zap,
  TrendingUp,
  Database,
  Cpu,
} from 'lucide-react';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend } from 'recharts';

interface MetricsSnapshot {
  timestamp: string;
  gpu_memory_used_mb: number;
  gpu_memory_total_mb: number;
  gpu_memory_percent: number;
  gpu_utilization_percent: number;
  temperature_celsius: number;
}

export default function WorkloadDetail() {
  const params = useParams();
  const [workload, setWorkload] = useState<Workload | null>(null);
  const [diagnosis, setDiagnosis] = useState<DiagnosisReport | null>(null);
  const [metrics, setMetrics] = useState<MetricsSnapshot[]>([]);
  const [loading, setLoading] = useState(true);
  const [diagnosing, setDiagnosing] = useState(false);

  useEffect(() => {
    if (params.id) {
      loadWorkload();
    }
  }, [params.id]);

  const loadWorkload = async () => {
    try {
      const data = await api.getWorkload(params.id as string);
      setWorkload(data);

      // Parse existing diagnosis if available
      if (data.failure_report) {
        try {
          const report = JSON.parse(data.failure_report);
          setDiagnosis(report);
        } catch (e) {
          console.error('Failed to parse diagnosis:', e);
        }
      }

      // Parse GPU metrics
      if (data.gpu_metrics) {
        try {
          const metricsData = JSON.parse(data.gpu_metrics);
          setMetrics(metricsData);
        } catch (e) {
          console.error('Failed to parse metrics:', e);
        }
      }
    } catch (error) {
      console.error('Failed to load workload:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleDiagnose = async () => {
    setDiagnosing(true);
    try {
      const report = await api.diagnoseWorkload(params.id as string);
      setDiagnosis(report);
      loadWorkload();
    } catch (error) {
      console.error('Failed to diagnose:', error);
    } finally {
      setDiagnosing(false);
    }
  };

  const formatDuration = (seconds?: number) => {
    if (!seconds) return 'N/A';
    const mins = Math.floor(seconds / 60);
    const secs = Math.floor(seconds % 60);
    return `${mins}m ${secs}s`;
  };

  const formatFailureType = (type: string) => {
    return type
      .split('_')
      .map((w) => w.charAt(0) + w.slice(1).toLowerCase())
      .join(' ');
  };

  const getChartData = () => {
    return metrics.map((m, idx) => ({
      index: idx,
      memory: m.gpu_memory_percent,
      utilization: m.gpu_utilization_percent,
      time: new Date(m.timestamp).toLocaleTimeString(),
    }));
  };

  const getPeakMemory = () => {
    if (metrics.length === 0) return null;
    const peak = metrics.reduce((max, m) =>
      m.gpu_memory_percent > max.gpu_memory_percent ? m : max
    );
    return {
      used: peak.gpu_memory_used_mb,
      total: peak.gpu_memory_total_mb,
      percent: peak.gpu_memory_percent,
    };
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
        <div className="text-white text-xl">Workload not found</div>
      </div>
    );
  }

  const peakMemory = getPeakMemory();
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
            </div>
            {workload.status === 'failed' && !diagnosis && (
              <button
                onClick={handleDiagnose}
                disabled={diagnosing}
                className="px-6 py-3 bg-blue-600 hover:bg-blue-700 text-white rounded-lg font-semibold disabled:opacity-50"
              >
                {diagnosing ? 'Diagnosing...' : 'Run AI Diagnosis'}
              </button>
            )}
          </div>
        </div>

        {/* Status Overview */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-8">
          <div className="bg-slate-800 border border-slate-700 rounded-lg p-4">
            <div className="flex items-center gap-2 mb-2">
              {workload.status === 'succeeded' ? (
                <CheckCircle className="w-5 h-5 text-green-400" />
              ) : workload.status === 'failed' ? (
                <AlertTriangle className="w-5 h-5 text-red-400" />
              ) : (
                <Activity className="w-5 h-5 text-blue-400" />
              )}
              <span className="text-sm text-slate-400">Status</span>
            </div>
            <div className="text-xl font-bold text-white capitalize">{workload.status}</div>
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
              <span className="text-sm text-slate-400">GPU Seconds Wasted</span>
            </div>
            <div className="text-xl font-bold text-white">
              {workload.wasted_gpu_seconds ? Math.floor(workload.wasted_gpu_seconds) : 0}s
            </div>
          </div>
        </div>

        {/* Diagnosis Report - CENTERPIECE */}
        {diagnosis && (
          <div className="bg-gradient-to-br from-slate-800 to-slate-900 border-2 border-blue-500/30 rounded-lg p-8 mb-8 shadow-2xl">
            <h2 className="text-3xl font-bold text-white mb-6 flex items-center gap-3">
              <Activity className="w-8 h-8 text-blue-400" />
              AI Diagnosis Report
            </h2>

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
                  <div className="text-sm font-medium text-slate-400 mb-2">Confidence</div>
                  <div className="text-2xl font-bold text-green-400">
                    {(diagnosis.confidence * 100).toFixed(0)}%
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
                {peakMemory && (
                  <div className="flex items-center gap-2">
                    <Database className="w-4 h-4 text-orange-400" />
                    <span className="text-slate-300">
                      Peak GPU Memory: <span className="font-semibold text-white">
                        {(peakMemory.used / 1024).toFixed(1)}GB / {(peakMemory.total / 1024).toFixed(1)}GB
                      </span> ({peakMemory.percent.toFixed(1)}%)
                    </span>
                  </div>
                )}
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

            {/* Recommended Fix - Highlighted */}
            <div className="mb-6">
              <h3 className="text-xl font-semibold text-white mb-3">Recommended Fix</h3>
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
                        Yes, safe to retry after applying fixes
                      </div>
                      <div className="text-slate-400 text-sm">
                        This issue can be resolved with configuration changes. Apply the recommended fixes and re-run the job.
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
                <h3 className="text-xl font-semibold text-white">GPU Memory Usage</h3>
              </div>
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
