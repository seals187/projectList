<script lang="ts">
  // Reporting: dashboard, customers, workload and templates over the customers
  // the user may see (views, periods and defaults: Settings → Reporting).
  // Filters: period, customer, person, label (URL + per user);
  // "Export" = CSV of the current view, "PDF" = print view of a customer,
  // "Schedule e-mail" = report e-mails (admins and members).
  import { page } from '$app/state';
  import { api, errorText } from '$lib/api';
  import { canWrite } from '$lib/session.svelte';
  import { sidebar } from '$lib/workspace.svelte';
  import { toast } from '$lib/toasts.svelte';
  import { t, type Key } from '$lib/i18n.svelte';
  import { addPageView } from '$lib/overviewFilter.svelte';
  import { pageState } from '$lib/pageState.svelte';
  import { download, percent, pickPeriod, pickView, reportQuery, toCsv, type Report, type ReportConfig, type ReportConfigResp, type ReportFilter, type ReportView } from '$lib/reporting';
  import type { TemplateCategory } from '$lib/types';
  import PageHeader, { type HeaderFilter } from '$lib/components/PageHeader.svelte';
  import ZoomArea from '$lib/components/ZoomArea.svelte';
  import type { FilterId } from '$lib/components/FilterDropdown.svelte';
  import ReportDashboard from '$lib/components/ReportDashboard.svelte';
  import ReportCustomers from '$lib/components/ReportCustomers.svelte';
  import ReportWorkload from '$lib/components/ReportWorkload.svelte';
  import ReportTemplates from '$lib/components/ReportTemplates.svelte';
  import ScheduleDialog from '$lib/components/ScheduleDialog.svelte';

  // settings (Settings → Reporting): enabled views, periods and their defaults
  let cfg = $state<ReportConfig | null>(null);
  $effect(() => {
    api.get<ReportConfigResp>('/reports/config').then(
      (r) => (cfg = r.config),
      (e) => toast(errorText(e))
    );
  });
  const VIEW_ICON = { dashboard: 'chart', customers: 'users', workload: 'user', templates: 'fileText' } as const;

  const ps = pageState('reports', {
    view: { def: '' as string, url: 'str' }, // '' = default view of the settings
    period: { def: '' as string, url: 'str' }, // '' = default period of the settings
    cust: { def: null as FilterId[] | null, url: 'ids' },
    who: { def: null as FilterId[] | null, url: 'ids' },
    tag: { def: null as FilterId[] | null, url: 'ids' }
  });
  const view = $derived<ReportView>(cfg ? pickView(ps.view, cfg) : 'dashboard');
  const nums = (v: FilterId[] | null) => (v ?? []).map(Number).filter((n) => n > 0);
  const filter = $derived<ReportFilter>({
    period: cfg ? pickPeriod(ps.period, cfg) : 0,
    customers: nums(ps.cust),
    assignees: nums(ps.who),
    tags: nums(ps.tag)
  });

  let report = $state<Report | null>(null);
  let categories = $state<TemplateCategory[]>([]);
  let scheduling = $state(false);
  $effect(() => {
    if (!cfg) return; // wait for the settings (default period)
    const q = reportQuery(filter);
    api.get<Report>(`/reports?${q}`).then(
      (r) => (report = r),
      (e) => toast(errorText(e))
    );
  });
  $effect(() => {
    api.get<TemplateCategory[]>('/template-categories').then(
      (r) => (categories = r),
      () => {}
    );
  });

  const filters = $derived<HeaderFilter[]>([
    {
      key: 'period',
      label: t('rep.filter.period'),
      options: (cfg?.periods ?? []).map((p) => ({ value: String(p), label: t(`rep.period.${p}` as Key) })),
      value: String(filter.period),
      defaultValue: String(cfg?.default_period ?? ''),
      onchange: (v: string) => (ps.period = v)
    },
    {
      key: 'cust',
      label: t('filter.customer'),
      items: sidebar.customers.map((c) => ({ id: c.id, label: c.name, entity: 'customer' as const })),
      value: ps.cust,
      onchange: (v: FilterId[] | null) => (ps.cust = v)
    },
    {
      key: 'who',
      label: t('rep.filter.assignee'),
      items: (report?.assignees ?? []).map((u) => ({ id: u.id, label: u.name, avatar: true })),
      value: ps.who,
      onchange: (v: FilterId[] | null) => (ps.who = v)
    },
    {
      key: 'tag',
      label: t('rep.filter.label'),
      items: sidebar.tags.map((x) => ({ id: x.id, label: x.name, color: x.color })),
      value: ps.tag,
      onchange: (v: FilterId[] | null) => (ps.tag = v)
    }
  ]);

  // ---- CSV export of the current view
  function exportCsv() {
    const r = report;
    if (!r) return;
    let rows: (string | number | null)[][] = [];
    if (view === 'dashboard') {
      const d = r.dashboard;
      rows = [
        [t('rep.csv.metric'), t('rep.csv.value'), t('rep.csv.prev')],
        [t('rep.kpi.completed'), d.completed, d.completed_prev],
        [t('rep.kpi.open'), d.open, d.open_prev],
        [t('rep.kpi.overdue'), d.overdue, d.overdue_prev],
        [t('rep.kpi.projectsDone'), d.projects_done, null],
        [],
        [t('rep.csv.week'), t('rep.chart.completed'), t('rep.chart.created')],
        ...d.weeks.map((w) => [`${w.label} (${w.start})`, w.completed, w.created])
      ];
    } else if (view === 'customers') {
      rows = [
        [t('rep.col.customer'), t('rep.col.projects'), t('rep.col.progress'), t('rep.col.open'), t('rep.col.overdue'), t('rep.col.nextDue')],
        ...r.customers.map((c) => [c.name, c.projects.length, `${percent(c.done, c.total)} %`, c.open, c.overdue, c.next_due])
      ];
    } else if (view === 'workload') {
      rows = [
        [t('rep.col.assignee'), t('rep.col.open'), t('rep.legend.week'), t('rep.col.overdue'), t('rep.col.done', { n: r.days })],
        ...r.workload.people.map((p) => [p.name, p.open, p.due_week, p.overdue, p.done]),
        [t('rep.unassigned.title'), r.workload.unassigned.total, null, null, null]
      ];
    } else {
      rows = [
        [t('lib.col.template'), t('lib.col.used'), t('rep.col.avgDuration'), t('rep.col.planned'), t('rep.col.onTime')],
        ...r.templates.rows.map((x) => [x.name, x.used, x.avg_days, x.planned, x.on_time])
      ];
    }
    download(`projectlist-${view}-${r.to}.csv`, toCsv(rows));
  }
</script>

<svelte:head><title>{t('nav.reports')} · ProjectList</title></svelte:head>

<PageHeader
  zoomPage="reports"
  icon="chart"
  title={t('nav.reports')}
  subtitle={t('rep.subtitle')}
  views={(cfg?.views ?? []).map((v) => ({ key: v, label: t(`rep.view.${v}`), icon: VIEW_ICON[v] }))}
  activeView={view}
  onViewChange={(v) => (ps.view = v)}
  primaryAction={{ label: t('rep.export'), onClick: exportCsv, icon: 'download' }}
  extraActions={canWrite() ? [{ label: t('rep.schedule'), icon: 'mail', onClick: () => (scheduling = true), testid: 'rep-schedule' }] : []}
  searchPlaceholder=""
  searchValue=""
  onSearch={() => {}}
  noSearch
  {filters}
  onSaveView={(name) => addPageView(name, 'reports', page.url.search.slice(1))} />

<ZoomArea page="reports">
{#if !report}
  <p class="text-sm text-zinc-400">{t('common.loading')}</p>
{:else if view === 'dashboard'}
  <ReportDashboard {report} today={report.to} />
{:else if view === 'customers'}
  <ReportCustomers {report} today={report.to} />
{:else if view === 'workload'}
  <ReportWorkload {report} today={report.to} />
{:else}
  <ReportTemplates {report} {categories} />
{/if}
</ZoomArea>

{#if scheduling}
  <ScheduleDialog {filter} onclose={() => (scheduling = false)} />
{/if}
