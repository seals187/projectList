<script lang="ts">
  // "Calendar" overview: month grid with the projects on their due date.
  // Writers move a project to another day via drag & drop (changes the due date).
  import { canWrite } from '$lib/state.svelte';
  import { locale, t } from '$lib/i18n.svelte';
  import { statusDotCls, statusDotStyle } from '$lib/projectStatus';
  import { dueUrgent, type OvCustomer, type OvProject } from '$lib/overview';
  import { patchProject } from '$lib/projectActions';
  import { addMonths, dayOfMonth, fromDay, mondayOf, monthLabel, monthStart, toDay, weekday, weekdayNames } from '$lib/dates';
  import ProjectMenu from './ProjectMenu.svelte';
  import Icon from './Icon.svelte';
  import EntityIcon from './EntityIcon.svelte';

  let { groups, today }: { groups: OvCustomer[]; today: string } = $props();

  const todayDay = $derived(toDay(today));
  // svelte-ignore state_referenced_locally
  let month = $state(monthStart(toDay(today))); // first day of the shown month

  const projects = $derived(groups.flatMap((g) => g.projects));
  const byDay = $derived.by(() => {
    const m = new Map<number, OvProject[]>();
    for (const p of projects) {
      if (!p.due_date) continue;
      const d = toDay(p.due_date);
      if (!m.has(d)) m.set(d, []);
      m.get(d)!.push(p);
    }
    return m;
  });
  const undated = $derived(projects.filter((p) => !p.due_date));

  // 6 weeks from the Monday before the first of the month (complete weeks)
  const days = $derived.by(() => {
    const first = mondayOf(month);
    const last = addMonths(month, 1) - 1;
    const weeks = Math.ceil((last - first + 1) / 7);
    return Array.from({ length: weeks * 7 }, (_, i) => first + i);
  });
  const nextMonth = $derived(addMonths(month, 1));
  // Height: header card, month bar and weekday row plus the "without due date" list
  // below stay visible; the week rows share the rest of the window (min. 2.75rem).
  const rows = $derived.by(() => {
    const weeks = days.length / 7;
    const reserve = undated.length ? '26rem' : '21.5rem';
    return `auto repeat(${weeks}, minmax(2.75rem, calc((100dvh - ${reserve}) / ${weeks})))`;
  });

  // drag & drop: new due date
  const editable = canWrite();
  let dragId = $state<number | null>(null);
  let overDay = $state<number | null>(null);
  function drop(e: DragEvent, day: number) {
    e.preventDefault();
    const id = dragId;
    dragId = null;
    overDay = null;
    const p = projects.find((x) => x.id === id);
    if (p && p.due_date !== fromDay(day)) patchProject(p.id, { due_date: fromDay(day) });
  }

  let menu = $state<{ project: OvProject; x: number; y: number } | null>(null);
  function projectMenu(e: MouseEvent, project: OvProject) {
    e.preventDefault();
    menu = { project, x: e.clientX, y: e.clientY };
  }
</script>

<div class="mb-2 flex items-center gap-2">
  <h2 class="min-w-44 text-base font-semibold capitalize">{monthLabel(month, locale())}</h2>
  <button type="button" class="btn-ghost px-1.5!" onclick={() => (month = addMonths(month, -1))} aria-label={t('calendar.prev')} title={t('calendar.prev')}>
    <Icon name="chevron" size={16} class="rotate-180" />
  </button>
  <button type="button" class="btn-ghost px-1.5!" onclick={() => (month = addMonths(month, 1))} aria-label={t('calendar.next')} title={t('calendar.next')}>
    <Icon name="chevron" size={16} />
  </button>
  <button type="button" class="btn-outline py-1!" onclick={() => (month = monthStart(todayDay))}>{t('calendar.today')}</button>
</div>

<div class="overflow-x-auto">
  <!-- rows share the visible height (no page scrolling); a busy day scrolls inside its cell -->
  <div class="surface grid min-w-[48rem] grid-cols-7 overflow-hidden rounded-xl border border-zinc-200 dark:border-zinc-800" style:grid-template-rows={rows}>
    {#each weekdayNames(locale()) as w, i (i)}
      <div class="border-b border-zinc-200 bg-zinc-50 px-2 py-1 text-xs font-semibold text-zinc-500 dark:border-zinc-800 dark:bg-zinc-900">{w}</div>
    {/each}
    {#each days as d (d)}
      {@const inMonth = d >= month && d < nextMonth}
      {@const items = byDay.get(d) ?? []}
      <div
        role="group"
        aria-label={fromDay(d)}
        class="min-h-0 overflow-y-auto border-zinc-200 p-1 dark:border-zinc-800
          {weekday(d) < 6 ? 'border-r' : ''} {d < days[days.length - 7] ? 'border-b' : ''}
          {inMonth ? '' : 'bg-zinc-50/70 dark:bg-zinc-900/40'}
          {weekday(d) >= 5 && inMonth ? 'bg-zinc-50/40 dark:bg-zinc-900/20' : ''}
          {overDay === d ? 'bg-accent-50! dark:bg-accent-950/40!' : ''}"
        ondragover={(e) => {
          if (dragId === null) return;
          e.preventDefault();
          overDay = d;
        }}
        ondragleave={() => overDay === d && (overDay = null)}
        ondrop={(e) => drop(e, d)}>
        <p class="mb-0.5 flex justify-end">
          <span
            class="flex size-5 items-center justify-center rounded-full text-[11px] tabular-nums
              {d === todayDay ? 'bg-accent-600 font-semibold text-white' : inMonth ? 'text-zinc-600 dark:text-zinc-300' : 'text-zinc-400'}">
            {dayOfMonth(d)}
          </span>
        </p>
        <div class="space-y-0.5">
          {#each items as p (p.id)}
            <a
              href="/projects/{p.id}"
              draggable={editable}
              ondragstart={(e) => {
                dragId = p.id;
                e.dataTransfer?.setData('text/plain', String(p.id));
              }}
              ondragend={() => {
                dragId = null;
                overDay = null;
              }}
              oncontextmenu={(e) => projectMenu(e, p)}
              title="{p.customer.name} · {p.name}"
              class="flex items-center gap-1.5 rounded-md border bg-white px-1.5 py-0.5 text-xs shadow-xs hover:shadow-sm dark:bg-zinc-900
                {dueUrgent(p, today) ? 'border-red-300 dark:border-red-500/50' : 'border-zinc-200 dark:border-zinc-800'}
                {dragId === p.id ? 'opacity-50' : ''}">
              <span class="size-2 shrink-0 rounded-full {statusDotCls(p.status)}" style={statusDotStyle(p.status)}></span>
              <EntityIcon type="customer" size="xxs" />
              <span class="min-w-0 flex-1 truncate font-medium">{p.name}</span>
            </a>
          {/each}
        </div>
      </div>
    {/each}
  </div>
</div>

{#if undated.length}
  <section class="mt-3">
    <h3 class="section-label mb-2">{t('calendar.undated', { n: undated.length })}</h3>
    <div class="flex flex-wrap gap-1.5">
      {#each undated as p (p.id)}
        <a
          href="/projects/{p.id}"
          draggable={editable}
          ondragstart={(e) => {
            dragId = p.id;
            e.dataTransfer?.setData('text/plain', String(p.id));
          }}
          ondragend={() => {
            dragId = null;
            overDay = null;
          }}
          oncontextmenu={(e) => projectMenu(e, p)}
          class="flex items-center gap-1.5 rounded-md border border-zinc-200 bg-white px-2 py-1 text-xs hover:shadow-sm dark:border-zinc-800 dark:bg-zinc-900">
          <span class="size-2 shrink-0 rounded-full {statusDotCls(p.status)}" style={statusDotStyle(p.status)}></span>
          <EntityIcon type="customer" size="xxs" />
          <span class="font-medium">{p.name}</span>
        </a>
      {/each}
    </div>
    {#if editable}<p class="mt-2 text-xs text-zinc-400">{t('calendar.dragHint')}</p>{/if}
  </section>
{/if}

{#if menu}
  {@const m = menu}
  <ProjectMenu project={m.project} customer={m.project.customer} x={m.x} y={m.y} onclose={() => (menu = null)} />
{/if}
