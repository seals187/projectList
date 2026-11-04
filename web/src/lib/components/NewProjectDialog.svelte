<script lang="ts">
  // Create a new project (overview): customer, name, status, due date and
  // optionally a first checklist from a template.
  import { untrack } from 'svelte';
  import { goto } from '$app/navigation';
  import { api, errorText } from '$lib/api';
  import { refreshSidebar, sidebar, toast } from '$lib/state.svelte';
  import { t } from '$lib/i18n.svelte';
  import { ALL_STATUSES } from '$lib/projectStatus';
  import type { Project, ProjectStatus, TemplateSummary } from '$lib/types';
  import Icon from './Icon.svelte';

  let {
    customerId = null,
    status = 'planning',
    onclose
  }: { customerId?: number | null; status?: ProjectStatus; onclose: () => void } = $props();

  // Take initial values from the props once (freely editable afterwards)
  const init = untrack(() => ({ customerId, status }));
  let customer = $state(init.customerId ?? sidebar.customers[0]?.id ?? 0);
  let name = $state('');
  let st = $state<ProjectStatus>(init.status);
  let due = $state('');
  let templateId = $state(0);
  let templates = $state<TemplateSummary[]>([]);
  let busy = $state(false);
  let dialog = $state<HTMLDialogElement>();

  $effect(() => {
    dialog?.showModal();
    api.get<TemplateSummary[]>('/templates').then(
      (r) => (templates = r),
      () => {}
    );
  });

  async function submit(e: SubmitEvent) {
    e.preventDefault();
    if (!name.trim() || !customer) return;
    busy = true;
    try {
      const p = await api.post<Project>(`/customers/${customer}/projects`, { name, status: st, due_date: due || null });
      if (templateId) {
        try {
          await api.post(`/projects/${p.id}/checklists/from-template`, { template_id: templateId, title: '' });
        } catch (err) {
          toast(errorText(err)); // project was created, only the template is missing
        }
      }
      refreshSidebar(0);
      onclose();
      goto(`/projects/${p.id}`);
    } catch (err) {
      toast(errorText(err));
    } finally {
      busy = false;
    }
  }
</script>

<dialog
  bind:this={dialog}
  onclose={onclose}
  onclick={(e) => e.target === dialog && dialog?.close()}
  class="m-auto w-[min(28rem,calc(100vw-2rem))] rounded-xl border border-zinc-200 bg-white p-0 text-zinc-900 shadow-xl backdrop:bg-zinc-950/30 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-100">
  <form onsubmit={submit} class="space-y-4 p-5">
    <div class="flex items-center justify-between">
      <h2 class="text-base font-semibold">{t('project.new')}</h2>
      <button type="button" class="btn-ghost px-1.5!" onclick={() => dialog?.close()} aria-label={t('common.cancel')}><Icon name="x" size={16} /></button>
    </div>
    <label class="block">
      <span class="label">{t('project.customer')}</span>
      <select class="input" bind:value={customer} required>
        {#each sidebar.customers as c (c.id)}<option value={c.id}>{c.name}</option>{/each}
      </select>
    </label>
    <label class="block">
      <span class="label">{t('project.name')}</span>
      <!-- svelte-ignore a11y_autofocus -->
      <input class="input" bind:value={name} required maxlength="200" autofocus />
    </label>
    <div class="grid grid-cols-2 gap-3">
      <label class="block">
        <span class="label">{t('common.status')}</span>
        <select class="input" bind:value={st}>
          {#each ALL_STATUSES as s (s)}<option value={s}>{t(`status.${s}`)}</option>{/each}
        </select>
      </label>
      <label class="block">
        <span class="label">{t('project.dueDate')}</span>
        <input class="input" type="date" bind:value={due} />
      </label>
    </div>
    <label class="block">
      <span class="label">{t('project.firstChecklist')}</span>
      <select class="input" bind:value={templateId}>
        <option value={0}>{t('project.noTemplate')}</option>
        {#each templates as tpl (tpl.id)}<option value={tpl.id}>{tpl.name}</option>{/each}
      </select>
    </label>
    <div class="flex justify-end gap-2 pt-1">
      <button type="button" class="btn-ghost" onclick={() => dialog?.close()}>{t('common.cancel')}</button>
      <button type="submit" class="btn-primary" disabled={busy || !name.trim() || !customer}>{t('common.create')}</button>
    </div>
  </form>
</dialog>
