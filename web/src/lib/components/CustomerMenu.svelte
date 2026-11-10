<script lang="ts">
  // Context menu of a customer (overview): open, new project, board filter and
  // (Cards, List) the card color of that view.
  import { goto } from '$app/navigation';
  import { t } from '$lib/i18n.svelte';
  import type { SidebarCustomer } from '$lib/types';
  import { sidebar } from '$lib/state.svelte';
  import { myColors, setMyColor } from '$lib/myColors.svelte';
  import { isFavorite, toggleFavorite } from '$lib/navMemory.svelte';
  import { cardHex, customerCardColor } from '$lib/cardColors';
  import CardColorPicker from './CardColorPicker.svelte';
  import ContextMenu from './ContextMenu.svelte';
  import Icon from './Icon.svelte';
  import EntityIcon from './EntityIcon.svelte';

  let {
    customer,
    x,
    y,
    hidden = false,
    canCreate,
    onclose,
    oncreate,
    onfilter,
    ondelete,
    colorView = null
  }: {
    customer: SidebarCustomer;
    x: number;
    y: number;
    hidden?: boolean; // customer currently hidden on the board
    canCreate: boolean;
    onclose: () => void;
    oncreate: () => void;
    onfilter?: (mode: 'only' | 'toggle' | 'all') => void; // overview only: customer filter
    colorView?: 'cards' | 'list' | null; // Cards / List: offer the personal card color of that view
    ondelete?: () => void; // set by the caller for admins: "Delete customer" (two clicks)
  } = $props();
  // first click arms (red, asks for confirmation), second click deletes
  let deleteArmed = $state(false);

  // current values from the store (the overview may pass a copy)
  const c = $derived(sidebar.customers.find((x) => x.id === customer.id) ?? customer);
  const wide = $derived(c.color_everywhere && !!c.color);
  const fav = $derived(isFavorite('customer', customer.id));
  let colorOpen = $state(false);
  const pickColor = (color: string) => colorView && setMyColor(colorView, c.id, color);

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
  <button type="button" class={itemCls} role="menuitem" onclick={run(() => toggleFavorite('customer', customer.id))}>
    <Icon name="star" size={15} class={fav ? 'fill-amber-400 text-amber-500' : 'text-zinc-400'} />{t(fav ? 'nav.removeFavorite' : 'nav.addFavorite')}
  </button>
  {#if canCreate}
    <button type="button" class={itemCls} role="menuitem" onclick={run(oncreate)}>
      <Icon name="plus" size={15} class="text-zinc-400" />{t('overview.newProject')}
    </button>
  {/if}
  <!-- personal card color of this view (everyone; the shared customer color wins) -->
  {#if colorView}
    {@const current = customerCardColor(c, colorView, myColors.all)}
    <button type="button" class={itemCls} role="menuitem" aria-expanded={colorOpen} onclick={() => (colorOpen = !colorOpen)}>
      <Icon name="palette" size={15} class="text-zinc-400" />{t('menu.cardColor')}
      <span class="ml-auto flex items-center gap-1.5">
        {#if current}<span class="size-3 rounded-full" style:background-color={cardHex(current)}></span>{/if}
        <Icon name="chevron" size={13} class="text-zinc-400 transition-transform {colorOpen ? 'rotate-90' : ''}" />
      </span>
    </button>
    {#if colorOpen}
      <div class="mx-1 mb-1.5 rounded-lg bg-zinc-50 p-1.5 dark:bg-zinc-800/50">
        {#if wide}
          <p class="px-1 pb-1 text-xs text-zinc-500">{t('menu.cardColorWide')}</p>
        {:else}
          <CardColorPicker label={t('menu.cardColor')} value={myColors.all[colorView][String(c.id)] ?? ''} onpick={pickColor} />
          <p class="px-1 pt-1 text-xs text-zinc-400">{t('menu.cardColorMine')}</p>
        {/if}
      </div>
    {/if}
  {/if}
  {#if onfilter}
    {@const filter = onfilter}
    <div class="my-1 border-t border-zinc-100 dark:border-zinc-800"></div>
    <button type="button" class={itemCls} role="menuitem" onclick={run(() => filter('only'))}>
      <Icon name="search" size={15} class="text-zinc-400" />{t('menu.onlyCustomer')}
    </button>
    <button type="button" class={itemCls} role="menuitem" onclick={run(() => filter('toggle'))}>
      <Icon name={hidden ? 'plus' : 'minus'} size={15} class="text-zinc-400" />{t(hidden ? 'menu.showCustomer' : 'menu.hideCustomer')}
    </button>
    <button type="button" class={itemCls} role="menuitem" onclick={run(() => filter('all'))}>
      <Icon name="users" size={15} class="text-zinc-400" />{t('menu.allCustomers')}
    </button>
  {/if}
  {#if ondelete}
    {@const del = ondelete}
    <div class="my-1 border-t border-zinc-100 dark:border-zinc-800"></div>
    <button
      type="button"
      role="menuitem"
      data-testid="customer-menu-delete"
      class="flex w-full items-center gap-2.5 rounded-md px-2 py-1.5 text-left {deleteArmed
        ? 'bg-red-600 font-medium text-white hover:bg-red-700'
        : 'text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-950/40'}"
      onclick={() => (deleteArmed ? run(del)() : (deleteArmed = true))}>
      <Icon name="trash" size={15} />{t(deleteArmed ? 'customer.deleteConfirm' : 'customer.delete')}
    </button>
  {/if}
</ContextMenu>
