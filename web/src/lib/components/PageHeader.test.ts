import { beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen } from '@testing-library/svelte';
import { i18n } from '$lib/i18n.svelte';
import PageHeader, { headerDate, type HeaderProps } from './PageHeader.svelte';

beforeEach(() => {
  i18n.lang = 'de';
});

const base = (p: Partial<HeaderProps>): HeaderProps => ({
  icon: 'home',
  title: 'Titel',
  searchPlaceholder: 'Suchen …',
  searchValue: '',
  onSearch: () => {},
  collapsed: false,
  onToggleCollapse: () => {},
  ...p
});

// the four page configurations (as in the routes)
const pages: Record<string, HeaderProps> = {
  overview: base({
    icon: 'home',
    title: 'Hallo Admin',
    subtitle: 'Freitag, 06.11.2026 · 2 Kunden · 3 Projekte',
    views: ['Karten', 'Board', 'Liste', 'Kalender', 'Zeitachse'].map((l, i) => ({ key: String(i), label: l, icon: 'grid' })),
    activeView: '0',
    primaryAction: { label: 'Projekt', onClick: () => {} },
    searchPlaceholder: 'Projekte suchen …',
    filters: ['Kunde', 'Label', 'Verantwortlich'].map((l) => ({ key: l, label: l, items: [{ id: 1, label: 'A' }], value: null, onchange: () => {} })),
    onSaveView: () => {},
    sortOptions: [{ value: 'activity', label: 'Letzte Aktivität' }],
    activeSort: 'activity'
  }),
  tasks: base({
    icon: 'listChecks',
    title: 'Meine Aufgaben',
    views: [
      { key: 'customers', label: 'Kunden', icon: 'users', count: 1 },
      { key: 'personal', label: 'Persönlich', icon: 'user', count: 0 }
    ],
    activeView: 'customers',
    searchPlaceholder: 'Aufgaben suchen …',
    filters: ['Kunde', 'Fälligkeit'].map((l) => ({ key: l, label: l, items: [], value: null, onchange: () => {} })),
    quickFilters: [
      { key: 'open', label: 'Offen', active: true },
      { key: 'today', label: 'Heute', active: false },
      { key: 'done', label: 'Erledigt', active: false }
    ],
    sortOptions: [{ value: 'due', label: 'Fälligkeit' }],
    activeSort: 'due'
  }),
  customers: base({
    icon: 'users',
    title: 'Kunden',
    views: [
      { key: 'cards', label: 'Karten', icon: 'grid' },
      { key: 'list', label: 'Liste', icon: 'list' }
    ],
    activeView: 'list',
    primaryAction: { label: 'Kunde', onClick: () => {} },
    searchPlaceholder: 'Kunden suchen …',
    filters: [{ key: 'status', label: 'Status', items: [], value: null, onchange: () => {} }],
    quickFilters: [{ key: 'all', label: 'Alle', count: 2, active: true }],
    sortOptions: [{ value: 'activity', label: 'Letzte Aktivität' }],
    activeSort: 'activity',
    collapsed: true
  }),
  templates: base({
    icon: 'fileText',
    title: 'Templates',
    views: [{ key: 'library', label: 'Bibliothek', icon: 'fileText', count: 6 }],
    activeView: 'library',
    primaryAction: { label: 'Template', onClick: () => {} },
    searchPlaceholder: 'Templates suchen …',
    sortOptions: [{ value: 'name', label: 'Name' }],
    activeSort: 'name'
  })
};

describe('PageHeader', () => {
  it.each(Object.entries(pages))('renders the %s configuration with the same structure', (_, props) => {
    render(PageHeader, { props });
    expect(screen.getByRole('heading', { level: 1 }).textContent).toBe(props.title);
    for (const v of props.views ?? []) expect(screen.getByRole('button', { name: new RegExp(v.label) })).toBeTruthy();
    expect(screen.getByPlaceholderText(props.searchPlaceholder)).toBeTruthy();
    if (props.primaryAction) expect(screen.getByRole('button', { name: props.primaryAction.label })).toBeTruthy();
    for (const f of props.filters ?? []) expect(screen.getByRole('button', { name: new RegExp(`^${f.label}:`) })).toBeTruthy();
    for (const c of props.quickFilters ?? []) expect(screen.getByRole('button', { name: new RegExp(`^${c.label}\\s*\\d*$`) }).getAttribute('aria-pressed')).toBe(String(c.active));
    // sorting, the toggle and "Display" are always at the right end of row 2
    const toggle = screen.getByRole('button', { name: props.collapsed ? 'Alle aufklappen' : 'Alle einklappen' });
    expect(toggle.nextElementSibling?.textContent?.trim()).toBe('Anzeige');
    expect(toggle.parentElement?.lastElementChild).toBe(toggle.nextElementSibling);
  });

  it('shows no primary action when none is given (My tasks)', () => {
    render(PageHeader, { props: pages.tasks });
    expect(screen.queryByRole('button', { name: /Checkliste/ })).toBeNull();
  });

  it('toggle: "Alle einklappen" when open or mixed, "Alle aufklappen" when collapsed', async () => {
    const onToggleCollapse = vi.fn();
    const { rerender } = render(PageHeader, { props: { ...pages.overview, collapsed: false, onToggleCollapse } });
    await fireEvent.click(screen.getByRole('button', { name: 'Alle einklappen' }));
    expect(onToggleCollapse).toHaveBeenCalledOnce();
    await rerender({ ...pages.overview, collapsed: true, onToggleCollapse });
    expect(screen.getByRole('button', { name: 'Alle aufklappen' })).toBeTruthy();
  });

  it('can disable the toggle (e.g. calendar)', () => {
    render(PageHeader, { props: { ...pages.overview, collapseDisabled: true } });
    expect((screen.getByRole('button', { name: 'Alle einklappen' }) as HTMLButtonElement).disabled).toBe(true);
  });

  it('reports view changes, chips and search', async () => {
    const onViewChange = vi.fn();
    const onQuickFilter = vi.fn();
    const onSearch = vi.fn();
    render(PageHeader, { props: { ...pages.tasks, onViewChange, onQuickFilter, onSearch } });
    await fireEvent.click(screen.getByRole('button', { name: /Persönlich/ }));
    expect(onViewChange).toHaveBeenCalledWith('personal');
    await fireEvent.click(screen.getByRole('button', { name: /^Heute/ }));
    expect(onQuickFilter).toHaveBeenCalledWith('today');
    await fireEvent.input(screen.getByPlaceholderText('Aufgaben suchen …'), { target: { value: 'abc' } });
    expect(onSearch).toHaveBeenCalledWith('abc');
  });

  it('"Anzeige": popover with switches, a click reports the new value', async () => {
    const onDisplayChange = vi.fn();
    const displayOptions = [
      { key: 'steps', label: 'Schritte anzeigen (6/20)', value: true },
      { key: 'allTags', label: 'Alle Tags anzeigen', value: false }
    ];
    render(PageHeader, { props: { ...pages.overview, displayOptions, onDisplayChange } });
    await fireEvent.click(screen.getByRole('button', { name: 'Anzeige' }));
    const steps = screen.getByRole('switch', { name: 'Schritte anzeigen (6/20)' });
    expect(steps.getAttribute('aria-checked')).toBe('true');
    expect(screen.getByText('Wird pro Benutzer gespeichert')).toBeTruthy();
    await fireEvent.click(steps);
    expect(onDisplayChange).toHaveBeenCalledWith('steps', false);
    await fireEvent.click(screen.getByRole('switch', { name: 'Alle Tags anzeigen' }));
    expect(onDisplayChange).toHaveBeenCalledWith('allTags', true);
  });

  it('"Anzeige" is disabled on pages without options', () => {
    render(PageHeader, { props: pages.overview });
    expect((screen.getByRole('button', { name: 'Anzeige' }) as HTMLButtonElement).disabled).toBe(true);
  });

  it('formats the date as dd.mm.yyyy with the weekday', () => {
    expect(headerDate(new Date(2026, 10, 6), 'de-DE')).toBe('Freitag, 06.11.2026');
    expect(headerDate(new Date(2026, 0, 2), 'en-GB')).toBe('Friday, 02.01.2026');
  });
});
