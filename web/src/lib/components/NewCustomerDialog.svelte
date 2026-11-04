<script lang="ts">
  // Neuen Kunden anlegen (Admin): Name, Ort, Ansprechpartner, E-Mail.
  import { goto } from '$app/navigation';
  import { api, errorText } from '$lib/api';
  import { refreshSidebar, toast } from '$lib/state.svelte';
  import { t } from '$lib/i18n.svelte';
  import type { Customer } from '$lib/types';
  import Icon from './Icon.svelte';

  let { onclose }: { onclose: () => void } = $props();

  let name = $state('');
  let location = $state('');
  let contactName = $state('');
  let contactEmail = $state('');
  let busy = $state(false);
  let dialog = $state<HTMLDialogElement>();

  $effect(() => {
    dialog?.showModal();
  });

  async function submit(e: SubmitEvent) {
    e.preventDefault();
    if (!name.trim()) return;
    busy = true;
    try {
      const c = await api.post<Customer>('/customers', { name, location, contact_name: contactName, contact_email: contactEmail });
      refreshSidebar(0);
      onclose();
      goto(`/customers/${c.id}`);
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
  class="m-auto w-[min(28rem,calc(100vw-2rem))] rounded-xl border border-zinc-200 bg-white p-0 text-zinc-900 shadow-xl backdrop:bg-zinc-950/30 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-100">
  <form onsubmit={submit} class="space-y-4 p-5">
    <div class="flex items-center justify-between">
      <h2 class="text-base font-semibold">{t('customers.new')}</h2>
      <button type="button" class="btn-ghost px-1.5!" onclick={() => dialog?.close()} aria-label={t('common.cancel')}><Icon name="x" size={16} /></button>
    </div>
    <label class="block">
      <span class="label">{t('project.name')}</span>
      <!-- svelte-ignore a11y_autofocus -->
      <input class="input" bind:value={name} required maxlength="200" placeholder={t('customers.namePlaceholder')} autofocus />
    </label>
    <label class="block">
      <span class="label">{t('customers.location')}</span>
      <input class="input" bind:value={location} maxlength="100" />
    </label>
    <div class="grid gap-3 sm:grid-cols-2">
      <label class="block">
        <span class="label">{t('customers.contact')}</span>
        <input class="input" bind:value={contactName} maxlength="100" />
      </label>
      <label class="block">
        <span class="label">{t('customers.email')}</span>
        <input class="input" type="email" bind:value={contactEmail} maxlength="254" />
      </label>
    </div>
    <div class="flex justify-end gap-2 pt-1">
      <button type="button" class="btn-ghost" onclick={() => dialog?.close()}>{t('common.cancel')}</button>
      <button type="submit" class="btn-primary" disabled={busy || !name.trim()}>{t('common.create')}</button>
    </div>
  </form>
</dialog>
