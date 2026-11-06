<script lang="ts">
  // "New checklist from template" with a preview of the items.
  import { api, errorText } from '$lib/api';
  import { toast } from '$lib/state.svelte';
  import { t } from '$lib/i18n.svelte';
  import { htmlToText } from '$lib/rich';
  import type { Checklist, Template, TemplateItem, TemplateSummary } from '$lib/types';
  import EntityIcon from './EntityIcon.svelte';

  let { projectId, oncreated, oncancel }: { projectId: number; oncreated: (c: Checklist) => void; oncancel: () => void } =
    $props();

  let templates = $state<TemplateSummary[] | null>(null);
  let selected = $state<number | null>(null);
  let preview = $state<TemplateItem[]>([]);
  let title = $state('');
  let busy = $state(false);

  $effect(() => {
    api.get<TemplateSummary[]>('/templates').then(
      (tpl) => (templates = tpl),
      (e) => toast(errorText(e))
    );
  });

  async function select(tpl: TemplateSummary) {
    selected = tpl.id;
    title = tpl.name;
    preview = [];
    try {
      const r = await api.get<{ template: Template; items: TemplateItem[] }>(`/templates/${tpl.id}`);
      if (selected === tpl.id) preview = r.items;
    } catch (e) {
      toast(errorText(e));
    }
  }

  async function create() {
    if (!selected) return;
    busy = true;
    try {
      oncreated(await api.post<Checklist>(`/projects/${projectId}/checklists/from-template`, { template_id: selected, title }));
    } catch (e) {
      toast(errorText(e));
    } finally {
      busy = false;
    }
  }
</script>

<div class="rounded-lg border border-zinc-200 p-4 dark:border-zinc-800">
  <div class="mb-3 flex items-center justify-between">
    <h3 class="font-serif text-[17px] font-bold">{t('picker.title')}</h3>
    <button type="button" class="btn-ghost" onclick={oncancel}>{t('common.cancel')}</button>
  </div>

  {#if templates === null}
    <p class="text-sm text-zinc-400">{t('common.loading')}</p>
  {:else if templates.length === 0}
    <p class="text-sm text-zinc-500">
      {t('picker.noTemplates')} <a href="/templates" class="text-accent-600 hover:underline">{t('picker.createTemplates')}</a>
    </p>
  {:else}
    <div class="grid gap-4 sm:grid-cols-2">
      <ul class="max-h-72 space-y-px overflow-y-auto">
        {#each templates as tpl (tpl.id)}
          <li>
            <button
              type="button"
              onclick={() => select(tpl)}
              class="flex w-full items-center justify-between rounded-md px-2.5 py-1.5 text-left text-sm {selected === tpl.id
                ? 'bg-accent-50 text-accent-700 dark:bg-accent-950/60 dark:text-accent-400'
                : 'hover:bg-zinc-100 dark:hover:bg-zinc-800'}">
              <span class="flex min-w-0 items-center gap-2"><EntityIcon type="template" size="xs" /><span class="truncate">{tpl.name}</span></span>
              <span class="text-xs text-zinc-400 tabular-nums">{tpl.item_count}</span>
            </button>
          </li>
        {/each}
      </ul>
      <div class="rounded-md bg-zinc-50 p-3 dark:bg-zinc-900">
        {#if selected}
          <p class="label">{t('picker.preview')}</p>
          <ul class="max-h-48 space-y-0.5 overflow-y-auto text-sm">
            {#each preview as it (it.id)}
              {#if it.kind === 'h1' || it.kind === 'h2' || it.kind === 'h3' || it.kind === 'h4'}
                <li class="pt-1.5 font-serif font-bold">{htmlToText(it.title)}</li>
              {:else}
                <li class="flex gap-2"><span class="mt-[5px] size-2.5 shrink-0 rounded-full border border-zinc-300 dark:border-zinc-600"></span>{htmlToText(it.title)}</li>
              {/if}
            {:else}
              <li class="text-zinc-400">{t('picker.empty')}</li>
            {/each}
          </ul>
          <div class="mt-3">
            <label class="label" for="tp-title">{t('picker.titleLabel')}</label>
            <input id="tp-title" class="input" bind:value={title} />
          </div>
          <button type="button" class="btn-primary mt-3 w-full" disabled={busy} onclick={create}>{t('picker.create')}</button>
          <p class="mt-2 text-xs text-zinc-500">{t('picker.snapshot')}</p>
        {:else}
          <p class="text-sm text-zinc-400">{t('picker.select')}</p>
        {/if}
      </div>
    </div>
  {/if}
</div>
