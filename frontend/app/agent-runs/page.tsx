"use client";

import { useEffect, useState } from "react";
import Link from "next/link";

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
      <div className="max-w-7xl mx-auto">
        {/* Header */}
        <div className="mb-8">
          <div className="flex items-center gap-4 mb-4">
            <Link
              href="/"
              className="text-blue-400 hover:text-blue-300 text-sm"
            >
              ← Back to Dashboard
            </Link>
          </div>
          <h1 className="text-3xl font-bold text-white">Agent Runs</h1>
          <p className="text-slate-400 mt-2">
            Monitor autonomous agent execution and diagnose failures
          </p>
        </div>

        {/* Navigation Tabs */}
        <div className="mb-6 border-b border-slate-700">
          <div className="flex gap-8">
            <Link href="/#dashboard-section" className="pb-4 text-slate-400 hover:text-white transition-colors">
              ML Jobs
            </Link>
            <div className="pb-4 border-b-2 border-blue-500 text-blue-400 font-medium">
              Agent Runs
            </div>
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
