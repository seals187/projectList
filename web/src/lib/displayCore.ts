// Display options of the page headers ("Display" popover): pure helpers.
// Stored per user as { "<page>.<view>": { "<option>": boolean } }.

export type DisplayValues = Record<string, boolean>;
export type DisplayStore = Record<string, DisplayValues>;

/** Options of one page/view: defaults overridden by stored booleans (unknown keys ignored). */
export function resolveDisplay(defaults: DisplayValues, stored: unknown): DisplayValues {
  const s = stored && typeof stored === 'object' ? (stored as Record<string, unknown>) : {};
  return Object.fromEntries(Object.entries(defaults).map(([k, d]) => [k, typeof s[k] === 'boolean' ? (s[k] as boolean) : d]));
}

/** New store with one option changed (immutable). */
export function withOption(store: DisplayStore, page: string, key: string, value: boolean): DisplayStore {
  return { ...store, [page]: { ...(store[page] ?? {}), [key]: value } };
}

/** Parses stored JSON (server or localStorage); anything invalid → empty. */
export function parseStore(v: unknown): DisplayStore {
  const o = typeof v === 'string' ? safeJson(v) : v;
  if (!o || typeof o !== 'object' || Array.isArray(o)) return {};
  const out: DisplayStore = {};
  for (const [page, vals] of Object.entries(o as Record<string, unknown>)) {
    if (!vals || typeof vals !== 'object') continue;
    const clean = Object.fromEntries(Object.entries(vals as Record<string, unknown>).filter(([, b]) => typeof b === 'boolean')) as DisplayValues;
    out[page] = clean;
  }
  return out;
}

function safeJson(s: string): unknown {
  try {
    return JSON.parse(s);
  } catch {
    return null;
  }
}
