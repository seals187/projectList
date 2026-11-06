import { describe, expect, it } from 'vitest';
import { render } from '@testing-library/svelte';
import { ENTITY_SIZES, ENTITY_TYPES, type EntityType } from '$lib/entityTypes';
import EntityIcon from './EntityIcon.svelte';

describe('EntityIcon', () => {
  for (const type of Object.keys(ENTITY_TYPES) as EntityType[]) {
    it(`renders ${type} with its colors`, () => {
      const { container } = render(EntityIcon, { props: { type, size: 'md' } });
      const box = container.querySelector(`[data-entity="${type}"]`) as HTMLElement;
      expect(box).toBeTruthy();
      expect(box.style.backgroundColor).toBe(`var(--type-${ENTITY_TYPES[type].css}-bg)`);
      expect(box.style.color).toBe(`var(--type-${ENTITY_TYPES[type].css}-fg)`);
      expect(box.style.width).toBe(`${ENTITY_SIZES.md.box}px`);
      expect(box.querySelector('svg')?.getAttribute('width')).toBe(String(Math.round(ENTITY_SIZES.md.icon * (ENTITY_TYPES[type].scale ?? 1))));
    });
  }

  it('sizes sm 22, md 26, lg 32 px and radius 7 px', () => {
    for (const [size, px] of [['sm', 22], ['md', 26], ['lg', 32]] as const) {
      const { container } = render(EntityIcon, { props: { type: 'project', size } });
      const box = container.querySelector('[data-entity]') as HTMLElement;
      expect(box.style.height).toBe(`${px}px`);
      expect(box.className).toContain('rounded-[7px]');
    }
  });
});
