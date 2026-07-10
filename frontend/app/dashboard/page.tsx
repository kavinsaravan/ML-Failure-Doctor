'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { api, Workload, Stats } from '@/lib/api';
import {
  Activity,
  AlertCircle,
  CheckCircle,
  Clock,
  TrendingUp,
  Timer,
  PlayCircle,
} from 'lucide-react';

export default function Dashboard() {
  const [workloads, setWorkloads] = useState<Workload[]>([]);
  const [stats, setStats] = useState<Stats | null>(null);
  const [loading, setLoading] = useState(true);
  const [runningJob, setRunningJob] = useState<string | null>(null);

  useEffect(() => {
    loadData();
    const interval = setInterval(loadData, 3000); // Refresh every 3s
    return () => clearInterval(interval);
  }, []);

  const loadData = async () => {
    try {
      const [workloadsData, statsData] = await Promise.all([
        api.getWorkloads(),
        api.getStats(),
      ]);
      setWorkloads(workloadsData);
      setStats(statsData);
    } catch (error) {
      console.error('Failed to load data:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleRunJob = async (template: string) => {
    setRunningJob(template);
    try {
      const result = await api.runWorkload(template);
      console.log('Job started:', result);
      setTimeout(loadData, 1000);
    } catch (error) {
      console.error('Failed to run job:', error);
    } finally {
      setRunningJob(null);
    }
  };

  const formatDuration = (seconds?: number) => {
    if (!seconds) return '—';
    const mins = Math.floor(seconds / 60);
    const secs = Math.floor(seconds % 60);
    return `${mins}m ${secs}s`;
  };

  const formatFailureType = (type?: string) => {
    if (!type) return '—';
    return type
      .split('_')
      .map((w) => w.charAt(0) + w.slice(1).toLowerCase())
      .join(' ');
  };

  const getStatusBadge = (status: string) => {
    const styles = {
      failed: 'bg-red-100 text-red-800 border-red-200',
      succeeded: 'bg-green-100 text-green-800 border-green-200',
      running: 'bg-blue-100 text-blue-800 border-blue-200',
      pending: 'bg-yellow-100 text-yellow-800 border-yellow-200',
    };
    return styles[status as keyof typeof styles] || 'bg-gray-100 text-gray-800';
  };

  const getMostCommonFailure = () => {
    if (!stats?.failure_types) return 'None';
    const entries = Object.entries(stats.failure_types);
    if (entries.length === 0) return 'None';
    const [type] = entries.sort((a, b) => b[1] - a[1])[0];
    return formatFailureType(type);
  };

  const successRate = stats
    ? ((stats.succeeded_workloads / (stats.total_workloads || 1)) * 100).toFixed(1)
    : '0';

  if (loading) {
    return (
      <div className="min-h-screen bg-slate-900 flex items-center justify-center">
        <div className="text-white text-xl">Loading...</div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-slate-900">
      <div className="container mx-auto px-4 py-8">
        {/* Header */}
        <div className="mb-8">
          <div className="flex items-center gap-3 mb-3">
            <h1 className="text-4xl font-bold text-white">CrashLens Dashboard</h1>
            <span className="px-3 py-1 bg-red-600 text-white text-sm font-semibold rounded">AMD ROCm</span>
          </div>
          <p className="text-slate-400">Monitor and diagnose ML workloads and AI agents on AMD GPUs</p>
          <div className="flex items-center gap-6 mt-3 text-sm">
            <div className="flex items-center gap-2">
              <span className="text-slate-500">Platform:</span>
              <span className="text-white font-medium">AMD ROCm 5.7+</span>
            </div>
            <div className="flex items-center gap-2">
              <span className="text-slate-500">Worker Type:</span>
              <span className="text-white font-medium">AMD GPU Worker</span>
            </div>
            <div className="flex items-center gap-2">
              <span className="text-slate-500">Metric Source:</span>
              <span className="text-white font-medium">rocm-smi compatible</span>
            </div>
          </div>
        </div>

        {/* Navigation Tabs */}
        <div className="mb-6 border-b border-slate-700">
          <div className="flex gap-8">
            <div className="pb-4 border-b-2 border-blue-500 text-blue-400 font-medium">
              ML Jobs
            </div>
            <Link href="/agent-runs" className="pb-4 text-slate-400 hover:text-white transition-colors">
              Agent Runs
            </Link>
          </div>
        </div>

        {/* Stats Cards */}
        <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4 mb-8">
          <div className="bg-slate-800 border border-slate-700 rounded-lg p-4">
            <div className="flex items-center gap-2 mb-2">
              <Activity className="w-4 h-4 text-blue-400" />
              <span className="text-sm text-slate-400">Total Workloads</span>
            </div>
            <div className="text-2xl font-bold text-white">{stats?.total_workloads || 0}</div>
          </div>

          <div className="bg-slate-800 border border-slate-700 rounded-lg p-4">
            <div className="flex items-center gap-2 mb-2">
              <AlertCircle className="w-4 h-4 text-red-400" />
              <span className="text-sm text-slate-400">Failed Workloads</span>
            </div>
            <div className="text-2xl font-bold text-white">{stats?.failed_workloads || 0}</div>
          </div>

          <div className="bg-slate-800 border border-slate-700 rounded-lg p-4">
            <div className="flex items-center gap-2 mb-2">
              <CheckCircle className="w-4 h-4 text-green-400" />
              <span className="text-sm text-slate-400">Success Rate</span>
            </div>
            <div className="text-2xl font-bold text-white">{successRate}%</div>
          </div>

          <div className="bg-slate-800 border border-slate-700 rounded-lg p-4">
            <div className="flex items-center gap-2 mb-2">
              <TrendingUp className="w-4 h-4 text-purple-400" />
              <span className="text-sm text-slate-400">Most Common</span>
            </div>
            <div className="text-sm font-semibold text-white truncate">
              {getMostCommonFailure()}
            </div>
          </div>

          <div className="bg-slate-800 border border-slate-700 rounded-lg p-4">
            <div className="flex items-center gap-2 mb-2">
              <Clock className="w-4 h-4 text-orange-400" />
              <span className="text-sm text-slate-400">Wasted GPU Time</span>
            </div>
            <div className="text-2xl font-bold text-white">
              {Math.floor(stats?.wasted_gpu_seconds || 0)}s
            </div>
          </div>

          <div className="bg-slate-800 border border-slate-700 rounded-lg p-4">
            <div className="flex items-center gap-2 mb-2">
              <Timer className="w-4 h-4 text-cyan-400" />
              <span className="text-sm text-slate-400">Avg Diagnosis</span>
            </div>
            <div className="text-2xl font-bold text-white">&lt;1s</div>
          </div>
        </div>

        {/* Quick Actions */}
        <div className="mb-8">
          <h2 className="text-xl font-semibold text-white mb-4">Quick Test Jobs</h2>
          <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-3">
            {[
              { template: 'successful', label: 'Successful Job', color: 'green' },
              { template: 'gpu_oom', label: 'GPU OOM', color: 'red' },
              { template: 'missing_checkpoint', label: 'Missing Checkpoint', color: 'orange' },
              { template: 'dependency_error', label: 'Dependency Error', color: 'yellow' },
              { template: 'data_path_error', label: 'Data Path Error', color: 'purple' },
              { template: 'timeout', label: 'Timeout', color: 'blue' },
            ].map(({ template, label, color }) => (
              <button
                key={template}
                onClick={() => handleRunJob(template)}
                disabled={runningJob === template}
                className={`px-4 py-3 bg-slate-800 hover:bg-slate-700 border border-slate-700 text-white rounded-lg text-sm font-medium flex items-center justify-center gap-2 transition-colors disabled:opacity-50 disabled:cursor-not-allowed`}
              >
                <PlayCircle className="w-4 h-4" />
                {runningJob === template ? 'Running...' : label}
              </button>
            ))}
          </div>
        </div>

        {/* Workloads Table */}
        <div className="bg-slate-800 border border-slate-700 rounded-lg overflow-hidden">
          <div className="px-6 py-4 border-b border-slate-700">
            <h2 className="text-xl font-semibold text-white">Recent Workloads</h2>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="bg-slate-700/50">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-slate-300 uppercase">
                    Workload
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-slate-300 uppercase">
                    Type
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-slate-300 uppercase">
                    Status
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-slate-300 uppercase">
                    Failure
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-slate-300 uppercase">
                    Runtime
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-slate-300 uppercase">
                    Actions
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-700">
                {workloads.map((workload) => (
                  <tr key={workload.id} className="hover:bg-slate-700/30">
                    <td className="px-6 py-4 whitespace-nowrap">
                      <Link
                        href={`/workloads/${workload.id}`}
                        className="text-blue-400 hover:text-blue-300 font-medium"
                      >
                        {workload.name}
                      </Link>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-slate-300">
                      {workload.type}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <span
                        className={`px-2 py-1 text-xs font-medium rounded border ${getStatusBadge(
                          workload.status
                        )}`}
                      >
                        {workload.status.toUpperCase()}
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-slate-300">
                      {formatFailureType(workload.failure_type)}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-slate-300">
                      {formatDuration(workload.runtime_seconds)}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <Link
                        href={`/workloads/${workload.id}`}
                        className="text-blue-400 hover:text-blue-300 text-sm"
                      >
                        View Details
                      </Link>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  );
}
