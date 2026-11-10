// Central type system: icon and colors per object type, used everywhere in the
// app (sidebar, search, breadcrumbs, page headers, lists). Colors are CSS
// variables (app.css: --type-<name>-bg / -fg, with dark variants); components
// render types only through <EntityIcon> and entityLabel() – no hard-coded
// icons or colors for these types elsewhere.

export type EntityType = 'customer' | 'project' | 'checklist' | 'table' | 'template' | 'savedView' | 'favGroup';

export type EntityIconName = 'building2' | 'folder' | 'listChecks' | 'sheet' | 'fileText' | 'filter' | 'bookmark';

export interface EntityTypeDef {
  icon: EntityIconName; // lucide shape (Icon.svelte)
  css: string; // CSS variable stem: --type-<css>-bg / --type-<css>-fg
  scale?: number; // optical size correction of the icon (shapes that fill their box more)
}

export const ENTITY_TYPES: Record<EntityType, EntityTypeDef> = {
  customer: { icon: 'building2', css: 'customer' },
  project: { icon: 'folder', css: 'project' },
  checklist: { icon: 'listChecks', css: 'checklist', scale: 0.82 },
  table: { icon: 'sheet', css: 'table', scale: 0.82 },
  template: { icon: 'fileText', css: 'template' },
  savedView: { icon: 'filter', css: 'saved-view' },
  favGroup: { icon: 'bookmark', css: 'fav-group' }
};

/** Colors of the type as reference values (light theme, same as app.css; for tests and docs). */
export const ENTITY_COLORS: Record<EntityType, { bg: string; fg: string; darkBg: string; darkFg: string }> = {
  customer: { bg: '#ecfdf5', fg: '#047857', darkBg: '#0f2e27', darkFg: '#6ee7b7' },
  project: { bg: '#eff6ff', fg: '#1d4ed8', darkBg: '#172340', darkFg: '#93c5fd' },
  checklist: { bg: '#f5f3ff', fg: '#6d28d9', darkBg: '#261a45', darkFg: '#c4b5fd' },
  table: { bg: '#ecfeff', fg: '#0e7490', darkBg: '#0c2a33', darkFg: '#67e8f9' },
  template: { bg: '#fff7ed', fg: '#c2410c', darkBg: '#3a1d0c', darkFg: '#fdba74' },
  savedView: { bg: 'transparent', fg: '#9ca3af', darkBg: 'transparent', darkFg: '#a1a1aa' },
  favGroup: { bg: 'transparent', fg: '#9ca3af', darkBg: 'transparent', darkFg: '#a1a1aa' }
};

/** Box and icon size per size step (px). */
export const ENTITY_SIZES = { xxs: { box: 14, icon: 9 }, xs: { box: 18, icon: 11 }, sm: { box: 22, icon: 13 }, md: { box: 26, icon: 15 }, lg: { box: 32, icon: 18 } } as const; // xxs: calendar entries, xs: breadcrumbs
export type EntitySize = keyof typeof ENTITY_SIZES;

/** i18n key of the type name ("Projekt", "Checkliste" …). */
export const entityLabelKey = (type: EntityType) => `entity.${type}` as const;

/** "Projekt · Kantonsspital Zug" – type and context, or only the type. */
export const entitySubtitle = (label: string, context: string) => (context ? `${label} · ${context}` : label);

/** Relative luminance and contrast ratio (WCAG) of two #rrggbb colors. */
export function contrast(a: string, b: string): number {
  const lum = (hex: string) => {
    const [r, g, bl] = [1, 3, 5].map((i) => parseInt(hex.slice(i, i + 2), 16) / 255).map((c) => (c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4));
    return 0.2126 * r + 0.7152 * g + 0.0722 * bl;
  };
  const [x, y] = [lum(a), lum(b)].sort((p, q) => q - p);
  return (x + 0.05) / (y + 0.05);
}
