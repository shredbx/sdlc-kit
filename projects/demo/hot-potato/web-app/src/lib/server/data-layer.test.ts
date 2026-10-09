import { describe, expect, it } from 'vitest';
import { createDataRegistry, createReadOnlyRepository } from '@sbx/app-svelte/server';
import { homePageModule } from './data-modules';

describe('Hot Potato data layer', () => {
  it('registers the local home-page repository and returns schema-validated content', async () => {
    const registry = createDataRegistry({ homePage: homePageModule });
    const homePage = await registry.get('homePage').repository.findById('home');

    expect(homePage?.menuItems).toHaveLength(8);
    expect(homePage?.processSteps).toHaveLength(4);
    expect(homePage?.nav).toHaveLength(3);
  });

  it('returns null when a record does not exist', async () => {
    const registry = createDataRegistry({ homePage: homePageModule });

    await expect(registry.get('homePage').repository.findById('missing')).resolves.toBeNull();
  });

  it('rejects repository records that do not match the registered schema', async () => {
    const repository = createReadOnlyRepository({
      records: [{}],
      schema: homePageModule.schema,
      getId: () => 'invalid'
    });

    await expect(repository.findMany()).rejects.toThrow();
  });
});
