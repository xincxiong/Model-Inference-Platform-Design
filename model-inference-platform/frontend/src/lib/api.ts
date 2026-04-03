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
