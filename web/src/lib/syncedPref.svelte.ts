// Synced personal preferences: one reactive value per server pref key.
//
// A pref loads from the /api/sidebar payload (its `prefs` object, or another
// field via `read`), applies changes at once (optimistic), saves itself via
// `PUT /api/prefs/<key>` (debounced, or immediately with debounce 0) and on
// failure reports the error and optionally rolls back to the last value the
// server confirmed. While a change is pending, server payloads do not
// overwrite it. Every pref registers itself; `syncPrefs(payload)` (called when
// the sidebar data arrives) updates all of them, and a pref created later
// takes over the last payload immediately.
//
// Transport and timers are injectable so the timing is unit tested
// (syncedPref.test.ts).
import { api, errorText } from './api';
import { toast } from './toasts.svelte';

/** What the server sends with the sidebar data (see SidebarData in types.ts). */
export interface PrefSource {
  prefs?: Record<string, unknown> | null;
  board_hidden?: number[] | null;
}

export interface PrefTransport {
  put(url: string, body: unknown): Promise<unknown>;
}

export interface PrefTimers {
  set(fn: () => void, ms: number): unknown;
  clear(handle: unknown): void;
}

/** How a failed save is reported: toast (default), silently, or by rejecting set(). */
export type PrefErrorMode = 'toast' | 'silent' | 'reject';

export interface SyncedPrefOptions<T> {
  /** Validates the raw server value (missing/invalid → a default). */
  parse: (raw: unknown) => T;
  /** Raw value in the payload (default: payload.prefs[key]). */
  read?: (src: PrefSource) => unknown;
  /** Delay in ms before saving; rapid changes are saved once. 0 = save immediately. Default 400. */
  debounce?: number;
  /** Endpoint below /api (default `/prefs/<key>`). */
  url?: string;
  /** Request body for a value (default: the value itself). */
  body?: (v: T) => unknown;
  /** Restore the last confirmed value when saving fails (default false). */
  rollback?: boolean;
  onError?: PrefErrorMode;
  /** Called after a server value was taken over (e.g. apply it elsewhere). */
  onSync?: (v: T) => void;
  /** Called after every local change (e.g. mirror into localStorage). */
  onChange?: (v: T) => void;
  transport?: PrefTransport;
  timers?: PrefTimers;
}

// defaults resolve the globals at call time (fake timers in tests)
const defaultTimers: PrefTimers = {
  set: (fn, ms) => setTimeout(fn, ms),
  clear: (h) => clearTimeout(h as ReturnType<typeof setTimeout>)
};
const defaultTransport: PrefTransport = { put: (url, body) => api.put(url, body) };

const snapshot = <T>(v: T): T => $state.snapshot(v) as T;

export class SyncedPref<T> {
  readonly key: string;
  /** Current value (reactive). Assigning it changes it locally without saving. */
  value = $state() as T;
  /** A server value has been taken over at least once. */
  synced = $state(false);
  /** A change waits for its save or is being saved. */
  pending = $state(false);

  #o: SyncedPrefOptions<T>;
  #confirmed: T; // last value known to be on the server
  #timer: unknown = undefined;
  #inflight = 0;
  #version = 0; // bumped by every local change
  #waiters: { resolve: () => void; reject: (e: unknown) => void }[] = [];

  constructor(key: string, o: SyncedPrefOptions<T>) {
    this.key = key;
    this.#o = o;
    this.value = o.parse(undefined);
    this.#confirmed = snapshot(this.value);
  }

  /** Takes over the server value unless a local change is pending. */
  sync(src: PrefSource) {
    if (this.pending) return;
    const raw = this.#o.read ? this.#o.read(src) : src.prefs?.[this.key];
    this.value = this.#o.parse(raw);
    this.#confirmed = snapshot(this.value);
    this.synced = true;
    this.#o.onSync?.(this.value);
  }

  /** Changes the value: shown at once, saved debounced (or immediately). Resolves after the save. */
  set(next: T | ((prev: T) => T)): Promise<void> {
    this.value = typeof next === 'function' ? (next as (prev: T) => T)(this.value) : next;
    this.#version++;
    this.pending = true;
    this.#o.onChange?.(this.value);
    const done = new Promise<void>((resolve, reject) => this.#waiters.push({ resolve, reject }));
    const timers = this.#o.timers ?? defaultTimers;
    if (this.#timer !== undefined) timers.clear(this.#timer);
    this.#timer = undefined;
    const delay = this.#o.debounce ?? 400;
    if (delay > 0) this.#timer = timers.set(() => void this.#flush(), delay);
    else void this.#flush();
    return done;
  }

  async #flush() {
    this.#timer = undefined;
    const waiters = this.#waiters;
    this.#waiters = [];
    const version = this.#version;
    const value = snapshot(this.value);
    this.#inflight++;
    let failure: { e: unknown } | null = null;
    try {
      await (this.#o.transport ?? defaultTransport).put(this.#o.url ?? `/prefs/${this.key}`, this.#o.body ? this.#o.body(value) : value);
      this.#confirmed = value;
    } catch (e) {
      failure = { e };
      // roll back only if nothing changed in the meantime
      if (this.#o.rollback && version === this.#version) this.value = snapshot(this.#confirmed);
    } finally {
      this.#inflight--;
      this.pending = this.#timer !== undefined || this.#inflight > 0;
    }
    const mode = this.#o.onError ?? 'toast';
    if (failure && mode === 'toast') toast(errorText(failure.e));
    for (const w of waiters) {
      if (failure && mode === 'reject') w.reject(failure.e);
      else w.resolve();
    }
  }
}

// ---- Registry ----------------------------------------------------------------------

const registry = new Set<SyncedPref<unknown>>();
let lastSource: PrefSource | null = null;

/** Creates and registers a synced pref (takes over the last payload if one arrived already). */
export function syncedPref<T>(key: string, o: SyncedPrefOptions<T>): SyncedPref<T> {
  const p = new SyncedPref(key, o);
  registry.add(p as SyncedPref<unknown>);
  if (lastSource) p.sync(lastSource);
  return p;
}

/** The sidebar data arrived: every registered pref takes over its server value. */
export function syncPrefs(src: PrefSource) {
  lastSource = src;
  for (const p of registry) p.sync(src);
}

/** Tests: forget the registry and the last payload. */
export function resetPrefRegistry() {
  registry.clear();
  lastSource = null;
}
