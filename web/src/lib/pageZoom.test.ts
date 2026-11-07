import { describe, expect, it } from 'vitest';
import { parsePageZoom, stepPageZoom, withPageZoom } from './pageZoom';

describe('page zoom', () => {
  it('steps within 70–110 %', () => {
    expect(stepPageZoom(1, 1)).toBe(1.05);
    expect(stepPageZoom(1.1, 1)).toBe(1.1);
    expect(stepPageZoom(0.75, -1)).toBe(0.7);
    expect(stepPageZoom(0.7, -1)).toBe(0.7);
    expect(stepPageZoom(0.85, 0)).toBe(1);
    expect(stepPageZoom(1.3, -1)).toBe(0.95); // unknown value: from 100 %
  });
  it('parses and stores per page', () => {
    expect(parsePageZoom({ overview: 0.9, tasks: 1, reports: 0.42, other: 0.8 })).toEqual({ overview: 0.9 });
    expect(parsePageZoom('x')).toEqual({});
    expect(withPageZoom({ overview: 0.9 }, 'reports', 0.8)).toEqual({ overview: 0.9, reports: 0.8 });
    expect(withPageZoom({ overview: 0.9 }, 'overview', 1)).toEqual({});
  });
});
