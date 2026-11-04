<script lang="ts">
  import { page } from '$app/state';
  import { goto } from '$app/navigation';
  import { api, errorText } from '$lib/api';
  import { canWrite, refreshSidebar, toast } from '$lib/state.svelte';
  import { fmtDate, shortName, t, todayISO } from '$lib/i18n.svelte';
  import { reorder, sortable } from '$lib/sortable';
  import type { Checklist, ChecklistSummary, Project, ProjectDetail, ProjectStatus } from '$lib/types';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import Editable from '$lib/components/Editable.svelte';
  import ConfirmButton from '$lib/components/ConfirmButton.svelte';
  import Progress from '$lib/components/Progress.svelte';
  import Avatar from '$lib/components/Avatar.svelte';
  import SaveStatus from '$lib/components/SaveStatus.svelte';
  import TemplatePicker from '$lib/components/TemplatePicker.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import { ALL_STATUSES } from '$lib/projectStatus';
  import { isFavorite, toggleFavorite } from '$lib/navMemory.svelte';

  let data = $state<ProjectDetail | null>(null);
  let notFound = $state(false);
  let mode = $state<'none' | 'blank' | 'template'>('none');
  let newTitle = $state('');

  const id = $derived(Number(page.params.id));
  const editable = $derived(canWrite());
  const today = todayISO();

  $effect(() => {
    load(id);
  });

  async function load(id: number) {
    notFound = false;
    mode = 'none';
    try {
      data = await api.get<ProjectDetail>(`/projects/${id}`);
    } catch {
      data = null;
      notFound = true;
    }
  }

  async function save(body: Partial<Project>) {
    if (!data) return;
    data.project = await api.patch<Project>(`/projects/${id}`, body);
    refreshSidebar(0);
  }

  async function setStatus(status: ProjectStatus) {
    try {
      await save({ status });
    } catch (e) {
      toast(errorText(e));
    }
  }

  async function setDue(value: string) {
    try {
      await save({ due_date: value || null });
    } catch (e) {
      toast(errorText(e));
    }
  }

  function summary(c: Checklist): ChecklistSummary {
    const due = c.items.reduce((m, i) => (i.kind === 'item' && i.due_date && i.due_date > m ? i.due_date : m), '');
    return { id: c.id, project_id: c.project_id, title: c.title, template_id: c.template_id, position: c.position, total: c.total, done: c.done, due_date: due, assignee_id: c.assignee_id, assignee_name: c.assignee_name ?? "" };
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
</script>

<svelte:head><title>{data?.project.name ?? ''} · ProjectList</title></svelte:head>

{#if notFound}
  <p class="text-zinc-500">{t('common.notFound')}</p>
{:else if data}
  <PageHeader crumbs={[{ label: data.customer.name, href: `/customers/${data.customer.id}` }]}>
    {#snippet actions()}
      {@const fav = isFavorite(id)}
      <button
        type="button"
        class="rounded-md p-1.5 hover:bg-zinc-100 dark:hover:bg-zinc-800 {fav ? 'text-amber-500' : 'text-zinc-400 hover:text-amber-500'}"
        onclick={() => toggleFavorite(id)}
        title={t(fav ? 'nav.removeFavorite' : 'nav.addFavorite')}
        aria-label={t(fav ? 'nav.removeFavorite' : 'nav.addFavorite')}
        aria-pressed={fav}>
        <Icon name="star" size={17} class={fav ? 'fill-amber-400' : ''} />
      </button>
      <SaveStatus />
      <label class="ml-2 flex items-center gap-1.5 text-xs text-zinc-500" title={t('project.dueDate')}>
        <Icon name="calendar" size={14} />
        <input
          type="date"
          class="input w-auto! py-1! text-xs!"
          value={data!.project.due_date ?? ''}
          disabled={!editable}
          onchange={(e) => setDue(e.currentTarget.value)}
          aria-label={t('project.dueDate')} />
      </label>
      <select
        class="input w-auto! py-1! text-xs!"
        value={data!.project.status}
        disabled={!editable}
        onchange={(e) => setStatus(e.currentTarget.value as ProjectStatus)}
        aria-label={t('common.status')}>
        {#each ALL_STATUSES as s (s)}<option value={s}>{t(`status.${s}`)}</option>{/each}
      </select>
    {/snippet}
    {#snippet title()}
      <Editable value={data!.project.name} required disabled={!editable} onsave={(name) => save({ name })} inputClass="font-serif text-2xl! font-bold" />
    {/snippet}
    {#snippet meta()}
      <Progress done={data!.checklists.reduce((s, c) => s + c.done, 0)} total={data!.checklists.reduce((s, c) => s + c.total, 0)} />
    {/snippet}
  </PageHeader>

  <div class="max-w-2xl text-sm text-zinc-600 dark:text-zinc-300">
    <Editable value={data.project.description} multiline placeholder={t('project.descPlaceholder')} disabled={!editable} onsave={(description) => save({ description })} />
  </div>

  <section class="mt-10">
    <h2 class="section-label mb-2">{t('project.checklists')}</h2>
    {#if data.checklists.length === 0}
      <p class="py-4 text-sm text-zinc-400">{t('project.noChecklists')}</p>
    {/if}
    <ul use:sortable={{ onsort: sortChecklists, disabled: !editable }}>
      {#each data.checklists as cl (cl.id)}
        <li data-id={cl.id} class="group relative rounded-md bg-white dark:bg-zinc-950 [&.dragging]:shadow-lg">
          {#if editable}
            <button type="button" data-handle class="absolute top-1/2 -left-5 -translate-y-1/2 cursor-grab touch-none text-zinc-300 md:opacity-0 md:group-hover:opacity-100 dark:text-zinc-600" aria-label={t('checklist.drag')}>
              <Icon name="grip" size={14} />
            </button>
          {/if}
          <a href="/checklists/{cl.id}" class="row-link">
            <span class="flex-1 truncate font-medium">{cl.title}</span>
            {#if cl.assignee_id && cl.assignee_name}
              <span class="hidden items-center gap-1.5 text-xs text-zinc-500 sm:inline-flex dark:text-zinc-400" title={t("checklist.assignedTo", { name: cl.assignee_name })}>
                <Avatar name={cl.assignee_name} id={cl.assignee_id} size={18} />{shortName(cl.assignee_name)}
              </span>
            {/if}
            {#if cl.due_date}
              <span class="hidden text-xs tabular-nums sm:inline {cl.done < cl.total && cl.due_date < today ? 'text-red-600' : 'text-zinc-400'}">{t('checklist.due', { date: fmtDate(cl.due_date) })}</span>
            {/if}
            <Progress done={cl.done} total={cl.total} width="w-16" compact />
          </a>
        </li>
      {/each}
    </ul>

    {#if editable}
      <div class="mt-3">
        {#if mode === 'blank'}
          <form onsubmit={addChecklist} class="flex max-w-xl gap-2">
            <!-- svelte-ignore a11y_autofocus -->
            <input class="input" placeholder={t('project.checklistTitle')} bind:value={newTitle} autofocus onkeydown={(e) => e.key === 'Escape' && (mode = 'none')} />
            <button class="btn-primary" type="submit">{t('common.create')}</button>
            <button class="btn-ghost" type="button" onclick={() => (mode = 'none')}>{t('common.cancel')}</button>
          </form>
        {:else if mode === 'template'}
          <TemplatePicker projectId={id} oncreated={created} oncancel={() => (mode = 'none')} />
        {:else}
          <div class="flex flex-wrap gap-2">
            <button type="button" class="btn-outline" onclick={() => (mode = 'blank')}><Icon name="plus" size={14} /> {t('project.newChecklist')}</button>
            <button type="button" class="btn-outline" onclick={() => (mode = 'template')}><Icon name="template" size={14} /> {t('project.fromTemplate')}</button>
          </div>
        {/if}
      </div>
    {/if}
  </section>

  {#if editable}
    <div class="mt-16 border-t border-zinc-100 pt-4 dark:border-zinc-800">
      <ConfirmButton onconfirm={remove} label={t('project.delete')} confirmLabel={t('project.deleteConfirm')} />
    </div>
  {/if}
{/if}
