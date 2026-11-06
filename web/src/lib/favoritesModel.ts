// Favorites in the sidebar (pure functions, testable): the DB rows (favorites
// with group and position, groups with position) as a tree for display and
// drag & drop (favLayout), back to the order the API stores, and the display
// data of each favorite (name, link, context line).
import type { FavLayout } from './favLayout';
import type { Favorite, FavoriteGroup, FavoriteKind, SidebarCustomer, TemplateSummary } from './types';

export const favKey = (f: { kind: FavoriteKind; id: number }) => `${f.kind}:${f.id}`;
export const groupKey = (id: number) => `g${id}`;

/** Ungrouped favorites first, then the groups – each ordered by position. */
export function favoritesLayout(favorites: readonly Favorite[], groups: readonly FavoriteGroup[]): FavLayout {
  const byPos = (list: Favorite[]) =>
    list
      .map((f, i) => ({ f, i }))
      .sort((a, b) => a.f.position - b.f.position || a.i - b.i)
      .map(({ f }) => favKey(f));
  const known = new Set(groups.map((g) => g.id));
  const root = favorites.filter((f) => f.group_id === null || !known.has(f.group_id));
  return [
    ...byPos(root).map((k) => ({ t: 'i' as const, k })),
    ...[...groups]
      .sort((a, b) => a.position - b.position || a.id - b.id)
      .map((g) => ({ t: 'f' as const, id: groupKey(g.id), name: g.name, open: !g.collapsed, items: byPos(favorites.filter((f) => f.group_id === g.id)) }))
  ];
}

export interface FavoritesOrder {
  groups: number[];
  items: { kind: FavoriteKind; id: number; group_id: number | null }[];
}

const parseKey = (k: string) => {
  const [kind, id] = k.split(':');
  return { kind: kind as FavoriteKind, id: Number(id) };
};

/** Order for PUT /api/favorites/order (favorites on the top level are "without group"). */
export function layoutToOrder(layout: FavLayout): FavoritesOrder {
  const groups: number[] = [];
  const root: FavoritesOrder['items'] = [];
  const grouped: FavoritesOrder['items'] = [];
  for (const n of layout) {
    if (n.t === 'i') root.push({ ...parseKey(n.k), group_id: null });
    else {
      const gid = Number(n.id.slice(1));
      groups.push(gid);
      for (const k of n.items) grouped.push({ ...parseKey(k), group_id: gid });
    }
  }
  return { groups, items: [...root, ...grouped] };
}

export interface FavDisplay {
  key: string;
  kind: FavoriteKind;
  id: number;
  href: string;
  name: string;
  context: string; // second line after the type: customer, project, number of projects / items
}

/**
 * Display data of the favorites the user can see (others – deleted, no access,
 * templates not loaded yet – are skipped).
 */
export function resolveFavorites(
  favorites: readonly Favorite[],
  customers: readonly SidebarCustomer[],
  templates: readonly TemplateSummary[] | null,
  tr: { nProjects: (n: number) => string; nItems: (n: number) => string }
): Map<string, FavDisplay> {
  const cust = new Map(customers.map((c) => [c.id, c]));
  const proj = new Map<number, { name: string; customer: string }>();
  const lists = new Map<number, { title: string; project: string }>();
  for (const c of customers)
    for (const p of c.projects) {
      proj.set(p.id, { name: p.name, customer: c.name });
      for (const cl of p.checklists) lists.set(cl.id, { title: cl.title, project: p.name });
    }
  const tpl = new Map((templates ?? []).map((x) => [x.id, x]));
  const out = new Map<string, FavDisplay>();
  for (const f of favorites) {
    const key = favKey(f);
    const base = { key, kind: f.kind, id: f.id };
    if (f.kind === 'customer') {
      const c = cust.get(f.id);
      if (c) out.set(key, { ...base, href: `/customers/${c.id}`, name: c.name, context: tr.nProjects(c.projects.length) });
    } else if (f.kind === 'project') {
      const p = proj.get(f.id);
      if (p) out.set(key, { ...base, href: `/projects/${f.id}`, name: p.name, context: p.customer });
    } else if (f.kind === 'checklist') {
      const l = lists.get(f.id);
      if (l) out.set(key, { ...base, href: `/checklists/${f.id}`, name: l.title, context: l.project });
    } else {
      const x = tpl.get(f.id);
      if (x) out.set(key, { ...base, href: `/templates/${f.id}`, name: x.name, context: tr.nItems(x.item_count) });
    }
  }
  return out;
}
