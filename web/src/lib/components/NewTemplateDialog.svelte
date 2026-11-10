<script lang="ts">
  // "+ Template" in the library: name, type (checklist or table, optionally from an Excel
  // file) and category → the template is created and opened for editing.
  import { untrack } from 'svelte';
  import { goto } from '$app/navigation';
  import { api, errorText } from '$lib/api';
  import { toast } from '$lib/state.svelte';
  import { t } from '$lib/i18n.svelte';
  import { readXlsx, titleFromFile } from '$lib/xlsxImport';
  import type { Template, TemplateCategory, TemplateKind } from '$lib/types';
  import Icon from './Icon.svelte';
  import EntityIcon from './EntityIcon.svelte';

  let {
    categories,
    categoryId = 0,
    onclose
  }: { categories: TemplateCategory[]; categoryId?: number; onclose: () => void } = $props();

  let name = $state('');
  let kind = $state<TemplateKind>('checklist');
  let category = $state(untrack(() => categoryId)); // 0 = without category
  let busy = $state(false);
  let dialog = $state<HTMLDialogElement>();

  $effect(() => {
    dialog?.showModal();
  });

  const KINDS = [
    { kind: 'checklist', icon: 'listChecks', hint: 'templates.kindChecklistHint' },
    { kind: 'table', icon: 'sheet', hint: 'templates.kindTableHint' }
  ] as const;

  // table: optionally start with the sheets of an Excel file (name suggested from the file)
  let file = $state<File | null>(null);
  function pickFile(f: File | undefined) {
    file = f ?? null;
    if (f && !name.trim()) name = titleFromFile(f.name);
  }

  async function submit(e: SubmitEvent) {
    e.preventDefault();
    if (!name.trim()) return;
    busy = true;
    try {
      const sheets = kind === 'table' && file ? await readXlsx(file) : null;
      if (kind === 'table' && file && !sheets) return; // not readable (toast shown)
      const tpl = await api.post<Template>('/templates', { name, kind });
      if (sheets) {
        try {
          await api.put(`/templates/${tpl.id}/sheet`, { data: sheets });
        } catch (err) {
          await api.del(`/templates/${tpl.id}`).catch(() => {}); // no empty template left behind
          throw err;
        }
      }
      if (category) await api.patch(`/templates/${tpl.id}`, { category_id: category });
      onclose();
      goto(`/templates/${tpl.id}`);
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
  class="m-auto w-[min(30rem,calc(100vw-2rem))] rounded-xl border border-zinc-200 bg-white p-0 text-zinc-900 shadow-xl backdrop:bg-zinc-950/30 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-100"
  data-testid="new-template-dialog">
  <form onsubmit={submit} class="space-y-4 p-5">
    <div class="flex items-center gap-3">
      <EntityIcon type="template" size="md" />
      <h2 class="min-w-0 flex-1 text-base font-semibold">{t('templates.newTitle')}</h2>
      <button type="button" class="btn-ghost px-1.5!" onclick={() => dialog?.close()} aria-label={t('common.cancel')}><Icon name="x" size={16} /></button>
    </div>

    <label class="block">
      <span class="label">{t('common.name')}</span>
      <!-- svelte-ignore a11y_autofocus -->
      <input class="input" bind:value={name} required maxlength="200" autofocus />
    </label>

    <fieldset>
      <legend class="label">{t('templates.kindLabel')}</legend>
      <div class="grid grid-cols-2 gap-2">
        {#each KINDS as k (k.kind)}
          <label
            class="flex cursor-pointer items-start gap-2.5 rounded-lg border p-3 text-sm transition-colors {kind === k.kind
              ? 'border-accent-500 bg-accent-50 dark:border-accent-400 dark:bg-accent-950/40'
              : 'border-zinc-200 hover:bg-zinc-50 dark:border-zinc-700 dark:hover:bg-zinc-800/60'}">
            <input type="radio" name="kind" value={k.kind} bind:group={kind} class="sr-only" />
            <Icon name={k.icon} size={18} class="mt-0.5 shrink-0 {kind === k.kind ? 'text-accent-600 dark:text-accent-400' : 'text-zinc-400'}" />
            <span class="min-w-0">
              <span class="block font-medium">{t(`templates.kind.${k.kind}`)}</span>
              <span class="block text-xs text-zinc-500">{t(k.hint)}</span>
            </span>
          </label>
        {/each}
      </div>
    </fieldset>

    {#if kind === 'table'}
      <label class="block">
        <span class="label">{t('templates.xlsxFile')}</span>
        <input
          type="file"
          accept=".xlsx,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
          class="block w-full text-sm text-zinc-600 file:mr-3 file:rounded-md file:border file:border-zinc-200 file:bg-white file:px-3 file:py-1.5 file:text-sm file:font-medium hover:file:bg-zinc-50 dark:text-zinc-300 dark:file:border-zinc-700 dark:file:bg-zinc-800 dark:file:text-zinc-200"
          onchange={(e) => pickFile(e.currentTarget.files?.[0])}
          data-testid="new-template-file" />
        <span class="mt-1 block text-xs text-zinc-500">{t('templates.xlsxHint')}</span>
      </label>
    {/if}

    {#if categories.length}
      <label class="block">
        <span class="label">{t('cat.title')}</span>
        <select class="input" bind:value={category}>
          <option value={0}>{t('cat.none')}</option>
          {#each categories as c (c.id)}<option value={c.id}>{c.name}</option>{/each}
        </select>
      </label>
    {/if}

    <div class="flex justify-end gap-2 pt-1">
      <button type="button" class="btn-ghost" onclick={() => dialog?.close()}>{t('common.cancel')}</button>
      <button type="submit" class="btn-primary" disabled={busy || !name.trim()} data-testid="new-template-create">{t('common.create')}</button>
    </div>
  </form>
</dialog>
