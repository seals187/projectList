import { redirect } from '@sveltejs/kit';

// /settings opens the appearance page
export const load = () => redirect(307, '/settings/appearance');
