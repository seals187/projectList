// Personal card colors of the overview (pref "card_colors", synced across the
// user's devices; only visible to the user). The shared customer color with
// "use in all views" wins over them (see cardColors.ts).
import { api, errorText } from './api';
import { sidebar, toast } from './state.svelte';
import { emptyMyColors, parseMyColors, withMyColor, type CardView, type MyColors } from './cardColors';

export const myColors = $state<{ all: MyColors }>({ all: emptyMyColors() });

let saveTimer: ReturnType<typeof setTimeout> | undefined;

/** Take over the server value when the sidebar data arrives (unless a change is pending). */
export function syncMyColors() {
  if (!saveTimer) myColors.all = parseMyColors(sidebar.prefs.card_colors);
}

/** Set one personal color ("" removes it): shown at once, saved debounced. */
export function setMyColor(view: CardView, id: number, color: string) {
  myColors.all = withMyColor(myColors.all, view, id, color);
  clearTimeout(saveTimer);
  saveTimer = setTimeout(async () => {
    const v = $state.snapshot(myColors.all);
    try {
      await api.put('/prefs/card_colors', v);
      sidebar.prefs.card_colors = v;
    } catch (e) {
      toast(errorText(e));
    } finally {
      saveTimer = undefined;
    }
  }, 300);
}
