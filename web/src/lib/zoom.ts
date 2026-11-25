// Zoom steps: checklist size and the content size of the main pages (view prefs).
import { prefs, setPref, ZOOM_STEPS } from './state.svelte';
import { stepPageZoom, withPageZoom, type ZoomPage } from './pageZoom';

/** Change the checklist size by one step (dir = +1 / -1); 0 resets to 100 %. */
export function stepZoom(dir: -1 | 0 | 1) {
  const i = ZOOM_STEPS.indexOf(prefs.zoom);
  const next = dir === 0 ? 1 : ZOOM_STEPS[Math.min(ZOOM_STEPS.length - 1, Math.max(0, (i < 0 ? ZOOM_STEPS.indexOf(1) : i) + dir))];
  setPref('zoom', next);
}

/** Content size of a main page one step smaller / bigger (dir ±1), 0 = 100 %. */
export function stepPageZoomOf(page: ZoomPage, dir: -1 | 0 | 1) {
  setPref('pageZoom', withPageZoom(prefs.pageZoom, page, stepPageZoom(prefs.pageZoom[page] ?? 1, dir)));
}
