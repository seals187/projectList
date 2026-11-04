<script lang="ts">
  // Context menu of a project (overview board): open, checklists, rename, status,
  // due date, person responsible, tags, favorite. Sections expand inline.
  import type { ComponentProps } from 'svelte';
  import { goto } from '$app/navigation';
  import { api, errorText } from '$lib/api';
  import { fmtDate, t } from '$lib/i18n.svelte';
  import { canWrite, refreshSidebar, toast } from '$lib/state.svelte';
  import { ALL_STATUSES, customerCode, customerTone, statusBadgeCls, statusBadgeStyle, statusDotCls, statusDotStyle, statusLabel } from '$lib/projectStatus';
  import { customerAssignees, patchProject, setProjectTags, storeProject } from '$lib/projectActions';
  import { isFavorite, toggleFavorite } from '$lib/navMemory.svelte';
  import { tagsByIds } from '$lib/tags';
  import type { Assignee, Checklist, ProjectStatus, SidebarCustomer, SidebarProject } from '$lib/types';
  import ContextMenu from './ContextMenu.svelte';
  import TagMenu from './TagMenu.svelte';
  import Avatar from './Avatar.svelte';
  import Icon from './Icon.svelte';

  let {
    project,
    customer,
    x,
    y,
    onclose,
    initial = null
  }: { project: SidebarProject; customer: SidebarCustomer; x: number; y: number; onclose: () => void; initial?: Section | null } = $props();

  type IconName = ComponentProps<typeof Icon>['name'];
  type Section = 'checklists' | 'rename' | 'status' | 'due' | 'owner' | 'tags';
  // svelte-ignore state_referenced_locally
  let section = $state<Section | null>(initial); // quick actions on the card open a section directly
  const toggle = (s: Section) => (section = section === s ? null : s);

  // always show the current state from the store (the overview passes a copy)
  const p = $derived(storeProject(project.id) ?? project);
  const editable = canWrite();
  const fav = $derived(isFavorite(project.id));

  let name = $state('');
  let newChecklist = $state('');
  let assignees = $state<Assignee[] | null>(null);

  $effect(() => {
    if (section === 'rename') name = p.name;
    if (section === 'owner' && !assignees) customerAssignees(customer.id).then((a) => (assignees = a), (e) => toast(errorText(e)));
  });

  function open(href: string) {
    onclose();
    goto(href);
  }

  async function rename(e: SubmitEvent) {
    e.preventDefault();
    if (name.trim() && name.trim() !== p.name && (await patchProject(p.id, { name: name.trim() }))) section = null;
  }

  const setStatus = (status: ProjectStatus) => patchProject(p.id, { status });
  const setDate = (key: 'start_date' | 'due_date', v: string) => patchProject(p.id, { [key]: v || null });
  const dateRange = $derived(
    p.start_date && p.due_date ? `${fmtDate(p.start_date)} – ${fmtDate(p.due_date)}` : p.due_date ? fmtDate(p.due_date) : p.start_date ? `${fmtDate(p.start_date)} –` : ''
  );
  const setOwner = (a: Assignee | null) => patchProject(p.id, { owner_id: a?.id ?? null }, a?.name);

  async function addChecklist(e: SubmitEvent) {
    e.preventDefault();
    if (!newChecklist.trim()) return;
    try {
      const c = await api.post<Checklist>(`/projects/${p.id}/checklists`, { title: newChecklist.trim() });
      refreshSidebar(0);
      open(`/checklists/${c.id}`);
    } catch (err) {
      toast(errorText(err));
    }
  }

  const itemCls = 'flex w-full items-center gap-2.5 rounded-md px-2 py-1.5 text-left hover:bg-zinc-100 dark:hover:bg-zinc-800';
  const valueCls = 'ml-auto max-w-[45%] truncate text-xs text-zinc-400';
  const subCls = 'mx-1 mb-1.5 rounded-lg bg-zinc-50 p-1.5 dark:bg-zinc-800/50';
</script>

{#snippet item(s: Section, icon: IconName, label: string, value = '')}
  <button type="button" class={itemCls} role="menuitem" aria-expanded={section === s} onclick={() => toggle(s)}>
    <Icon name={icon} size={15} class="shrink-0 text-zinc-400" />
    <span>{label}</span>
    <span class={valueCls}>{value}</span>
    <Icon name="chevron" size={13} class="shrink-0 text-zinc-400 transition-transform {section === s ? 'rotate-90' : ''} {value ? '' : 'ml-auto'}" />
  </button>
{/snippet}

<ContextMenu {x} {y} {onclose} label={p.name}>
  <div class="mb-1 flex items-start gap-2.5 border-b border-zinc-100 px-2 pt-1 pb-2.5 dark:border-zinc-800">
    <span class="mt-0.5 flex size-6 shrink-0 items-center justify-center rounded text-[9px] font-bold {customerTone(customer.id)}">{customerCode(customer.name)}</span>
    <div class="min-w-0 flex-1">
      <p class="truncate font-semibold">{p.name}</p>
      <p class="truncate text-xs text-zinc-500">{customer.name}</p>
    </div>
    <span class="badge shrink-0 {statusBadgeCls(p.status)}" style={statusBadgeStyle(p.status)}>{statusLabel(p.status)}</span>
  </div>

  <button type="button" class={itemCls} role="menuitem" onclick={() => open(`/projects/${p.id}`)}>
    <Icon name="external" size={15} class="text-zinc-400" />{t('menu.openProject')}
  </button>

  {@render item('checklists', 'checkSquare', t('project.checklists'), String(p.checklists.length))}
  {#if section === 'checklists'}
    <div class={subCls}>
      {#each p.checklists as cl (cl.id)}
        <button type="button" class="{itemCls} text-[13px]" onclick={() => open(`/checklists/${cl.id}`)}>
          <span class="min-w-0 flex-1 truncate">{cl.title}</span>
          <span class="shrink-0 text-xs text-zinc-400 tabular-nums">{cl.done}/{cl.total}</span>
        </button>
      {:else}
        <p class="px-2 py-1 text-[13px] text-zinc-400">{t('project.noChecklists')}</p>
      {/each}
      {#if editable}
        <form onsubmit={addChecklist} class="mt-1 flex gap-1.5">
          <input class="input py-1! text-[13px]!" placeholder={t('menu.newChecklist')} bind:value={newChecklist} aria-label={t('menu.newChecklist')} />
          <button type="submit" class="btn-primary px-2! py-1!" disabled={!newChecklist.trim()} aria-label={t('common.create')}><Icon name="plus" size={14} /></button>
        </form>
      {/if}
    </div>
  {/if}

  {#if editable}
    {@render item('rename', 'fileText', t('menu.rename'))}
    {#if section === 'rename'}
      <form onsubmit={rename} class="{subCls} flex gap-1.5">
        <!-- svelte-ignore a11y_autofocus -->
        <input class="input py-1! text-[13px]!" bind:value={name} maxlength="200" autofocus aria-label={t('menu.rename')} />
        <button type="submit" class="btn-primary px-2! py-1!" disabled={!name.trim()} aria-label={t('common.save')}><Icon name="check" size={14} /></button>
      </form>
    {/if}

    {@render item('status', 'board', t('common.status'), statusLabel(p.status))}
    {#if section === 'status'}
      <div class={subCls}>
        {#each ALL_STATUSES as s (s)}
          <button type="button" class="{itemCls} text-[13px]" onclick={() => setStatus(s)} aria-pressed={p.status === s}>
            <span class="size-2 shrink-0 rounded-full {statusDotCls(s)}" style={statusDotStyle(s)}></span>
            <span class="flex-1">{statusLabel(s)}</span>
            {#if p.status === s}<Icon name="check" size={14} class="text-accent-600 dark:text-accent-400" />{/if}
          </button>
        {/each}
      </div>
    {/if}

    {@render item('due', 'calendar', t('project.schedule'), dateRange)}
    {#if section === 'due'}
      <div class="{subCls} space-y-1.5">
        {#each [{ key: 'start_date', label: t('project.startDate'), value: p.start_date }, { key: 'due_date', label: t('project.dueDate'), value: p.due_date }] as const as d (d.key)}
          <label class="flex items-center gap-1.5 text-[13px]">
            <span class="w-20 shrink-0 text-zinc-500">{d.label}</span>
            <input type="date" class="input py-1! text-[13px]!" value={d.value ?? ''} onchange={(e) => setDate(d.key, e.currentTarget.value)} />
            {#if d.value}
              <button type="button" class="btn-ghost px-1.5!" onclick={() => setDate(d.key, '')} title={t('menu.clearDate')} aria-label="{t('menu.clearDate')}: {d.label}"><Icon name="x" size={14} /></button>
            {/if}
          </label>
        {/each}
      </div>
    {/if}

    {@render item('owner', 'user', t('project.owner'), p.owner_name)}
    {#if section === 'owner'}
      <div class={subCls}>
        {#if !assignees}
          <p class="px-2 py-1 text-[13px] text-zinc-400">…</p>
        {:else}
          <button type="button" class="{itemCls} text-[13px]" onclick={() => setOwner(null)} aria-pressed={!p.owner_id}>
            <span class="flex size-5 items-center justify-center text-zinc-400"><Icon name="user" size={14} /></span>
            <span class="flex-1 text-zinc-500">{t('project.noOwner')}</span>
            {#if !p.owner_id}<Icon name="check" size={14} class="text-accent-600 dark:text-accent-400" />{/if}
          </button>
          {#each assignees as a (a.id)}
            <button type="button" class="{itemCls} text-[13px]" onclick={() => setOwner(a)} aria-pressed={p.owner_id === a.id}>
              <Avatar name={a.name} id={a.id} size={20} />
              <span class="min-w-0 flex-1 truncate">{a.name}</span>
              {#if p.owner_id === a.id}<Icon name="check" size={14} class="text-accent-600 dark:text-accent-400" />{/if}
            </button>
          {/each}
        {/if}
      </div>
    {/if}

    {@render item('tags', 'droplet', t('tags.title'), tagsByIds(p.tag_ids).map((tag) => tag.name).join(', '))}
    {#if section === 'tags'}
      <div class={subCls}>
        <TagMenu ids={p.tag_ids} onchange={(next) => setProjectTags(p.id, next).then(() => {})} onescape={onclose} />
      </div>
    {/if}
  {/if}

  <button type="button" class={itemCls} role="menuitem" onclick={() => toggleFavorite(p.id)}>
    <Icon name="star" size={15} class={fav ? 'fill-amber-400 text-amber-500' : 'text-zinc-400'} />{t(fav ? 'nav.removeFavorite' : 'nav.addFavorite')}
  </button>
</ContextMenu>
