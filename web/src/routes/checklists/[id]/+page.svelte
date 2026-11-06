<script lang="ts">
  import { page } from '$app/state';
  import { goto } from '$app/navigation';
  import { tick } from 'svelte';
  import { api, errorText } from '$lib/api';
  import { canWrite, current, refreshSidebar, toast } from '$lib/state.svelte';
  import { fmtDate, t } from '$lib/i18n.svelte';
  import { computeSections, countProgress } from '$lib/blocks';
  import type { ChecklistDetail, Item, Ops, Template } from '$lib/types';
  import DetailHeader from '$lib/components/DetailHeader.svelte';
  import BlockEditor from '$lib/components/BlockEditor.svelte';
  import Editable from '$lib/components/Editable.svelte';
  import FavoriteStar from '$lib/components/FavoriteStar.svelte';
  import ConfirmButton from '$lib/components/ConfirmButton.svelte';
  import Progress from '$lib/components/Progress.svelte';
  import SaveStatus from '$lib/components/SaveStatus.svelte';
  import AssigneePicker from '$lib/components/AssigneePicker.svelte';

  let data = $state<ChecklistDetail | null>(null);
  let notFound = $state(false);
  let saveAs = $state(false);
  let templateName = $state('');

  const id = $derived(Number(page.params.id));
  const editable = $derived(canWrite());

  $effect(() => {
    load(id);
  });

  async function load(id: number) {
    notFound = false;
    saveAs = false;
    try {
      data = await api.get<ChecklistDetail>(`/checklists/${id}`);
      await tick();
      if (location.hash) document.querySelector(location.hash)?.scrollIntoView({ block: 'start' });
    } catch {
      data = null;
      notFound = true;
    }
  }

  const progress = $derived(countProgress(data?.checklist.items ?? []));
  const due = $derived.by(() => {
    let max = '';
    for (const it of data?.checklist.items ?? []) if (it.kind === 'item' && it.due_date && it.due_date > max) max = it.due_date;
    return max;
  });

  // Provide H1 sections for the sidebar.
  $effect(() => {
    current.checklistId = data?.checklist.id ?? null;
    current.sections = computeSections(data?.checklist.items ?? []);
    return () => {
      current.checklistId = null;
      current.sections = [];
    };
  });

  const ops: Ops<Item> = {
    create: (body) => api.post<Item>(`/checklists/${id}/items`, body),
    patch: (itemId, body) => api.patch<Item>(`/items/${itemId}`, body),
    remove: (itemId) => api.del(`/items/${itemId}`),
    reorder: (ids) => api.put(`/checklists/${id}/items/order`, { ids }),
    toggle: (itemId, done) => api.post<Item>(`/items/${itemId}/done`, { done })
  };

  async function rename(title: string) {
    await api.patch(`/checklists/${id}`, { title });
    if (data) data.checklist.title = title;
    refreshSidebar(0);
  }

  async function assign(assigneeId: number | null) {
    if (!data) return;
    const prev = { id: data.checklist.assignee_id, name: data.checklist.assignee_name };
    data.checklist.assignee_id = assigneeId;
    data.checklist.assignee_name = data.assignees.find((a) => a.id === assigneeId)?.name ?? null;
    try {
      await api.patch(`/checklists/${id}`, { assignee_id: assigneeId });
    } catch (err) {
      data.checklist.assignee_id = prev.id;
      data.checklist.assignee_name = prev.name;
      toast(errorText(err));
    }
  }

  async function saveAsTemplate(e: SubmitEvent) {
    e.preventDefault();
    try {
      const tpl = await api.post<Template>(`/checklists/${id}/save-as-template`, { name: templateName });
      toast(t('checklist.templateSaved', { name: tpl.name }), 'info');
      saveAs = false;
    } catch (err) {
      toast(errorText(err));
    }
  }

  async function remove() {
    if (!data) return;
    const projectId = data.project.id;
    try {
      await api.del(`/checklists/${id}`);
      refreshSidebar(0);
      goto(`/projects/${projectId}`);
    } catch (e) {
      toast(errorText(e));
    }
  }
</script>

<svelte:head><title>{data?.checklist.title ?? ''} · ProjectList</title></svelte:head>

{#if notFound}
  <p class="text-zinc-500">{t('common.notFound')}</p>
{:else if data}
  <DetailHeader crumbs={[{ label: data.customer.name, href: `/customers/${data.customer.id}` }, { label: data.project.name, href: `/projects/${data.project.id}` }]}>
    {#snippet actions()}
      <SaveStatus />
      {#if editable}
        <button type="button" class="btn-outline ml-2" onclick={() => ((saveAs = !saveAs), (templateName = data!.checklist.title))}>
          {t('checklist.saveAsTemplate')}
        </button>
      {/if}
    {/snippet}
    {#snippet title()}
      <Editable value={data!.checklist.title} required disabled={!editable} onsave={rename} inputClass="font-serif text-2xl! font-bold" />
    {/snippet}
    {#snippet meta()}
      <FavoriteStar kind="checklist" id={data!.checklist.id} />
      <Progress done={progress.done} total={progress.total} label={false} />
      <span class="text-[13px] text-zinc-500 tabular-nums dark:text-zinc-400">
        {t('checklist.progress', progress)}{#if due}{' · '}{t('checklist.due', { date: fmtDate(due) })}{/if}
      </span>
      <span class="ml-1">
        <AssigneePicker
          assignees={data!.assignees}
          value={data!.checklist.assignee_id}
          name={data!.checklist.assignee_name}
          disabled={!editable}
          onchange={assign} />
      </span>
    {/snippet}
  </DetailHeader>

  {#if saveAs}
    <form onsubmit={saveAsTemplate} class="mb-6 flex max-w-xl gap-2">
      <!-- svelte-ignore a11y_autofocus -->
      <input class="input" placeholder={t('checklist.templateName')} bind:value={templateName} required autofocus />
      <button class="btn-primary" type="submit">{t('common.save')}</button>
      <button class="btn-ghost" type="button" onclick={() => (saveAs = false)}>{t('common.cancel')}</button>
    </form>
  {/if}

  {#key data.checklist.id}
    <BlockEditor bind:items={data.checklist.items} {ops} {editable} collapseKey="checklist:{id}" assignees={data.assignees} onchange={() => refreshSidebar()} onerror={() => load(id)} />
  {/key}

  {#if editable}
    <div class="mt-16 border-t border-zinc-100 pt-4 dark:border-zinc-800">
      <ConfirmButton onconfirm={remove} label={t('checklist.delete')} confirmLabel={t('checklist.deleteConfirm')} />
    </div>
  {/if}
{/if}
