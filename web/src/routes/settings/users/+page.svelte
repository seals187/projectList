<script lang="ts">
  import { goto } from '$app/navigation';
  import { api, errorText } from '$lib/api';
  import { isAdmin, session, sidebar, toast } from '$lib/state.svelte';
  import { t } from '$lib/i18n.svelte';
  import type { AdminUser, Role } from '$lib/types';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import SettingsTabs from '$lib/components/SettingsTabs.svelte';
  import Editable from '$lib/components/Editable.svelte';
  import ConfirmButton from '$lib/components/ConfirmButton.svelte';
  import Avatar from '$lib/components/Avatar.svelte';
  import Icon from '$lib/components/Icon.svelte';

  let users = $state<AdminUser[] | null>(null);
  let open = $state<number | null>(null);
  let creating = $state(false);
  let form = $state({ name: '', email: '', password: '', role: 'member' as Role });
  let resetPw = $state('');

  $effect(() => {
    if (!isAdmin()) {
      goto('/settings');
      return;
    }
    api.get<AdminUser[]>('/users').then(
      (u) => (users = u),
      (e) => toast(errorText(e))
    );
  });

  async function create(e: SubmitEvent) {
    e.preventDefault();
    try {
      const u = await api.post<AdminUser>('/users', { ...form, customer_ids: [] });
      users?.push(u);
      form = { name: '', email: '', password: '', role: 'member' };
      creating = false;
      open = u.id;
    } catch (err) {
      toast(errorText(err));
    }
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

  async function toggleCustomer(u: AdminUser, cid: number) {
    const ids = u.customer_ids.includes(cid) ? u.customer_ids.filter((x) => x !== cid) : [...u.customer_ids, cid];
    await patchSafe(u, { customer_ids: ids });
  }

  async function setPassword(u: AdminUser) {
    if (await patchSafe(u, { password: resetPw })) {
      resetPw = '';
      toast(t('users.passwordSet'), 'info');
    }
  }

  async function remove(u: AdminUser) {
    try {
      await api.del(`/users/${u.id}`);
      users = users?.filter((x) => x.id !== u.id) ?? null;
    } catch (e) {
      toast(errorText(e));
    }
  }
</script>

<svelte:head><title>{t('settings.users')} · ProjectList</title></svelte:head>

<PageHeader>
  {#snippet actions()}
    {#if !creating}<button type="button" class="btn-primary" onclick={() => (creating = true)}><Icon name="plus" size={14} /> {t('users.add')}</button>{/if}
  {/snippet}
  {#snippet title()}{t('nav.settings')}{/snippet}
</PageHeader>
<SettingsTabs />

<p class="mb-6 max-w-2xl text-sm text-zinc-500">{t('users.hint')}</p>

{#if creating}
  <form onsubmit={create} class="mb-8 grid max-w-2xl gap-3 rounded-lg border border-zinc-200 p-4 sm:grid-cols-2 dark:border-zinc-800">
    <div><label class="label" for="n">{t('common.name')}</label><input id="n" class="input" required bind:value={form.name} /></div>
    <div><label class="label" for="e">{t('common.email')}</label><input id="e" type="email" class="input" required bind:value={form.email} /></div>
    <div>
      <label class="label" for="p">{t('users.passwordMin')}</label>
      <input id="p" type="password" autocomplete="new-password" minlength="8" class="input" required bind:value={form.password} />
    </div>
    <div>
      <label class="label" for="r">{t('common.role')}</label>
      <select id="r" class="input" bind:value={form.role}>
        <option value="member">{t('role.member')}</option><option value="viewer">{t('role.viewer')}</option><option value="admin">{t('role.admin')}</option>
      </select>
    </div>
    <div class="flex gap-2 sm:col-span-2">
      <button class="btn-primary" type="submit">{t('common.create')}</button>
      <button class="btn-ghost" type="button" onclick={() => (creating = false)}>{t('common.cancel')}</button>
    </div>
  </form>
{/if}

<ul class="max-w-3xl">
  {#each users ?? [] as u (u.id)}
    <li class="border-b border-zinc-100 last:border-0 dark:border-zinc-800/70">
      <button
        type="button"
        class="-mx-2 flex w-[calc(100%+1rem)] items-center gap-3 rounded-md px-2 py-2.5 text-left hover:bg-zinc-50 dark:hover:bg-zinc-900/70"
        onclick={() => ((open = open === u.id ? null : u.id), (resetPw = ''))}
        aria-expanded={open === u.id}>
        <Avatar name={u.name} id={u.id} size={28} />
        <div class="min-w-0 flex-1">
          <p class="truncate font-medium">{u.name} {#if u.id === session.user?.id}<span class="text-xs font-normal text-zinc-400">{t('users.you')}</span>{/if}</p>
          <p class="truncate text-xs text-zinc-400">{u.email}</p>
        </div>
        <span class="badge {u.role === 'admin' ? 'bg-accent-50 text-accent-700 dark:bg-accent-950/60 dark:text-accent-400' : 'bg-zinc-100 text-zinc-600 dark:bg-zinc-800 dark:text-zinc-300'}">{t(`role.${u.role}`)}</span>
        <span class="hidden w-24 text-right text-xs text-zinc-400 sm:inline">{u.role === 'admin' ? t('users.allCustomers') : t('users.customersCount', { n: u.customer_ids.length })}</span>
        <Icon name="chevron" size={14} class="text-zinc-400 transition-transform {open === u.id ? 'rotate-90' : ''}" />
      </button>

      {#if open === u.id}
        <div class="space-y-5 pt-2 pb-6 pl-10">
          <div class="grid gap-4 sm:grid-cols-3">
            <div class="text-sm"><span class="label">{t('common.name')}</span><Editable value={u.name} required onsave={(name) => patch(u, { name })} /></div>
            <div class="text-sm"><span class="label">{t('common.email')}</span><Editable value={u.email} required onsave={(email) => patch(u, { email })} /></div>
            <div>
              <label class="label" for="role-{u.id}">{t('common.role')}</label>
              <select
                id="role-{u.id}"
                class="input"
                value={u.role}
                onchange={async (e) => {
                  const el = e.currentTarget;
                  if (!(await patchSafe(u, { role: el.value }))) el.value = u.role;
                }}>
                <option value="admin">{t('role.admin')}</option><option value="member">{t('role.member')}</option><option value="viewer">{t('role.viewer')}</option>
              </select>
            </div>
          </div>

          {#if u.role !== 'admin'}
            <div>
              <span class="label">{t('users.assigned')}</span>
              <div class="flex flex-wrap gap-1.5">
                {#each sidebar.customers as c (c.id)}
                  {@const on = u.customer_ids.includes(c.id)}
                  <button
                    type="button"
                    onclick={() => toggleCustomer(u, c.id)}
                    class="inline-flex items-center gap-1 rounded-full border px-2.5 py-1 text-xs {on
                      ? 'border-accent-500 bg-accent-50 text-accent-700 dark:bg-accent-950/60 dark:text-accent-400'
                      : 'border-zinc-300 text-zinc-600 hover:border-zinc-400 dark:border-zinc-700 dark:text-zinc-300'}"
                    aria-pressed={on}>
                    {#if on}<Icon name="check" size={12} />{/if}
                    {c.name}
                  </button>
                {:else}
                  <span class="text-sm text-zinc-400">{t('users.noCustomers')}</span>
                {/each}
              </div>
            </div>
          {/if}

          <div class="flex flex-wrap items-end gap-2">
            <div>
              <label class="label" for="pw-{u.id}">{t('users.newPassword')}</label>
              <input id="pw-{u.id}" type="password" autocomplete="new-password" class="input w-56!" bind:value={resetPw} />
            </div>
            <button type="button" class="btn-outline" disabled={resetPw.length < 8} onclick={() => setPassword(u)}>{t('users.setPassword')}</button>
            {#if u.id !== session.user?.id}
              <div class="ml-auto"><ConfirmButton onconfirm={() => remove(u)} label={t('users.delete')} confirmLabel={t('users.deleteConfirm')} /></div>
            {/if}
          </div>
        </div>
      {/if}
    </li>
  {/each}
</ul>
