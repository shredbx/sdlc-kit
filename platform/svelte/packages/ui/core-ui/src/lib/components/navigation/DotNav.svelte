<script lang="ts" module>
	export interface DotNavItem {
		id: string;
		label: string;
	}
</script>

<script lang="ts">
	import { tick } from 'svelte';
	import { page } from '$app/stores';

	let {
		items,
		offset = 0,
		rootMargin = '0px',
		threshold = [0, 0.25, 0.5, 0.75, 1]
	}: {
		items?: DotNavItem[];
		offset?: number;
		rootMargin?: string;
		threshold?: number | number[];
	} = $props();

	let discovered = $state<DotNavItem[]>([]);
	let activeId = $state<string | null>(null);
	let navEl: HTMLElement | undefined = $state();
	let expanded = $state(false);
	let collapseTimer: ReturnType<typeof setTimeout> | undefined;

	const resolved = $derived(items ?? discovered);

	function expand() {
		clearTimeout(collapseTimer);
		expanded = true;
	}

	function scheduleCollapse() {
		clearTimeout(collapseTimer);
		collapseTimer = setTimeout(() => { expanded = false; }, 300);
	}

	// Rewire on every route change — layout-mounted DotNav does not remount
	// when SvelteKit swaps +page.svelte, so the Sections under it are new DOM
	// nodes each time. Depend on $page.url.pathname so the effect tears down
	// the old observer and attaches a new one whenever the route changes.
	$effect(() => {
		// eslint-disable-next-line @typescript-eslint/no-unused-vars
		const _routeDep = $page.url.pathname;
		let observer: IntersectionObserver | null = null;
		let cancelled = false;

		(async () => {
			await tick();
			if (cancelled) return;
			const targets = collectTargets(items);
			if (items === undefined) {
				discovered = targets.map((el) => ({
					id: el.dataset.dotNavSection!,
					label: el.dataset.dotNavLabel ?? el.dataset.dotNavSection!
				}));
			}
			if (targets.length === 0) {
				activeId = null;
				return;
			}
			const ratios = new Map<string, number>();
			observer = new IntersectionObserver(
				(entries) => {
					for (const entry of entries) {
						const id = (entry.target as HTMLElement).dataset.dotNavSection;
						if (!id) continue;
						ratios.set(id, entry.isIntersecting ? entry.intersectionRatio : 0);
					}
					activeId = pickActive(ratios, targets);
				},
				{ rootMargin, threshold }
			);
			for (const el of targets) observer.observe(el);
			activeId = null;
		})();

		return () => {
			cancelled = true;
			observer?.disconnect();
		};
	});

	function collectTargets(explicit: DotNavItem[] | undefined): HTMLElement[] {
		if (explicit && explicit.length > 0) {
			return explicit
				.map((it) =>
					document.querySelector<HTMLElement>(`[data-dot-nav-section="${CSS.escape(it.id)}"]`)
				)
				.filter((el): el is HTMLElement => el !== null);
		}
		return Array.from(
			document.querySelectorAll<HTMLElement>('[data-dot-nav-section]')
		);
	}

	function pickActive(
		ratios: Map<string, number>,
		targets: HTMLElement[]
	): string | null {
		let bestId: string | null = null;
		let bestRatio = 0;
		for (const el of targets) {
			const id = el.dataset.dotNavSection;
			if (!id) continue;
			const r = ratios.get(id) ?? 0;
			if (r > bestRatio) {
				bestRatio = r;
				bestId = id;
			}
		}
		return bestId;
	}

	// No JS scroll — native <a href="#slug"> + scroll-margin-top on SectionShell handles it
</script>

{#if resolved.length > 1}
	<nav
		class="dot-nav"
		class:expanded
		aria-label="Page sections"
		bind:this={navEl}
		onmouseenter={expand}
		onmouseleave={scheduleCollapse}
	>
		<ul class="dot-nav-list">
			{#each resolved as item (item.id)}
				<li class="dot-nav-item">
					<a
						href={`#${item.id}`}
						class="dot-nav-link"
						class:active={activeId === item.id}
						aria-current={activeId === item.id ? 'true' : undefined}
						aria-label={item.label}
						>
						<span class="dot-nav-dot" aria-hidden="true"></span>
						<span class="dot-nav-label">{item.label}</span>
					</a>
				</li>
			{/each}
		</ul>
	</nav>
{/if}

<style>
	.dot-nav {
		position: fixed;
		top: 50%;
		right: 1.5rem;
		transform: translateY(-50%);
		z-index: 50;
		pointer-events: auto;
	}

	.dot-nav-list {
		list-style: none;
		margin: 0;
		padding: 12px 8px;
		display: flex;
		flex-direction: column;
		gap: 12px;
		align-items: center;
	}

	.dot-nav-item {
		margin: 0;
		padding: 0;
	}

	.dot-nav-link {
		display: block;
		position: relative;
		width: 10px;
		height: 10px;
		border-radius: 50%;
		background: color-mix(in srgb, var(--color-text, #fff) 15%, transparent);
		border: 1.5px solid color-mix(in srgb, var(--color-text, #fff) 25%, transparent);
		transition: transform 0.3s ease, background 0.3s ease, border-color 0.3s ease, box-shadow 0.3s ease;
		text-decoration: none;
	}

	/* Invisible hit area around each dot */
	.dot-nav-link::after {
		content: '';
		position: absolute;
		inset: -8px;
	}

	.dot-nav-dot {
		display: none;
	}

	/* Hovered dot */
	.dot-nav-link:hover {
		background: rgba(var(--color-accent-rgb, 233, 69, 96), 0.6);
		border-color: var(--color-accent, #e94560);
		transform: scale(1.8);
		box-shadow: 0 0 8px rgba(var(--color-accent-rgb, 233, 69, 96), 0.5);
	}

	/* Active dot — resting */
	.dot-nav-link.active {
		background: var(--color-accent, #e94560);
		border-color: var(--color-accent, #e94560);
		transform: scale(1.2);
	}

	/* Expanded — all dots awaken */
	.dot-nav.expanded .dot-nav-link {
		transform: scale(1.1);
	}
	.dot-nav.expanded .dot-nav-link.active {
		transform: scale(1.3);
	}
	.dot-nav.expanded .dot-nav-link:hover {
		transform: scale(1.8);
		background: rgba(var(--color-accent-rgb, 233, 69, 96), 0.6);
		border-color: var(--color-accent, #e94560);
		box-shadow: 0 0 8px rgba(var(--color-accent-rgb, 233, 69, 96), 0.5);
	}
	.dot-nav.expanded .dot-nav-link.active:hover {
		transform: scale(1.8);
	}

	.dot-nav-label {
		position: absolute;
		right: 20px;
		top: 50%;
		transform: translateY(-50%);
		font-size: 10px;
		color: var(--color-text, #fff);
		white-space: nowrap;
		opacity: 0;
		pointer-events: none;
		transition: opacity 0.2s ease, color 0.2s ease;
		font-family: var(--font-mono, ui-monospace, monospace);
		text-shadow: 0 0 8px rgba(var(--color-accent-rgb, 233, 69, 96), 0.3);
		background: rgba(0, 0, 0, 0.7);
		backdrop-filter: blur(8px);
		-webkit-backdrop-filter: blur(8px);
		padding: 2px 8px;
		border-radius: 4px;
	}

	/* Expanded — all labels visible + clickable */
	.dot-nav.expanded .dot-nav-label {
		opacity: 0.65;
		pointer-events: auto;
	}

	/* Hovered item label — full brightness */
	.dot-nav-link:hover .dot-nav-label {
		opacity: 1;
		color: var(--color-accent, #e94560);
		text-shadow: 0 0 12px rgba(var(--color-accent-rgb, 233, 69, 96), 0.6);
	}

	/* Keyboard focus */
	.dot-nav-link:focus-visible .dot-nav-label {
		opacity: 1;
		pointer-events: auto;
	}

	.dot-nav-link:focus-visible {
		outline: 2px solid var(--color-accent, #e94560);
		outline-offset: 4px;
	}

	/* Light mode — invert label pill for readable contrast */
	:global([data-theme='light']) .dot-nav-link {
		background: color-mix(in srgb, var(--color-text) 12%, transparent);
		border-color: color-mix(in srgb, var(--color-text) 20%, transparent);
	}

	:global([data-theme='light']) .dot-nav-link:hover {
		background: color-mix(in srgb, var(--color-accent) 15%, transparent);
		border-color: var(--color-accent);
		box-shadow: 0 0 0 2px color-mix(in srgb, var(--color-accent) 12%, transparent);
	}

	:global([data-theme='light']) .dot-nav-link.active {
		background: var(--color-accent);
		border-color: var(--color-accent);
	}

	:global([data-theme='light']) .dot-nav.expanded .dot-nav-link:hover {
		box-shadow: 0 0 0 2px color-mix(in srgb, var(--color-accent) 12%, transparent);
	}

	:global([data-theme='light']) .dot-nav-label {
		background: rgba(255, 255, 255, 0.92);
		backdrop-filter: blur(12px);
		-webkit-backdrop-filter: blur(12px);
		color: var(--color-text);
		text-shadow: none;
		border: 1px solid var(--color-border);
		box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
	}

	:global([data-theme='light']) .dot-nav-link:hover .dot-nav-label {
		color: var(--color-accent);
		text-shadow: none;
	}

	@media (max-width: 767px) {
		.dot-nav {
			display: none;
		}
	}

	@media (prefers-reduced-motion: reduce) {
		.dot-nav-link,
		.dot-nav-label {
			transition: none;
		}
		.dot-nav.expanded .dot-nav-link {
			transform: none;
		}
		.dot-nav.expanded .dot-nav-link:hover {
			transform: scale(1.3);
			box-shadow: none;
		}
		.dot-nav-link.active {
			transform: scale(1.2);
		}
	}
</style>
