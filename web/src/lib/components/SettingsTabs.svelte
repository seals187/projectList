<script lang="ts">
  import { page } from '$app/state';
  import { isAdmin } from '$lib/state.svelte';
  import { t } from '$lib/i18n.svelte';

  const tabs = $derived([
    { href: '/settings', label: t('settings.general') },
    ...(isAdmin() ? [{ href: '/settings/users', label: t('settings.users') }] : [])
  ]);
</script>

<nav class="mb-8 flex gap-1 border-b border-zinc-200 dark:border-zinc-800">
  {#each tabs as tab (tab.href)}
    <a
      href={tab.href}
      class="-mb-px border-b-2 px-3 py-2 text-sm {page.url.pathname === tab.href
        ? 'border-accent-600 font-medium text-accent-700 dark:text-accent-400'
        : 'border-transparent text-zinc-500 hover:text-zinc-800 dark:hover:text-zinc-200'}">{tab.label}</a>
  {/each}
</nav>
