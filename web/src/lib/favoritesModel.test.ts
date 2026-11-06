import { describe, expect, it } from 'vitest';
import { moveNode } from './favLayout';
import { favoritesLayout, layoutToOrder, resolveFavorites } from './favoritesModel';
import type { Favorite, FavoriteGroup, SidebarCustomer, TemplateSummary } from './types';

const fav = (kind: Favorite['kind'], id: number, group_id: number | null, position: number): Favorite => ({ kind, id, group_id, position });
const groups: FavoriteGroup[] = [
  { id: 7, name: 'Clients', position: 1, collapsed: false },
  { id: 3, name: 'NEW_SRV_Installations', position: 0, collapsed: true }
];
const favorites = [fav('project', 2, 3, 1), fav('customer', 1, 7, 0), fav('checklist', 9, null, 0), fav('project', 1, 3, 0), fav('template', 4, null, 1)];

describe('favoritesModel', () => {
  it('layout: ungrouped first, then groups by position, items by position', () => {
    const l = favoritesLayout(favorites, groups);
    expect(l.map((n) => (n.t === 'i' ? n.k : `${n.name}[${n.items.join(',')}]${n.open ? '' : '-'}`))).toEqual([
      'checklist:9',
      'template:4',
      'NEW_SRV_Installations[project:1,project:2]-',
      'Clients[customer:1]'
    ]);
  });

  it('a favorite of an unknown group counts as ungrouped', () => {
    expect(favoritesLayout([fav('project', 5, 99, 0)], [])).toEqual([{ t: 'i', k: 'project:5' }]);
  });

  it('drag & drop → order for the API (moving between groups and ordering groups)', () => {
    let l = favoritesLayout(favorites, groups);
    l = moveNode(l, { t: 'i', k: 'project:2' }, { pos: 'into', folder: 'g7' }); // into "Clients"
    l = moveNode(l, { t: 'f', id: 'g7' }, { pos: 'before', ref: { t: 'f', id: 'g3' } }); // "Clients" first
    l = moveNode(l, { t: 'i', k: 'template:4' }, { pos: 'before', ref: { t: 'i', k: 'checklist:9' } });
    expect(layoutToOrder(l)).toEqual({
      groups: [7, 3],
      items: [
        { kind: 'template', id: 4, group_id: null },
        { kind: 'checklist', id: 9, group_id: null },
        { kind: 'customer', id: 1, group_id: 7 },
        { kind: 'project', id: 2, group_id: 7 },
        { kind: 'project', id: 1, group_id: 3 }
      ]
    });
  });

  it('context line per type: customer → projects, project → customer, checklist → project, template → items', () => {
    const customers = [
      {
        id: 1,
        name: '(ZU) Kantonsspital Zug',
        projects: [
          { id: 1, name: '(ZU) New THF PROD SRV', checklists: [{ id: 9, title: 'ZU-CL-New_OBP_SRV' }] },
          { id: 2, name: '(ZU) New Opale PROD SRV', checklists: [] },
          { id: 3, name: 'Drittes', checklists: [] }
        ]
      }
    ] as unknown as SidebarCustomer[];
    const templates = [{ id: 4, name: 'Server-Installation', description: '', item_count: 12 }] as TemplateSummary[];
    const tr = { nProjects: (n: number) => `${n} Projekte`, nItems: (n: number) => `${n} Einträge` };
    const r = resolveFavorites(favorites, customers, templates, tr);
    expect(r.get('customer:1')).toMatchObject({ name: '(ZU) Kantonsspital Zug', context: '3 Projekte', href: '/customers/1' });
    expect(r.get('project:2')).toMatchObject({ name: '(ZU) New Opale PROD SRV', context: '(ZU) Kantonsspital Zug', href: '/projects/2' });
    expect(r.get('checklist:9')).toMatchObject({ name: 'ZU-CL-New_OBP_SRV', context: '(ZU) New THF PROD SRV', href: '/checklists/9' });
    expect(r.get('template:4')).toMatchObject({ name: 'Server-Installation', context: '12 Einträge', href: '/templates/4' });
    // not visible (deleted / no access / templates not loaded) → skipped
    expect(resolveFavorites([fav('project', 77, null, 0), fav('template', 4, null, 0)], customers, null, tr).size).toBe(0);
  });
});
