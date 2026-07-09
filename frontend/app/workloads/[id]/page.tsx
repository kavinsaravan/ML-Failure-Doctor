'use client';

import { useEffect, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
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
} from 'lucide-react';

export default function WorkloadDetail() {
  const params = useParams();
  const router = useRouter();
  const [workload, setWorkload] = useState<Workload | null>(null);
  const [diagnosis, setDiagnosis] = useState<DiagnosisReport | null>(null);
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
      loadWorkload(); // Reload to get updated failure_report
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
                {diagnosing ? 'Diagnosing...' : 'Run Diagnosis'}
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

        {/* Diagnosis Report */}
        {diagnosis && (
          <div className="bg-slate-800 border border-slate-700 rounded-lg p-6 mb-8">
            <h2 className="text-2xl font-bold text-white mb-4">AI Diagnosis Report</h2>

            <div className="grid gap-4">
              {/* Failure Type & Confidence */}
              <div className="flex items-center justify-between pb-4 border-b border-slate-700">
                <div>
                  <div className="text-sm text-slate-400 mb-1">Failure Type</div>
                  <div className="text-lg font-semibold text-white">
                    {formatFailureType(diagnosis.failure_type)}
                  </div>
                </div>
                <div className="text-right">
                  <div className="text-sm text-slate-400 mb-1">Confidence</div>
                  <div className="text-lg font-semibold text-white">
                    {(diagnosis.confidence * 100).toFixed(0)}%
                  </div>
                </div>
              </div>

              {/* Root Cause */}
              <div>
                <div className="text-sm font-medium text-slate-400 mb-2">Root Cause</div>
                <div className="text-white bg-slate-900 rounded p-4">{diagnosis.root_cause}</div>
              </div>

              {/* Evidence */}
              {diagnosis.evidence && diagnosis.evidence.length > 0 && (
                <div>
                  <div className="text-sm font-medium text-slate-400 mb-2">Evidence</div>
                  <div className="bg-slate-900 rounded p-4 space-y-2">
                    {diagnosis.evidence.map((line, idx) => (
                      <div key={idx} className="text-red-300 text-sm font-mono">
                        {line}
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {/* Recommended Fix */}
              <div>
                <div className="text-sm font-medium text-slate-400 mb-2">Recommended Fix</div>
                <div className="text-white bg-slate-900 rounded p-4 whitespace-pre-line">
                  {diagnosis.recommended_fix}
                </div>
              </div>

              {/* Safe to Retry */}
              <div className="flex items-center gap-2 pt-4 border-t border-slate-700">
                {diagnosis.safe_to_retry ? (
                  <>
                    <CheckCircle className="w-5 h-5 text-green-400" />
                    <span className="text-green-400 font-medium">Safe to retry after fixes</span>
                  </>
                ) : (
                  <>
                    <AlertTriangle className="w-5 h-5 text-orange-400" />
                    <span className="text-orange-400 font-medium">
                      Configuration changes required before retry
                    </span>
                  </>
                )}
              </div>
            </div>
          </div>
        )}

        {/* Logs */}
        {workload.job_logs && (
          <div className="bg-slate-800 border border-slate-700 rounded-lg overflow-hidden">
            <div className="px-6 py-4 border-b border-slate-700">
              <h2 className="text-xl font-semibold text-white">Job Logs</h2>
            </div>
            <div className="p-6 bg-black overflow-x-auto max-h-96">
              <pre className="text-sm text-green-400 font-mono">{workload.job_logs}</pre>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
