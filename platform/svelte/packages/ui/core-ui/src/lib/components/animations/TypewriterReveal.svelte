<script lang="ts">
	/**
	 * TypewriterReveal — typewriter reveal effect
	 *
	 * State machine: idle → animating → idle
	 *
	 * Triggers: mouseenter OR autoplay (fires once on mount).
	 * mouseleave does NOT cancel — animation always finishes.
	 * animation end: re-arms to idle, next mouseenter triggers again.
	 *
	 * During typing, each character appears in its final color —
	 * before/after in --color-text, target in --color-accent.
	 */

	interface Props {
		before?: string;
		target?: string;
		after?: string;
		text?: string;
		/** Milliseconds per character for typing, default 60 */
		speed?: number;
		/** Auto-trigger animation on mount instead of waiting for hover */
		autoplay?: boolean;
	}

	let {
		before = '',
		target = '',
		after = '',
		text = '',
		speed = 60,
		autoplay = false
	}: Props = $props();

	let phase = $state<'idle' | 'animating'>('idle');
	let visibleCount = $state(0);

	const animText = $derived((before + target + after) || text);
	const beforeLen = $derived(before.length);
	const targetLen = $derived(target.length);

	const visBefore = $derived(Math.min(visibleCount, beforeLen));
	const visTarget = $derived(Math.min(Math.max(visibleCount - beforeLen, 0), targetLen));
	const visAfter  = $derived(Math.max(visibleCount - beforeLen - targetLen, 0));

	import { onMount } from 'svelte';

	let timerId: ReturnType<typeof setTimeout> | undefined;

	function trigger() {
		if (phase !== 'idle') return;
		phase = 'animating';
		visibleCount = 0;

		if (typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
			visibleCount = animText.length;
			timerId = setTimeout(() => { phase = 'idle'; }, 600);
			return;
		}

		timerId = setTimeout(() => typeNext(0, animText.length), 150);
	}

	function typeNext(count: number, len: number) {
		if (count < len) {
			visibleCount = count + 1;
			timerId = setTimeout(() => typeNext(count + 1, len), speed);
		} else {
			phase = 'idle';
		}
	}

	onMount(() => {
		if (autoplay) trigger();
		return () => {
			if (timerId !== undefined) clearTimeout(timerId);
		};
	});
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<span class="tr-wrap" onmouseenter={trigger}>{#if phase === 'animating'}{#if before || target || after}{#if visBefore > 0}<span class="tr-before">{before.slice(0, visBefore)}</span>{/if}{#if visTarget > 0}<span class="tr-target">{target.slice(0, visTarget)}</span>{/if}{#if visAfter > 0}<span class="tr-after">{after.slice(0, visAfter)}</span>{/if}{:else}<span class="tr-anim">{text.slice(0, visibleCount)}</span>{/if}<span class="tr-cursor" aria-hidden="true">|</span>{:else}{#if before}<span class="tr-before">{before}</span>{/if}<span class="tr-target">{target || text}</span>{#if after}<span class="tr-after">{after}</span>{/if}{/if}</span>

<style>
	.tr-wrap {
		display: inline;
		color: var(--color-accent);
		cursor: default;
	}

	.tr-before,
	.tr-after {
		color: var(--color-text);
	}

	.tr-target {
		display: inline;
		color: inherit;
	}

	.tr-anim {
		display: inline;
		color: inherit;
	}

	.tr-cursor {
		display: inline;
		color: var(--color-accent);
		font-weight: 700;
		animation: tr-blink 0.6s step-end infinite;
	}

	@keyframes tr-blink {
		0%, 100% { opacity: 1; }
		50% { opacity: 0; }
	}

	@media (prefers-reduced-motion: reduce) {
		.tr-cursor {
			animation: none;
		}
	}
</style>
