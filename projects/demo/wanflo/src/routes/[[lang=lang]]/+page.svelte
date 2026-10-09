<script lang="ts">
	// HOME — short hero, then the two doors carry the explanation. Target ~220-300 visible
	// words. It ranks for the brand term only; the service keywords belong to /website and
	// /mobile, so being short here is by design, not by accident.
	import {
		featuredWorkIn,
		localePath,
		packageCopy,
		packagesIn,
		translator,
		workCountLabel
	} from '$lib/content';
	import Hero from '$lib/sections/Hero.svelte';
	import WorkGrid from '$lib/sections/WorkGrid.svelte';
	import StepLine from '$lib/sections/StepLine.svelte';
	import TechRow from '$lib/sections/TechRow.svelte';
	import ContactBlock from '$lib/sections/ContactBlock.svelte';
	import Seo from '$lib/sections/Seo.svelte';

	let { data } = $props();
	const locale = $derived(data.locale);
	const t = $derived(translator(locale));

	const websiteHref = $derived(localePath(locale, 'website'));
	const mobileHref = $derived(localePath(locale, 'mobile'));

	const doors = $derived([
		{
			group: 'web' as const,
			titleKey: 'home.door.web.title',
			bodyKey: 'home.door.web.body',
			ctaKey: 'home.door.web.cta',
			href: websiteHref
		},
		{
			group: 'mobile' as const,
			titleKey: 'home.door.mobile.title',
			bodyKey: 'home.door.mobile.body',
			ctaKey: 'home.door.mobile.cta',
			href: mobileHref
		}
	]);
</script>

<Seo {locale} titleKey="home.meta.title" descriptionKey="home.meta.description" />

<Hero
	{locale}
	titleKey="home.hero.title"
	leadKey="home.hero.lead"
	secondaryKey="home.hero.secondary"
	secondaryHref="#work"
	portrait
/>

<!-- THE TWO DOORS — this is the explanation, not a teaser. ~45 words each. -->
<section class="section-tight">
	<div class="shell">
		<h2 class="sr-only">{t('home.doors.title')}</h2>
		<div class="doors">
			{#each doors as door (door.group)}
				<article class="door card">
					<h3>{t(door.titleKey)}</h3>
					<p class="door-body">{t(door.bodyKey)}</p>
					<ul class="door-packs" role="list">
						{#each packagesIn(door.group) as pkg (pkg.id)}
							<li>{packageCopy(pkg, locale).name}</li>
						{/each}
					</ul>
					<div class="door-foot">
						<!-- COUNTED from work.yml — never a hardcoded number that can drift -->
						<span class="chip">{workCountLabel(door.group, locale)}</span>
						<a class="link-arrow" href={door.href}>{t(door.ctaKey)}</a>
					</div>
				</article>
			{/each}
		</div>
	</div>
</section>

<!-- SELECTED WORK — a taste, with a way through to the full lists -->
<section class="section-tight" id="work">
	<div class="shell">
		<h2 class="section-title">{t('home.work.title')}</h2>
		<p class="section-lead">{t('home.work.lead')}</p>
		<div class="work-cols">
			<WorkGrid
				items={featuredWorkIn('web')}
				{locale}
				href={websiteHref}
				allKey="home.work.all.web"
				headingKey="home.door.web.title"
			/>
			<WorkGrid
				items={featuredWorkIn('mobile')}
				{locale}
				href={mobileHref}
				allKey="home.work.all.mobile"
				headingKey="home.door.mobile.title"
			/>
		</div>
	</div>
</section>

<StepLine {locale} />
<TechRow {locale} />

<ContactBlock {locale} titleKey="home.contact.title" bodyKey="home.contact.body" />

<style>
	.doors {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
		gap: 18px;
	}
	.door {
		display: flex;
		flex-direction: column;
		padding: clamp(22px, 3vw, 30px);
	}
	h3 {
		font-family: var(--font-display);
		font-weight: 800;
		font-size: 1.3rem;
		margin: 0 0 10px;
	}
	.door-body {
		color: var(--dim);
		margin: 0 0 18px;
		font-size: 0.98rem;
		line-height: 1.6;
	}
	.door-packs {
		flex: 1;
		display: flex;
		flex-direction: column;
		gap: 7px;
		margin: 0 0 20px;
		padding: 0;
		list-style: none;
	}
	.door-packs li {
		position: relative;
		padding-left: 18px;
		font-family: var(--font-display);
		font-weight: 700;
		font-size: 0.95rem;
	}
	.door-packs li::before {
		content: '▸';
		position: absolute;
		left: 0;
		color: var(--accent);
	}
	.door-foot {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 14px;
		flex-wrap: wrap;
		padding-top: 16px;
		border-top: 1px solid var(--line);
	}
	.work-cols {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
		gap: clamp(24px, 4vw, 48px);
		margin-top: 20px;
	}
	/* a rule between the two groups, so they read as two lists rather than one row */
	@media (min-width: 760px) {
		.work-cols > :global(* + *) {
			padding-left: clamp(24px, 4vw, 48px);
			border-left: 1px solid var(--line);
		}
	}
</style>
