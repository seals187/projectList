<script lang="ts">
  // "My tasks" list with check-off (optimistic) and context link.
  import { api, errorText } from '$lib/api';
  import { canWrite, checkRadius, refreshSidebar, session, toast } from '$lib/state.svelte';
  import { fmtDate, t, todayISO } from '$lib/i18n.svelte';
  import { sanitizeHtml } from '$lib/rich';
  import type { Task } from '$lib/types';
  import Icon from './Icon.svelte';

  let { tasks = $bindable() }: { tasks: Task[] } = $props();
  const today = todayISO();

  async function toggle(task: Task) {
    const done = !task.done_at;
    const prev = task.done_at;
    task.done_at = done ? new Date().toISOString() : null;
    try {
      await api.post(`/items/${task.id}/done`, { done });
      refreshSidebar();
    } catch (e) {
      task.done_at = prev;
      toast(errorText(e));
    }
  }
</script>

<ul>
  {#each tasks as task (task.id)}
    {@const done = !!task.done_at}
    <li class="-mx-2 flex items-start gap-3 rounded-md px-2 py-1.5 hover:bg-zinc-50 dark:hover:bg-zinc-900/70">
      <button
        type="button"
        role="checkbox"
        aria-checked={done}
        aria-label={t('checklist.markDone')}
        disabled={!canWrite() || !session.user}
        onclick={() => toggle(task)}
        class="mt-[3px] flex size-4 shrink-0 items-center justify-center border transition-colors {checkRadius()} {done
          ? 'border-accent-600 bg-accent-600 text-white'
          : 'border-zinc-500 hover:border-accent-500 dark:border-zinc-400'}">
        {#if done}<Icon name="check" size={10} class="stroke-[3]" />{/if}
      </button>
      <div class="min-w-0 flex-1">
        <div class="rich text-sm leading-6 {done ? 'text-zinc-600 line-through dark:text-zinc-400' : ''}">{@html sanitizeHtml(task.title)}</div>
        <a href="/checklists/{task.checklist_id}" class="block truncate text-xs text-zinc-400 hover:text-accent-600">
          {task.customer_name} › {task.project_name} › {task.checklist_title}
        </a>
      </div>
      {#if task.due_date}
        <span class="mt-0.5 shrink-0 text-xs tabular-nums">
          {#if task.due_date === today}
            <span class="rounded bg-red-50 px-1.5 py-px text-[11px] font-medium text-red-600 dark:bg-red-950/50 dark:text-red-400">{t('checklist.today')}</span>
          {:else}
            <span class={!done && task.due_date < today ? 'font-medium text-red-600 dark:text-red-400' : 'text-zinc-500'}>{fmtDate(task.due_date)}</span>
          {/if}
        </span>
      {/if}
    </li>
  {/each}
</ul>
