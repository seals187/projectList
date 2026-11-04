<script lang="ts">
  // Meine Aufgaben:
  //  - „Kunden“: Kunden → Projekte → Checklisten, in denen mir etwas zugewiesen ist
  //    (Checkliste oder Eintrag), mit Kundenfortschritt und Schalter je Projekt,
  //    ob es in den Kundenfortschritt einfliesst.
  //  - „Persönlich“: mir zugewiesene Einträge zum Abhaken.
  //  Filter: Offen / Heute (fällig oder überfällig) / Erledigt.
  import { SvelteSet } from 'svelte/reactivity';
  import { api, errorText } from '$lib/api';
  import { refreshSidebar, session, sidebar, toast } from '$lib/state.svelte';
  import { locale, t, todayISO, withoutText } from '$lib/i18n.svelte';
  import { barCls, customerCode, customerLabel, customerProgress, customerTone, daysUntil, STATUS_BADGE } from '$lib/projectStatus';
  import { pct } from '$lib/overview';
  import type { OverviewExtra, SidebarChecklist, SidebarCustomer, SidebarProject, Task } from '$lib/types';
  import SaveStatus from '$lib/components/SaveStatus.svelte';
  import TaskList from '$lib/components/TaskList.svelte';
  import ProgressToggle from '$lib/components/ProgressToggle.svelte';
  import Icon from '$lib/components/Icon.svelte';

  type Tab = 'customers' | 'personal';
  type Filter = 'open' | 'today' | 'done';
  const FILTERS: Filter[] = ['open', 'today', 'done'];

  const today = todayISO();
  let tab = $state<Tab>('customers');
  let filter = $state<Filter>('open');
  let extra = $state<Record<string, OverviewExtra>>({});
  let openTasks = $state<Task[] | null>(null);
  let doneTasks = $state<Task[] | null>(null);
  const closedCustomers = new SvelteSet<number>(); // Standard: Kunden offen
  const openProjects = new SvelteSet<number>(); // Standard: Projekte zu

  $effect(() => {
    refreshSidebar(0);
    api.get<Record<string, OverviewExtra>>('/overview').then(
      (r) => (extra = r),
      () => {}
    );
    api.get<Task[]>('/my-tasks').then(
      (r) => (openTasks = r),
      (e) => toast(errorText(e))
    );
  });
  $effect(() => {
    if (filter === 'done' && doneTasks === null)
      api.get<Task[]>('/my-tasks?done=1').then(
        (r) => (doneTasks = r),
        (e) => toast(errorText(e))
      );
  });

  // ---- Kunden-Ansicht ----------------------------------------------------------
  const isMine = (p: SidebarProject) => !!extra[p.id]?.team.some((m) => m.id === session.user?.id);
  const clOpen = (cl: SidebarChecklist) => cl.total === 0 || cl.done < cl.total;
  const clDueToday = (cl: SidebarChecklist) => !!cl.next_due && cl.next_due <= today;
  function matches(p: SidebarProject): boolean {
    if (filter === 'done') return p.status === 'done' || (p.total > 0 && p.done === p.total);
    if (p.status === 'done') return false;
    if (filter === 'today') return p.checklists.some(clDueToday);
    return p.total === 0 || p.done < p.total || p.checklists.some(clOpen);
  }

  interface Group {
    customer: SidebarCustomer;
    projects: SidebarProject[];
    checklists: number;
  }
  const groups = $derived.by((): Group[] =>
    sidebar.customers
      .map((c) => {
        const projects = c.projects.filter((p) => p.status !== 'archived' && isMine(p) && matches(p));
        return { customer: c, projects, checklists: projects.reduce((s, p) => s + p.checklists.length, 0) };
      })
      .filter((g) => g.projects.length > 0)
  );

  // ---- Persönlich ----------------------------------------------------------------
  let personal = $state<Task[]>([]);
  $effect(() => {
    const src = filter === 'done' ? (doneTasks ?? []) : (openTasks ?? []);
    personal = filter === 'today' ? src.filter((x) => x.due_date && x.due_date <= today) : [...src];
  });
  const personalCount = $derived(
    filter === 'done' ? (doneTasks?.length ?? 0) : filter === 'today' ? (openTasks ?? []).filter((x) => x.due_date && x.due_date <= today).length : (openTasks?.length ?? 0)
  );

  // ---- Darstellung -----------------------------------------------------------------
  const dateLine = new Date().toLocaleDateString(locale(), { weekday: 'long', day: '2-digit', month: '2-digit', year: 'numeric' });
  const shortDate = (iso: string) => new Date(iso + 'T00:00:00').toLocaleDateString(locale(), { day: '2-digit', month: '2-digit' });
  function dueLabel(iso: string): { text: string; cls: string } {
    const d = daysUntil(iso, today);
    if (d < -1) return { text: shortDate(iso), cls: 'font-semibold text-red-600 dark:text-red-400' };
    if (d === -1) return { text: t('due.yesterday'), cls: 'font-semibold text-red-600 dark:text-red-400' };
    if (d === 0) return { text: t('due.today'), cls: 'font-semibold text-accent-600 dark:text-accent-400' };
    if (d === 1) return { text: t('due.tomorrow'), cls: 'text-zinc-600 dark:text-zinc-300' };
    return { text: shortDate(iso), cls: 'text-zinc-500' };
  }
  const tabCls = (on: boolean) =>
    `inline-flex items-center gap-2 rounded-md px-3 py-1.5 text-sm ${on ? 'bg-white font-semibold shadow-xs dark:bg-zinc-700' : 'text-zinc-500 hover:text-zinc-800 dark:hover:text-zinc-200'}`;
  const chipCls = (on: boolean) =>
    `rounded-full px-3 py-1 text-xs font-medium ${on ? 'bg-zinc-900 text-white dark:bg-zinc-100 dark:text-zinc-900' : 'bg-zinc-100 text-zinc-600 hover:bg-zinc-200 dark:bg-zinc-800 dark:text-zinc-300 dark:hover:bg-zinc-700'}`;
  const toggleSet = (s: SvelteSet<number>, id: number) => (s.has(id) ? s.delete(id) : s.add(id));

  // Spalten einer Projekt-/Checklistenzeile: Pfeil | Name | Fortschritt | Fällig | Schalter | Link
  const rowCols = 'grid grid-cols-[1.5rem_minmax(0,1fr)_11rem_4rem_2rem_1.25rem] items-center gap-x-3';
</script>

<svelte:head><title>{t('nav.myTasks')} · ProjectList</title></svelte:head>

<div class="mx-auto max-w-5xl">
  <header class="mb-6 flex items-start justify-between gap-4">
    <div>
      <p class="text-sm text-zinc-500 capitalize">{dateLine}</p>
      <h1 class="mt-1 text-[28px] leading-tight font-bold tracking-tight">{t('nav.myTasks')}</h1>
    </div>
    <SaveStatus />
  </header>

  <div class="mb-5 flex flex-wrap items-center justify-between gap-3">
    <div class="inline-flex gap-1 rounded-lg bg-zinc-100 p-1 dark:bg-zinc-800/70" role="tablist">
      <button type="button" role="tab" aria-selected={tab === 'customers'} class={tabCls(tab === 'customers')} onclick={() => (tab = 'customers')}>
        <Icon name="users" size={15} />{t('tasks.tab.customers')} <span class="text-xs font-normal text-zinc-400 tabular-nums">{groups.length}</span>
      </button>
      <button type="button" role="tab" aria-selected={tab === 'personal'} class={tabCls(tab === 'personal')} onclick={() => (tab = 'personal')}>
        <Icon name="user" size={15} />{t('tasks.tab.personal')} <span class="text-xs font-normal text-zinc-400 tabular-nums">{personalCount}</span>
      </button>
    </div>
    <div class="flex gap-1.5" role="group">
      {#each FILTERS as f (f)}
        <button type="button" class={chipCls(filter === f)} aria-pressed={filter === f} onclick={() => (filter = f)}>{t(`tasks.filter.${f}`)}</button>
      {/each}
    </div>
  </div>

  {#if tab === 'customers'}
    <div class="space-y-4 rounded-2xl bg-zinc-100/70 p-3 dark:bg-zinc-900/50">
      {#each groups as g (g.customer.id)}
        {@const cust = customerProgress(g.customer.projects)}
        {@const open = !closedCustomers.has(g.customer.id)}
        <section class="rounded-xl border border-zinc-200 bg-white px-4 py-3 shadow-xs dark:border-zinc-800 dark:bg-zinc-900">
          <header class="flex items-center gap-3">
            <span class="flex size-9 shrink-0 items-center justify-center rounded-lg text-xs font-bold {customerTone(g.customer.id)}">{customerCode(g.customer.name)}</span>
            <div class="min-w-0 flex-1">
              <a href="/customers/{g.customer.id}" class="block truncate text-[15px] font-semibold hover:underline">{customerLabel(g.customer.name)}</a>
              <p class="text-xs text-zinc-500">{t('tasks.counts', { p: g.projects.length, c: g.checklists })}</p>
            </div>
            <div class="hidden w-40 text-right sm:block">
              <div class="flex items-center justify-end gap-3">
                <span class="h-1.5 w-24 overflow-hidden rounded-full bg-zinc-200 dark:bg-zinc-800">
                  <span class="block h-full rounded-full {barCls(cust.done, cust.total)}" style:width="{pct(cust.done, cust.total)}%"></span>
                </span>
                <span class="w-10 text-sm font-semibold tabular-nums">{pct(cust.done, cust.total)}%</span>
              </div>
              {#if cust.excluded}<p class="mt-0.5 text-[11px] text-zinc-400">{withoutText(cust.excluded)}</p>{/if}
            </div>
            <button
              type="button"
              class="rounded p-1 text-zinc-400 hover:bg-zinc-100 hover:text-zinc-700 dark:hover:bg-zinc-800 dark:hover:text-zinc-200"
              onclick={() => toggleSet(closedCustomers, g.customer.id)}
              aria-expanded={open}
              aria-label="{t(open ? 'overview.hideProjects' : 'overview.showProjects')}: {g.customer.name}">
              <Icon name="chevronDown" size={16} class="transition-transform {open ? '' : '-rotate-90'}" />
            </button>
          </header>

          {#if open}
            <ul class="mt-2 divide-y divide-zinc-100 dark:divide-zinc-800">
              {#each g.projects as p (p.id)}
                {@const pOpen = openProjects.has(p.id)}
                {@const muted = !p.include_in_customer_progress}
                <li class="py-1">
                  <!-- Projektzeile: Klick klappt die Checklisten auf -->
                  <div
                    class="{rowCols} cursor-pointer rounded-lg px-1 py-1.5 hover:bg-zinc-50 dark:hover:bg-zinc-800/50"
                    role="button"
                    tabindex="0"
                    aria-expanded={pOpen}
                    aria-label="{t('tasks.showChecklists')}: {p.name}"
                    onclick={() => toggleSet(openProjects, p.id)}
                    onkeydown={(e) => (e.key === 'Enter' || e.key === ' ') && (e.preventDefault(), toggleSet(openProjects, p.id))}>
                    <span class="flex size-6 items-center justify-center rounded text-zinc-400 {pOpen ? 'bg-zinc-100 dark:bg-zinc-800' : ''}">
                      <Icon name="chevron" size={13} class="transition-transform {pOpen ? 'rotate-90' : ''}" />
                    </span>
                    <span class="flex min-w-0 items-center gap-2">
                      <Icon name="folder" size={15} class="text-zinc-400" />
                      <span class="truncate font-semibold {muted ? 'text-zinc-400 dark:text-zinc-500' : ''}">{p.name}</span>
                      <span class="badge shrink-0 {STATUS_BADGE[p.status]}">{t(`status.${p.status}`)}</span>
                      <span class="hidden shrink-0 text-xs text-zinc-400 md:inline">{t('overview.nChecklists', { n: p.checklists.length })}</span>
                    </span>
                    <span class="flex items-center gap-2.5 {muted ? 'opacity-50 grayscale' : ''}">
                      <span class="h-1.5 flex-1 overflow-hidden rounded-full bg-zinc-200 dark:bg-zinc-800">
                        <span class="block h-full rounded-full {barCls(p.done, p.total)}" style:width="{pct(p.done, p.total)}%"></span>
                      </span>
                      <span class="w-9 text-right text-xs text-zinc-500 tabular-nums">{pct(p.done, p.total)}%</span>
                    </span>
                    <span class="text-right text-xs text-zinc-500 tabular-nums">{p.due_date ? shortDate(p.due_date) : ''}</span>
                    <ProgressToggle project={p} customerName={g.customer.name} />
                    <a href="/projects/{p.id}" class="text-zinc-300 hover:text-zinc-700 dark:text-zinc-600 dark:hover:text-zinc-200" onclick={(e) => e.stopPropagation()} aria-label="{t('tasks.openProject')}: {p.name}" title={t('tasks.openProject')}>
                      <Icon name="chevron" size={14} />
                    </a>
                  </div>

                  {#if pOpen}
                    <ul class="mb-1 ml-9 space-y-0.5">
                      {#each p.checklists as cl (cl.id)}
                        {@const due = cl.next_due && clOpen(cl) ? dueLabel(cl.next_due) : null}
                        <li>
                          <a href="/checklists/{cl.id}" class="{rowCols} rounded-lg px-1 py-1.5 text-sm hover:bg-accent-50/60 dark:hover:bg-accent-950/30">
                            <Icon name="fileText" size={15} class="justify-self-center text-zinc-400" />
                            <span class="truncate">{cl.title}</span>
                            <span class="flex items-center gap-2.5">
                              <span class="h-1.5 flex-1 overflow-hidden rounded-full bg-zinc-200 dark:bg-zinc-800">
                                <span class="block h-full rounded-full {barCls(cl.done, cl.total)}" style:width="{pct(cl.done, cl.total)}%"></span>
                              </span>
                              <span class="w-9 text-right text-xs text-zinc-500 tabular-nums">{cl.done}/{cl.total}</span>
                            </span>
                            <span class="text-right text-xs tabular-nums {due?.cls ?? ''}">{due?.text ?? ''}</span>
                            <span></span>
                            <span class="text-zinc-300 dark:text-zinc-600"><Icon name="chevron" size={14} /></span>
                          </a>
                        </li>
                      {/each}
                    </ul>
                  {/if}
                </li>
              {/each}
            </ul>
          {/if}
        </section>
      {:else}
        <p class="px-2 py-6 text-center text-sm text-zinc-400">{t('tasks.noProjects')}</p>
      {/each}
    </div>
  {:else}
    <div class="rounded-2xl border border-zinc-200 bg-white px-4 py-3 dark:border-zinc-800 dark:bg-zinc-900">
      {#if (filter === 'done' ? doneTasks : openTasks) === null}
        <p class="py-4 text-sm text-zinc-400">{t('common.loading')}</p>
      {:else if personal.length === 0}
        <p class="py-6 text-center text-sm text-zinc-400">{t('tasks.noPersonal')}</p>
      {:else}
        <TaskList bind:tasks={personal} />
      {/if}
    </div>
  {/if}
</div>
