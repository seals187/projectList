// Open checklists as tabs (pure functions, testable): a list of checklist ids
// in tab order. Opening adds a tab right after the active one (or activates an
// existing one), closing picks the neighbour to show next, drag & drop reorders.

export const MAX_TABS = 12;

/** Opens a checklist: already open → unchanged; otherwise inserted after `afterId` (or at the end). */
export function openTab(ids: readonly number[], id: number, afterId?: number | null): number[] {
  if (ids.includes(id)) return [...ids];
  const at = afterId != null && ids.includes(afterId) ? ids.indexOf(afterId) + 1 : ids.length;
  const out = [...ids.slice(0, at), id, ...ids.slice(at)];
  // too many tabs: drop the oldest ones on the left (never the new one)
  while (out.length > MAX_TABS) out.splice(out[0] === id ? 1 : 0, 1);
  return out;
}

/** Closes a tab; `next` = tab to show when the closed one was active (right neighbour, else left, else none). */
export function closeTab(ids: readonly number[], id: number): { ids: number[]; next: number | null } {
  const i = ids.indexOf(id);
  if (i < 0) return { ids: [...ids], next: null };
  const out = ids.filter((x) => x !== id);
  return { ids: out, next: out[i] ?? out[i - 1] ?? null };
}

/** Moves a tab before another one (null = to the end). */
export function moveTab(ids: readonly number[], id: number, beforeId: number | null): number[] {
  if (id === beforeId || !ids.includes(id)) return [...ids];
  const rest = ids.filter((x) => x !== id);
  const at = beforeId == null ? rest.length : rest.indexOf(beforeId);
  return at < 0 ? [...rest, id] : [...rest.slice(0, at), id, ...rest.slice(at)];
}

/** Keeps only checklists that still exist / are visible. */
export const pruneTabs = (ids: readonly number[], exists: (id: number) => boolean) => ids.filter(exists);
