<script lang="ts">
  // Archive: all archived projects of all visible customers in one table
  // (customer, date and person of archiving, checklist progress), with search,
  // filters customer / period, chip "Only incomplete" and sorting; restore or
  // delete per row or for a selection. State per user in localStorage and in the URL.
  import { page } from '$app/state';
  import { api, errorText } from '$lib/api';
  import { toast } from '$lib/state.svelte';
  import { t } from '$lib/i18n.svelte';
  import { ARCHIVE_PERIODS, ARCHIVE_SORTS, filterArchive, sortArchive, type ArchivePeriod, type ArchiveSort } from '$lib/archive';
  import { addPageView } from '$lib/overviewFilter.svelte';
  import { pageState } from '$lib/pageState.svelte';
  import type { ArchivedProject } from '$lib/types';
  import PageHeader, { type HeaderFilter } from '$lib/components/PageHeader.svelte';
  import type { FilterId } from '$lib/components/FilterDropdown.svelte';
  import ArchiveTable from '$lib/components/ArchiveTable.svelte';

  const ps = pageState('archive', {
    q: { def: '', url: 'str' },
    cust: { def: null as FilterId[] | null, url: 'ids' },
    period: { def: 'all' as string, url: 'str' },
    chip: { def: '' as string, url: 'str' },
    sort: { def: 'archived' as string, url: 'str' }
  });
  const period = $derived<ArchivePeriod>(ARCHIVE_PERIODS.includes(ps.period as ArchivePeriod) ? (ps.period as ArchivePeriod) : 'all');
  const sort = $derived<ArchiveSort>(ARCHIVE_SORTS.includes(ps.sort as ArchiveSort) ? (ps.sort as ArchiveSort) : 'archived');
  const incompleteOnly = $derived(ps.chip === 'incomplete');

  let rows = $state<ArchivedProject[]>([]);
  let loaded = $state(false);
  async function load() {
    try {
      rows = await api.get<ArchivedProject[]>('/archive');
    } catch (err) {
      toast(errorText(err));
    } finally {
      loaded = true;
    }
  }
  load();

  // customers that have something in the archive
  const customers = $derived([...new Map(rows.map((r) => [r.customer_id, r.customer_name])).entries()].sort((a, b) => a[1].localeCompare(b[1])));
  const shown = $derived(
    sortArchive(filterArchive(rows, { q: ps.q, customers: ps.cust as number[] | null, period, incompleteOnly }, new Date()), sort)
  );

  const filters = $derived<HeaderFilter[]>([
    {
      key: 'cust',
      label: t('filter.customer'),
      items: customers.map(([id, name]) => ({ id, label: name, entity: 'customer' as const })),
      value: ps.cust,
      onchange: (v: FilterId[] | null) => (ps.cust = v)
    },
    {
      key: 'period',
      label: t('archive.period'),
      options: ARCHIVE_PERIODS.map((p) => ({ value: p, label: t(`archive.period.${p}`) })),
      value: period,
      defaultValue: 'all',
      onchange: (v: string) => (ps.period = v)
    }
  ]);
  const sortLabel = (s: ArchiveSort) => (s === 'archived' ? t('archive.sort.archived') : s === 'customer' ? t('archive.sort.customer') : t(`sort.${s}`));
</script>

<svelte:head><title>{t('nav.archive')} · ProjectList</title></svelte:head>

<div>
  <PageHeader
    icon="archive"
    title={t('nav.archive')}
    subtitle="{t('archive.subtitle')} · {t('archive.count', { n: rows.length, m: shown.length })}"
    searchPlaceholder={t('ph.search.archive')}
    searchValue={ps.q}
    onSearch={(q) => (ps.q = q)}
    {filters}
    quickFilters={[{ key: 'incomplete', label: t('archive.incomplete'), active: incompleteOnly }]}
    onQuickFilter={(k) => (ps.chip = ps.chip === k ? '' : k)}
    onSaveView={(name) => addPageView(name, 'archive', page.url.search.slice(1))}
    sortOptions={ARCHIVE_SORTS.map((s) => ({ value: s, label: sortLabel(s) }))}
    activeSort={sort}
    onSortChange={(s) => (ps.sort = s)} />

  {#if loaded}
    <ArchiveTable rows={shown} showCustomer empty={rows.length ? t('archive.noMatch') : t('archive.empty')} onchanged={load} />
  {/if}
</div>
