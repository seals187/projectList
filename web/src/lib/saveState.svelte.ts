/** Save status for the "saved" indicator (counted by api.ts for every write). */
export const saveState = $state({ pending: 0, saved: false, error: false });
