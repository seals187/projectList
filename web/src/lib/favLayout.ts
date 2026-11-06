// Order and folders of the sidebar favorites (pure functions, testable).
// The favorites themselves live in the DB (customer/project/checklist); the
// layout is a per-user pref "fav_layout": a list of root nodes, each either a
// favorite (key "kind:id") or a folder with its favorites (one level only).
// Favorites missing from the layout are appended at the end, keys without a
// favorite are dropped, empty folders stay.

export type FavNode = { t: 'i'; k: string } | { t: 'f'; id: string; name: string; open: boolean; items: string[] };
export type FavLayout = FavNode[];
export type FavRef = { t: 'i'; k: string } | { t: 'f'; id: string };
export type DropTarget = { pos: 'before' | 'after'; ref: FavRef } | { pos: 'into'; folder: string } | { pos: 'end' };

const isStr = (v: unknown): v is string => typeof v === 'string' && v.length > 0 && v.length <= 80;

/** Validates a stored layout (malformed parts are skipped). */
export function parseLayout(v: unknown): FavLayout {
  if (!Array.isArray(v)) return [];
  const out: FavLayout = [];
  for (const n of v) {
    if (!n || typeof n !== 'object') continue;
    const o = n as Record<string, unknown>;
    if (o.t === 'i' && isStr(o.k)) out.push({ t: 'i', k: o.k });
    else if (o.t === 'f' && isStr(o.id) && typeof o.name === 'string')
      out.push({ t: 'f', id: o.id, name: o.name.slice(0, 60), open: o.open !== false, items: Array.isArray(o.items) ? o.items.filter(isStr) : [] });
  }
  return out;
}

/** Layout for the current favorites: unknown keys removed (and duplicates), new favorites appended. */
export function resolveLayout(layout: FavLayout, keys: readonly string[]): FavLayout {
  const exist = new Set(keys);
  const seen = new Set<string>();
  const keep = (k: string) => exist.has(k) && !seen.has(k) && (seen.add(k), true);
  const out: FavLayout = [];
  for (const n of layout) {
    if (n.t === 'i') {
      if (keep(n.k)) out.push(n);
    } else out.push({ ...n, items: n.items.filter(keep) });
  }
  for (const k of keys) if (keep(k)) out.push({ t: 'i', k });
  return out;
}

const sameRef = (a: FavRef, n: FavNode) => (a.t === 'i' ? n.t === 'i' && n.k === a.k : n.t === 'f' && n.id === a.id);
const refEq = (a: FavRef, b: FavRef) => (a.t === 'i' ? b.t === 'i' && a.k === b.k : b.t === 'f' && a.id === b.id);

/** Folder that contains the favorite (null = root). */
export function folderOf(layout: FavLayout, key: string): string | null {
  for (const n of layout) if (n.t === 'f' && n.items.includes(key)) return n.id;
  return null;
}

/** Takes the node out of the layout (copy); returns the new layout and the removed node. */
function take(layout: FavLayout, ref: FavRef): [FavLayout, FavNode | null] {
  let found: FavNode | null = null;
  const out: FavLayout = [];
  for (const n of layout) {
    if (sameRef(ref, n)) {
      found = n;
      continue;
    }
    if (n.t === 'f' && ref.t === 'i' && n.items.includes(ref.k)) {
      found = { t: 'i', k: ref.k };
      out.push({ ...n, items: n.items.filter((k) => k !== ref.k) });
    } else out.push(n.t === 'f' ? { ...n, items: [...n.items] } : n);
  }
  return [out, found];
}

/** Moves a favorite or folder to the drop target. Folders only move on the root level. */
export function moveNode(layout: FavLayout, src: FavRef, target: DropTarget): FavLayout {
  if ((target.pos === 'before' || target.pos === 'after') && refEq(src, target.ref)) return layout;
  if (target.pos === 'into' && src.t === 'f' && src.id === target.folder) return layout;
  const [rest, node] = take(layout, src);
  if (!node) return layout;

  if (node.t === 'f') {
    // a folder dropped on/into something inside a folder → before/after that folder
    let t = target;
    if (t.pos === 'into') t = { pos: 'after', ref: { t: 'f', id: t.folder } };
    if (t.pos !== 'end' && t.ref.t === 'i') {
      const f = folderOf(rest, t.ref.k);
      if (f) t = { pos: t.pos, ref: { t: 'f', id: f } };
    }
    if (t.pos === 'end') return [...rest, node];
    const ref = t.ref;
    const i = rest.findIndex((n) => sameRef(ref, n));
    if (i < 0) return [...rest, node];
    rest.splice(t.pos === 'before' ? i : i + 1, 0, node);
    return rest;
  }

  const key = node.k;
  if (target.pos === 'end') return [...rest, node];
  if (target.pos === 'into') {
    const f = rest.find((n) => n.t === 'f' && n.id === target.folder);
    if (!f || f.t !== 'f') return [...rest, node];
    f.items.push(key);
    f.open = true;
    return rest;
  }
  const ref = target.ref;
  if (ref.t === 'i') {
    const fid = folderOf(rest, ref.k);
    if (fid) {
      const f = rest.find((n) => n.t === 'f' && n.id === fid) as Extract<FavNode, { t: 'f' }>;
      const i = f.items.indexOf(ref.k);
      f.items.splice(target.pos === 'before' ? i : i + 1, 0, key);
      return rest;
    }
  }
  const i = rest.findIndex((n) => sameRef(ref, n));
  if (i < 0) return [...rest, node];
  rest.splice(target.pos === 'before' ? i : i + 1, 0, node);
  return rest;
}

/** One step up (-1) or down (+1) within its folder or the root level. */
export function stepNode(layout: FavLayout, src: FavRef, dir: -1 | 1): FavLayout {
  const fid = src.t === 'i' ? folderOf(layout, src.k) : null;
  if (fid && src.t === 'i') {
    const f = layout.find((n) => n.t === 'f' && n.id === fid) as Extract<FavNode, { t: 'f' }>;
    const i = f.items.indexOf(src.k);
    const j = i + dir;
    if (j < 0 || j >= f.items.length) return layout;
    const items = [...f.items];
    [items[i], items[j]] = [items[j], items[i]];
    return layout.map((n) => (n === f ? { ...f, items } : n));
  }
  const i = layout.findIndex((n) => sameRef(src, n));
  const j = i + dir;
  if (i < 0 || j < 0 || j >= layout.length) return layout;
  const out = [...layout];
  [out[i], out[j]] = [out[j], out[i]];
  return out;
}

export function addFolder(layout: FavLayout, id: string, name: string): FavLayout {
  return [...layout, { t: 'f', id, name: name.trim().slice(0, 60), open: true, items: [] }];
}

export function renameFolder(layout: FavLayout, id: string, name: string): FavLayout {
  const v = name.trim().slice(0, 60);
  return v ? layout.map((n) => (n.t === 'f' && n.id === id ? { ...n, name: v } : n)) : layout;
}

export function setFolderOpen(layout: FavLayout, id: string, open: boolean): FavLayout {
  return layout.map((n) => (n.t === 'f' && n.id === id ? { ...n, open } : n));
}

/** Deletes the folder; its favorites stay, at the folder's position. */
export function deleteFolder(layout: FavLayout, id: string): FavLayout {
  return layout.flatMap((n): FavNode[] => (n.t === 'f' && n.id === id ? n.items.map((k) => ({ t: 'i', k })) : [n]));
}

/** A favorite resolved for display (Sidebar). */
export interface FavEntry {
  key: string; // "kind:id"
  kind: 'customer' | 'project' | 'checklist';
  id: number;
  href: string;
  label: string;
  hint: string; // tooltip addition (project of a checklist)
  on: boolean; // current page
}
