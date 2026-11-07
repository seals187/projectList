import { describe, expect, it } from 'vitest';
import { dayMonth, linePoints, niceMax, pctChange, reportQuery, signed, swissDate, toCsv } from './reporting';

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
