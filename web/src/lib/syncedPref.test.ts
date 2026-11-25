import { beforeEach, describe, expect, it, vi } from 'vitest';

// default transport and error text are not used (injected below), but must load
vi.mock('./api', () => ({ api: { put: vi.fn() }, errorText: (e: unknown) => `err:${(e as Error).message}` }));
const toasts = vi.hoisted(() => ({ list: [] as string[] }));
vi.mock('./toasts.svelte', () => ({ toast: (text: string) => toasts.list.push(text) }));

import { resetPrefRegistry, syncedPref, syncPrefs, type PrefTimers, type PrefTransport } from './syncedPref.svelte';

// manual clock: timers fire only when advanced
function fakeTimers() {
  let now = 0;
  let next = 1;
  const pending = new Map<number, { at: number; fn: () => void }>();
  const timers: PrefTimers = {
    set: (fn, ms) => {
      const id = next++;
      pending.set(id, { at: now + ms, fn });
      return id;
    },
    clear: (h) => void pending.delete(h as number)
  };
  async function advance(ms: number) {
    now += ms;
    for (const [id, t] of [...pending].sort((a, b) => a[1].at - b[1].at)) {
      if (t.at > now) continue;
      pending.delete(id);
      t.fn();
    }
    await flush();
  }
  return { timers, advance, count: () => pending.size };
}

// in-memory server: records PUTs, can fail or be held open
function fakeServer() {
  const calls: { url: string; body: unknown }[] = [];
  let fail = false;
  let holding = false;
  let held: { res: () => void; rej: (e: Error) => void } | null = null;
  const transport: PrefTransport = {
    put: (url, body) => {
      calls.push({ url, body: JSON.parse(JSON.stringify(body)) });
      if (holding) {
        holding = false;
        return new Promise<void>((res, rej) => (held = { res, rej }));
      }
      return fail ? Promise.reject(new Error('offline')) : Promise.resolve();
    }
  };
  return {
    transport,
    calls,
    fail: (v: boolean) => (fail = v),
    holdNext: () => (holding = true),
    release: (ok = true) => {
      const h = held;
      held = null;
      if (ok) h?.res();
      else h?.rej(new Error('offline'));
    }
  };
}

const flush = async () => {
  for (let i = 0; i < 5; i++) await Promise.resolve();
};

const ids = (v: unknown) => (Array.isArray(v) ? v.filter((x) => Number.isInteger(x)) : []);

describe('syncedPref', () => {
  beforeEach(() => {
    resetPrefRegistry();
    toasts.list = [];
  });

  it('starts with the parsed default and takes over the server value', () => {
    const p = syncedPref('colors', { parse: ids, transport: fakeServer().transport, timers: fakeTimers().timers });
    expect(p.value).toEqual([]);
    expect(p.synced).toBe(false);
    syncPrefs({ prefs: { colors: [1, 'x', 2] } });
    expect(p.value).toEqual([1, 2]);
    expect(p.synced).toBe(true);
  });

  it('applies at once and saves debounced: rapid changes = one request with the last value', async () => {
    const srv = fakeServer();
    const clock = fakeTimers();
    const p = syncedPref('overview', { parse: ids, debounce: 400, transport: srv.transport, timers: clock.timers });
    p.set([1]);
    p.set((v) => [...v, 2]);
    expect(p.value).toEqual([1, 2]);
    expect(p.pending).toBe(true);
    await clock.advance(399);
    expect(srv.calls).toEqual([]);
    await clock.advance(1);
    expect(srv.calls).toEqual([{ url: '/prefs/overview', body: [1, 2] }]);
    expect(p.pending).toBe(false);
  });

  it('custom endpoint, body and payload field (board-hidden)', async () => {
    const srv = fakeServer();
    const clock = fakeTimers();
    const p = syncedPref('board_hidden', { parse: ids, read: (s) => s.board_hidden, url: '/prefs/board-hidden', body: (v) => ({ customer_ids: v }), transport: srv.transport, timers: clock.timers });
    syncPrefs({ prefs: {}, board_hidden: [4] });
    expect(p.value).toEqual([4]);
    p.set([4, 5]);
    await clock.advance(400);
    expect(srv.calls).toEqual([{ url: '/prefs/board-hidden', body: { customer_ids: [4, 5] } }]);
  });

  it('does not overwrite a pending change with a server payload (waiting or in flight)', async () => {
    const srv = fakeServer();
    const clock = fakeTimers();
    const p = syncedPref('k', { parse: ids, transport: srv.transport, timers: clock.timers });
    p.set([1]);
    syncPrefs({ prefs: { k: [9] } });
    expect(p.value).toEqual([1]);
    srv.holdNext();
    await clock.advance(400);
    syncPrefs({ prefs: { k: [9] } });
    expect(p.value).toEqual([1]); // still saving
    srv.release();
    await flush();
    syncPrefs({ prefs: { k: [9] } });
    expect(p.value).toEqual([9]); // saved: server wins again
  });

  it('immediate save with rollback to the last confirmed value and a toast', async () => {
    const srv = fakeServer();
    const p = syncedPref('saved_views', { parse: ids, debounce: 0, rollback: true, transport: srv.transport });
    syncPrefs({ prefs: { saved_views: [1] } });
    await p.set([1, 2]);
    expect(srv.calls).toHaveLength(1);
    srv.fail(true);
    await p.set([1, 2, 3]);
    expect(p.value).toEqual([1, 2]); // back to what the server has
    expect(toasts.list).toEqual(['err:offline']);
    expect(p.pending).toBe(false);
  });

  it('no rollback when a newer change is already waiting', async () => {
    const srv = fakeServer();
    const clock = fakeTimers();
    const p = syncedPref('k', { parse: ids, debounce: 100, rollback: true, transport: srv.transport, timers: clock.timers });
    p.set([1]);
    await clock.advance(100); // [1] saved = confirmed
    srv.holdNext();
    p.set([2]);
    await clock.advance(100); // request [2] in flight
    p.set([3]); // newer change waits
    srv.release(false); // request [2] fails: [3] is kept
    await flush();
    expect(p.value).toEqual([3]);
    expect(p.pending).toBe(true);
    srv.fail(true);
    await clock.advance(100); // request [3] fails, nothing newer → back to the confirmed [1]
    expect(p.value).toEqual([1]);
    expect(srv.calls.map((c) => c.body)).toEqual([[1], [2], [3]]);
  });

  it('without rollback the value stays; silent mode shows no toast', async () => {
    const srv = fakeServer();
    const clock = fakeTimers();
    const p = syncedPref('display', { parse: ids, onError: 'silent', transport: srv.transport, timers: clock.timers });
    srv.fail(true);
    p.set([3]);
    await clock.advance(400);
    expect(p.value).toEqual([3]);
    expect(toasts.list).toEqual([]);
  });

  it('reject mode: set() rejects after rolling back', async () => {
    const srv = fakeServer();
    const p = syncedPref<string>('background', { parse: (v) => (typeof v === 'string' ? v : ''), debounce: 0, rollback: true, onError: 'reject', transport: srv.transport });
    syncPrefs({ prefs: { background: 'g1' } });
    srv.fail(true);
    await expect(p.set('g2')).rejects.toThrow('offline');
    expect(p.value).toBe('g1');
    expect(toasts.list).toEqual([]);
  });

  it('hooks: onChange after local changes, onSync after server values', async () => {
    const changed: unknown[] = [];
    const synced: unknown[] = [];
    const p = syncedPref('ui', { parse: ids, onChange: (v) => changed.push([...v]), onSync: (v) => synced.push([...v]), transport: fakeServer().transport, timers: fakeTimers().timers });
    p.set([1]);
    expect(changed).toEqual([[1]]);
    expect(synced).toEqual([]);
    const q = syncedPref('ui2', { parse: ids, onSync: (v) => synced.push([...v]) });
    syncPrefs({ prefs: { ui2: [5] } });
    expect(synced).toEqual([[5]]); // p is pending and skipped
    expect(q.value).toEqual([5]);
  });

  it('a pref created after the payload arrived takes it over at once', () => {
    syncPrefs({ prefs: { late: [7] } });
    const p = syncedPref('late', { parse: ids, transport: fakeServer().transport, timers: fakeTimers().timers });
    expect(p.value).toEqual([7]);
    expect(p.synced).toBe(true);
  });

  it('every registered pref syncs from one payload', () => {
    const a = syncedPref('a', { parse: ids });
    const b = syncedPref('b', { parse: ids });
    syncPrefs({ prefs: { a: [1], b: [2] } });
    expect([a.value, b.value]).toEqual([[1], [2]]);
  });
});
