import { z } from 'zod';
import { links, menuItems, nav, processSteps } from '../../pages/home/content';
import { createReadOnlyRepository } from '@sbx/app-svelte/server';

const linkSchema = z.object({
  label: z.string(),
  href: z.string(),
  external: z.boolean().optional()
});

export const homePageSchema = z.object({
  id: z.string(),
  links: z.object({
    grab: linkSchema,
    maps: linkSchema,
    phone: linkSchema
  }),
  nav: z.array(linkSchema),
  menuItems: z.array(
    z.object({
      name: z.string(),
      image: z.object({ src: z.string(), alt: z.string() }),
      priceLabel: z.string()
    })
  ),
  processSteps: z.array(
    z.object({
      number: z.string(),
      title: z.string(),
      body: z.string()
    })
  )
});

export const homePageModule = {
  schema: homePageSchema,
  repository: createReadOnlyRepository({
    records: [{ id: 'home', links, nav, menuItems, processSteps }],
    schema: homePageSchema,
    getId: (record) => record.id
  })
};
