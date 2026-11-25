import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { BlockListModel } from './blockListModel';
import type { Item, Kind, Ops } from './types';

// ---- In-memory Ops adapter (the test-side counterpart of the checklist/template adapters) ----

type Call = [string, ...unknown[]];

function fakeOps(opts: { checklist?: boolean } = {}) {
  let nextId = 100;
  const calls: Call[] = [];
  const held: (() => void)[] = [];
  const fail = new Set<string>();
  let holding = false;
  // the server fills in names/timestamps itself
  const echo = (id: number, body: Record<string, unknown> = {}) =>
    ({ id, ...body, assignee_name: body.assignee_id ? 'Server Name' : null, title: 'SERVER TEXT' }) as unknown as Item;
  const run = <R>(name: string, args: unknown[], res: () => R): Promise<R> => {
    calls.push([name, ...args]);
    const go = () => (fail.delete(name) ? Promise.reject(new Error(`${name} failed`)) : Promise.resolve(res()));
    if (!holding) return go();
    return new Promise<R>((resolve, reject) => held.push(() => go().then(resolve, reject)));
  };
  const ops: Ops<Item> = {
    create: (body) => run('create', [body], () => echo(nextId++, body)),
    patch: (id, body) => run('patch', [id, body], () => echo(id, body)),
    remove: (id) => run('remove', [id], () => undefined),
    reorder: (ids) => run('reorder', [ids], () => undefined)
  };
  if (opts.checklist)
    ops.toggle = (id, done) =>
      run('toggle', [id, done], () => ({ id, done_at: done ? '2026-01-01T00:00:00Z' : null, done_by: done ? 9 : null, done_by_name: done ? 'Srv' : null }) as Item);
  return {
    ops,
    calls,
    /** Server calls only by name and id. */
    names: () => calls.map((c) => (typeof c[1] === 'number' ? `${c[0]}:${c[1]}` : c[0])),
    /** Holds every call until release() (slow network). */
    hold: () => (holding = true),
    release: () => {
      holding = false;
      for (const h of held.splice(0)) h();
    },
    failNext: (name: string) => fail.add(name)
  };
}

function row(id: number, kind: Kind = 'text', extra: Partial<Item> = {}): Item {
  return {
    id, kind, title: `t${id}`, description: '', note_color: '', bg_color: '', indent: 0, lang: '', position: 0, checklist_id: 1,
    assignee_id: null, assignee_name: null, due_date: null, done_at: null, done_by: null, done_by_name: null, status: '', ...extra
  };
}

function setup(rows: Item[], opts: { checklist?: boolean } = {}) {
  rows.forEach((r, i) => (r.position = i));
  const srv = fakeOps(opts);
  const state = { items: rows };
  const onChange = vi.fn();
  const onError = vi.fn();
  const onRemove = vi.fn();
  const model = new BlockListModel<Item>({
    items: () => state.items,
    setItems: (next) => (state.items = next),
    ops: () => srv.ops,
    user: () => ({ id: 7, name: 'Me' }),
    onChange,
    onError,
    onRemove
  });
  // the editor renders every row once, which records the saved text
  for (const r of rows) {
    model.baseline(r.id, 'title', r.title);
    model.baseline(r.id, 'description', r.description);
  }
  const ids =() => state.items.map((b) => b.id);
  const titles = () => state.items.map((b) => b.title);
  const get = (key: number) => state.items.find((b) => b.id === key)!;
  return { model, srv, state, ids, titles, get, onChange, onError, onRemove };
}

beforeEach(() => vi.useFakeTimers());
afterEach(() => vi.useRealTimers());

describe('BlockListModel: insert and delete', () => {
  it('inserts optimistically with a local key, shifts positions and creates on the server', async () => {
    const { model, srv, state, ids, onChange } = setup([row(1), row(2)]);
    const key = model.insert(1, { kind: 'item', title: 'new' });
    expect(key).toBeLessThan(0);
    expect(ids()).toEqual([1, key, 2]);
    expect(state.items.map((b) => b.position)).toEqual([0, 1, 2]);
    expect(onChange).toHaveBeenCalledTimes(1);
    expect(model.serverId(key)).toBeUndefined();
    await model.settled();
    expect(srv.calls[0]).toEqual(['create', { kind: 'item', title: 'new', description: '', note_color: '', bg_color: '', indent: 0, lang: '', position: 1 }]);
    expect(model.serverId(key)).toBe(100);
  });

  it('inserts at the start with afterKey null', () => {
    const { model, ids } = setup([row(1)]);
    const key = model.insert(null, {});
    expect(ids()).toEqual([key, 1]);
  });

  it('sends assignee/due/status/done only for checklists', async () => {
    const tpl = setup([row(1)]);
    tpl.model.insert(1, { assignee_id: 3, due_date: '2026-02-01', status: 'ok', done_at: 'x' });
    await tpl.model.settled();
    expect(tpl.srv.names()).toEqual(['create']);
    expect(tpl.srv.calls[0][1]).not.toHaveProperty('assignee_id');

    const cl = setup([row(1)], { checklist: true });
    const key = cl.model.insert(1, { assignee_id: 3, due_date: '2026-02-01', status: 'ok', done_at: 'x' });
    await cl.model.settled();
    expect(cl.srv.calls[0][1]).toMatchObject({ assignee_id: 3, due_date: '2026-02-01', status: 'ok' });
    expect(cl.srv.names()).toEqual(['create', 'toggle:100']);
    expect(cl.get(key).done_by_name).toBe('Srv');
  });

  it('removes a block and calls onRemove; undo recreates it under the same key', async () => {
    const { model, srv, ids, get, onRemove } = setup([row(1), row(2, 'item', { description: 'n' }), row(3)]);
    model.remove(2);
    expect(ids()).toEqual([1, 3]);
    expect(onRemove).toHaveBeenCalledWith(2);
    await model.undo();
    expect(ids()).toEqual([1, 2, 3]);
    expect(get(2)).toMatchObject({ kind: 'item', title: 't2', description: 'n' });
    await model.settled();
    expect(srv.names()).toEqual(['remove:2', 'create']);
    // the recreated row has a new server id, later writes use it
    expect(model.serverId(2)).toBe(100);
    await model.redo();
    expect(ids()).toEqual([1, 3]);
    await model.settled();
    expect(srv.names().at(-1)).toBe('remove:100');
  });
});

describe('BlockListModel: text', () => {
  it('applies input at once and commits after the debounce', async () => {
    const { model, srv, get } = setup([row(1)]);
    model.baseline(1, 'title', 't1');
    model.text(1, 'title', 'ab');
    expect(get(1).title).toBe('ab');
    expect(model.pendingText).toBe(true);
    await model.settled();
    expect(srv.calls).toEqual([]);
    vi.advanceTimersByTime(700);
    await model.settled();
    expect(srv.calls).toEqual([['patch', 1, { title: 'ab' }]]);
    expect(model.pendingText).toBe(false);
    // the server text is never taken over
    expect(get(1).title).toBe('ab');
  });

  it('does not commit unchanged text', async () => {
    const { model, srv } = setup([row(1)]);
    model.baseline(1, 'title', 't1');
    model.text(1, 'title', 't1');
    model.commitText(1, 'title');
    await model.settled();
    expect(srv.calls).toEqual([]);
    expect(model.canUndo).toBe(false);
  });

  it('merges rapid commits of one field into one undo step', async () => {
    const { model, get } = setup([row(1)]);
    model.baseline(1, 'title', 't1');
    model.text(1, 'title', 'a');
    model.commitText(1, 'title');
    vi.advanceTimersByTime(500);
    model.text(1, 'title', 'ab');
    model.commitText(1, 'title');
    await model.undo();
    expect(get(1).title).toBe('t1');
    expect(model.canUndo).toBe(false);
    await model.redo();
    expect(get(1).title).toBe('ab');
  });

  it('keeps separate undo steps after a pause', async () => {
    const { model, get } = setup([row(1)]);
    model.baseline(1, 'title', 't1');
    model.text(1, 'title', 'a');
    model.commitText(1, 'title');
    vi.advanceTimersByTime(2500);
    model.text(1, 'title', 'ab');
    model.commitText(1, 'title');
    await model.undo();
    expect(get(1).title).toBe('a');
  });

  it('undo flushes pending input first', async () => {
    const { model, srv, get } = setup([row(1)]);
    model.baseline(1, 'title', 't1');
    model.text(1, 'title', 'typed');
    await model.undo();
    expect(get(1).title).toBe('t1');
    await model.settled();
    expect(srv.calls).toEqual([
      ['patch', 1, { title: 'typed' }],
      ['patch', 1, { title: 't1' }]
    ]);
  });

  it('flush commits every pending field', async () => {
    const { model, srv } = setup([row(1), row(2)]);
    model.text(1, 'title', 'x');
    model.text(2, 'description', 'y');
    model.flush();
    await model.settled();
    expect(srv.calls).toEqual([
      ['patch', 1, { title: 'x' }],
      ['patch', 2, { description: 'y' }]
    ]);
  });

  it('update commits pending text of the other field first and drops pending text of a patched field', async () => {
    const { model, srv } = setup([row(1)]);
    model.text(1, 'description', 'note');
    model.text(1, 'title', 'lost');
    model.update(1, { title: 'set' });
    vi.advanceTimersByTime(1000);
    await model.settled();
    expect(srv.calls).toEqual([
      ['patch', 1, { description: 'note' }],
      ['patch', 1, { title: 'set' }]
    ]);
  });
});

describe('BlockListModel: split and merge', () => {
  it('split keeps the head, moves the tail into a new block below, one undo step', async () => {
    const { model, ids, titles, get } = setup([row(1, 'bullet', { indent: 2 }), row(2)]);
    model.baseline(1, 'title', 't1');
    const key = model.split(1, 'he', 'ad')!;
    expect(ids()).toEqual([1, key, 2]);
    expect(titles()).toEqual(['he', 'ad', 't2']);
    expect(get(key)).toMatchObject({ kind: 'bullet', indent: 2 });
    await model.undo();
    expect(ids()).toEqual([1, 2]);
    expect(titles()).toEqual(['t1', 't2']);
    await model.redo();
    expect(titles()).toEqual(['he', 'ad', 't2']);
    expect(ids()).toEqual([1, key, 2]);
  });

  it('split of a heading continues with text', () => {
    const { model, get } = setup([row(1, 'h2')]);
    expect(get(model.split(1, 'a', 'b')!).kind).toBe('text');
  });

  it('insertAbove adds an empty sibling above', () => {
    const { model, ids, get } = setup([row(1), row(2, 'item', { indent: 1 })]);
    const key = model.insertAbove(2)!;
    expect(ids()).toEqual([1, key, 2]);
    expect(get(key)).toMatchObject({ kind: 'item', indent: 1, title: '' });
  });

  it('backspace at start merges into the previous block (one undo step)', async () => {
    const { model, srv, ids, get } = setup([row(1, 'text', { title: 'a<b>b</b>' }), row(2, 'text', { title: 'cd', description: 'note' })]);
    model.baseline(2, 'title', 'cd');
    expect(model.backspaceAtStart(2)).toEqual({ key: 1, at: 2 });
    expect(ids()).toEqual([1]);
    expect(get(1)).toMatchObject({ title: 'a<b>b</b>cd', description: 'note' });
    await model.settled();
    expect(srv.calls).toEqual([
      ['patch', 1, { title: 'a<b>b</b>cd', description: 'note' }],
      ['remove', 2]
    ]);
    await model.undo();
    expect(ids()).toEqual([1, 2]);
    expect(get(1)).toMatchObject({ title: 'a<b>b</b>', description: '' });
    expect(get(2).title).toBe('cd');
  });

  it('backspace keeps the previous note when it has one', () => {
    const { model, get } = setup([row(1, 'text', { description: 'keep' }), row(2, 'text', { description: 'drop' })]);
    model.backspaceAtStart(2);
    expect(get(1).description).toBe('keep');
  });

  it('backspace on a non-text block turns it into text, then outdents', () => {
    const { model, get } = setup([row(1), row(2, 'item', { indent: 1 })]);
    expect(model.backspaceAtStart(2)).toBeNull();
    expect(get(2)).toMatchObject({ kind: 'text', indent: 1 });
    expect(model.backspaceAtStart(2)).toBeNull();
    expect(get(2).indent).toBe(0);
  });

  it('backspace removes a divider above, moves into code, does nothing on the first block', () => {
    const { model, ids } = setup([row(1), row(2, 'divider'), row(3), row(4, 'code'), row(5)]);
    expect(model.backspaceAtStart(1)).toBeNull();
    expect(model.backspaceAtStart(3)).toBeNull();
    expect(ids()).toEqual([1, 3, 4, 5]);
    expect(model.backspaceAtStart(5)).toEqual({ key: 4, at: 'end' });
    expect(ids()).toEqual([1, 3, 4, 5]);
  });

  it('exitList outdents, then turns the item into text', () => {
    const { model, get } = setup([row(1, 'item', { indent: 1 })]);
    model.exitList(1);
    expect(get(1)).toMatchObject({ kind: 'item', indent: 0 });
    model.exitList(1);
    expect(get(1).kind).toBe('text');
  });

  it('indent clamps to 0..4 and skips no-op changes', () => {
    const { model, get } = setup([row(1, 'text', { indent: 4 })]);
    model.indent(1, 1);
    expect(model.canUndo).toBe(false);
    model.indent(1, -1);
    expect(get(1).indent).toBe(3);
  });
});

describe('BlockListModel: convert, add, paste', () => {
  it('converts text to code as plain text and back as escaped html', () => {
    const { model, get } = setup([row(1, 'text', { title: 'a <b>x</b> &amp;' })]);
    model.convert(1, 'code');
    expect(get(1)).toMatchObject({ kind: 'code', title: 'a x &' });
    get(1).title = 'l1\n<l2>';
    model.convert(1, 'text');
    expect(get(1).title).toBe('l1<br>&lt;l2&gt;');
  });

  it('convert with clear empties the title (slash command)', () => {
    const { model, get } = setup([row(1, 'text', { title: '/h1' })]);
    expect(model.convert(1, 'h1', true)).toEqual({});
    expect(get(1)).toMatchObject({ kind: 'h1', title: '' });
  });

  it('convert to divider adds a text block below; undo is one step', async () => {
    const { model, ids, get } = setup([row(1, 'text', { indent: 1 })]);
    const res = model.convert(1, 'divider')!;
    expect(ids()).toEqual([1, res.inserted]);
    expect(get(1)).toMatchObject({ kind: 'divider', title: '' });
    expect(get(res.inserted!)).toMatchObject({ kind: 'text', indent: 1 });
    await model.undo();
    expect(ids()).toEqual([1]);
    expect(get(1)).toMatchObject({ kind: 'text', title: 't1' });
    expect(model.canUndo).toBe(false);
  });

  it('convert of a missing block returns null', () => {
    expect(setup([]).model.convert(5, 'h1')).toBeNull();
  });

  it('add: headings unindented, divider brings a text block', () => {
    const { model, ids, get } = setup([row(1)]);
    const h = model.add(1, 'h1', 3);
    expect(get(h).indent).toBe(0);
    const b = model.add(h, 'bullet', 3);
    expect(get(b).indent).toBe(3);
    const t = model.add(b, 'divider', 3);
    expect(ids().slice(-2).map((k) => get(k).kind)).toEqual(['divider', 'text']);
    expect(ids().at(-1)).toBe(t);
  });

  it('pasteLines: title + one block per non-blank line, one undo step', async () => {
    const { model, ids, titles } = setup([row(1, 'item'), row(2)]);
    model.baseline(1, 'title', 't1');
    const last = model.pasteLines(1, 't1 first', ['second <x>', '  ', 'third']);
    expect(titles()).toEqual(['t1 first', 'second &lt;x&gt;', 'third', 't2']);
    expect(ids()[2]).toBe(last);
    await model.undo();
    expect(titles()).toEqual(['t1', 't2']);
  });
});

describe('BlockListModel: update, reorder, done', () => {
  it('update is optimistic, persisted and undoable; server fields are merged, text never', async () => {
    const { model, srv, get, onChange } = setup([row(1, 'item')], { checklist: true });
    model.update(1, { assignee_id: 4, bg_color: 'blue' } as Partial<Item>);
    expect(get(1)).toMatchObject({ assignee_id: 4, bg_color: 'blue' });
    expect(onChange).toHaveBeenCalledTimes(1);
    await model.settled();
    expect(srv.calls).toEqual([['patch', 1, { assignee_id: 4, bg_color: 'blue' }]]);
    expect(get(1)).toMatchObject({ assignee_name: 'Server Name', title: 't1' });
    await model.undo();
    expect(get(1)).toMatchObject({ assignee_id: null, bg_color: '' });
  });

  it('reorder sets positions and sends server ids; incomplete lists are ignored', async () => {
    const { model, srv, state, ids } = setup([row(1), row(2), row(3)]);
    model.reorder([3, 1]);
    expect(ids()).toEqual([1, 2, 3]);
    model.reorder([3, 1, 2]);
    expect(ids()).toEqual([3, 1, 2]);
    expect(state.items.map((b) => b.position)).toEqual([0, 1, 2]);
    await model.settled();
    expect(srv.calls).toEqual([['reorder', [3, 1, 2]]]);
    await model.undo();
    expect(ids()).toEqual([1, 2, 3]);
  });

  it('setDone marks items optimistically with the current user and merges the server answer', async () => {
    const { model, srv, get } = setup([row(1, 'item'), row(2, 'text')], { checklist: true });
    model.setDone(1, true);
    expect(get(1)).toMatchObject({ done_by: 7, done_by_name: 'Me' });
    expect(get(1).done_at).toBeTruthy();
    model.setDone(1, true); // no-op
    model.setDone(2, true); // not an item
    await model.settled();
    expect(srv.calls).toEqual([['toggle', 1, true]]);
    expect(get(1)).toMatchObject({ done_at: '2026-01-01T00:00:00Z', done_by_name: 'Srv' });
    await model.undo();
    expect(get(1)).toMatchObject({ done_at: null, done_by: null, done_by_name: null });
    await model.settled();
    expect(srv.calls.at(-1)).toEqual(['toggle', 1, false]);
  });

  it('setDone is ignored without a toggle op (templates)', () => {
    const { model, get } = setup([row(1, 'item')]);
    model.setDone(1, true);
    expect(get(1).done_at).toBeNull();
    expect(model.canUndo).toBe(false);
  });
});

describe('BlockListModel: sync queue', () => {
  it('runs server calls strictly in edit order, one at a time', async () => {
    const { model, srv } = setup([row(1), row(2)], { checklist: true });
    srv.hold();
    const k = model.insert(2, { kind: 'item' });
    model.update(1, { indent: 1 });
    model.reorder([k, 1, 2]);
    model.remove(2);
    await Promise.resolve();
    // only the first call is in flight
    expect(srv.names()).toEqual(['create']);
    srv.release();
    await model.settled();
    expect(srv.names()).toEqual(['create', 'patch:1', 'reorder', 'remove:2']);
    expect(srv.calls[2]).toEqual(['reorder', [100, 1, 2]]);
  });

  it('maps the local key to the server id when the create resolves after further edits', async () => {
    const { model, srv, get } = setup([row(1)], { checklist: true });
    srv.hold();
    const k = model.insert(1, { kind: 'item' });
    model.text(k, 'title', 'typed early');
    model.commitText(k, 'title');
    model.update(k, { status: 'ok' } as Partial<Item>);
    model.setDone(k, true);
    expect(get(k).title).toBe('typed early');
    srv.release();
    await model.settled();
    expect(srv.calls.slice(1)).toEqual([
      ['patch', 100, { title: 'typed early' }],
      ['patch', 100, { status: 'ok' }],
      ['toggle', 100, true]
    ]);
    expect(model.serverId(k)).toBe(100);
  });

  it('deleting a block whose create is still pending removes it by its server id', async () => {
    const { model, srv } = setup([row(1)]);
    srv.hold();
    const k = model.insert(1, {});
    model.remove(k);
    srv.release();
    await model.settled();
    expect(srv.names()).toEqual(['create', 'remove:100']);
    expect(model.serverId(k)).toBeUndefined();
  });

  it('skips patches of blocks deleted meanwhile', async () => {
    const { model, srv } = setup([row(1), row(2)]);
    srv.hold();
    model.update(2, { indent: 1 });
    model.remove(2);
    srv.release();
    await model.settled();
    // the patch was queued while the block existed, but runs after the delete
    expect(srv.names()).toEqual(['remove:2']);
  });

  it('a failed write reports the error once and later writes still run', async () => {
    const { model, srv, get, onError } = setup([row(1, 'item')], { checklist: true });
    srv.failNext('patch');
    model.update(1, { assignee_id: 4 } as Partial<Item>);
    model.update(1, { indent: 1 });
    await model.settled();
    expect(onError).toHaveBeenCalledTimes(1);
    expect((onError.mock.calls[0][0] as Error).message).toBe('patch failed');
    expect(srv.names()).toEqual(['patch:1', 'patch:1']);
    // optimistic state stays, the failed answer is not merged
    expect(get(1)).toMatchObject({ assignee_id: 4, assignee_name: null, indent: 1 });
  });

  it('a failed create leaves the block without server id; its later writes are skipped', async () => {
    const { model, srv, onError } = setup([row(1)]);
    srv.failNext('create');
    const k = model.insert(1, {});
    model.update(k, { indent: 1 });
    model.reorder([k, 1]);
    model.remove(k);
    await model.settled();
    expect(onError).toHaveBeenCalledTimes(1);
    expect(srv.names()).toEqual(['create']);
  });
});
