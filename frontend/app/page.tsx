'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { api, Workload, Stats } from '@/lib/api';
import {
  Zap,
  Shield,
  TrendingUp,
  Activity,
  AlertCircle,
  CheckCircle,
  Clock,
  Timer,
  PlayCircle,
} from 'lucide-react';

export default function Home() {
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

  const scrollToDashboard = () => {
    const dashboardElement = document.getElementById('dashboard-section');
    if (dashboardElement) {
      dashboardElement.scrollIntoView({ behavior: 'smooth' });
    }
  };

  return (
    <div className="min-h-screen bg-slate-950">
      {/* Hero Section */}
      <div className="relative bg-gradient-to-br from-slate-900 via-slate-800 to-slate-900 overflow-hidden">
        {/* Animated background elements */}
        <div className="absolute inset-0 overflow-hidden">
          <div className="absolute -top-40 -right-40 w-80 h-80 bg-blue-500/10 rounded-full blur-3xl animate-pulse"></div>
          <div className="absolute -bottom-40 -left-40 w-80 h-80 bg-purple-500/10 rounded-full blur-3xl animate-pulse delay-1000"></div>
        </div>

        <div className="container mx-auto px-4 py-20 relative">
          <div className="text-center max-w-6xl mx-auto">
            <div className="inline-block mb-4 px-4 py-2 bg-blue-500/10 border border-blue-500/20 rounded-full">
              <span className="text-blue-400 text-sm font-semibold">AI-Powered Reliability Platform</span>
            </div>
            <h1 className="text-6xl md:text-7xl font-bold text-white mb-6 leading-tight">
              Diagnose failed GPU workloads
              <br />
              and AI agents in{' '}
              <span className="bg-gradient-to-r from-blue-400 to-purple-400 bg-clip-text text-transparent">
                seconds
              </span>
            </h1>
            <p className="text-xl text-slate-300 mb-12 leading-relaxed max-w-3xl mx-auto">
              CrashLens monitors AMD GPU workloads, captures logs and metrics, and uses Gemma to generate root-cause analysis and repair recommendations.
            </p>
          </div>

          {/* Features */}
          <div className="grid md:grid-cols-3 gap-8 mt-12 max-w-6xl mx-auto">
            <div className="group bg-gradient-to-br from-slate-800 to-slate-800/50 border border-slate-700 hover:border-blue-500/50 rounded-xl p-6 transition-all duration-300 hover:shadow-xl hover:shadow-blue-500/10 hover:-translate-y-1">
              <div className="w-14 h-14 bg-gradient-to-br from-blue-500 to-blue-600 rounded-xl flex items-center justify-center mb-4 group-hover:scale-110 transition-transform duration-300">
                <Zap className="w-7 h-7 text-white" />
              </div>
              <h3 className="text-xl font-semibold text-white mb-3">
                Instant Diagnosis
              </h3>
              <p className="text-slate-400 leading-relaxed">
                AI-powered failure classification with confidence scoring and evidence extraction from logs.
              </p>
            </div>

            <div className="group bg-gradient-to-br from-slate-800 to-slate-800/50 border border-slate-700 hover:border-green-500/50 rounded-xl p-6 transition-all duration-300 hover:shadow-xl hover:shadow-green-500/10 hover:-translate-y-1">
              <div className="w-14 h-14 bg-gradient-to-br from-green-500 to-green-600 rounded-xl flex items-center justify-center mb-4 group-hover:scale-110 transition-transform duration-300">
                <Shield className="w-7 h-7 text-white" />
              </div>
              <h3 className="text-xl font-semibold text-white mb-3">
                AMD ROCm Native
              </h3>
              <p className="text-slate-400 leading-relaxed">
                Built for AMD GPUs with HIP/ROCm error detection and rocm-smi metrics integration.
              </p>
            </div>

            <div className="group bg-gradient-to-br from-slate-800 to-slate-800/50 border border-slate-700 hover:border-purple-500/50 rounded-xl p-6 transition-all duration-300 hover:shadow-xl hover:shadow-purple-500/10 hover:-translate-y-1">
              <div className="w-14 h-14 bg-gradient-to-br from-purple-500 to-purple-600 rounded-xl flex items-center justify-center mb-4 group-hover:scale-110 transition-transform duration-300">
                <TrendingUp className="w-7 h-7 text-white" />
              </div>
              <h3 className="text-xl font-semibold text-white mb-3">
                Cost Tracking
              </h3>
              <p className="text-slate-400 leading-relaxed">
                Monitor wasted GPU-seconds on failed jobs and optimize resource usage.
              </p>
            </div>
          </div>

          {/* Get Started Button */}
          <div className="flex justify-center mt-12">
            <button
              onClick={scrollToDashboard}
              className="group px-10 py-4 bg-gradient-to-r from-blue-600 to-blue-700 hover:from-blue-700 hover:to-blue-800 text-white rounded-xl font-semibold transition-all duration-300 shadow-lg shadow-blue-500/20 hover:shadow-xl hover:shadow-blue-500/30 hover:scale-105"
            >
              <span className="flex items-center gap-2">
                Get Started
                <svg className="w-5 h-5 group-hover:translate-y-1 transition-transform duration-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                </svg>
              </span>
            </button>
          </div>

          {/* Supported Failure Types */}
          <div className="mt-20 max-w-5xl mx-auto mb-8">
            <h2 className="text-3xl md:text-4xl font-bold text-white text-center mb-4">
              Automatically Detects
            </h2>
            <p className="text-slate-400 text-center mb-10 max-w-2xl mx-auto">
              Instantly identify and diagnose common GPU workload failures
            </p>
            <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
              {[
                { icon: '💾', text: 'GPU Out of Memory' },
                { icon: '📁', text: 'Missing Checkpoint' },
                { icon: '📦', text: 'Dependency Errors' },
                { icon: '🗂️', text: 'Data Path Errors' },
                { icon: '⏱️', text: 'Job Timeouts' },
                { icon: '🔧', text: 'ROCm Driver Issues' },
              ].map((item) => (
                <div
                  key={item.text}
                  className="group bg-slate-800/50 backdrop-blur border border-slate-700 hover:border-slate-600 rounded-xl px-5 py-4 text-slate-300 hover:text-white transition-all duration-300 hover:shadow-lg hover:shadow-slate-700/20 hover:-translate-y-0.5"
                >
                  <div className="flex items-center justify-center gap-3">
                    <span className="text-2xl group-hover:scale-110 transition-transform duration-300">{item.icon}</span>
                    <span className="font-medium">{item.text}</span>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>

      {/* Dashboard Section */}
      <div id="dashboard-section" className="container mx-auto px-4 py-8">
        {/* Header */}
        <div className="mb-8">
          <div className="flex items-center gap-3 mb-3">
            <h2 className="text-4xl font-bold text-white">CrashLens Dashboard</h2>
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
              GPU Workloads
            </div>
            <Link href="/agent-runs" className="pb-4 text-slate-400 hover:text-white transition-colors">
              Agent Runs
            </Link>
          </div>
        </div>

        {loading ? (
          <div className="flex items-center justify-center h-64">
            <div className="text-white text-xl">Loading...</div>
          </div>
        ) : (
          <>
            {/* Stats Cards */}
            <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4 mb-8">
              <div className="group bg-gradient-to-br from-slate-800 to-slate-800/50 border border-slate-700 hover:border-blue-500/30 rounded-xl p-5 transition-all duration-300 hover:shadow-lg hover:shadow-blue-500/10">
                <div className="flex items-center gap-2 mb-3">
                  <div className="p-1.5 bg-blue-500/10 rounded-lg">
                    <Activity className="w-4 h-4 text-blue-400" />
                  </div>
                  <span className="text-xs text-slate-400 font-medium uppercase tracking-wide">Total Workloads</span>
                </div>
                <div className="text-3xl font-bold text-white">{stats?.total_workloads || 0}</div>
              </div>

              <div className="group bg-gradient-to-br from-slate-800 to-slate-800/50 border border-slate-700 hover:border-red-500/30 rounded-xl p-5 transition-all duration-300 hover:shadow-lg hover:shadow-red-500/10">
                <div className="flex items-center gap-2 mb-3">
                  <div className="p-1.5 bg-red-500/10 rounded-lg">
                    <AlertCircle className="w-4 h-4 text-red-400" />
                  </div>
                  <span className="text-xs text-slate-400 font-medium uppercase tracking-wide">Failed Workloads</span>
                </div>
                <div className="text-3xl font-bold text-white">{stats?.failed_workloads || 0}</div>
              </div>

              <div className="group bg-gradient-to-br from-slate-800 to-slate-800/50 border border-slate-700 hover:border-green-500/30 rounded-xl p-5 transition-all duration-300 hover:shadow-lg hover:shadow-green-500/10">
                <div className="flex items-center gap-2 mb-3">
                  <div className="p-1.5 bg-green-500/10 rounded-lg">
                    <CheckCircle className="w-4 h-4 text-green-400" />
                  </div>
                  <span className="text-xs text-slate-400 font-medium uppercase tracking-wide">Success Rate</span>
                </div>
                <div className="text-3xl font-bold text-white">{successRate}%</div>
              </div>

              <div className="group bg-gradient-to-br from-slate-800 to-slate-800/50 border border-slate-700 hover:border-purple-500/30 rounded-xl p-5 transition-all duration-300 hover:shadow-lg hover:shadow-purple-500/10">
                <div className="flex items-center gap-2 mb-3">
                  <div className="p-1.5 bg-purple-500/10 rounded-lg">
                    <TrendingUp className="w-4 h-4 text-purple-400" />
                  </div>
                  <span className="text-xs text-slate-400 font-medium uppercase tracking-wide">Most Common</span>
                </div>
                <div className="text-sm font-semibold text-white truncate">
                  {getMostCommonFailure()}
                </div>
              </div>

              <div className="group bg-gradient-to-br from-slate-800 to-slate-800/50 border border-slate-700 hover:border-orange-500/30 rounded-xl p-5 transition-all duration-300 hover:shadow-lg hover:shadow-orange-500/10">
                <div className="flex items-center gap-2 mb-3">
                  <div className="p-1.5 bg-orange-500/10 rounded-lg">
                    <Clock className="w-4 h-4 text-orange-400" />
                  </div>
                  <span className="text-xs text-slate-400 font-medium uppercase tracking-wide">Wasted GPU Time</span>
                </div>
                <div className="text-3xl font-bold text-white">
                  {Math.floor(stats?.wasted_gpu_seconds || 0)}s
                </div>
              </div>

              <div className="group bg-gradient-to-br from-slate-800 to-slate-800/50 border border-slate-700 hover:border-cyan-500/30 rounded-xl p-5 transition-all duration-300 hover:shadow-lg hover:shadow-cyan-500/10">
                <div className="flex items-center gap-2 mb-3">
                  <div className="p-1.5 bg-cyan-500/10 rounded-lg">
                    <Timer className="w-4 h-4 text-cyan-400" />
                  </div>
                  <span className="text-xs text-slate-400 font-medium uppercase tracking-wide">Avg Diagnosis</span>
                </div>
                <div className="text-3xl font-bold text-white">&lt;1s</div>
              </div>
            </div>

            {/* Quick Actions */}
            <div className="mb-8">
              <h2 className="text-2xl font-semibold text-white mb-5">Quick Test Jobs</h2>
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
                    className={`group px-4 py-3 bg-slate-800 hover:bg-slate-700 border border-slate-700 hover:border-slate-600 text-white rounded-xl text-sm font-medium flex items-center justify-center gap-2 transition-all duration-300 disabled:opacity-50 disabled:cursor-not-allowed hover:shadow-lg hover:shadow-slate-700/20 hover:-translate-y-0.5 ${runningJob === template ? 'animate-pulse' : ''}`}
                  >
                    <PlayCircle className="w-4 h-4 group-hover:scale-110 transition-transform duration-300" />
                    {runningJob === template ? 'Running...' : label}
                  </button>
                ))}
              </div>
            </div>

            {/* Workloads Table */}
            <div className="bg-gradient-to-br from-slate-800 to-slate-800/50 border border-slate-700 rounded-xl overflow-hidden shadow-xl">
              <div className="px-6 py-5 border-b border-slate-700 bg-slate-800/50">
                <h2 className="text-2xl font-semibold text-white">Recent Workloads</h2>
              </div>
              <div className="overflow-x-auto">
                <table className="w-full">
                  <thead className="bg-slate-700/30">
                    <tr>
                      <th className="px-6 py-4 text-left text-xs font-semibold text-slate-300 uppercase tracking-wider">
                        Workload
                      </th>
                      <th className="px-6 py-4 text-left text-xs font-semibold text-slate-300 uppercase tracking-wider">
                        Type
                      </th>
                      <th className="px-6 py-4 text-left text-xs font-semibold text-slate-300 uppercase tracking-wider">
                        Status
                      </th>
                      <th className="px-6 py-4 text-left text-xs font-semibold text-slate-300 uppercase tracking-wider">
                        Failure
                      </th>
                      <th className="px-6 py-4 text-left text-xs font-semibold text-slate-300 uppercase tracking-wider">
                        Runtime
                      </th>
                      <th className="px-6 py-4 text-left text-xs font-semibold text-slate-300 uppercase tracking-wider">
                        Actions
                      </th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-700/50">
                    {workloads.map((workload) => (
                      <tr key={workload.id} className="hover:bg-slate-700/20 transition-colors duration-200">
                        <td className="px-6 py-4 whitespace-nowrap">
                          <Link
                            href={`/workloads/${workload.id}`}
                            className="text-blue-400 hover:text-blue-300 font-medium transition-colors duration-200 hover:underline"
                          >
                            {workload.name}
                          </Link>
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-slate-300 font-medium">
                          {workload.type}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap">
                          <span
                            className={`px-3 py-1.5 text-xs font-semibold rounded-lg border ${getStatusBadge(
                              workload.status
                            )}`}
                          >
                            {workload.status.toUpperCase()}
                          </span>
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-slate-300">
                          {formatFailureType(workload.failure_type)}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-slate-300 font-mono text-sm">
                          {formatDuration(workload.runtime_seconds)}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap">
                          <Link
                            href={`/workloads/${workload.id}`}
                            className="inline-flex items-center gap-1 text-blue-400 hover:text-blue-300 text-sm font-medium transition-colors duration-200"
                          >
                            View Details
                            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                            </svg>
                          </Link>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          </>
        )}
      </div>
    </div>
  );
}
