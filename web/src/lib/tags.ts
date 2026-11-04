// Project tags: free color (#rrggbb) as chip background with a readable text color.
import type { Tag } from './types';
import { sidebar } from './state.svelte';

/**
 * Suggested colors in the pickers (tags, project status): 18 hues in a light,
 * a medium and a dark shade. Any other color via the color input.
 */
export const TAG_PRESETS = [
  // light
  '#94a3b8', '#f87171', '#fb923c', '#fbbf24', '#facc15', '#a3e635', '#4ade80', '#34d399', '#2dd4bf',
  '#22d3ee', '#38bdf8', '#60a5fa', '#818cf8', '#a78bfa', '#c084fc', '#e879f9', '#f472b6', '#fb7185',
  // medium
  '#64748b', '#dc2626', '#ea580c', '#d97706', '#ca8a04', '#65a30d', '#16a34a', '#059669', '#0d9488',
  '#0891b2', '#0284c7', '#2563eb', '#4f46e5', '#7c3aed', '#9333ea', '#c026d3', '#db2777', '#e11d48',
  // dark
  '#1e293b', '#991b1b', '#9a3412', '#92400e', '#854d0e', '#3f6212', '#166534', '#065f46', '#115e59',
  '#155e75', '#075985', '#1e40af', '#3730a3', '#5b21b6', '#6b21a8', '#86198f', '#9d174d', '#9f1239'
];

/** Black or white text, whichever contrasts better with the background (WCAG luminance). */
export function tagTextColor(hex: string): string {
  const ch = (i: number) => {
    const c = parseInt(hex.slice(i, i + 2), 16) / 255;
    return c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4;
  };
  const l = 0.2126 * ch(1) + 0.7152 * ch(3) + 0.0722 * ch(5);
  return l > 0.4 ? '#18181b' : '#ffffff';
}

/** Inline style of a tag chip. */
export const tagStyle = (color: string) => `background-color:${color};color:${tagTextColor(color)}`;

/** Resolves tag IDs to tags from the catalog (unknown IDs are skipped), sorted by name. */
export function tagsByIds(ids: readonly number[] | undefined): Tag[] {
  if (!ids?.length) return [];
  return sidebar.tags.filter((t) => ids.includes(t.id));
}
