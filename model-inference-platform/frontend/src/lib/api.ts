const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

function getApiKey(): string {
  if (typeof window === 'undefined') return '';
  return localStorage.getItem('api_key') || '';
}

export function setApiKey(key: string) {
  localStorage.setItem('api_key', key);
}

export function getStoredApiKey(): string {
  return getApiKey();
}

async function apiFetch(path: string, options: RequestInit = {}) {
  const res = await fetch(`${API_URL}${path}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${getApiKey()}`,
      ...options.headers,
    },
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: { message: res.statusText } }));
    throw new Error(err.error?.message || res.statusText);
  }
  return res.json();
}

export async function fetchModels() {
  return apiFetch('/api/models');
}

export async function fetchModelsList() {
  return apiFetch('/v1/models');
}

export async function createApiKey(name: string, serviceTier?: string) {
  return apiFetch('/api/api-keys', {
    method: 'POST',
    body: JSON.stringify({ name, service_tier: serviceTier || 'default' }),
  });
}

export async function listApiKeys() {
  return apiFetch('/api/api-keys');
}

export async function deleteApiKey(id: string) {
  return apiFetch(`/api/api-keys/${id}`, { method: 'DELETE' });
}

export async function fetchUsage() {
  return apiFetch('/api/usage');
}

export async function redeemPromo(code: string) {
  return apiFetch('/api/billing/redeem', {
    method: 'POST',
    body: JSON.stringify({ code }),
  });
}

export async function fetchDedicatedTemplates() {
  return apiFetch('/v0/dedicated_endpoints/templates');
}

export async function listDedicatedEndpoints() {
  return apiFetch('/v0/dedicated_endpoints');
}

export async function createDedicatedEndpoint(body: Record<string, unknown>) {
  return apiFetch('/v0/dedicated_endpoints', { method: 'POST', body: JSON.stringify(body) });
}

export async function patchDedicatedEndpoint(id: string, body: Record<string, unknown>) {
  return apiFetch(`/v0/dedicated_endpoints/${id}`, { method: 'PATCH', body: JSON.stringify(body) });
}

export async function deleteDedicatedEndpoint(id: string) {
  return apiFetch(`/v0/dedicated_endpoints/${id}`, { method: 'DELETE' });
}

export async function listFineTuningJobs() {
  return apiFetch('/v1/fine_tuning/jobs');
}

export async function createFineTuningJob(body: Record<string, unknown>) {
  return apiFetch('/v1/fine_tuning/jobs', { method: 'POST', body: JSON.stringify(body) });
}

export async function getFineTuningJob(id: string) {
  return apiFetch(`/v1/fine_tuning/jobs/${id}`);
}

export async function cancelFineTuningJob(id: string) {
  return apiFetch(`/v1/fine_tuning/jobs/${id}/cancel`, { method: 'POST' });
}

// ── Files API ──────────────────────────────────────────────────────────────

export async function uploadFile(file: File, purpose = 'batch') {
  const form = new FormData();
  form.append('file', file);
  form.append('purpose', purpose);
  const res = await fetch(`${API_URL}/v1/files`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${getApiKey()}` },
    body: form,
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: { message: res.statusText } }));
    throw new Error(err.error?.message || res.statusText);
  }
  return res.json();
}

export async function listFiles(purpose?: string) {
  const q = purpose ? `?purpose=${purpose}` : '';
  return apiFetch(`/v1/files${q}`);
}

export async function getFile(id: string) {
  return apiFetch(`/v1/files/${id}`);
}

export async function deleteFile(id: string) {
  return apiFetch(`/v1/files/${id}`, { method: 'DELETE' });
}

// ── Batch API ──────────────────────────────────────────────────────────────

export async function createBatch(body: { input_file_id: string; endpoint: string; completion_window?: string; metadata?: Record<string, string> }) {
  return apiFetch('/v1/batches', { method: 'POST', body: JSON.stringify(body) });
}

export async function listBatches() {
  return apiFetch('/v1/batches');
}

export async function getBatch(id: string) {
  return apiFetch(`/v1/batches/${id}`);
}

export async function cancelBatch(id: string) {
  return apiFetch(`/v1/batches/${id}/cancel`, { method: 'POST' });
}

// ── Datasets API ───────────────────────────────────────────────────────────

export async function createDataset(body: { name: string; description?: string; file_id?: string; metadata?: Record<string, string> }) {
  return apiFetch('/v1/datasets', { method: 'POST', body: JSON.stringify(body) });
}

export async function listDatasets() {
  return apiFetch('/v1/datasets');
}

export async function getDataset(id: string) {
  return apiFetch(`/v1/datasets/${id}`);
}

export async function deleteDataset(id: string) {
  return apiFetch(`/v1/datasets/${id}`, { method: 'DELETE' });
}

export async function getDatasetContent(id: string, page = 1, limit = 20) {
  return apiFetch(`/v1/datasets/${id}/content?page=${page}&limit=${limit}`);
}

export function getDatasetExportUrl(id: string, format: 'jsonl' | 'csv' = 'jsonl') {
  return `${API_URL}/v1/datasets/${id}/export?format=${format}`;
}

export async function queryDataset(id: string, filter?: string, limit = 50) {
  const q = new URLSearchParams({ limit: String(limit) });
  if (filter) q.set('filter', filter);
  return apiFetch(`/v1/datasets/${id}/query?${q.toString()}`);
}

// ── Deployments API ────────────────────────────────────────────────────────

export async function listDeployments() {
  return apiFetch('/v1/deployments');
}

export async function getDeployment(id: string) {
  return apiFetch(`/v1/deployments/${id}`);
}

export async function createDeployment(body: {
  name: string;
  model_name: string;
  billing_mode: 'token' | 'tpu' | 'unit';
  min_replicas?: number;
  max_replicas?: number;
  description?: string;
}) {
  return apiFetch('/v1/deployments', { method: 'POST', body: JSON.stringify(body) });
}

export async function patchDeployment(id: string, body: {
  name?: string;
  description?: string;
  min_replicas?: number;
  max_replicas?: number;
  status?: string;
}) {
  return apiFetch(`/v1/deployments/${id}`, { method: 'PATCH', body: JSON.stringify(body) });
}

export async function deleteDeployment(id: string) {
  return apiFetch(`/v1/deployments/${id}`, { method: 'DELETE' });
}

// ── Members API ────────────────────────────────────────────────────────────

export async function listMembers() {
  return apiFetch('/api/members');
}

export async function inviteMember(email: string, role: 'owner' | 'admin' | 'member' | 'viewer' = 'member') {
  return apiFetch('/api/members', { method: 'POST', body: JSON.stringify({ email, role }) });
}

export async function patchMemberRole(id: string, role: string) {
  return apiFetch(`/api/members/${id}`, { method: 'PATCH', body: JSON.stringify({ role }) });
}

export async function removeMember(id: string) {
  return apiFetch(`/api/members/${id}`, { method: 'DELETE' });
}

export function streamChat(
  model: string,
  messages: { role: string; content: string }[],
  params: { temperature?: number; max_tokens?: number; top_p?: number },
  onChunk: (text: string) => void,
  onDone: () => void,
  onError: (err: string) => void,
) {
  const controller = new AbortController();

  fetch(`${API_URL}/v1/chat/completions`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${getApiKey()}`,
    },
    body: JSON.stringify({
      model,
      messages,
      stream: true,
      ...params,
    }),
    signal: controller.signal,
  })
    .then(async (res) => {
      if (!res.ok) {
        const err = await res.json().catch(() => ({ error: { message: res.statusText } }));
        onError(err.error?.message || res.statusText);
        return;
      }

      const reader = res.body?.getReader();
      if (!reader) return;

      const decoder = new TextDecoder();
      let buffer = '';

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split('\n');
        buffer = lines.pop() || '';

        for (const line of lines) {
          const trimmed = line.trim();
          if (!trimmed || !trimmed.startsWith('data: ')) continue;
          const data = trimmed.slice(6);
          if (data === '[DONE]') {
            onDone();
            return;
          }
          try {
            const chunk = JSON.parse(data);
            const content = chunk.choices?.[0]?.delta?.content || '';
            if (content) onChunk(content);
          } catch {}
        }
      }
      onDone();
    })
    .catch((err) => {
      if (err.name !== 'AbortError') onError(err.message);
    });

  return controller;
}
