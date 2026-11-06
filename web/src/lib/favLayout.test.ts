import { describe, expect, it } from 'vitest';
import { addFolder, deleteFolder, moveNode, parseLayout, renameFolder, resolveLayout, stepNode, type FavLayout } from './favLayout';

const I = (k: string) => ({ t: 'i' as const, k });
const F = (id: string, items: string[], name = id, open = true) => ({ t: 'f' as const, id, name, open, items });
const flat = (l: FavLayout) => l.map((n) => (n.t === 'i' ? n.k : `${n.id}[${n.items.join(',')}]`)).join(' ');

describe('favLayout', () => {
  it('parse: malformed parts are skipped', () => {
    expect(parseLayout(null)).toEqual([]);
    expect(parseLayout([I('project:1'), { t: 'x' }, { t: 'f', id: 'a', name: 'A', items: ['project:2', 5] }, 'junk'])).toEqual([I('project:1'), F('a', ['project:2'], 'A')]);
  });

  it('resolve: drops removed favorites and duplicates, appends new ones, keeps empty folders', () => {
    const l = [I('project:1'), F('a', ['customer:2', 'project:1']), I('gone:9'), F('b', [])];
    expect(flat(resolveLayout(l, ['project:1', 'customer:2', 'checklist:3']))).toBe('project:1 a[customer:2] b[] checklist:3');
  });

  it('move a favorite before/after another, into a folder and to the end', () => {
    const l = [I('a'), I('b'), I('c'), F('f', ['d'])];
    expect(flat(moveNode(l, I('c'), { pos: 'before', ref: I('a') }))).toBe('c a b f[d]');
    expect(flat(moveNode(l, I('a'), { pos: 'after', ref: I('c') }))).toBe('b c a f[d]');
    expect(flat(moveNode(l, I('a'), { pos: 'into', folder: 'f' }))).toBe('b c f[d,a]');
    expect(flat(moveNode(l, I('b'), { pos: 'before', ref: I('d') }))).toBe('a c f[b,d]');
    expect(flat(moveNode(l, I('d'), { pos: 'end' }))).toBe('a b c f[] d');
    expect(moveNode(l, I('a'), { pos: 'before', ref: I('a') })).toBe(l);
  });

  it('folders move on the root level only (a drop inside a folder means beside it)', () => {
    const l = [I('a'), F('f', ['b']), F('g', ['c'])];
    expect(flat(moveNode(l, { t: 'f', id: 'g' }, { pos: 'before', ref: I('a') }))).toBe('g[c] a f[b]');
    expect(flat(moveNode(l, { t: 'f', id: 'g' }, { pos: 'before', ref: I('b') }))).toBe('a g[c] f[b]');
    expect(flat(moveNode(l, { t: 'f', id: 'g' }, { pos: 'into', folder: 'f' }))).toBe('a f[b] g[c]');
    expect(flat(moveNode(l, { t: 'f', id: 'f' }, { pos: 'end' }))).toBe('a g[c] f[b]');
  });

  it('dropping into a closed folder opens it', () => {
    const l = [I('a'), F('f', [], 'f', false)];
    const r = moveNode(l, I('a'), { pos: 'into', folder: 'f' });
    expect(r[0]).toMatchObject({ t: 'f', open: true, items: ['a'] });
  });

  it('step up/down within the folder or the root level', () => {
    const l = [I('a'), F('f', ['b', 'c']), I('d')];
    expect(flat(stepNode(l, I('c'), -1))).toBe('a f[c,b] d');
    expect(flat(stepNode(l, I('b'), -1))).toBe('a f[b,c] d');
    expect(flat(stepNode(l, I('a'), 1))).toBe('f[b,c] a d');
    expect(flat(stepNode(l, I('d'), 1))).toBe('a f[b,c] d');
  });

  it('add, rename and delete folders (favorites stay at the folder position)', () => {
    let l: FavLayout = [I('a'), I('b')];
    l = addFolder(l, 'x', ' Kunden ');
    l = moveNode(l, I('a'), { pos: 'into', folder: 'x' });
    l = renameFolder(l, 'x', 'Wichtig');
    expect(l[1]).toMatchObject({ name: 'Wichtig', items: ['a'] });
    expect(renameFolder(l, 'x', '  ')).toBe(l);
    expect(flat(deleteFolder([I('c'), ...l], 'x'))).toBe('c b a');
  });
});
