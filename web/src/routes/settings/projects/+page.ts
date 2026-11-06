import { redirect } from '@sveltejs/kit';

// old address of the tag settings
export const load = () => redirect(307, '/settings/tags');
