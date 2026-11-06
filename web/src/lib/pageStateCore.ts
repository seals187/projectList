// Page header state (view, filters, sorting, expand/collapse) – pure helpers:
// serialization for localStorage and the URL query. The reactive wrapper is
// pageState.svelte.ts. Kept free of SvelteKit imports so it can be unit tested.

/** How a field appears in the URL: not at all, as text, as a list of numbers / texts. */
export type UrlKind = 'str' | 'ids' | 'strs';
export interface FieldSpec {
  def: unknown; // default value (also decides the type when reading stored data)
  url?: UrlKind; // undefined = only localStorage (e.g. expanded groups)
}
export type Spec = Record<string, FieldSpec>;
export type Values<S extends Spec> = { [K in keyof S]: S[K]['def'] };

const same = (a: unknown, b: unknown) => JSON.stringify(a) === JSON.stringify(b);

export function defaults<S extends Spec>(spec: S): Values<S> {
  return Object.fromEntries(Object.entries(spec).map(([k, f]) => [k, structuredClone(f.def)])) as Values<S>;
}

/** Same kind of value as the default (arrays stay arrays, null allowed for "all"). */
function fits(v: unknown, def: unknown): boolean {
  if (v === null) return def === null || Array.isArray(def);
  if (Array.isArray(def) || def === null) return Array.isArray(v) && v.every((x) => typeof x === 'number' || typeof x === 'string');
  return typeof v === typeof def;
}

/** Stored values (localStorage JSON); unknown keys and wrong types are ignored. */
export function fromStored<S extends Spec>(raw: string | null, spec: S): Partial<Values<S>> {
  let o: Record<string, unknown>;
  try {
    o = JSON.parse(raw ?? '{}');
  } catch {
    return {};
  }
  if (!o || typeof o !== 'object') return {};
  const out: Record<string, unknown> = {};
  for (const [k, f] of Object.entries(spec)) if (k in o && fits(o[k], f.def)) out[k] = o[k];
  return out as Partial<Values<S>>;
}

/** URL query → values of the URL fields that are present. */
export function fromQuery<S extends Spec>(search: URLSearchParams | string, spec: S): Partial<Values<S>> {
  const q = typeof search === 'string' ? new URLSearchParams(search) : search;
  const out: Record<string, unknown> = {};
  for (const [k, f] of Object.entries(spec)) {
    if (!f.url || !q.has(k)) continue;
    const raw = q.get(k) ?? '';
    if (f.url === 'str') out[k] = raw;
    else {
      const parts = raw === '' ? [] : raw.split(',');
      out[k] = f.url === 'ids' ? parts.map(Number).filter(Number.isFinite) : parts;
    }
  }
  return out as Partial<Values<S>>;
}

/** Values → URL query (without "?"); defaults are left out, keys sorted. */
export function toQuery<S extends Spec>(values: Values<S>, spec: S): string {
  const q = new URLSearchParams();
  for (const k of Object.keys(spec).sort()) {
    const f = spec[k];
    const v = values[k];
    if (!f.url || same(v, f.def) || v === null || v === undefined) continue;
    q.set(k, Array.isArray(v) ? v.join(',') : String(v));
  }
  return q.toString();
}

/** Start values: defaults, then the stored state, then the URL (a shared link wins). */
export function initial<S extends Spec>(spec: S, stored: string | null, search: string): Values<S> {
  return { ...defaults(spec), ...fromStored(stored, spec), ...fromQuery(search, spec) };
}

/** Normalized query for comparisons (sorted keys). */
export function normQuery(search: string): string {
  const q = new URLSearchParams(search.startsWith('?') ? search.slice(1) : search);
  q.sort();
  return q.toString();
}

/**
 * Expand/collapse toggle: true when everything is collapsed (button then offers
 * "Expand all"); mixed or all open → false ("Collapse all").
 */
export function allCollapsed(openFlags: boolean[]): boolean {
  return openFlags.length > 0 && openFlags.every((o) => !o);
}
