// Block list model of the block editor (checklists and templates).
//
// Owns everything about the list that is not DOM: block edits (insert,
// delete, split, merge, convert, reorder, done), debounced text commits,
// undo/redo history and the server sync. Local state is authoritative:
// edits mutate the rows immediately (optimistic) and all writes run
// sequentially through one queue, so positions and newly created IDs stay
// consistent. New blocks get a negative local key at once (focus/typing
// without waiting); the server ID is mapped to that key once the create
// resolves. The server is reached only through the `Ops` seam.
//
// The rows live outside (a bindable prop in the component); the model reads
// and replaces them through `items`/`setItems`, so reactivity stays with
// the owner.

import { isHeading, nextKind } from './blocks';
import { History } from './history.svelte';
import { escapeHtml, htmlToText } from './rich';
import type { Item, Kind, Ops, Row } from './types';

export type Field = 'title' | 'description';
export type Block<T extends Row> = T & Partial<Item>;

export interface BlockListConfig<T extends Row> {
  /** Current rows (mutated in place). */
  items: () => T[];
  /** Replaces the row array (reorder). */
  setItems: (next: T[]) => void;
  /** Server adapter; read at the time a write runs. */
  ops: () => Ops<T>;
  /** Who marks items done (optimistic done_by/done_by_name). */
  user?: () => { id: number; name: string } | null | undefined;
  /** After every local change of the list (not on server echoes). */
  onChange?: () => void;
  /** A queued server write failed; later writes still run. */
  onError?: (e: unknown) => void;
  /** A block left the list (delete, undo of an insert). */
  onRemove?: (key: number) => void;
  /** Debounce for text input before it becomes one undo step + PATCH (ms). */
  textDelay?: number;
}

/** Where the caret goes after Backspace at the start of a block. */
export type CaretTarget = { key: number; at: 'end' | number };

/** Server-computed fields taken from responses (never the text). */
const SERVER_FIELDS = ['assignee_id', 'assignee_name', 'due_date', 'done_at', 'done_by', 'done_by_name'] as const;
const TEXT_FIELDS = ['title', 'description'] as const;

export class BlockListModel<T extends Row> {
  private readonly history = new History();
  private readonly realIds = new Map<number, number>();
  private readonly saved = new Map<string, string>(); // last saved text per key:field
  private readonly timers = new Map<string, ReturnType<typeof setTimeout>>();
  private tempKey = -1;
  private queue: Promise<unknown> = Promise.resolve();

  constructor(private readonly cfg: BlockListConfig<T>) {}

  // ---- Queries ------------------------------------------------------------------

  get canUndo() {
    return this.history.canUndo;
  }
  get canRedo() {
    return this.history.canRedo;
  }

  /** Server ID of a block key (undefined while its create is still pending). */
  serverId(key: number): number | undefined {
    return this.realIds.get(key) ?? (key > 0 ? key : undefined);
  }

  /** True while text input waits for its debounced commit. */
  get pendingText() {
    return this.timers.size > 0;
  }

  /** Resolves once every queued server write (also ones queued meanwhile) has finished. */
  async settled(): Promise<void> {
    let q: Promise<unknown>;
    do {
      q = this.queue;
      await q;
    } while (q !== this.queue);
  }

  // ---- Text ---------------------------------------------------------------------

  /** Remembers rendered text as the saved baseline (unless the model already knows one). */
  baseline(key: number, field: Field, html: string) {
    const k = `${key}:${field}`;
    if (!this.saved.has(k)) this.saved.set(k, html);
  }

  /** Live text input: applied at once, committed (undo step + PATCH) after a pause. Returns the block. */
  text(key: number, field: Field, value: string): Block<T> | undefined {
    const b = this.find(key);
    if (!b) return;
    b[field] = value;
    const k = `${key}:${field}`;
    clearTimeout(this.timers.get(k));
    this.timers.set(k, setTimeout(() => this.commitText(key, field), this.cfg.textDelay ?? 700));
    return b;
  }

  /** Commits pending text of one field now (e.g. on blur). */
  commitText(key: number, field: Field) {
    const k = `${key}:${field}`;
    clearTimeout(this.timers.get(k));
    this.timers.delete(k);
    const b = this.find(key);
    if (!b) return;
    const before = this.saved.get(k) ?? '';
    const after = b[field];
    if (before === after) return;
    this.saved.set(k, after);
    this.history.push({ mergeKey: k, undo: () => this.setText(key, field, before), redo: () => this.setText(key, field, after) });
    this.persist(key, { [field]: after });
  }

  /** Commits all pending text input. */
  flush() {
    for (const k of [...this.timers.keys()]) {
      const [key, field] = k.split(':');
      this.commitText(Number(key), field as Field);
    }
  }

  // ---- Edits (each one undo step) ----------------------------------------------------

  update(key: number, patch: Partial<Block<T>>) {
    this.updateBlock(key, patch, true);
  }

  /** Inserts a block after afterKey (null = at the start) and returns its local key. */
  insert(afterKey: number | null, data: Partial<Block<T>>): number {
    return this.insertBlock(afterKey, data, {});
  }

  remove(key: number) {
    this.deleteBlock(key, true);
  }

  /** New order of all block keys (ignored unless it covers every block). */
  reorder(keys: number[]) {
    this.reorderBlocks(keys, true);
  }

  setDone(key: number, done: boolean) {
    this.setDoneBlock(key, done, true);
  }

  /** Tab / Shift+Tab: indent within 0..4. */
  indent(key: number, delta: number) {
    const b = this.find(key);
    if (!b) return;
    const indent = Math.max(0, Math.min(4, b.indent + delta));
    if (indent !== b.indent) this.update(key, { indent } as Partial<Block<T>>);
  }

  /** Enter in an empty list item: one level out, at level 0 back to text. */
  exitList(key: number) {
    const b = this.find(key);
    if (!b) return;
    if (b.indent > 0) this.update(key, { indent: b.indent - 1 } as Partial<Block<T>>);
    else this.update(key, { kind: 'text' } as Partial<Block<T>>);
  }

  /** Enter at the start of a non-empty block: new empty sibling above. */
  insertAbove(key: number): number | undefined {
    const b = this.find(key);
    if (!b) return;
    const i = this.indexOf(key);
    return this.insert(i > 0 ? this.list[i - 1].id : null, { kind: nextKind(b.kind), indent: b.indent } as Partial<Block<T>>);
  }

  /** Enter inside a block: the title keeps `before`, a new sibling below gets `after`. */
  split(key: number, before: string, after: string): number | undefined {
    const b = this.find(key);
    if (!b) return;
    let added: number | undefined;
    this.history.group(() => {
      this.text(key, 'title', before);
      this.commitText(key, 'title');
      added = this.insert(key, { kind: nextKind(b.kind), title: after, indent: b.indent } as Partial<Block<T>>);
    });
    return added;
  }

  /**
   * Backspace at the start of a block: back to text, outdent, remove a
   * divider above, or merge into the previous block. Returns where the caret
   * should go (null = stay).
   */
  backspaceAtStart(key: number): CaretTarget | null {
    const b = this.find(key);
    if (!b) return null;
    if (b.kind !== 'text') {
      this.update(key, { kind: 'text' } as Partial<Block<T>>);
      return null;
    }
    if (b.indent > 0) {
      this.update(key, { indent: b.indent - 1 } as Partial<Block<T>>);
      return null;
    }
    const i = this.indexOf(key);
    if (i <= 0) return null;
    const prev = this.list[i - 1];
    if (prev.kind === 'divider') {
      this.remove(prev.id);
      return null;
    }
    if (prev.kind === 'code') return { key: prev.id, at: 'end' };
    const offset = htmlToText(prev.title).length;
    this.history.group(() => {
      this.commitText(key, 'title');
      const patch = { title: prev.title + b.title } as Partial<Block<T>>;
      if (b.description && !prev.description) patch.description = b.description;
      this.updateBlock(prev.id, patch, true);
      this.deleteBlock(key, true);
    });
    return { key: prev.id, at: offset };
  }

  /**
   * Converts a block (clear = empty the text, e.g. after "/command").
   * A divider gets a new text block below; its key is returned as `inserted`.
   * Returns null when the block does not exist.
   */
  convert(key: number, kind: Kind, clear = false): { inserted?: number } | null {
    const b = this.find(key);
    if (!b) return null;
    const out: { inserted?: number } = {};
    this.history.group(() => {
      if (kind === 'divider') {
        this.update(key, { kind: 'divider', title: '' } as Partial<Block<T>>);
        out.inserted = this.insert(key, { kind: 'text', indent: b.indent } as Partial<Block<T>>);
        return;
      }
      let title = clear ? '' : b.title;
      if (!clear && kind === 'code' && b.kind !== 'code') title = htmlToText(b.title);
      if (!clear && b.kind === 'code' && kind !== 'code') title = escapeHtml(b.title).replace(/\n/g, '<br>');
      this.update(key, { kind, title } as Partial<Block<T>>);
    });
    return out;
  }

  /** Toolbar "add": new block after afterKey (headings unindented); a divider brings a text block along. Returns the key to focus. */
  add(afterKey: number | null, kind: Kind, indent: number): number {
    if (kind === 'divider') {
      let k = 0;
      this.history.group(() => {
        const d = this.insert(afterKey, { kind: 'divider' } as Partial<Block<T>>);
        k = this.insert(d, { kind: 'text' } as Partial<Block<T>>);
      });
      return k;
    }
    return this.insert(afterKey, { kind, indent: isHeading(kind) ? 0 : indent } as Partial<Block<T>>);
  }

  /**
   * Multiline paste into a title: the title becomes `title` (when given), each
   * further non-blank line a new block below. One undo step. Returns the last key.
   */
  pasteLines(key: number, title: string | undefined, lines: string[]): number {
    const b = this.find(key);
    let after = key;
    this.history.group(() => {
      if (title !== undefined) this.text(key, 'title', title);
      this.commitText(key, 'title');
      if (!b) return;
      for (const line of lines) {
        if (!line.trim()) continue;
        after = this.insert(after, { kind: nextKind(b.kind), title: escapeHtml(line), indent: b.indent } as Partial<Block<T>>);
      }
    });
    return after;
  }

  async undo() {
    this.flush();
    await this.history.undo();
  }

  async redo() {
    this.flush();
    await this.history.redo();
  }

  // ---- Internals ------------------------------------------------------------------

  private get list() {
    return this.cfg.items() as Block<T>[];
  }
  private find(key: number) {
    return this.list.find((b) => b.id === key);
  }
  private indexOf(key: number) {
    return this.list.findIndex((b) => b.id === key);
  }

  private enqueue(fn: () => Promise<unknown>) {
    this.queue = this.queue.then(fn).catch((e) => this.cfg.onError?.(e));
  }

  /** Applies server-computed fields (names, timestamps), never the text. */
  private mergeServer(key: number, res: Partial<Item>) {
    const b = this.find(key);
    if (!b) return;
    for (const f of SERVER_FIELDS) {
      if (f in res) (b as unknown as Record<string, unknown>)[f] = res[f];
    }
  }

  private persist(key: number, body: Record<string, unknown>) {
    this.enqueue(async () => {
      const id = this.serverId(key);
      if (id == null || !this.find(key)) return;
      this.mergeServer(key, (await this.cfg.ops().patch(id, body)) as Partial<Item>);
    });
  }

  private setText(key: number, field: Field, value: string) {
    const b = this.find(key);
    if (!b) return;
    b[field] = value;
    this.saved.set(`${key}:${field}`, value);
    this.persist(key, { [field]: value });
    this.cfg.onChange?.();
  }

  private updateBlock(key: number, patch: Partial<Block<T>>, record: boolean) {
    const b = this.find(key);
    if (!b) return;
    for (const f of TEXT_FIELDS) {
      if (f in patch) {
        clearTimeout(this.timers.get(`${key}:${f}`));
        this.timers.delete(`${key}:${f}`);
      } else this.commitText(key, f);
    }
    const before: Record<string, unknown> = {};
    for (const k of Object.keys(patch)) before[k] = (b as unknown as Record<string, unknown>)[k];
    Object.assign(b, patch);
    for (const f of TEXT_FIELDS) if (f in patch) this.saved.set(`${key}:${f}`, b[f]);
    if (record)
      this.history.push({
        undo: () => this.updateBlock(key, before as Partial<Block<T>>, false),
        redo: () => this.updateBlock(key, patch, false)
      });
    this.persist(key, patch as Record<string, unknown>);
    this.cfg.onChange?.();
  }

  private snapshot(b: Block<T>) {
    return {
      kind: b.kind, title: b.title, description: b.description, note_color: b.note_color, bg_color: b.bg_color ?? '', indent: b.indent, lang: b.lang,
      assignee_id: b.assignee_id ?? null, assignee_name: b.assignee_name ?? null, due_date: b.due_date ?? null,
      done_at: b.done_at ?? null, done_by: b.done_by ?? null, done_by_name: b.done_by_name ?? null, status: b.status ?? ''
    } as Partial<Block<T>>;
  }

  private insertBlock(afterKey: number | null, data: Partial<Block<T>>, opts: { key?: number; record?: boolean }): number {
    const items = this.cfg.items();
    const blocks = this.list;
    const isChecklist = !!this.cfg.ops().toggle;
    const key = opts.key ?? this.tempKey--;
    const ai = afterKey == null ? -1 : this.indexOf(afterKey);
    const after = ai >= 0 ? blocks[ai] : null;
    const position = after ? after.position + 1 : (blocks[0]?.position ?? 0);
    for (const b of blocks) if (b.position >= position) b.position++;
    const blk = {
      id: key, kind: 'text', title: '', description: '', note_color: '', bg_color: '', indent: 0, lang: '', position,
      assignee_id: null, assignee_name: null, due_date: null, done_at: null, done_by: null, done_by_name: null, status: '',
      ...data
    } as unknown as T;
    items.splice(ai + 1, 0, blk);
    this.saved.set(`${key}:title`, blk.title);
    this.saved.set(`${key}:description`, blk.description);

    const body: Record<string, unknown> = {
      kind: blk.kind, title: blk.title, description: blk.description, note_color: blk.note_color, bg_color: blk.bg_color ?? '',
      indent: blk.indent, lang: blk.lang, position
    };
    const d = data as Partial<Item>;
    if (isChecklist && d.assignee_id) body.assignee_id = d.assignee_id;
    if (isChecklist && d.due_date) body.due_date = d.due_date;
    if (isChecklist && d.status) body.status = d.status;
    this.enqueue(async () => {
      const ops = this.cfg.ops();
      const res = await ops.create(body);
      this.realIds.set(key, res.id);
      if (d.done_at && ops.toggle) this.mergeServer(key, (await ops.toggle(res.id, true)) as Partial<Item>);
    });

    if (opts.record !== false) {
      let snap: { after: number | null; data: Partial<Block<T>> } | null = null;
      this.history.push({
        undo: () => {
          const b = this.find(key);
          if (!b) return;
          const i = this.indexOf(key);
          snap = { after: i > 0 ? this.list[i - 1].id : null, data: this.snapshot(b) };
          this.deleteBlock(key, false);
        },
        redo: () => {
          if (snap) this.insertBlock(snap.after, snap.data, { key, record: false });
        }
      });
    }
    this.cfg.onChange?.();
    return key;
  }

  private deleteBlock(key: number, record: boolean) {
    const i = this.indexOf(key);
    if (i < 0) return;
    for (const f of TEXT_FIELDS) {
      clearTimeout(this.timers.get(`${key}:${f}`));
      this.timers.delete(`${key}:${f}`);
    }
    const b = this.list[i];
    const afterKey = i > 0 ? this.list[i - 1].id : null;
    const data = this.snapshot(b);
    this.cfg.items().splice(i, 1);
    this.cfg.onRemove?.(key);
    this.enqueue(async () => {
      const id = this.serverId(key);
      this.realIds.delete(key);
      if (id != null) await this.cfg.ops().remove(id);
    });
    if (record)
      this.history.push({
        undo: () => {
          this.insertBlock(afterKey, data, { key, record: false });
        },
        redo: () => this.deleteBlock(key, false)
      });
    this.cfg.onChange?.();
  }

  private reorderBlocks(keys: number[], record: boolean) {
    const items = this.cfg.items();
    const prev = items.map((b) => b.id);
    const byKey = new Map(items.map((b) => [b.id, b]));
    const next = keys.map((k) => byKey.get(k)).filter((b): b is T => !!b);
    if (next.length !== items.length) return;
    next.forEach((b, i) => (b.position = i));
    this.cfg.setItems(next);
    this.enqueue(async () => {
      const ids = keys.map((k) => this.serverId(k));
      if (ids.some((id) => id == null)) return;
      await this.cfg.ops().reorder(ids as number[]);
    });
    if (record) this.history.push({ undo: () => this.reorderBlocks(prev, false), redo: () => this.reorderBlocks(keys, false) });
  }

  private setDoneBlock(key: number, done: boolean, record: boolean) {
    const b = this.find(key);
    if (!b || !this.cfg.ops().toggle || b.kind !== 'item') return;
    if (!!b.done_at === done) return;
    const user = this.cfg.user?.();
    b.done_at = done ? new Date().toISOString() : null;
    b.done_by = done ? (user?.id ?? null) : null;
    b.done_by_name = done ? (user?.name ?? null) : null;
    this.enqueue(async () => {
      const id = this.serverId(key);
      if (id != null) this.mergeServer(key, (await this.cfg.ops().toggle!(id, done)) as Partial<Item>);
    });
    if (record) this.history.push({ undo: () => this.setDoneBlock(key, !done, false), redo: () => this.setDoneBlock(key, done, false) });
    this.cfg.onChange?.();
  }
}
