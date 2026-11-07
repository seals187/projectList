import { describe, expect, it } from 'vitest';
import { cardHex, customerCardColor, customerWide, emptyMyColors, parseCardColor, parseMyColors, projectCardColor, projectColor, stripeStyle, tintStyle, withMyColor } from './cardColors';

const cust = (o: Partial<{ id: number; color: string; color_everywhere: boolean }> = {}) => ({ id: 1, color: '', color_everywhere: false, ...o });

describe('card colors', () => {
  const mine = withMyColor(withMyColor(withMyColor(emptyMyColors(), 'cards', 1, 'blue'), 'list', 1, 'green-strong'), 'board', 9, 'violet');

  it('personal colors per view without the shared customer color', () => {
    const c = cust({ color: 'red' }); // set, but not "use in all views"
    expect(customerCardColor(c, 'cards', mine)).toBe('blue');
    expect(customerCardColor(c, 'list', mine)).toBe('green-strong');
    expect(projectCardColor({ id: 9 }, c, mine)).toBe('violet');
    expect(projectCardColor({ id: 8 }, c, mine)).toBe('');
    expect(customerCardColor(cust({ id: 2 }), 'cards', mine)).toBe('');
  });

  it('shared customer color wins everywhere when switched on', () => {
    const c = cust({ color: 'red', color_everywhere: true });
    expect(customerCardColor(c, 'cards', mine)).toBe('red');
    expect(customerCardColor(c, 'list', mine)).toBe('red');
    expect(projectCardColor({ id: 9 }, c, mine)).toBe('red');
    expect(customerWide(c)).toBe('red');
    expect(customerWide(cust({ color: 'red' }))).toBe('');
    expect(customerWide(cust({ color_everywhere: true }))).toBe('');
  });

  it('withMyColor sets and removes without touching the input', () => {
    const a = emptyMyColors();
    const b = withMyColor(a, 'board', 5, 'teal');
    expect(a.board).toEqual({});
    expect(withMyColor(b, 'board', 5, '').board).toEqual({});
  });

  it('parseMyColors drops unknown colors, ids and views', () => {
    expect(parseMyColors({ cards: { '1': 'blue', x: 'red', '2': 'nope' }, board: { '3': 'red-strong' }, other: { '4': 'red' } })).toEqual({
      cards: { '1': 'blue' },
      list: {},
      board: { '3': 'red-strong' }
    });
    expect(parseMyColors(null)).toEqual(emptyMyColors());
    expect(parseMyColors('x')).toEqual(emptyMyColors());
  });

  it('shades, hex and tint style', () => {
    expect(parseCardColor('blue-strong')).toEqual({ base: 'blue', strong: true });
    expect(parseCardColor('nope-strong')).toBeNull();
    expect(cardHex('blue-strong')).toBe('#3b82f6');
    expect(tintStyle('blue')).toBe('--tint:#3b82f6');
    expect(tintStyle('blue-strong')).toBe('--tint:#3b82f6;--tint-l:30%;--tint-d:40%');
    expect(tintStyle('')).toBeUndefined();
  });
});

describe('project color stripe', () => {
  it('personal project color as a stripe', () => {
    const mine = withMyColor(emptyMyColors(), 'board', 7, 'blue-strong');
    expect(projectColor(7, mine)).toBe('blue-strong');
    expect(projectColor(8, mine)).toBe('');
    expect(stripeStyle('blue-strong')).toBe('box-shadow: inset 3px 0 0 #3b82f6');
    expect(stripeStyle('')).toBeUndefined();
  });
});
