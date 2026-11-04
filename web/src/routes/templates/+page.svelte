<script lang="ts">
  import { goto } from '$app/navigation';
  import { api, errorText } from '$lib/api';
  import { canWrite, toast } from '$lib/state.svelte';
  import { t } from '$lib/i18n.svelte';
  import type { Template, TemplateSummary } from '$lib/types';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import Icon from '$lib/components/Icon.svelte';

  let templates = $state<TemplateSummary[] | null>(null);
  let newName = $state('');

  $effect(() => {
    api.get<TemplateSummary[]>('/templates').then(
      (r) => (templates = r),
      (e) => toast(errorText(e))
    );
  });

  async function create(e: SubmitEvent) {
    e.preventDefault();
    if (!newName.trim()) return;
    try {
      const tpl = await api.post<Template>('/templates', { name: newName });
      goto(`/templates/${tpl.id}`);
    } catch (err) {
      toast(errorText(err));
    }
  }
</script>

<svelte:head><title>{t('nav.templates')} · ProjectList</title></svelte:head>

<PageHeader>
  {#snippet title()}{t('nav.templates')}{/snippet}
</PageHeader>
<p class="-mt-4 mb-6 text-sm text-zinc-500">{t('templates.subtitle')}</p>

<ul>
  {#if templates === null}
    <li class="py-2 text-sm text-zinc-400">{t('common.loading')}</li>
  {:else}
    {#each templates as tpl (tpl.id)}
      <li>
        <a href="/templates/{tpl.id}" class="row-link">
          <Icon name="template" class="text-zinc-400" />
          <div class="min-w-0 flex-1">
            <p class="truncate font-medium">{tpl.name}</p>
            {#if tpl.description}<p class="truncate text-xs text-zinc-400">{tpl.description}</p>{/if}
          </div>
          <span class="text-xs text-zinc-400">{t('templates.items', { n: tpl.item_count })}</span>
        </a>
      </li>
    {:else}
      <li class="py-2 text-sm text-zinc-400">{t('templates.empty')}</li>
    {/each}
  {/if}
</ul>
{#if canWrite()}
  <form onsubmit={create} class="mt-1 flex items-center gap-2 py-1">
    <Icon name="plus" class="text-zinc-400" />
    <input class="flex-1 bg-transparent py-1 text-sm outline-none placeholder:text-zinc-400" placeholder={t('templates.new')} bind:value={newName} />
    {#if newName.trim()}<button class="btn-primary py-1!" type="submit">{t('common.create')}</button>{/if}
  </form>
{/if}
