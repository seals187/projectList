import { describe, expect, it } from 'vitest';
import { customerSummary, dayMonth, fitTags, percent } from './listMetrics';

const p = (done: number, total: number, extra: Partial<Parameters<typeof customerSummary>[0][number]> = {}) => ({
  done,
  total,
  status: 'active',
  due_date: null as string | null,
  include_in_customer_progress: true,
  ...extra
});

describe('steps and percent', () => {
  it('percent of done steps, rounded; no steps = 0', () => {
    expect(percent(2, 8)).toBe(25);
    expect(percent(3, 31)).toBe(10);
    expect(percent(1, 3)).toBe(33);
    expect(percent(0, 0)).toBe(0);
    expect(percent(8, 8)).toBe(100);
  });

  it('customer: steps and progress only over included projects, open over all', () => {
    const s = customerSummary([p(2, 8), p(3, 31), p(1, 12, { include_in_customer_progress: false })], '2026-11-06');
    expect(s).toMatchObject({ projects: 3, done: 5, total: 39, pct: 13, open: 6 + 28 + 11 });
  });

  it('customer: next due date of a running project, today or later', () => {
    const s = customerSummary(
      [
        p(0, 1, { due_date: '2026-11-01' }), // past
        p(0, 1, { due_date: '2026-11-20' }),
        p(0, 1, { due_date: '2026-11-13' }),
        p(1, 1, { due_date: '2026-11-10', status: 'done' }) // finished
      ],
      '2026-11-06'
    );
    expect(s.next).toBe('2026-11-13');
    expect(dayMonth(s.next!)).toBe('13.11.');
  });
});

describe('tag shortening', () => {
  it('shows all tags when they fit', () => {
    expect(fitTags([50, 60, 40], 170, 24, 10)).toBe(3);
  });
  it('keeps room for "+N" when not all fit', () => {
    // 50 + 10 + 60 = 120, + 10 + 24 = 154 <= 160 → 2 tags + "+1"
    expect(fitTags([50, 60, 40], 160, 24, 10)).toBe(2);
    expect(fitTags([50, 60, 40], 100, 24, 10)).toBe(1);
    expect(fitTags([50, 60, 40], 30, 24, 10)).toBe(0);
  });
  it('no tags = nothing to show', () => {
    expect(fitTags([], 100, 24, 10)).toBe(0);
  });
});
