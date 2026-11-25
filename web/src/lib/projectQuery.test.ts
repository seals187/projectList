import { describe, expect, it } from 'vitest';
import {
  boardColumns,
  boardDropOrder,
  buildOverview,
  calendarDays,
  checklistOpen,
  customerCards,
  customerProgress,
  dueBucket,
  dueUrgent,
  effectiveHidden,
  isMine,
  matchesChip,
  myCustomers,
  nextDue,
  nextHidden,
  NO_OWNER,
  openItems,
  pct,
  personalSource,
  personalTasks,
  progressRatio,
  projectComplete,
  projectDue,
  queryOverview,
  schedule,
  searchOverview,
  sortOverview,
  taskGroups,
  timelineRange,
  timelineRows,
  type OvCustomer,
  type TaskQuery
} from './projectQuery';
import { addMonths, dayOfMonth, daysUntil, fromDay, isoWeek, mondayOf, monthStart, toDay, weekday } from './dates';
import type { OverviewExtra, SidebarChecklist, SidebarCustomer, SidebarProject, Task } from './types';

const TODAY = '2026-11-25';

const cl = (id: number, over: Partial<SidebarChecklist> = {}): SidebarChecklist => ({ id, title: `CL ${id}`, total: 2, done: 1, next_due: '', ...over });

function proj(id: number, over: Partial<SidebarProject> = {}): SidebarProject {
  return {
    id,
    name: `P${id}`,
    status: 'active',
    start_date: null,
    due_date: null,
    created_at: '2026-11-01',
    include_in_customer_progress: true,
    tag_ids: [],
    owner_id: null,
    owner_name: '',
    board_position: 0,
    background: '',
    total: 4,
    done: 1,
    checklists: [],
    tables: [],
    ...over
  };
}

const cust = (id: number, name: string, projects: SidebarProject[]): SidebarCustomer => ({ id, name, location: '', background: '', color: '', color_everywhere: false, projects });

const ov = (customers: SidebarCustomer[], extra: Record<string, OverviewExtra> = {}) => buildOverview(customers, extra);
const names = (g: OvCustomer[]) => g.map((x) => `${x.customer.name}:${x.projects.map((p) => p.name).join(',')}`);

// ---- dates.ts ---------------------------------------------------------------------

describe('dates', () => {
  it('day numbers round-trip and are time zone free', () => {
    expect(toDay('1970-01-01')).toBe(0);
    expect(fromDay(toDay('2026-03-29'))).toBe('2026-03-29'); // DST switch in Europe
    expect(toDay('2026-03-30') - toDay('2026-03-29')).toBe(1);
  });
  it('weekday, Monday and month helpers', () => {
    const d = toDay('2026-11-25'); // Wednesday
    expect(weekday(d)).toBe(2);
    expect(fromDay(mondayOf(d))).toBe('2026-11-23');
    expect(fromDay(monthStart(d))).toBe('2026-11-01');
    expect(fromDay(addMonths(d, 2))).toBe('2027-01-01');
    expect(fromDay(addMonths(d, -11))).toBe('2025-12-01');
    expect(dayOfMonth(d)).toBe(25);
  });
  it('ISO week numbers around the year boundary', () => {
    expect(isoWeek(toDay('2026-01-01'))).toBe(1);
    expect(isoWeek(toDay('2027-01-01'))).toBe(53); // 2026 has 53 ISO weeks
    expect(isoWeek(toDay('2026-11-25'))).toBe(48);
  });
  it('daysUntil: positive in the future, negative when overdue', () => {
    expect(daysUntil('2026-11-28', TODAY)).toBe(3);
    expect(daysUntil(TODAY, TODAY)).toBe(0);
    expect(daysUntil('2026-11-20', TODAY)).toBe(-5);
    expect(daysUntil('2026-03-30', '2026-03-28')).toBe(2);
  });
});

// ---- concepts -------------------------------------------------------------------------

describe('progress, open, due, overdue', () => {
  it('progress in percent and as a sort ratio', () => {
    expect(pct(1, 3)).toBe(33);
    expect(pct(0, 0)).toBe(0);
    expect(progressRatio({ done: 0, total: 0 })).toBe(-1);
    expect(progressRatio({ done: 1, total: 4 })).toBe(0.25);
  });
  it('open items and open checklists', () => {
    expect(openItems({ done: 1, total: 4 })).toBe(3);
    expect(openItems({ done: 5, total: 4 })).toBe(0);
    expect(checklistOpen({ done: 0, total: 0 })).toBe(true);
    expect(checklistOpen({ done: 2, total: 2 })).toBe(false);
  });
  it('complete = status done or all items checked', () => {
    expect(projectComplete(proj(1, { status: 'done' }))).toBe(true);
    expect(projectComplete(proj(1, { done: 4, total: 4 }))).toBe(true);
    expect(projectComplete(proj(1, { done: 0, total: 0 }))).toBe(false);
  });
  it('project due (overview) vs. next due (My tasks)', () => {
    const p = proj(1, { due_date: '2026-12-10', checklists: [cl(1, { next_due: '2026-12-01' }), cl(2, { done: 2, total: 2, next_due: '2026-11-26' })] });
    expect(projectDue(p)).toBe('2026-12-10');
    expect(nextDue(p)).toBe('2026-12-01'); // closed checklist does not count
    expect(projectDue({ ...p, status: 'done' })).toBe('');
    expect(nextDue({ ...p, status: 'done' })).toBe('2026-12-01'); // open checklist of a done project still counts
    expect(nextDue(proj(2))).toBe('');
  });
  it('urgent: not done and due within 3 days or overdue', () => {
    expect(dueUrgent(proj(1, { due_date: '2026-11-28' }), TODAY)).toBe(true);
    expect(dueUrgent(proj(1, { due_date: '2026-11-29' }), TODAY)).toBe(false);
    expect(dueUrgent(proj(1, { due_date: '2026-11-01' }), TODAY)).toBe(true);
    expect(dueUrgent(proj(1, { due_date: '2026-11-01', status: 'done' }), TODAY)).toBe(false);
    expect(dueUrgent(proj(1), TODAY)).toBe(false);
  });
  it('due buckets', () => {
    expect(dueBucket(null, TODAY)).toBe('none');
    expect(dueBucket('2026-11-24', TODAY)).toBe('overdue');
    expect(dueBucket(TODAY, TODAY)).toBe('today');
    expect(dueBucket('2026-12-01', TODAY)).toBe('week');
    expect(dueBucket('2026-12-02', TODAY)).toBe('later');
  });
  it('customer progress: included projects only, archived never', () => {
    const r = customerProgress([proj(1, { done: 1, total: 2 }), proj(2, { done: 3, total: 3, include_in_customer_progress: false }), proj(3, { status: 'archived', done: 5, total: 9 })]);
    expect(r).toEqual({ done: 1, total: 2, excluded: 1 });
  });
  it('schedule: time vs. progress', () => {
    const p = proj(1, { start_date: '2026-11-01', due_date: '2026-12-01', done: 0, total: 10 });
    expect(schedule(p, '2026-11-16')).toMatchObject({ elapsed: 50, progress: 0, state: 'behind', behind: 50 });
    expect(schedule({ ...p, done: 5 }, '2026-11-16')?.state).toBe('onTrack');
    expect(schedule(p, '2026-12-05')?.state).toBe('overdue');
    expect(schedule({ ...p, status: 'done' }, '2026-12-05')?.state).toBe('done');
    expect(schedule(proj(2), TODAY)).toBeNull();
  });
});

// ---- overview ---------------------------------------------------------------------------

describe('overview query', () => {
  const a = cust(1, 'Alpha', [
    proj(1, { name: 'Zeta', tag_ids: [7], owner_id: 3, owner_name: 'Ann', due_date: '2026-12-20' }),
    proj(2, { name: 'beta', done: 4, total: 4, due_date: '2026-12-01' }),
    proj(3, { name: 'Old', status: 'archived' })
  ]);
  const b = cust(2, 'Bravo', [proj(4, { name: 'Gamma', owner_id: 3, owner_name: 'Ann', done: 0, total: 0 })]);
  const c = cust(3, 'Charlie', []);
  const extra = { 1: { team: [{ id: 3, name: 'Ann' }], next_step: '<p>Call <b>them</b></p>', updated_at: '2026-11-20', checklists: { '11': { assignee_id: null, assignee_name: '', overdue: 2 } } } };
  const groups = ov([a, b, c], extra);

  it('builds the model: archived left out, extras merged, customer progress', () => {
    expect(groups[0].projects.map((p) => p.id)).toEqual([1, 2]);
    expect(groups[0].projects[0]).toMatchObject({ next: 'Call them', updated: '2026-11-20', team: [{ id: 3, name: 'Ann' }] });
    expect(groups[0].projects[0].clExtra[11].overdue).toBe(2);
    expect(groups[0].projects[1].updated).toBe('2026-11-01'); // no activity: creation date
    expect(groups[0]).toMatchObject({ done: 5, total: 8, excluded: 0 });
  });

  it('search: customer name keeps all its projects, otherwise project names', () => {
    expect(names(searchOverview(groups, 'alp'))).toEqual(['Alpha:Zeta,beta']);
    expect(names(searchOverview(groups, ' GAM '))).toEqual(['Bravo:Gamma']);
    expect(searchOverview(groups, '')).toBe(groups);
  });

  it('sort: name, progress, due (project due date), activity', () => {
    expect(names(sortOverview(groups, 'name'))).toEqual(['Alpha:beta,Zeta', 'Bravo:Gamma', 'Charlie:']);
    expect(names(sortOverview(groups, 'progress'))).toEqual(['Alpha:beta,Zeta', 'Bravo:Gamma', 'Charlie:']);
    // beta is complete but not "done" → its due date still counts
    expect(names(sortOverview(groups, 'due'))).toEqual(['Alpha:beta,Zeta', 'Bravo:Gamma', 'Charlie:']);
    expect(names(sortOverview(groups, 'activity'))).toEqual(['Alpha:Zeta,beta', 'Bravo:Gamma', 'Charlie:']);
  });

  it('filters: hidden customers, labels and owners (customers without match left out)', () => {
    const base = { hidden: [], tags: [], owners: [], q: '', sort: 'name' as const };
    expect(names(queryOverview(groups, base))).toEqual(['Alpha:beta,Zeta', 'Bravo:Gamma', 'Charlie:']);
    expect(names(queryOverview(groups, { ...base, hidden: [2, 99] }))).toEqual(['Alpha:beta,Zeta', 'Charlie:']);
    expect(names(queryOverview(groups, { ...base, tags: [7] }))).toEqual(['Alpha:Zeta']);
    expect(names(queryOverview(groups, { ...base, owners: [3] }))).toEqual(['Alpha:Zeta', 'Bravo:Gamma']);
    expect(names(queryOverview(groups, { ...base, owners: [NO_OWNER] }))).toEqual(['Alpha:beta']);
    expect(names(queryOverview(groups, { ...base, q: 'gamma' }))).toEqual(['Bravo:Gamma']);
  });

  it('effective hidden ignores deleted customers', () => {
    expect([...effectiveHidden(groups, [2, 99])]).toEqual([2]);
  });

  it('customer filter transitions: only, toggle, all', () => {
    const all = [1, 2, 3];
    expect(nextHidden([], all, 2, 'only')).toEqual([1, 3]);
    expect(nextHidden([1], all, 2, 'toggle')).toEqual([1, 2]);
    expect(nextHidden([1, 2], all, 2, 'toggle')).toEqual([1]);
    expect(nextHidden([1, 2], all, 2, 'all')).toEqual([]);
  });
});

describe('board', () => {
  const labels = { status: (s: string) => s.toUpperCase(), noOwner: 'Nobody' };
  const c1 = cust(1, 'Alpha', [
    proj(1, { status: 'active', board_position: 2, owner_id: 5, owner_name: 'Zoe' }),
    proj(2, { status: 'planning', board_position: 0 }),
    proj(3, { status: 'active', board_position: 0, owner_id: 4, owner_name: 'Al' })
  ]);
  const c2 = cust(2, 'Bravo', [proj(4, { status: 'active', board_position: 1 })]);
  const groups = ov([c1, c2]);

  it('status columns in board order and by position', () => {
    const cols = boardColumns(groups, 'status', labels);
    expect(cols.map((c) => c.key)).toEqual(['planning', 'active', 'review', 'blocked', 'done']);
    expect(cols[1].title).toBe('ACTIVE');
    expect(cols[1].items.map((p) => p.id)).toEqual([3, 4, 1]);
  });
  it('customer columns: status, then position', () => {
    const cols = boardColumns(groups, 'customer', labels);
    expect(cols.map((c) => c.items.map((p) => p.id))).toEqual([[2, 3, 1], [4]]);
  });
  it('owner columns: alphabetically, nobody last', () => {
    const cols = boardColumns(groups, 'owner', labels);
    expect(cols.map((c) => c.title)).toEqual(['Al', 'Zoe', 'Nobody']);
    expect(cols[2].items.map((p) => p.id)).toEqual([2, 4]);
  });

  it('drop order: insert before a card, keep hidden ones in place, null when unchanged', () => {
    const all = groups.flatMap((g) => g.projects); // active: 3 (0), 4 (1), 1 (2)
    const visible = new Set([3, 1, 2]); // 4 filtered out
    expect(boardDropOrder(all, visible, all.find((p) => p.id === 1)!, 'active', 3)).toEqual([1, 3, 4]);
    // at the end: after the last visible card
    expect(boardDropOrder(all, visible, all.find((p) => p.id === 2)!, 'active', null)).toEqual([3, 4, 1, 2]);
    expect(boardDropOrder(all, new Set([3, 2]), all.find((p) => p.id === 2)!, 'active', null)).toEqual([3, 2, 4, 1]);
    // into an empty column
    expect(boardDropOrder(all, visible, all.find((p) => p.id === 2)!, 'review', null)).toEqual([2]);
    // same place
    expect(boardDropOrder(all, new Set([1, 3, 4]), all.find((p) => p.id === 1)!, 'active', null)).toBeNull();
    // at the end of the visible cards = before the hidden one
    expect(boardDropOrder(all, visible, all.find((p) => p.id === 1)!, 'active', null)).toEqual([3, 1, 4]);
  });
});

describe('cards, timeline, calendar', () => {
  it('cards: overdue/open/next/team, overdue customers first, empty ones last', () => {
    const groups = ov(
      [
        cust(1, 'Empty', []),
        cust(2, 'Calm', [proj(1, { due_date: '2026-12-09', checklists: [cl(1, { next_due: '2026-11-20' }), cl(2, { next_due: '2026-12-03' })] }), proj(2, { status: 'done', due_date: '2026-11-26' })]),
        cust(3, 'Late', [proj(3)])
      ],
      { 3: { team: [{ id: 1, name: 'A' }], next_step: '', updated_at: null, checklists: { '9': { assignee_id: 1, assignee_name: 'A', overdue: 3 } } } }
    );
    const cards = customerCards(groups, TODAY);
    expect(cards.map((c) => c.customer.name)).toEqual(['Late', 'Calm', 'Empty']);
    expect(cards[0]).toMatchObject({ overdue: 3, open: 3, next: '', team: [{ id: 1, name: 'A' }] });
    // past checklist due date skipped; done project ignored
    expect(cards[1]).toMatchObject({ overdue: 0, open: 6, next: '2026-12-03' });
  });

  it('timeline: range from start (or creation) to due, rows by start then name', () => {
    expect(timelineRange(proj(1, { start_date: '2026-11-10', due_date: '2026-11-05' }))).toEqual([toDay('2026-11-05'), toDay('2026-11-10')]);
    expect(timelineRange(proj(1))).toEqual([toDay('2026-11-01'), toDay('2026-11-01')]);
    const rows = timelineRows(ov([cust(1, 'A', [proj(1, { name: 'b', start_date: '2026-11-03' }), proj(2, { name: 'a', start_date: '2026-11-03' }), proj(3, { name: 'c' })])]));
    expect(rows[0].projects.map((p) => p.name)).toEqual(['c', 'a', 'b']);
  });

  it('calendar: projects per due day and undated', () => {
    const ps = [proj(1, { due_date: TODAY }), proj(2), proj(3, { due_date: TODAY })];
    const { byDay, undated } = calendarDays(ps);
    expect(byDay.get(toDay(TODAY))?.map((p) => p.id)).toEqual([1, 3]);
    expect(undated.map((p) => p.id)).toEqual([2]);
  });
});

// ---- My tasks ---------------------------------------------------------------------------

describe('my tasks', () => {
  const me = 7;
  const extra: Record<string, OverviewExtra> = { 2: { team: [{ id: me, name: 'Me' }], next_step: '', updated_at: null, checklists: {} } };
  const customers = [
    cust(1, 'Alpha', [
      proj(1, { name: 'Owned', owner_id: me, due_date: '2026-12-20', checklists: [cl(1, { title: 'Setup', next_due: '2026-11-26' })] }),
      proj(2, { name: 'Assigned', due_date: '2026-11-30' }),
      proj(3, { name: 'Other', owner_id: 9 }),
      proj(4, { name: 'Archived', owner_id: me, status: 'archived' })
    ]),
    cust(2, 'Bravo', [proj(5, { name: 'Finished', owner_id: me, status: 'done', checklists: [cl(5, { next_due: TODAY })] })])
  ];
  const q = (over: Partial<TaskQuery> = {}): TaskQuery => ({ cust: null, due: null, q: '', sort: 'due', today: TODAY, ...over });

  it('mine: owner or someone assigned, never archived', () => {
    expect(isMine(customers[0].projects[0], me, extra)).toBe(true);
    expect(isMine(customers[0].projects[1], me, extra)).toBe(true);
    expect(isMine(customers[0].projects[2], me, extra)).toBe(false);
    expect(isMine(customers[0].projects[1], null, extra)).toBe(false);
    const mine = myCustomers(customers, { userId: me, extra, chip: 'open', today: TODAY });
    expect(mine.map((g) => g.projects.map((p) => p.name))).toEqual([['Owned', 'Assigned']]);
  });

  it('chips: open, today, done', () => {
    expect(matchesChip(customers[1].projects[0], 'done', TODAY)).toBe(true);
    expect(matchesChip(customers[1].projects[0], 'today', TODAY)).toBe(false); // done projects never in today/open
    expect(matchesChip(customers[0].projects[0], 'today', '2026-11-26')).toBe(true);
    expect(matchesChip(customers[0].projects[0], 'today', TODAY)).toBe(false);
    expect(matchesChip(proj(9, { done: 2, total: 2 }), 'open', TODAY)).toBe(false);
  });

  it('groups: sorted by next due date (checklists first), filters by due bucket of the next due date', () => {
    const mine = myCustomers(customers, { userId: me, extra, chip: 'open', today: TODAY });
    expect(taskGroups(mine, q())[0].projects.map((p) => p.name)).toEqual(['Owned', 'Assigned']); // 11-26 < 11-30
    expect(taskGroups(mine, q({ sort: 'name' }))[0].projects.map((p) => p.name)).toEqual(['Assigned', 'Owned']);
    expect(taskGroups(mine, q({ due: ['week'] }))[0].projects.map((p) => p.name)).toEqual(['Owned', 'Assigned']);
    expect(taskGroups(mine, q({ due: ['later'] }))).toEqual([]);
    expect(taskGroups(mine, q({ q: 'setup' }))[0].projects.map((p) => p.name)).toEqual(['Owned']); // checklist title
    expect(taskGroups(mine, q({ cust: [2] }))).toEqual([]);
    expect(taskGroups(mine, q())[0].checklists).toBe(1);
  });

  it('personal tasks: chip source, filters and sorting', () => {
    const task = (id: number, title: string, due: string | null, customer_id = 1): Task =>
      ({ id, title, description: '', note_color: '', due_date: due, done_at: null, checklist_id: 1, checklist_title: 'CL', project_id: 1, project_name: 'P', customer_id, customer_name: 'C' }) as Task;
    const src = [task(1, 'b', '2026-12-10'), task(2, 'a', null, 2), task(3, 'c', '2026-11-20')];
    expect(personalSource(src, 'today', TODAY).map((x) => x.id)).toEqual([3]);
    expect(personalSource(src, 'open', TODAY)).toBe(src);
    expect(personalTasks(src, q()).map((x) => x.id)).toEqual([3, 1, 2]);
    expect(personalTasks(src, q({ sort: 'name' })).map((x) => x.id)).toEqual([2, 1, 3]);
    expect(personalTasks(src, q({ sort: 'progress' })).map((x) => x.id)).toEqual([1, 2, 3]);
    expect(personalTasks(src, q({ cust: [2] })).map((x) => x.id)).toEqual([2]);
    expect(personalTasks(src, q({ due: ['overdue', 'none'] })).map((x) => x.id)).toEqual([3, 2]);
  });
});
