const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

export interface Workload {
  id: number;
  name: string;
  type: string;
  status: string;
  failure_type?: string;
  created_at: string;
  started_at?: string;
  finished_at?: string;
  runtime_seconds?: number;
  exit_code?: number;
  wasted_gpu_seconds?: number;
  job_logs?: string;
  gpu_metrics?: string;
  failure_report?: string;
}

export interface Stats {
  total_workloads: number;
  failed_workloads: number;
  succeeded_workloads: number;
  wasted_gpu_seconds: number;
  failure_types: { [key: string]: number };
}

export interface DiagnosisReport {
  failure_type: string;
  confidence: number;
  root_cause: string;
  evidence: string[];
  recommended_fix: string;
  safe_to_retry: boolean;
  diagnosed_at: string;
}

export const api = {
  async getWorkloads(): Promise<Workload[]> {
    const res = await fetch(`${API_URL}/workloads`);
    if (!res.ok) throw new Error('Failed to fetch workloads');
    return res.json();
  },

  async getWorkload(id: string): Promise<Workload> {
    const res = await fetch(`${API_URL}/workloads/${id}`);
    if (!res.ok) throw new Error('Failed to fetch workload');
    return res.json();
  },

  async getStats(): Promise<Stats> {
    const res = await fetch(`${API_URL}/summary`);
    if (!res.ok) throw new Error('Failed to fetch stats');
    return res.json();
  },

  async runWorkload(template: string, type: string = 'ML_JOB'): Promise<{ workload_id: number }> {
    const res = await fetch(`${API_URL}/workloads/run`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ type, template }),
    });
    if (!res.ok) throw new Error('Failed to run workload');
    return res.json();
  },

  async diagnoseWorkload(id: string): Promise<DiagnosisReport> {
    const res = await fetch(`${API_URL}/workloads/${id}/diagnose`, {
      method: 'POST',
    });
    if (!res.ok) throw new Error('Failed to diagnose workload');
    return res.json();
  },
};
