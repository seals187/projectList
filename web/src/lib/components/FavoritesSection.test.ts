import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createRawSnippet } from 'svelte';
import { fireEvent, render, screen } from '@testing-library/svelte';

vi.mock('$app/navigation', () => ({ goto: vi.fn() }));
vi.mock('$lib/api', () => ({
  api: { get: vi.fn(() => Promise.resolve([])), put: vi.fn(() => Promise.resolve(null)), del: vi.fn(() => Promise.resolve(null)), post: vi.fn(), patch: vi.fn(() => Promise.resolve(null)) },
  errorText: () => 'error'
}));

import { api } from '$lib/api';
import { goto } from '$app/navigation';
import { i18n } from '$lib/i18n.svelte';
import { sidebar } from '$lib/state.svelte';
import type { SidebarCustomer } from '$lib/types';
import FavoritesSection from './FavoritesSection.svelte';

const head = createRawSnippet(() => ({ render: () => '<p>Favoriten</p>' }));
const customers = [
  {
    id: 1,
    name: '(ZU) Kantonsspital Zug',
    projects: [
      { id: 1, name: '(ZU) New THF PROD SRV', checklists: [{ id: 9, title: 'ZU-CL-New_OBP_SRV' }] },
      { id: 2, name: '(ZU) New Opale PROD SRV', checklists: [] }
    ]
  }
] as unknown as SidebarCustomer[];

describe('FavoritesSection', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    i18n.lang = 'de';
    sidebar.customers = customers;
    sidebar.favoriteGroups = [{ id: 3, name: 'NEW_SRV_Installations', position: 0, collapsed: false }];
    sidebar.favorites = [
      { kind: 'project', id: 1, group_id: 3, position: 0 },
      { kind: 'checklist', id: 9, group_id: 3, position: 1 },
      { kind: 'customer', id: 1, group_id: null, position: 0 }
    ];
  });

  const renderIt = () => render(FavoritesSection, { props: { closed: false, head, activePath: '/checklists/9' } });

  it('two-line entries with type and context, no star, groups with count', () => {
    renderIt();
    const subs = screen.getAllByTestId('fav-subtitle').map((e) => e.textContent);
    expect(subs).toEqual(['Kunde · 2 Projekte', 'Projekt · (ZU) Kantonsspital Zug', 'Checkliste · (ZU) New THF PROD SRV']);
    expect(document.querySelector('[data-entity="favGroup"]')).toBeTruthy();
    expect(screen.getByTestId('fav-group').textContent).toContain('NEW_SRV_Installations');
    expect(document.querySelectorAll('[data-testid="fav-item"] svg path[d^="M12 2l3"]').length).toBe(0); // no star
    // the open page is highlighted
    expect(screen.getByRole('link', { current: 'page' }).textContent).toContain('ZU-CL-New_OBP_SRV');
  });

  it('context menu: open, open in new tab, move to group, remove', async () => {
    const open = vi.spyOn(window, 'open').mockImplementation(() => null);
    renderIt();
    const entry = () => screen.getByRole('link', { name: (name) => name.startsWith('(ZU) Kantonsspital') });

    await fireEvent.contextMenu(entry());
    await fireEvent.click(screen.getByRole('menuitem', { name: 'Öffnen' }));
    expect(goto).toHaveBeenCalledWith('/customers/1');

    await fireEvent.contextMenu(entry());
    await fireEvent.click(screen.getByRole('menuitem', { name: 'In neuem Tab öffnen' }));
    expect(open).toHaveBeenCalledWith('/customers/1', '_blank', 'noopener');

    await fireEvent.contextMenu(entry());
    await fireEvent.click(screen.getByRole('menuitem', { name: /In Gruppe verschieben/ }));
    await fireEvent.click(screen.getByRole('menuitem', { name: 'NEW_SRV_Installations' }));
    expect(api.put).toHaveBeenCalledWith('/favorites/order', {
      groups: [3],
      items: [
        { kind: 'project', id: 1, group_id: 3 },
        { kind: 'checklist', id: 9, group_id: 3 },
        { kind: 'customer', id: 1, group_id: 3 }
      ]
    });
    expect(sidebar.favorites.find((f) => f.kind === 'customer')).toMatchObject({ group_id: 3, position: 2 });

    await fireEvent.contextMenu(screen.getByRole('link', { name: (name) => name.startsWith('(ZU) New THF') }));
    await fireEvent.click(screen.getByRole('menuitem', { name: 'Aus Favoriten entfernen' }));
    expect(api.del).toHaveBeenCalledWith('/favorites/project/1');
    expect(sidebar.favorites.some((f) => f.kind === 'project' && f.id === 1)).toBe(false);
  });

  it('group menu: rename and delete (entries move to "without group")', async () => {
    renderIt();
    await fireEvent.contextMenu(screen.getByRole('button', { name: /NEW_SRV_Installations/ }));
    await fireEvent.click(screen.getByRole('menuitem', { name: 'Gruppe löschen' }));
    expect(api.del).toHaveBeenCalledWith('/favorite-groups/3');
    expect(sidebar.favoriteGroups).toEqual([]);
    expect(sidebar.favorites.every((f) => f.group_id === null)).toBe(true);
  });
});
