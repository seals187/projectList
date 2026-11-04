import { goto } from '$app/navigation';
import { saveState, session } from './state.svelte';
import { t, tMaybe } from './i18n.svelte';

let csrfToken = '';

export function setCsrf(token: string) {
  csrfToken = token;
}

/** API error: code is a language-neutral error code (e.g. "not_found"). */
export class ApiError extends Error {
  constructor(
    public status: number,
    public code: string,
    public fields: Record<string, string> = {}
  ) {
    super(code);
  }
}

const silentWrites = new Set(['/auth/login', '/auth/logout']);

async function request<T>(method: string, url: string, body?: unknown): Promise<T> {
  const headers: Record<string, string> = {};
  if (body !== undefined) headers['Content-Type'] = 'application/json';
  if (method !== 'GET') headers['X-CSRF-Token'] = csrfToken;

  // "saved" indicator: count all writing requests.
  const tracked = method !== 'GET' && !silentWrites.has(url);
  if (tracked) saveState.pending++;

  try {
    let res: Response;
    try {
      res = await fetch('/api' + url, {
        method,
        headers,
        body: body !== undefined ? JSON.stringify(body) : undefined,
        credentials: 'same-origin'
      });
    } catch {
      throw new ApiError(0, 'network');
    }

    if (res.status === 401 && url !== '/auth/login' && url !== '/auth/me') {
      session.user = null;
      goto('/login');
    }
    if (!res.ok) {
      const data = await res.json().catch(() => ({}));
      throw new ApiError(res.status, data.error ?? 'generic', data.fields ?? {});
    }
    if (tracked) {
      saveState.error = false;
      saveState.saved = true;
    }
    if (res.status === 204) return undefined as T;
    return (await res.json()) as T;
  } catch (e) {
    if (tracked) saveState.error = true;
    throw e;
  } finally {
    if (tracked) saveState.pending--;
  }
}

export const api = {
  get: <T>(url: string) => request<T>('GET', url),
  post: <T>(url: string, body: unknown = {}) => request<T>('POST', url, body),
  put: <T>(url: string, body: unknown) => request<T>('PUT', url, body),
  patch: <T>(url: string, body: unknown) => request<T>('PATCH', url, body),
  del: (url: string) => request<void>('DELETE', url)
};

/** "min:8" → "at least 8 characters" etc. */
function fieldMessage(code: string) {
  const [name, a, b] = code.split(':');
  if (name === 'min' || name === 'max') return t(`fields.${name}`, { n: a });
  if (name === 'length') return t('fields.length', { min: a, max: b });
  return tMaybe(`fields.${name}`) ?? code;
}

/** Localized error message including field errors. */
export function errorText(e: unknown): string {
  if (!(e instanceof ApiError)) return t('errors.generic');
  const base = tMaybe(`errors.${e.code}`) ?? t('errors.generic');
  const fields = Object.entries(e.fields);
  if (!fields.length) return base;
  return `${base}: ${fields.map(([k, v]) => `${k} – ${fieldMessage(v)}`).join(', ')}`;
}
