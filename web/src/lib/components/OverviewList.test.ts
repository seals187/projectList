import { beforeEach, describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/svelte';
import { i18n } from '$lib/i18n.svelte';
import { openCustomers } from '$lib/overviewOpen.svelte';
import type { OvCustomer } from '$lib/overview';
import OverviewList from './OverviewList.svelte';

const customer = { id: 1, name: '(AB) Appenzell Spital', projects: [] } as unknown as OvCustomer['customer'];
const project = (id: number, name: string, done: number, total: number, include = true) =>
  ({
    id,
    name,
    done,
    total,
    status: 'planning',
    due_date: '2026-11-16',
    include_in_customer_progress: include,
    tag_ids: [],
    checklists: [{ id: id * 10, title: 'Checkliste', total, done, next_due: '' }],
    owner_id: 1,
    owner_name: 'Admin',
    customer,
    team: [],
    next: '',
    updated: '2026-11-06',
    clExtra: {}
  }) as unknown as OvCustomer['projects'][number];

const groups: OvCustomer[] = [
  { customer, projects: [project(1, 'eOpale interne Installation', 2, 8), project(2, 'TEST 7.60 Migration', 3, 31), project(3, 'Extern', 1, 12, false)], done: 5, total: 39, excluded: 1 },
  { customer: { ...customer, id: 2, name: '(GH) Ganzer Hof' }, projects: [], done: 0, total: 0, excluded: 0 }
];
const all = { steps: true, percent: true, allTags: false, due: true, owner: true };
const props = (display = all) => ({ groups, today: '2026-11-06', canCreate: true, oncreate: () => {}, display });

describe('OverviewList', () => {
  beforeEach(() => {
    i18n.lang = 'de';
    openCustomers.add(1);
  });

  it('customer card: summary with steps over included projects, open steps, next due date', () => {
    render(OverviewList, { props: props() });
    expect(screen.getByTestId('customer-summary').textContent?.replace(/\s+/g, ' ').trim()).toBe('3 Projekte · 5/39 Schritte · 45 offen · nächste 16.11.');
    expect(screen.getAllByTestId('project-row')).toHaveLength(3);
  });

  it('project row: percent and steps next to it (narrow layout C1)', () => {
    render(OverviewList, { props: props() });
    const row = screen.getAllByTestId('project-row')[0];
    expect(row.textContent).toContain('25%');
    expect(row.querySelector('[data-testid="steps-inline"]')?.textContent).toBe('2/8');
    expect(row.getAttribute('href')).toBe('/projects/1');
  });

  it('"Schritte anzeigen" off: no steps in rows or summary; "Fortschritt in %" off: no percent', () => {
    render(OverviewList, { props: props({ ...all, steps: false, percent: false }) });
    expect(screen.queryAllByTestId('steps-inline')).toHaveLength(0);
    expect(screen.getByTestId('customer-summary').textContent).not.toContain('Schritte');
    expect(screen.getAllByTestId('project-row')[0].textContent).not.toContain('25%');
  });

  it('customer without projects: slim card with "+ Projekt aus Template"', () => {
    render(OverviewList, { props: props() });
    expect(screen.getByRole('button', { name: /Projekt aus Template/ })).toBeTruthy();
  });
});
