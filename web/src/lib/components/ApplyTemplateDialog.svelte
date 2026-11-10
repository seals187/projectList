<script lang="ts">
  // "Use" a template: choose customer → project (or a new project), adjust the
  // title, fill in the variables of the template → the checklist (or the table of a
  // table template) is created and opened.
  import { untrack } from 'svelte';
  import { goto } from '$app/navigation';
  import { api, errorText } from '$lib/api';
  import { refreshSidebar, sidebar, toast } from '$lib/state.svelte';
  import { t, type Key } from '$lib/i18n.svelte';
  import type { Checklist, Project, Table, TemplateSummary, TemplateVariable } from '$lib/types';
  import Icon from './Icon.svelte';
  import { templateSizeText } from '$lib/templateText';
  import EntityIcon from './EntityIcon.svelte';

  let { template, customerId = null, onclose }: { template: TemplateSummary; customerId?: number | null; onclose: () => void } = $props();

  const NEW = -1; // "new project" in the project select
  const isTable = $derived(template.kind === 'table');
  const init = untrack(() => ({ customerId, name: template.name }));
  let customer = $state(init.customerId ?? sidebar.customers[0]?.id ?? 0);
  let project = $state(0);
  let newName = $state('');
  let title = $state(init.name);
  let vars = $state<TemplateVariable[] | null>(null);
  let values = $state<Record<string, string>>({});
  let busy = $state(false);
  let dialog = $state<HTMLDialogElement>();

  const projects = $derived((sidebar.customers.find((c) => c.id === customer)?.projects ?? []).filter((p) => p.status !== 'archived'));
  // first project of the customer preselected (or "new" when it has none)
  $effect(() => {
    const list = projects;
    untrack(() => {
      if (project !== NEW && !list.some((p) => p.id === project)) project = list[0]?.id ?? NEW;
    });
  });

  $effect(() => {
    dialog?.showModal();
    api.get<TemplateVariable[]>(`/templates/${template.id}/variables`).then(
      (r) => {
        vars = r;
        values = Object.fromEntries(r.filter((v) => asked(v)).map((v) => [v.name, v.default_value]));
      },
      (e) => toast(errorText(e))
    );
  });

  /** Variables the user fills in (custom and integration variables). */
  const asked = (v: TemplateVariable) => v.source !== 'system' && v.source !== 'unknown';
  const sourceKey = (s: string) => `var.source.${s}` as Key;

  async function submit(e: SubmitEvent) {
    e.preventDefault();
    if (!customer || (project === NEW && !newName.trim())) return;
    busy = true;
    try {
      let pid = project;
      if (pid === NEW) pid = (await api.post<Project>(`/customers/${customer}/projects`, { name: newName.trim() })).id;
      const body = { template_id: template.id, title: title.trim(), values };
      if (isTable) {
        const tb = await api.post<Table>(`/projects/${pid}/tables/from-template`, body);
        refreshSidebar(0);
        toast(t('apply.tableDone', { name: tb.title }), 'info');
        dialog?.close();
        goto(`/tables/${tb.id}`);
        return;
      }
      const cl = await api.post<Checklist>(`/projects/${pid}/checklists/from-template`, body);
      refreshSidebar(0);
      toast(t('apply.done', { name: cl.title }), 'info');
      dialog?.close();
      goto(`/checklists/${cl.id}`);
    } catch (err) {
      toast(errorText(err));
    } finally {
      busy = false;
    }
  }
</script>

<dialog
  bind:this={dialog}
  {onclose}
  onclick={(e) => e.target === dialog && dialog?.close()}
  class="m-auto w-[min(32rem,calc(100vw-2rem))] rounded-xl border border-zinc-200 bg-white p-0 text-zinc-900 shadow-xl backdrop:bg-zinc-950/30 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-100"
  data-testid="apply-dialog">
  <form onsubmit={submit} class="space-y-4 p-5">
    <div class="flex items-center gap-3">
      <EntityIcon type="template" size="md" />
      <div class="min-w-0 flex-1">
        <h2 class="text-base font-semibold">{t('apply.title')}</h2>
        <p class="truncate text-xs text-zinc-500">{template.name} · {templateSizeText(template)}</p>
      </div>
      <button type="button" class="btn-ghost px-1.5!" onclick={() => dialog?.close()} aria-label={t('common.cancel')}><Icon name="x" size={16} /></button>
    </div>

    <div class="grid grid-cols-2 gap-3">
      <label class="block">
        <span class="label">{t('project.customer')}</span>
        <select class="input" bind:value={customer} required>
          {#each sidebar.customers as c (c.id)}<option value={c.id}>{c.name}</option>{/each}
        </select>
      </label>
      <label class="block">
        <span class="label">{t('apply.project')}</span>
        <select class="input" bind:value={project}>
          {#each projects as p (p.id)}<option value={p.id}>{p.name}</option>{/each}
          <option value={NEW}>+ {t('apply.newProject')}</option>
        </select>
      </label>
    </div>
    {#if project === NEW}
      <label class="block">
        <span class="label">{t('project.name')}</span>
        <!-- svelte-ignore a11y_autofocus -->
        <input class="input" bind:value={newName} required maxlength="200" autofocus placeholder={t('apply.newProjectName')} />
      </label>
    {/if}
    <label class="block">
      <span class="label">{t(isTable ? 'apply.tableTitle' : 'apply.checklistTitle')}</span>
      <input class="input" bind:value={title} maxlength="200" />
    </label>

    {#if vars && vars.length}
      <div class="space-y-2 rounded-lg bg-zinc-50 p-3 dark:bg-zinc-800/50">
        <p class="text-[11px] font-semibold tracking-wider text-zinc-400 uppercase">{t('apply.variables')}</p>
        {#each vars.filter(asked) as v (v.name)}
          <label class="block">
            <span class="mb-0.5 flex items-center gap-2 text-[13px]">
              <code class="var-chip">{`{{${v.name}}}`}</code>
              <span class="truncate text-zinc-500">{v.description}</span>
            </span>
            <input class="input py-1! text-[13px]!" bind:value={values[v.name]} maxlength="500" required={v.source === 'custom'} placeholder={v.source === 'custom' ? '' : t(sourceKey(v.source))} />
          </label>
        {/each}
        {#if vars.some((v) => v.source === 'system')}
          <p class="flex flex-wrap items-center gap-1.5 text-xs text-zinc-500">
            {t('apply.auto')}
            {#each vars.filter((v) => v.source === 'system') as v (v.name)}<code class="var-chip">{`{{${v.name}}}`}</code>{/each}
          </p>
        {/if}
        {#if vars.some((v) => v.source === 'unknown')}
          <p class="flex flex-wrap items-center gap-1.5 text-xs text-amber-700 dark:text-amber-400">
            <Icon name="alert" size={13} />{t('apply.unknown')}
            {#each vars.filter((v) => v.source === 'unknown') as v (v.name)}<code class="var-chip">{`{{${v.name}}}`}</code>{/each}
          </p>
        {/if}
      </div>
    {/if}

    <div class="flex justify-end gap-2 pt-1">
      <button type="button" class="btn-ghost" onclick={() => dialog?.close()}>{t('common.cancel')}</button>
      <button type="submit" class="btn-primary" disabled={busy || !customer || (project === NEW && !newName.trim())}>{t(isTable ? 'apply.tableSubmit' : 'apply.submit')}</button>
    </div>
  </form>
</dialog>
