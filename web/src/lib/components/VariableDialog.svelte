<script lang="ts">
  // Create or edit a custom template variable (admins): name ({{name}}),
  // description, default value, source (asked when applying or delivered by an
  // integration). Existing variables can be deleted.
  import { untrack } from 'svelte';
  import { api, errorText } from '$lib/api';
  import { toast } from '$lib/state.svelte';
  import { t, type Key } from '$lib/i18n.svelte';
  import { validVariableName } from '$lib/templateLibrary';
  import type { TemplateVariable } from '$lib/types';
  import Icon from './Icon.svelte';

  let { variable = null, onsaved, onclose }: { variable?: TemplateVariable | null; onsaved: () => void; onclose: () => void } = $props();

  const SOURCES = ['custom', 'easyredmine', 'otobo'];
  const init = untrack(() => variable);
  let name = $state(init?.name ?? '');
  let description = $state(init?.description ?? '');
  let defaultValue = $state(init?.default_value ?? '');
  let source = $state(init?.source ?? 'custom');
  let busy = $state(false);
  let confirmDel = $state(false);
  let dialog = $state<HTMLDialogElement>();
  const nameOk = $derived(validVariableName(name.trim()));

  $effect(() => {
    dialog?.showModal();
  });

  async function save(e: SubmitEvent) {
    e.preventDefault();
    if (!nameOk) return;
    busy = true;
    const body = { name: name.trim(), description, default_value: defaultValue, source };
    try {
      if (init?.id) await api.patch(`/template-variables/${init.id}`, body);
      else await api.post('/template-variables', body);
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
      await api.del(`/template-variables/${init!.id}`);
      onsaved();
      dialog?.close();
    } catch (err) {
      toast(errorText(err));
    }
  }
</script>

<dialog
  bind:this={dialog}
  {onclose}
  onclick={(e) => e.target === dialog && dialog?.close()}
  class="m-auto w-[min(28rem,calc(100vw-2rem))] rounded-xl border border-zinc-200 bg-white p-0 text-zinc-900 shadow-xl backdrop:bg-zinc-950/30 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-100"
  data-testid="variable-dialog">
  <form onsubmit={save} class="space-y-4 p-5">
    <div class="flex items-center justify-between">
      <h2 class="text-base font-semibold">{t(init ? 'var.edit' : 'var.new')}</h2>
      <button type="button" class="btn-ghost px-1.5!" onclick={() => dialog?.close()} aria-label={t('common.cancel')}><Icon name="x" size={16} /></button>
    </div>
    <label class="block">
      <span class="label">{t('var.name')}</span>
      <div class="flex items-center gap-1 font-mono text-sm">
        <span class="text-zinc-400">{'{{'}</span>
        <!-- svelte-ignore a11y_autofocus -->
        <input class="input font-mono" bind:value={name} maxlength="60" placeholder="server.hostname" autofocus required />
        <span class="text-zinc-400">{'}}'}</span>
      </div>
      {#if name && !nameOk}<span class="mt-1 block text-xs text-red-600 dark:text-red-400">{t('var.nameRule')}</span>{/if}
    </label>
    <label class="block">
      <span class="label">{t('var.description')}</span>
      <input class="input" bind:value={description} maxlength="200" />
    </label>
    <label class="block">
      <span class="label">{t('var.source')}</span>
      <select class="input" bind:value={source}>
        {#each SOURCES as s (s)}<option value={s}>{t(`var.source.${s}` as Key)}</option>{/each}
      </select>
      <span class="mt-1 block text-xs text-zinc-500">{t(source === 'custom' ? 'var.sourceHint.custom' : 'var.sourceHint.integration')}</span>
    </label>
    <label class="block">
      <span class="label">{t('var.default')}</span>
      <input class="input" bind:value={defaultValue} maxlength="500" placeholder={t('var.defaultPlaceholder')} />
    </label>
    <div class="flex items-center gap-2 pt-1">
      {#if init?.id}
        {#if confirmDel}
          <button type="button" class="btn bg-red-600 text-white hover:bg-red-700" onclick={remove}>{t('var.deleteConfirm')}</button>
          <button type="button" class="btn-ghost" onclick={() => (confirmDel = false)}>{t('common.cancel')}</button>
        {:else}
          <button type="button" class="btn-danger" onclick={() => (confirmDel = true)}><Icon name="trash" size={14} />{t('common.delete')}</button>
        {/if}
      {/if}
      <span class="flex-1"></span>
      <button type="button" class="btn-ghost" onclick={() => dialog?.close()}>{t('common.cancel')}</button>
      <button type="submit" class="btn-primary" disabled={busy || !nameOk}>{t('common.save')}</button>
    </div>
  </form>
</dialog>
