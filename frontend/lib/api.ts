const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

let apiKey = '';
let fireworksKey = '';
let fireworksModel = '';

// The key stays in memory; it is never embedded in the build or stored on disk.
const getHeaders = (key = apiKey) => ({
  'ngrok-skip-browser-warning': 'true',
  'Content-Type': 'application/json',
  ...(key ? { Authorization: `Bearer ${key}` } : {}),
});

async function checkResponse(res: Response, message: string) {
  if (res.ok) return;
  if (res.status === 401 && typeof window !== 'undefined') {
    apiKey = '';
    fireworksKey = ''; fireworksModel = '';
    window.dispatchEvent(new Event('crashlens-auth-required'));
  }
  const detail = await res.text();
  throw new Error(detail || message);
}

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
  gpu_platform?: string;
}

export interface DiagnosisReport {
  ai_unavailable_reason?: string;
  source: "ai" | "rules";
  confidence_basis?: string;
  prevention?: string;
  failure_type: string;
  confidence: number;
  root_cause: string;
  evidence: string[];
  recommended_fix: string;
  safe_to_retry: boolean;
  diagnosed_at: string;
}

export const api = {
  setFireworksCredentials(key: string, model: string) { fireworksKey = key.trim(); fireworksModel = model.trim(); },
  clearFireworksCredentials() { fireworksKey = ''; fireworksModel = ''; },
  hasAPIKey() { return apiKey.length > 0; },
  setAPIKey(key: string) { if (key !== apiKey) { fireworksKey = ''; fireworksModel = ''; } apiKey = key; },
  async getAccessRequirements(): Promise<{ auth_required: boolean }> {
    const res = await fetch(`${API_URL}/health`, { headers: getHeaders(), cache: 'no-store' });
    await checkResponse(res, 'Backend unavailable');
    return res.json();
  },
  async checkAccess(key = apiKey): Promise<void> {
    const res = await fetch(`${API_URL}/session`, { headers: getHeaders(key), cache: 'no-store' });
    if (!res.ok) throw new Error(res.status === 401 ? 'Enter a valid API key.' : 'Backend unavailable');
  },
  async getWorkloads(): Promise<Workload[]> {
    const res = await fetch(`${API_URL}/workloads`, {
      headers: getHeaders(),
    });
    await checkResponse(res, 'Failed to fetch workloads');
    return res.json();
  },

  async getWorkload(id: string): Promise<Workload> {
    const res = await fetch(`${API_URL}/workloads/${id}`, {
      headers: getHeaders(),
    });
    await checkResponse(res, 'Failed to fetch workload');
    return res.json();
  },

  async getStats(): Promise<Stats> {
    const res = await fetch(`${API_URL}/summary`, {
      headers: getHeaders(),
    });
    await checkResponse(res, 'Failed to fetch stats');
    return res.json();
  },

  async runWorkload(template: string, type: string = 'ML_JOB'): Promise<{ workload_id: number }> {
    const res = await fetch(`${API_URL}/workloads/run`, {
      method: 'POST',
      headers: getHeaders(),
      body: JSON.stringify({ type, template }),
    });
    await checkResponse(res, 'Failed to run workload');
    return res.json();
  },

  async diagnoseWorkload(id: string, refresh = false): Promise<DiagnosisReport> {
    const res = await fetch(`${API_URL}/workloads/${id}/diagnose${refresh ? "?refresh=true" : ""}`, {
      method: 'POST',
      headers: { ...getHeaders(), ...(fireworksKey ? {
        'X-Fireworks-API-Key': fireworksKey, 'X-Fireworks-Model': fireworksModel,
      } : {}) },
    });
    await checkResponse(res, 'Failed to diagnose workload');
    return res.json();
  },

  async clearAllWorkloads(): Promise<void> {
    const res = await fetch(`${API_URL}/workloads/clear`, {
      method: 'DELETE',
      headers: getHeaders(),
    });
    await checkResponse(res, 'Failed to clear workloads');
  },
};
