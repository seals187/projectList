// Open checklist tabs of the signed-in user, kept in localStorage (per browser).
import { session } from './state.svelte';

export const checklistTabs = $state<{ ids: number[]; loadedFor: number | null }>({ ids: [], loadedFor: null });

const key = () => `checklist-tabs:${session.user?.id ?? 0}`;

/** Loads the tabs once per user (no-op afterwards). */
export function loadTabs() {
  const uid = session.user?.id ?? 0;
  if (checklistTabs.loadedFor === uid) return;
  try {
    const v = JSON.parse(localStorage.getItem(key()) ?? '[]');
    checklistTabs.ids = Array.isArray(v) ? v.filter((x) => Number.isInteger(x)) : [];
  } catch {
    checklistTabs.ids = [];
  }
  checklistTabs.loadedFor = uid;
}

export function setTabs(ids: number[]) {
  checklistTabs.ids = ids;
  try {
    localStorage.setItem(key(), JSON.stringify(ids));
  } catch {
    /* ignore */
  }
}
