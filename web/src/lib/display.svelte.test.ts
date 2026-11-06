import { beforeEach, describe, expect, it, vi } from 'vitest';
import { parseStore, resolveDisplay, withOption } from './displayCore';

// server API and the synced sidebar prefs, simulated
const srv = vi.hoisted(() => ({ put: vi.fn(), fail: false, prefs: {} as Record<string, unknown> }));
vi.mock('./api', () => ({
  api: {
    put: (url: string, body: unknown) => {
      srv.put(url, body);
      return srv.fail ? Promise.reject(new Error('offline')) : Promise.resolve();
    }
  }
}));
vi.mock('./state.svelte', () => ({ session: { user: { id: 3 } }, sidebar: { get prefs() { return srv.prefs; } } }));
const mem = new Map<string, string>();
vi.stubGlobal('localStorage', {
  getItem: (k: string) => mem.get(k) ?? null,
  setItem: (k: string, v: string) => void mem.set(k, String(v)),
  removeItem: (k: string) => void mem.delete(k),
  clear: () => mem.clear()
});

import { displayOf, displayStore, setDisplay, syncDisplay } from './display.svelte';

const DEF = { steps: true, percent: true, allTags: false };

describe('display options (pure)', () => {
  it('defaults, overridden by stored booleans only', () => {
    expect(resolveDisplay(DEF, { steps: false, allTags: 'yes', other: true })).toEqual({ steps: false, percent: true, allTags: false });
    expect(resolveDisplay(DEF, null)).toEqual(DEF);
  });
  it('toggling one option keeps the others and other pages', () => {
    const s = withOption({ 'overview.list': { steps: false }, customers: { due: false } }, 'overview.list', 'percent', false);
    expect(s).toEqual({ 'overview.list': { steps: false, percent: false }, customers: { due: false } });
  });
  it('parses stored JSON defensively', () => {
    expect(parseStore('{"a":{"x":true,"y":1}}')).toEqual({ a: { x: true } });
    expect(parseStore('nope')).toEqual({});
    expect(parseStore([1])).toEqual({});
  });
});

describe('display options (persistence)', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    mem.clear();
    srv.put.mockClear();
    srv.fail = false;
    srv.prefs = {};
    displayStore.all = {};
    displayStore.loaded = false;
  });

  it('saves per user and page/view on the server (debounced) and in localStorage', async () => {
    setDisplay('overview.list', 'steps', false);
    expect(displayOf('overview.list', DEF).steps).toBe(false);
    expect(JSON.parse(mem.get('display:3')!)).toEqual({ 'overview.list': { steps: false } });
    await vi.advanceTimersByTimeAsync(500);
    expect(srv.put).toHaveBeenCalledWith('/prefs/display', { 'overview.list': { steps: false } });
    expect(displayOf('customers.list', { due: true }).due).toBe(true); // other page untouched
  });

  it('takes over the server value when the sidebar data arrives', () => {
    srv.prefs = { display: { 'overview.list': { allTags: true } } };
    syncDisplay();
    expect(displayOf('overview.list', DEF)).toEqual({ steps: true, percent: true, allTags: true });
  });

  it('falls back to localStorage when the server has nothing / saving fails', async () => {
    srv.fail = true;
    setDisplay('overview.list', 'percent', false);
    await vi.advanceTimersByTimeAsync(500);
    // next start: server empty → localStorage
    displayStore.all = {};
    syncDisplay();
    expect(displayOf('overview.list', DEF).percent).toBe(false);
  });
});
