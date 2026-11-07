// Texts of automation rules (templates page, "Automations"): trigger title,
// condition and schedule in the UI language.
import type { ComponentProps } from 'svelte';
import { t, type Key } from './i18n.svelte';
import { sidebar } from './state.svelte';
import { statusLabel } from './projectStatus';
import type { AutomationRule, AutomationSchedule, ProjectStatus } from './types';
import type Icon from './components/Icon.svelte';

type IconName = ComponentProps<typeof Icon>['name'];

export const APP_TRIGGERS = ['project_created', 'project_tag', 'project_status', 'schedule'] as const;
export const EXTERNAL_TRIGGERS = ['external_project', 'external_ticket'] as const;
export const INTEGRATIONS = ['easyredmine', 'otobo'] as const;

export const isExternal = (kind: string) => (EXTERNAL_TRIGGERS as readonly string[]).includes(kind);

export function triggerIcon(kind: string): IconName {
  return ({ project_created: 'folder', project_tag: 'tag', project_status: 'board', schedule: 'calendar' } as Record<string, IconName>)[kind] ?? 'zap';
}

export const sourceLabel = (s: string) => t(`var.source.${s}` as Key);

export function triggerTitle(r: Pick<AutomationRule, 'trigger_kind' | 'source'>): string {
  if (isExternal(r.trigger_kind)) return `${sourceLabel(r.source)}: ${t(`auto.trigger.${r.trigger_kind}` as Key)}`;
  return t(`auto.trigger.${r.trigger_kind}` as Key);
}

const p2 = (n: number | undefined) => String(n ?? 0).padStart(2, '0');

export function scheduleText(s: AutomationSchedule | null): string {
  if (!s) return '';
  if (s.freq === 'yearly') return t('auto.every.yearly', { date: `${p2(s.day)}.${p2(s.month)}.` });
  if (s.freq === 'monthly') return t('auto.every.monthly', { day: String(s.day ?? 1) });
  return t('auto.every.weekly', { day: t(`weekday.${s.weekday ?? 1}` as Key) });
}

/** Value of a condition for display (tag / customer / status names instead of IDs). */
function conditionValue(field: string, value: string): string {
  if (field === 'tag') return sidebar.tags.find((x) => String(x.id) === value)?.name ?? value;
  if (field === 'customer') return sidebar.customers.find((x) => String(x.id) === value)?.name ?? value;
  if (field === 'status') return statusLabel(value as ProjectStatus);
  return `„${value}“`;
}

/** Second line of a rule: condition or schedule. */
export function triggerDetail(r: Pick<AutomationRule, 'trigger_kind' | 'conditions' | 'schedule' | 'target_name'>): string {
  if (r.trigger_kind === 'schedule') return scheduleText(r.schedule);
  if (!r.conditions.length) return t('auto.always');
  return r.conditions
    .map((c) => t(c.op === 'contains' ? 'auto.when.contains' : 'auto.when.equals', { field: fieldLabel(c.field), value: conditionValue(c.field, c.value) }))
    .join(' · ');
}

export function fieldLabel(field: string): string {
  const known = ['name', 'customer', 'status', 'tag'];
  return known.includes(field) ? t(`auto.field.${field}` as Key) : field;
}
