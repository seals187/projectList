<script lang="ts">
  // Settings → About: version of the app and link to the source code.
  // App version from package.json (bumped with every release); if the server
  // build reports a different version (e.g. "dev" for a local build), it is shown too.
  import { version as appVersion } from '../../../../package.json';
  import { t } from '$lib/i18n.svelte';
  import { session } from '$lib/state.svelte';
  import SettingsShell from '$lib/components/SettingsShell.svelte';
  import SetCard from '$lib/components/SetCard.svelte';
  import SetRow from '$lib/components/SetRow.svelte';
  import Icon from '$lib/components/Icon.svelte';

  const REPO = 'https://github.com/seals187/projectList';
  const server = $derived(session.version.replace(/^v/, ''));
</script>

<SettingsShell title={t('set.nav.about')} subtitle={t('set.about.subtitle')}>
  <div class="max-w-[680px]">
    <SetCard>
      <div class="flex items-center gap-4 px-[18px] py-4">
        <img src="/favicon.svg" alt="" width="44" height="44" />
        <div class="min-w-0 flex-1">
          <p class="text-[18px] font-bold">ProjectList</p>
          <p class="text-[13.5px] text-zinc-500 dark:text-zinc-400">{t('set.about.tagline')}</p>
        </div>
      </div>
      <SetRow title={t('set.about.version')}>
        <span class="text-[15px] font-semibold tabular-nums" data-testid="app-version">{appVersion}</span>
        {#if server && server !== appVersion}
          <span class="text-[13px] text-zinc-400" title={t('set.about.serverBuild')}>({t('set.about.server')}: {server})</span>
        {/if}
      </SetRow>
      <SetRow title={t('set.about.source')} desc={t('set.about.sourceDesc')}>
        <a href={REPO} target="_blank" rel="noopener noreferrer" class="btn-outline" data-testid="github-link">
          <Icon name="github" size={15} />GitHub
          <Icon name="external" size={13} class="text-zinc-400" />
        </a>
      </SetRow>
      <SetRow title={t('set.about.changes')} desc={t('set.about.changesDesc')}>
        <a href="{REPO}/releases" target="_blank" rel="noopener noreferrer" class="btn-ghost">{t('set.about.releases')} <Icon name="external" size={13} class="text-zinc-400" /></a>
      </SetRow>
    </SetCard>
  </div>
</SettingsShell>
