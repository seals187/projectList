import { describe, expect, it } from 'vitest';
import { allCollapsed, fromQuery, fromStored, initial, normQuery, toQuery, type Spec } from './pageStateCore';

const spec = {
  view: { def: 'cards' as string, url: 'str' },
  q: { def: '', url: 'str' },
  cust: { def: null as number[] | null, url: 'ids' },
  due: { def: null as string[] | null, url: 'strs' },
  open: { def: [] as number[] } // localStorage only
} satisfies Spec;

describe('pageStateCore', () => {
  it('leaves defaults out of the URL and sorts the keys', () => {
    expect(toQuery({ view: 'cards', q: '', cust: null, due: null, open: [1] }, spec)).toBe('');
    expect(toQuery({ view: 'list', q: 'abc', cust: [3, 1], due: ['today'], open: [1] }, spec)).toBe('cust=3%2C1&due=today&q=abc&view=list');
  });

  it('reads the URL fields back (numbers stay numbers, empty list = none)', () => {
    expect(fromQuery('view=list&cust=3,1&due=today,week&open=9', spec)).toEqual({ view: 'list', cust: [3, 1], due: ['today', 'week'] });
    expect(fromQuery('cust=', spec)).toEqual({ cust: [] });
  });

  it('round-trips through the URL', () => {
    const v = { view: 'board', q: 'x y', cust: [7], due: ['none'], open: [] as number[] };
    expect({ ...initial(spec, null, ''), ...fromQuery(toQuery(v, spec), spec) }).toEqual(v);
  });

  it('ignores stored values of the wrong type', () => {
    expect(fromStored('{"view":5,"q":"hi","cust":"x","open":[2],"other":1}', spec)).toEqual({ q: 'hi', open: [2] });
    expect(fromStored('not json', spec)).toEqual({});
  });

  it('starts from defaults, then storage, then the URL (a shared link wins)', () => {
    const v = initial(spec, '{"view":"list","q":"stored","open":[4]}', '?view=calendar');
    expect(v).toEqual({ view: 'calendar', q: 'stored', cust: null, due: null, open: [4] });
  });

  it('normalizes queries for comparison', () => {
    expect(normQuery('?b=2&a=1')).toBe(normQuery('a=1&b=2'));
  });

  it('collapse toggle: only "all collapsed" offers expand; mixed counts as open', () => {
    expect(allCollapsed([false, false])).toBe(true);
    expect(allCollapsed([true, false])).toBe(false);
    expect(allCollapsed([true, true])).toBe(false);
    expect(allCollapsed([])).toBe(false);
  });
});
