import { beforeEach, describe, expect, it, vi } from 'vitest';
import { flushSync } from 'svelte';

// SvelteKit's page state and shallow routing, simulated
const nav = vi.hoisted(() => ({ page: { url: new URL('http://app/tasks'), state: {} } }));
vi.mock('$app/state', () => ({
  get page() {
    return nav.page;
  }
}));
vi.mock('$app/navigation', () => ({
  replaceState: vi.fn((url: URL) => {
    nav.page.url = new URL(url);
  })
}));
vi.mock('./state.svelte', () => ({ session: { user: { id: 7 } } }));
// Node's own (flag-gated) localStorage global shadows jsdom's: simple in-memory one
const mem = new Map<string, string>();
vi.stubGlobal('localStorage', {
  getItem: (k: string) => mem.get(k) ?? null,
  setItem: (k: string, v: string) => void mem.set(k, String(v)),
  removeItem: (k: string) => void mem.delete(k),
  clear: () => mem.clear()
});

import { pageState } from './pageState.svelte';

const spec = {
  chip: { def: 'open' as string, url: 'str' as const },
  sort: { def: 'due' as string, url: 'str' as const },
  closed: { def: [] as number[] }
};

function mount(search = '') {
  nav.page.url = new URL(`http://app/tasks${search}`);
  let st!: ReturnType<typeof pageState<typeof spec>>;
  const stop = $effect.root(() => {
    st = pageState('tasks', spec);
  });
  flushSync();
  return { st, stop };
}

describe('pageState', () => {
  beforeEach(() => localStorage.clear());

  it('saves per page and user in localStorage and mirrors URL fields into the URL', () => {
    const { st, stop } = mount();
    st.chip = 'done';
    st.closed = [3];
    flushSync();
    expect(JSON.parse(localStorage.getItem('page-state:7:tasks')!)).toEqual({ chip: 'done', sort: 'due', closed: [3] });
    expect(nav.page.url.search).toBe('?chip=done'); // collapse state stays out of the URL
    stop();
  });

  it('restores the stored state on the next visit', () => {
    localStorage.setItem('page-state:7:tasks', JSON.stringify({ chip: 'today', sort: 'name', closed: [1, 2] }));
    const { st, stop } = mount();
    expect(st.chip).toBe('today');
    expect(st.closed).toEqual([1, 2]);
    expect(nav.page.url.search).toBe('?chip=today&sort=name');
    stop();
  });

  it('a shared link wins over the stored state', () => {
    localStorage.setItem('page-state:7:tasks', JSON.stringify({ chip: 'today' }));
    const { st, stop } = mount('?chip=done&sort=progress');
    expect(st.chip).toBe('done');
    expect(st.sort).toBe('progress');
    stop();
  });
});
