<script lang="ts">
	// /website — proof first, then packages, then objection handling. Structurally identical
	// to /mobile: same components, same order, only the group filter and the copy keys
	// differ. That symmetry is the point of the shared content model.
	import { localePath, packageCopy, packagesIn, translator, workIn } from '$lib/content';
	import Hero from '$lib/sections/Hero.svelte';
	import WorkList from '$lib/sections/WorkList.svelte';
	import PackageCards from '$lib/sections/PackageCards.svelte';
	import StepLine from '$lib/sections/StepLine.svelte';
	import FaqList from '$lib/sections/FaqList.svelte';
	import CrossLink from '$lib/sections/CrossLink.svelte';
	import ContactBlock from '$lib/sections/ContactBlock.svelte';
	import Seo from '$lib/sections/Seo.svelte';

	let { data } = $props();
	const locale = $derived(data.locale);
	const t = $derived(translator(locale));

	const items = workIn('web');
	const packs = packagesIn('web');

	// Service + FAQPage, matching exactly what is visible. No price is emitted — free text
	// is not machine-readable and a fabricated number is worse than none (SC8c).
	const jsonLd = $derived([
		{
			'@context': 'https://schema.org',
			'@type': 'Service',
			name: t('website.hero.title'),
			description: t('website.meta.description'),
			serviceType: t('nav.website'),
			hasOfferCatalog: {
				'@type': 'OfferCatalog',
				name: t('website.packages.title'),
				itemListElement: packs.map((p) => {
					const c = packageCopy(p, locale);
					return {
						'@type': 'OfferCatalog',
						name: c.name,
						...(c.summary ? { description: c.summary } : {})
					};
				})
			}
		},
		{
			'@context': 'https://schema.org',
			'@type': 'FAQPage',
			mainEntity: ['q1', 'q2', 'q3', 'q4', 'q5'].map((q) => ({
				'@type': 'Question',
				name: t(`website.faq.${q}.q`),
				acceptedAnswer: { '@type': 'Answer', text: t(`website.faq.${q}.a`) }
			}))
		}
	]);
</script>

<Seo
	{locale}
	titleKey="website.meta.title"
	descriptionKey="website.meta.description"
	path="website"
	{jsonLd}
/>

<Hero {locale} titleKey="website.hero.title" leadKey="website.hero.lead" />

<section class="section-tight">
	<div class="shell">
		<h2 class="section-title">{t('website.work.title')}</h2>
		<p class="section-lead">{t('website.work.lead')}</p>
		<WorkList {items} {locale} />
		<p class="fine">{t('common.work.confidential')}</p>
	</div>
</section>

<StepLine {locale} />

<section class="section-tight" id="packages">
	<div class="shell">
		<h2 class="section-title">{t('website.packages.title')}</h2>
		<p class="section-lead">{t('website.packages.lead')}</p>
		<PackageCards items={packs} {locale} />
	</div>
</section>

<FaqList {locale} prefix="website" />

<CrossLink
	{locale}
	textKey="website.crosslink"
	ctaKey="website.crosslink.cta"
	href={localePath(locale, 'mobile')}
/>

<ContactBlock {locale} titleKey="home.contact.title" bodyKey="home.contact.body" />
