import { beforeEach, describe, expect, it, vi } from 'vitest';

// the app singleton is created at import; its transport is never used here
vi.mock('./api', () => ({ api: {}, errorText: () => 'error' }));

import { createWorkspace, type WorkspaceTimers, type WorkspaceTransport } from './workspace.svelte';
import type { SidebarCustomer, SidebarData, SidebarProject } from './types';

const proj = (id: number, over: Partial<SidebarProject> = {}): SidebarProject => ({
  id,
  name: `P${id}`,
  status: 'active',
  start_date: null,
  due_date: null,
  created_at: '2026-11-01',
  include_in_customer_progress: true,
  tag_ids: [],
  owner_id: null,
  owner_name: '',
  board_position: 0,
  background: '',
  total: 0,
  done: 0,
  checklists: [],
  tables: [],
  ...over
});
const cust = (id: number, projects: SidebarProject[]): SidebarCustomer => ({ id, name: `C${id}`, location: '', background: '', color: '', color_everywhere: false, projects });

function payload(): SidebarData {
  return {
    customers: [
      cust(1, [
        proj(1, { checklists: [{ id: 11, title: 'A', total: 1, done: 0, next_due: '' }, { id: 12, title: 'B', total: 0, done: 0, next_due: '' }], tables: [{ id: 21, title: 'T' }] }),
        proj(2, { board_position: 1 })
      ]),
      cust(2, [proj(3, { status: 'planning' })])
    ],
    my_tasks: 2,
    favorites: [],
    tags: [{ id: 5, name: 'x', color: '#000' }] as SidebarData['tags'],
    status_labels: {},
    prefs: { overview: { group: 'owner' } },
    board_hidden: [2]
  };
}

// in-memory server: GET /sidebar serves `data`; writes are recorded and can fail
function fakeServer() {
  const calls: string[] = [];
  let data = payload();
  let failNext = false;
  let gate: Promise<void> | null = null;
  const respond = async <T>(method: string, url: string, body?: unknown, result?: unknown): Promise<T> => {
    calls.push(`${method} ${url}${body === undefined ? '' : ' ' + JSON.stringify(body)}`);
    if (gate) await gate;
    if (failNext) {
      failNext = false;
      throw new Error('boom');
    }
    return (method === 'GET' && url === '/sidebar' ? structuredClone(data) : result) as T;
  };
  const transport: WorkspaceTransport = {
    get: (url) => respond('GET', url),
    post: (url, body) => respond('POST', url, body, { id: 99, title: 'new', name: 'new' }),
    put: (url, body) => respond('PUT', url, body, url.endsWith('/tags') ? { tag_ids: [7, 8] } : undefined),
    patch: (url, body) => respond('PATCH', url, body, { id: 1, name: 'saved' }),
    del: (url) => respond('DELETE', url)
  };
  let open: () => void = () => {};
  return {
    transport,
    calls,
    setData: (d: SidebarData) => (data = d),
    failNext: () => (failNext = true),
    hold: () => (gate = new Promise((r) => (open = r))),
    release: () => {
      gate = null;
      open();
    }
  };
}

function fakeTimers() {
  const pending = new Map<number, { ms: number; fn: () => void }>();
  let next = 1;
  const timers: WorkspaceTimers = {
    set: (fn, ms) => {
      pending.set(next, { ms, fn });
      return next++;
    },
    clear: (h) => void pending.delete(h as number)
  };
  /** Fires all timers (debounced reloads) and lets the requests settle. */
  async function run() {
    const list = [...pending.values()];
    pending.clear();
    list.forEach((t) => t.fn());
    await settle();
  }
  return { timers, run, delays: () => [...pending.values()].map((t) => t.ms) };
}

const settle = async () => {
  for (let i = 0; i < 10; i++) await Promise.resolve();
};

async function setup() {
  const srv = fakeServer();
  const clock = fakeTimers();
  const loaded: SidebarData[] = [];
  const ws = createWorkspace({ transport: srv.transport, timers: clock.timers, onLoaded: (r) => loaded.push(r) });
  ws.refresh(0);
  await clock.run();
  srv.calls.length = 0;
  return { ws, srv, clock, loaded };
}

describe('workspace store', () => {
  let t: Awaited<ReturnType<typeof setup>>;
  beforeEach(async () => {
    t = await setup();
  });

  it('loads the read model and hands the payload to the prefs', () => {
    const { ws, loaded } = t;
    expect(ws.state.loaded).toBe(true);
    expect(ws.state.customers.map((c) => c.id)).toEqual([1, 2]);
    expect(ws.state.myTasks).toBe(2);
    expect(ws.state.favoriteGroups).toEqual([]);
    expect(ws.state.statusColors).toEqual({});
    expect(loaded[0].board_hidden).toEqual([2]);
  });

  it('reloads are debounced: rapid calls = one request', async () => {
    const { ws, srv, clock } = t;
    ws.refresh();
    ws.refresh();
    ws.refresh(0);
    expect(clock.delays()).toEqual([0]);
    await clock.run();
    expect(srv.calls).toEqual(['GET /sidebar']);
  });

  it('updateProject: optimistic, then invalidates', async () => {
    const { ws, srv, clock } = t;
    const done = ws.updateProject(2, { status: 'blocked', owner_id: 4 }, 'Ann');
    expect(ws.findProject(2)).toMatchObject({ status: 'blocked', owner_id: 4, owner_name: 'Ann' });
    await done;
    expect(srv.calls).toEqual(['PATCH /projects/2 {"status":"blocked","owner_id":4}']);
    expect(clock.delays()).toEqual([0]);
    srv.setData({ ...payload(), my_tasks: 9 });
    await clock.run();
    expect(ws.state.myTasks).toBe(9); // invalidation reloaded the read model
  });

  it('updateProject: rolls back and rejects on error, no reload', async () => {
    const { ws, srv, clock } = t;
    srv.failNext();
    await expect(ws.updateProject(2, { name: 'X', owner_id: null })).rejects.toThrow('boom');
    expect(ws.findProject(2)).toMatchObject({ name: 'P2', owner_id: null, owner_name: '' });
    expect(clock.delays()).toEqual([]);
  });

  it('board move: status and positions at once, rolled back on error, no reload', async () => {
    const { ws, srv, clock } = t;
    srv.hold();
    const p = ws.moveProject('planning', [1, 3]);
    expect(ws.findProject(1)).toMatchObject({ status: 'planning', board_position: 0 });
    expect(ws.findProject(3)).toMatchObject({ status: 'planning', board_position: 1 });
    srv.failNext();
    srv.release();
    await expect(p).rejects.toThrow();
    expect(ws.findProject(1)).toMatchObject({ status: 'active', board_position: 0 });
    expect(ws.findProject(3)).toMatchObject({ status: 'planning', board_position: 0 });
    expect(srv.calls).toEqual(['PUT /board/order {"status":"planning","ids":[1,3]}']);
    expect(clock.delays()).toEqual([]);
  });

  it('tags: optimistic, then the server answer wins', async () => {
    const { ws } = t;
    const p = ws.setProjectTags(1, [7]);
    expect(ws.findProject(1)!.tag_ids).toEqual([7]);
    expect(await p).toEqual([7, 8]);
    expect(ws.findProject(1)!.tag_ids).toEqual([7, 8]);
  });

  it('checklists: rename (rollback), reorder, delete drops it from the tree', async () => {
    const { ws, srv } = t;
    srv.failNext();
    await expect(ws.renameChecklist(11, 'New')).rejects.toThrow();
    expect(ws.findChecklist(11)!.title).toBe('A');
    await ws.renameChecklist(11, 'New');
    expect(ws.findChecklist(11)!.title).toBe('New');
    await ws.reorderChecklists(1, [12, 11]);
    expect(ws.findProject(1)!.checklists.map((c) => c.id)).toEqual([12, 11]);
    await ws.deleteChecklist(12);
    expect(ws.findProject(1)!.checklists.map((c) => c.id)).toEqual([11]);
    expect(srv.calls.slice(-3)).toEqual(['PATCH /checklists/11 {"title":"New"}', 'PUT /projects/1/checklists/order {"ids":[12,11]}', 'DELETE /checklists/12']);
  });

  it('tables, projects, customers, tags: deletions update the tree after success', async () => {
    const { ws, srv } = t;
    await ws.renameTable(21, 'U');
    expect(ws.findProject(1)!.tables[0].title).toBe('U');
    await ws.deleteTable(21);
    expect(ws.findProject(1)!.tables).toEqual([]);
    srv.failNext();
    await expect(ws.deleteProject(2)).rejects.toThrow();
    expect(ws.findProject(2)).toBeDefined();
    await ws.deleteProject(2);
    expect(ws.findProject(2)).toBeUndefined();
    await ws.deleteCustomer(2);
    expect(ws.state.customers.map((c) => c.id)).toEqual([1]);
    await ws.deleteTag(5);
    expect(ws.state.tags).toEqual([]);
  });

  it('customer update reloads only for fields of the read model', async () => {
    const { ws, clock } = t;
    await ws.updateCustomer(1, { notes: 'n' });
    expect(clock.delays()).toEqual([]);
    const p = ws.updateCustomer(1, { name: 'Renamed' });
    expect(ws.findCustomer(1)!.name).toBe('Renamed');
    await p;
    expect(clock.delays()).toEqual([0]);
  });

  it('colors and backgrounds: optimistic without reload, rollback on error', async () => {
    const { ws, srv, clock } = t;
    await ws.setCustomerColors(1, { color: '#ff0000' });
    expect(ws.findCustomer(1)).toMatchObject({ color: '#ff0000', color_everywhere: false });
    srv.failNext();
    await expect(ws.setProjectBackground(1, 'g3')).rejects.toThrow();
    expect(ws.findProject(1)!.background).toBe('');
    await ws.setCustomerBackground(2, 'g1');
    expect(ws.findCustomer(2)!.background).toBe('g1');
    expect(clock.delays()).toEqual([]);
  });

  it('item done: debounced reload (400 ms)', async () => {
    const { ws, srv, clock } = t;
    await ws.setItemDone(5, true);
    expect(srv.calls).toEqual(['POST /items/5/done {"done":true}']);
    expect(clock.delays()).toEqual([400]);
  });

  it('batch: several writes, one reload at the end (also after an error)', async () => {
    const { ws, srv, clock } = t;
    const r = await ws.batch(async () => {
      const p = await ws.createProject(1, { name: 'N' });
      await ws.createChecklistFromTemplate(p.id, { template_id: 3, title: '' });
      expect(clock.delays()).toEqual([]);
      return p.id;
    });
    expect(r).toBe(99);
    expect(clock.delays()).toEqual([0]);
    await clock.run();
    expect(srv.calls).toEqual(['POST /customers/1/projects {"name":"N"}', 'POST /projects/99/checklists/from-template {"template_id":3,"title":""}', 'GET /sidebar']);

    srv.calls.length = 0;
    await expect(
      ws.batch(async () => {
        await ws.updateProject(1, { status: 'active' });
        srv.failNext();
        await ws.deleteProject(1);
      })
    ).rejects.toThrow();
    expect(clock.delays()).toEqual([0]); // the first write still invalidates
  });

  it('a failed reload keeps the current read model', async () => {
    const { ws, srv, clock } = t;
    srv.failNext();
    ws.refresh(0);
    await clock.run();
    expect(ws.state.customers).toHaveLength(2);
  });
});
