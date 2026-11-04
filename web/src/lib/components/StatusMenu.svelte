<script lang="ts">
  // Board column menu (admins): name, color and WIP limit of a project status,
  // for everyone. Empty name / no color / limit 0 = default.
  import { api, errorText } from '$lib/api';
  import { t } from '$lib/i18n.svelte';
  import { sidebar, toast } from '$lib/state.svelte';
  import { statusColor, statusDotCls, statusDotStyle } from '$lib/projectStatus';
  import type { ProjectStatus } from '$lib/types';
  import ContextMenu from './ContextMenu.svelte';
  import ColorPicker from './ColorPicker.svelte';
  import Icon from './Icon.svelte';

  let { status, x, y, onclose }: { status: ProjectStatus; x: number; y: number; onclose: () => void } = $props();

  // svelte-ignore state_referenced_locally
  let name = $state(sidebar.statusLabels[status] ?? '');
  // svelte-ignore state_referenced_locally
  let wip = $state<number | null>(sidebar.statusWip[status] ?? null);
  const fallback = $derived(t(`status.${status}`));

  const curLabel = () => sidebar.statusLabels[status] ?? '';
  const curWip = () => sidebar.statusWip[status] ?? 0;
  const wipValue = () => Math.max(0, Math.min(999, Math.floor(wip ?? 0)));

  async function save(label: string, color: string, limit: number) {
    const prev = { label: sidebar.statusLabels[status], color: sidebar.statusColors[status], wip: sidebar.statusWip[status] };
    // show immediately, revert on error
    const set = <T,>(rec: Partial<Record<ProjectStatus, T>>, v: T | undefined) => {
      if (v) rec[status] = v;
      else delete rec[status];
    };
    set(sidebar.statusLabels, label);
    set(sidebar.statusColors, color);
    set(sidebar.statusWip, limit);
    try {
      await api.put(`/status-labels/${status}`, { label, color, wip_limit: limit });
    } catch (err) {
      set(sidebar.statusLabels, prev.label);
      set(sidebar.statusColors, prev.color);
      set(sidebar.statusWip, prev.wip);
      toast(errorText(err));
    }
  }

  // name and WIP limit: saved on Enter, blur or when the menu closes
  function saveFields() {
    const label = name.trim();
    const limit = wipValue();
    if (label !== curLabel() || limit !== curWip()) save(label, statusColor(status), limit);
  }
  const enter = (e: KeyboardEvent) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      saveFields();
      onclose();
    }
  };
</script>

<ContextMenu {x} {y} onclose={() => (saveFields(), onclose())} label={t('menu.statusSettings')}>
  <div class="space-y-3 p-1.5">
    <div>
      <label class="section-label mb-1 block" for="status-name">{t('menu.statusName')}</label>
      <div class="flex items-center gap-2">
        <span class="size-3 shrink-0 rounded-full {statusDotCls(status)}" style={statusDotStyle(status)}></span>
        <input id="status-name" class="input py-1! text-sm!" maxlength="40" placeholder={fallback} bind:value={name} onkeydown={enter} onblur={saveFields} />
      </div>
    </div>
    <div>
      <p class="section-label mb-1.5">{t('menu.statusColor')}</p>
      <ColorPicker value={statusColor(status)} onpick={(c) => save(name.trim(), c, wipValue())} />
    </div>
    <div>
      <label class="section-label mb-1 block" for="status-wip">{t('wip.title')}</label>
      <div class="flex items-center gap-2">
        <input id="status-wip" type="number" min="0" max="999" class="input w-20! py-1! text-sm!" placeholder="–" bind:value={wip} onkeydown={enter} onblur={saveFields} />
        <span class="text-xs text-zinc-500">{t('wip.hint')}</span>
      </div>
    </div>
    {#if sidebar.statusLabels[status] || statusColor(status) || sidebar.statusWip[status]}
      <button
        type="button"
        class="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-sm text-zinc-600 hover:bg-zinc-100 dark:text-zinc-300 dark:hover:bg-zinc-800"
        onclick={() => {
          name = '';
          wip = null;
          save('', '', 0);
        }}>
        <Icon name="undo" size={14} class="text-zinc-400" />{t('settings.resetDefaults')}
      </button>
    {/if}
  </div>
</ContextMenu>
