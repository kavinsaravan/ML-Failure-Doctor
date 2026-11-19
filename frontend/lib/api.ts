const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
const READ_TIMEOUT_MS = 10_000;

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
  last_heartbeat_at?: string;
  heartbeat_stale?: boolean;
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

export interface MetricsSnapshot {
  timestamp: string;
  gpu_memory_used_mb: number;
  gpu_memory_total_mb: number;
  gpu_memory_percent: number;
  gpu_memory_peak_mb?: number | null;
  gpu_utilization_percent: number | null;
  temperature_celsius: number | null;
  source?: string;
}

export function getWorkloadStatus(workload: Pick<Workload, 'status' | 'heartbeat_stale'>) {
  return workload.status === 'running' && workload.heartbeat_stale ? 'stale' : workload.status;
}

export function getMemoryPeaks(metrics: MetricsSnapshot[]) {
  const sampledDevice = metrics.length ? metrics.reduce((peak, sample) =>
    sample.gpu_memory_percent > peak.gpu_memory_percent ? sample : peak) : null;
  const peaks = metrics.map(sample => sample.gpu_memory_peak_mb).filter(
    (peak): peak is number => typeof peak === 'number' && Number.isFinite(peak) && peak >= 0);
  return { sampledDevice, frameworkAllocatedMB: peaks.length ? Math.max(...peaks) : null };
}

export interface DiagnosisReport {
  requested_model?: string;
  served_model?: string;
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
  setFireworksCredentials(key: string, model = 'auto') { fireworksKey = key.trim(); fireworksModel = model.trim() || 'auto'; },
  clearFireworksCredentials() { fireworksKey = ''; fireworksModel = ''; },
  setAPIKey(key: string) { if (key !== apiKey) { fireworksKey = ''; fireworksModel = ''; } apiKey = key; },
  getAPIURL() { return API_URL; },
  async createWorkspace(name: string): Promise<{ api_key: string; owner_id: string; name: string }> {
    const res = await fetch(`${API_URL}/workspaces`, {
      method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ name }),
    });
    if (!res.ok) throw new Error(await res.text() || 'Workspace creation failed');
    return res.json();
  },
  async getAccessRequirements(): Promise<{ auth_required: boolean; workspace_creation_enabled?: boolean }> {
    const res = await fetch(`${API_URL}/health`, { headers: getHeaders(), cache: 'no-store', signal: AbortSignal.timeout(READ_TIMEOUT_MS) });
    await checkResponse(res, 'Backend unavailable');
    return res.json();
  },
  async checkAccess(key = apiKey): Promise<void> {
    const res = await fetch(`${API_URL}/session`, { headers: getHeaders(key), cache: 'no-store', signal: AbortSignal.timeout(READ_TIMEOUT_MS) });
    if (!res.ok) throw new Error(res.status === 401 ? 'Enter a valid API key.' : 'Backend unavailable');
  },
  async getWorkloads(): Promise<Workload[]> {
    const res = await fetch(`${API_URL}/workloads`, {
      headers: getHeaders(),
      signal: AbortSignal.timeout(READ_TIMEOUT_MS),
    });
    await checkResponse(res, 'Failed to fetch workloads');
    return res.json();
  },

  async getWorkload(id: string): Promise<Workload> {
    const res = await fetch(`${API_URL}/workloads/${id}`, {
      headers: getHeaders(),
      signal: AbortSignal.timeout(READ_TIMEOUT_MS),
    });
    await checkResponse(res, 'Failed to fetch workload');
    return res.json();
  },

  async getStats(): Promise<Stats> {
    const res = await fetch(`${API_URL}/summary`, {
      headers: getHeaders(),
      signal: AbortSignal.timeout(READ_TIMEOUT_MS),
    });
    await checkResponse(res, 'Failed to fetch stats');
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
