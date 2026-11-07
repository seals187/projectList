<script lang="ts">
  // Customer page: header (code, name, edit, "+ Project"), location / contact /
  // email, notes, and a table of the projects (status with its custom name, labels, progress,
  // due date, person responsible).
  import { page } from '$app/state';
  import { goto } from '$app/navigation';
  import { api, errorText } from '$lib/api';
  import { canWrite, isAdmin, openNewProject, refreshSidebar, sidebar, toast } from '$lib/state.svelte';
  import { fmtDate, shortName, t, todayISO } from '$lib/i18n.svelte';
  import type { Customer, CustomerPhone, Project, ProjectWithProgress, SidebarProject } from '$lib/types';
  import ProjectMenu from '$lib/components/ProjectMenu.svelte';
  import Editable from '$lib/components/Editable.svelte';
  import ConfirmButton from '$lib/components/ConfirmButton.svelte';
  import EntityIcon from '$lib/components/EntityIcon.svelte';
  import FavoriteStar from '$lib/components/FavoriteStar.svelte';
  import SaveStatus from '$lib/components/SaveStatus.svelte';
  import Avatar from '$lib/components/Avatar.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import BackgroundPicker from '$lib/components/BackgroundPicker.svelte';
  import HeaderBand from '$lib/components/HeaderBand.svelte';
  import { barCls, statusBadgeCls, statusBadgeStyle, statusLabel } from '$lib/projectStatus';
  import { dueUrgent, pct } from '$lib/overview';
  import { tagsByIds } from '$lib/tags';
  import { cardHex, projectColor, stripeStyle } from '$lib/cardColors';
  import { myColors } from '$lib/myColors.svelte';
  import { cardColorLabel } from '$lib/cardColorLabel';
  import { setCustomerColors } from '$lib/projectActions';
  import CardColorPicker from '$lib/components/CardColorPicker.svelte';
  import Switch from '$lib/components/Switch.svelte';

  type Field = 'name' | 'notes' | 'location' | 'contact_name' | 'contact_email';

  let customer = $state<Customer | null>(null);
  let projects = $state<ProjectWithProgress[]>([]);
  let notFound = $state(false);
  let newProject = $state('');
  let showArchived = $state(false);
  let editing = $state(false);

  const id = $derived(Number(page.params.id));
  const editable = $derived(canWrite());
  const today = todayISO();
  // labels and person responsible per project come with the sidebar data
  // card colors live in the sidebar store (shared with the overview)
  const sc = $derived(sidebar.customers.find((c) => c.id === id));
  const sidebarProjects = $derived(new Map(sidebar.customers.find((c) => c.id === id)?.projects.map((p) => [p.id, p]) ?? []));

  $effect(() => {
    load(id);
  });

  async function load(id: number) {
    notFound = false;
    editing = false;
    try {
      const r = await api.get<{ customer: Customer; projects: ProjectWithProgress[] }>(`/customers/${id}`);
      customer = r.customer;
      projects = r.projects;
    } catch {
      customer = null;
      notFound = true;
    }
  }

  // right click on a project row: project menu; saved changes reload the table
  let menu = $state<{ project: SidebarProject; x: number; y: number } | null>(null);
  function projectMenu(e: MouseEvent, projectId: number) {
    const sp = sidebarProjects.get(projectId);
    if (!sp || !sc) return; // not in the store (yet): browser menu
    e.preventDefault();
    menu = { project: sp, x: e.clientX, y: e.clientY };
  }
  async function reloadProjects() {
    try {
      projects = (await api.get<{ projects: ProjectWithProgress[] }>(`/customers/${id}`)).projects;
    } catch (err) {
      toast(errorText(err));
    }
  }

  async function save(field: Field, value: string) {
    customer = await api.patch<Customer>(`/customers/${id}`, { [field]: value });
    if (field === 'name' || field === 'location') refreshSidebar(0);
  }

  // phone numbers: edited as a list (label + number) while "Edit" is on, saved on leaving a field
  let phones = $state<CustomerPhone[]>([]);
  $effect(() => {
    if (editing && customer) phones = (customer.phones ?? []).map((p) => ({ ...p }));
  });
  async function savePhones() {
    if (!customer) return;
    const list = phones.filter((p) => p.number.trim() || p.label.trim());
    if (JSON.stringify(list) === JSON.stringify(customer.phones ?? [])) return;
    try {
      customer = await api.patch<Customer>(`/customers/${id}`, { phones: list });
    } catch (err) {
      toast(errorText(err));
    }
  }
  const telHref = (n: string) => `tel:${n.replace(/[^0-9+]/g, '')}`;

  async function addProject(e: SubmitEvent) {
    e.preventDefault();
    if (!newProject.trim()) return;
    try {
      const p = await api.post<Project>(`/customers/${id}/projects`, { name: newProject });
      newProject = '';
      refreshSidebar(0);
      goto(`/projects/${p.id}`);
    } catch (err) {
      toast(errorText(err));
    }
  }

  async function remove() {
    try {
      await api.del(`/customers/${id}`);
      refreshSidebar(0);
      goto('/customers');
    } catch (err) {
      toast(errorText(err));
    }
  }

  const visible = $derived(projects.filter((p) => showArchived || p.status !== 'archived'));
  const archivedCount = $derived(projects.filter((p) => p.status === 'archived').length);

  const FIELDS = [
    ['location', 'customers.location'],
    ['contact_name', 'customers.contact'],
    ['contact_email', 'customers.email']
  ] as const;

  // Columns: project | status | labels | progress | due | person responsible
  const cols = 'grid grid-cols-[minmax(14rem,2fr)_9rem_minmax(12rem,1.6fr)_minmax(9rem,1.1fr)_7.5rem_8.5rem] items-center gap-x-4';

  // background of the customer (also behind its projects unless they have their own)
  let bgOpen = $state(false);
  async function pickBg(value: string) {
    if (!customer) return;
    const sc = sidebar.customers.find((x) => x.id === id);
    const prev = sc?.background ?? '';
    if (sc) sc.background = value;
    try {
      await api.put(`/customers/${id}/background`, { background: value });
    } catch (e) {
      if (sc) sc.background = prev;
      toast(errorText(e));
    }
  }
</script>

<svelte:head><title>{customer?.name ?? ''} · ProjectList</title></svelte:head>

{#if notFound}
  <p class="text-zinc-500">{t('common.notFound')}</p>
{:else if customer}
  {@const c = customer}
  <HeaderBand>
    <!-- Header -->
    <header class="flex flex-wrap items-center justify-between gap-4">
      <div class="flex min-w-0 items-center gap-3">
        <EntityIcon type="customer" size="lg" />
        <h1 class="min-w-0 font-serif text-[28px] in-[.band]:text-xl leading-tight font-bold tracking-tight">
          {#if editing}
            <Editable value={c.name} required onsave={(v) => save('name', v)} inputClass="font-serif text-2xl! font-bold" />
          {:else}
            <span class="block truncate">{c.name}</span>
          {/if}
        </h1>
        <FavoriteStar kind="customer" id={c.id} />
        <SaveStatus />
      </div>
      {#if editable}
        <div class="flex items-center gap-2">
          <button type="button" class="btn-outline px-2.5!" onclick={() => (bgOpen = true)} title={t('bg.title')} aria-label={t('bg.title')}><Icon name="image" size={15} /></button>
          <button type="button" class="btn-outline" aria-pressed={editing} onclick={() => (editing = !editing)}>
            {#if editing}<Icon name="check" size={15} />{t('customer.doneEditing')}{:else}{t('customer.edit')}{/if}
          </button>
          <button type="button" class="btn-primary" onclick={() => openNewProject(c.id)}><Icon name="plus" size={15} /> {t('overview.addProject')}</button>
        </div>
      {/if}
    </header>

    <!-- Location and contact -->
    <dl class="mt-5 mb-3 flex flex-wrap gap-x-12 gap-y-3 in-[.band]:mt-2 in-[.band]:gap-y-1.5 text-sm">
      {#each FIELDS as [field, label] (field)}
        {#if editing || c[field]}
          <div class="min-w-0">
            <dt class="text-[11px] font-semibold tracking-wider text-zinc-400 uppercase">{t(label)}</dt>
            <dd class="mt-0.5 min-w-0">
              {#if editing}
                <Editable value={c[field]} placeholder="–" onsave={(v) => save(field, v)} />
              {:else if field === 'contact_email'}
                <a href="mailto:{c.contact_email}" class="text-accent-600 hover:underline dark:text-accent-400">{c.contact_email}</a>
              {:else}
                {c[field]}
              {/if}
            </dd>
          </div>
        {/if}
      {/each}
      <!-- phone numbers -->
      {#if editing || c.phones?.length}
        <div class="min-w-0" data-testid="customer-phones">
          <dt class="text-[11px] font-semibold tracking-wider text-zinc-400 uppercase">{t('customers.phone')}</dt>
          <dd class="mt-0.5 min-w-0">
            {#if editing}
              <div class="space-y-1.5">
                {#each phones as p, i (i)}
                  <div class="flex items-center gap-1.5">
                    <input class="input w-28! py-0.5! text-[13px]!" maxlength="40" placeholder={t('customers.phoneLabel')} bind:value={p.label} onblur={savePhones} aria-label={t('customers.phoneLabel')} />
                    <input class="input w-44! py-0.5! text-[13px]! tabular-nums" type="tel" maxlength="40" placeholder="+41 41 123 45 67" bind:value={p.number} onblur={savePhones} aria-label={t('customers.phone')} />
                    <button
                      type="button"
                      class="rounded p-1 text-zinc-400 hover:text-red-600"
                      onclick={() => ((phones = phones.filter((_, j) => j !== i)), savePhones())}
                      title={t('customers.phoneRemove')}
                      aria-label={t('customers.phoneRemove')}>
                      <Icon name="x" size={13} />
                    </button>
                  </div>
                {/each}
                {#if phones.length < 10}
                  <button type="button" class="text-[13px] font-medium text-accent-600 hover:underline dark:text-accent-400" onclick={() => (phones = [...phones, { label: '', number: '' }])}>
                    + {t('customers.phoneAdd')}
                  </button>
                {/if}
              </div>
            {:else}
              <ul class="space-y-0.5">
                {#each c.phones as p, i (i)}
                  <li class="flex items-baseline gap-2">
                    {#if p.label}<span class="text-zinc-500 dark:text-zinc-400">{p.label}</span>{/if}
                    <a href={telHref(p.number)} class="text-accent-600 tabular-nums hover:underline dark:text-accent-400">{p.number}</a>
                  </li>
                {/each}
              </ul>
            {/if}
          </dd>
        </div>
      {/if}
      <!-- card color in the overview (customer-wide, optionally in all views) -->
      {#if sc && (editing || sc.color)}
        <div class="min-w-0" data-testid="customer-color">
          <dt class="text-[11px] font-semibold tracking-wider text-zinc-400 uppercase">{t('customer.color')}</dt>
          <dd class="mt-0.5 min-w-0">
            {#if editing}
              <div class="-ml-1 max-w-[17rem]">
                <CardColorPicker label={t('customer.color')} value={sc.color} onpick={(v) => setCustomerColors(sc.id, { color: v })} />
              </div>
              <label class="mt-1 flex items-center gap-2 text-[13px]">
                <Switch label={t('customer.colorEverywhere')} checked={sc.color_everywhere} onchange={(v) => setCustomerColors(sc.id, { everywhere: v })} />
                {t('customer.colorEverywhere')}
              </label>
              <p class="mt-1 max-w-[17rem] text-xs text-zinc-500">{t('customer.colorHint')}</p>
            {:else}
              <span class="flex items-center gap-2">
                <span class="size-3.5 rounded-full" style:background-color={cardHex(sc.color)}></span>
                {cardColorLabel(sc.color)}
                {#if sc.color_everywhere}<span class="text-zinc-500">· {t('customer.colorEverywhere')}</span>{/if}
              </span>
            {/if}
          </dd>
        </div>
      {/if}
    </dl>

    <!-- Notes (always editable inline) -->
    <div class="max-w-3xl text-sm text-zinc-600 dark:text-zinc-300">
      <Editable value={c.notes} multiline placeholder={t('customer.notesPlaceholder')} disabled={!editable} onsave={(v) => save('notes', v)} />
    </div>
  </HeaderBand>

  <!-- Projects -->
  <section class="mt-8">
    <div class="mb-2 flex items-center justify-between">
      <h2 class="section-label flex items-center gap-2">
        {t('customer.projects')}
        <span class="rounded-md bg-zinc-100 px-1.5 text-[11px] font-semibold text-zinc-500 tabular-nums dark:bg-zinc-800">{visible.length}</span>
      </h2>
      {#if archivedCount}
        <button type="button" class="text-xs text-zinc-400 hover:text-zinc-700 dark:hover:text-zinc-200" onclick={() => (showArchived = !showArchived)}>
          {showArchived ? t('customer.hideArchived') : t('customer.showArchived', { n: archivedCount })}
        </button>
      {/if}
    </div>

    <div class="surface overflow-x-auto rounded-xl border border-zinc-200 dark:border-zinc-800">
      <div class="min-w-[56rem]">
        <div class="{cols} border-b border-zinc-200 px-4 py-2.5 text-[11px] font-semibold tracking-wider text-zinc-400 uppercase dark:border-zinc-800">
          <span>{t('customer.col.project')}</span>
          <span>{t('common.status')}</span>
          <span>{t('tags.title')}</span>
          <span>{t('list.col.progress')}</span>
          <span>{t('overview.col.due')}</span>
          <span>{t('project.owner')}</span>
        </div>

        {#each visible as p (p.id)}
          {@const tags = tagsByIds(sidebarProjects.get(p.id)?.tag_ids)}
          {@const sp = sidebarProjects.get(p.id)}
          <a
            href="/projects/{p.id}"
            oncontextmenu={(e) => projectMenu(e, p.id)}
            data-testid="customer-project-row"
            style={stripeStyle(projectColor(p.id, myColors.all))}
            class="{cols} border-b border-zinc-100 px-4 py-3 text-sm hover:bg-zinc-50 dark:border-zinc-800/70 dark:hover:bg-zinc-800/40
              {p.status === 'blocked' ? 'bg-red-50/50 dark:bg-red-950/15' : ''}">
            <span class="flex min-w-0 items-center gap-2.5">
              <EntityIcon type="project" size="sm" />
              <span class="truncate font-semibold {p.status === 'archived' ? 'text-zinc-400 line-through' : ''}" title={p.name}>{p.name}</span>
            </span>
            <span class="min-w-0"><span class="badge inline-block max-w-full truncate {statusBadgeCls(p.status)}" style={statusBadgeStyle(p.status)} title={statusLabel(p.status)}>{statusLabel(p.status)}</span></span>
            <span class="flex min-w-0 items-center gap-2.5 text-xs text-zinc-600 dark:text-zinc-400" title={tags.map((x) => x.name).join(', ')}>
              {#each tags.slice(0, 2) as tag (tag.id)}
                <span class="flex min-w-0 items-center gap-1"><span class="size-2 shrink-0 rounded-full" style:background-color={tag.color}></span><span class="truncate">{tag.name}</span></span>
              {/each}
              {#if tags.length > 2}<span class="shrink-0 text-zinc-400">+{tags.length - 2}</span>{/if}
            </span>
            <span class="flex items-center gap-2">
              {#if p.total}
                <span class="block h-1.5 min-w-0 flex-1 overflow-hidden rounded-full bg-zinc-200 dark:bg-zinc-800">
                  <span class="block h-full rounded-full {barCls(p.done, p.total)}" style:width="{pct(p.done, p.total)}%"></span>
                </span>
                <span class="shrink-0 text-xs text-zinc-500 tabular-nums">{p.done}/{p.total}</span>
              {:else}
                <span class="text-xs text-zinc-400">{t('customer.noChecklist')}</span>
              {/if}
            </span>
            <span class="flex items-center gap-1 text-xs tabular-nums {dueUrgent(p, today) ? 'font-semibold text-red-600 dark:text-red-400' : 'text-zinc-600 dark:text-zinc-300'}">
              {#if p.due_date}<Icon name="calendar" size={12} />{fmtDate(p.due_date)}{:else}<span class="text-zinc-400">{t('customer.noDue')}</span>{/if}
            </span>
            <span class="flex min-w-0 items-center gap-1.5 text-xs text-zinc-600 dark:text-zinc-300">
              {#if sp?.owner_id && sp.owner_name}<Avatar name={sp.owner_name} id={sp.owner_id} size={20} /><span class="truncate" title={sp.owner_name}>{shortName(sp.owner_name)}</span>{:else}<span class="text-zinc-400">–</span>{/if}
            </span>
          </a>
        {:else}
          <p class="border-b border-zinc-100 px-4 py-4 text-sm text-zinc-400 dark:border-zinc-800/70">{t('customer.noProjects')}</p>
        {/each}

        {#if editable}
          <form onsubmit={addProject} class="flex items-center gap-2 px-4 py-2.5">
            <Icon name="plus" size={15} class="text-zinc-400" />
            <input class="flex-1 bg-transparent py-0.5 text-sm outline-none placeholder:text-zinc-400" placeholder={t('customer.newProject')} bind:value={newProject} aria-label={t('customer.newProject')} />
            {#if newProject.trim()}<button class="btn-primary py-1!" type="submit">{t('common.create')}</button>{/if}
          </form>
        {/if}
      </div>
    </div>
  </section>

  {#if isAdmin()}
    <div class="surface mt-10 border-t border-zinc-100 pt-4 in-[.has-bg]:w-fit in-[.has-bg]:rounded-xl in-[.has-bg]:p-1.5 dark:border-zinc-800">
      <ConfirmButton class="btn-ghost text-red-600! hover:bg-red-50! dark:text-red-400! dark:hover:bg-red-950/40!" onconfirm={remove} label={t('customer.delete')} confirmLabel={t('customer.deleteConfirm')}>
        <Icon name="trash" size={15} />{t('customer.delete')}
      </ConfirmButton>
    </div>
  {/if}
{/if}

{#if menu && sc}
  <ProjectMenu project={menu.project} customer={sc} x={menu.x} y={menu.y} onchanged={reloadProjects} onclose={() => (menu = null)} />
{/if}

{#if bgOpen && customer}
  <BackgroundPicker
    title="{t('bg.title')}: {customer.name}"
    value={sidebar.customers.find((x) => x.id === id)?.background ?? ''}
    noneLabel={t('bg.inheritUser')}
    onpick={pickBg}
    onclose={() => (bgOpen = false)} />
{/if}
