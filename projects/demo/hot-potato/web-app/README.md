# Web app routes, content, and SEO

## Where to edit

- `src/config/site.ts` is the only source for the public site origin, site name,
  and default social-sharing image. Canonicals, schema URLs, `robots.txt`, and
  `sitemap.xml` are generated from this config; do not duplicate the origin in
  an environment file or another config.
- `src/config/restaurant.ts` holds verified business facts shared by the page
  and its Restaurant structured data.
- `src/pages/home/content.ts` holds homepage copy, navigation, and menu data.
- `src/pages/home/seo.ts` holds the homepage title, description, and canonical
  path. Its share image uses the site default unless the page overrides it.
- `src/pages/home/structuredData.ts` builds the homepage's structured data from
  the shared site and restaurant config.
- `src/pages/seoPages.ts` registers canonical page paths for the sitemap.
- `src/lib/seo/SeoHead.svelte` renders page metadata into the HTML head during
  server rendering and prerendering.

The app uses the Node adapter. At request time, the root route loads checked-in
home-page content through the app framework's read-only repository; the content
is validated against the site's runtime schema and passed to the page component.
Search engines and social crawlers receive rendered HTML and metadata in the
initial response; rendering does not depend on client-side JavaScript. Facebook
app metadata is intentionally not configured.

## Adding pages

For a new URL such as `/menu/`, add `src/routes/menu/+page.server.ts` to load
validated page content through the registered read-only repository, plus
`src/routes/menu/+page.svelte` to render its page component. Place page
components, content, SEO metadata, and structured data in `src/pages/menu/`.
Render metadata with `SeoHead`, then add its canonical path to
`src/pages/seoPages.ts` for the sitemap. Keep route-specific content in the
page folder and business-wide facts in `src/config/restaurant.ts`.
