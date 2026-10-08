import type { createDataRegistry } from '@sbx/app-svelte/server';
import type { homePageModule } from './lib/server/data-modules';

declare global {
  namespace App {
    interface Locals {
      data: ReturnType<typeof createDataRegistry<{ homePage: typeof homePageModule }>>;
    }
  }
}

export {};
