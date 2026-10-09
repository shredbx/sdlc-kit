import { createDataRegistry } from '@sbx/app-svelte/server';
import { homePageModule } from './lib/server/data-modules';
import type { Handle } from '@sveltejs/kit';

export const handle: Handle = async ({ event, resolve }) => {
  event.locals.data = createDataRegistry({ homePage: homePageModule });
  return resolve(event);
};
