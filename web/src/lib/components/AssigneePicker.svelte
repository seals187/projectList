<script lang="ts">
  // Compact picker for the assigned person (e.g. for a whole checklist).
  // An invisible <select> sits over the display: native controls on
  // desktop and mobile, without a custom dropdown.
  import { t } from '$lib/i18n.svelte';
  import type { Assignee } from '$lib/types';
  import Avatar from './Avatar.svelte';
  import Icon from './Icon.svelte';

  let {
    assignees,
    value,
    name,
    disabled = false,
    compact = false,
    onchange
  }: {
    assignees: Assignee[];
    value: number | null;
    name: string | null;
    disabled?: boolean;
    compact?: boolean;
    onchange: (id: number | null) => void;
  } = $props();
</script>

<label
  class="relative inline-flex max-w-full items-center gap-1.5 rounded-md text-[13px] whitespace-nowrap
    {compact ? 'px-1 py-0.5' : 'border border-zinc-200 px-2 py-0.5 dark:border-zinc-700'}
    {disabled ? '' : 'cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-800/60'}
    {value ? 'text-zinc-700 dark:text-zinc-200' : 'text-zinc-400'}"
  title={t('checklist.assignChecklist')}>
  {#if value && name}
    <Avatar {name} id={value} size={18} />
    <span class="truncate">{name}</span>
  {:else if !disabled}
    <Icon name="user" size={14} />
    <span>{t('checklist.assignChecklist')}</span>
  {:else}
    <Icon name="user" size={14} />
    <span>{t('checklist.nobody')}</span>
  {/if}
  {#if !disabled}<Icon name="chevronDown" size={12} class="shrink-0 opacity-60" />{/if}
  <select
    class="absolute inset-0 cursor-pointer opacity-0 disabled:cursor-default"
    {disabled}
    value={value ?? ''}
    aria-label={t('checklist.assignChecklist')}
    onchange={(e) => {
      const v = e.currentTarget.value;
      onchange(v ? Number(v) : null);
    }}>
    <option value="">{t('checklist.nobody')}</option>
    {#each assignees as a (a.id)}<option value={a.id}>{a.name}</option>{/each}
  </select>
</label>
