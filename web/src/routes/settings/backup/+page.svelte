<script lang="ts">
  // Settings → Backup (admin): download a consistent copy of the database.
  import { goto } from '$app/navigation';
  import { t } from '$lib/i18n.svelte';
  import { isAdmin } from '$lib/state.svelte';
  import SettingsShell from '$lib/components/SettingsShell.svelte';
  import SetCard from '$lib/components/SetCard.svelte';
  import SetRow from '$lib/components/SetRow.svelte';
  import Icon from '$lib/components/Icon.svelte';

  $effect(() => {
    if (!isAdmin()) goto('/settings');
  });
</script>

<SettingsShell title={t('set.nav.backup')} subtitle={t('set.backup.subtitle')}>
  <div class="max-w-[680px]">
    <SetCard>
      <SetRow title={t('set.backup.db')} desc={t('settings.backupHint')}>
        <a href="/api/admin/backup" class="btn-outline" download><Icon name="download" size={14} /> {t('settings.backupDownload')}</a>
      </SetRow>
    </SetCard>
  </div>
</SettingsShell>
