<script lang="ts" generics="T extends Row">
  // Block-Editor für Checklisten und Templates.
  //
  // Jeder Block ist eine Zeile (eigene DB-Zeile) mit contenteditable-Titel und
  // einer Notiz in der rechten Spalte. Lokaler Zustand ist maßgeblich; alle
  // Schreibzugriffe laufen der Reihe nach über eine Warteschlange, damit
  // Positionen und neu angelegte IDs konsistent bleiben. Neue Blöcke bekommen
  // sofort eine negative lokale ID (Fokus/Tippen ohne Wartezeit); die
  // Server-ID wird in realIds nachgetragen.
  import { tick } from 'svelte';
  import { errorText } from '$lib/api';
  import { checkRadius, prefs, session, toast } from '$lib/state.svelte';
  import { fmtDate, fmtDateTime, shortName, t, tMaybe, todayISO } from '$lib/i18n.svelte';
  import { sortable } from '$lib/sortable';
  import { History } from '$lib/history.svelte';
  import { SvelteSet } from 'svelte/reactivity';
  import { colorLabel, HUES, ROW_BG, SHADES, SWATCH } from '$lib/palette';
  import { filterKinds, headingLevel, headingStats, hiddenBlocks, isHeading, sectionColors, isTextual, nextKind, numbering, SHORTCUTS } from '$lib/blocks';
  import {
    caretAtEnd,
    caretAtStart,
    caretOnFirstLine,
    caretOnLastLine,
    clearFormatting,
    escapeHtml,
    extractAfterCaret,
    htmlToText,
    insertPlainText,
    notifyInput,
    placeCaret,
    sanitizeHtml,
    selectionIn,
    selectionWithin,
    unwrapInSelection,
    wrapSelection
  } from '$lib/rich';
  import type { Assignee, EditorApi, Item, Kind, NoteColor, Ops, Row } from '$lib/types';
  import EditorToolbar from './EditorToolbar.svelte';
  import SlashMenu from './SlashMenu.svelte';
  import CodeBlock from './CodeBlock.svelte';
  import ConfirmButton from './ConfirmButton.svelte';
  import Icon from './Icon.svelte';

  type Blk = T & Partial<Item>;
  type Field = 'title' | 'description';

  let {
    items = $bindable(),
    ops,
    editable,
    assignees = [],
    onchange,
    onerror,
    collapseKey
  }: {
    items: T[];
    ops: Ops<T>;
    editable: boolean;
    assignees?: Assignee[];
    onchange?: () => void;
    onerror?: () => void;
    /** Schlüssel für den lokal gespeicherten Einklapp-Zustand (z. B. "checklist:1"). */
    collapseKey?: string;
  } = $props();

  const isChecklist = $derived(!!ops.toggle);
  const today = todayISO();
  const history = new History();

  const blocks = $derived(items as Blk[]);
  const nums = $derived(numbering(items));
  const hStats = $derived(headingStats(items));

  // ---- Abschnittsfarben ---------------------------------------------------------
  // Die Farbe hängt an der Überschrift (bg_color) und gilt für ihren Abschnitt.
  // Abgerundet wird nur am Anfang/Ende eines farbigen Bereichs (sichtbare Zeilen).

  const secColors = $derived(sectionColors(items));
  // Farbe, die eine Zeile ohne eigene Farbe vom Abschnitt erben würde (für "Wie Abschnitt")
  const inheritedColors = $derived(sectionColors(items.map((b) => (isHeading(b.kind) ? b : { ...b, bg_color: '' as const }))));
  // Ränder (Abstand, Rundung) nur an den Grenzen farbiger Abschnitte (Überschriften).
  // Einzeln gefärbte Zeilen ändern nur den Hintergrund und schließen lückenlos an.
  const secEdges = $derived.by(() => {
    const vis = blocks.filter((b) => !hidden.has(b.id));
    const out = new Map<number, { first: boolean; last: boolean }>();
    vis.forEach((b, i) => {
      const r = inheritedColors.get(b.id)?.region;
      if (r == null) return;
      out.set(b.id, {
        first: inheritedColors.get(vis[i - 1]?.id)?.region !== r,
        last: inheritedColors.get(vis[i + 1]?.id)?.region !== r
      });
    });
    return out;
  });
  let bgMenu = $state<number | null>(null);

  function sectionRowCls(key: number): string {
    const sec = secColors.get(key); // tatsächliche Farbe (eigene Zeilenfarbe oder Abschnitt)
    const e = secEdges.get(key); // Lage im farbigen Abschnitt
    const bg = sec ? (ROW_BG[sec.color] ?? '') : '';
    const g = grid.get(key);
    // Rasterzeilen: eckig, Abstand am Abschnittsende außen statt innen (sonst wird die Zelle höher)
    if (g) return `${bg} ${gridCls} ${g.joinPrev ? '-mt-px' : ''} ${e?.last ? 'mb-1.5' : ''}`;
    if (!e) return `${bg} rounded-md`;
    const radius = e.first && e.last ? 'rounded-md' : e.first ? 'rounded-t-md' : e.last ? 'rounded-b-md' : '';
    return `${bg} ${radius} ${e.first ? 'mt-1' : ''} ${e.last ? 'mb-1 pb-1.5' : ''}`;
  }

  // ---- Rasterlinien -------------------------------------------------------------
  // Mit prefs.gridLines bekommt jede Inhaltszeile (nicht Überschriften/Trennlinien)
  // einen Rahmen; aufeinanderfolgende Zeilen teilen sich die Linie (-mt-px).
  const gridCls = 'border border-zinc-300 dark:border-zinc-700';
  const isCell = (b: Blk) => !isHeading(b.kind) && b.kind !== 'divider';
  const grid = $derived.by(() => {
    const out = new Map<number, { joinPrev: boolean }>();
    if (!prefs.gridLines) return out;
    let prevCell = false;
    for (const b of blocks) {
      if (hidden.has(b.id)) continue;
      const cell = isCell(b);
      if (cell) out.set(b.id, { joinPrev: prevCell });
      prevCell = cell;
    }
    return out;
  });

  // ---- Status (Freitext rechts neben der Checkbox) -----------------------------
  const statusOptions = $derived.by(() => {
    const set = new Set<string>(t('checklist.statusSuggestions').split('|'));
    for (const b of blocks) if (b.status) set.add(b.status);
    return [...set];
  });
  const listId = `status-${Math.random().toString(36).slice(2, 8)}`;

  // ---- Einklappen ---------------------------------------------------------------
  // Pro Benutzer/Browser in localStorage, nicht in der Datenbank: Einklappen ist
  // eine Ansichtssache und soll andere Bearbeiter nicht beeinflussen.

  const collapsed = new SvelteSet<number>(loadCollapsed());
  const hidden = $derived(hiddenBlocks(items, collapsed));
  const headingIds = $derived(blocks.filter((b) => isHeading(b.kind)).map((b) => b.id));
  const anyExpanded = $derived(headingIds.some((id) => !collapsed.has(id)));

  function loadCollapsed(): number[] {
    if (!collapseKey) return [];
    try {
      const v = JSON.parse(localStorage.getItem(`collapsed:${collapseKey}`) ?? '[]');
      return Array.isArray(v) ? v.filter((n) => typeof n === 'number') : [];
    } catch {
      return [];
    }
  }

  function saveCollapsed() {
    if (!collapseKey) return;
    const ids = [...collapsed].map((k) => sid(k)).filter((id): id is number => id != null);
    try {
      if (ids.length) localStorage.setItem(`collapsed:${collapseKey}`, JSON.stringify(ids));
      else localStorage.removeItem(`collapsed:${collapseKey}`);
    } catch {
      /* privater Modus o. Ä. */
    }
  }

  function toggleCollapse(key: number) {
    if (collapsed.has(key)) collapsed.delete(key);
    else collapsed.add(key);
    // Lag der Fokus in einem jetzt verborgenen Block, wandert er auf die Überschrift.
    const row = (document.activeElement as HTMLElement | null)?.closest('[data-id]') as HTMLElement | null;
    if (row && hiddenBlocks(items, collapsed).has(Number(row.dataset.id))) focusBlock(key, 'end');
    saveCollapsed();
  }

  function setAllCollapsed(on: boolean) {
    collapsed.clear();
    if (on) for (const id of headingIds) collapsed.add(id);
    saveCollapsed();
  }

  /** Klappt alle Überschriften auf, unter denen der Block verborgen ist. */
  function reveal(key: number) {
    let i = indexOf(key);
    if (i < 0) return;
    let lvl = headingLevel(blocks[i].kind) || 5;
    let changed = false;
    for (i--; i >= 0 && lvl > 1; i--) {
      const l = headingLevel(blocks[i].kind);
      if (l && l < lvl) {
        if (collapsed.delete(blocks[i].id)) changed = true;
        lvl = l;
      }
    }
    if (changed) saveCollapsed();
  }

  /** Drag & Drop liefert nur sichtbare IDs; verborgene Blöcke wandern mit ihrer Überschrift. */
  function reorderVisible(visible: number[]) {
    const groups = new Map<number, number[]>();
    let cur: number[] | null = null;
    for (const b of blocks) {
      if (!hidden.has(b.id) || !cur) {
        cur = [b.id];
        groups.set(b.id, cur);
      } else cur.push(b.id);
    }
    reorder(visible.flatMap((id) => groups.get(id) ?? []));
  }

  // ---- Zustand ----------------------------------------------------------------

  let focused = $state<{ key: number; field: 'title' | 'note' } | null>(null);
  let slash = $state<{ key: number; query: string; active: number } | null>(null);
  let editField = $state<{ key: number; field: 'assignee' | 'due' } | null>(null);
  let marks = $state({ bold: false, italic: false, underline: false, strike: false, code: false });
  let root = $state<HTMLElement>();

  const titleEls = new Map<number, HTMLElement>();
  const noteEls = new Map<number, HTMLElement>();
  const codeEls = new Map<number, HTMLTextAreaElement>();
  const realIds = new Map<number, number>();
  const saved = new Map<string, string>(); // zuletzt gespeicherter Text je key:field
  const timers = new Map<string, ReturnType<typeof setTimeout>>();
  let tempKey = -1;
  let savedRange: Range | null = null;

  const slashList = $derived(slash ? filterKinds(slash.query, t) : []);

  // ---- Warteschlange & Server -------------------------------------------------

  let queue: Promise<unknown> = Promise.resolve();
  function enqueue(fn: () => Promise<unknown>) {
    queue = queue.then(fn).catch((e) => {
      toast(errorText(e));
      onerror?.();
    });
  }

  function sid(key: number): number | undefined {
    return realIds.get(key) ?? (key > 0 ? key : undefined);
  }

  const find = (key: number) => blocks.find((b) => b.id === key);
  const indexOf = (key: number) => blocks.findIndex((b) => b.id === key);

  /** Übernimmt serverseitig berechnete Felder (Namen, Zeitstempel), nie den Text. */
  function mergeServer(key: number, res: Partial<Item>) {
    const b = find(key);
    if (!b) return;
    for (const f of ['assignee_id', 'assignee_name', 'due_date', 'done_at', 'done_by', 'done_by_name'] as const) {
      if (f in res) (b as unknown as Record<string, unknown>)[f] = res[f];
    }
  }

  function persist(key: number, body: Record<string, unknown>) {
    enqueue(async () => {
      const id = sid(key);
      if (id == null || !find(key)) return;
      mergeServer(key, (await ops.patch(id, body)) as Partial<Item>);
    });
  }

  // ---- Text -------------------------------------------------------------------

  function onText(key: number, field: Field, value: string, el?: HTMLElement) {
    const b = find(key);
    if (!b) return;
    b[field] = value;
    const k = `${key}:${field}`;
    clearTimeout(timers.get(k));
    timers.set(k, setTimeout(() => commitText(key, field), 700));
    if (field === 'title' && el) afterTitleInput(b, el);
  }

  function commitText(key: number, field: Field) {
    const k = `${key}:${field}`;
    clearTimeout(timers.get(k));
    timers.delete(k);
    const b = find(key);
    if (!b) return;
    const before = saved.get(k) ?? '';
    const after = b[field];
    if (before === after) return;
    saved.set(k, after);
    history.push({ mergeKey: k, undo: () => setText(key, field, before), redo: () => setText(key, field, after) });
    persist(key, { [field]: after });
  }

  function setText(key: number, field: Field, value: string) {
    const b = find(key);
    if (!b) return;
    b[field] = value;
    saved.set(`${key}:${field}`, value);
    persist(key, { [field]: value });
    onchange?.();
  }

  function flushAll() {
    for (const k of [...timers.keys()]) {
      const [key, field] = k.split(':');
      commitText(Number(key), field as Field);
    }
  }

  // ---- Block-Operationen (mit Undo) -------------------------------------------

  function updateBlock(key: number, patch: Partial<Blk>, record = true) {
    const b = find(key);
    if (!b) return;
    for (const f of ['title', 'description'] as const) {
      if (f in patch) {
        clearTimeout(timers.get(`${key}:${f}`));
        timers.delete(`${key}:${f}`);
      } else commitText(key, f);
    }
    const before: Record<string, unknown> = {};
    for (const k of Object.keys(patch)) before[k] = (b as unknown as Record<string, unknown>)[k];
    Object.assign(b, patch);
    for (const f of ['title', 'description'] as const) if (f in patch) saved.set(`${key}:${f}`, b[f]);
    if (record) history.push({ undo: () => updateBlock(key, before as Partial<Blk>, false), redo: () => updateBlock(key, patch, false) });
    persist(key, patch as Record<string, unknown>);
    onchange?.();
  }

  function snapshot(b: Blk) {
    return {
      kind: b.kind, title: b.title, description: b.description, note_color: b.note_color, bg_color: b.bg_color ?? '', indent: b.indent, lang: b.lang,
      assignee_id: b.assignee_id ?? null, assignee_name: b.assignee_name ?? null, due_date: b.due_date ?? null,
      done_at: b.done_at ?? null, done_by: b.done_by ?? null, done_by_name: b.done_by_name ?? null, status: b.status ?? ''
    } as Partial<Blk>;
  }

  /** Fügt einen Block nach afterKey ein (null = an den Anfang) und gibt die lokale ID zurück. */
  function insertBlock(afterKey: number | null, data: Partial<Blk>, opts: { key?: number; record?: boolean; focus?: 'start' | 'end' } = {}) {
    const key = opts.key ?? tempKey--;
    const ai = afterKey == null ? -1 : indexOf(afterKey);
    const after = ai >= 0 ? blocks[ai] : null;
    const position = after ? after.position + 1 : (blocks[0]?.position ?? 0);
    for (const b of blocks) if (b.position >= position) b.position++;
    const blk = {
      id: key, kind: 'text', title: '', description: '', note_color: '', bg_color: '', indent: 0, lang: '', position,
      assignee_id: null, assignee_name: null, due_date: null, done_at: null, done_by: null, done_by_name: null, status: '',
      ...data
    } as unknown as T;
    items.splice(ai + 1, 0, blk);
    saved.set(`${key}:title`, blk.title);
    saved.set(`${key}:description`, blk.description);

    const body: Record<string, unknown> = {
      kind: blk.kind, title: blk.title, description: blk.description, note_color: blk.note_color, bg_color: blk.bg_color ?? '',
      indent: blk.indent, lang: blk.lang, position
    };
    const d = data as Partial<Item>;
    if (isChecklist && d.assignee_id) body.assignee_id = d.assignee_id;
    if (isChecklist && d.due_date) body.due_date = d.due_date;
    if (isChecklist && d.status) body.status = d.status;
    enqueue(async () => {
      const res = await ops.create(body);
      realIds.set(key, res.id);
      if (d.done_at && ops.toggle) mergeServer(key, (await ops.toggle(res.id, true)) as Partial<Item>);
    });

    if (opts.record !== false) {
      let snap: { after: number | null; data: Partial<Blk> } | null = null;
      history.push({
        undo: () => {
          const b = find(key);
          if (!b) return;
          const i = indexOf(key);
          snap = { after: i > 0 ? blocks[i - 1].id : null, data: snapshot(b) };
          deleteBlock(key, false);
        },
        redo: () => {
          if (snap) insertBlock(snap.after, snap.data, { key, record: false });
        }
      });
    }
    if (opts.focus) focusBlock(key, opts.focus);
    onchange?.();
    return key;
  }

  function deleteBlock(key: number, record = true) {
    const i = indexOf(key);
    if (i < 0) return;
    for (const f of ['title', 'description'] as const) {
      clearTimeout(timers.get(`${key}:${f}`));
      timers.delete(`${key}:${f}`);
    }
    const b = blocks[i];
    const afterKey = i > 0 ? blocks[i - 1].id : null;
    const data = snapshot(b);
    items.splice(i, 1);
    if (slash?.key === key) slash = null;
    enqueue(async () => {
      const id = sid(key);
      realIds.delete(key);
      if (id != null) await ops.remove(id);
    });
    if (record)
      history.push({
        undo: () => {
          insertBlock(afterKey, data, { key, record: false });
        },
        redo: () => deleteBlock(key, false)
      });
    onchange?.();
  }

  function reorder(keys: number[], record = true) {
    const prev = blocks.map((b) => b.id);
    const byKey = new Map(items.map((b) => [b.id, b]));
    const next = keys.map((k) => byKey.get(k)).filter((b): b is T => !!b);
    if (next.length !== items.length) return;
    next.forEach((b, i) => (b.position = i));
    items = next;
    enqueue(async () => {
      const ids = keys.map(sid);
      if (ids.some((id) => id == null)) return;
      await ops.reorder(ids as number[]);
    });
    if (record) history.push({ undo: () => reorder(prev, false), redo: () => reorder(keys, false) });
  }

  function setDone(key: number, done: boolean, record = true) {
    const b = find(key);
    if (!b || !ops.toggle || b.kind !== 'item') return;
    if (!!b.done_at === done) return;
    b.done_at = done ? new Date().toISOString() : null;
    b.done_by = done ? (session.user?.id ?? null) : null;
    b.done_by_name = done ? (session.user?.name ?? null) : null;
    enqueue(async () => {
      const id = sid(key);
      if (id != null) mergeServer(key, (await ops.toggle!(id, done)) as Partial<Item>);
    });
    if (record) history.push({ undo: () => setDone(key, !done, false), redo: () => setDone(key, done, false) });
    onchange?.();
  }

  async function undo() {
    flushAll();
    await history.undo();
  }
  async function redo() {
    flushAll();
    await history.redo();
  }

  // ---- Fokus & Navigation -----------------------------------------------------

  async function focusBlock(key: number, where: 'start' | 'end' | number, field: 'title' | 'note' = 'title') {
    reveal(key);
    await tick();
    const b = find(key);
    if (!b) return;
    if (b.kind === 'code' && field === 'title') {
      const ta = codeEls.get(key);
      if (ta) {
        ta.focus();
        const pos = where === 'start' ? 0 : ta.value.length;
        ta.setSelectionRange(pos, pos);
      }
      return;
    }
    const el = field === 'title' ? titleEls.get(key) : noteEls.get(key);
    if (el) placeCaret(el, where === 'start' ? 0 : where === 'end' ? -1 : where);
  }

  function neighbour(key: number, dir: -1 | 1) {
    let i = indexOf(key) + dir;
    while (i >= 0 && i < blocks.length) {
      if (blocks[i].kind !== 'divider' && !hidden.has(blocks[i].id)) return blocks[i];
      i += dir;
    }
    return null;
  }

  // ---- Tastatur im Titel --------------------------------------------------------

  function onTitleKey(e: KeyboardEvent, b: Blk) {
    const el = e.currentTarget as HTMLElement;
    const key = b.id;

    if (slash?.key === key) {
      if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
        e.preventDefault();
        const n = slashList.length;
        if (n) slash.active = (slash.active + (e.key === 'ArrowDown' ? 1 : n - 1)) % n;
        return;
      }
      if (e.key === 'Enter' || e.key === 'Tab') {
        e.preventDefault();
        const k = slashList[slash.active];
        if (k) chooseSlash(k.kind);
        return;
      }
      if (e.key === 'Escape') {
        e.preventDefault();
        slash = null;
        return;
      }
    }

    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      splitBlock(b, el);
    } else if (e.key === 'Backspace' && caretAtStart(el) && selectionIn(el)?.collapsed) {
      e.preventDefault();
      backspaceAtStart(b);
    } else if (e.key === 'Tab') {
      e.preventDefault();
      const indent = Math.max(0, Math.min(4, b.indent + (e.shiftKey ? -1 : 1)));
      if (indent !== b.indent) updateBlock(key, { indent } as Partial<Blk>);
    } else if (e.key === 'ArrowUp' && caretOnFirstLine(el)) {
      const p = neighbour(key, -1);
      if (p) {
        e.preventDefault();
        focusBlock(p.id, 'end');
      }
    } else if (e.key === 'ArrowDown' && caretOnLastLine(el)) {
      const n = neighbour(key, 1);
      if (n) {
        e.preventDefault();
        focusBlock(n.id, 'start');
      }
    } else if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'e') {
      e.preventDefault();
      format('code');
    }
  }

  function splitBlock(b: Blk, el: HTMLElement) {
    const key = b.id;
    const listy = b.kind === 'item' || b.kind === 'bullet' || b.kind === 'numbered' || b.kind === 'quote';
    if (listy && !el.textContent) {
      // Leerer Listeneintrag + Enter → Liste verlassen
      if (b.indent > 0) updateBlock(key, { indent: b.indent - 1 } as Partial<Blk>);
      else updateBlock(key, { kind: 'text' } as Partial<Blk>);
      return;
    }
    if (caretAtStart(el) && el.textContent) {
      const i = indexOf(key);
      insertBlock(i > 0 ? blocks[i - 1].id : null, { kind: nextKind(b.kind), indent: b.indent } as Partial<Blk>);
      return;
    }
    history.group(() => {
      const after = extractAfterCaret(el);
      onText(key, 'title', sanitizeHtml(el.innerHTML));
      commitText(key, 'title');
      insertBlock(key, { kind: nextKind(b.kind), title: after, indent: b.indent } as Partial<Blk>, { focus: 'start' });
    });
  }

  function backspaceAtStart(b: Blk) {
    const key = b.id;
    if (b.kind !== 'text') {
      updateBlock(key, { kind: 'text' } as Partial<Blk>);
      return;
    }
    if (b.indent > 0) {
      updateBlock(key, { indent: b.indent - 1 } as Partial<Blk>);
      return;
    }
    const i = indexOf(key);
    if (i <= 0) return;
    const prev = blocks[i - 1];
    if (prev.kind === 'divider') {
      deleteBlock(prev.id);
      return;
    }
    if (prev.kind === 'code') {
      focusBlock(prev.id, 'end');
      return;
    }
    const offset = htmlToText(prev.title).length;
    history.group(() => {
      commitText(key, 'title');
      const patch: Partial<Blk> = { title: prev.title + b.title } as Partial<Blk>;
      if (b.description && !prev.description) patch.description = b.description;
      updateBlock(prev.id, patch);
      deleteBlock(key);
    });
    focusBlock(prev.id, offset);
  }

  /** Slash-Menü und Markdown-Kürzel nach einer Eingabe im Titel. */
  function afterTitleInput(b: Blk, el: HTMLElement) {
    const text = el.textContent ?? '';
    if (text.startsWith('/') && !/\s/.test(text) && text.length <= 20) {
      slash = { key: b.id, query: text.slice(1), active: slash?.key === b.id ? slash.active : 0 };
      return;
    }
    if (slash?.key === b.id) slash = null;
    if (b.kind !== 'text') return;
    for (const [re, kind] of SHORTCUTS) {
      if (re.test(text)) {
        convert(b.id, kind, true);
        return;
      }
    }
  }

  function chooseSlash(kind: Kind) {
    if (!slash) return;
    const key = slash.key;
    slash = null;
    convert(key, kind, true);
  }

  /** Wandelt einen Block um (clear = Text leeren, z. B. nach "/befehl"). */
  function convert(key: number, kind: Kind, clear = false) {
    const b = find(key);
    if (!b) return;
    history.group(() => {
      if (kind === 'divider') {
        updateBlock(key, { kind: 'divider', title: '' } as Partial<Blk>);
        insertBlock(key, { kind: 'text', indent: b.indent } as Partial<Blk>, { focus: 'start' });
        return;
      }
      let title = clear ? '' : b.title;
      if (!clear && kind === 'code' && b.kind !== 'code') title = htmlToText(b.title);
      if (!clear && b.kind === 'code' && kind !== 'code') title = escapeHtml(b.title).replace(/\n/g, '<br>');
      updateBlock(key, { kind, title } as Partial<Blk>);
    });
    focusBlock(key, 'end');
  }

  // ---- Einfügen aus der Zwischenablage ---------------------------------------

  function onPaste(e: ClipboardEvent, b: Blk, field: 'title' | 'note') {
    e.preventDefault();
    const text = e.clipboardData?.getData('text/plain') ?? '';
    const lines = text.replace(/\r\n?/g, '\n').split('\n');
    if (field === 'note' || lines.length === 1) {
      insertPlainText(text);
      return;
    }
    // Mehrzeilig in den Titel: erste Zeile hier, weitere Zeilen als neue Blöcke.
    history.group(() => {
      insertPlainText(lines[0]);
      const el = titleEls.get(b.id);
      if (el) onText(b.id, 'title', sanitizeHtml(el.innerHTML));
      commitText(b.id, 'title');
      let after = b.id;
      for (const line of lines.slice(1)) {
        if (!line.trim()) continue;
        after = insertBlock(after, { kind: nextKind(b.kind), title: escapeHtml(line), indent: b.indent } as Partial<Blk>);
      }
      focusBlock(after, 'end');
    });
  }

  // ---- Inline-Formatierung (Toolbar) ------------------------------------------

  function activeEl(): HTMLElement | null {
    if (!focused) return null;
    const el = focused.field === 'title' ? titleEls.get(focused.key) : noteEls.get(focused.key);
    return el && el.isContentEditable ? el : null;
  }

  function updateMarks() {
    const el = activeEl();
    if (!el || !selectionIn(el)) return;
    marks = {
      bold: document.queryCommandState('bold'),
      italic: document.queryCommandState('italic'),
      underline: document.queryCommandState('underline'),
      strike: document.queryCommandState('strikeThrough'),
      code: selectionWithin(el, 'code')
    };
  }

  function format(cmd: 'bold' | 'italic' | 'underline' | 'strikeThrough' | 'code') {
    const el = activeEl();
    if (!el) return;
    if (cmd === 'code') {
      if (selectionWithin(el, 'code')) unwrapInSelection(el, 'code');
      else wrapSelection(el, 'code', {}, 'code');
    } else {
      document.execCommand('styleWithCSS', false, 'false');
      document.execCommand(cmd);
    }
    notifyInput(el);
    updateMarks();
  }

  function color(c: string) {
    const el = activeEl();
    if (!el) return;
    if (c) wrapSelection(el, 'span', { 'data-color': c }, 'span[data-color]');
    else unwrapInSelection(el, 'span[data-color]');
    notifyInput(el);
  }

  function highlight(c: string) {
    const el = activeEl();
    if (!el) return;
    if (c) wrapSelection(el, 'mark', { 'data-bg': c }, 'mark');
    else unwrapInSelection(el, 'mark');
    notifyInput(el);
  }

  function saveSelection() {
    const el = activeEl();
    const r = el ? selectionIn(el) : null;
    savedRange = r ? r.cloneRange() : null;
  }

  // ---- Links: Text und Adresse bearbeiten ----------------------------------------
  // Klick auf einen Link (oder der Link-Button) öffnet einen kleinen Dialog mit
  // Text + URL. Ohne Auswahl wird ein neuer Link an der Cursorposition eingefügt.

  let linkEdit = $state<{
    el: HTMLElement;
    a: HTMLAnchorElement | null;
    range: Range | null;
    text: string;
    url: string;
    x: number;
    y: number;
  } | null>(null);

  /** Ohne Schema: E-Mail-Adresse → mailto:, sonst https:// (nur http(s)/mailto sind erlaubt). */
  function normalizeUrl(u: string) {
    u = u.trim();
    if (!u) return '';
    if (/^(https?:\/\/|mailto:)/i.test(u)) return u;
    return (u.includes('@') && !u.includes('/') ? 'mailto:' : 'https://') + u.replace(/^\/+/, '');
  }

  function openLinkEditor(el: HTMLElement, a: HTMLAnchorElement | null, range: Range | null) {
    const rect = (a ?? range)?.getBoundingClientRect();
    const box = rect && rect.height ? rect : el.getBoundingClientRect();
    const w = 340;
    linkEdit = {
      el,
      a,
      range,
      text: a ? (a.textContent ?? '') : (range?.toString() ?? ''),
      url: a ? (a.getAttribute('href') ?? '') : '',
      x: Math.max(8, Math.min(box.left, innerWidth - w - 8)),
      y: Math.min(box.bottom + 6, innerHeight - 200)
    };
  }

  /** Toolbar: bestehenden Link unter dem Cursor bearbeiten oder neuen anlegen. */
  function openLink() {
    const el = activeEl();
    if (!el) return;
    const r = selectionIn(el);
    let n: Node | null = r?.commonAncestorContainer ?? null;
    while (n && n !== el && !(n instanceof HTMLAnchorElement)) n = n.parentNode;
    openLinkEditor(el, n instanceof HTMLAnchorElement ? n : null, r ? r.cloneRange() : null);
  }

  function applyLink(remove = false) {
    if (!linkEdit) return;
    const { el, a } = linkEdit;
    const url = remove ? '' : normalizeUrl(linkEdit.url);
    const text = linkEdit.text.trim();
    let caretAfter: Node | null = null;
    if (a) {
      if (!url) {
        caretAfter = a.lastChild;
        a.replaceWith(...a.childNodes);
      } else {
        a.setAttribute('href', url);
        if (text && text !== a.textContent) a.textContent = text;
        caretAfter = a;
      }
    } else if (url) {
      let r = linkEdit.range;
      if (!r || !el.contains(r.commonAncestorContainer)) {
        r = document.createRange();
        r.selectNodeContents(el);
        r.collapse(false);
      }
      const na = document.createElement('a');
      na.setAttribute('href', url);
      if (!r.collapsed && (!text || text === r.toString())) na.appendChild(r.extractContents());
      else {
        r.deleteContents();
        na.textContent = text || url.replace(/^mailto:/i, '');
      }
      na.querySelectorAll('a').forEach((x) => x.replaceWith(...x.childNodes)); // keine verschachtelten Links
      r.insertNode(na);
      caretAfter = na;
    }
    linkEdit = null;
    el.focus();
    if (caretAfter && el.contains(caretAfter)) {
      const r = document.createRange();
      r.setStartAfter(caretAfter);
      r.collapse(true);
      const sel = getSelection()!;
      sel.removeAllRanges();
      sel.addRange(r);
    }
    notifyInput(el);
  }

  function clear() {
    const el = activeEl();
    if (!el) return;
    clearFormatting(el);
    notifyInput(el);
  }

  function setKind(kind: Kind) {
    const b = focused ? find(focused.key) : null;
    if (!b) {
      add(kind);
      return;
    }
    if (kind === 'divider') {
      insertBlock(b.id, { kind: 'divider' } as Partial<Blk>);
      return;
    }
    convert(b.id, b.kind === kind && kind !== 'text' ? 'text' : kind);
  }

  /** Neuen Block nach dem fokussierten (oder am Ende) einfügen. */
  function add(kind: Kind) {
    const after = focused && find(focused.key) ? focused.key : (blocks[blocks.length - 1]?.id ?? null);
    const indent = focused ? (find(focused.key)?.indent ?? 0) : 0;
    if (kind === 'divider') {
      history.group(() => {
        const d = insertBlock(after, { kind: 'divider' } as Partial<Blk>);
        insertBlock(d, { kind: 'text' } as Partial<Blk>, { focus: 'start' });
      });
      return;
    }
    insertBlock(after, { kind, indent: isHeading(kind) ? 0 : indent } as Partial<Blk>, { focus: 'start' });
  }

  /** "+" am Zeilenrand: neuer Block darunter mit geöffnetem Befehlsmenü. */
  async function plus(afterKey: number) {
    const key = insertBlock(afterKey, { kind: 'text', title: '/' } as Partial<Blk>, { focus: 'end' });
    await tick();
    slash = { key, query: '', active: 0 };
  }

  const api: EditorApi = {
    get canUndo() {
      return history.canUndo;
    },
    get canRedo() {
      return history.canRedo;
    },
    get kind() {
      return focused ? (find(focused.key)?.kind ?? null) : null;
    },
    get marks() {
      return marks;
    },
    undo: () => undo(),
    redo: () => redo(),
    setKind,
    format,
    color,
    highlight,
    saveSelection,
    openLink,
    clear,
    add
  };

  // ---- contenteditable-Anbindung ---------------------------------------------

  function rich(node: HTMLElement, p: { html: string; key: number; field: 'title' | 'note'; editable: boolean }) {
    let cur = p;
    let last = p.html;
    node.innerHTML = p.html;
    node.contentEditable = p.editable ? 'true' : 'false';
    (p.field === 'title' ? titleEls : noteEls).set(p.key, node);
    if (!saved.has(`${p.key}:${p.field === 'title' ? 'title' : 'description'}`))
      saved.set(`${p.key}:${p.field === 'title' ? 'title' : 'description'}`, p.html);

    const onInput = () => {
      if (node.innerHTML === '<br>') node.innerHTML = '';
      last = sanitizeHtml(node.innerHTML);
      onText(cur.key, cur.field === 'title' ? 'title' : 'description', last, node);
    };
    const onBlur = () => commitText(cur.key, cur.field === 'title' ? 'title' : 'description');
    const onClick = (e: MouseEvent) => {
      const a = (e.target as HTMLElement).closest('a');
      if (a && (e.ctrlKey || e.metaKey || !node.isContentEditable)) {
        e.preventDefault();
        window.open(a.href, '_blank', 'noopener');
      } else if (a && node.isContentEditable && node.contains(a)) {
        // Klick in einen Link: Text/Adresse bearbeiten (Strg+Klick öffnet ihn)
        openLinkEditor(node, a as HTMLAnchorElement, null);
      }
    };
    node.addEventListener('input', onInput);
    node.addEventListener('blur', onBlur);
    node.addEventListener('click', onClick);
    return {
      update(np: typeof p) {
        cur = np;
        node.contentEditable = np.editable ? 'true' : 'false';
        if (np.html !== last) {
          last = np.html;
          node.innerHTML = np.html;
          if (document.activeElement === node) placeCaret(node, -1);
        }
      },
      destroy() {
        const map = cur.field === 'title' ? titleEls : noteEls;
        if (map.get(cur.key) === node) map.delete(cur.key);
        node.removeEventListener('input', onInput);
        node.removeEventListener('blur', onBlur);
        node.removeEventListener('click', onClick);
      }
    };
  }

  function onRootKey(e: KeyboardEvent) {
    if (!(e.ctrlKey || e.metaKey)) return;
    // Im Status-Feld gilt das native Rückgängig des Eingabefelds.
    if ((e.target as HTMLElement).tagName === "INPUT") return;
    const k = e.key.toLowerCase();
    if (k === 'z' || k === 'y') {
      e.preventDefault();
      if (k === 'y' || e.shiftKey) redo();
      else undo();
    }
  }

  function onFocusOut(e: FocusEvent) {
    if (root && !root.contains(e.relatedTarget as Node | null)) {
      // Fokus verlässt den Editor (Toolbar-Buttons verhindern das per mousedown)
      setTimeout(() => {
        if (root && !root.contains(document.activeElement)) {
          focused = null;
          slash = null;
        }
      }, 0);
    }
  }

  $effect(() => {
    const onSel = () => updateMarks();
    document.addEventListener('selectionchange', onSel);
    const beforeUnload = () => flushAll();
    window.addEventListener('beforeunload', beforeUnload);
    return () => {
      document.removeEventListener('selectionchange', onSel);
      window.removeEventListener('beforeunload', beforeUnload);
      flushAll();
    };
  });

  // ---- Darstellung ------------------------------------------------------------

  // Farben: siehe $lib/palette (ROW_BG für Zeilen/Abschnitte, SWATCH für das Auswahlmenü)
  const iconMt: Partial<Record<Kind, string>> = { h1: 'mt-1', h2: 'mt-0.5', h3: 'mt-0.5', h4: 'mt-0' };

  const barColors: Record<NoteColor, string> = {
    '': 'bg-zinc-300 dark:bg-zinc-600',
    green: 'bg-emerald-500',
    orange: 'bg-amber-500',
    red: 'bg-red-500',
    indigo: 'bg-accent-500'
  };
  const colorCycle: NoteColor[] = ['', 'green', 'orange', 'red', 'indigo'];

  const titleCls: Record<Kind, string> = {
    item: 'text-sm leading-5',
    text: 'text-sm leading-5',
    bullet: 'text-sm leading-5',
    numbered: 'text-sm leading-5',
    quote: 'text-sm leading-5 italic text-zinc-600 dark:text-zinc-300',
    h1: 'font-serif text-[18px] leading-7 font-bold',
    h2: 'font-serif text-[15px] leading-6 font-bold',
    h3: 'text-sm leading-6 font-semibold',
    h4: 'text-[13px] leading-5 font-semibold text-zinc-500 dark:text-zinc-400',
    code: '',
    divider: ''
  };
  // vertikale Mitte der ersten Zeile je Überschriftengröße (Zeilenhöhe 28/24/24/20 px)
  const headingMt: Partial<Record<Kind, string>> = { h1: 'mt-1.5', h2: 'mt-1', h3: 'mt-1', h4: 'mt-0.5' };
  const rowSpacing: Partial<Record<Kind, string>> = { h1: 'pt-5', h2: 'pt-3', h3: 'pt-2', h4: 'pt-2' };
  const spacingPx: Partial<Record<Kind, number>> = { h1: 20, h2: 12, h3: 8, h4: 8 };
  const noteSpacing: Partial<Record<Kind, string>> = { h1: 'md:pt-5', h2: 'md:pt-3', h3: 'md:pt-2', h4: 'md:pt-2' };
  const placeholderFor = (b: Blk) =>
    isHeading(b.kind) ? t(`blocks.${b.kind as 'h1' | 'h2' | 'h3' | 'h4'}`) : t('editor.placeholder');

  const gutterBtn =
    'flex size-5 items-center justify-center rounded text-zinc-300 hover:bg-zinc-100 hover:text-zinc-600 md:opacity-0 md:group-hover/row:opacity-100 dark:text-zinc-600 dark:hover:bg-zinc-800';
</script>

<svelte:window
  onpointerdown={(e) => {
    if (bgMenu !== null && !(e.target as Element | null)?.closest?.('[data-bgmenu]')) bgMenu = null;
    const tg = e.target as Element | null;
    if (linkEdit && !tg?.closest?.("[data-linkpop]") && !tg?.closest?.(".rich a")) linkEdit = null;
  }}
  onkeydown={(e) => {
    if (e.key === 'Escape' && bgMenu !== null) bgMenu = null;
    if (e.key === 'Escape' && linkEdit) linkEdit = null;
  }} />

<div bind:this={root} onkeydown={onRootKey} onfocusout={onFocusOut} role="presentation">
  {#if editable}<EditorToolbar {api} />{/if}

  {#if linkEdit}
    <!-- Link bearbeiten: außerhalb des gezoomten Bereichs, damit die Position stimmt -->
    <form
      data-linkpop
      class="fixed z-50 w-[340px] space-y-2.5 rounded-lg border border-zinc-200 bg-white p-3 shadow-xl dark:border-zinc-700 dark:bg-zinc-900"
      style:left="{linkEdit.x}px"
      style:top="{linkEdit.y}px"
      onsubmit={(e) => {
        e.preventDefault();
        applyLink();
      }}>
      <label class="block">
        <span class="label">{t('editor.linkText')}</span>
        <input class="input py-1! text-sm!" bind:value={linkEdit.text} placeholder={t('editor.linkTextPlaceholder')} />
      </label>
      <label class="block">
        <span class="label">{t('editor.linkUrl')}</span>
        <!-- svelte-ignore a11y_autofocus -->
        <input class="input py-1! text-sm!" bind:value={linkEdit.url} placeholder={t('editor.linkPlaceholder')} autofocus />
      </label>
      <div class="flex flex-wrap items-center gap-1.5 pt-0.5">
        <button type="submit" class="btn-primary px-3! py-1! text-xs!" disabled={!linkEdit.url.trim()}>{t('common.save')}</button>
        {#if linkEdit.a}
          <a
            href={normalizeUrl(linkEdit.url) || linkEdit.a.href}
            target="_blank"
            rel="noopener noreferrer"
            class="btn-ghost px-2! py-1! text-xs!">
            <Icon name="external" size={12} />{t('editor.openLink')}
          </a>
          <button type="button" class="btn-ghost px-2! py-1! text-xs! text-red-600! dark:text-red-400!" onclick={() => applyLink(true)}>
            {t('editor.removeLink')}
          </button>
        {/if}
        <button type="button" class="btn-ghost ml-auto px-2! py-1! text-xs!" onclick={() => (linkEdit = null)}>{t('common.cancel')}</button>
      </div>
    </form>
  {/if}

  <!-- Checkbox eines Eintrags; Position (links/rechts) und Form kommen aus den persönlichen Einstellungen -->
  {#snippet checkbox(b: Blk, done: boolean, extra: string)}
    <button
      type="button"
      role="checkbox"
      aria-checked={done}
      aria-label={t('checklist.markDone')}
      disabled={!editable || !isChecklist}
      onclick={() => setDone(b.id, !done)}
      title={done && b.done_by_name ? t('checklist.doneBy', { name: b.done_by_name, date: fmtDateTime(b.done_at!) }) : undefined}
      class="mt-[2px] flex size-4 shrink-0 items-center justify-center border transition-colors {checkRadius()} {extra} {done
        ? 'border-accent-600 bg-accent-600 text-white'
        : 'border-zinc-500 enabled:hover:border-accent-500 dark:border-zinc-400'}">
      {#if done}<Icon name="check" size={10} class="stroke-[3]" />{/if}
    </button>
  {/snippet}

  <!-- Hintergrundfarbe: bei Überschriften für den ganzen Abschnitt, sonst nur für die Zeile -->
  {#snippet bgPicker(b: Blk, cls: string, label: string, hide: string)}
  <div class="relative shrink-0 {cls}" data-bgmenu>
    <button
      type="button"
      class="flex size-5 items-center justify-center rounded text-zinc-400 hover:bg-black/5 hover:text-zinc-700 dark:hover:bg-white/10 dark:hover:text-zinc-200
        {bgMenu === b.id ? 'opacity-100' : `opacity-0 group-hover/row:opacity-100 focus:opacity-100 ${hide}`}"
      onmousedown={(e) => e.preventDefault()}
      onclick={() => (bgMenu = bgMenu === b.id ? null : b.id)}
      aria-expanded={bgMenu === b.id}
      title={label}
      aria-label={label}>
      <Icon name="droplet" size={13} />
    </button>
    {#if bgMenu === b.id}
      <!-- Zeile in farbigem Abschnitt: "Wie Abschnitt" ('') und "Ohne Farbe" ('none') unterscheiden -->
      {@const inherited = isHeading(b.kind) ? undefined : inheritedColors.get(b.id)?.color}
      {@const noneValue = inherited ? 'none' : ''}
      {@const sel = 'ring-2 ring-accent-500 ring-offset-1 dark:ring-offset-zinc-900'}
      <div class="absolute top-6 right-0 z-30 w-max rounded-lg border border-zinc-200 bg-white p-2 shadow-lg dark:border-zinc-700 dark:bg-zinc-900" role="menu">
        <div class="mb-1.5 flex gap-1">
        {#if inherited}
          <button
            type="button"
            role="menuitem"
            onmousedown={(e) => e.preventDefault()}
            onclick={() => {
              bgMenu = null;
              updateBlock(b.id, { bg_color: '' } as Partial<Blk>);
            }}
            title={t('editor.likeSection')}
            aria-label={t('editor.likeSection')}
            class="flex size-6 items-center justify-center rounded-md text-zinc-600 dark:text-zinc-200 {SWATCH[inherited] ?? ''} hover:scale-110 {!b.bg_color ? sel : ''}">
            <Icon name="undo" size={12} />
          </button>
        {/if}
        <button
          type="button"
          role="menuitem"
          onmousedown={(e) => e.preventDefault()}
          onclick={() => {
            bgMenu = null;
            updateBlock(b.id, { bg_color: noneValue } as Partial<Blk>);
          }}
          title={inherited ? t('editor.noColor') : t('editor.none')}
          aria-label={inherited ? t('editor.noColor') : t('editor.none')}
          class="flex size-6 items-center justify-center rounded-md border border-zinc-300 bg-white text-zinc-400 hover:border-zinc-500 dark:border-zinc-600 dark:bg-zinc-950 {(b.bg_color ?? '') === noneValue ? sel : ''}">
          <Icon name="x" size={12} />
        </button>
        </div>
        <!-- je Zeile eine Stufe: hell, mittel, kräftig (wie Word/Excel) -->
        <div class="grid grid-cols-6 gap-1">
          {#each SHADES as shade (shade)}
            {#each HUES as hue (hue)}
              {@const c = hue + shade}
              <button
                type="button"
                role="menuitem"
                onmousedown={(e) => e.preventDefault()}
                onclick={() => {
                  bgMenu = null;
                  updateBlock(b.id, { bg_color: c } as Partial<Blk>);
                }}
                title={colorLabel(c, (k) => tMaybe(k) ?? k)}
                aria-label={colorLabel(c, (k) => tMaybe(k) ?? k)}
                class="size-6 rounded-md border border-black/10 {SWATCH[c]} hover:scale-110 {b.bg_color === c ? sel : ''}"></button>
            {/each}
          {/each}
        </div>
      </div>
    {/if}
  </div>
  {/snippet}

  <!-- Status-Freitext eines Eintrags ("Erfüllt", "Erreicht" …); feste Breite = bündige Spalte -->
  {#snippet statusField(b: Blk)}
    {#if editable}
      <input
        type="text"
        class="w-20 shrink-0 rounded border border-transparent bg-transparent px-1.5 py-0 text-xs leading-5 text-zinc-700 outline-none md:w-28
          placeholder:text-transparent group-hover/row:placeholder:text-zinc-300 hover:border-zinc-200 focus:border-accent-400 focus:bg-white focus:placeholder:text-zinc-400
          dark:text-zinc-200 dark:group-hover/row:placeholder:text-zinc-600 dark:hover:border-zinc-700 dark:focus:bg-zinc-900"
        value={b.status ?? ''}
        placeholder={t('checklist.status')}
        aria-label={t('checklist.status')}
        maxlength="60"
        list={listId}
        onfocus={() => (focused = { key: b.id, field: 'title' })}
        onchange={(e) => {
          const v = e.currentTarget.value.trim();
          if (v !== (b.status ?? '')) updateBlock(b.id, { status: v } as Partial<Blk>);
        }}
        onkeydown={(e) => {
          if (e.key === 'Enter') {
            e.preventDefault();
            e.currentTarget.blur();
          } else if (e.key === 'Escape') {
            e.currentTarget.value = b.status ?? '';
            e.currentTarget.blur();
          }
        }} />
    {:else if b.status}
      <span class="w-20 shrink-0 truncate px-1.5 text-xs leading-5 text-zinc-700 md:w-28 dark:text-zinc-200" title={b.status}>{b.status}</span>
    {:else}
      <span class="w-20 shrink-0 md:w-28"></span>
    {/if}
  {/snippet}

  <datalist id={listId}>
    {#each statusOptions as o (o)}<option value={o}></option>{/each}
  </datalist>

  <!-- Größe per CSS zoom: skaliert Text, Checkboxen und Notizspalte gemeinsam -->
  <div style:zoom={prefs.zoom === 1 ? undefined : prefs.zoom}>

  <div class="grid md:grid-cols-[minmax(0,1fr)_18rem] 2xl:grid-cols-[minmax(0,1fr)_24rem]">
    <div class="flex justify-end pr-2 pb-1.5">
      {#if headingIds.length}
        <button
          type="button"
          class="flex items-center gap-1 text-[11px] font-medium text-zinc-400 hover:text-zinc-700 dark:hover:text-zinc-200"
          onclick={() => setAllCollapsed(anyExpanded)}>
          <Icon name={anyExpanded ? 'chevron' : 'chevronDown'} size={12} />
          {anyExpanded ? t('editor.collapseAll') : t('editor.expandAll')}
        </button>
      {/if}
    </div>
    <div class="hidden pb-1.5 pl-4 text-[11px] font-semibold tracking-wider text-zinc-400 uppercase md:block">{t('checklist.notes')}</div>
  </div>

  <div use:sortable={{ onsort: (ids) => reorderVisible(ids), disabled: !editable }}>
    {#each blocks as b, i (b.id)}
      {@const done = isChecklist && b.kind === 'item' && !!b.done_at}
      {@const isFocused = focused?.key === b.id}
      {@const isCollapsed = isHeading(b.kind) && collapsed.has(b.id)}
      {@const stat = hStats.get(b.id)}
      <div
        data-id={b.id}
        id={b.kind === 'h1' ? `h-${b.id}` : undefined}
        class="group/row relative scroll-mt-20 {sectionRowCls(b.id)} {hidden.has(b.id) ? 'hidden' : 'grid'} md:grid-cols-[minmax(0,1fr)_18rem] 2xl:grid-cols-[minmax(0,1fr)_24rem]
          {isFocused && editable ? 'ring-1 ring-accent-400/80 dark:ring-accent-500/60' : ''}
          [&.dragging]:bg-white [&.dragging]:shadow-lg dark:[&.dragging]:bg-zinc-900">
        <!-- Hauptspalte -->
        <div class="relative flex min-w-0 items-start gap-2 py-[3px] pr-2 {i > 0 ? (rowSpacing[b.kind] ?? '') : ''}" style:padding-left="{32 + b.indent * 24}px">
          {#if editable}
            <div class="absolute flex items-center" style:left="{b.indent * 24}px" style:top="{(i > 0 ? (spacingPx[b.kind] ?? 0) : 0) + (b.kind === "h1" ? 7 : b.kind === "h2" ? 5 : 3)}px">
              <button type="button" class="{gutterBtn} max-md:hidden" onclick={() => plus(b.id)} title={t('editor.addBlock')} aria-label={t('editor.addBlock')}>
                <Icon name="plus" size={13} />
              </button>
              <button type="button" data-handle class="{gutterBtn} cursor-grab touch-none" title={t('checklist.drag')} aria-label={t('checklist.drag')}>
                <Icon name="grip" size={12} />
              </button>
            </div>
          {/if}

          {#if b.kind === 'divider'}
            <!-- svelte-ignore a11y_no_noninteractive_tabindex, a11y_no_noninteractive_element_interactions -->
            <div
              class="flex h-5 flex-1 items-center rounded outline-none focus-visible:ring-2 focus-visible:ring-accent-400"
              tabindex={editable ? 0 : -1}
              role="separator"
              onfocus={() => (focused = { key: b.id, field: 'title' })}
              onkeydown={(e) => {
                if (e.key === 'Backspace' || e.key === 'Delete') {
                  e.preventDefault();
                  const p = neighbour(b.id, -1);
                  deleteBlock(b.id);
                  if (p) focusBlock(p.id, 'end');
                } else if (e.key === 'ArrowUp' || e.key === 'ArrowDown') {
                  const n = neighbour(b.id, e.key === 'ArrowUp' ? -1 : 1);
                  if (n) focusBlock(n.id, e.key === 'ArrowUp' ? 'end' : 'start');
                }
              }}>
              <hr class="w-full border-zinc-200 dark:border-zinc-700" />
            </div>
          {:else if b.kind === 'code'}
            <div class="min-w-0 flex-1">
              <CodeBlock
                code={b.title}
                lang={b.lang}
                {editable}
                oninput={(v) => onText(b.id, 'title', v)}
                onlang={(lang) => updateBlock(b.id, { lang } as Partial<Blk>)}
                onfocus={() => (focused = { key: b.id, field: 'title' })}
                register={(el) => (el ? codeEls.set(b.id, el) : codeEls.delete(b.id))}
                onkeydown={(e, el) => {
                  if (e.key === 'Backspace' && el.value === '') {
                    e.preventDefault();
                    convert(b.id, 'text');
                  } else if ((e.key === 'Enter' && (e.ctrlKey || e.metaKey)) || e.key === 'Escape') {
                    e.preventDefault();
                    const n = neighbour(b.id, 1);
                    if (n) focusBlock(n.id, 'start');
                    else insertBlock(b.id, { kind: 'text', indent: b.indent } as Partial<Blk>, { focus: 'start' });
                  } else if (e.key === 'ArrowUp' && el.selectionStart === 0) {
                    const p = neighbour(b.id, -1);
                    if (p) {
                      e.preventDefault();
                      focusBlock(p.id, 'end');
                    }
                  } else if (e.key === 'ArrowDown' && el.selectionEnd === el.value.length) {
                    const n = neighbour(b.id, 1);
                    if (n) {
                      e.preventDefault();
                      focusBlock(n.id, 'start');
                    }
                  }
                }} />
            </div>
            <!-- Platzhalter wie Checkbox (rechts) und Status: Codeblock endet bündig mit dem Text der Einträge -->
            {#if prefs.checkPos === 'right'}<span class="ml-1 size-4 shrink-0" aria-hidden="true"></span>{/if}
            {#if isChecklist}<span class="w-20 shrink-0 md:w-28" aria-hidden="true"></span>{/if}
          {:else}
            <!-- Marker je Blocktyp -->
            {#if b.kind === 'item' && prefs.checkPos === 'left'}
              {@render checkbox(b, done, '')}
            {:else if b.kind === 'bullet'}
              <span class="mt-[2px] flex size-4 shrink-0 items-center justify-center text-zinc-500">•</span>
            {:else if b.kind === 'numbered'}
              <span class="mt-px min-w-4 shrink-0 text-right text-sm leading-5 text-zinc-500 tabular-nums">{nums.get(b.id)}.</span>
            {:else if b.kind === 'quote'}
              <span class="w-0.5 shrink-0 self-stretch rounded-full bg-zinc-300 dark:bg-zinc-600"></span>
            {:else if isHeading(b.kind)}
              <button
                type="button"
                onclick={() => toggleCollapse(b.id)}
                onmousedown={(e) => e.preventDefault()}
                aria-expanded={!isCollapsed}
                title={isCollapsed ? t('editor.expand') : t('editor.collapse')}
                aria-label={isCollapsed ? t('editor.expand') : t('editor.collapse')}
                class="flex size-4 shrink-0 items-center justify-center rounded transition-colors hover:bg-zinc-100 hover:text-zinc-700 dark:hover:bg-zinc-800 dark:hover:text-zinc-200
                  {headingMt[b.kind]} {isCollapsed ? 'text-zinc-800 dark:text-zinc-200' : 'text-zinc-500 dark:text-zinc-400'}">
                <Icon name="chevronDown" size={14} class="transition-transform {isCollapsed ? '-rotate-90' : ''}" />
              </button>
            {/if}

            <div class="relative min-w-0 {isHeading(b.kind) ? '' : 'flex-1'}">
              <div
                use:rich={{ html: b.title, key: b.id, field: 'title', editable }}
                class="rich min-h-5 outline-none {titleCls[b.kind]} {done ? 'text-zinc-400 line-through dark:text-zinc-500' : ''}
                  {isHeading(b.kind) ? 'ph-always' : 'ph-focus'}"
                data-placeholder={placeholderFor(b)}
                role="textbox"
                tabindex="0"
                aria-multiline="false"
                onfocus={() => (focused = { key: b.id, field: 'title' })}
                onkeydown={(e) => onTitleKey(e, b)}
                onpaste={(e) => onPaste(e, b, 'title')}>
              </div>
            </div>

            {#if stat && isChecklist && stat.total && (b.kind === 'h1' || isCollapsed)}
              <span class="shrink-0 text-xs text-zinc-400 tabular-nums {headingMt[b.kind]} leading-4">{stat.done}/{stat.total}</span>
            {/if}
            {#if stat && isCollapsed && stat.blocks}
              <button
                type="button"
                onclick={() => toggleCollapse(b.id)}
                class="shrink-0 rounded bg-zinc-100 px-1.5 text-[11px] leading-4 text-zinc-500 hover:bg-zinc-200 dark:bg-zinc-800 dark:text-zinc-400 dark:hover:bg-zinc-700 {headingMt[b.kind]}">
                {t('editor.hiddenBlocks', { n: stat.blocks })}
              </button>
            {/if}
            {#if isHeading(b.kind)}<span class="flex-1"></span>{/if}

            {#if isHeading(b.kind) && editable}
              <!-- Abschnittsfarbe -->
              {@render bgPicker(b, iconMt[b.kind] ?? "", t("editor.sectionColor"), "max-md:opacity-100")}
            {/if}

            {#if isChecklist && b.kind === 'item'}
              <!-- Zuständig -->
              {#if editField?.key === b.id && editField.field === 'assignee'}
                <select
                  class="input w-auto! py-0! text-xs!"
                  value={b.assignee_id ?? ''}
                  onblur={() => (editField = null)}
                  onchange={(e) => {
                    const v = e.currentTarget.value;
                    editField = null;
                    updateBlock(b.id, { assignee_id: v ? Number(v) : null, assignee_name: assignees.find((a) => a.id === Number(v))?.name ?? null } as Partial<Blk>);
                  }}>
                  <option value="">{t('checklist.nobody')}</option>
                  {#each assignees as a (a.id)}<option value={a.id}>{a.name}</option>{/each}
                </select>
              {:else if b.assignee_name}
                <button
                  type="button"
                  disabled={!editable}
                  class="mt-px shrink-0 text-xs leading-5 whitespace-nowrap text-zinc-500 hover:text-zinc-800 dark:text-zinc-400 dark:hover:text-zinc-200"
                  onclick={() => (editField = { key: b.id, field: 'assignee' })}>{shortName(b.assignee_name)}</button>
              {:else if editable}
                <button
                  type="button"
                  class="mt-[3px] shrink-0 text-zinc-300 opacity-0 group-hover/row:opacity-100 hover:text-zinc-600 focus:opacity-100 max-md:hidden dark:text-zinc-600"
                  title={t('checklist.assign')}
                  onclick={() => (editField = { key: b.id, field: 'assignee' })}><Icon name="user" size={14} /></button>
              {/if}

              <!-- Fälligkeit -->
              {#if editField?.key === b.id && editField.field === 'due'}
                <input
                  type="date"
                  class="input w-auto! py-0! text-xs!"
                  value={b.due_date ?? ''}
                  onblur={() => (editField = null)}
                  onchange={(e) => updateBlock(b.id, { due_date: e.currentTarget.value || null } as Partial<Blk>)} />
              {:else if b.due_date}
                <button
                  type="button"
                  disabled={!editable}
                  class="mt-px shrink-0 text-xs leading-5 whitespace-nowrap tabular-nums"
                  onclick={() => (editField = { key: b.id, field: 'due' })}>
                  {#if b.due_date === today && !done}
                    <span class="rounded bg-red-50 px-1.5 py-px text-[11px] font-medium text-red-600 dark:bg-red-950/50 dark:text-red-400">{t('checklist.today')}</span>
                  {:else}
                    <span class={!done && b.due_date < today ? 'font-medium text-red-600 dark:text-red-400' : 'text-zinc-500 dark:text-zinc-400'}>{fmtDate(b.due_date)}</span>
                  {/if}
                </button>
              {:else if editable}
                <button
                  type="button"
                  class="mt-[3px] shrink-0 text-zinc-300 opacity-0 group-hover/row:opacity-100 hover:text-zinc-600 focus:opacity-100 max-md:hidden dark:text-zinc-600"
                  title={t('checklist.setDue')}
                  onclick={() => (editField = { key: b.id, field: 'due' })}><Icon name="calendar" size={14} /></button>
              {/if}
            {/if}

            {#if b.kind === 'item' && prefs.checkPos === 'right'}
              {@render checkbox(b, done, 'ml-1')}
            {/if}
            {#if b.kind === 'item' && isChecklist}
              {@render statusField(b)}
            {/if}
          {/if}

          {#if editable}
            {#if editable && ['text', 'bullet', 'numbered', 'quote', 'h1', 'h2', 'h3', 'h4'].includes(b.kind)}
              <!-- in einen Checklistenpunkt (mit Checkbox) umwandeln -->
              <button
                type="button"
                class="mt-px flex size-5 shrink-0 items-center justify-center rounded text-zinc-400 opacity-0 group-hover/row:opacity-100 hover:bg-black/5 hover:text-accent-600 focus:opacity-100 max-md:hidden dark:hover:bg-white/10 dark:hover:text-accent-400
                  {isHeading(b.kind) ? (iconMt[b.kind] ?? '') : ''}"
                onmousedown={(e) => e.preventDefault()}
                onclick={() => convert(b.id, 'item')}
                title={t('editor.toItem')}
                aria-label={t('editor.toItem')}>
                <Icon name="checkSquare" size={14} />
              </button>
            {/if}
            {#if editable && !isHeading(b.kind) && b.kind !== 'divider'}
              <!-- Zeilenfarbe -->
              {@render bgPicker(b, 'mt-px max-md:hidden', t('editor.rowColor'), '')}
            {/if}
            <span class="flex shrink-0 opacity-0 group-hover/row:opacity-100 focus-within:opacity-100 max-md:hidden">
              <ConfirmButton onconfirm={() => deleteBlock(b.id)} class="btn-danger p-0.5! leading-none"><Icon name="trash" size={14} /></ConfirmButton>
            </span>
          {/if}

          {#if slash?.key === b.id}
            <SlashMenu list={slashList} active={slash.active} onselect={chooseSlash} onhover={(n) => slash && (slash.active = n)} />
          {/if}
        </div>

        <!-- Notiz -->
        <div
          class="min-w-0 items-start gap-2.5 pb-1 pl-8 bg-white md:border-l md:border-zinc-300 md:py-[3px] md:pl-4 dark:bg-zinc-950 dark:md:border-zinc-700 {i > 0 ? (noteSpacing[b.kind] ?? "") : ""}
            {b.description ? "flex" : "hidden group-focus-within/row:flex md:flex"}">
          {#if b.kind !== 'divider'}
            {#if b.description}
              <button
                type="button"
                disabled={!editable}
                onclick={() => updateBlock(b.id, { note_color: colorCycle[(colorCycle.indexOf(b.note_color) + 1) % colorCycle.length] } as Partial<Blk>)}
                title={t('checklist.noteColor')}
                aria-label={t('checklist.noteColor')}
                class="mt-[3px] h-[14px] w-[3px] shrink-0 rounded-full md:ml-0 {barColors[b.note_color]}"></button>
            {/if}
            <div
              use:rich={{ html: b.description, key: b.id, field: 'note', editable }}
              class="rich ph-hover min-h-5 min-w-0 flex-1 text-[13px] leading-5 text-zinc-600 outline-none dark:text-zinc-300"
              data-placeholder={editable ? t('checklist.addNote') : ''}
              role="textbox"
              tabindex="0"
              onfocus={() => (focused = { key: b.id, field: 'note' })}
              onkeydown={(e) => {
                if (e.key === 'Enter' && !e.shiftKey) {
                  e.preventDefault();
                  (e.currentTarget as HTMLElement).blur();
                } else if (e.key === 'Escape') (e.currentTarget as HTMLElement).blur();
              }}
              onpaste={(e) => onPaste(e, b, 'note')}>
            </div>
          {/if}
        </div>
      </div>
    {/each}
  </div>

  {#if editable}
    <button
      type="button"
      class="mt-1 flex w-full items-center gap-2 rounded-md py-1.5 pl-8 text-left text-sm text-zinc-400 hover:bg-zinc-50 hover:text-zinc-600 dark:hover:bg-zinc-900/70"
      onclick={() => {
        const last = blocks[blocks.length - 1];
        insertBlock(last?.id ?? null, { kind: last?.kind === "item" ? "item" : "text", indent: last?.indent ?? 0 } as Partial<Blk>, { focus: "start" });
      }}>
      <Icon name="plus" size={14} />
      {t('editor.addBlock')}
    </button>
  {:else if blocks.length === 0}
    <p class="py-6 pl-8 text-sm text-zinc-400">{t('checklist.empty')}</p>
  {/if}
  </div>
</div>
