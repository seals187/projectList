<script lang="ts">
  // Settings → Overview: view the overview opens with.
  import { t } from '$lib/i18n.svelte';
  import { OVERVIEW_VIEWS, prefs, setOverviewDefault } from '$lib/state.svelte';
  import SettingsShell from '$lib/components/SettingsShell.svelte';
  import SetCard from '$lib/components/SetCard.svelte';
  import SetRow from '$lib/components/SetRow.svelte';
  import Seg from '$lib/components/Seg.svelte';

  const options = $derived(['last' as const, ...OVERVIEW_VIEWS].map((v) => ({ value: v, label: v === 'last' ? t('settings.lastUsedView') : t(`overview.view.${v}`) })));
</script>

<SettingsShell title={t('set.nav.overview')} subtitle={t('set.overview.subtitle')}>
  <div class="max-w-[760px]">
    <SetCard>
      <SetRow title={t('settings.defaultView')} desc={t('settings.defaultViewHint')}>
        <Seg label={t('settings.defaultView')} {options} value={prefs.overviewDefault} onchange={(v) => setOverviewDefault(v)} />
      </SetRow>
    </SetCard>
  </div>
</SettingsShell>
