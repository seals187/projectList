<script lang="ts">
  // Schalter „zählt zum Kundenfortschritt“ für ein Projekt (Tortendiagramm-Icon).
  // An: lila auf hellem Lila; aus: grau. Admins und Members ändern per Klick
  // (sofort sichtbar, bei Fehler zurück), Viewer sehen nur den Zustand.
  import { api, errorText } from '$lib/api';
  import { canWrite, toast } from '$lib/state.svelte';
  import { t } from '$lib/i18n.svelte';
  import { customerLabel } from '$lib/projectStatus';
  import type { SidebarProject } from '$lib/types';
  import Icon from './Icon.svelte';

  let { project, customerName }: { project: SidebarProject; customerName: string } = $props();

  const on = $derived(project.include_in_customer_progress);
  const editable = $derived(canWrite());
  const title = $derived(t(on ? 'progress.inTitle' : 'progress.outTitle'));
  const text = $derived(
    t(on ? 'progress.inText' : 'progress.outText', { customer: customerLabel(customerName) }) +
      (editable ? ' ' + t(on ? 'progress.clickExclude' : 'progress.clickInclude') : '')
  );

  async function toggle(e: MouseEvent) {
    e.preventDefault(); // liegt in Links/Zeilen: Projekt nicht öffnen
    e.stopPropagation();
    if (!editable) return;
    const next = !on;
    project.include_in_customer_progress = next;
    try {
      await api.patch(`/projects/${project.id}`, { include_in_customer_progress: next });
    } catch (err) {
      project.include_in_customer_progress = !next;
      toast(errorText(err));
    }
  }
</script>

<span class="group/pt relative inline-flex">
  <button
    type="button"
    class="flex size-7 items-center justify-center rounded-md transition-colors
      {on ? 'bg-accent-50 text-accent-600 dark:bg-accent-950/60 dark:text-accent-400' : 'text-zinc-400 ring-1 ring-zinc-200 ring-inset dark:text-zinc-500 dark:ring-zinc-700'}
      {editable ? (on ? 'hover:bg-accent-100 dark:hover:bg-accent-900/60' : 'hover:bg-zinc-100 hover:text-zinc-600 dark:hover:bg-zinc-800') : 'cursor-default'}"
    onclick={toggle}
    aria-pressed={on}
    aria-disabled={!editable}
    aria-label="{title}: {text}">
    <Icon name={on ? 'pieChart' : 'pieChartOff'} size={15} />
  </button>
  <!-- Tooltip -->
  <span
    role="tooltip"
    class="pointer-events-none absolute top-full right-0 z-30 mt-2 hidden w-60 rounded-lg bg-zinc-900 px-3 py-2 text-left text-xs leading-snug text-zinc-200 shadow-lg group-hover/pt:block group-focus-within/pt:block dark:bg-zinc-700">
    <span class="block font-semibold text-white">{title}</span>
    {text}
  </span>
</span>
