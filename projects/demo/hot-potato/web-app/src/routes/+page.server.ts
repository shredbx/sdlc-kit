import { error } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals }) => {
  const homePage = await locals.data.get('homePage').repository.findById('home');
  if (!homePage) {
    error(404, 'Home page content was not found');
  }

  return { homePage };
};
