// Size of the content of the main pages (overview, My tasks, customers,
// templates, reporting): 70–110 % in 5 % steps, per page and user (synced with
// the "ui" pref). The page header keeps its size; only the content is zoomed.

export const ZOOM_PAGES = ['overview', 'tasks', 'customers', 'templates', 'reports'] as const;
export type ZoomPage = (typeof ZOOM_PAGES)[number];
export type PageZoom = Partial<Record<ZoomPage, number>>;

export const PAGE_ZOOM_STEPS: readonly number[] = [0.7, 0.75, 0.8, 0.85, 0.9, 0.95, 1, 1.05, 1.1];

/** Validated per-page zoom (unknown pages and values are dropped; 100 % is not stored). */
export function parsePageZoom(v: unknown): PageZoom {
  const out: PageZoom = {};
  if (!v || typeof v !== 'object' || Array.isArray(v)) return out;
  for (const page of ZOOM_PAGES) {
    const z = (v as Record<string, unknown>)[page];
    if (typeof z === 'number' && PAGE_ZOOM_STEPS.includes(z) && z !== 1) out[page] = z;
  }
  return out;
}

/** One step smaller / bigger (dir ±1), 0 = back to 100 %. */
export function stepPageZoom(current: number, dir: -1 | 0 | 1): number {
  if (dir === 0) return 1;
  const i = PAGE_ZOOM_STEPS.indexOf(current);
  const base = i < 0 ? PAGE_ZOOM_STEPS.indexOf(1) : i;
  return PAGE_ZOOM_STEPS[Math.min(PAGE_ZOOM_STEPS.length - 1, Math.max(0, base + dir))];
}

/** Copy with the zoom of one page set (100 % removes the entry). */
export function withPageZoom(all: PageZoom, page: ZoomPage, zoom: number): PageZoom {
  const next = { ...all };
  if (zoom === 1) delete next[page];
  else next[page] = zoom;
  return next;
}
