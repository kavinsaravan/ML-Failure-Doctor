"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import Link from "next/link";

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

interface AgentRun {
  id: string;
  agent_name: string;
  task: string;
  status: string;
  failure_type?: string;
  total_tool_calls: number;
  total_model_calls: number;
  total_tokens?: number;
  total_latency_ms?: number;
}

interface AgentStep {
  id: string;
  step_index: number;
  step_type: string;
  name: string;
  input?: string;
  output?: string;
  status: string;
  latency_ms?: number;
}

interface Diagnosis {
  root_cause: string;
  evidence: string[];
  recommended_fixes: string[];
  prevention: string;
  failure_message?: string;
}

export default function AgentTracePage() {
  const params = useParams();
  const id = params.id as string;

  const [agentRun, setAgentRun] = useState<AgentRun | null>(null);
  const [steps, setSteps] = useState<AgentStep[]>([]);
  const [diagnosis, setDiagnosis] = useState<Diagnosis | null>(null);
  const [loading, setLoading] = useState(true);
  const [diagnosing, setDiagnosing] = useState(false);

  useEffect(() => {
    fetchAgentTrace();
  }, [id]);

  const fetchAgentTrace = async () => {
    try {
      const [runResponse, stepsResponse] = await Promise.all([
        fetch(`${API_URL}/agent-runs/${id}`),
        fetch(`${API_URL}/agent-runs/${id}/steps`),
      ]);

      const runData = await runResponse.json();
      const stepsData = await stepsResponse.json();

      setAgentRun(runData);
      setSteps(stepsData || []);
    } catch (error) {
      console.error("Failed to fetch agent trace:", error);
    } finally {
      setLoading(false);
    }
  };

  const runDiagnosis = async () => {
    setDiagnosing(true);
    try {
      const response = await fetch(
        `${API_URL}/agent-runs/${id}/diagnose`,
        {
          method: "POST",
        }
      );
      const data = await response.json();
      setDiagnosis(data);
    } catch (error) {
      console.error("Failed to run diagnosis:", error);
    } finally {
      setDiagnosing(false);
    }
  };

  const getStepIcon = (stepType: string) => {
    switch (stepType) {
      case "TOOL_CALL":
        return "🔧";
      case "MODEL_CALL":
        return "🤖";
      case "DECISION":
        return "🎯";
      case "ERROR":
        return "❌";
      case "FINAL_RESPONSE":
        return "✅";
      default:
        return "📝";
    }
  };

  const getStepColor = (status: string) => {
    switch (status) {
      case "completed":
      case "success":
        return "border-green-500/30 bg-[#2a3f4f]";
      case "failed":
      case "error":
        return "border-red-500/30 bg-[#2a3f4f]";
      default:
        return "border-slate-700/50 bg-[#2a3f4f]";
    }
  };

  const formatLatency = (ms?: number) => {
    if (!ms) return "";
    if (ms < 1000) return `${ms}ms`;
    return `${(ms / 1000).toFixed(1)}s`;
  };

  if (loading) {
    return (
      <div className="min-h-screen bg-[#0f1629] p-8">
        <div className="max-w-5xl mx-auto">
          <div className="flex items-center justify-center h-64">
            <div className="text-slate-400">Loading agent trace...</div>
          </div>
        </div>
      </div>
    );
  }

  if (!agentRun) {
    return (
      <div className="min-h-screen bg-[#0f1629] p-8">
        <div className="max-w-5xl mx-auto">
          <div className="text-center">
            <p className="text-slate-400">Agent run not found</p>
            <Link
              href="/agent-runs"
              className="text-blue-400 hover:text-blue-300 mt-4 inline-block"
            >
              ← Back to Agent Runs
            </Link>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-[#0f1629] p-8">
      <div className="max-w-5xl mx-auto">
        {/* Header */}
        <div className="mb-8">
          <Link
            href="/agent-runs"
            className="text-blue-400 hover:text-blue-300 text-sm mb-4 inline-block"
          >
            ← Back to Agent Runs
          </Link>
          <h1 className="text-3xl font-bold text-white">
            {agentRun.agent_name}
          </h1>
          <p className="text-slate-400 mt-2">{agentRun.task}</p>

          <div className="flex items-center gap-4 mt-4">
            <span
              className={`px-3 py-1 text-sm font-medium rounded-full ${
                agentRun.status === "completed"
                  ? "bg-green-500/20 text-green-400 border border-green-500/30"
                  : "bg-red-500/20 text-red-400 border border-red-500/30"
              }`}
            >
              {agentRun.status}
            </span>
            {agentRun.failure_type && (
              <span className="text-red-400 font-medium">
                {agentRun.failure_type}
              </span>
            )}
          </div>

          {/* Metrics */}
          <div className="grid grid-cols-4 gap-4 mt-6">
            <div className="bg-[#1e2a3f] border border-slate-700/50 rounded-lg shadow p-4">
              <div className="text-sm text-slate-400">Tool Calls</div>
              <div className="text-2xl font-bold text-white">
                {agentRun.total_tool_calls}
              </div>
            </div>
            <div className="bg-[#1e2a3f] border border-slate-700/50 rounded-lg shadow p-4">
              <div className="text-sm text-slate-400">Model Calls</div>
              <div className="text-2xl font-bold text-white">
                {agentRun.total_model_calls}
              </div>
            </div>
            <div className="bg-[#1e2a3f] border border-slate-700/50 rounded-lg shadow p-4">
              <div className="text-sm text-slate-400">Tokens</div>
              <div className="text-2xl font-bold text-white">
                {agentRun.total_tokens || 0}
              </div>
            </div>
            <div className="bg-[#1e2a3f] border border-slate-700/50 rounded-lg shadow p-4">
              <div className="text-sm text-slate-400">Total Latency</div>
              <div className="text-2xl font-bold text-white">
                {formatLatency(agentRun.total_latency_ms)}
              </div>
            </div>
          </div>
        </div>

        {/* Execution Timeline */}
        <div className="mb-8">
          <h2 className="text-xl font-bold text-white mb-4">
            Execution Trace
          </h2>
          <div className="space-y-4">
            {steps.map((step, index) => (
              <div
                key={step.id}
                className={`rounded-lg shadow-sm border-2 ${getStepColor(
                  step.status
                )} p-6 relative`}
              >
                {/* Step Number & Type */}
                <div className="flex items-start gap-4">
                  <div className="flex-shrink-0 text-2xl">
                    {getStepIcon(step.step_type)}
                  </div>
                  <div className="flex-1">
                    <div className="flex items-center gap-3 mb-2">
                      <span className="text-sm font-mono text-slate-500">
                        Step {step.step_index}
                      </span>
                      <span className="font-semibold text-white">
                        {step.step_type}
                      </span>
                      <span className="text-slate-300">{step.name}</span>
                      {step.latency_ms && (
                        <span className="text-sm text-slate-400 ml-auto">
                          {formatLatency(step.latency_ms)}
                        </span>
                      )}
                    </div>

                    {/* Input */}
                    {step.input && step.input !== "" && (
                      <div className="mb-3">
                        <div className="text-xs text-slate-400 uppercase tracking-wide mb-1">
                          Input
                        </div>
                        <div className="bg-[#1a2332] rounded p-3 text-sm text-slate-300 font-mono">
                          {step.input}
                        </div>
                      </div>
                    )}

                    {/* Output */}
                    {step.output && step.output !== "" && (
                      <div>
                        <div className="text-xs text-slate-400 uppercase tracking-wide mb-1">
                          Output
                        </div>
                        <div className="bg-[#1a2332] rounded p-3 text-sm text-slate-300">
                          {step.output}
                        </div>
                      </div>
                    )}

                    {/* Status Badge */}
                    <div className="mt-3">
                      <span
                        className={`text-xs px-2 py-1 rounded ${
                          step.status === "completed"
                            ? "bg-green-500/20 text-green-400"
                            : step.status === "failed"
                            ? "bg-red-500/20 text-red-400"
                            : "bg-slate-700 text-slate-300"
                        }`}
                      >
                        {step.status}
                      </span>
                    </div>
                  </div>
                </div>

                {/* Timeline connector */}
                {index < steps.length - 1 && (
                  <div className="absolute left-8 top-full w-0.5 h-4 bg-slate-700" />
                )}
              </div>
            ))}
          </div>
        </div>

        {/* Diagnosis Section */}
        {agentRun.status === "failed" && (
          <div className="mt-8">
            {!diagnosis ? (
              <button
                onClick={runDiagnosis}
                disabled={diagnosing}
                className="w-full bg-blue-600 hover:bg-blue-700 disabled:bg-slate-600 text-white font-medium py-3 px-4 rounded-lg transition-colors"
              >
                {diagnosing ? "Analyzing..." : "Run AI Diagnosis"}
              </button>
            ) : (
              <div className="bg-[#1e2a3f] rounded-lg shadow-lg border-2 border-blue-500/30 p-6">
                <h2 className="text-xl font-bold text-white mb-4 flex items-center gap-2">
                  <span>🔍</span>
                  AI Diagnosis
                </h2>

                {/* Root Cause */}
                <div className="mb-4">
                  <h3 className="text-sm font-semibold text-slate-400 uppercase tracking-wide mb-2">
                    Root Cause
                  </h3>
                  <p className="text-slate-200">{diagnosis.root_cause}</p>
                </div>

                {/* Evidence */}
                <div className="mb-4">
                  <h3 className="text-sm font-semibold text-slate-400 uppercase tracking-wide mb-2">
                    Evidence
                  </h3>
                  <ul className="space-y-1">
                    {diagnosis.evidence.map((item, idx) => (
                      <li key={idx} className="text-slate-300 flex gap-2">
                        <span className="text-slate-500">•</span>
                        <span>{item}</span>
                      </li>
                    ))}
                  </ul>
                </div>

                {/* Recommended Repairs */}
                <div className="mb-4">
                  <h3 className="text-sm font-semibold text-slate-400 uppercase tracking-wide mb-2">
                    Recommended Repairs
                  </h3>
                  <ul className="space-y-1">
                    {diagnosis.recommended_fixes.map((fix, idx) => (
                      <li key={idx} className="text-slate-300 flex gap-2">
                        <span className="text-green-400">✓</span>
                        <span>{fix}</span>
                      </li>
                    ))}
                  </ul>
                </div>

                {/* Prevention */}
                <div>
                  <h3 className="text-sm font-semibold text-slate-400 uppercase tracking-wide mb-2">
                    Prevention
                  </h3>
                  <p className="text-slate-200">{diagnosis.prevention}</p>
                </div>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
