import { describe, expect, it } from 'vitest';
import { computeSections, countProgress, headingLevel, headingStats, hiddenBlocks, nextKind, numbering, sectionColors } from './blocks';
import type { Kind, Row } from './types';

type R = Row & { done_at?: string | null; bg_color?: string };

/** Rows from a compact spec: [kind, indent?, done?] with ids 1..n. */
function rows(spec: (Kind | [Kind, number?, boolean?])[]): R[] {
  return spec.map((s, i) => {
    const [kind, indent = 0, done = false] = Array.isArray(s) ? s : [s];
    return { id: i + 1, kind, title: `t${i + 1}`, description: '', note_color: '', indent, lang: '', position: i, done_at: done ? 'x' : null };
  });
}

describe('numbering', () => {
  it('counts consecutive numbered blocks and restarts after another block', () => {
    const n = numbering(rows(['numbered', 'numbered', 'text', 'numbered']));
    expect([...n]).toEqual([[1, 1], [2, 2], [4, 1]]);
  });

  it('numbers each indent level separately and resets deeper levels', () => {
    const n = numbering(rows(['numbered', ['numbered', 1], ['numbered', 1], 'numbered', ['numbered', 1]]));
    expect([...n]).toEqual([[1, 1], [2, 1], [3, 2], [4, 2], [5, 1]]);
  });

  it('a deeper non-numbered block keeps the outer count', () => {
    const n = numbering(rows(['numbered', ['text', 1], 'numbered']));
    expect(n.get(3)).toBe(2);
  });

  it('a non-numbered block at the same level ends the run', () => {
    const n = numbering(rows([['numbered', 1], ['bullet', 1], ['numbered', 1]]));
    expect(n.get(3)).toBe(1);
  });
});

describe('headingStats', () => {
  it('counts blocks and items up to the next heading of the same or higher level', () => {
    const s = headingStats(rows(['h1', ['item', 0, true], 'h2', 'item', 'text', 'h2', ['item', 0, true], 'h1', 'item']));
    expect(s.get(1)).toEqual({ blocks: 6, done: 2, total: 3 });
    expect(s.get(3)).toEqual({ blocks: 2, done: 0, total: 1 });
    expect(s.get(6)).toEqual({ blocks: 1, done: 1, total: 1 });
    expect(s.get(8)).toEqual({ blocks: 1, done: 0, total: 1 });
  });

  it('a lower heading counts as a block of the higher section', () => {
    const s = headingStats(rows(['h2', 'h3', 'item']));
    expect(s.get(1)).toEqual({ blocks: 2, done: 0, total: 1 });
    expect(s.get(2)).toEqual({ blocks: 1, done: 0, total: 1 });
  });

  it('ignores blocks before the first heading', () => {
    const s = headingStats(rows(['item', 'h1']));
    expect([...s.keys()]).toEqual([2]);
    expect(s.get(2)).toEqual({ blocks: 0, done: 0, total: 0 });
  });
});

describe('hiddenBlocks', () => {
  const list = rows(['h1', 'text', 'h2', 'item', 'h1', 'text']);

  it('hides everything under a collapsed heading up to the next heading of the same or higher level', () => {
    expect([...hiddenBlocks(list, new Set([1]))]).toEqual([2, 3, 4]);
  });

  it('a collapsed subheading hides only its own section', () => {
    expect([...hiddenBlocks(list, new Set([3]))]).toEqual([4]);
  });

  it('a collapsed heading inside a hidden section does not extend the hidden range', () => {
    expect([...hiddenBlocks(list, new Set([1, 3]))]).toEqual([2, 3, 4]);
  });

  it('ignores collapsed ids of non-headings and nothing collapsed hides nothing', () => {
    expect(hiddenBlocks(list, new Set([2])).size).toBe(0);
    expect(hiddenBlocks(list, new Set()).size).toBe(0);
  });
});

describe('countProgress', () => {
  it('counts only checklist items', () => {
    expect(countProgress(rows([['item', 0, true], 'item', ['text', 0, true], 'h1']))).toEqual({ done: 1, total: 2 });
    expect(countProgress([])).toEqual({ done: 0, total: 0 });
  });
});

describe('computeSections', () => {
  it('lists H1 sections with their item progress', () => {
    const s = computeSections(rows(['item', 'h1', ['item', 0, true], 'h2', 'item', 'h1']));
    expect(s).toEqual([
      { id: 2, title: 't2', done: 1, total: 2 },
      { id: 6, title: 't6', done: 0, total: 0 }
    ]);
  });
});

describe('sectionColors', () => {
  it('colors a heading section, own row colors win, "none" opts out, nested sections take precedence', () => {
    const list = rows(['h1', 'text', 'text', 'text', 'h2', 'text', 'h1', 'text']);
    list[0].bg_color = 'blue';
    list[2].bg_color = 'red';
    list[3].bg_color = 'none';
    list[4].bg_color = 'green';
    const c = sectionColors(list);
    expect(c.get(1)).toEqual({ color: 'blue', region: 1 });
    expect(c.get(2)).toEqual({ color: 'blue', region: 1 });
    expect(c.get(3)).toEqual({ color: 'red', region: 3 });
    expect(c.has(4)).toBe(false);
    expect(c.get(6)).toEqual({ color: 'green', region: 5 });
    expect(c.has(7)).toBe(false);
    expect(c.has(8)).toBe(false);
  });
});

describe('kind helpers', () => {
  it('headingLevel and nextKind', () => {
    expect(['h1', 'h4', 'text'].map((k) => headingLevel(k as Kind))).toEqual([1, 4, 0]);
    expect((['item', 'bullet', 'numbered', 'quote', 'h1', 'code'] as Kind[]).map(nextKind)).toEqual(['item', 'bullet', 'numbered', 'text', 'text', 'text']);
  });
});
