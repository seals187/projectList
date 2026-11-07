<script lang="ts">
  // Activity log of a project (newest first): avatar, person, what, when.
  import { fmtDateTime, shortName, t } from '$lib/i18n.svelte';
  import { activityText, activityType, groupActivity, relativeTime } from '$lib/activity';
  import EntityIcon from './EntityIcon.svelte';
  import type { ActivityEntry } from '$lib/types';
  import Avatar from './Avatar.svelte';
  import Icon from './Icon.svelte';

  let { entries, limit = 0, compact = false }: { entries: ActivityEntry[] | null; limit?: number; compact?: boolean } = $props();

  // entries written by an automation rule (no user)
  const isAuto = (a: ActivityEntry) => !a.user_id && a.detail === 'automation';
  const groups = $derived.by(() => {
    const g = groupActivity(entries ?? []);
    return limit ? g.slice(0, limit) : g;
  });
</script>

{#if entries === null}
  <p class="py-2 text-sm text-zinc-400">{t('common.loading')}</p>
{:else if !groups.length}
  <p class="py-2 text-sm text-zinc-400">{t('activity.none')}</p>
{:else}
  <ul class={compact ? 'space-y-3' : 'divide-y divide-zinc-100 dark:divide-zinc-800'}>
    {#each groups as a (a.id)}
      <li class="flex items-start gap-2.5 {compact ? '' : 'py-2.5'}">
        {#if a.user_id}
          <Avatar name={a.user_name} id={a.user_id} size={compact ? 20 : 24} />
        {:else if isAuto(a)}
          <span class="flex shrink-0 items-center justify-center rounded-full bg-accent-100 text-accent-700 dark:bg-accent-950/60 dark:text-accent-300" style:width="{compact ? 20 : 24}px" style:height="{compact ? 20 : 24}px"><Icon name="zap" size={compact ? 11 : 13} /></span>
        {:else}
          <span class="flex shrink-0 items-center justify-center rounded-full bg-zinc-200 text-[10px] text-zinc-500 dark:bg-zinc-800" style:width="{compact ? 20 : 24}px" style:height="{compact ? 20 : 24}px">?</span>
        {/if}
        <EntityIcon type={activityType(a.kind)} size="xs" class="mt-0.5" />
        <p class="min-w-0 flex-1 text-sm leading-snug {compact ? 'text-[13px]' : ''}">
          <span class="font-semibold">{a.user_id ? shortName(a.user_name) : isAuto(a) ? t('activity.automation') : t('activity.deletedUser')}</span>
          <span class="text-zinc-600 dark:text-zinc-300">{activityText(a)}</span>
        </p>
        <time class="shrink-0 text-xs whitespace-nowrap text-zinc-400" datetime={a.created_at} title={fmtDateTime(a.created_at)}>{relativeTime(a.created_at)}</time>
      </li>
    {/each}
  </ul>
{/if}
