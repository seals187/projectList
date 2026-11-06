<script lang="ts">
  // "List" overview: one opaque card per customer (code badge, name, summary,
  // progress ring; collapsible, also via "Collapse all" in the header), below it
  // the project rows: project · status · progress · [steps] · due · responsible.
  //
  // Two layouts (display option "Show steps" + available width):
  //   C1 "steps in the bar" (narrow window < 1400 px or many tags): tags on their
  //      own line below the name, steps small next to the percentage.
  //   C2 "steps as a column" (enough width): tags next to the name, own column
  //      "Steps" with a large number.
  // Below 900 px window width, due date and responsible move into a second line.
  import { openCustomers, toggleCustomer } from '$lib/overviewOpen.svelte';
  import { t } from '$lib/i18n.svelte';
  import { customerTone, customerCode, statusBadgeCls, statusBadgeStyle, statusLabel } from '$lib/projectStatus';
  import { dueUrgent, type OvCustomer } from '$lib/overview';
  import { customerSummary, dayMonth, percent } from '$lib/listMetrics';
  import type { DisplayValues } from '$lib/displayCore';
  import { tagsByIds } from '$lib/tags';
  import Avatar from './Avatar.svelte';
  import Icon from './Icon.svelte';
  import ProgressRing from './ProgressRing.svelte';
  import TagDots from './TagDots.svelte';

  let {
    groups,
    today,
    canCreate,
    oncreate,
    display
  }: { groups: OvCustomer[]; today: string; canCreate: boolean; oncreate: (customerId: number) => void; display: DisplayValues } = $props();

  let width = $state(1200); // list width
  let winWidth = $state(1400);
  const MANY_TAGS = 4;
  const manyTags = $derived(groups.some((g) => g.projects.some((p) => p.tag_ids.length > MANY_TAGS)));
  // C2 only when steps are shown and there is room; otherwise C1
  const wide = $derived(winWidth >= 1400 && width >= 1000 && !manyTags);
  const stepsCol = $derived(display.steps && wide);
  const narrow = $derived(winWidth < 900);

  // columns: project | status | progress | [steps] | [due] | [responsible]
  // (one grid per customer card, the rows use subgrid → aligned columns; the
  // status column takes the width of the longest status label)
  const columns = $derived(
    [
      narrow ? 'minmax(8rem,1fr)' : 'minmax(13rem,1fr)',
      'max-content',
      narrow ? 'minmax(6rem,9rem)' : stepsCol ? 'minmax(8rem,13rem)' : 'minmax(8rem,14rem)',
      stepsCol ? '5.5rem' : null,
      !narrow && display.due ? '6rem' : null,
      !narrow && display.owner ? '9.5rem' : null
    ]
      .filter(Boolean)
      .join(' ')
  );
  const barColor = (done: number, total: number) => (total > 0 && done === total ? 'bg-emerald-500' : 'bg-[var(--accent-primary)]');
</script>

<svelte:window bind:innerWidth={winWidth} />

<div bind:clientWidth={width} class="space-y-4" data-layout={stepsCol ? 'c2' : 'c1'}>
  {#each groups as g (g.customer.id)}
    {@const s = customerSummary(g.projects, today)}
    {@const open = openCustomers.has(g.customer.id)}
    {#if g.projects.length === 0}
      <!-- customer without projects: slim card -->
      <section class="surface ol-card flex items-center gap-3" aria-label={g.customer.name}>
        <span class="flex size-9 shrink-0 items-center justify-center rounded-[10px] text-xs font-bold {customerTone(g.customer.id)}">{customerCode(g.customer.name)}</span>
        <div class="min-w-0 flex-1">
          <a href="/customers/{g.customer.id}" class="block truncate text-[15px] font-bold hover:underline">{g.customer.name}</a>
          <p class="text-xs text-zinc-500">{t('overview.nProjects', { n: 0 })}</p>
        </div>
        {#if canCreate}
          <button type="button" class="shrink-0 text-sm font-semibold text-[var(--accent-primary)] hover:underline dark:text-accent-300" onclick={() => oncreate(g.customer.id)}>
            + {t('cards.projectFromTemplate')}
          </button>
        {/if}
      </section>
    {:else}
      <section class="surface ol-card" aria-label={g.customer.name}>
        <!-- customer head: click toggles -->
        <div class="flex items-center gap-3">
          <button
            type="button"
            class="flex min-w-0 flex-1 items-center gap-3 text-left"
            onclick={() => toggleCustomer(g.customer.id)}
            aria-expanded={open}
            aria-label="{t(open ? 'overview.hideProjects' : 'overview.showProjects')}: {g.customer.name}">
            <span class="flex size-9 shrink-0 items-center justify-center rounded-[10px] text-xs font-bold {customerTone(g.customer.id)}">{customerCode(g.customer.name)}</span>
            <span class="min-w-0 flex-1">
              <span class="block truncate text-[15px] font-bold">{g.customer.name}</span>
              <span class="block truncate text-xs text-zinc-500" data-testid="customer-summary">
                {t('overview.nProjects', { n: s.projects })}{#if display.steps}{' · '}{t('list.steps', { done: s.done, total: s.total })}{/if}
                · {t('list.nOpenShort', { n: s.open })}{#if s.next}{' · '}{t('list.nextShort', { date: dayMonth(s.next) })}{/if}
              </span>
            </span>
            <Icon name="chevronDown" size={15} class="shrink-0 text-zinc-400 transition-transform {open ? '' : '-rotate-90'}" />
          </button>
          <ProgressRing done={s.done} total={s.total} size={40} />
        </div>

        {#if open}
          <div class="mt-3 grid gap-x-4" style:grid-template-columns={columns}>
          <!-- column header -->
          <div class="ol-head col-span-full grid-cols-subgrid">
            <span>{t('list.col.projectName')}</span>
            <span>{t('overview.col.status')}</span>
            <span>{t('list.col.progress')}</span>
            {#if stepsCol}<span>{t('list.col.steps')}</span>{/if}
            {#if !narrow && display.due}<span>{t('overview.col.due')}</span>{/if}
            {#if !narrow && display.owner}<span>{t('list.col.team')}</span>{/if}
          </div>

          {#each g.projects as p (p.id)}
            {@const tags = tagsByIds(p.tag_ids)}
            {@const pc = percent(p.done, p.total)}
            {@const urgent = dueUrgent(p, today)}
            <a href="/projects/{p.id}" class="ol-row col-span-full grid-cols-subgrid" data-testid="project-row">
              <!-- project: name + tags (C2: same line, C1: own line + checklists) -->
              {#if wide}
                <span class="flex min-w-0 items-center gap-3">
                  <span class="max-w-[70%] shrink-0 truncate text-[13.5px] font-bold" title={p.name}>{p.name}</span>
                  <TagDots {tags} wrap={display.allTags} class="flex-1" />
                </span>
              {:else}
                <span class="min-w-0">
                  <span class="block truncate text-[13.5px] font-bold" title={p.name}>{p.name}</span>
                  <span class="mt-0.5 flex min-w-0 items-center gap-2.5">
                    <TagDots {tags} wrap={display.allTags} class="max-w-full" />
                    <span class="shrink-0 text-xs text-zinc-400">· {t('list.nChecklists', { n: p.checklists.length })}</span>
                  </span>
                </span>
              {/if}
              <span><span class="badge rounded-full! px-2.5! whitespace-nowrap {statusBadgeCls(p.status)}" style={statusBadgeStyle(p.status)}>{statusLabel(p.status)}</span></span>
              <span class="flex min-w-0 items-center gap-2.5">
                <span class="block h-1.5 min-w-0 flex-1 overflow-hidden rounded-full bg-[#eceef2] dark:bg-zinc-800">
                  <span class="block h-full rounded-full {barColor(p.done, p.total)}" style:width="{pc}%"></span>
                </span>
                {#if display.percent}<span class="w-9 shrink-0 text-right text-[13px] font-bold tabular-nums">{pc}%</span>{/if}
                {#if display.steps && !stepsCol}<span class="w-11 shrink-0 text-right text-xs text-zinc-400 tabular-nums" data-testid="steps-inline">{p.done}/{p.total}</span>{/if}
              </span>
              {#if stepsCol}
                <span class="leading-tight" data-testid="steps-col">
                  <span class="block tabular-nums"><span class="text-base font-bold">{p.done}</span><span class="text-sm text-zinc-400">/{p.total}</span></span>
                  <span class="block text-[11px] text-zinc-400">{t('list.stepsWord')}</span>
                </span>
              {/if}
              {#snippet due()}
                <span class="flex items-center gap-1.5 text-[13px] tabular-nums {urgent ? 'font-semibold text-red-600 dark:text-red-400' : 'text-zinc-600 dark:text-zinc-300'}">
                  {#if p.due_date}<Icon name="calendar" size={13} class="text-zinc-400" />{dayMonth(p.due_date)}{:else}<span class="text-zinc-400">–</span>{/if}
                </span>
              {/snippet}
              {#snippet owner()}
                <span class="flex min-w-0 items-center gap-1.5 text-[13px]">
                  {#if p.owner_id && p.owner_name}<Avatar name={p.owner_name} id={p.owner_id} size={20} /><span class="truncate">{p.owner_name}</span>{:else}<span class="text-zinc-400">–</span>{/if}
                </span>
              {/snippet}
              {#if narrow}
                {#if display.due || display.owner}
                  <span class="col-span-full flex items-center gap-5 pt-0.5">
                    {#if display.due}{@render due()}{/if}
                    {#if display.owner}{@render owner()}{/if}
                  </span>
                {/if}
              {:else}
                {#if display.due}{@render due()}{/if}
                {#if display.owner}{@render owner()}{/if}
              {/if}
            </a>
          {/each}
          </div>
        {/if}
      </section>
    {/if}
  {/each}
</div>
