import type { Kind, Row } from './types';
import type { Key } from './i18n.svelte';

/** Block types in the order of the type menu. */
export const KINDS: { kind: Kind; label: Key; desc: Key; icon: string; group: 'headings' | 'blocks' }[] = [
  { kind: 'h1', label: 'blocks.h1', desc: 'blocks.h1Desc', icon: 'H1', group: 'headings' },
  { kind: 'h2', label: 'blocks.h2', desc: 'blocks.h2Desc', icon: 'H2', group: 'headings' },
  { kind: 'h3', label: 'blocks.h3', desc: 'blocks.h3Desc', icon: 'H3', group: 'headings' },
  { kind: 'h4', label: 'blocks.h4', desc: 'blocks.h4Desc', icon: 'H4', group: 'headings' },
  { kind: 'item', label: 'blocks.item', desc: 'blocks.itemDesc', icon: 'item', group: 'blocks' },
  { kind: 'text', label: 'blocks.text', desc: 'blocks.textDesc', icon: 'text', group: 'blocks' },
  { kind: 'bullet', label: 'blocks.bullet', desc: 'blocks.bulletDesc', icon: 'bullet', group: 'blocks' },
  { kind: 'numbered', label: 'blocks.numbered', desc: 'blocks.numberedDesc', icon: 'numbered', group: 'blocks' },
  { kind: 'quote', label: 'blocks.quote', desc: 'blocks.quoteDesc', icon: 'quote', group: 'blocks' },
  { kind: 'code', label: 'blocks.code', desc: 'blocks.codeDesc', icon: 'code', group: 'blocks' },
  { kind: 'divider', label: 'blocks.divider', desc: 'blocks.dividerDesc', icon: 'divider', group: 'blocks' }
];

export const isHeading = (k: Kind) => k === 'h1' || k === 'h2' || k === 'h3' || k === 'h4';
/** Blocks with editable inline text. */
export const isTextual = (k: Kind) => k !== 'code' && k !== 'divider';

/** Type of the new block after Enter. */
export function nextKind(k: Kind): Kind {
  return k === 'item' || k === 'bullet' || k === 'numbered' ? k : 'text';
}

/** Markdown shortcut at line start → block type. */
export const SHORTCUTS: [RegExp, Kind][] = [
  [/^####\s$/, 'h4'],
  [/^###\s$/, 'h3'],
  [/^##\s$/, 'h2'],
  [/^#\s$/, 'h1'],
  [/^\[\s?\]\s$/, 'item'],
  [/^[-*]\s$/, 'bullet'],
  [/^1[.)]\s$/, 'numbered'],
  [/^>\s$/, 'quote'],
  [/^```$/, 'code'],
  [/^---$/, 'divider']
];

/** Numbering for "numbered" blocks (consecutive per indent level). */
export function numbering(rows: Row[]): Map<number, number> {
  const out = new Map<number, number>();
  const counters: number[] = [];
  for (const r of rows) {
    const lvl = r.indent ?? 0;
    if (r.kind === 'numbered') {
      counters.length = lvl + 1;
      counters[lvl] = (counters[lvl] ?? 0) + 1;
      out.set(r.id, counters[lvl]);
    } else {
      counters.length = lvl;
    }
  }
  return out;
}

export interface SectionStat {
  id: number;
  title: string;
  done: number;
  total: number;
}

/** H1 sections with progress of the following items (up to the next H1). */
export function computeSections(items: (Row & { done_at?: string | null })[], idOf: (r: Row) => number = (r) => r.id): SectionStat[] {
  const out: SectionStat[] = [];
  let cur: SectionStat | null = null;
  for (const it of items) {
    if (it.kind === 'h1') {
      cur = { id: idOf(it), title: it.title, done: 0, total: 0 };
      out.push(cur);
    } else if (it.kind === 'item' && cur) {
      cur.total++;
      if (it.done_at) cur.done++;
    }
  }
  return out;
}

/** Level of a heading (1–4), otherwise 0. */
export const headingLevel = (k: Kind) => (isHeading(k) ? Number(k[1]) : 0);

export interface HeadingStat {
  blocks: number; // blocks in the section (up to the next heading of the same or higher level)
  done: number;
  total: number;
}

/**
 * Progress and size per heading. A section extends up to the next heading
 * of the same or higher level (so H2 ends at H1 or H2).
 */
export function headingStats(rows: (Row & { done_at?: string | null })[]): Map<number, HeadingStat> {
  const out = new Map<number, HeadingStat>();
  const open: { level: number; stat: HeadingStat }[] = [];
  for (const r of rows) {
    const lvl = headingLevel(r.kind);
    if (lvl) while (open.length && open[open.length - 1].level >= lvl) open.pop();
    for (const o of open) {
      o.stat.blocks++;
      if (r.kind === 'item') {
        o.stat.total++;
        if (r.done_at) o.stat.done++;
      }
    }
    if (lvl) {
      const stat = { blocks: 0, done: 0, total: 0 };
      out.set(r.id, stat);
      open.push({ level: lvl, stat });
    }
  }
  return out;
}


/**
 * Section color per block: a heading's color applies to its whole section;
 * nested colored sections take precedence. `region` is the ID of the
 * heading the color comes from (to separate adjacent sections).
 */
export function sectionColors(rows: (Row & { bg_color?: string })[]): Map<number, { color: string; region: number }> {
  const out = new Map<number, { color: string; region: number }>();
  const open: { level: number; id: number; color: string }[] = [];
  for (const r of rows) {
    const lvl = headingLevel(r.kind);
    if (lvl) {
      while (open.length && open[open.length - 1].level >= lvl) open.pop();
      open.push({ level: lvl, id: r.id, color: r.bg_color ?? '' });
    } else if (r.bg_color === 'none') {
      continue; // explicitly no background, even in a colored section
    } else if (r.bg_color) {
      // own row color takes precedence over the section color (own region)
      out.set(r.id, { color: r.bg_color, region: r.id });
      continue;
    }
    for (let i = open.length - 1; i >= 0; i--) {
      if (open[i].color) {
        out.set(r.id, { color: open[i].color, region: open[i].id });
        break;
      }
    }
  }
  return out;
}

/** IDs of blocks hidden under collapsed headings. */
export function hiddenBlocks(rows: Row[], collapsed: { has(id: number): boolean }): Set<number> {
  const out = new Set<number>();
  let hideLevel = 0; // 0 = nothing hidden
  for (const r of rows) {
    const lvl = headingLevel(r.kind);
    if (hideLevel && lvl && lvl <= hideLevel) hideLevel = 0;
    if (hideLevel) {
      out.add(r.id);
      continue;
    }
    if (lvl && collapsed.has(r.id)) hideLevel = lvl;
  }
  return out;
}

/** Progress (only checklist items count). */
export function countProgress(items: (Row & { done_at?: string | null })[]) {
  let done = 0,
    total = 0;
  for (const it of items) {
    if (it.kind !== 'item') continue;
    total++;
    if (it.done_at) done++;
  }
  return { done, total };
}

/** Filters the block types for the slash menu. */
export function filterKinds(query: string, tr: (k: Key) => string) {
  const q = query.trim().toLowerCase();
  return KINDS.filter((k) => !q || tr(k.label).toLowerCase().includes(q) || k.kind.startsWith(q) || tr(k.desc).toLowerCase().includes(q));
}
