import { describe, expect, it } from 'vitest';
import { dayMonth, linePoints, moveItem, niceMax, pctChange, pickPeriod, pickView, reportQuery, signed, swissDate, toCsv, type ReportView } from './reporting';

describe('reporting helpers', () => {
  it('query string', () => {
    expect(reportQuery({ period: 30, customers: [], assignees: [], tags: [] })).toBe('period=30');
    expect(reportQuery({ period: 7, customers: [1, 2], assignees: [3], tags: [4] })).toBe('period=7&customers=1%2C2&assignees=3&tags=4');
  });
  it('changes', () => {
    expect(pctChange(142, 120)).toBe(18);
    expect(pctChange(5, 0)).toBeNull();
    expect(signed(18, ' %')).toBe('+18 %');
    expect(signed(-9, ' %')).toBe('−9 %');
    expect(signed(0)).toBe('±0');
  });
  it('dates', () => {
    expect(dayMonth('2026-11-13')).toBe('13.11.');
    expect(dayMonth('')).toBe('–');
    expect(swissDate('2026-10-01')).toBe('01.10.2026');
  });
  it('csv', () => {
    expect(toCsv([['a', 1], ['x;y', 'say "hi"']])).toBe('﻿a;1\r\n"x;y";"say ""hi"""');
  });
  it('chart geometry', () => {
    expect(linePoints([0, 5, 10], 100, 50, 10)).toEqual([
      [0, 50],
      [50, 25],
      [100, 0]
    ]);
    expect(niceMax(0)).toBe(4);
    expect(niceMax(7)).toBe(10);
    expect(niceMax(130)).toBe(200);
  });
});

describe('reporting settings helpers', () => {
  const cfg = { periods: [14, 60], default_period: 60, views: ['dashboard', 'workload'] as ReportView[], default_view: 'workload' as const };
  it('falls back to the defaults of the settings', () => {
    expect(pickPeriod('14', cfg)).toBe(14);
    expect(pickPeriod('30', cfg)).toBe(60);
    expect(pickPeriod('', cfg)).toBe(60);
    expect(pickView('dashboard', cfg)).toBe('dashboard');
    expect(pickView('customers', cfg)).toBe('workload');
  });
  it('moves entries up and down, not past the ends', () => {
    expect(moveItem(['a', 'b', 'c'], 1, -1)).toEqual(['b', 'a', 'c']);
    expect(moveItem(['a', 'b', 'c'], 1, 1)).toEqual(['a', 'c', 'b']);
    expect(moveItem(['a', 'b'], 0, -1)).toEqual(['a', 'b']);
  });
});
