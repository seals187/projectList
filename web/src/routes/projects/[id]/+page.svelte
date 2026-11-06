<script lang="ts">
  // Project page: header (breadcrumb, status, favorite, actions), progress with
  // context, tabs (checklists, activity, files, notes) and on the right the
  // properties panel and the latest activity.
  import { page } from '$app/state';
  import { goto } from '$app/navigation';
  import { api, errorText } from '$lib/api';
  import { canWrite, refreshSidebar, sidebar, toast } from '$lib/state.svelte';
  import { fmtDate, t, todayISO } from '$lib/i18n.svelte';
  import { reorder, sortable } from '$lib/sortable';
  import type { ActivityEntry, Checklist, ChecklistSummary, Project, ProjectDetail, ProjectStatus } from '$lib/types';
  import { ALL_STATUSES, barCls, statusBadgeCls, statusBadgeStyle, statusLabel } from '$lib/projectStatus';
  import { pct } from '$lib/overview';
  import { toDay } from '$lib/dates';
  import EntityIcon from '$lib/components/EntityIcon.svelte';
  import FavoriteStar from '$lib/components/FavoriteStar.svelte';
  import Editable from '$lib/components/Editable.svelte';
  import ConfirmButton from '$lib/components/ConfirmButton.svelte';
  import Avatar from '$lib/components/Avatar.svelte';
  import SaveStatus from '$lib/components/SaveStatus.svelte';
  import TemplatePicker from '$lib/components/TemplatePicker.svelte';
  import TagPicker from '$lib/components/TagPicker.svelte';
  import AssigneePicker from '$lib/components/AssigneePicker.svelte';
  import WatcherPicker from '$lib/components/WatcherPicker.svelte';
  import ActivityFeed from '$lib/components/ActivityFeed.svelte';
  import ProjectFiles from '$lib/components/ProjectFiles.svelte';
  import ContextMenu from '$lib/components/ContextMenu.svelte';
  import HeaderBand from '$lib/components/HeaderBand.svelte';
  import { pageBg } from '$lib/pageBg.svelte';
  import BackgroundPicker from '$lib/components/BackgroundPicker.svelte';
  import Icon from '$lib/components/Icon.svelte';

  type Tab = 'checklists' | 'activity' | 'files' | 'notes';
  const TABS: Tab[] = ['checklists', 'activity', 'files', 'notes'];

  let data = $state<ProjectDetail | null>(null);
  let notFound = $state(false);
  let mode = $state<'none' | 'blank' | 'template'>('none');
  let newTitle = $state('');
  let tab = $state<Tab>('checklists');
  let activity = $state<ActivityEntry[] | null>(null);
  let more = $state<{ x: number; y: number } | null>(null);
  let dates = $state<{ x: number; y: number } | null>(null);
  let activityOpen = $state(false); // latest activity: collapsed = only the newest entry

  const id = $derived(Number(page.params.id));
  const editable = $derived(canWrite());
  const today = todayISO();

  $effect(() => {
    load(id);
  });
  // tab from the URL hash (#activity, #files, …) so it can be linked
  $effect(() => {
    const h = page.url.hash.slice(1) as Tab;
    tab = TABS.includes(h) ? h : 'checklists';
  });
  function setTab(next: Tab) {
    tab = next;
    history.replaceState(history.state, '', next === 'checklists' ? location.pathname : `#${next}`);
  }

  async function load(id: number) {
    notFound = false;
    mode = 'none';
    activity = null;
    try {
      data = await api.get<ProjectDetail>(`/projects/${id}`);
      loadActivity();
    } catch {
      data = null;
      notFound = true;
    }
  }
  async function loadActivity() {
    try {
      activity = await api.get<ActivityEntry[]>(`/projects/${id}/activity?limit=200`);
    } catch {
      activity = [];
    }
  }

  async function save(body: Partial<Project>) {
    if (!data) return;
    try {
      data.project = await api.patch<Project>(`/projects/${id}`, body);
      refreshSidebar(0);
      loadActivity();
    } catch (e) {
      toast(errorText(e));
    }
  }

  // ---- Progress with context --------------------------------------------------------
  const done = $derived(data?.checklists.reduce((s, c) => s + c.done, 0) ?? 0);
  const total = $derived(data?.checklists.reduce((s, c) => s + c.total, 0) ?? 0);
  const daysLeft = $derived(data?.project.due_date ? toDay(data.project.due_date) - toDay(today) : null);
  const dueText = $derived(
    daysLeft === null ? '' : daysLeft > 1 ? t('project.daysLeft', { n: daysLeft }) : daysLeft === 1 ? t('project.dueTomorrow') : daysLeft === 0 ? t('project.dueToday') : t('project.overdue', { n: -daysLeft })
  );

  // ---- Checklists --------------------------------------------------------------------
  function summary(c: Checklist): ChecklistSummary {
    const due = c.items.reduce((m, i) => (i.kind === 'item' && i.due_date && i.due_date > m ? i.due_date : m), '');
    return { id: c.id, project_id: c.project_id, title: c.title, template_id: c.template_id, position: c.position, total: c.total, done: c.done, due_date: due, assignee_id: c.assignee_id, assignee_name: c.assignee_name ?? '' };
  }

  async function addChecklist(e: SubmitEvent) {
    e.preventDefault();
    if (!data || !newTitle.trim()) return;
    try {
      const c = await api.post<Checklist>(`/projects/${id}/checklists`, { title: newTitle });
      refreshSidebar(0);
      goto(`/checklists/${c.id}`);
    } catch (err) {
      toast(errorText(err));
    }
  }

  function created(c: Checklist) {
    data?.checklists.push(summary(c));
    mode = 'none';
    refreshSidebar(0);
    goto(`/checklists/${c.id}`);
  }

  function newChecklist() {
    setTab('checklists');
    mode = 'blank';
  }

  async function sortChecklists(ids: number[]) {
    if (!data) return;
    const prev = data.checklists;
    data.checklists = reorder(prev, ids);
    try {
      await api.put(`/projects/${id}/checklists/order`, { ids });
      refreshSidebar(0);
    } catch (e) {
      data.checklists = prev;
      toast(errorText(e));
    }
  }

  // assign / delete a checklist directly in the list (shown immediately, reverted on error)
  async function assignChecklist(clId: number, assigneeId: number | null) {
    const cl = data?.checklists.find((c) => c.id === clId);
    if (!cl || !data) return;
    const prev = { id: cl.assignee_id, name: cl.assignee_name };
    cl.assignee_id = assigneeId;
    cl.assignee_name = data.assignees.find((a) => a.id === assigneeId)?.name ?? '';
    try {
      await api.patch(`/checklists/${clId}`, { assignee_id: assigneeId });
      refreshSidebar(0);
    } catch (e) {
      cl.assignee_id = prev.id;
      cl.assignee_name = prev.name;
      toast(errorText(e));
    }
  }

  async function deleteChecklist(clId: number) {
    if (!data) return;
    try {
      await api.del(`/checklists/${clId}`);
      data.checklists = data.checklists.filter((c) => c.id !== clId);
      refreshSidebar(0);
      loadActivity();
    } catch (e) {
      toast(errorText(e));
    }
  }

  // ---- Actions -----------------------------------------------------------------------
  async function duplicate() {
    if (!data) return;
    try {
      const p = await api.post<Project>(`/projects/${id}/duplicate`, { name: `${data.project.name} (${t('project.copySuffix')})` });
      refreshSidebar(0);
      toast(t('project.duplicated'), 'info');
      goto(`/projects/${p.id}`);
    } catch (e) {
      toast(errorText(e));
    }
  }

  const archived = $derived(data?.project.status === 'archived');
  const toggleArchive = () => save({ status: archived ? 'active' : 'archived' });

  async function remove() {
    if (!data) return;
    const customerId = data.customer.id;
    try {
      await api.del(`/projects/${id}`);
      refreshSidebar(0);
      goto(`/customers/${customerId}`);
    } catch (e) {
      toast(errorText(e));
    }
  }

  // background of the project (also behind its checklists)
  let bgOpen = $state(false);
  const projectBg = $derived(storeProjectBg(id));
  function storeProjectBg(pid: number) {
    for (const c of sidebar.customers) for (const p of c.projects) if (p.id === pid) return p.background;
    return '';
  }
  async function pickBg(value: string) {
    const sp = sidebar.customers.flatMap((c) => c.projects).find((p) => p.id === id);
    const prev = sp?.background ?? '';
    if (sp) sp.background = value;
    try {
      await api.put(`/projects/${id}/background`, { background: value });
    } catch (e) {
      if (sp) sp.background = prev;
      toast(errorText(e));
    }
  }

  async function setWatchers(ids: number[]) {
    if (!data) return;
    try {
      data.watchers = await api.put<{ id: number; name: string }[]>(`/projects/${id}/watchers`, { user_ids: ids });
    } catch (e) {
      toast(errorText(e));
    }
  }

  const tabCls = (on: boolean) =>
    `-mb-px border-b-2 px-1 pb-2.5 text-sm in-[.band]:pb-1.5 transition-colors ${on ? 'border-accent-600 font-semibold text-accent-700 dark:border-accent-400 dark:text-accent-400' : 'border-transparent text-zinc-500 hover:text-zinc-800 dark:hover:text-zinc-200'}`;
  const iconBtn =
    'flex size-9 items-center justify-center rounded-lg border border-zinc-200 bg-white text-zinc-500 shadow-xs hover:bg-zinc-50 hover:text-zinc-800 dark:border-zinc-700 dark:bg-zinc-900 dark:hover:bg-zinc-800 dark:hover:text-zinc-200';
  const propLabel = 'flex items-center gap-2 text-zinc-500';
</script>

<svelte:head><title>{data?.project.name ?? ''} · ProjectList</title></svelte:head>

{#if notFound}
  <p class="text-zinc-500">{t('common.notFound')}</p>
{:else if data}
  {@const d = data}
  {#snippet tabBar()}
    <div class="mb-4 flex gap-6 border-b border-zinc-200 in-[.band]:mt-1.5 in-[.band]:border-b-0 dark:border-zinc-800" role="tablist">
      {#each TABS as tb (tb)}
        <button type="button" role="tab" aria-selected={tab === tb} class={tabCls(tab === tb)} onclick={() => setTab(tb)}>
          {t(`project.tab.${tb}`)}
          {#if tb === 'checklists'}<span class="ml-0.5 text-zinc-400 tabular-nums">{d.checklists.length}</span>{/if}
          {#if tb === 'files' && d.files_count}<span class="ml-0.5 text-zinc-400 tabular-nums">{d.files_count}</span>{/if}
        </button>
      {/each}
    </div>
  {/snippet}
  <!-- Header: breadcrumb, title with status and favorite, actions -->
  <HeaderBand>
    <header class="flex flex-wrap items-start justify-between gap-4">
      <div class="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1">
        <EntityIcon type="project" size="lg" />
        <h1 class="min-w-0 font-serif text-[28px] in-[.band]:text-xl leading-tight font-bold tracking-tight">
          <Editable value={d.project.name} required disabled={!editable} onsave={(name) => save({ name })} inputClass="font-serif text-2xl! font-bold" />
        </h1>
        <span class="badge {statusBadgeCls(d.project.status)}" style={statusBadgeStyle(d.project.status)}>{statusLabel(d.project.status)}</span>
        <FavoriteStar kind="project" {id} />
        <SaveStatus />
      </div>
      {#if editable}
        <div class="flex items-center gap-2">
          <button type="button" class={iconBtn} onclick={duplicate} title={t('project.duplicate')} aria-label={t('project.duplicate')}><Icon name="copy" size={16} /></button>
          <button type="button" class={iconBtn} onclick={toggleArchive} title={t(archived ? 'project.unarchive' : 'project.archive')} aria-label={t(archived ? 'project.unarchive' : 'project.archive')}>
            <Icon name={archived ? 'undo' : 'archive'} size={16} />
          </button>
          <button
            type="button"
            class={iconBtn}
            onclick={(e) => {
              const r = e.currentTarget.getBoundingClientRect();
              more = more ? null : { x: r.right - 288, y: r.bottom + 4 };
            }}
            title={t('menu.actions')}
            aria-label={t('menu.actions')}>
            <Icon name="dots" size={16} />
          </button>
          <button type="button" class="btn-primary" onclick={newChecklist}><Icon name="plus" size={15} /> {t('project.checklistShort')}</button>
        </div>
      {/if}
    </header>

    <!-- Progress with context -->
    <div class="mt-3 mb-6 flex in-[.band]:mt-1 in-[.band]:mb-0 flex-wrap items-center gap-x-4 gap-y-1">
      <span class="block h-1.5 w-56 overflow-hidden rounded-full bg-zinc-200 dark:bg-zinc-800">
        <span class="block h-full rounded-full {barCls(done, total)}" style:width="{pct(done, total)}%"></span>
      </span>
      <span class="text-sm text-zinc-500">
        <span class="font-medium text-zinc-700 tabular-nums dark:text-zinc-200">{pct(done, total)}%</span>
        · {t('project.itemsOf', { done, total })}
        {#if dueText}· <span class={daysLeft !== null && daysLeft < 0 ? 'font-semibold text-red-600 dark:text-red-400' : ''}>{dueText}</span>{/if}
      </span>
    </div>
    <!-- with a background the tabs sit on the header band (like Planka) -->
    {#if pageBg.active}{@render tabBar()}{/if}
  </HeaderBand>

  <div class="grid gap-6 lg:grid-cols-[minmax(0,1fr)_20rem]">
    <!-- Tabs -->
    <div class="min-w-0">
      {#if !pageBg.active}{@render tabBar()}{/if}

      {#if tab === 'checklists'}
        <div class="surface overflow-hidden rounded-xl border border-zinc-200 dark:border-zinc-800">
          <ul use:sortable={{ onsort: sortChecklists, disabled: !editable }} class="divide-y divide-zinc-100 dark:divide-zinc-800">
            {#each d.checklists as cl (cl.id)}
              {@const complete = cl.total > 0 && cl.done === cl.total}
              <li data-id={cl.id} class="group relative flex items-center bg-white dark:bg-zinc-900 [&.dragging]:shadow-lg">
                {#if editable}
                  <button type="button" data-handle class="cursor-grab touch-none py-3 pr-1 pl-2.5 text-zinc-500 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-zinc-100" aria-label={t('checklist.drag')}>
                    <Icon name="grip" size={14} />
                  </button>
                {/if}
                <a href="/checklists/{cl.id}" class="flex min-w-0 flex-1 items-center gap-3 py-3 pr-4 {editable ? 'pl-1' : 'pl-4'} hover:bg-zinc-50 dark:hover:bg-zinc-900/60">
                  <EntityIcon type="checklist" size="sm" />
                  <span class="min-w-0 flex-1 truncate font-medium">{cl.title}</span>
                  {#if cl.due_date && !complete}
                    <span class="hidden text-xs tabular-nums sm:inline {cl.due_date < today ? 'text-red-600' : 'text-zinc-400'}">{t('checklist.due', { date: fmtDate(cl.due_date) })}</span>
                  {/if}
                  <span class="hidden h-1.5 w-28 overflow-hidden rounded-full bg-zinc-200 sm:block dark:bg-zinc-800">
                    <span class="block h-full rounded-full {barCls(cl.done, cl.total)}" style:width="{pct(cl.done, cl.total)}%"></span>
                  </span>
                  <span class="w-12 text-right text-xs text-zinc-500 tabular-nums">{cl.done}/{cl.total}</span>
                  <span class="w-10 text-right text-xs text-zinc-400">{complete ? t('project.checklistDone') : ''}</span>
                </a>
                <!-- outside the link: assign and delete without opening the checklist -->
                <span class="flex w-40 shrink-0 items-center justify-start pl-2">
                  {#if editable}
                    <span class="max-w-full min-w-0 [&_label]:max-w-full">
                      <AssigneePicker
                        assignees={d.assignees}
                        value={cl.assignee_id}
                        name={cl.assignee_name || null}
                        compact
                        onchange={(a) => assignChecklist(cl.id, a)} />
                    </span>
                  {:else if cl.assignee_id && cl.assignee_name}
                    <span class="flex min-w-0 items-center gap-1.5 text-[13px] text-zinc-600 dark:text-zinc-300" title={t('checklist.assignedTo', { name: cl.assignee_name })}>
                      <Avatar name={cl.assignee_name} id={cl.assignee_id} size={18} /><span class="truncate">{cl.assignee_name}</span>
                    </span>
                  {/if}
                </span>
                {#if editable}
                  <span class="min-w-9 shrink-0 pr-2 text-right whitespace-nowrap [@media(hover:hover)]:opacity-0 group-hover:opacity-100 focus-within:opacity-100">
                    <ConfirmButton class="btn-ghost px-1.5! text-zinc-400 hover:text-red-600" onconfirm={() => deleteChecklist(cl.id)} confirmLabel={t('checklist.deleteConfirm')} label={t('checklist.delete')}>
                      <Icon name="trash" size={14} />
                    </ConfirmButton>
                  </span>
                {/if}
              </li>
            {/each}
          </ul>
          {#if !d.checklists.length && !editable}
            <p class="px-4 py-4 text-sm text-zinc-400">{t('project.noChecklists')}</p>
          {/if}
          {#if editable}
            <div class="border-t border-zinc-100 px-4 py-2.5 dark:border-zinc-800 {d.checklists.length ? '' : 'border-t-0'}">
              {#if mode === 'blank'}
                <form onsubmit={addChecklist} class="flex gap-2">
                  <!-- svelte-ignore a11y_autofocus -->
                  <input class="input" placeholder={t('project.checklistTitle')} bind:value={newTitle} autofocus onkeydown={(e) => e.key === 'Escape' && (mode = 'none')} />
                  <button class="btn-primary" type="submit">{t('common.create')}</button>
                  <button class="btn-ghost" type="button" onclick={() => (mode = 'none')}>{t('common.cancel')}</button>
                </form>
              {:else if mode === 'template'}
                <TemplatePicker projectId={id} oncreated={created} oncancel={() => (mode = 'none')} />
              {:else}
                <p class="flex items-center gap-1.5 text-sm text-zinc-400">
                  <Icon name="plus" size={14} />
                  <button type="button" class="hover:text-zinc-700 dark:hover:text-zinc-200" onclick={() => (mode = 'blank')}>{t('project.addChecklist')}</button>
                  {t('project.or')}
                  <button type="button" class="hover:text-zinc-700 dark:hover:text-zinc-200" onclick={() => (mode = 'template')}>{t('project.fromTemplateLower')}</button>
                </p>
              {/if}
            </div>
          {/if}
        </div>
      {:else if tab === 'activity'}
        <ActivityFeed entries={activity} />
      {:else if tab === 'files'}
        <ProjectFiles
          projectId={id}
          {editable}
          onchange={(n) => {
            if (data && data.files_count !== n) {
              data.files_count = n;
              loadActivity();
            }
          }} />
      {:else}
        <div class="surface max-w-2xl rounded-xl border border-zinc-200 p-4 text-sm text-zinc-700 dark:border-zinc-800 dark:text-zinc-300">
          <Editable value={d.project.description} multiline placeholder={t('project.descPlaceholder')} disabled={!editable} onsave={(description) => save({ description })} />
        </div>
      {/if}
    </div>

    <!-- Properties and latest activity -->
    <aside class="space-y-4">
      <section class="surface rounded-xl border border-zinc-200 p-4 dark:border-zinc-800">
        <h2 class="mb-3 text-sm font-semibold">{t('project.properties')}</h2>
        <dl class="grid grid-cols-[7.5rem_minmax(0,1fr)] items-center gap-x-2 gap-y-2.5 text-sm">
          <dt class={propLabel}><Icon name="flag" size={14} />{t('common.status')}</dt>
          <dd>
            <select
              class="badge cursor-pointer appearance-none border-0 pr-2 outline-none focus:ring-2 focus:ring-accent-500/30 {statusBadgeCls(d.project.status)}"
              style={statusBadgeStyle(d.project.status)}
              value={d.project.status}
              disabled={!editable}
              onchange={(e) => save({ status: e.currentTarget.value as ProjectStatus })}
              aria-label={t('common.status')}>
              {#each ALL_STATUSES as s (s)}<option value={s}>{statusLabel(s)}</option>{/each}
            </select>
          </dd>

          <dt class={propLabel}><Icon name="users" size={14} />{t('project.customer')}</dt>
          <dd class="flex min-w-0 items-center gap-1.5">
            <EntityIcon type="customer" size="xs" />
            <a href="/customers/{d.customer.id}" class="truncate hover:underline">{d.customer.name}</a>
          </dd>

          <dt class={propLabel}><Icon name="user" size={14} />{t('project.owner')}</dt>
          <dd class="min-w-0">
            <AssigneePicker
              assignees={d.assignees}
              value={d.project.owner_id}
              name={d.assignees.find((a) => a.id === d.project.owner_id)?.name ?? null}
              disabled={!editable}
              label={t('project.owner')}
              onchange={(owner_id) => save({ owner_id })} />
          </dd>

          <dt class={propLabel}><Icon name="calendar" size={14} />{t('project.startEnd')}</dt>
          <dd class="min-w-0">
            <button
              type="button"
              class="-mx-1 rounded-md px-1 py-0.5 text-left tabular-nums {editable ? 'hover:bg-zinc-100 dark:hover:bg-zinc-800' : 'cursor-default'}"
              disabled={!editable}
              onclick={(e) => {
                const r = e.currentTarget.getBoundingClientRect();
                dates = dates ? null : { x: r.left, y: r.bottom + 4 };
              }}>
              {#if d.project.start_date || d.project.due_date}
                {d.project.start_date ? fmtDate(d.project.start_date) : '…'}
                <span class="text-zinc-400">→</span>
                <span class={daysLeft !== null && daysLeft < 0 && d.project.status !== 'done' ? 'font-semibold text-red-600 dark:text-red-400' : ''}>{d.project.due_date ? fmtDate(d.project.due_date) : '…'}</span>
              {:else}
                <span class="text-zinc-400">{editable ? t('project.setDates') : '–'}</span>
              {/if}
            </button>
          </dd>

          <dt class="{propLabel} self-start pt-0.5"><Icon name="tag" size={14} />{t('tags.title')}</dt>
          <dd class="min-w-0">
            <TagPicker projectId={id} bind:ids={d.tag_ids} disabled={!editable} />
          </dd>

          <dt class={propLabel}><Icon name="eye" size={14} />{t('watchers.title')}</dt>
          <dd class="min-w-0"><WatcherPicker people={d.assignees} value={d.watchers} disabled={!editable} onchange={setWatchers} /></dd>

        </dl>
      </section>

      <!-- latest activity: only the newest entry, more when expanded -->
      <section class="surface rounded-xl border border-zinc-200 p-4 dark:border-zinc-800">
        <div class="mb-3 flex items-center justify-between gap-2">
          <button
            type="button"
            class="-ml-1 flex items-center gap-1 rounded px-1 text-sm font-semibold hover:text-zinc-600 dark:hover:text-zinc-300"
            onclick={() => (activityOpen = !activityOpen)}
            aria-expanded={activityOpen}
            data-testid="activity-toggle">
            <Icon name="chevronDown" size={14} class="text-zinc-400 transition-transform {activityOpen ? '' : '-rotate-90'}" />
            {t('activity.latest')}
          </button>
          {#if activity?.length}
            <button type="button" class="text-xs text-accent-600 hover:underline dark:text-accent-400" onclick={() => setTab('activity')}>{t('nav.all')}</button>
          {/if}
        </div>
        <ActivityFeed entries={activity} limit={activityOpen ? 5 : 1} compact />
      </section>
    </aside>
  </div>

  {#if dates}
    <ContextMenu x={dates.x} y={dates.y} label={t('project.startEnd')} onclose={() => (dates = null)}>
      <div class="space-y-2 p-1.5 text-sm">
        {#each [{ key: 'start_date', label: t('project.startDate'), value: d.project.start_date, min: undefined, max: d.project.due_date }, { key: 'due_date', label: t('project.dueDate'), value: d.project.due_date, min: d.project.start_date, max: undefined }] as const as f (f.key)}
          <label class="flex items-center gap-2">
            <span class="w-20 shrink-0 text-zinc-500">{f.label}</span>
            <input type="date" class="input py-1! text-sm!" value={f.value ?? ''} min={f.min ?? undefined} max={f.max ?? undefined} onchange={(e) => save({ [f.key]: e.currentTarget.value || null })} />
            {#if f.value}
              <button type="button" class="btn-ghost px-1.5!" onclick={() => save({ [f.key]: null })} title={t('menu.clearDate')} aria-label="{t('menu.clearDate')}: {f.label}"><Icon name="x" size={14} /></button>
            {/if}
          </label>
        {/each}
      </div>
    </ContextMenu>
  {/if}

  {#if more}
    <ContextMenu x={more.x} y={more.y} label={t('menu.actions')} onclose={() => (more = null)}>
      <button type="button" class="flex w-full items-center gap-2.5 rounded-md px-2 py-1.5 text-left hover:bg-zinc-100 dark:hover:bg-zinc-800" onclick={() => ((more = null), setTab('notes'))}>
        <Icon name="fileText" size={15} class="text-zinc-400" />{t('project.editNotes')}
      </button>
      <button type="button" class="flex w-full items-center gap-2.5 rounded-md px-2 py-1.5 text-left hover:bg-zinc-100 dark:hover:bg-zinc-800" onclick={() => ((more = null), setTab('files'))}>
        <Icon name="download" size={15} class="rotate-180 text-zinc-400" />{t('files.upload')}
      </button>
      <button type="button" class="flex w-full items-center gap-2.5 rounded-md px-2 py-1.5 text-left hover:bg-zinc-100 dark:hover:bg-zinc-800" onclick={() => ((more = null), (bgOpen = true))}>
        <Icon name="image" size={15} class="text-zinc-400" />{t('bg.title')}
      </button>
      <div class="my-1 border-t border-zinc-100 dark:border-zinc-800"></div>
      <div class="px-1 py-0.5">
        <ConfirmButton onconfirm={remove} label={t('project.delete')} confirmLabel={t('project.deleteConfirm')} />
      </div>
    </ContextMenu>
  {/if}
{/if}

{#if bgOpen && data}
  <BackgroundPicker title="{t('bg.title')}: {data.project.name}" value={projectBg} noneLabel={t('bg.inheritCustomer')} onpick={pickBg} onclose={() => (bgOpen = false)} />
{/if}
