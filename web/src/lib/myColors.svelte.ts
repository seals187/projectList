// Personal card colors of the overview (pref "card_colors", synced across the
// user's devices; only visible to the user). The shared customer color with
// "use in all views" wins over them (see cardColors.ts).
import { parseMyColors, withMyColor, type CardView, type MyColors } from './cardColors';
import { syncedPref } from './syncedPref.svelte';

const pref = syncedPref<MyColors>('card_colors', { parse: parseMyColors, debounce: 300 });

/** All personal colors (reactive, read only; change them with setMyColor). */
export const myColors: { readonly all: MyColors } = {
  get all() {
    return pref.value;
  }
};

/** Set one personal color ("" removes it): shown at once, saved debounced. */
export function setMyColor(view: CardView, id: number, color: string) {
  pref.set((all) => withMyColor(all, view, id, color));
}
