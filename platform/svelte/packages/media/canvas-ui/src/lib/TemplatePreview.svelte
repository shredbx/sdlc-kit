<script lang="ts">
	// Live template thumbnail (#4) — lazy-fetches a template's kit Document and renders
	// its first page into a small <canvas> via the SHARED export renderer (renderDocPreview
	// → resolveFrame + renderPage), so a card is byte-identical to the editor/export, just
	// scaled. No baked snapshot file: cheap-enough for the ~25-40 templates a gallery holds.
	// Renders only once the card is on-screen (IntersectionObserver) so a long, scrolled
	// gallery never rasterizes off-screen cards.
	import type { Document, FormatterRegistry } from '@sbx/canvas-kit';
	import { renderDocPreview } from './rasterize.js';

	interface Props {
		/** Template id — fetched lazily via {@link load}. */
		id: string;
		/** Lazy doc fetch (consumer-owned); null on failure → the card shows its fallback. */
		load: (id: string) => Promise<Document | null>;
		/** Same image proxy the stage/export use, so a bound cover loads CORS-clean. */
		resolveImageSrc?: (src: string) => string;
		/** Consumer formatters (currency/area/…) so bound-field fallbacks read correctly. */
		formatters?: FormatterRegistry;
	}

	let { id, load, resolveImageSrc = (s) => s, formatters = {} }: Props = $props();

	let canvas = $state<HTMLCanvasElement | null>(null);
	let host = $state<HTMLDivElement | null>(null);
	let phase = $state<'idle' | 'loading' | 'ready' | 'empty'>('idle');
	let visible = $state(false);

	// Reveal-on-visible: only rasterize a card the user can actually see.
	$effect(() => {
		if (!host || visible) return;
		const io = new IntersectionObserver(
			(entries) => {
				if (entries.some((e) => e.isIntersecting)) {
					visible = true;
					io.disconnect();
				}
			},
			{ rootMargin: '120px' }
		);
		io.observe(host);
		return () => io.disconnect();
	});

	// Fetch + render once visible. id is the trigger — a re-keyed card (different template)
	// re-runs. Guards against a torn-down canvas between the await and the draw.
	$effect(() => {
		if (!visible || !canvas) return;
		const el = canvas;
		const templateId = id;
		phase = 'loading';
		let cancelled = false;
		void (async () => {
			const doc = await load(templateId);
			if (cancelled || !doc) {
				if (!cancelled) phase = 'empty';
				return;
			}
			// Size the backing store to the displayed box × DPR for a crisp thumbnail.
			const dpr = typeof window !== 'undefined' ? Math.min(window.devicePixelRatio || 1, 2) : 1;
			const rect = el.getBoundingClientRect();
			el.width = Math.max(1, Math.round(rect.width * dpr));
			el.height = Math.max(1, Math.round(rect.height * dpr));
			const ok = await renderDocPreview(el, doc, resolveImageSrc, formatters);
			if (!cancelled) phase = ok ? 'ready' : 'empty';
		})();
		return () => {
			cancelled = true;
		};
	});
</script>

<div class="tpv" bind:this={host} data-state={phase}>
	<canvas bind:this={canvas} class="tpv__canvas" class:tpv__canvas--ready={phase === 'ready'}></canvas>
	{#if phase !== 'ready'}
		<span class="tpv__fallback" aria-hidden="true"></span>
	{/if}
</div>

<style>
	.tpv {
		position: relative;
		width: 100%;
		height: 100%;
		display: block;
		background: var(--cv-color-neutral-100, #f5f5f5);
		overflow: hidden;
	}

	.tpv__canvas {
		display: block;
		width: 100%;
		height: 100%;
		object-fit: contain;
		opacity: 0;
		transition: opacity 0.18s ease;
	}

	.tpv__canvas--ready {
		opacity: 1;
	}

	/* Quiet shimmer placeholder until the live render lands (or on failure). */
	.tpv__fallback {
		position: absolute;
		inset: 0;
		background: linear-gradient(
			110deg,
			var(--cv-color-neutral-100, #f0f0f0) 30%,
			var(--cv-color-neutral-200, #e4e4e4) 50%,
			var(--cv-color-neutral-100, #f0f0f0) 70%
		);
		background-size: 200% 100%;
		animation: tpv-shimmer 1.3s ease-in-out infinite;
	}

	@keyframes tpv-shimmer {
		0% {
			background-position: 150% 0;
		}
		100% {
			background-position: -50% 0;
		}
	}

	@media (prefers-reduced-motion: reduce) {
		.tpv__canvas {
			transition: none;
		}
		.tpv__fallback {
			animation: none;
		}
	}
</style>
