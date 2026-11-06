<script lang="ts">
  // Context menu of a customer (overview board): open, new project, board filter.
  import { goto } from '$app/navigation';
  import { t } from '$lib/i18n.svelte';
  import type { SidebarCustomer } from '$lib/types';
  import ContextMenu from './ContextMenu.svelte';
  import Icon from './Icon.svelte';
  import EntityIcon from './EntityIcon.svelte';

  let {
    customer,
    x,
    y,
    hidden,
    canCreate,
    onclose,
    oncreate,
    onfilter
  }: {
    customer: SidebarCustomer;
    x: number;
    y: number;
    hidden: boolean; // customer currently hidden on the board
    canCreate: boolean;
    onclose: () => void;
    oncreate: () => void;
    onfilter: (mode: 'only' | 'toggle' | 'all') => void;
  } = $props();

  const itemCls = 'flex w-full items-center gap-2.5 rounded-md px-2 py-1.5 text-left hover:bg-zinc-100 dark:hover:bg-zinc-800';
  // run the action first: closing the menu drops the customer reference in the parent
  const run = (fn: () => void) => () => {
    fn();
    onclose();
  };
</script>

<ContextMenu {x} {y} {onclose} label={customer.name}>
  <div class="mb-1 flex items-center gap-2.5 border-b border-zinc-100 px-2 pt-1 pb-2.5 dark:border-zinc-800">
    <EntityIcon type="customer" size="sm" />
    <div class="min-w-0 flex-1">
      <p class="truncate font-semibold">{customer.name}</p>
      <p class="text-xs text-zinc-500">{t('overview.nProjects', { n: customer.projects.length })}</p>
    </div>
  </div>
  <button type="button" class={itemCls} role="menuitem" onclick={run(() => goto(`/customers/${customer.id}`))}>
    <Icon name="external" size={15} class="text-zinc-400" />{t('menu.openCustomer')}
  </button>
  {#if canCreate}
    <button type="button" class={itemCls} role="menuitem" onclick={run(oncreate)}>
      <Icon name="plus" size={15} class="text-zinc-400" />{t('overview.newProject')}
    </button>
  {/if}
  <div class="my-1 border-t border-zinc-100 dark:border-zinc-800"></div>
  <button type="button" class={itemCls} role="menuitem" onclick={run(() => onfilter('only'))}>
    <Icon name="search" size={15} class="text-zinc-400" />{t('menu.onlyCustomer')}
  </button>
  <button type="button" class={itemCls} role="menuitem" onclick={run(() => onfilter('toggle'))}>
    <Icon name={hidden ? 'plus' : 'minus'} size={15} class="text-zinc-400" />{t(hidden ? 'menu.showCustomer' : 'menu.hideCustomer')}
  </button>
  <button type="button" class={itemCls} role="menuitem" onclick={run(() => onfilter('all'))}>
    <Icon name="users" size={15} class="text-zinc-400" />{t('menu.allCustomers')}
  </button>
</ContextMenu>
