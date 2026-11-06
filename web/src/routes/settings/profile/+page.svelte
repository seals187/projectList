<script lang="ts">
  // Settings → Profile & account: own name/e-mail/role, sign out, change password.
  import { goto } from '$app/navigation';
  import { api, errorText } from '$lib/api';
  import { t } from '$lib/i18n.svelte';
  import { session, toast } from '$lib/state.svelte';
  import SettingsShell from '$lib/components/SettingsShell.svelte';
  import SetCard from '$lib/components/SetCard.svelte';
  import SetRow from '$lib/components/SetRow.svelte';
  import Avatar from '$lib/components/Avatar.svelte';
  import Icon from '$lib/components/Icon.svelte';

  let current = $state('');
  let next = $state('');
  let busy = $state(false);

  async function changePassword(e: SubmitEvent) {
    e.preventDefault();
    busy = true;
    try {
      await api.put('/auth/password', { current_password: current, new_password: next });
      current = next = '';
      toast(t('settings.passwordChanged'), 'info');
    } catch (err) {
      toast(errorText(err));
    } finally {
      busy = false;
    }
  }

  async function logout() {
    try {
      await api.post('/auth/logout');
    } finally {
      session.user = null;
      goto('/login');
    }
  }
</script>

<SettingsShell title={t('set.nav.profile')} subtitle={t('set.profile.subtitle')}>
  <div class="max-w-[680px]">
    <SetCard>
      <div class="flex flex-wrap items-center gap-4 px-[18px] py-4">
        <Avatar name={session.user?.name ?? ''} id={session.user?.id ?? 0} size={48} />
        <div class="min-w-0 flex-1">
          <p class="truncate text-[17px] font-semibold">{session.user?.name}</p>
          <p class="truncate text-[13.5px] text-zinc-500">{session.user?.email} · {t(`role.${session.user?.role ?? 'viewer'}`)}</p>
        </div>
        <button type="button" class="btn-outline" onclick={logout}><Icon name="logout" size={14} /> {t('nav.logout')}</button>
      </div>
    </SetCard>

    <form onsubmit={changePassword}>
      <SetCard title={t('settings.password')}>
        <SetRow title={t('settings.currentPassword')} labelFor="cur">
          <input id="cur" type="password" autocomplete="current-password" class="input w-64!" required bind:value={current} />
        </SetRow>
        <SetRow title={t('settings.newPassword')} labelFor="new">
          <input id="new" type="password" autocomplete="new-password" minlength="8" class="input w-64!" required bind:value={next} />
        </SetRow>
        <div class="flex justify-end px-[18px] py-3">
          <button class="btn-primary" type="submit" disabled={busy || next.length < 8 || !current}>{t('common.save')}</button>
        </div>
      </SetCard>
    </form>
  </div>
</SettingsShell>
