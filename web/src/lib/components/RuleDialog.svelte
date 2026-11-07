<script lang="ts">
  // Create or edit an automation rule (admins): trigger (+ condition or
  // schedule) → template, optional checklist title, on/off. Triggers of
  // integrations (Easy Redmine, OTOBO) are shown but not available yet.
  import { untrack } from 'svelte';
  import { api, errorText } from '$lib/api';
  import { sidebar, toast } from '$lib/state.svelte';
  import { locale, t, type Key } from '$lib/i18n.svelte';
  import { ALL_STATUSES, statusLabel } from '$lib/projectStatus';
  import { APP_TRIGGERS, EXTERNAL_TRIGGERS, INTEGRATIONS, sourceLabel } from '$lib/automationText';
  import type { AutomationCondition, AutomationRule, AutomationSchedule, TemplateSummary } from '$lib/types';
  import Icon from './Icon.svelte';
  import Switch from './Switch.svelte';

  let {
    rule = null,
    templates,
    onsaved,
    onclose
  }: { rule?: AutomationRule | null; templates: TemplateSummary[]; onsaved: () => void; onclose: () => void } = $props();

  const init = untrack(() => rule);
  const firstCond = init?.conditions[0];
  let trigger = $state(init?.trigger_kind ?? 'project_created');
  // condition: project_created → name contains / customer equals; tag / status → equals
  let condField = $state<'none' | 'name' | 'customer'>(init?.trigger_kind === 'project_created' && firstCond ? (firstCond.field as 'name' | 'customer') : 'none');
  let condValue = $state(firstCond?.value ?? '');
  let tagId = $state(init?.trigger_kind === 'project_tag' ? (firstCond?.value ?? '') : String(sidebar.tags[0]?.id ?? ''));
  let status = $state(init?.trigger_kind === 'project_status' ? (firstCond?.value ?? 'done') : 'done');
  let sched = $state<AutomationSchedule>(init?.schedule ?? { freq: 'yearly', month: 1, day: 1 });
  let targetCustomer = $state(sidebar.customers.find((c) => c.projects.some((p) => p.id === init?.target_project_id))?.id ?? sidebar.customers[0]?.id ?? 0);
  let targetProject = $state(init?.target_project_id ?? 0);
  let templateId = $state(init?.template_id ?? untrack(() => templates[0]?.id) ?? 0);
  let title = $state(init?.checklist_title ?? '');
  let enabled = $state(init?.enabled ?? true);
  let busy = $state(false);
  let confirmDel = $state(false);
  let dialog = $state<HTMLDialogElement>();

  const targetProjects = $derived((sidebar.customers.find((c) => c.id === targetCustomer)?.projects ?? []).filter((p) => p.status !== 'archived'));
  $effect(() => {
    const list = targetProjects;
    untrack(() => {
      if (!list.some((p) => p.id === targetProject)) targetProject = list[0]?.id ?? 0;
    });
  });
  $effect(() => {
    dialog?.showModal();
  });

  function conditions(): AutomationCondition[] {
    if (trigger === 'project_tag') return [{ field: 'tag', op: 'equals', value: tagId }];
    if (trigger === 'project_status') return [{ field: 'status', op: 'equals', value: status }];
    if (trigger === 'project_created' && condField !== 'none' && condValue.trim())
      return [{ field: condField, op: condField === 'name' ? 'contains' : 'equals', value: condValue.trim() }];
    return [];
  }
  const valid = $derived(
    !!templateId &&
      (trigger !== 'schedule' || !!targetProject) &&
      (trigger !== 'project_tag' || !!tagId) &&
      (trigger !== 'project_created' || condField === 'none' || !!condValue.trim())
  );

  async function save(e: SubmitEvent) {
    e.preventDefault();
    if (!valid) return;
    busy = true;
    const body = {
      trigger_kind: trigger,
      conditions: conditions(),
      schedule: trigger === 'schedule' ? sched : undefined,
      template_id: templateId,
      target_project_id: trigger === 'schedule' ? targetProject : null,
      checklist_title: title,
      enabled
    };
    try {
      if (init) await api.patch(`/automations/${init.id}`, body);
      else await api.post('/automations', body);
      onsaved();
      dialog?.close();
    } catch (err) {
      toast(errorText(err));
    } finally {
      busy = false;
    }
  }
  async function remove() {
    try {
      await api.del(`/automations/${init!.id}`);
      onsaved();
      dialog?.close();
    } catch (err) {
      toast(errorText(err));
    }
  }
  const MONTHS = Array.from({ length: 12 }, (_, i) => i + 1);
  const DAYS = Array.from({ length: 31 }, (_, i) => i + 1);
</script>

<dialog
  bind:this={dialog}
  {onclose}
  onclick={(e) => e.target === dialog && dialog?.close()}
  class="m-auto w-[min(32rem,calc(100vw-2rem))] rounded-xl border border-zinc-200 bg-white p-0 text-zinc-900 shadow-xl backdrop:bg-zinc-950/30 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-100"
  data-testid="rule-dialog">
  <form onsubmit={save} class="space-y-4 p-5">
    <div class="flex items-center justify-between">
      <h2 class="text-base font-semibold">{t(init ? 'auto.edit' : 'auto.new')}</h2>
      <button type="button" class="btn-ghost px-1.5!" onclick={() => dialog?.close()} aria-label={t('common.cancel')}><Icon name="x" size={16} /></button>
    </div>

    <!-- WHEN -->
    <fieldset class="space-y-2.5 rounded-lg bg-zinc-50 p-3 dark:bg-zinc-800/50">
      <legend class="sr-only">{t('auto.when')}</legend>
      <p class="flex items-center gap-1.5 text-[11px] font-semibold tracking-wider text-zinc-400 uppercase"><Icon name="zap" size={12} />{t('auto.when')}</p>
      <select class="input" bind:value={trigger} aria-label={t('auto.trigger')}>
        {#each APP_TRIGGERS as k (k)}<option value={k}>{t(`auto.trigger.${k}` as Key)}</option>{/each}
        <optgroup label={t('auto.integrationsSoon')}>
          {#each INTEGRATIONS as src (src)}
            {#each EXTERNAL_TRIGGERS as k (k)}<option value={k} disabled>{sourceLabel(src)}: {t(`auto.trigger.${k}` as Key)}</option>{/each}
          {/each}
        </optgroup>
      </select>

      {#if trigger === 'project_created'}
        <div class="flex gap-2">
          <select
            class="input w-auto!"
            bind:value={condField}
            onchange={() => (condValue = condField === 'customer' ? String(sidebar.customers[0]?.id ?? '') : '')}
            aria-label={t('auto.condition')}>
            <option value="none">{t('auto.always')}</option>
            <option value="name">{t('auto.cond.nameContains')}</option>
            <option value="customer">{t('auto.cond.customerIs')}</option>
          </select>
          {#if condField === 'name'}
            <input class="input" bind:value={condValue} maxlength="200" placeholder="Migration" aria-label={t('auto.field.name')} />
          {:else if condField === 'customer'}
            <select class="input" bind:value={condValue} aria-label={t('auto.field.customer')}>
              {#each sidebar.customers as c (c.id)}<option value={String(c.id)}>{c.name}</option>{/each}
            </select>
          {/if}
        </div>
      {:else if trigger === 'project_tag'}
        <label class="flex items-center gap-2 text-sm">
          <span class="shrink-0 text-zinc-500">{t('auto.field.tag')}</span>
          <select class="input" bind:value={tagId}>
            {#each sidebar.tags as tag (tag.id)}<option value={String(tag.id)}>{tag.name}</option>{/each}
          </select>
        </label>
      {:else if trigger === 'project_status'}
        <label class="flex items-center gap-2 text-sm">
          <span class="shrink-0 text-zinc-500">{t('auto.cond.statusTo')}</span>
          <select class="input" bind:value={status}>
            {#each ALL_STATUSES as s (s)}<option value={s}>{statusLabel(s)}</option>{/each}
          </select>
        </label>
      {:else if trigger === 'schedule'}
        <div class="flex flex-wrap gap-2">
          <select class="input w-auto!" bind:value={sched.freq} aria-label={t('auto.freq')}>
            <option value="yearly">{t('auto.freq.yearly')}</option>
            <option value="monthly">{t('auto.freq.monthly')}</option>
            <option value="weekly">{t('auto.freq.weekly')}</option>
          </select>
          {#if sched.freq === 'weekly'}
            <select class="input w-auto!" bind:value={sched.weekday} aria-label={t('auto.weekday')}>
              {#each [1, 2, 3, 4, 5, 6, 7] as d (d)}<option value={d}>{t(`weekday.${d}` as Key)}</option>{/each}
            </select>
          {:else}
            <select class="input w-auto!" bind:value={sched.day} aria-label={t('auto.day')}>
              {#each DAYS as d (d)}<option value={d}>{d}.</option>{/each}
            </select>
            {#if sched.freq === 'yearly'}
              <select class="input w-auto!" bind:value={sched.month} aria-label={t('auto.month')}>
                {#each MONTHS as m (m)}<option value={m}>{new Date(2026, m - 1, 1).toLocaleDateString(locale(), { month: 'long' })}</option>{/each}
              </select>
            {/if}
          {/if}
        </div>
        <div class="grid grid-cols-2 gap-2">
          <label class="block">
            <span class="label">{t('project.customer')}</span>
            <select class="input" bind:value={targetCustomer}>
              {#each sidebar.customers as c (c.id)}<option value={c.id}>{c.name}</option>{/each}
            </select>
          </label>
          <label class="block">
            <span class="label">{t('apply.project')}</span>
            <select class="input" bind:value={targetProject}>
              {#each targetProjects as p (p.id)}<option value={p.id}>{p.name}</option>{/each}
            </select>
          </label>
        </div>
      {/if}
    </fieldset>

    <!-- THEN -->
    <fieldset class="space-y-2.5 rounded-lg bg-zinc-50 p-3 dark:bg-zinc-800/50">
      <legend class="sr-only">{t('auto.then')}</legend>
      <p class="flex items-center gap-1.5 text-[11px] font-semibold tracking-wider text-zinc-400 uppercase"><Icon name="arrowRight" size={12} />{t('auto.then')}</p>
      <label class="block">
        <span class="label">{t('auto.template')}</span>
        <select class="input" bind:value={templateId} required>
          {#each templates as tpl (tpl.id)}<option value={tpl.id}>{tpl.name}</option>{/each}
        </select>
      </label>
      <label class="block">
        <span class="label">{t('apply.checklistTitle')}</span>
        <input class="input" bind:value={title} maxlength="200" placeholder={templates.find((x) => x.id === templateId)?.name ?? ''} />
        <span class="mt-1 block text-xs text-zinc-500">{t('auto.titleHint')}</span>
      </label>
    </fieldset>

    <label class="flex items-center gap-2.5 text-sm">
      <Switch label={t('auto.enabled')} checked={enabled} onchange={(v) => (enabled = v)} />
      {t('auto.enabled')}
    </label>

    <div class="flex items-center gap-2 pt-1">
      {#if init}
        {#if confirmDel}
          <button type="button" class="btn bg-red-600 text-white hover:bg-red-700" onclick={remove}>{t('auto.deleteConfirm')}</button>
          <button type="button" class="btn-ghost" onclick={() => (confirmDel = false)}>{t('common.cancel')}</button>
        {:else}
          <button type="button" class="btn-danger" onclick={() => (confirmDel = true)}><Icon name="trash" size={14} />{t('common.delete')}</button>
        {/if}
      {/if}
      <span class="flex-1"></span>
      <button type="button" class="btn-ghost" onclick={() => dialog?.close()}>{t('common.cancel')}</button>
      <button type="submit" class="btn-primary" disabled={busy || !valid}>{t('common.save')}</button>
    </div>
  </form>
</dialog>
