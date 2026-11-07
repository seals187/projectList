<script lang="ts">
  // Context menu of a project (overview, customer pages, My tasks): open, favorite,
  // checklists, rename, status, due date, person responsible, tags, delete.
  // Sections expand inline.
  import type { ComponentProps } from 'svelte';
  import { goto } from '$app/navigation';
  import { api, errorText } from '$lib/api';
  import { fmtDate, t, type Key } from '$lib/i18n.svelte';
  import { canWrite, refreshSidebar, sidebar, toast } from '$lib/state.svelte';
  import { ALL_STATUSES, statusBadgeCls, statusBadgeStyle, statusDotCls, statusDotStyle, statusLabel } from '$lib/projectStatus';
  import { customerAssignees, patchProject, setProjectTags, storeProject } from '$lib/projectActions';
  import { myColors, setMyColor } from '$lib/myColors.svelte';
  import { isFavorite, toggleFavorite } from '$lib/navMemory.svelte';
  import { cardHex } from '$lib/cardColors';
  import CardColorPicker from './CardColorPicker.svelte';
  import { tagsByIds } from '$lib/tags';
  import type { Assignee, Checklist, ProjectStatus, SidebarCustomer, SidebarProject } from '$lib/types';
  import ContextMenu from './ContextMenu.svelte';
  import TagMenu from './TagMenu.svelte';
  import Avatar from './Avatar.svelte';
  import Icon from './Icon.svelte';
  import EntityIcon from './EntityIcon.svelte';

  let {
    project,
    customer,
    x,
    y,
    onclose,
    initial = null,
    boardColor = false,
    onchanged
  }: {
    project: SidebarProject;
    customer: SidebarCustomer;
    x: number;
    y: number;
    onclose: () => void;
    initial?: Section | null;
    boardColor?: boolean; // board: hint that a shared customer color tints the card
    onchanged?: () => void; // after a saved change (pages with their own project list reload it)
  } = $props();

  type IconName = ComponentProps<typeof Icon>['name'];
  type Section = 'checklists' | 'rename' | 'status' | 'due' | 'owner' | 'tags' | 'color';
  // the customer color wins in the board when "use in all views" is on
  const cust = $derived(sidebar.customers.find((c) => c.id === customer.id) ?? customer);
  const customerWide = $derived(cust.color_everywhere && !!cust.color);
  // svelte-ignore state_referenced_locally
  let section = $state<Section | null>(initial); // quick actions on the card open a section directly
  const toggle = (s: Section) => (section = section === s ? null : s);

  // always show the current state from the store (the overview passes a copy)
  const p = $derived(storeProject(project.id) ?? project);
  const editable = canWrite();

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
    if (name.trim() && name.trim() !== p.name && changed(await patchProject(p.id, { name: name.trim() }))) section = null;
  }

  // report saved changes (null = failed)
  function changed<T>(r: T): T {
    if (r) onchanged?.();
    return r;
  }
  const fav = $derived(isFavorite('project', project.id));
  const setStatus = (status: ProjectStatus) => patchProject(p.id, { status }).then(changed);
  const setDate = (key: 'start_date' | 'due_date', v: string) => patchProject(p.id, { [key]: v || null }).then(changed);
  const dateRange = $derived(
    p.start_date && p.due_date ? `${fmtDate(p.start_date)} – ${fmtDate(p.due_date)}` : p.due_date ? fmtDate(p.due_date) : p.start_date ? `${fmtDate(p.start_date)} –` : ''
  );
  const setOwner = (a: Assignee | null) => patchProject(p.id, { owner_id: a?.id ?? null }, a?.name).then(changed);

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

  // delete: inline confirmation (warning when checklists would be deleted too)
  let confirmDelete = $state(false);
  let deleting = $state(false);
  const count = (n: number, one: Key, many: Key) => (n === 1 ? t(one) : t(many, { n }));
  function deleteWarning() {
    const items = p.checklists.reduce((s, c) => s + c.total, 0);
    return t('project.deleteWarn', {
      checklists: count(p.checklists.length, 'project.del.cl1', 'project.del.cln'),
      items: count(items, 'project.del.it1', 'project.del.itn')
    });
  }
  async function removeProject() {
    deleting = true;
    try {
      await api.del(`/projects/${p.id}`);
      // drop it from the overview right away; the reload brings the rest in line
      for (const c of sidebar.customers) c.projects = c.projects.filter((x) => x.id !== p.id);
      refreshSidebar(0);
      toast(t('project.deleted', { name: p.name }));
      onchanged?.();
      onclose();
    } catch (err) {
      deleting = false;
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
    <EntityIcon type="customer" size="sm" class="mt-0.5" />
    <div class="min-w-0 flex-1">
      <p class="truncate font-semibold">{p.name}</p>
      <p class="truncate text-xs text-zinc-500">{customer.name}</p>
    </div>
    <span class="badge shrink-0 {statusBadgeCls(p.status)}" style={statusBadgeStyle(p.status)}>{statusLabel(p.status)}</span>
  </div>

  <button type="button" class={itemCls} role="menuitem" onclick={() => open(`/projects/${p.id}`)}>
    <Icon name="external" size={15} class="text-zinc-400" />{t('menu.openProject')}
  </button>
  <button type="button" class={itemCls} role="menuitem" onclick={() => (toggleFavorite('project', p.id), onclose())}>
    <Icon name="star" size={15} class={fav ? 'fill-amber-400 text-amber-500' : 'text-zinc-400'} />{t(fav ? 'nav.removeFavorite' : 'nav.addFavorite')}
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
        <TagMenu ids={p.tag_ids} onchange={(next) => setProjectTags(p.id, next).then(changed).then(() => {})} onescape={onclose} />
      </div>
    {/if}
  {/if}

  <!-- personal project color (everyone, also viewers): board card tint, elsewhere a stripe on the
       project row; in the board a shared customer color "in all views" tints the card instead -->
  {@const own = myColors.all.board[String(p.id)] ?? ''}
  <button type="button" class={itemCls} role="menuitem" aria-expanded={section === 'color'} onclick={() => toggle('color')}>
    <Icon name="palette" size={15} class="shrink-0 text-zinc-400" />
    <span>{t('menu.projectColor')}</span>
    <span class="ml-auto flex items-center gap-1.5">
      {#if own}<span class="size-3 rounded-full" style:background-color={cardHex(own)}></span>{/if}
      <Icon name="chevron" size={13} class="shrink-0 text-zinc-400 transition-transform {section === 'color' ? 'rotate-90' : ''}" />
    </span>
  </button>
  {#if section === 'color'}
    <div class={subCls}>
      <CardColorPicker label={t('menu.projectColor')} value={own} onpick={(c) => setMyColor('board', p.id, c)} />
      <p class="px-1 pt-1 text-xs text-zinc-400">{t('menu.projectColorHint')}</p>
      {#if boardColor && customerWide}<p class="px-1 pt-1 text-xs text-zinc-500">{t('menu.projectColorWide')}</p>{/if}
    </div>
  {/if}

  {#if editable}
    <div class="my-1 border-t border-zinc-100 dark:border-zinc-800"></div>
    {#if !confirmDelete}
      <button type="button" class="{itemCls} text-red-600 dark:text-red-400" role="menuitem" onclick={() => ((confirmDelete = true), (section = null))}>
        <Icon name="trash" size={15} class="shrink-0" />{t('project.delete')}
      </button>
    {:else}
      <div class="mx-1 mb-1 rounded-lg border border-red-200 bg-red-50 p-2.5 text-[13px] dark:border-red-500/30 dark:bg-red-500/10" role="alertdialog" aria-label={t('project.delete')} data-testid="project-delete-confirm">
        <p class="font-semibold text-red-700 dark:text-red-300">{t('project.deleteAsk', { name: p.name })}</p>
        {#if p.checklists.length}
          <p class="mt-1.5 flex gap-1.5 text-red-700 dark:text-red-300">
            <Icon name="alert" size={14} class="mt-0.5 shrink-0" />
            <span>{deleteWarning()}</span>
          </p>
        {/if}
        <p class="mt-1.5 text-zinc-500 dark:text-zinc-400">{t('project.deleteFinal')}</p>
        <div class="mt-2.5 flex justify-end gap-1.5">
          <button type="button" class="btn-outline py-1!" onclick={() => (confirmDelete = false)}>{t('common.cancel')}</button>
          <!-- svelte-ignore a11y_autofocus -->
          <button type="button" class="btn bg-red-600 py-1! text-white hover:bg-red-700" onclick={removeProject} disabled={deleting} autofocus>{t('common.delete')}</button>
        </div>
      </div>
    {/if}
  {/if}
</ContextMenu>
