<script lang="ts">
  // Invite a user (admin): name, e-mail, role. The server creates the account
  // and sends the invitation link by e-mail (if configured); the link is shown
  // afterwards in any case so it can also be passed on by hand.
  import { api, errorText } from '$lib/api';
  import { i18n, t } from '$lib/i18n.svelte';
  import { toast } from '$lib/state.svelte';
  import type { AdminUser, Role } from '$lib/types';
  import Icon from './Icon.svelte';
  import LinkBox from './LinkBox.svelte';

  let { onclose, oncreated }: { onclose: () => void; oncreated: (u: AdminUser) => void } = $props();

  let name = $state('');
  let email = $state('');
  let role = $state<Role>('member');
  let busy = $state(false);
  let result = $state<{ link: string; emailed: boolean } | null>(null);
  let dialog = $state<HTMLDialogElement>();

  $effect(() => {
    dialog?.showModal();
  });

  async function submit(e: SubmitEvent) {
    e.preventDefault();
    busy = true;
    try {
      const r = await api.post<{ user: AdminUser; link: string; emailed: boolean }>('/users/invite', { name, email, role, customer_ids: [], lang: i18n.lang });
      oncreated(r.user);
      result = { link: r.link, emailed: r.emailed };
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
  class="m-auto w-[min(30rem,calc(100vw-2rem))] rounded-2xl border border-zinc-200 bg-white p-0 text-zinc-900 shadow-xl backdrop:bg-zinc-950/30 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-100">
  <div class="space-y-4 p-6">
    <div class="flex items-center justify-between">
      <h2 class="text-[17px] font-semibold">{t('set.users.invite')}</h2>
      <button type="button" class="btn-ghost px-1.5!" onclick={() => dialog?.close()} aria-label={t('common.close')}><Icon name="x" size={16} /></button>
    </div>
    {#if result}
      <LinkBox link={result.link} emailed={result.emailed} {email} />
      <div class="flex justify-end"><button type="button" class="btn-primary" onclick={() => dialog?.close()}>{t('common.done')}</button></div>
    {:else}
      <form onsubmit={submit} class="space-y-4">
        <p class="text-[13.5px] text-zinc-500">{t('set.users.inviteHint')}</p>
        <label class="block">
          <span class="label">{t('common.name')}</span>
          <!-- svelte-ignore a11y_autofocus -->
          <input class="input" bind:value={name} required maxlength="100" autofocus />
        </label>
        <label class="block">
          <span class="label">{t('common.email')}</span>
          <input class="input" type="email" bind:value={email} required maxlength="254" />
        </label>
        <label class="block">
          <span class="label">{t('common.role')}</span>
          <select class="input" bind:value={role}>
            <option value="member">{t('role.member')}</option><option value="viewer">{t('role.viewer')}</option><option value="admin">{t('role.admin')}</option>
          </select>
        </label>
        <div class="flex justify-end gap-2 pt-1">
          <button type="button" class="btn-ghost" onclick={() => dialog?.close()}>{t('common.cancel')}</button>
          <button type="submit" class="btn-primary" disabled={busy || !name.trim() || !email.trim()}><Icon name="mail" size={14} />{t('set.users.sendInvite')}</button>
        </div>
      </form>
    {/if}
  </div>
</dialog>
