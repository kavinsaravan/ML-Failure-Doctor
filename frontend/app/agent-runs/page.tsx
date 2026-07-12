"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { Activity, AlertCircle, CheckCircle, Clock, Zap, Brain } from "lucide-react";

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

interface AgentRun {
  id: string;
  workload_id: string;
  agent_name: string;
  task: string;
  status: string;
  failure_type?: string;
  total_tool_calls: number;
  total_model_calls: number;
  total_tokens?: number;
  total_latency_ms?: number;
  started_at: string;
  finished_at?: string;
}

export default function AgentRunsPage() {
  const [agentRuns, setAgentRuns] = useState<AgentRun[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchAgentRuns();
  }, []);

  const fetchAgentRuns = async () => {
    try {
      const response = await fetch(`${API_URL}/agent-runs`);
      const data = await response.json();
      setAgentRuns(data || []);
    } catch (error) {
      console.error("Failed to fetch agent runs:", error);
    } finally {
      setLoading(false);
    }
  };

  const getStatusBadge = (status: string) => {
    const colors = {
      completed: "bg-green-100 text-green-800",
      failed: "bg-red-100 text-red-800",
      running: "bg-blue-100 text-blue-800",
    };
    return colors[status as keyof typeof colors] || "bg-gray-100 text-gray-800";
  };

  const formatLatency = (ms?: number) => {
    if (!ms) return "—";
    if (ms < 1000) return `${ms}ms`;
    return `${(ms / 1000).toFixed(1)}s`;
  };

  // Calculate stats
  const totalRuns = agentRuns.length;
  const failedRuns = agentRuns.filter(r => r.status === 'failed').length;
  const completedRuns = agentRuns.filter(r => r.status === 'completed').length;
  const successRate = totalRuns > 0 ? ((completedRuns / totalRuns) * 100).toFixed(1) : '0';
  const totalToolCalls = agentRuns.reduce((sum, r) => sum + r.total_tool_calls, 0);
  const totalModelCalls = agentRuns.reduce((sum, r) => sum + r.total_model_calls, 0);
  const avgLatency = agentRuns.length > 0
    ? agentRuns.reduce((sum, r) => sum + (r.total_latency_ms || 0), 0) / agentRuns.length
    : 0;

  if (loading) {
    return (
      <div className="min-h-screen bg-slate-950 p-8">
        <div className="max-w-7xl mx-auto">
          <div className="flex items-center justify-center h-64">
            <div className="text-white text-xl">Loading agent runs...</div>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-slate-950 p-8">
      <div className="max-w-[1600px] mx-auto">
        {/* Header */}
        <div className="mb-8 text-center">
          <Link
            href="/"
            className="text-blue-400 hover:text-blue-300 inline-flex items-center gap-2 mb-4"
          >
            ← Back to Home
          </Link>
          <div className="flex items-center justify-center gap-3 mb-3">
            <h1 className="text-4xl font-bold text-white">CrashLens Dashboard</h1>
            <span className="px-3 py-1 bg-red-600 text-white text-sm font-semibold rounded">AMD ROCm</span>
          </div>
          <p className="text-slate-400">Monitor and diagnose ML workloads and AI agents on AMD GPUs</p>
          <div className="flex items-center justify-center gap-6 mt-3 text-sm">
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
          <div className="flex gap-8 justify-center">
            <Link href="/dashboard" className="pb-4 text-slate-400 hover:text-white transition-colors">
              GPU Workloads
            </Link>
            <div className="pb-4 border-b-2 border-blue-500 text-blue-400 font-medium">
              Agent Runs
            </div>
          </div>
        </div>

        {/* Stats Cards */}
        <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4 mb-8">
          <div className="group bg-gradient-to-br from-slate-800 to-slate-800/50 border border-slate-700 hover:border-blue-500/30 rounded-xl p-5 transition-all duration-300 hover:shadow-lg hover:shadow-blue-500/10">
            <div className="flex items-center gap-2 mb-3">
              <div className="p-1.5 bg-blue-500/10 rounded-lg">
                <Activity className="w-4 h-4 text-blue-400" />
              </div>
              <span className="text-xs text-slate-400 font-medium uppercase tracking-wide">Total Runs</span>
            </div>
            <div className="text-3xl font-bold text-white">{totalRuns}</div>
          </div>

          <div className="group bg-gradient-to-br from-slate-800 to-slate-800/50 border border-slate-700 hover:border-red-500/30 rounded-xl p-5 transition-all duration-300 hover:shadow-lg hover:shadow-red-500/10">
            <div className="flex items-center gap-2 mb-3">
              <div className="p-1.5 bg-red-500/10 rounded-lg">
                <AlertCircle className="w-4 h-4 text-red-400" />
              </div>
              <span className="text-xs text-slate-400 font-medium uppercase tracking-wide">Failed Runs</span>
            </div>
            <div className="text-3xl font-bold text-white">{failedRuns}</div>
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
                <Zap className="w-4 h-4 text-purple-400" />
              </div>
              <span className="text-xs text-slate-400 font-medium uppercase tracking-wide">Tool Calls</span>
            </div>
            <div className="text-3xl font-bold text-white">{totalToolCalls}</div>
          </div>

          <div className="group bg-gradient-to-br from-slate-800 to-slate-800/50 border border-slate-700 hover:border-cyan-500/30 rounded-xl p-5 transition-all duration-300 hover:shadow-lg hover:shadow-cyan-500/10">
            <div className="flex items-center gap-2 mb-3">
              <div className="p-1.5 bg-cyan-500/10 rounded-lg">
                <Brain className="w-4 h-4 text-cyan-400" />
              </div>
              <span className="text-xs text-slate-400 font-medium uppercase tracking-wide">Model Calls</span>
            </div>
            <div className="text-3xl font-bold text-white">{totalModelCalls}</div>
          </div>

          <div className="group bg-gradient-to-br from-slate-800 to-slate-800/50 border border-slate-700 hover:border-orange-500/30 rounded-xl p-5 transition-all duration-300 hover:shadow-lg hover:shadow-orange-500/10">
            <div className="flex items-center gap-2 mb-3">
              <div className="p-1.5 bg-orange-500/10 rounded-lg">
                <Clock className="w-4 h-4 text-orange-400" />
              </div>
              <span className="text-xs text-slate-400 font-medium uppercase tracking-wide">Avg Latency</span>
            </div>
            <div className="text-3xl font-bold text-white">{formatLatency(avgLatency)}</div>
          </div>
        </div>

        {/* Agent Runs Table */}
        {agentRuns.length === 0 ? (
          <div className="bg-slate-800 border border-slate-700 rounded-lg shadow p-8 text-center">
            <p className="text-slate-300">No agent runs yet</p>
            <p className="text-sm text-slate-500 mt-2">
              Run agent templates to see execution traces here
            </p>
          </div>
        ) : (
          <div className="bg-slate-800 border border-slate-700 rounded-lg shadow overflow-x-auto">
            <table className="min-w-full divide-y divide-slate-700">
              <thead className="bg-slate-700/50">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-slate-300 uppercase tracking-wider whitespace-nowrap">
                    Agent Name
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-slate-300 uppercase tracking-wider whitespace-nowrap">
                    Task
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-slate-300 uppercase tracking-wider whitespace-nowrap">
                    Status
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-slate-300 uppercase tracking-wider whitespace-nowrap">
                    Failure Type
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-slate-300 uppercase tracking-wider whitespace-nowrap">
                    Tool Calls
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-slate-300 uppercase tracking-wider whitespace-nowrap">
                    Model Calls
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-slate-300 uppercase tracking-wider whitespace-nowrap">
                    Latency
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-slate-300 uppercase tracking-wider whitespace-nowrap">
                    Actions
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-700">
                {agentRuns.map((run) => (
                  <tr key={run.id} className="hover:bg-slate-700/30">
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="text-sm font-medium text-white">
                        {run.agent_name}
                      </div>
                    </td>
                    <td className="px-6 py-4">
                      <div className="text-sm text-slate-300 max-w-xs truncate">
                        {run.task}
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <span
                        className={`px-2 py-1 text-xs font-medium rounded border ${getStatusBadge(
                          run.status
                        )}`}
                      >
                        {run.status}
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="text-sm text-slate-300">
                        {run.failure_type ? (
                          <span className="text-red-400 font-medium">
                            {run.failure_type}
                          </span>
                        ) : (
                          <span className="text-slate-500">—</span>
                        )}
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-300">
                      {run.total_tool_calls}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-300">
                      {run.total_model_calls}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-300">
                      {formatLatency(run.total_latency_ms)}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm">
                      <Link
                        href={`/agent-runs/${run.id}`}
                        className="text-blue-400 hover:text-blue-300 font-medium"
                      >
                        View Trace →
                      </Link>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}
