import { describe, expect, it } from 'vitest';
import { applyNav, backList, clearHistory, crumbsFor, describe as describeEntry, type Hist, type NavTexts } from './navHistory';
import type { SidebarCustomer, TemplateSummary } from './types';

const nav = (type: string, path: string, delta?: number, search = '') => ({ type, delta, path, search });
const paths = (h: Hist) => h.entries.map((e) => e.path).join(' ') + ` @${h.index}`;

describe('applyNav', () => {
  it('links push, back/forward (popstate) move the position, a new link drops the forward part', () => {
    let h = applyNav(null, nav('enter', '/'));
    h = applyNav(h, nav('link', '/customers/1'));
    h = applyNav(h, nav('link', '/projects/2'));
    h = applyNav(h, nav('link', '/checklists/9'));
    expect(paths(h)).toBe('/ /customers/1 /projects/2 /checklists/9 @3');
    h = applyNav(h, nav('popstate', '/customers/1', -2));
    expect(paths(h)).toBe('/ /customers/1 /projects/2 /checklists/9 @1');
    h = applyNav(h, nav('popstate', '/projects/2', 1));
    expect(h.index).toBe(2);
    h = applyNav(h, nav('link', '/tasks'));
    expect(paths(h)).toBe('/ /customers/1 /projects/2 /tasks @3');
  });

  it('same page again is not a new entry; reload keeps the stored history', () => {
    let h = applyNav(null, nav('enter', '/'));
    h = applyNav(h, nav('link', '/tasks'));
    expect(applyNav(h, nav('goto', '/tasks'))).toBe(h);
    expect(paths(applyNav(h, nav('enter', '/tasks')))).toBe('/ /tasks @1');
    expect(paths(applyNav(h, nav('enter', '/other')))).toBe('/other @0');
  });

  it('back list: newest first with the step for history.go()', () => {
    let h = applyNav(null, nav('enter', '/'));
    for (const p of ['/a', '/b', '/c']) h = applyNav(h, nav('link', p));
    expect(backList(h).map((b) => `${b.entry.path}:${b.delta}`)).toEqual(['/b:-1', '/a:-2', '/:-3']);
    expect(backList(h, 2)).toHaveLength(2);
  });

  it('menu: all earlier pages, main pages and settings sub pages included', () => {
    let h = applyNav(null, nav('enter', '/'));
    for (const p of ['/projects/2', '/settings/appearance', '/tasks', '/customers']) h = applyNav(h, nav('link', p));
    expect(backList(h).map((b) => b.entry.path)).toEqual(['/tasks', '/settings/appearance', '/projects/2', '/']);
  });

  it('clear: only the current page remains, then the history grows again', () => {
    let h = applyNav(null, nav('enter', '/'));
    for (const p of ['/a', '/b']) h = applyNav(h, nav('link', p));
    h = applyNav(h, nav('popstate', '/a', -1)); // with something "forward" as well
    h = clearHistory(h);
    expect(paths(h)).toBe('/a @0');
    expect(backList(h)).toEqual([]);
    expect(paths(applyNav(h, nav('link', '/tasks')))).toBe('/a /tasks @1');
  });
});

describe('breadcrumb and history labels', () => {
  const customers = [
    { id: 1, name: 'Kantonsspital Zug', projects: [{ id: 2, name: 'New OBP SRV', checklists: [{ id: 9, title: 'Installation' }], tables: [{ id: 7, title: 'Kalkulation' }] }] }
  ] as unknown as SidebarCustomer[];
  const templates = [{ id: 4, name: 'Server', description: '', item_count: 3 }] as TemplateSummary[];
  const data = { customers, templates };
  const tx: NavTexts = {
    overview: 'Übersicht',
    view: (v) => ({ list: 'Liste', timeline: 'Zeitachse', cards: 'Karten' })[v] ?? null,
    myTasks: 'Meine Aufgaben',
    customers: 'Kunden',
    templates: 'Templates',
    reports: 'Reporting',
    settings: 'Einstellungen',
    settingsPage: (s) => ({ appearance: 'Darstellung' })[s] ?? null
  };

  const home = { label: 'Übersicht', icon: 'home', href: '/' };

  const customersList = { label: 'Kunden', icon: 'building2', href: '/customers' };

  it('checklist: overview / customers / customer / project / checklist with types and links', () => {
    expect(crumbsFor('/checklists/9', '', data, tx)).toEqual([
      home,
      customersList,
      { label: 'Kantonsspital Zug', type: 'customer', href: '/customers/1' },
      { label: 'New OBP SRV', type: 'project', href: '/projects/2' },
      { label: 'Installation', type: 'checklist' }
    ]);
  });

  it('table: like a checklist, below its project', () => {
    expect(crumbsFor('/tables/7', '', data, tx)).toEqual([
      home,
      customersList,
      { label: 'Kantonsspital Zug', type: 'customer', href: '/customers/1' },
      { label: 'New OBP SRV', type: 'project', href: '/projects/2' },
      { label: 'Kalkulation', type: 'table' }
    ]);
    expect(crumbsFor('/tables/99', '', data, tx)).toEqual([home, customersList, { label: '…', type: 'table' }]);
  });

  it('project and customer: below the customers list', () => {
    expect(crumbsFor('/projects/2', '', data, tx)).toEqual([
      home,
      customersList,
      { label: 'Kantonsspital Zug', type: 'customer', href: '/customers/1' },
      { label: 'New OBP SRV', type: 'project' }
    ]);
    expect(crumbsFor('/customers/1', '', data, tx)).toEqual([home, customersList, { label: 'Kantonsspital Zug', type: 'customer' }]);
  });

  it('list pages and templates', () => {
    expect(crumbsFor('/customers', '', data, tx)).toEqual([home, { label: 'Kunden', icon: 'building2' }]);
    expect(crumbsFor('/', '?view=list', data, tx)).toEqual([{ label: 'Übersicht', icon: 'list' }]); // no double "overview"
    expect(crumbsFor('/templates/4', '', data, tx)).toEqual([
      home,
      { label: 'Templates', icon: 'template', href: '/templates' },
      { label: 'Server', type: 'template' }
    ]);
  });

  it('history entries: object name with type, pages with view / sub page', () => {
    expect(describeEntry('/projects/2', '', data, tx)).toEqual({ label: 'New OBP SRV', type: 'project' });
    expect(describeEntry('/', '?view=list', data, tx)).toEqual({ label: 'Übersicht', icon: 'list', sub: 'Liste' });
    expect(describeEntry('/', '?view=timeline&q=x', data, tx)).toEqual({ label: 'Übersicht', icon: 'timeline', sub: 'Zeitachse' });
    expect(describeEntry('/', '', data, tx)).toEqual({ label: 'Übersicht', icon: 'grid', sub: 'Karten' }); // cards = default
    expect(describeEntry('/settings/appearance', '', data, tx)).toEqual({ label: 'Einstellungen · Darstellung', icon: 'settings' });
    expect(describeEntry('/tasks', '', data, tx)).toEqual({ label: 'Meine Aufgaben', icon: 'tasks' });
  });
});
