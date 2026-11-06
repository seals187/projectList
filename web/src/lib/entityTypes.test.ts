// @ts-expect-error -- no @types/node in this project; vitest runs in Node
import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';
import { contrast, ENTITY_COLORS, ENTITY_TYPES, entitySubtitle, type EntityType } from './entityTypes';

const css: string = readFileSync('src/app.css', 'utf8'); // vitest runs in web/
const block = (sel: string) => css.slice(css.indexOf(`${sel} {\n  --type-customer-bg`), css.indexOf('}', css.indexOf(`${sel} {\n  --type-customer-bg`)));

describe('entity types', () => {
  it('every type has an icon and a CSS stem', () => {
    expect(Object.keys(ENTITY_TYPES).sort()).toEqual(['checklist', 'customer', 'favGroup', 'project', 'savedView', 'template']);
    expect(ENTITY_TYPES.customer.icon).toBe('building2');
    expect(ENTITY_TYPES.favGroup.icon).toBe('bookmark');
  });

  it('app.css defines the same colors as ENTITY_COLORS (light and dark)', () => {
    const light = block(':root');
    const dark = block('.dark');
    for (const [type, c] of Object.entries(ENTITY_COLORS) as [EntityType, (typeof ENTITY_COLORS)[EntityType]][]) {
      const stem = ENTITY_TYPES[type].css;
      expect(light).toContain(`--type-${stem}-bg: ${c.bg};`);
      expect(light).toContain(`--type-${stem}-fg: ${c.fg};`);
      expect(dark).toContain(`--type-${stem}-bg: ${c.darkBg};`);
      expect(dark).toContain(`--type-${stem}-fg: ${c.darkFg};`);
    }
  });

  it('icon color on its box has at least 4.5:1 contrast (light and dark)', () => {
    for (const c of Object.values(ENTITY_COLORS)) {
      if (c.bg === 'transparent') continue;
      expect(contrast(c.fg, c.bg)).toBeGreaterThanOrEqual(4.5);
      expect(contrast(c.darkFg, c.darkBg)).toBeGreaterThanOrEqual(4.5);
    }
  });

  it('subtitle "type · context"', () => {
    expect(entitySubtitle('Projekt', 'Kantonsspital Zug')).toBe('Projekt · Kantonsspital Zug');
    expect(entitySubtitle('Kunde', '')).toBe('Kunde');
  });
});
