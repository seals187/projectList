<script lang="ts" generics="T extends Row">
  // Block editor for checklists and templates.
  //
  // Each block is a row (its own DB row) with a contenteditable title and
  // a note in the right column. This component only translates DOM and
  // keyboard events into edits of the block list model ($lib/blockListModel),
  // which owns the edits, undo/redo and the sequential server sync, and
  // renders the rows.
  import { tick } from 'svelte';
  import { errorText } from '$lib/api';
  import { checkRadius, NOTE_MAX, NOTE_MIN, prefs, setPref } from '$lib/state.svelte';
  import { session } from '$lib/session.svelte';
  import { toast } from '$lib/toasts.svelte';
  import { fmtDate, fmtDateTime, shortName, t, tMaybe, todayISO } from '$lib/i18n.svelte';
  import { sortable } from '$lib/sortable';
  import { BlockListModel, type Field } from '$lib/blockListModel';
  import { SvelteSet } from 'svelte/reactivity';
  import { colorLabel, HUES, ROW_BG, SHADES, SWATCH } from '$lib/palette';
  import { filterKinds, headingLevel, headingStats, hiddenBlocks, isHeading, sectionColors, isTextual, numbering, SHORTCUTS } from '$lib/blocks';
  import {
    caretAtEnd,
    caretAtStart,
    caretOnFirstLine,
    caretOnLastLine,
    clearFormatting,
    extractAfterCaret,
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
    /** Key for the locally stored collapse state (e.g. "checklist:1"). */
    collapseKey?: string;
  } = $props();

  const isChecklist = $derived(!!ops.toggle);
  const today = todayISO();

  const blocks = $derived(items as Blk[]);
  const nums = $derived(numbering(items));
  const hStats = $derived(headingStats(items));

  // ---- Section colors -----------------------------------------------------------
  // The color belongs to the heading (bg_color) and applies to its section.
  // Only the start/end of a colored area is rounded (visible rows).

  const secColors = $derived(sectionColors(items));
  // Color a row without its own color would inherit from the section (for "Same as section")
  const inheritedColors = $derived(sectionColors(items.map((b) => (isHeading(b.kind) ? b : { ...b, bg_color: '' as const }))));
  // Edges (spacing, rounding) only at the boundaries of colored sections (headings).
  // Individually colored rows only change the background and join seamlessly.
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
    const sec = secColors.get(key); // actual color (own row color or section)
    const e = secEdges.get(key); // position within the colored section
    const bg = sec ? (ROW_BG[sec.color] ?? '') : '';
    const g = grid.get(key);
    // Grid rows: square corners, spacing at section end outside instead of inside (otherwise the cell gets taller)
    if (g) return `${bg} ${gridCls} ${g.joinPrev ? '-mt-px' : ''} ${e?.last ? 'mb-1.5' : ''}`;
    // rows without a frame (headings, dividers) get an invisible one of the same width,
    // so the note column lines up with the framed rows
    const pad = prefs.gridLines ? 'border-x border-transparent' : '';
    if (!e) return `${bg} ${pad} rounded-md`;
    const radius = e.first && e.last ? 'rounded-md' : e.first ? 'rounded-t-md' : e.last ? 'rounded-b-md' : '';
    return `${bg} ${pad} ${radius} ${e.first ? 'mt-1' : ''} ${e.last ? 'mb-1 pb-1.5' : ''}`;
  }

  // ---- Grid lines ---------------------------------------------------------------
  // With prefs.gridLines every content row (not headings/dividers) gets
  // a border; consecutive rows share the line (-mt-px).
  const gridCls = 'border border-zinc-500 dark:border-zinc-500'; // dark lines like in Excel
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

  // ---- Status (free text right of the checkbox) --------------------------------
  const statusOptions = $derived.by(() => {
    const set = new Set<string>(t('checklist.statusSuggestions').split('|'));
    for (const b of blocks) if (b.status) set.add(b.status);
    return [...set];
  });
  const listId = `status-${Math.random().toString(36).slice(2, 8)}`;

  // ---- Collapsing ---------------------------------------------------------------
  // Per user/browser in localStorage, not in the database: collapsing is
  // a view concern and should not affect other editors.

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
      /* private mode or similar */
    }
  }

  function toggleCollapse(key: number) {
    if (collapsed.has(key)) collapsed.delete(key);
    else collapsed.add(key);
    // If focus was in a now hidden block, it moves to the heading.
    const row = (document.activeElement as HTMLElement | null)?.closest('[data-id]') as HTMLElement | null;
    if (row && hiddenBlocks(items, collapsed).has(Number(row.dataset.id))) focusBlock(key, 'end');
    saveCollapsed();
  }

  function setAllCollapsed(on: boolean) {
    collapsed.clear();
    if (on) for (const id of headingIds) collapsed.add(id);
    saveCollapsed();
  }

  /** Expands all headings under which the block is hidden. */
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

  /** Drag & drop only yields visible IDs; hidden blocks move along with their heading. */
  function reorderVisible(visible: number[]) {
    const groups = new Map<number, number[]>();
    let cur: number[] | null = null;
    for (const b of blocks) {
      if (!hidden.has(b.id) || !cur) {
        cur = [b.id];
        groups.set(b.id, cur);
      } else cur.push(b.id);
    }
    model.reorder(visible.flatMap((id) => groups.get(id) ?? []));
  }

  // ---- State ------------------------------------------------------------------

  let focused = $state<{ key: number; field: 'title' | 'note' } | null>(null);
  let slash = $state<{ key: number; query: string; active: number } | null>(null);
  let editField = $state<{ key: number; field: 'assignee' | 'due' } | null>(null);
  let marks = $state({ bold: false, italic: false, underline: false, strike: false, code: false });
  let root = $state<HTMLElement>();

  // Note column width: dragged at its left edge, saved as a view preference.
  // noteW is the rendered width (px, unzoomed), 0 while the column is hidden (mobile).
  let editorEl = $state<HTMLElement>();
  let noteW = $state(0);
  let noteResizing = $state(false);
  // the main column keeps at least 240 px
  const clampNote = (w: number) =>
    Math.round(Math.max(NOTE_MIN, Math.min(NOTE_MAX, editorEl ? editorEl.offsetWidth - 240 : NOTE_MAX, w)));
  function startNoteResize(e: PointerEvent) {
    if (e.button !== 0) return;
    e.preventDefault();
    const handle = e.currentTarget as HTMLElement;
    handle.setPointerCapture(e.pointerId);
    const startX = e.clientX;
    const startW = noteW;
    const zoom = prefs.zoom || 1;
    noteResizing = true;
    document.body.style.cursor = 'col-resize';
    document.body.style.userSelect = 'none';
    // during the drag only the live value changes; saved on release
    const move = (ev: PointerEvent) => (prefs.noteWidth = clampNote(startW - (ev.clientX - startX) / zoom));
    const up = () => {
      noteResizing = false;
      document.body.style.cursor = '';
      document.body.style.userSelect = '';
      setPref('noteWidth', prefs.noteWidth);
      handle.removeEventListener('pointermove', move);
      handle.removeEventListener('pointerup', up);
      handle.removeEventListener('pointercancel', up);
    };
    handle.addEventListener('pointermove', move);
    handle.addEventListener('pointerup', up);
    handle.addEventListener('pointercancel', up);
  }

  const titleEls = new Map<number, HTMLElement>();
  const noteEls = new Map<number, HTMLElement>();
  const codeEls = new Map<number, HTMLTextAreaElement>();
  let savedRange: Range | null = null;

  const slashList = $derived(slash ? filterKinds(slash.query, t) : []);

  // ---- Model (edits, history, server sync) --------------------------------------

  const model = new BlockListModel<T>({
    items: () => items,
    setItems: (next) => (items = next),
    ops: () => ops,
    user: () => session.user,
    onChange: () => onchange?.(),
    onError: (e) => {
      toast(errorText(e));
      onerror?.();
    },
    onRemove: (key) => {
      if (slash?.key === key) slash = null;
    }
  });

  const sid = (key: number) => model.serverId(key);
  const find = (key: number) => blocks.find((b) => b.id === key);
  const indexOf = (key: number) => blocks.findIndex((b) => b.id === key);

  /** Text input from the DOM; titles also drive the slash menu and Markdown shortcuts. */
  function onText(key: number, field: Field, value: string, el?: HTMLElement) {
    const b = model.text(key, field, value);
    if (b && field === 'title' && el) afterTitleInput(b, el);
  }

  function insertBlock(afterKey: number | null, data: Partial<Blk>, focus?: 'start' | 'end') {
    const key = model.insert(afterKey, data);
    if (focus) focusBlock(key, focus);
    return key;
  }

  const undo = () => model.undo();
  const redo = () => model.redo();

  // ---- Focus & navigation -----------------------------------------------------

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

  // ---- Keyboard in the title ----------------------------------------------------

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
      const to = model.backspaceAtStart(key);
      if (to) focusBlock(to.key, to.at);
    } else if (e.key === 'Tab') {
      e.preventDefault();
      model.indent(key, e.shiftKey ? -1 : 1);
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
      // Empty list item + Enter → leave the list
      model.exitList(key);
      return;
    }
    if (caretAtStart(el) && el.textContent) {
      model.insertAbove(key);
      return;
    }
    const after = extractAfterCaret(el);
    const added = model.split(key, sanitizeHtml(el.innerHTML), after);
    if (added != null) focusBlock(added, 'start');
  }

  /** Slash menu and Markdown shortcuts after input in the title. */
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

  /** Converts a block (clear = empty the text, e.g. after "/command"). */
  function convert(key: number, kind: Kind, clear = false) {
    const res = model.convert(key, kind, clear);
    if (!res) return;
    if (res.inserted != null) focusBlock(res.inserted, 'start');
    focusBlock(key, 'end');
  }

  // ---- Pasting from the clipboard --------------------------------------------

  function onPaste(e: ClipboardEvent, b: Blk, field: 'title' | 'note') {
    e.preventDefault();
    const text = e.clipboardData?.getData('text/plain') ?? '';
    const lines = text.replace(/\r\n?/g, '\n').split('\n');
    if (field === 'note' || lines.length === 1) {
      insertPlainText(text);
      return;
    }
    // Multiline into the title: first line here, further lines as new blocks.
    insertPlainText(lines[0]);
    const el = titleEls.get(b.id);
    const last = model.pasteLines(b.id, el ? sanitizeHtml(el.innerHTML) : undefined, lines.slice(1));
    focusBlock(last, 'end');
  }

  // ---- Inline formatting (toolbar) --------------------------------------------

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

  // ---- Links: edit text and address ---------------------------------------------
  // Clicking a link (or the link button) opens a small dialog with
  // text + URL. Without a selection, a new link is inserted at the cursor position.

  let linkEdit = $state<{
    el: HTMLElement;
    a: HTMLAnchorElement | null;
    range: Range | null;
    text: string;
    url: string;
    x: number;
    y: number;
  } | null>(null);

  /** Without a scheme: email address → mailto:, otherwise https:// (only http(s)/mailto are allowed). */
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

  /** Toolbar: edit the existing link under the cursor or create a new one. */
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
      na.querySelectorAll('a').forEach((x) => x.replaceWith(...x.childNodes)); // no nested links
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
      model.insert(b.id, { kind: 'divider' } as Partial<Blk>);
      return;
    }
    convert(b.id, b.kind === kind && kind !== 'text' ? 'text' : kind);
  }

  /** Insert a new block after the focused one (or at the end). */
  function add(kind: Kind) {
    const after = focused && find(focused.key) ? focused.key : (blocks[blocks.length - 1]?.id ?? null);
    const indent = focused ? (find(focused.key)?.indent ?? 0) : 0;
    focusBlock(model.add(after, kind, indent), 'start');
  }

  /** "+" at the row edge: new block below with the command menu open. */
  async function plus(afterKey: number) {
    const key = insertBlock(afterKey, { kind: 'text', title: '/' } as Partial<Blk>, 'end');
    await tick();
    slash = { key, query: '', active: 0 };
  }

  const api: EditorApi = {
    get canUndo() {
      return model.canUndo;
    },
    get canRedo() {
      return model.canRedo;
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

  // ---- contenteditable binding -----------------------------------------------

  function rich(node: HTMLElement, p: { html: string; key: number; field: 'title' | 'note'; editable: boolean }) {
    let cur = p;
    let last = p.html;
    node.innerHTML = p.html;
    node.contentEditable = p.editable ? 'true' : 'false';
    (p.field === 'title' ? titleEls : noteEls).set(p.key, node);
    model.baseline(p.key, p.field === 'title' ? 'title' : 'description', p.html);

    const onInput = () => {
      if (node.innerHTML === '<br>') node.innerHTML = '';
      last = sanitizeHtml(node.innerHTML);
      onText(cur.key, cur.field === 'title' ? 'title' : 'description', last, node);
    };
    const onBlur = () => model.commitText(cur.key, cur.field === 'title' ? 'title' : 'description');
    const onClick = (e: MouseEvent) => {
      const a = (e.target as HTMLElement).closest('a');
      if (a && (e.ctrlKey || e.metaKey || !node.isContentEditable)) {
        e.preventDefault();
        window.open(a.href, '_blank', 'noopener');
      } else if (a && node.isContentEditable && node.contains(a)) {
        // Click in a link: edit text/address (Ctrl+click opens it)
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
    // In the status field, the input's native undo applies.
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
      // Focus leaves the editor (toolbar buttons prevent this via mousedown)
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
    const beforeUnload = () => model.flush();
    window.addEventListener('beforeunload', beforeUnload);
    return () => {
      document.removeEventListener('selectionchange', onSel);
      window.removeEventListener('beforeunload', beforeUnload);
      model.flush();
    };
  });

  // ---- Rendering --------------------------------------------------------------

  // Colors: see $lib/palette (ROW_BG for rows/sections, SWATCH for the picker menu)
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
  // vertical center of the first line per heading size (line height 28/24/24/20 px)
  const headingMt: Partial<Record<Kind, string>> = { h1: 'mt-1.5', h2: 'mt-1', h3: 'mt-1', h4: 'mt-0.5' };
  const rowSpacing: Partial<Record<Kind, string>> = { h1: 'pt-5', h2: 'pt-3', h3: 'pt-2', h4: 'pt-2' };
  const spacingPx: Partial<Record<Kind, number>> = { h1: 20, h2: 12, h3: 8, h4: 8 };
  const noteSpacing: Partial<Record<Kind, string>> = { h1: 'md:pt-5', h2: 'md:pt-3', h3: 'md:pt-2', h4: 'md:pt-2' };
  const placeholderFor = (b: Blk) =>
    isHeading(b.kind) ? t(`blocks.${b.kind as 'h1' | 'h2' | 'h3' | 'h4'}`) : t('editor.placeholder');

  const gutterBtn =
    'flex size-5 items-center justify-center rounded text-zinc-500 hover:bg-black/5 hover:text-zinc-900 md:opacity-0 md:group-hover/row:opacity-100 dark:text-zinc-400 dark:hover:bg-white/10 dark:hover:text-zinc-100';
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
    <!-- Edit link: outside the zoomed area so the position is correct -->
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

  <!-- Checkbox of an item; position (left/right) and shape come from the personal settings -->
  {#snippet checkbox(b: Blk, done: boolean, extra: string)}
    <button
      type="button"
      role="checkbox"
      aria-checked={done}
      aria-label={t('checklist.markDone')}
      disabled={!editable || !isChecklist}
      onclick={() => model.setDone(b.id, !done)}
      title={done && b.done_by_name ? t('checklist.doneBy', { name: b.done_by_name, date: fmtDateTime(b.done_at!) }) : undefined}
      class="mt-[2px] flex size-4 shrink-0 items-center justify-center border transition-colors {checkRadius()} {extra} {done
        ? 'border-accent-600 bg-accent-600 text-white'
        : 'border-zinc-500 enabled:hover:border-accent-500 dark:border-zinc-400'}">
      {#if done}<Icon name="check" size={10} class="stroke-[3]" />{/if}
    </button>
  {/snippet}

  <!-- Background color: for headings it applies to the whole section, otherwise only to the row -->
  {#snippet bgPicker(b: Blk, cls: string, label: string, hide: string)}
  <div class="relative shrink-0 {cls}" data-bgmenu>
    <button
      type="button"
      class="flex size-5 items-center justify-center rounded text-zinc-500 hover:bg-black/5 hover:text-zinc-900 dark:text-zinc-400 dark:hover:bg-white/10 dark:hover:text-zinc-100
        {bgMenu === b.id ? 'opacity-100' : `opacity-0 group-hover/row:opacity-100 focus:opacity-100 ${hide}`}"
      onmousedown={(e) => e.preventDefault()}
      onclick={() => (bgMenu = bgMenu === b.id ? null : b.id)}
      aria-expanded={bgMenu === b.id}
      title={label}
      aria-label={label}>
      <Icon name="droplet" size={13} />
    </button>
    {#if bgMenu === b.id}
      <!-- Row in a colored section: distinguish "Same as section" ('') and "No color" ('none') -->
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
              model.update(b.id, { bg_color: '' } as Partial<Blk>);
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
            model.update(b.id, { bg_color: noneValue } as Partial<Blk>);
          }}
          title={inherited ? t('editor.noColor') : t('editor.none')}
          aria-label={inherited ? t('editor.noColor') : t('editor.none')}
          class="flex size-6 items-center justify-center rounded-md border border-zinc-300 bg-white text-zinc-400 hover:border-zinc-500 dark:border-zinc-600 dark:bg-zinc-900 {(b.bg_color ?? '') === noneValue ? sel : ''}">
          <Icon name="x" size={12} />
        </button>
        </div>
        <!-- one shade per row: light, medium, strong (like Word/Excel) -->
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
                  model.update(b.id, { bg_color: c } as Partial<Blk>);
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

  <!-- Free-text status of an item ("Fulfilled", "Achieved" …); fixed width = aligned column -->
  {#snippet statusField(b: Blk, cls = '')}
    {#if editable}
      <input
        type="text"
        class="{cls} w-20 shrink-0 rounded border border-transparent bg-transparent px-1.5 py-0 text-xs leading-5 text-zinc-700 outline-none md:w-28
          placeholder:text-zinc-400 hover:border-zinc-300 focus:border-accent-400 focus:bg-white focus:placeholder:text-zinc-400
          dark:text-zinc-200 dark:placeholder:text-zinc-500 dark:hover:border-zinc-600 dark:focus:bg-zinc-900"
        value={b.status ?? ''}
        placeholder={t('checklist.status')}
        aria-label={t('checklist.status')}
        maxlength="60"
        list={listId}
        onfocus={() => (focused = { key: b.id, field: 'title' })}
        onchange={(e) => {
          const v = e.currentTarget.value.trim();
          if (v !== (b.status ?? '')) model.update(b.id, { status: v } as Partial<Blk>);
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
      <span class="{cls} w-20 shrink-0 truncate px-1.5 text-xs leading-5 text-zinc-700 md:w-28 dark:text-zinc-200" title={b.status}>{b.status}</span>
    {:else}
      <span class="{cls} w-20 shrink-0 md:w-28"></span>
    {/if}
  {/snippet}

  <datalist id={listId}>
    {#each statusOptions as o (o)}<option value={o}></option>{/each}
  </datalist>

  <!-- Size via CSS zoom: scales text, checkboxes and note column together -->
  <div
    bind:this={editorEl}
    class="relative"
    style:zoom={prefs.zoom === 1 ? undefined : prefs.zoom}
    style:--note-w={prefs.noteWidth ? `${prefs.noteWidth}px` : undefined}>

  {#if noteW}
    <!-- drag handle on the left edge of the note column (desktop): change its width,
         double-click = default width -->
    <!-- svelte-ignore a11y_no_noninteractive_element_interactions, a11y_no_noninteractive_tabindex -->
    <div
      class="absolute inset-y-0 z-10 hidden w-2 -translate-x-1/2 cursor-col-resize touch-none transition-colors hover:bg-accent-400/40 md:block {noteResizing ? 'bg-accent-500/50' : ''}"
      style:left="calc(100% - {noteW}px)"
      role="separator"
      aria-orientation="vertical"
      aria-valuenow={noteW}
      aria-valuemin={NOTE_MIN}
      aria-valuemax={NOTE_MAX}
      aria-label={t('checklist.resizeNotes')}
      title={t('checklist.resizeNotes')}
      tabindex="0"
      onpointerdown={startNoteResize}
      ondblclick={() => setPref('noteWidth', 0)}
      onkeydown={(e) => {
        if (e.key === 'ArrowLeft' || e.key === 'ArrowRight') {
          e.preventDefault();
          setPref('noteWidth', clampNote(noteW + (e.key === 'ArrowLeft' ? 16 : -16)));
        }
      }}></div>
  {/if}

  <div class="grid md:grid-cols-[minmax(0,1fr)_var(--note-w,18rem)] 2xl:grid-cols-[minmax(0,1fr)_var(--note-w,24rem)]">
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
    <div class="hidden pb-1.5 pl-4 text-[11px] font-semibold tracking-wider text-zinc-400 uppercase md:block" bind:offsetWidth={noteW}>{t('checklist.notes')}</div>
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
        class="group/row relative scroll-mt-20 {sectionRowCls(b.id)} {hidden.has(b.id) ? 'hidden' : 'grid'} md:grid-cols-[minmax(0,1fr)_var(--note-w,18rem)] 2xl:grid-cols-[minmax(0,1fr)_var(--note-w,24rem)]
          {isFocused && editable ? 'ring-1 ring-accent-400/80 dark:ring-accent-500/60' : ''}
          [&.dragging]:bg-white [&.dragging]:shadow-lg dark:[&.dragging]:bg-zinc-900">
        <!-- Main column -->
        <div class="relative flex min-w-0 items-start gap-2 py-(--dens-px) pr-2 {i > 0 ? (rowSpacing[b.kind] ?? '') : ''}" style:padding-left="{32 + b.indent * 24}px">
          {#if editable}
            <div class="absolute flex items-center" style:left="{b.indent * 24}px" style:top="{(i > 0 ? (spacingPx[b.kind] ?? 0) : 0) + (b.kind === "h1" ? 7 : b.kind === "h2" ? 5 : 3)}px">
              <button type="button" class="{gutterBtn} max-md:hidden" onclick={() => plus(b.id)} title={t('editor.addBlock')} aria-label={t('editor.addBlock')}>
                <Icon name="plus" size={13} />
              </button>
              <button type="button" data-handle class="{gutterBtn} cursor-grab touch-none" title={t('checklist.drag')} aria-label={t('checklist.drag')}>
                <Icon name="grip" size={14} />
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
                  model.remove(b.id);
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
                onlang={(lang) => model.update(b.id, { lang } as Partial<Blk>)}
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
                    else insertBlock(b.id, { kind: 'text', indent: b.indent } as Partial<Blk>, 'start');
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
            <!-- Placeholders like checkbox (right) and status: code block ends flush with the item text -->
            {#if prefs.checkPos === 'right'}<span class="ml-1 size-4 shrink-0" aria-hidden="true"></span>{/if}
            {#if isChecklist}<span class="w-20 shrink-0 md:w-28" aria-hidden="true"></span>{/if}
          {:else}
            <!-- Marker per block type -->
            {#if b.kind === 'item' && prefs.checkPos === 'left'}
              {@render checkbox(b, done, '')}
              {#if isChecklist}{@render statusField(b, '-ml-1')}{/if}
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
                class="rich min-h-5 outline-none {titleCls[b.kind]} {done ? 'text-zinc-600 line-through dark:text-zinc-400' : ''}
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
              <!-- Section color -->
              {@render bgPicker(b, iconMt[b.kind] ?? "", t("editor.sectionColor"), "max-md:opacity-100")}
            {/if}

            {#if isChecklist && b.kind === 'item'}
              <!-- Assignee -->
              {#if editField?.key === b.id && editField.field === 'assignee'}
                <select
                  class="input w-auto! py-0! text-xs!"
                  value={b.assignee_id ?? ''}
                  onblur={() => (editField = null)}
                  onchange={(e) => {
                    const v = e.currentTarget.value;
                    editField = null;
                    model.update(b.id, { assignee_id: v ? Number(v) : null, assignee_name: assignees.find((a) => a.id === Number(v))?.name ?? null } as Partial<Blk>);
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
                  class="mt-[3px] shrink-0 text-zinc-500 opacity-0 group-hover/row:opacity-100 hover:text-zinc-900 focus:opacity-100 max-md:hidden dark:text-zinc-400 dark:hover:text-zinc-100"
                  title={t('checklist.assign')}
                  onclick={() => (editField = { key: b.id, field: 'assignee' })}><Icon name="user" size={14} /></button>
              {/if}

              <!-- Due date -->
              {#if editField?.key === b.id && editField.field === 'due'}
                <input
                  type="date"
                  class="input w-auto! py-0! text-xs!"
                  value={b.due_date ?? ''}
                  onblur={() => (editField = null)}
                  onchange={(e) => model.update(b.id, { due_date: e.currentTarget.value || null } as Partial<Blk>)} />
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
                  class="mt-[3px] shrink-0 text-zinc-500 opacity-0 group-hover/row:opacity-100 hover:text-zinc-900 focus:opacity-100 max-md:hidden dark:text-zinc-400 dark:hover:text-zinc-100"
                  title={t('checklist.setDue')}
                  onclick={() => (editField = { key: b.id, field: 'due' })}><Icon name="calendar" size={14} /></button>
              {/if}
            {/if}

            {#if b.kind === 'item' && prefs.checkPos === 'right'}
              {@render checkbox(b, done, 'ml-1')}
            {/if}
            {#if b.kind === 'item' && isChecklist && prefs.checkPos === 'right'}
              {@render statusField(b, '-ml-1')}
            {/if}
          {/if}

          {#if editable}
            {#if editable && ['text', 'bullet', 'numbered', 'quote', 'h1', 'h2', 'h3', 'h4'].includes(b.kind)}
              <!-- convert into a checklist item (with checkbox) -->
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
              <!-- Row color -->
              {@render bgPicker(b, 'mt-px max-md:hidden', t('editor.rowColor'), '')}
            {/if}
            <span class="flex shrink-0 opacity-0 group-hover/row:opacity-100 focus-within:opacity-100 max-md:hidden">
              <ConfirmButton onconfirm={() => model.remove(b.id)} class="btn-danger p-0.5! leading-none"><Icon name="trash" size={14} /></ConfirmButton>
            </span>
          {/if}

          {#if slash?.key === b.id}
            <SlashMenu list={slashList} active={slash.active} onselect={chooseSlash} onhover={(n) => slash && (slash.active = n)} />
          {/if}
        </div>

        <!-- Note -->
        <div
          class="min-w-0 items-start gap-2.5 pb-1 pl-8 bg-white md:border-l md:border-zinc-500 md:py-(--dens-px) md:pl-4 dark:bg-zinc-900 dark:md:border-zinc-500 {i > 0 ? (noteSpacing[b.kind] ?? "") : ""}
            {b.description ? "flex" : "hidden group-focus-within/row:flex md:flex"}">
          {#if b.kind !== 'divider'}
            {#if b.description}
              <button
                type="button"
                disabled={!editable}
                onclick={() => model.update(b.id, { note_color: colorCycle[(colorCycle.indexOf(b.note_color) + 1) % colorCycle.length] } as Partial<Blk>)}
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
        insertBlock(last?.id ?? null, { kind: last?.kind === "item" ? "item" : "text", indent: last?.indent ?? 0 } as Partial<Blk>, "start");
      }}>
      <Icon name="plus" size={14} />
      {t('editor.addBlock')}
    </button>
  {:else if blocks.length === 0}
    <p class="py-6 pl-8 text-sm text-zinc-400">{t('checklist.empty')}</p>
  {/if}
  </div>
</div>
