// Reactive page header state per page and user: localStorage (everything) and
// URL query (the fields with `url`, so links can be shared). Must be called
// while a component initializes (uses $effect).
// External fields (`ext`) live elsewhere (e.g. the overview filters, synced in
// the database) and only take part in the URL.
import { untrack } from 'svelte';
import { page } from '$app/state';
import { replaceState } from '$app/navigation';
import { syncCurrentUrl } from './navHistory.svelte';
import { session } from './state.svelte';
import { defaults, fromQuery, initial, normQuery, toQuery, type Spec, type UrlKind, type Values } from './pageStateCore';

export interface ExtField {
  def: unknown;
  url: UrlKind;
  get: () => unknown;
  set: (v: never) => void;
  keep?: boolean; // missing in a new URL: keep the value instead of resetting it (e.g. the view)
}

const storageKey = (name: string) => `page-state:${session.user?.id ?? 0}:${name}`;

function read(key: string): string | null {
  try {
    return localStorage.getItem(key);
  } catch {
    return null;
  }
}

export function pageState<S extends Spec>(name: string, spec: S, ext: Record<string, ExtField> = {}): Values<S> {
  const key = storageKey(name);
  const extSpec: Spec = Object.fromEntries(Object.entries(ext).map(([k, f]) => [k, { def: f.def, url: f.url }]));
  const urlSpec: Spec = { ...spec, ...extSpec };
  const st = $state(initial(spec, read(key), page.url.search)) as Values<S>;
  let written = normQuery(page.url.search);

  const applyExt = (vals: Record<string, unknown>) => {
    for (const [k, f] of Object.entries(ext)) if (k in vals) f.set(vals[k] as never);
  };
  applyExt(fromQuery(page.url.search, extSpec)); // a shared link wins over the stored filters

  // save + mirror into the URL
  $effect(() => {
    const snap = $state.snapshot(st) as Values<S>;
    const extVals = Object.fromEntries(Object.entries(ext).map(([k, f]) => [k, $state.snapshot(f.get())]));
    try {
      localStorage.setItem(key, JSON.stringify(snap));
    } catch {
      /* ignore */
    }
    const q = toQuery({ ...snap, ...extVals } as Values<Spec>, urlSpec);
    untrack(() => {
      // compare with the real address: page.url is not updated by replaceState
      // (shallow routing), so after one rewrite it would be stale and a later
      // change back to the defaults (e.g. view "cards" = no ?view) never written
      const current = typeof location === 'undefined' ? page.url.search : location.search;
      if (normQuery(current) === q) return;
      written = q;
      const url = new URL(typeof location === 'undefined' ? page.url : location.href);
      url.search = q;
      try {
        replaceState(url, page.state);
        syncCurrentUrl(url); // history of the back bar (e.g. overview view icon)
      } catch {
        /* router not ready yet */
      }
    });
  });

  // URL changed from outside (saved view, link, nav click on the same page):
  // fields missing in the URL go back to their defaults
  $effect(() => {
    const search = page.url.search; // dependency: runs on every real navigation
    // value from the real address (page.url may be stale after our own replaceState)
    const q = normQuery(typeof location === 'undefined' ? search : location.search);
    untrack(() => {
      if (q === written) return;
      written = q;
      const d = defaults(urlSpec);
      const resettable = Object.entries(urlSpec).filter(([k, f]) => f.url && !ext[k]?.keep);
      const vals = { ...Object.fromEntries(resettable.map(([k]) => [k, d[k]])), ...fromQuery(q, urlSpec) };
      for (const k of Object.keys(spec)) if (k in vals) (st as Record<string, unknown>)[k] = vals[k];
      applyExt(vals);
    });
  });

  return st;
}
