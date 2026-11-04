<script lang="ts">
  // Watchers of a project: avatars; click opens a checklist of the people with access.
  import { t } from '$lib/i18n.svelte';
  import type { Assignee } from '$lib/types';
  import ContextMenu from './ContextMenu.svelte';
  import Avatar from './Avatar.svelte';
  import Icon from './Icon.svelte';

  let {
    people,
    value,
    disabled = false,
    onchange
  }: { people: Assignee[]; value: { id: number; name: string }[]; disabled?: boolean; onchange: (ids: number[]) => void } = $props();

  let menu = $state<{ x: number; y: number } | null>(null);
  const ids = $derived(value.map((w) => w.id));
  const toggle = (id: number) => onchange(ids.includes(id) ? ids.filter((x) => x !== id) : [...ids, id]);
</script>

<button
  type="button"
  class="-mx-1 flex min-h-7 items-center rounded-md px-1 text-left {disabled ? 'cursor-default' : 'hover:bg-zinc-100 dark:hover:bg-zinc-800'}"
  {disabled}
  title={value.map((w) => w.name).join(', ')}
  onclick={(e) => {
    const r = e.currentTarget.getBoundingClientRect();
    menu = menu ? null : { x: r.left, y: r.bottom + 4 };
  }}>
  {#if value.length}
    <span class="flex -space-x-1.5">
      {#each value.slice(0, 6) as w (w.id)}<span class="rounded-full ring-2 ring-white dark:ring-zinc-900"><Avatar name={w.name} id={w.id} size={22} /></span>{/each}
    </span>
    {#if value.length > 6}<span class="ml-1.5 text-xs text-zinc-500">+{value.length - 6}</span>{/if}
  {:else}
    <span class="text-sm text-zinc-400">{disabled ? '–' : t('watchers.add')}</span>
  {/if}
</button>

{#if menu}
  <ContextMenu x={menu.x} y={menu.y} label={t('watchers.title')} onclose={() => (menu = null)}>
    <p class="px-2 pt-1 pb-1.5 text-xs text-zinc-500">{t('watchers.hint')}</p>
    {#each people as p (p.id)}
      {@const on = ids.includes(p.id)}
      <button type="button" role="menuitemcheckbox" aria-checked={on} class="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-[13px] hover:bg-zinc-100 dark:hover:bg-zinc-800" onclick={() => toggle(p.id)}>
        <span class="flex size-4 shrink-0 items-center justify-center rounded border {on ? 'border-accent-600 bg-accent-600 text-white' : 'border-zinc-300 dark:border-zinc-600'}">
          {#if on}<Icon name="check" size={11} />{/if}
        </span>
        <Avatar name={p.name} id={p.id} size={18} />
        <span class="min-w-0 flex-1 truncate">{p.name}</span>
      </button>
    {:else}
      <p class="px-2 py-1.5 text-[13px] text-zinc-400">{t('filter.noEntries')}</p>
    {/each}
  </ContextMenu>
{/if}
