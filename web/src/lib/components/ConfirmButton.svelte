<script lang="ts">
  // Two-step delete without a modal: 1st click asks for confirmation, 2nd click executes.
  import type { Snippet } from 'svelte';
  import { t } from '$lib/i18n.svelte';

  let {
    onconfirm,
    label,
    confirmLabel,
    class: cls = 'btn-danger',
    children
  }: {
    onconfirm: () => void;
    label?: string;
    confirmLabel?: string;
    class?: string;
    children?: Snippet;
  } = $props();

  let armed = $state(false);
  let timer: ReturnType<typeof setTimeout>;

  function click() {
    if (armed) {
      clearTimeout(timer);
      armed = false;
      onconfirm();
      return;
    }
    armed = true;
    timer = setTimeout(() => (armed = false), 3000);
  }
</script>

<!-- armed state via data attribute: its selector beats the caller's hover/dark classes
     (otherwise e.g. hover:bg-red-50! showed white text on light red while hovering) -->
<button
  type="button"
  class="{cls} data-armed:bg-red-600! data-armed:text-white! data-armed:hover:bg-red-700! data-armed:hover:text-white!"
  data-armed={armed || undefined}
  onclick={click}
  title={label ?? t('common.delete')}>
  {#if armed}{confirmLabel ?? t('common.confirmDelete')}{:else if children}{@render children()}{:else}{label ?? t('common.delete')}{/if}
</button>
