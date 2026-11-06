<script lang="ts">
  // Settings → Users & permissions (admin): search, role chips, table
  // (user · role · customers · last active). A click on a row opens the
  // details on the right: name/e-mail, role, customer access, password reset
  // link (or resend the invitation) and remove. "Invite user" creates an
  // account and sends the invitation link.
  import { goto } from '$app/navigation';
  import { api, errorText } from '$lib/api';
  import { i18n, locale, t } from '$lib/i18n.svelte';
  import { isAdmin, session, sidebar, toast } from '$lib/state.svelte';
  import { relTime } from '$lib/relTime';
  import type { AdminUser, Role } from '$lib/types';
  import SettingsShell from '$lib/components/SettingsShell.svelte';
  import Editable from '$lib/components/Editable.svelte';
  import ConfirmButton from '$lib/components/ConfirmButton.svelte';
  import Avatar from '$lib/components/Avatar.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import EntityIcon from '$lib/components/EntityIcon.svelte';
  import InviteUserDialog from '$lib/components/InviteUserDialog.svelte';
  import LinkBox from '$lib/components/LinkBox.svelte';

  const ROLES: Role[] = ['admin', 'member', 'viewer'];
  const ROLE_PILL: Record<Role, string> = {
    admin: 'bg-accent-500/12 text-accent-700 dark:bg-accent-400/20 dark:text-accent-200',
    member: 'bg-sky-500/12 text-sky-700 dark:bg-sky-400/15 dark:text-sky-300',
    viewer: 'bg-zinc-100 text-zinc-600 dark:bg-zinc-800 dark:text-zinc-300'
  };

  let users = $state<AdminUser[] | null>(null);
  let selectedId = $state<number | null>(null);
  let query = $state('');
  let roleFilter = $state<Role | 'all'>('all');
  let inviting = $state(false);
  let linkResult = $state<{ userId: number; link: string; emailed: boolean } | null>(null);
  let busy = $state(false);
  let now = $state(new Date());

  $effect(() => {
    if (!isAdmin()) {
      goto('/settings');
      return;
    }
    api.get<AdminUser[]>('/users').then(
      (u) => (users = u),
      (e) => toast(errorText(e))
    );
    const id = setInterval(() => (now = new Date()), 60_000);
    return () => clearInterval(id);
  });

  const counts = $derived(Object.fromEntries(ROLES.map((r) => [r, users?.filter((u) => u.role === r).length ?? 0])) as Record<Role, number>);
  const shown = $derived.by(() => {
    const q = query.trim().toLocaleLowerCase();
    return (users ?? []).filter((u) => (roleFilter === 'all' || u.role === roleFilter) && (!q || u.name.toLocaleLowerCase().includes(q) || u.email.toLocaleLowerCase().includes(q)));
  });
  const selected = $derived(users?.find((u) => u.id === selectedId) ?? null);

  function lastActive(u: AdminUser): { text: string; cls: string } {
    if (u.invited) return { text: t('set.users.invited'), cls: 'text-orange-600 dark:text-orange-400' };
    if (!u.last_seen_at) return { text: t('set.users.never'), cls: 'text-zinc-400' };
    return { text: relTime(u.last_seen_at, now, locale(), 5) ?? t('set.users.activeNow'), cls: 'text-zinc-500' };
  }

  async function patch(u: AdminUser, body: Record<string, unknown>) {
    Object.assign(u, await api.patch<AdminUser>(`/users/${u.id}`, body));
  }
  async function patchSafe(u: AdminUser, body: Record<string, unknown>) {
    try {
      await patch(u, body);
      return true;
    } catch (e) {
      toast(errorText(e));
      return false;
    }
  }

  function toggleCustomer(u: AdminUser, cid: number) {
    patchSafe(u, { customer_ids: u.customer_ids.includes(cid) ? u.customer_ids.filter((x) => x !== cid) : [...u.customer_ids, cid] });
  }
  const allSelected = (u: AdminUser) => sidebar.customers.every((c) => u.customer_ids.includes(c.id));

  async function resetLink(u: AdminUser) {
    busy = true;
    try {
      const r = await api.post<{ link: string; emailed: boolean }>(`/users/${u.id}/reset-link`, { lang: i18n.lang });
      linkResult = { userId: u.id, link: r.link, emailed: r.emailed };
    } catch (e) {
      toast(errorText(e));
    } finally {
      busy = false;
    }
  }

  async function remove(u: AdminUser) {
    try {
      await api.del(`/users/${u.id}`);
      users = users?.filter((x) => x.id !== u.id) ?? null;
      selectedId = null;
    } catch (e) {
      toast(errorText(e));
    }
  }

  function select(id: number) {
    selectedId = selectedId === id ? null : id;
    linkResult = null;
  }
</script>

<SettingsShell title={t('set.nav.users')} subtitle={t('set.users.subtitle')} aside={selected ? details : undefined} asideClass="bg-white dark:bg-zinc-900">
  {#snippet actions()}
    <button type="button" class="ph-primary" onclick={() => (inviting = true)}><Icon name="mail" size={15} />{t('set.users.invite')}</button>
  {/snippet}

  <!-- search and role chips -->
  <div class="mb-4 flex flex-wrap items-center gap-2">
    <label class="flex h-[38px] w-full max-w-xs items-center gap-2 rounded-[10px] border border-zinc-200 bg-white px-3 text-zinc-400 focus-within:border-accent-400 dark:border-zinc-700 dark:bg-zinc-900">
      <Icon name="search" size={15} />
      <input class="min-w-0 flex-1 bg-transparent text-[14px] text-zinc-900 outline-none placeholder:text-zinc-400 dark:text-zinc-100" placeholder={t('set.users.search')} bind:value={query} aria-label={t('set.users.search')} />
    </label>
    <div class="flex flex-wrap gap-1.5" role="group" aria-label={t('common.role')}>
      {#each ['all', ...ROLES] as const as r (r)}
        <button
          type="button"
          class="ph-chip h-[34px]! px-3.5! text-[13.5px]! {roleFilter === r ? 'ph-chip-on' : 'ph-chip-off'}"
          aria-pressed={roleFilter === r}
          onclick={() => (roleFilter = r)}>
          {r === 'all' ? t('filter.all') : t(`role.${r}`)}
          <span class="tabular-nums">{r === 'all' ? (users?.length ?? 0) : counts[r]}</span>
        </button>
      {/each}
    </div>
  </div>

  <!-- table -->
  <div class="@container overflow-x-auto rounded-[14px] border border-zinc-200/90 bg-white dark:border-zinc-800 dark:bg-zinc-900">
    <table class="w-full min-w-[440px] text-left">
      <thead>
        <tr class="border-b border-zinc-100 text-[11.5px] font-semibold tracking-wider text-zinc-400 uppercase dark:border-zinc-800">
          <th class="py-3 pr-3 pl-5 font-semibold">{t('set.users.col.user')}</th>
          <th class="px-3 py-3 font-semibold">{t('common.role')}</th>
          <th class="px-3 py-3 font-semibold @max-[620px]:hidden">{t('nav.customers')}</th>
          <th class="py-3 pr-5 pl-3 font-semibold whitespace-nowrap">{t('set.users.col.lastActive')}</th>
        </tr>
      </thead>
      <tbody class="divide-y divide-zinc-100 dark:divide-zinc-800">
        {#each shown as u (u.id)}
          {@const la = lastActive(u)}
          {@const on = selectedId === u.id}
          <tr
            class="cursor-pointer transition-colors {on ? 'bg-accent-500/[0.05] shadow-[inset_3px_0_0_var(--accent-primary)] dark:bg-accent-400/10' : 'hover:bg-zinc-50 dark:hover:bg-zinc-800/40'}"
            onclick={() => select(u.id)}
            data-testid="user-row">
            <td class="py-3 pr-3 pl-5">
              <button type="button" class="flex items-center gap-3 text-left" onclick={(e) => (e.stopPropagation(), select(u.id))} aria-expanded={on}>
                <Avatar name={u.name} id={u.id} size={38} />
                <span class="min-w-0">
                  <span class="block truncate text-[15px] font-semibold">{u.name} {#if u.id === session.user?.id}<span class="font-normal text-zinc-400">{t('users.you')}</span>{/if}</span>
                  <span class="block truncate text-[13.5px] text-zinc-500">{u.email}</span>
                </span>
              </button>
            </td>
            <td class="px-3 py-3">
              <span class="relative inline-flex items-center">
                <select
                  class="h-[30px] cursor-pointer appearance-none rounded-lg py-0 pr-7 pl-3 text-[13.5px] font-semibold outline-none {ROLE_PILL[u.role]}"
                  value={u.role}
                  aria-label="{t('common.role')}: {u.name}"
                  onclick={(e) => e.stopPropagation()}
                  onchange={async (e) => {
                    const el = e.currentTarget;
                    if (!(await patchSafe(u, { role: el.value }))) el.value = u.role;
                  }}>
                  {#each ROLES as r (r)}<option value={r}>{t(`role.${r}`)}</option>{/each}
                </select>
                <Icon name="chevronDown" size={13} class="pointer-events-none absolute right-2 opacity-70" />
              </span>
            </td>
            <td class="px-3 py-3 text-[14.5px] whitespace-nowrap @max-[620px]:hidden">{u.role === 'admin' ? t('users.allCustomers') : t('users.customersCount', { n: u.customer_ids.length })}</td>
            <td class="py-3 pr-5 pl-3 text-[14px] whitespace-nowrap {la.cls}" data-testid="last-active">{la.text}</td>
          </tr>
        {:else}
          <tr><td colspan="4" class="px-5 py-5 text-sm text-zinc-400">{users ? t('set.noMatch') : '…'}</td></tr>
        {/each}
      </tbody>
    </table>
  </div>
  <p class="mt-3 text-[13px] text-zinc-500">{t('users.hint')}</p>

</SettingsShell>

  {#snippet details()}
    {#if selected}
      {@const u = selected}
      <div class="flex h-full flex-col p-6" aria-label={u.name}>
        <div class="flex items-start gap-3">
          <Avatar name={u.name} id={u.id} size={52} />
          <div class="min-w-0 flex-1 pt-0.5">
            <div class="text-[18px] font-semibold"><Editable value={u.name} required onsave={(name) => patch(u, { name })} /></div>
            <div class="text-[14px] text-zinc-500"><Editable value={u.email} required onsave={(email) => patch(u, { email })} /></div>
          </div>
          <button type="button" class="rounded-md p-1 text-zinc-400 hover:bg-zinc-100 hover:text-zinc-700 dark:hover:bg-zinc-800 dark:hover:text-zinc-200" onclick={() => (selectedId = null)} aria-label={t('common.close')}>
            <Icon name="x" size={18} />
          </button>
        </div>

        <p class="mt-6 mb-2 text-[14.5px] font-semibold">{t('common.role')}</p>
        <div class="space-y-2" role="radiogroup" aria-label={t('common.role')}>
          {#each ROLES as r (r)}
            {@const on = u.role === r}
            <button
              type="button"
              role="radio"
              aria-checked={on}
              class="flex w-full items-start gap-3 rounded-xl border px-4 py-3 text-left transition-colors {on
                ? 'border-accent-300 bg-accent-50 dark:border-accent-500/50 dark:bg-accent-400/10'
                : 'border-zinc-200 hover:border-zinc-300 dark:border-zinc-700 dark:hover:border-zinc-600'}"
              onclick={() => !on && patchSafe(u, { role: r })}>
              <span class="mt-0.5 flex size-[18px] shrink-0 items-center justify-center rounded-full border-2 {on ? 'border-[var(--accent-primary)]' : 'border-zinc-300 dark:border-zinc-600'}">
                {#if on}<span class="size-2 rounded-full bg-[var(--accent-primary)]"></span>{/if}
              </span>
              <span>
                <span class="block text-[15px] font-semibold">{t(`role.${r}`)}</span>
                <span class="block text-[13.5px] text-zinc-500">{t(`set.users.roleDesc.${r}`)}</span>
              </span>
            </button>
          {/each}
        </div>

        <div class="mt-6 mb-1 flex items-center justify-between">
          <p class="text-[14.5px] font-semibold">{t('set.users.access')}</p>
          {#if u.role !== 'admin' && sidebar.customers.length}
            <button
              type="button"
              class="text-[13.5px] font-medium text-accent-600 hover:underline dark:text-accent-400"
              onclick={() => patchSafe(u, { customer_ids: allSelected(u) ? [] : sidebar.customers.map((c) => c.id) })}>
              {allSelected(u) ? t('set.users.selectNone') : t('set.users.selectAll')}
            </button>
          {/if}
        </div>
        {#if u.role === 'admin'}
          <p class="text-[13.5px] text-zinc-500">{t('set.users.adminAll')}</p>
        {:else}
          <ul class="divide-y divide-zinc-100 border-t border-zinc-100 dark:divide-zinc-800 dark:border-zinc-800">
            {#each sidebar.customers as c (c.id)}
              {@const has = u.customer_ids.includes(c.id)}
              <li>
                <label class="flex cursor-pointer items-center gap-3 py-2.5">
                  <input type="checkbox" class="size-[18px] accent-[var(--accent-primary)]" checked={has} onchange={() => toggleCustomer(u, c.id)} />
                  <EntityIcon type="customer" size="sm" />
                  <span class="truncate text-[14.5px]">{c.name.replace(/^\s*\([^)]{1,4}\)\s*/, '')}</span>
                </label>
              </li>
            {:else}
              <li class="py-2 text-sm text-zinc-400">{t('users.noCustomers')}</li>
            {/each}
          </ul>
        {/if}

        <div class="mt-auto space-y-3 border-t border-zinc-100 pt-5 dark:border-zinc-800">
          {#if linkResult?.userId === u.id}
            <LinkBox link={linkResult.link} emailed={linkResult.emailed} email={u.email} />
          {/if}
          {#if u.id !== session.user?.id}
            <button type="button" class="btn-outline h-10 w-full justify-center" onclick={() => resetLink(u)} disabled={busy}>
              <Icon name={u.invited ? 'mail' : 'lock'} size={15} />{u.invited ? t('set.users.resendInvite') : t('set.users.resetPassword')}
            </button>
            <ConfirmButton class="flex items-center gap-2 rounded-md px-1 py-1 text-[14.5px] text-red-600 hover:underline dark:text-red-400" onconfirm={() => remove(u)} confirmLabel={t('users.deleteConfirm')}>
              <Icon name="trash" size={15} />{t('set.users.remove')}
            </ConfirmButton>
          {:else}
            <a href="/settings/profile" class="btn-outline h-10 w-full justify-center"><Icon name="lock" size={15} />{t('settings.password')}</a>
          {/if}
        </div>
      </div>
    {/if}
  {/snippet}

{#if inviting}
  <InviteUserDialog
    onclose={() => (inviting = false)}
    oncreated={(u) => {
      users = [...(users ?? []), u].sort((a, b) => a.name.localeCompare(b.name));
      selectedId = u.id;
      linkResult = null;
    }} />
{/if}
