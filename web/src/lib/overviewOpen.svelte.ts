// Übersicht (Karten und Liste): welche Kunden ihre Projekte aufgeklappt zeigen.
// Standard: eingeklappt. Gemerkt werden nur die aufgeklappten Kunden (pro Browser).
import { SvelteSet } from 'svelte/reactivity';

const KEY = 'overview-open';

function load(): number[] {
  try {
    const v = JSON.parse(localStorage.getItem(KEY) ?? '[]');
    return Array.isArray(v) ? v.filter((x) => Number.isInteger(x)) : [];
  } catch {
    return [];
  }
}

export const openCustomers = new SvelteSet<number>(load());

function persist() {
  try {
    localStorage.setItem(KEY, JSON.stringify([...openCustomers]));
  } catch {
    /* ignorieren */
  }
}

export function toggleCustomer(id: number) {
  if (openCustomers.has(id)) openCustomers.delete(id);
  else openCustomers.add(id);
  persist();
}

/** Alle auf- bzw. zuklappen. */
export function setAllCustomers(ids: number[], open: boolean) {
  openCustomers.clear();
  if (open) ids.forEach((id) => openCustomers.add(id));
  persist();
}
