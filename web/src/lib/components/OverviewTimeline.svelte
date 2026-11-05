<script lang="ts">
  // "Timeline" overview (Gantt): one bar per project from its start date (or
  // creation date) to its due date, grouped by customer. Writers drag a bar to
  // move it, or its ends to change the start / due date.
  import { goto } from '$app/navigation';
  import { canWrite } from '$lib/state.svelte';
  import { fmtDate, locale, t } from '$lib/i18n.svelte';
  import { customerCode, customerTone, statusColor, statusDotCls, statusDotStyle, statusLabel } from '$lib/projectStatus';
  import { pct, type OvCustomer, type OvProject } from '$lib/overview';
  import { patchProject } from '$lib/projectActions';
  import { addMonths, dayOfMonth, fromDay, isoWeek, mondayOf, monthLabel, monthStart, toDay, weekday, weekdayNames } from '$lib/dates';
  import ProjectMenu from './ProjectMenu.svelte';
  import Icon from './Icon.svelte';

  let { groups, today }: { groups: OvCustomer[]; today: string } = $props();

  type Zoom = 'weeks' | 'months';
  const ZOOM = { weeks: { dayW: 30, days: 63 }, months: { dayW: 8, days: 196 } } as const;
  let zoom = $state<Zoom>('weeks');
  const dayW = $derived(ZOOM[zoom].dayW);
  const span = $derived(ZOOM[zoom].days);
  const NAME_W = 240; // px, left column

  const todayDay = $derived(toDay(today));
  const startFor = (z: Zoom, around: number) => (z === 'weeks' ? mondayOf(around) - 14 : addMonths(monthStart(around), -1));
  // svelte-ignore state_referenced_locally
  let from = $state(startFor('weeks', toDay(today)));
  const to = $derived(from + span); // exclusive
  function setZoom(z: Zoom) {
    zoom = z;
    from = startFor(z, todayDay);
  }
  const shift = (dir: -1 | 1) => (from = zoom === 'weeks' ? from + dir * 28 : addMonths(from, dir * 2));

  // ---- Bars ------------------------------------------------------------------------
  const startOf = (p: OvProject) => toDay(p.start_date ?? p.created_at);
  function range(p: OvProject): [number, number] {
    const s = startOf(p);
    const e = p.due_date ? toDay(p.due_date) : s;
    return s <= e ? [s, e] : [e, s];
  }

  const rows = $derived(
    groups.map((g) => ({ customer: g.customer, projects: [...g.projects].sort((a, b) => range(a)[0] - range(b)[0] || a.name.localeCompare(b.name)) }))
  );

  // header: months (and days or calendar weeks)
  const months = $derived.by(() => {
    const out: { day: number; label: string }[] = [];
    for (let m = monthStart(from); m < to; m = addMonths(m, 1)) out.push({ day: Math.max(m, from), label: monthLabel(m, locale(), zoom === 'months' || m <= from) });
    return out;
  });
  const days = $derived(Array.from({ length: span }, (_, i) => from + i));
  const weekStarts = $derived(days.filter((d) => weekday(d) === 0));
  const wd = $derived(weekdayNames(locale()));

  // ---- Dragging --------------------------------------------------------------------------
  const editable = canWrite();
  type Mode = 'move' | 'start' | 'end';
  let drag = $state<{ id: number; mode: Mode; x0: number; delta: number; moved: boolean } | null>(null);

  function down(e: PointerEvent, p: OvProject, mode: Mode) {
    if (e.button !== 0) return;
    e.stopPropagation();
    if (!editable) return;
    (e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
    drag = { id: p.id, mode, x0: e.clientX, delta: 0, moved: false };
  }
  function move(e: PointerEvent) {
    if (!drag) return;
    const dx = e.clientX - drag.x0;
    if (Math.abs(dx) > 3) drag.moved = true;
    drag.delta = Math.round(dx / dayW);
  }
  // range of a bar while it is being dragged
  function shown(p: OvProject): [number, number] {
    const [s, e] = range(p);
    if (!drag || drag.id !== p.id) return [s, e];
    const d = drag.delta;
    if (drag.mode === 'move') return [s + d, e + d];
    if (drag.mode === 'start') return [Math.min(s + d, e), e];
    return [s, Math.max(e + d, s)];
  }
  function up(p: OvProject) {
    if (!drag) return;
    const { moved, delta, mode } = drag;
    const [s, e] = shown(p);
    drag = null;
    if (!moved) {
      if (mode === 'move') goto(`/projects/${p.id}`);
      return;
    }
    if (!delta) return;
    const body: { start_date?: string; due_date?: string } = {};
    if (mode !== 'end') body.start_date = fromDay(s);
    // without a due date only the start moves
    if (mode !== 'start' && (p.due_date || mode === 'end')) body.due_date = fromDay(e);
    patchProject(p.id, body);
  }

  let menu = $state<{ project: OvProject; x: number; y: number } | null>(null);
  function projectMenu(e: MouseEvent, project: OvProject) {
    e.preventDefault();
    menu = { project, x: e.clientX, y: e.clientY };
  }

  const width = $derived(NAME_W + span * dayW);
  const x = (day: number) => (day - from) * dayW;
  // weekends shaded (weeks view; "from" is always a Monday there)
  const weekendBg = $derived(
    zoom === 'weeks'
      ? `background-image:repeating-linear-gradient(90deg,transparent 0 ${5 * dayW}px,var(--weekend) ${5 * dayW}px ${7 * dayW}px);background-position-x:${NAME_W}px`
      : ''
  );
  const barColor = (p: OvProject) => statusColor(p.status);
</script>

<svelte:window onpointermove={move} />

<div class="mb-3 flex flex-wrap items-center gap-2">
  <h2 class="min-w-44 text-lg font-semibold capitalize">{monthLabel(from, locale())}</h2>
  <button type="button" class="btn-ghost px-1.5!" onclick={() => shift(-1)} aria-label={t('calendar.prev')} title={t('calendar.prev')}>
    <Icon name="chevron" size={16} class="rotate-180" />
  </button>
  <button type="button" class="btn-ghost px-1.5!" onclick={() => shift(1)} aria-label={t('calendar.next')} title={t('calendar.next')}>
    <Icon name="chevron" size={16} />
  </button>
  <button type="button" class="btn-outline py-1!" onclick={() => (from = startFor(zoom, todayDay))}>{t('calendar.today')}</button>
  <div class="ml-auto inline-flex gap-0.5 rounded-lg bg-zinc-100 p-0.5 text-sm dark:bg-zinc-800/70" role="group" aria-label={t('timeline.zoom')}>
    {#each ['weeks', 'months'] as const as z (z)}
      <button
        type="button"
        class="rounded-md px-2.5 py-0.5 {zoom === z ? 'seg-on' : 'sel-off font-medium'}"
        aria-pressed={zoom === z}
        onclick={() => setZoom(z)}>
        {t(`timeline.${z}`)}
      </button>
    {/each}
  </div>
</div>

<div class="surface overflow-x-auto rounded-xl border border-zinc-200 dark:border-zinc-800 [--weekend:rgb(0_0_0/0.035)] dark:[--weekend:rgb(255_255_255/0.035)]">
  <div class="relative text-sm" style:width="{width}px" style={weekendBg}>
    <!-- header -->
    <div class="sticky top-0 z-20 border-b border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-950">
      <div class="flex h-7">
        <div class="sticky left-0 z-10 shrink-0 border-r border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-950" style:width="{NAME_W}px"></div>
        <div class="relative flex-1">
          {#each months as m (m.day)}
            <span class="absolute top-0 h-full border-l border-zinc-200 px-1.5 pt-1 text-xs font-semibold whitespace-nowrap text-zinc-600 capitalize dark:border-zinc-800 dark:text-zinc-300" style:left="{x(m.day)}px">
              {m.label}
            </span>
          {/each}
        </div>
      </div>
      <div class="flex h-6 text-[10px] text-zinc-400">
        <div class="sticky left-0 z-10 flex shrink-0 items-center border-r border-zinc-200 bg-white px-3 font-semibold tracking-wider uppercase dark:border-zinc-800 dark:bg-zinc-950" style:width="{NAME_W}px">
          {t('overview.project')}
        </div>
        <div class="relative flex-1">
          {#if zoom === 'weeks'}
            {#each days as d (d)}
              <span class="absolute top-0 flex h-full flex-col items-center justify-center leading-none tabular-nums {d === todayDay ? 'font-bold text-accent-600 dark:text-accent-400' : ''}" style:left="{x(d)}px" style:width="{dayW}px" title={fmtDate(fromDay(d))}>
                {dayOfMonth(d)}<span class="text-[8px] opacity-70">{wd[weekday(d)].slice(0, 2)}</span>
              </span>
            {/each}
          {:else}
            {#each weekStarts as d (d)}
              <span class="absolute top-0 flex h-full items-center border-l border-zinc-100 pl-1 tabular-nums dark:border-zinc-800/60" style:left="{x(d)}px">{isoWeek(d)}</span>
            {/each}
          {/if}
        </div>
      </div>
    </div>

    <!-- today -->
    {#if todayDay >= from && todayDay < to}
      <div class="pointer-events-none absolute top-0 bottom-0 z-10 w-px bg-accent-500/70" style:left="{NAME_W + x(todayDay) + dayW / 2}px"></div>
    {/if}

    {#each rows as r (r.customer.id)}
      <div class="flex h-8 items-center border-b border-zinc-100 bg-zinc-50/80 dark:border-zinc-800/70 dark:bg-zinc-900/60">
        <a
          href="/customers/{r.customer.id}"
          class="sticky left-0 z-10 flex h-full shrink-0 items-center gap-2 border-r border-zinc-200 bg-zinc-50 px-3 font-semibold hover:underline dark:border-zinc-800 dark:bg-zinc-900"
          style:width="{NAME_W}px">
          <span class="flex size-5 shrink-0 items-center justify-center rounded text-[8px] font-bold {customerTone(r.customer.id)}">{customerCode(r.customer.name)}</span>
          <span class="truncate">{r.customer.name}</span>
        </a>
      </div>
      {#each r.projects as p (p.id)}
        {@const [s, e] = shown(p)}
        {@const c = barColor(p)}
        {@const left = x(s)}
        {@const w = (e - s + 1) * dayW}
        <div class="flex h-9 items-center border-b border-zinc-100 dark:border-zinc-800/70">
          <a
            href="/projects/{p.id}"
            class="sticky left-0 z-10 flex h-full shrink-0 items-center gap-2 border-r border-zinc-200 bg-white pr-2 pl-5 hover:text-accent-700 dark:border-zinc-800 dark:bg-zinc-950 dark:hover:text-accent-400"
            style:width="{NAME_W}px"
            oncontextmenu={(ev) => projectMenu(ev, p)}
            title={p.name}>
            <span class="size-2 shrink-0 rounded-full {statusDotCls(p.status)}" style={statusDotStyle(p.status)}></span>
            <span class="min-w-0 flex-1 truncate">{p.name}</span>
          </a>
          <div class="relative h-full flex-1 overflow-hidden">
            {#if e >= from && s < to}
              <!-- bar: status color, filled part = progress -->
              <div
                role="button"
                tabindex="-1"
                class="group/bar absolute top-1.5 bottom-1.5 flex items-center overflow-hidden rounded-md text-xs font-medium shadow-xs select-none
                  {editable ? (drag?.id === p.id ? 'cursor-grabbing' : 'cursor-grab') : 'cursor-pointer'}
                  {!p.due_date ? 'border border-dashed border-zinc-400 dark:border-zinc-500' : ''}
                  {c ? '' : statusDotCls(p.status)}"
                style:left="{left}px"
                style:width="{Math.max(w, dayW)}px"
                style:background-color={c || undefined}
                title="{p.name} · {statusLabel(p.status)} · {fmtDate(fromDay(s))}{p.due_date ? ` – ${fmtDate(fromDay(e))}` : ''} · {pct(p.done, p.total)}%"
                onpointerdown={(ev) => down(ev, p, 'move')}
                onpointerup={() => up(p)}
                oncontextmenu={(ev) => projectMenu(ev, p)}>
                <span class="absolute inset-0 bg-white/55 dark:bg-zinc-950/55"></span>
                <span class="absolute inset-y-0 left-0 opacity-90 {c ? '' : statusDotCls(p.status)}" style:background-color={c || undefined} style:width="{pct(p.done, p.total)}%"></span>
                <span class="relative truncate px-2 text-zinc-900 dark:text-zinc-50">{p.name}</span>
                {#if editable}
                  <span class="absolute inset-y-0 left-0 w-2 cursor-ew-resize opacity-0 group-hover/bar:opacity-100 group-hover/bar:bg-black/10" onpointerdown={(ev) => down(ev, p, 'start')} onpointerup={() => up(p)} role="presentation"></span>
                  <span class="absolute inset-y-0 right-0 w-2 cursor-ew-resize opacity-0 group-hover/bar:opacity-100 group-hover/bar:bg-black/10" onpointerdown={(ev) => down(ev, p, 'end')} onpointerup={() => up(p)} role="presentation"></span>
                {/if}
              </div>
            {:else}
              <!-- outside the shown range: arrow towards it -->
              <span class="absolute top-1/2 -translate-y-1/2 text-xs text-zinc-400 {s >= to ? 'right-2' : 'left-2'}">
                {s >= to ? `${fmtDate(fromDay(s))} →` : `← ${fmtDate(fromDay(e))}`}
              </span>
            {/if}
          </div>
        </div>
      {/each}
    {/each}
  </div>
</div>
<p class="mt-2 text-xs text-zinc-400">{t(editable ? 'timeline.hintEdit' : 'timeline.hint')}</p>

{#if menu}
  {@const m = menu}
  <ProjectMenu project={m.project} customer={m.project.customer} x={m.x} y={m.y} onclose={() => (menu = null)} />
{/if}
