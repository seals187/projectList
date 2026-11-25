<script lang="ts">
  // Table of archived projects (archive page and the archive tab of a customer):
  // project, optionally customer and "by", date of archiving, checklist progress;
  // per row (on hover) "Restore" and "Delete permanently"; checkboxes for both
  // as bulk actions. group = "year": rows under collapsible year headings.
  import { errorText } from '$lib/api';
  import { canWrite } from '$lib/session.svelte';
  import { workspace } from '$lib/workspace.svelte';
  import { toast } from '$lib/toasts.svelte';
  import { fmtDate, t } from '$lib/i18n.svelte';
  import { groupByYear, localDay, type YearGroup } from '$lib/archive';
  import { barCls } from '$lib/projectStatus';
  import { pct } from '$lib/projectQuery';
  import type { ArchivedProject } from '$lib/types';
  import EntityIcon from './EntityIcon.svelte';
  import ConfirmButton from './ConfirmButton.svelte';
  import Avatar from './Avatar.svelte';
  import Icon from './Icon.svelte';

  let {
    rows,
    showCustomer = false,
    group = 'none',
    empty,
    onchanged
  }: {
    rows: ArchivedProject[];
    showCustomer?: boolean; // archive page: customer and "by" columns
    group?: 'none' | 'year';
    empty: string; // text when there are no rows
    onchanged: () => void; // after restoring / deleting
  } = $props();

  const editable = $derived(canWrite());
  const cols = $derived(
    showCustomer
      ? 'grid grid-cols-[1.5rem_minmax(12rem,2fr)_minmax(9rem,1.3fr)_6.5rem_2.5rem_minmax(9rem,1fr)_minmax(13rem,auto)] items-center gap-x-4'
      : 'grid grid-cols-[1.5rem_minmax(12rem,2fr)_10rem_minmax(9rem,1fr)_minmax(13rem,auto)] items-center gap-x-4'
  );

  const groups = $derived<YearGroup[]>(group === 'year' ? groupByYear(rows) : [{ year: null, rows }]);
  let closed = $state(new Set<string>());
  const groupKey = (g: YearGroup) => g.year ?? '?';
  function toggleGroup(k: string) {
    const next = new Set(closed);
    if (next.has(k)) next.delete(k);
    else next.add(k);
    closed = next;
  }

  // selection (only rows that are still shown)
  let selected = $state(new Set<number>());
  const visibleIds = $derived(new Set(rows.map((r) => r.id)));
  const sel = $derived([...selected].filter((id) => visibleIds.has(id)));
  const allSelected = $derived(rows.length > 0 && sel.length === rows.length);
  function toggle(id: number) {
    const next = new Set(selected);
    if (next.has(id)) next.delete(id);
    else next.add(id);
    selected = next;
  }
  const toggleAll = () => (selected = allSelected ? new Set() : new Set(rows.map((r) => r.id)));

  let busy = $state(false);
  async function run(ids: number[], op: (id: number) => Promise<unknown>, doneKey: 'archive.restored' | 'archive.deleted') {
    if (!ids.length || busy) return;
    busy = true;
    let ok = 0;
    try {
      // one reload of the workspace for the whole selection
      await workspace.batch(async () => {
        for (const id of ids) {
          await op(id);
          ok++;
        }
      });
    } catch (err) {
      toast(errorText(err));
    } finally {
      busy = false;
      if (ok) toast(t(doneKey, { n: ok }), 'info');
      selected = new Set([...selected].filter((id) => !ids.includes(id)));
      onchanged();
    }
  }
  const restore = (ids: number[]) => run(ids, (id) => workspace.updateProject(id, { status: 'active' }), 'archive.restored');
  const remove = (ids: number[]) => run(ids, (id) => workspace.deleteProject(id), 'archive.deleted');

  const day = (p: ArchivedProject) => (p.archived_at ? fmtDate(localDay(p.archived_at)) : null);
</script>

<div class="surface overflow-x-auto rounded-xl border border-zinc-200 dark:border-zinc-800" data-testid="archive-table">
  <div class={showCustomer ? 'min-w-[60rem]' : 'min-w-[46rem]'}>
    {#if showCustomer}
      <!-- column heads -->
      <div class="{cols} border-b border-zinc-200 px-4 py-2.5 text-[11px] font-semibold tracking-wider text-zinc-400 uppercase dark:border-zinc-800">
        <span>
          {#if editable && rows.length}
            <input type="checkbox" class="size-4 accent-accent-600" checked={allSelected} onchange={toggleAll} aria-label={t('archive.selectAll')} />
          {/if}
        </span>
        <span>{t('archive.col.project')}</span>
        <span>{t('archive.col.customer')}</span>
        <span>{t('archive.col.archived')}</span>
        <span>{t('archive.col.by')}</span>
        <span>{t('archive.col.checklists')}</span>
        <span></span>
      </div>
    {/if}

    <!-- bulk actions -->
    {#if editable && sel.length}
      <div class="flex items-center gap-3 border-b border-zinc-200 bg-accent-50/60 px-4 py-2 text-sm dark:border-zinc-800 dark:bg-accent-950/30" data-testid="archive-bulk">
        <span class="font-semibold">{t('archive.selected', { n: sel.length })}</span>
        <button type="button" class="btn-outline py-1!" disabled={busy} onclick={() => restore(sel)}><Icon name="undo" size={14} />{t('archive.restore')}</button>
        <ConfirmButton class="btn-ghost py-1! text-red-600! hover:bg-red-50! dark:text-red-400! dark:hover:bg-red-950/40!" onconfirm={() => remove(sel)} label={t('archive.delete')} confirmLabel={t('archive.deleteConfirm')}>
          <Icon name="trash" size={14} />{t('archive.delete')}
        </ConfirmButton>
        <button type="button" class="ml-auto text-xs text-zinc-500 hover:text-zinc-800 dark:hover:text-zinc-200" onclick={() => (selected = new Set())}>{t('common.cancel')}</button>
      </div>
    {/if}

    {#if !rows.length}
      <p class="px-4 py-6 text-center text-sm text-zinc-400">{empty}</p>
    {/if}
    {#each rows.length ? groups : [] as g (groupKey(g))}
      {@const k = groupKey(g)}
      {@const open = !closed.has(k)}
      {#if group === 'year'}
        <button
          type="button"
          class="flex w-full items-center gap-2 border-b border-zinc-200 bg-zinc-50 px-4 py-2 text-left text-sm font-semibold dark:border-zinc-800 dark:bg-zinc-800/40"
          onclick={() => toggleGroup(k)}
          aria-expanded={open}>
          <Icon name="chevronDown" size={13} class="text-zinc-400 transition-transform {open ? '' : '-rotate-90'}" />
          {g.year ?? t('archive.unknownDate')}
          <span class="font-normal text-zinc-400 tabular-nums">{g.rows.length}</span>
        </button>
      {/if}
      {#if open}
        {#each g.rows as p (p.id)}
          {@const d = day(p)}
          <div class="{cols} group relative border-b border-zinc-100 px-4 py-2.5 text-sm hover:bg-accent-50/40 dark:border-zinc-800/70 dark:hover:bg-zinc-800/40" data-testid="archive-row">
            <span class="relative z-10">
              {#if editable}
                <input type="checkbox" class="size-4 accent-accent-600" checked={selected.has(p.id)} onchange={() => toggle(p.id)} aria-label="{t('archive.select')}: {p.name}" />
              {/if}
            </span>
            <span class="flex min-w-0 items-center gap-2.5">
              <EntityIcon type="project" size="sm" />
              <!-- the link covers the whole row; checkbox, customer and buttons sit above it -->
              <a href="/projects/{p.id}" class="truncate font-medium after:absolute after:inset-0 hover:underline" title={p.name}>{p.name}</a>
            </span>
            {#if showCustomer}
              <a href="/customers/{p.customer_id}?tab=archive" class="relative z-10 truncate text-accent-700 hover:underline dark:text-accent-400" title={p.customer_name}>{p.customer_name}</a>
              <span class="text-zinc-600 tabular-nums dark:text-zinc-300">{d ?? '–'}</span>
              <span>
                {#if p.archived_by && p.archived_by_name}
                  <span class="relative z-10 inline-flex" title={p.archived_by_name}><Avatar name={p.archived_by_name} id={p.archived_by} size={26} /></span>
                {:else}<span class="text-zinc-400">–</span>{/if}
              </span>
            {:else}
              <span class="text-zinc-500 tabular-nums">{d ? t('archive.archivedOnLower', { date: d }) : t('archive.unknownDate')}</span>
            {/if}
            <span class="flex items-center gap-2">
              {#if p.total}
                <span class="block h-1.5 max-w-36 min-w-0 flex-1 overflow-hidden rounded-full bg-zinc-200 dark:bg-zinc-800">
                  <span class="block h-full rounded-full {barCls(p.done, p.total)}" style:width="{pct(p.done, p.total)}%"></span>
                </span>
                <span class="shrink-0 text-xs text-zinc-500 tabular-nums">{p.done}/{p.total}</span>
              {:else}
                <span class="text-xs text-zinc-400">{t('customer.noChecklist')}</span>
              {/if}
            </span>
            <span class="relative z-10 flex justify-end gap-1.5 opacity-0 group-focus-within:opacity-100 group-hover:opacity-100 [@media(hover:none)]:opacity-100">
              {#if editable}
                <button type="button" class="btn-outline py-1!" disabled={busy} onclick={() => restore([p.id])}><Icon name="undo" size={14} />{t('archive.restore')}</button>
                <ConfirmButton
                  class="btn-outline px-2! py-1! text-red-600! hover:bg-red-50! dark:text-red-400! dark:hover:bg-red-950/40!"
                  onconfirm={() => remove([p.id])}
                  label={t('archive.delete')}
                  confirmLabel={t('archive.deleteConfirm')}>
                  <Icon name="trash" size={14} />
                </ConfirmButton>
              {/if}
            </span>
          </div>
        {/each}
      {/if}
    {/each}
  </div>
</div>
