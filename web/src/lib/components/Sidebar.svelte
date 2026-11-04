<script lang="ts">
  // Seitenleiste: Suche + Neues Projekt, Navigation, Favoriten, Zuletzt besucht,
  // Kunden → Projekte → Checklisten → Abschnitte, unten Benutzer, Theme, Abmelden.
  import { untrack } from 'svelte';
  import { SvelteSet } from 'svelte/reactivity';
  import { page } from '$app/state';
  import { goto } from '$app/navigation';
  import { api } from '$lib/api';
  import { canWrite, current, openNewProject, palette, session, setTheme, sidebar, theme, toggleSidebar, type Theme } from '$lib/state.svelte';
  import { t } from '$lib/i18n.svelte';
  import { htmlToText } from '$lib/rich';
  import { customerCode, customerLabel, customerTone, STATUS_DOT } from '$lib/projectStatus';
  import { clearRecent, isFavorite, migrateLocalFavorites, rememberVisit, toggleFavorite, type RecentKind } from '$lib/navMemory.svelte';
  import type { Snippet } from 'svelte';
  import type { SidebarChecklist, SidebarCustomer, SidebarProject } from '$lib/types';
  import Icon from './Icon.svelte';
  import Avatar from './Avatar.svelte';

  // Aufgeklappte Kunden pro Browser merken (Standard: zu; der aktive Kunde klappt automatisch auf).
  let expanded = $state<Record<number, boolean>>(loadExpanded());

  function loadExpanded(): Record<number, boolean> {
    try {
      return JSON.parse(localStorage.getItem('sidebar-expanded') ?? '{}');
    } catch {
      return {};
    }
  }

  const isOpen = (id: number) => expanded[id] === true;

  function toggleCustomer(id: number) {
    expanded[id] = !isOpen(id);
    try {
      localStorage.setItem('sidebar-expanded', JSON.stringify(expanded));
    } catch {
      /* ignorieren */
    }
  }

  // Nachschlagetabellen für Favoriten und „Zuletzt“
  const lookup = $derived.by(() => {
    const customers = new Map<number, SidebarCustomer>();
    const projects = new Map<number, { p: SidebarProject; c: SidebarCustomer }>();
    const checklists = new Map<number, { cl: SidebarChecklist; p: SidebarProject }>();
    for (const c of sidebar.customers) {
      customers.set(c.id, c);
      for (const p of c.projects) {
        projects.set(p.id, { p, c });
        for (const cl of p.checklists) checklists.set(cl.id, { cl, p });
      }
    }
    return { customers, projects, checklists };
  });

  // Aktive Elemente aus der URL ableiten.
  const active = $derived.by(() => {
    const m = page.url.pathname.match(/^\/(customers|projects|checklists)\/(\d+)/);
    let customerId: number | null = null,
      projectId: number | null = null,
      checklistId: number | null = null;
    if (m) {
      const id = Number(m[2]);
      if (m[1] === 'customers') customerId = id;
      if (m[1] === 'projects') {
        projectId = id;
        customerId = lookup.projects.get(id)?.c.id ?? null;
      }
      if (m[1] === 'checklists') {
        const x = lookup.checklists.get(id);
        checklistId = id;
        projectId = x?.p.id ?? null;
        customerId = projectId ? (lookup.projects.get(projectId)?.c.id ?? null) : null;
      }
    }
    return { customerId, projectId, checklistId };
  });

  // Beim Wechsel zu einem anderen Kunden (bzw. dessen Projekt/Checkliste) diesen
  // einmal aufklappen. Danach darf man ihn frei zuklappen – auch während man dort ist.
  let lastAutoOpened: number | null = null;
  $effect(() => {
    const cid = active.customerId;
    if (cid === lastAutoOpened) return;
    lastAutoOpened = cid;
    if (cid) untrack(() => !isOpen(cid) && toggleCustomer(cid));
  });

  // Projekte: Checklisten darunter auf-/zuklappen. Beim Öffnen eines Projekts
  // (oder einer seiner Checklisten) klappt es einmal auf; ein weiterer Klick auf
  // das Projekt, während man darauf ist, klappt es wieder zu.
  const openProjects = new SvelteSet<number>();
  let lastAutoProject: number | null = null;
  $effect(() => {
    const pid = active.projectId;
    if (pid === lastAutoProject) return;
    lastAutoProject = pid;
    if (pid) openProjects.add(pid);
  });

  // Klick auf einen Kunden: aufgeklappt → zuklappen (ohne Seitenwechsel);
  // zugeklappt → aufklappen und zum Kunden wechseln.
  function onCustomerClick(e: MouseEvent, id: number) {
    if (e.ctrlKey || e.metaKey || e.shiftKey || e.button !== 0) return;
    if (isOpen(id)) e.preventDefault();
    toggleCustomer(id);
  }

  function onProjectClick(e: MouseEvent, id: number) {
    if (e.ctrlKey || e.metaKey || e.shiftKey || e.button !== 0) return; // neuer Tab usw.
    if (active.projectId === id && !active.checklistId) {
      // schon auf dem Projekt: nur auf-/zuklappen
      e.preventDefault();
      if (openProjects.has(id)) openProjects.delete(id);
      else openProjects.add(id);
    } else {
      openProjects.add(id);
    }
  }

  // Favoriten/Zuletzt: pro Benutzer laden, Besuche merken
  $effect(() => {
    if (sidebar.customers.length) untrack(migrateLocalFavorites); // einmalig alte Browser-Favoriten übernehmen
  });
  const visit = $derived.by((): { kind: RecentKind; id: number } | null => {
    const tpl = page.url.pathname.match(/^\/templates\/(\d+)/);
    if (tpl) return { kind: 'template', id: Number(tpl[1]) };
    if (active.checklistId && lookup.checklists.has(active.checklistId)) return { kind: 'checklist', id: active.checklistId };
    if (page.url.pathname.startsWith('/projects/') && active.projectId) return { kind: 'project', id: active.projectId };
    if (page.url.pathname.startsWith('/customers/') && active.customerId && lookup.customers.has(active.customerId))
      return { kind: 'customer', id: active.customerId };
    return null;
  });
  $effect(() => {
    const v = visit;
    if (v && session.user) untrack(() => rememberVisit(v.kind, v.id)); // liest+schreibt recent → nicht abhängig machen
  });

  const favorites = $derived(sidebar.favorites.map((id) => lookup.projects.get(id)).filter((x) => !!x));
  // „Zuletzt“: Titel/übergeordnetes Element liefert der Server; die aktuelle Seite bleibt (hervorgehoben) drin.
  const RECENT_META = {
    customer: { icon: 'briefcase', path: '/customers/' },
    project: { icon: 'board', path: '/projects/' },
    checklist: { icon: 'tasks', path: '/checklists/' },
    template: { icon: 'copy', path: '/templates/' }
  } as const;
  const recent = $derived(
    sidebar.recent.slice(0, 5).map((r) => ({
      ...r,
      key: `${r.kind}:${r.id}`,
      href: RECENT_META[r.kind].path + r.id,
      icon: RECENT_META[r.kind].icon,
      title: r.kind === 'customer' ? customerLabel(r.title) : r.title,
      sub: [t(`recent.kind.${r.kind}`), r.kind === 'project' ? customerLabel(r.parent) : r.parent].filter(Boolean).join(' · '),
      on: !!visit && visit.kind === r.kind && visit.id === r.id
    }))
  );

  // Abschnitte (Favoriten, Zuletzt, Kunden) einklappbar, pro Browser gemerkt
  type SectionKey = 'favorites' | 'recent' | 'customers';
  let closedSections = $state<Partial<Record<SectionKey, boolean>>>(loadSections());
  function loadSections() {
    try {
      return JSON.parse(localStorage.getItem('sidebar-sections') ?? '{}');
    } catch {
      return {};
    }
  }
  function toggleSection(k: SectionKey) {
    closedSections[k] = !closedSections[k];
    try {
      localStorage.setItem('sidebar-sections', JSON.stringify(closedSections));
    } catch {
      /* ignorieren */
    }
  }

  // Theme per Klick durchschalten: Hell → Dunkel → Dunkelgrau → System
  const THEME_ORDER: Theme[] = ['light', 'dark', 'dim', 'system'];
  const themeIcon = $derived(theme.value === 'light' ? 'sun' : theme.value === 'system' ? 'monitor' : 'moon');
  const nextTheme = () => setTheme(THEME_ORDER[(THEME_ORDER.indexOf(theme.value) + 1) % THEME_ORDER.length]);

  async function logout() {
    try {
      await api.post('/auth/logout');
    } finally {
      session.user = null;
      goto('/login');
    }
  }

  const path = $derived(page.url.pathname);
  const nav = $derived([
    { href: '/', icon: 'home' as const, label: t('nav.overview'), on: path === '/' },
    { href: '/tasks', icon: 'tasks' as const, label: t('nav.myTasks'), on: path === '/tasks', badge: sidebar.myTasks },
    { href: '/customers', icon: 'users' as const, label: t('nav.customers'), on: /^\/(customers|projects|checklists)(\/|$)/.test(path) },
    { href: '/templates', icon: 'template' as const, label: t('nav.templates'), on: path.startsWith('/templates') }
  ]);

  const navCls = (on: boolean) =>
    `flex items-center gap-2.5 rounded-lg px-2.5 py-1.5 ${
      on
        ? 'bg-accent-50 font-semibold text-accent-700 dark:bg-accent-950/60 dark:text-accent-400'
        : 'text-zinc-700 hover:bg-zinc-200/50 dark:text-zinc-300 dark:hover:bg-zinc-800'
    }`;
  const subCls = (on: boolean) =>
    `flex items-center gap-2 rounded-md px-2 py-1 text-[13px] ${
      on
        ? 'bg-accent-50 font-medium text-accent-700 dark:bg-accent-950/60 dark:text-accent-400'
        : 'text-zinc-600 hover:bg-zinc-200/50 dark:text-zinc-400 dark:hover:bg-zinc-800'
    }`;
  const actionCls = 'text-[11px] font-semibold text-accent-600 hover:underline dark:text-accent-400';
  const canCreate = $derived(canWrite() && sidebar.customers.length > 0);
</script>

<nav class="flex h-full flex-col text-sm">
  <!-- Kopf: Logo, Name, Untertitel, Einklappen -->
  <div class="flex items-center gap-2.5 pt-4 pr-3 pb-3 pl-4">
    <a href="/" class="flex min-w-0 flex-1 items-center gap-2.5">
      <img src="/favicon.svg" alt="" width="30" height="30" class="shrink-0" />
      <span class="min-w-0">
        <span class="block truncate text-[15px] leading-tight font-semibold tracking-tight">ProjectList</span>
        {#if session.subtitle}<span class="block truncate text-xs text-zinc-500">{session.subtitle}</span>{/if}
      </span>
    </a>
    <button
      type="button"
      class="hidden rounded-md p-1 text-zinc-400 hover:bg-zinc-200/70 hover:text-zinc-700 md:block dark:hover:bg-zinc-800 dark:hover:text-zinc-200"
      onclick={() => toggleSidebar(true)}
      title="{t('nav.collapseSidebar')} (Ctrl+\)"
      aria-label={t('nav.collapseSidebar')}>
      <Icon name="sidebar" size={16} />
    </button>
  </div>

  <!-- Suche + Neues Projekt -->
  <div class="flex gap-2 px-3">
    <button
      type="button"
      onclick={() => (palette.open = true)}
      class="flex min-w-0 flex-1 items-center gap-2 rounded-lg border border-zinc-200 bg-white px-2.5 py-1.5 text-zinc-400 hover:border-zinc-300 dark:border-zinc-800 dark:bg-zinc-950 dark:hover:border-zinc-700">
      <Icon name="search" size={14} />
      <span class="flex-1 truncate text-left">{t('nav.search')}</span>
      <kbd class="rounded border border-zinc-200 px-1 font-sans text-[10px] dark:border-zinc-700">Ctrl K</kbd>
    </button>
    {#if canCreate}
      <button
        type="button"
        class="flex size-[34px] shrink-0 items-center justify-center rounded-lg bg-accent-600 text-white shadow-xs hover:bg-accent-700"
        onclick={() => openNewProject(active.customerId)}
        title={t('overview.newProject')}
        aria-label={t('overview.newProject')}>
        <Icon name="plus" size={17} />
      </button>
    {/if}
  </div>

  <div class="mt-3 flex-1 overflow-y-auto px-3 pb-4">
    <ul class="space-y-px">
      {#each nav as n (n.href)}
        <li>
          <a href={n.href} class={navCls(n.on)}>
            <Icon name={n.icon} class={n.on ? '' : 'text-zinc-500'} />
            <span class="flex-1">{n.label}</span>
            {#if n.badge}<span class="rounded-md bg-zinc-200/70 px-1.5 text-[11px] text-zinc-600 tabular-nums dark:bg-zinc-800 dark:text-zinc-400">{n.badge}</span>{/if}
          </a>
        </li>
      {/each}
    </ul>

    {#snippet sectionHead(k: SectionKey, label: string, extra?: Snippet)}
      <div class="mt-5 mb-1 flex items-center justify-between pr-2.5 pl-1">
        <button
          type="button"
          class="flex items-center gap-1 rounded py-0.5 text-[11px] font-semibold tracking-wider text-zinc-400 uppercase hover:text-zinc-600 dark:hover:text-zinc-300"
          onclick={() => toggleSection(k)}
          aria-expanded={!closedSections[k]}>
          <Icon name="chevronDown" size={12} class="transition-transform {closedSections[k] ? '-rotate-90' : ''}" />
          {label}
        </button>
        {@render extra?.()}
      </div>
    {/snippet}

    {#if favorites.length}
      {@render sectionHead('favorites', t('nav.favorites'))}
    {/if}
    {#if favorites.length && !closedSections.favorites}
      <ul class="space-y-px">
        {#each favorites as f (f.p.id)}
          <li class="group relative">
            <a href="/projects/{f.p.id}" class="{navCls(active.projectId === f.p.id && !active.checklistId)} pr-8 font-normal!">
              <Icon name="star" size={15} class="fill-amber-400 text-amber-500" />
              <span class="min-w-0 flex-1 truncate">{f.p.name}</span>
              <span class="text-[11px] text-zinc-400 group-hover:invisible">{customerCode(f.c.name)}</span>
            </a>
            <button
              type="button"
              class="absolute top-1/2 right-1.5 hidden -translate-y-1/2 rounded p-1 text-zinc-400 group-hover:block hover:text-zinc-700 dark:hover:text-zinc-200"
              onclick={() => toggleFavorite(f.p.id)}
              title={t('nav.removeFavorite')}
              aria-label="{t('nav.removeFavorite')}: {f.p.name}">
              <Icon name="x" size={13} />
            </button>
          </li>
        {/each}
      </ul>
    {/if}

    {#if recent.length}
      {#snippet clearBtn()}
        <button type="button" class="text-[11px] text-zinc-400 hover:text-zinc-700 dark:hover:text-zinc-200" onclick={clearRecent}>{t('nav.clear')}</button>
      {/snippet}
      {@render sectionHead('recent', t('nav.recent'), clearBtn)}
      {#if !closedSections.recent}
        <ul class="space-y-px">
          {#each recent as r (r.key)}
            <li>
              <a
                href={r.href}
                title={r.title}
                class="flex items-start gap-2.5 rounded-lg px-2.5 py-1.5 {r.on
                  ? 'bg-zinc-200/60 dark:bg-zinc-800'
                  : 'hover:bg-zinc-200/50 dark:hover:bg-zinc-800/60'}">
                <Icon name={r.icon} size={15} class="mt-0.5 text-zinc-500" />
                <span class="min-w-0 flex-1">
                  <span class="block truncate leading-snug {r.on ? 'font-medium text-zinc-900 dark:text-zinc-50' : 'text-zinc-700 dark:text-zinc-300'}">{r.title}</span>
                  <span class="block truncate text-xs leading-snug text-zinc-400">{r.sub}</span>
                </span>
              </a>
            </li>
          {/each}
        </ul>
      {/if}
    {/if}

    {#snippet allLink()}
      <a href="/customers" class={actionCls}>{t('nav.all')}</a>
    {/snippet}
    {@render sectionHead('customers', t('nav.customersSection'), allLink)}
    <ul class="space-y-px {closedSections.customers ? 'hidden' : ''}">
      {#each sidebar.customers as c (c.id)}
        {@const isCustomer = active.customerId === c.id}
        {@const open = isOpen(c.id)}
        {@const projects = c.projects.filter((p) => p.status !== 'archived' || p.id === active.projectId)}
        <li>
          <div class="flex items-center rounded-lg {isCustomer ? 'bg-zinc-200/60 dark:bg-zinc-800' : 'hover:bg-zinc-200/50 dark:hover:bg-zinc-800/60'}">
            <button
              type="button"
              class="ml-0.5 rounded p-1 text-zinc-400 hover:text-zinc-700 dark:hover:text-zinc-200"
              onclick={() => toggleCustomer(c.id)}
              aria-expanded={open}
              aria-label="{t(open ? 'overview.hideProjects' : 'overview.showProjects')}: {c.name}">
              <Icon name="chevron" size={13} class="transition-transform {open ? 'rotate-90' : ''}" />
            </button>
            <a href="/customers/{c.id}" onclick={(e) => onCustomerClick(e, c.id)} class="flex min-w-0 flex-1 items-center gap-2 py-1.5 pr-1" title={c.name}>
              <span class="flex h-5 min-w-5 shrink-0 items-center justify-center rounded px-0.5 text-[9px] font-bold {customerTone(c.id)}">{customerCode(c.name)}</span>
              <span class="truncate {isCustomer ? 'font-semibold text-zinc-900 dark:text-zinc-50' : 'text-zinc-700 dark:text-zinc-300'}">{customerLabel(c.name)}</span>
            </a>
            <span class="pr-2.5 text-xs text-zinc-400 tabular-nums">{projects.length}</span>
          </div>

          {#if open}
            <ul class="my-0.5 ml-[1.05rem] space-y-px border-l border-zinc-200 pl-2 dark:border-zinc-800">
              {#each projects as p (p.id)}
                {@const isProject = active.projectId === p.id}
                {@const fav = isFavorite(p.id)}
                <li>
                  <div class="group relative">
                    <a href="/projects/{p.id}" title={t(`status.${p.status}`)} onclick={(e) => onProjectClick(e, p.id)} aria-expanded={p.checklists.length ? openProjects.has(p.id) : undefined} class="{subCls(isProject && !active.checklistId)} pr-7">
                      <span class="size-1.5 shrink-0 rounded-full {STATUS_DOT[p.status]}" aria-hidden="true"></span>
                      <span class="truncate">{p.name}</span>
                    </a>
                    <button
                      type="button"
                      class="absolute top-1/2 right-1 -translate-y-1/2 rounded p-0.5 {fav ? 'text-amber-500' : 'hidden text-zinc-400 group-hover:block hover:text-amber-500'}"
                      onclick={() => toggleFavorite(p.id)}
                      title={t(fav ? 'nav.removeFavorite' : 'nav.addFavorite')}
                      aria-label="{t(fav ? 'nav.removeFavorite' : 'nav.addFavorite')}: {p.name}"
                      aria-pressed={fav}>
                      <Icon name="star" size={12} class={fav ? 'fill-amber-400' : ''} />
                    </button>
                  </div>
                  {#if openProjects.has(p.id) && p.checklists.length}
                    <ul class="space-y-px pl-3">
                      {#each p.checklists as cl (cl.id)}
                        {@const isChecklist = active.checklistId === cl.id}
                        <li>
                          <a href="/checklists/{cl.id}" class={subCls(isChecklist)}><span class="truncate">{cl.title}</span></a>
                          {#if isChecklist && current.checklistId === cl.id && current.sections.length}
                            <ul class="mt-px mb-1 pl-3">
                              {#each current.sections as s (s.id)}
                                <li>
                                  <a
                                    href="#h-{s.id}"
                                    class="flex items-center gap-2 rounded px-2 py-0.5 text-xs text-zinc-500 hover:text-zinc-800 dark:text-zinc-400 dark:hover:text-zinc-200">
                                    <span class="flex-1 truncate">{htmlToText(s.title)}</span>
                                    <span class="tabular-nums {s.total && s.done === s.total ? 'text-emerald-600' : ''}">{s.done}/{s.total}</span>
                                  </a>
                                </li>
                              {/each}
                            </ul>
                          {/if}
                        </li>
                      {/each}
                    </ul>
                  {/if}
                </li>
              {/each}
              {#if canWrite()}
                <li>
                  <button type="button" class="{subCls(false)} w-full text-zinc-400! hover:text-zinc-700! dark:hover:text-zinc-200!" onclick={() => openNewProject(c.id)}>
                    <Icon name="plus" size={13} />
                    {t('overview.newProject')}
                  </button>
                </li>
              {:else if !projects.length}
                <li class="px-2 py-1 text-xs text-zinc-400">–</li>
              {/if}
            </ul>
          {/if}
        </li>
      {/each}
    </ul>
  </div>

  <!-- Fuß: Benutzer (→ Einstellungen), Theme, Abmelden -->
  <div class="flex items-center gap-1 border-t border-zinc-200 px-3 py-2.5 dark:border-zinc-800">
    <a
      href="/settings"
      class="flex min-w-0 flex-1 items-center gap-2.5 rounded-lg px-1.5 py-1 hover:bg-zinc-200/50 dark:hover:bg-zinc-800 {path.startsWith('/settings') ? 'bg-zinc-200/60 dark:bg-zinc-800' : ''}"
      title={t('nav.settings')}>
      <Avatar name={session.user?.name ?? ''} id={session.user?.id ?? 0} size={30} />
      <span class="truncate font-semibold">{session.user?.name}</span>
    </a>
    <button
      type="button"
      class="rounded-md p-1.5 text-zinc-500 hover:bg-zinc-200/60 hover:text-zinc-800 dark:hover:bg-zinc-800 dark:hover:text-zinc-100"
      onclick={nextTheme}
      title="{t('settings.theme')}: {t(`theme.${theme.value}`)}"
      aria-label="{t('settings.theme')}: {t(`theme.${theme.value}`)}">
      <Icon name={themeIcon} size={16} />
    </button>
    <button
      type="button"
      class="rounded-md p-1.5 text-zinc-500 hover:bg-zinc-200/60 hover:text-zinc-800 dark:hover:bg-zinc-800 dark:hover:text-zinc-100"
      onclick={logout}
      title={t('nav.logout')}
      aria-label={t('nav.logout')}>
      <Icon name="logout" size={16} />
    </button>
  </div>
</nav>
