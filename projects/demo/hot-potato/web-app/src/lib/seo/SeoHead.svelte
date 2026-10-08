<script lang="ts">
  import { site } from '../../config/site';
  import { resolveSeo } from './resolve';
  import { serializeJsonLd } from './jsonLd';
  import type { PageSeo } from './types';

  let {
    seo,
    structuredData
  }: {
    seo: PageSeo;
    structuredData?: unknown;
  } = $props();

  const resolved = $derived(resolveSeo(seo, site));
</script>

<svelte:head>
  <title>{resolved.title}</title>
  <meta name="description" content={resolved.description} />
  <meta name="robots" content={resolved.robots} />
  <link rel="canonical" href={resolved.canonicalUrl} />
  <meta property="og:type" content="website" />
  <meta property="og:site_name" content={site.name} />
  <meta property="og:url" content={resolved.canonicalUrl} />
  <meta property="og:title" content={seo.openGraphTitle ?? resolved.title} />
  <meta property="og:description" content={seo.openGraphDescription ?? resolved.description} />
  <meta property="og:image" content={resolved.imageUrl} />
  <meta property="og:image:alt" content={resolved.imageAlt} />
  <meta property="og:image:width" content="1200" />
  <meta property="og:image:height" content="630" />
  <meta name="twitter:card" content="summary_large_image" />
  <meta name="twitter:title" content={resolved.title} />
  <meta name="twitter:description" content={resolved.description} />
  <meta name="twitter:image" content={resolved.imageUrl} />
  <meta name="twitter:image:alt" content={resolved.imageAlt} />
  {#if structuredData}
    {@html `<script type="application/ld+json">${serializeJsonLd(structuredData)}</script>`}
  {/if}
</svelte:head>
