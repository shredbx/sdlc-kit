<script lang="ts">
	// /mobile — proof first, then packages, then objection handling. Structurally identical
	// to /website: same components, same order, only the group filter and the copy keys
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

	const items = workIn('mobile');
	const packs = packagesIn('mobile');

	// Service + FAQPage, matching exactly what is visible. No price is emitted — free text
	// is not machine-readable and a fabricated number is worse than none (SC8c).
	const jsonLd = $derived([
		{
			'@context': 'https://schema.org',
			'@type': 'Service',
			name: t('mobile.hero.title'),
			description: t('mobile.meta.description'),
			serviceType: t('nav.mobile'),
			hasOfferCatalog: {
				'@type': 'OfferCatalog',
				name: t('mobile.packages.title'),
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
				name: t(`mobile.faq.${q}.q`),
				acceptedAnswer: { '@type': 'Answer', text: t(`mobile.faq.${q}.a`) }
			}))
		}
	]);
</script>

<Seo
	{locale}
	titleKey="mobile.meta.title"
	descriptionKey="mobile.meta.description"
	path="mobile"
	{jsonLd}
/>

<Hero {locale} titleKey="mobile.hero.title" leadKey="mobile.hero.lead" />

<section class="section-tight">
	<div class="shell">
		<h2 class="section-title">{t('mobile.work.title')}</h2>
		<p class="section-lead">{t('mobile.work.lead')}</p>
		<WorkList {items} {locale} framed />
		<p class="fine">{t('common.work.confidential')}</p>
	</div>
</section>

<StepLine {locale} />

<section class="section-tight" id="packages">
	<div class="shell">
		<h2 class="section-title">{t('mobile.packages.title')}</h2>
		<p class="section-lead">{t('mobile.packages.lead')}</p>
		<PackageCards items={packs} {locale} />
	</div>
</section>

<FaqList {locale} prefix="mobile" />

<CrossLink
	{locale}
	textKey="mobile.crosslink"
	ctaKey="mobile.crosslink.cta"
	href={localePath(locale, 'website')}
/>

<ContactBlock {locale} titleKey="home.contact.title" bodyKey="home.contact.body" />
