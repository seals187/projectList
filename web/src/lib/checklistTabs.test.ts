import { describe, expect, it } from 'vitest';
import { closeTab, MAX_TABS, moveTab, openTab, pruneTabs } from './checklistTabs';

describe('checklist tabs', () => {
  it('open: after the active tab, existing tabs stay where they are', () => {
    expect(openTab([], 1)).toEqual([1]);
    expect(openTab([1, 2, 3], 9, 1)).toEqual([1, 9, 2, 3]);
    expect(openTab([1, 2, 3], 9)).toEqual([1, 2, 3, 9]);
    expect(openTab([1, 2, 3], 2, 3)).toEqual([1, 2, 3]);
  });

  it('open: at most MAX_TABS, the oldest on the left goes', () => {
    const full = Array.from({ length: MAX_TABS }, (_, i) => i + 1);
    const r = openTab(full, 99, MAX_TABS);
    expect(r).toHaveLength(MAX_TABS);
    expect(r[0]).toBe(2);
    expect(r.at(-1)).toBe(99);
  });

  it('close: right neighbour, else left, else none', () => {
    expect(closeTab([1, 2, 3], 2)).toEqual({ ids: [1, 3], next: 3 });
    expect(closeTab([1, 2, 3], 3)).toEqual({ ids: [1, 2], next: 2 });
    expect(closeTab([1], 1)).toEqual({ ids: [], next: null });
    expect(closeTab([1, 2], 7)).toEqual({ ids: [1, 2], next: null });
  });

  it('move: before another tab or to the end', () => {
    expect(moveTab([1, 2, 3, 4], 4, 2)).toEqual([1, 4, 2, 3]);
    expect(moveTab([1, 2, 3, 4], 1, null)).toEqual([2, 3, 4, 1]);
    expect(moveTab([1, 2, 3], 2, 2)).toEqual([1, 2, 3]);
  });

  it('prune: deleted / no longer visible checklists disappear', () => {
    expect(pruneTabs([1, 2, 3], (id) => id !== 2)).toEqual([1, 3]);
  });
});
