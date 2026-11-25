// The signed-in user (set by the layout from /auth/me, login and invite).
import type { User } from './types';

export const session = $state<{ user: User | null; ready: boolean; subtitle: string; version: string }>({ user: null, ready: false, subtitle: '', version: '' });

export function canWrite() {
  return session.user?.role === 'admin' || session.user?.role === 'member';
}

export function isAdmin() {
  return session.user?.role === 'admin';
}
