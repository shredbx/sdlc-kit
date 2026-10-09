import {
	packageCopy,
	packages,
	packagesIn,
	priceLabel,
	site,
	techStack,
	work,
	workIn,
	workNote,
	workType
} from '$lib/content';

export const prerender = true;

// llms.txt — the plain-text answer for AI answer engines. Generated from content/*.yml, so
// it can never drift from what the pages say: edit the YAML and this regenerates.
// No price is stated unless the owner actually wrote one (free text, verbatim) — a
// fabricated figure would be worse here than anywhere, because it gets quoted.

const listPackages = (group: 'web' | 'mobile') =>
	packagesIn(group)
		.map((p) => {
			const c = packageCopy(p, 'en');
			const price = p.price ? ` — ${p.price}` : '';
			const summary = c.summary ? `\n  ${c.summary}` : '';
			return `- ${c.name}${price}${summary}\n  ${c.features.join(' · ')}`;
		})
		.join('\n');

const listWork = (group: 'web' | 'mobile') =>
	workIn(group)
		.map((w) => {
			// "no url" is NOT "archived" — most apps here simply have no public store link.
			const link = w.url ? ` — ${w.url}` : w.archived ? ' — archived' : '';
			const desc = workNote(w, 'en');
			const note = desc ? `\n  ${desc}` : '';
			const credit = w.credit ? ` [${w.credit.role}: ${w.credit.label}]` : '';
			return `- ${w.name}: ${workType(w, 'en')} (${w.stack.join(' · ')})${link}${credit}${note}`;
		})
		.join('\n');

export function GET() {
	const body = `# ${site.name}

> ${site.name} is a ${site.base}-based web and mobile development company. We design, build
> and launch websites and mobile apps, and keep them running afterwards. We work with
> clients anywhere.

## What we do

Websites — landing pages, catalogues, booking sites and real-estate platforms, built on the
client's own domain and data, multilingual, fast.

Mobile apps — iOS, Android and cross-platform. Built, taken through store review into the
App Store and Google Play, and maintained after launch.

## Website packages

${listPackages('web')}

## Mobile packages

${listPackages('mobile')}

Prices are quoted in ${site.currency}. Where no price is listed above, it is scoped per
request rather than fixed.

## Shipped work — websites (${workIn('web').length})

${listWork('web')}

## Shipped work — mobile apps (${workIn('mobile').length})

${listWork('mobile')}

## Technology

${techStack().join(' · ')}

## How we work

1. Talk — you describe what you need, we come back with scope and a price.
2. Build — you see it working as it goes, not only at the end.
3. Launch and keep running — we ship it and maintain it afterwards.

## Contact

WhatsApp: ${site.contact.whatsapp.display}${site.contact.whatsapp.handle ? ` (${site.contact.whatsapp.handle})` : ''}
Email: ${site.contact.email}
Based in: ${site.base}
Web: ${site.url}

There is no contact form — visitors reach a person directly by WhatsApp or email.

## Pages

- ${site.url}/ — overview
- ${site.url}/website — website development, packages, shipped sites
- ${site.url}/mobile — mobile app development, packages, shipped apps
- ${site.url}/contacts — direct contact channels

Available in English (/), Thai (/th) and Russian (/ru).
`;
	return new Response(body, { headers: { 'Content-Type': 'text/plain; charset=utf-8' } });
}
