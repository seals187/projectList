<script lang="ts">
  // "List" overview: key figures, then a table grouped by customer. Per project
  // its checklists as sub-rows (status, progress, due date, person responsible)
  // and "time vs. progress": progress bar with a marker for the time elapsed.
  import { openCustomers, toggleCustomer } from '$lib/overviewOpen.svelte';
  import { fmtDate, shortName, t, withoutText } from '$lib/i18n.svelte';
  import { barCls, customerCode, customerTone, daysUntil, statusBadgeCls, statusBadgeStyle, statusLabel } from '$lib/projectStatus';
  import { dueUrgent, pct, schedule, type OvCustomer, type OvProject, type Schedule } from '$lib/overview';
  import { relativeTime } from '$lib/activity';
  import { tagsByIds } from '$lib/tags';
  import type { SidebarChecklist } from '$lib/types';
  import Avatar from './Avatar.svelte';
  import Icon from './Icon.svelte';

  let {
    groups,
    today,
    canCreate,
    oncreate
  }: { groups: OvCustomer[]; today: string; canCreate: boolean; oncreate: (customerId: number) => void } = $props();

  // ---- Key figures ----------------------------------------------------------------------
  const all = $derived(groups.flatMap((g) => g.projects));
  const running = $derived(all.filter((p) => p.status === 'active' || p.status === 'review' || p.status === 'blocked'));
  const openItems = $derived(all.reduce((s, p) => s + (p.total - p.done), 0));
  const overdueItems = $derived(all.reduce((s, p) => s + Object.values(p.clExtra).reduce((n, c) => n + c.overdue, 0), 0));
  const dueSoon = $derived(
    all.filter((p) => p.due_date && p.status !== 'done' && daysUntil(p.due_date, today) <= 14).sort((a, b) => a.due_date!.localeCompare(b.due_date!))
  );
  // next upcoming due date (overdue projects are counted above, but are not "next")
  const nextDue = $derived(dueSoon.find((p) => p.due_date! >= today)?.due_date ?? null);
  const schedules = $derived(new Map(all.map((p) => [p.id, schedule(p, today)])));
  const late = $derived(all.filter((p) => p.status !== 'done' && p.status !== 'archived' && ['behind', 'overdue'].includes(schedules.get(p.id)?.state ?? '')));
  const blocked = $derived(all.filter((p) => p.status === 'blocked'));
  const shortDate = (iso: string) => fmtDate(iso).slice(0, 5); // DD.MM

  // ---- Rows ------------------------------------------------------------------------------
  const overdueOf = (p: OvProject, cl: SidebarChecklist) => p.clExtra[cl.id]?.overdue ?? 0;
  function checklistState(p: OvProject, cl: SidebarChecklist): { text: string; cls: string } {
    const overdue = overdueOf(p, cl);
    if (cl.total > 0 && cl.done === cl.total) return { text: t('list.cl.done'), cls: 'text-emerald-600 dark:text-emerald-400' };
    if (overdue) return { text: t('list.cl.overdue', { n: overdue }), cls: 'font-semibold text-red-600 dark:text-red-400' };
    if (cl.done === 0) return { text: t('list.cl.notStarted'), cls: 'text-zinc-400' };
    return { text: t('list.cl.running'), cls: 'text-zinc-500' };
  }
  function verdict(s: Schedule): { text: string; cls: string } {
    if (s.state === 'done') return { text: t('list.sched.done'), cls: 'text-emerald-600 dark:text-emerald-400' };
    if (s.state === 'overdue') return { text: t('list.sched.overdue'), cls: 'text-red-600 dark:text-red-400' };
    if (s.state === 'behind') return { text: t('list.sched.behind', { n: s.behind }), cls: s.behind >= 25 ? 'text-red-600 dark:text-red-400' : 'text-amber-600 dark:text-amber-400' };
    return { text: t('list.sched.onTrack'), cls: 'text-emerald-600 dark:text-emerald-400' };
  }

  // Columns: project/checklist | status | progress | time vs. progress | due | responsible | updated
  const cols =
    'grid grid-cols-[minmax(15rem,2fr)_7.5rem_minmax(8.5rem,1fr)_minmax(10rem,1.15fr)_6.5rem_8.5rem_6.5rem] items-center gap-x-4';
  const kpi = 'card px-4 py-3';
</script>

<!-- Key figures -->
<div class="mb-4 grid grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-5">
  <div class={kpi}>
    <p class="text-sm text-zinc-500">{t('overview.activeProjects')}</p>
    <p class="mt-0.5 text-2xl font-bold tabular-nums">{running.length} <span class="text-sm font-normal text-zinc-400">{t('overview.ofN', { n: all.length })}</span></p>
  </div>
  <div class={kpi}>
    <p class="text-sm text-zinc-500">{t('list.openItems')}</p>
    <p class="mt-0.5 text-2xl font-bold tabular-nums">
      {openItems}
      {#if overdueItems}<span class="text-sm font-normal text-red-600 dark:text-red-400">{t('list.nOverdue', { n: overdueItems })}</span>{/if}
    </p>
  </div>
  <div class={kpi}>
    <p class="text-sm text-zinc-500">{t('overview.dueSoon')}</p>
    <p class="mt-0.5 truncate text-2xl font-bold tabular-nums">
      {dueSoon.length}
      {#if nextDue}<span class="text-sm font-normal text-zinc-400">{t('list.next', { date: shortDate(nextDue) })}</span>{/if}
    </p>
  </div>
  <div class={kpi}>
    <p class="text-sm text-zinc-500">{t('list.late')}</p>
    <p class="mt-0.5 truncate text-2xl font-bold tabular-nums {late.length ? 'text-amber-600 dark:text-amber-400' : ''}">
      {late.length} <span class="text-sm font-normal text-zinc-400">{t('list.lateHint')}</span>
    </p>
  </div>
  <div class={kpi}>
    <p class="text-sm text-zinc-500">{t('overview.blocked')}</p>
    <p class="mt-0.5 truncate text-2xl font-bold tabular-nums {blocked.length ? 'text-red-600 dark:text-red-400' : ''}">
      {blocked.length}
      <span class="text-sm font-normal text-zinc-400">{blocked.map((p) => p.name).join(', ')}</span>
    </p>
  </div>
</div>

<!-- Legend for "time vs. progress" -->
<p class="surface mb-3 flex w-fit items-center gap-4 rounded-md text-xs text-zinc-500 in-[.has-bg]:px-2.5 in-[.has-bg]:py-1">
  <span class="flex items-center gap-1.5"><span class="h-2 w-4 rounded-full bg-accent-500"></span>{t('list.legendProgress')}</span>
  <span class="flex items-center gap-1.5"><span class="h-3 w-0.5 rounded bg-zinc-800 dark:bg-zinc-200"></span>{t('list.legendTime')}</span>
</p>

<div class="surface overflow-x-auto rounded-xl border border-zinc-200 dark:border-zinc-800">
  <div class="min-w-[72rem]">
    <div class="{cols} border-b border-zinc-200 px-4 py-2.5 text-[11px] font-semibold tracking-wider text-zinc-400 uppercase dark:border-zinc-800">
      <span class="pl-9">{t('list.col.project')}</span>
      <span>{t('overview.col.status')}</span>
      <span>{t('list.col.progress')}</span>
      <span>{t('list.col.schedule')}</span>
      <span>{t('overview.col.due')}</span>
      <span>{t('list.col.team')}</span>
      <span>{t('list.col.updated')}</span>
    </div>

    {#each groups as g (g.customer.id)}
      {@const open = openCustomers.has(g.customer.id)}
      <div class="flex items-center gap-3 border-b border-zinc-200 bg-zinc-50/80 px-4 py-2.5 dark:border-zinc-800 dark:bg-zinc-900/50">
        <button type="button" class="rounded p-0.5 text-zinc-400 hover:text-zinc-700 dark:hover:text-zinc-200" onclick={() => toggleCustomer(g.customer.id)} aria-expanded={open} aria-label="{t(open ? 'overview.hideProjects' : 'overview.showProjects')}: {g.customer.name}">
          <Icon name="chevronDown" size={15} class="transition-transform {open ? '' : '-rotate-90'}" />
        </button>
        <span class="flex size-7 shrink-0 items-center justify-center rounded-lg text-[11px] font-bold {customerTone(g.customer.id)}">{customerCode(g.customer.name)}</span>
        <a href="/customers/{g.customer.id}" class="truncate font-semibold hover:underline">{g.customer.name}</a>
        <span class="shrink-0 text-sm text-zinc-400">
          {t('overview.nProjects', { n: g.projects.length })}{g.projects.length ? ` · ${t('overview.overall', { pct: pct(g.done, g.total) })}` : ''}{g.excluded ? ` · ${withoutText(g.excluded)}` : ''}
        </span>
        {#if canCreate}
          <button type="button" class="ml-auto shrink-0 text-sm font-medium text-accent-600 hover:underline dark:text-accent-400" onclick={() => oncreate(g.customer.id)}>
            + {t('overview.addProject')}
          </button>
        {/if}
      </div>

      {#if open}
        {#each g.projects as p (p.id)}
          {@const s = schedules.get(p.id)}
          {@const tags = tagsByIds(p.tag_ids)}
          <div class="border-b border-zinc-100 last:border-b-0 dark:border-zinc-800/70 {p.status === 'blocked' ? 'bg-red-50/50 dark:bg-red-950/15' : ''}">
            <!-- project -->
            <div class="{cols} px-4 py-3 text-sm">
              <span class="flex min-w-0 items-start gap-2.5 pl-9">
                <Icon name="folder" size={16} class="mt-0.5 shrink-0 text-zinc-400" />
                <span class="min-w-0">
                  <a href="/projects/{p.id}" class="block truncate font-semibold hover:text-accent-700 hover:underline dark:hover:text-accent-400" title={p.name}>{p.name}</a>
                  {#if tags.length}
                    <span class="mt-0.5 flex flex-wrap gap-x-2.5 text-xs text-zinc-500">
                      {#each tags as tag (tag.id)}<span class="flex items-center gap-1"><span class="size-2 rounded-full" style:background-color={tag.color}></span>{tag.name}</span>{/each}
                    </span>
                  {/if}
                </span>
              </span>
              <span><span class="badge {statusBadgeCls(p.status)}" style={statusBadgeStyle(p.status)}>{statusLabel(p.status)}</span></span>
              <span class="flex items-center gap-2">
                <span class="block h-1.5 min-w-0 flex-1 overflow-hidden rounded-full bg-zinc-200 dark:bg-zinc-800">
                  <span class="block h-full rounded-full {barCls(p.done, p.total)}" style:width="{pct(p.done, p.total)}%"></span>
                </span>
                <span class="w-9 text-right font-semibold tabular-nums">{pct(p.done, p.total)}%</span>
              </span>
              <!-- time vs. progress -->
              {#if s}
                {@const v = verdict(s)}
                <span class="min-w-0" title={t('list.schedTitle', { time: s.elapsed, progress: s.progress })}>
                  <span class="relative block h-2 rounded-full bg-zinc-200 dark:bg-zinc-800">
                    <span class="absolute inset-y-0 left-0 rounded-full {s.state === 'done' ? 'bg-emerald-500' : 'bg-accent-500'}" style:width="{s.progress}%"></span>
                    <span class="absolute -top-1 -bottom-1 w-0.5 -translate-x-1/2 rounded bg-zinc-800 dark:bg-zinc-200" style:left="{s.elapsed}%"></span>
                  </span>
                  <span class="mt-1 flex justify-between text-[11px] text-zinc-400 tabular-nums"><span>{shortDate(s.start)}</span><span>{shortDate(s.due)}</span></span>
                  <span class="block text-xs font-semibold {v.cls}">{v.text}</span>
                </span>
              {:else}
                <span class="text-xs text-zinc-400">{t('list.sched.noDue')}</span>
              {/if}
              <span class="flex items-center gap-1 tabular-nums {dueUrgent(p, today) ? 'font-semibold text-red-600 dark:text-red-400' : 'text-zinc-600 dark:text-zinc-300'}">
                {#if p.due_date}<Icon name="calendar" size={13} />{fmtDate(p.due_date)}{:else}<span class="text-zinc-400">–</span>{/if}
              </span>
              <span class="flex min-w-0 items-center gap-1.5 text-xs text-zinc-600 dark:text-zinc-300">
                {#if p.owner_id && p.owner_name}<Avatar name={p.owner_name} id={p.owner_id} size={20} /><span class="truncate" title="{t('project.owner')}: {p.owner_name}">{shortName(p.owner_name)}</span>{:else}<span class="text-zinc-400">–</span>{/if}
              </span>
              <span class="flex items-center gap-1 text-xs text-zinc-400" title={p.updated}><Icon name="clock" size={12} />{relativeTime(p.updated)}</span>
            </div>

            <!-- checklists -->
            {#each p.checklists as cl, i (cl.id)}
              {@const st = checklistState(p, cl)}
              {@const x = p.clExtra[cl.id]}
              <div class="{cols} px-4 pb-2 text-sm {i === p.checklists.length - 1 ? 'pb-3' : ''}">
                <span class="relative flex min-w-0 items-center gap-2 pl-[3.25rem]">
                  <span class="absolute top-[-0.6rem] left-[2.85rem] h-[1.15rem] w-2.5 rounded-bl border-b border-l border-zinc-300 dark:border-zinc-700"></span>
                  <Icon name="fileText" size={14} class="shrink-0 text-zinc-400" />
                  <a href="/checklists/{cl.id}" class="truncate text-zinc-700 hover:text-accent-700 hover:underline dark:text-zinc-300 dark:hover:text-accent-400" title={cl.title}>{cl.title}</a>
                </span>
                <span class="text-xs {st.cls}">{st.text}</span>
                <span class="flex items-center gap-2">
                  <span class="block h-1 min-w-0 flex-1 overflow-hidden rounded-full bg-zinc-200 dark:bg-zinc-800">
                    <span class="block h-full rounded-full {barCls(cl.done, cl.total)}" style:width="{pct(cl.done, cl.total)}%"></span>
                  </span>
                  <span class="w-9 text-right text-xs text-zinc-500 tabular-nums">{cl.done}/{cl.total}</span>
                </span>
                <span></span>
                <span class="text-xs text-zinc-500 tabular-nums">{cl.next_due ? fmtDate(cl.next_due) : ''}</span>
                <span class="flex min-w-0 items-center gap-1.5 text-xs text-zinc-600 dark:text-zinc-300">
                  {#if x?.assignee_id}<Avatar name={x.assignee_name} id={x.assignee_id} size={18} /><span class="truncate">{shortName(x.assignee_name)}</span>{/if}
                </span>
                <span></span>
              </div>
            {/each}
          </div>
        {:else}
          <p class="border-b border-zinc-100 px-4 py-2.5 pl-[3.25rem] text-sm text-zinc-400 last:border-b-0 dark:border-zinc-800/70">
            {t('overview.noProjectsYet')}{#if canCreate}{' – '}<button type="button" class="font-medium text-accent-600 hover:underline dark:text-accent-400" onclick={() => oncreate(g.customer.id)}>{t('overview.createProject')}</button>{/if}
          </p>
        {/each}
      {/if}
    {/each}
  </div>
</div>
