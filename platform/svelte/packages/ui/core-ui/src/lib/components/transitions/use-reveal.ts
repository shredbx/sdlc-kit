/**
 * use:reveal — Svelte action for mount-time scroll-triggered reveal animations.
 *
 * Adapted from the IntersectionObserver pattern used in EditorialTrio.svelte
 * and EditorialSplitCounter.svelte. Centralises the pattern so every Reveal*
 * component wrapper shares one implementation and one prefers-reduced-motion
 * short-circuit.
 *
 * Behaviour:
 *   1. On mount, observe the element with IntersectionObserver.
 *   2. When intersection changes, call params.onChange(isIntersecting).
 *   3. If `once` (default true), disconnect after the first intersect.
 *   4. If `prefers-reduced-motion: reduce`, skip observation entirely and
 *      call onChange(true) immediately.
 *
 * Decision #0212: Reveal family uses per-component mount-time IO; Hideout
 * family uses the same action with once=false so the callback fires both
 * directions; Page family is route-lifecycle, not this action.
 *
 * State ownership: the component owns the revealed state via $state and
 * passes onChange to the action. This keeps Svelte's CSS scoper aware of
 * the state, avoiding "unused selector" warnings.
 *
 * @example
 * ```svelte
 * <script>
 *   let revealed = $state(false);
 * </script>
 * <div use:reveal={{ threshold: 0.3, onChange: (v) => (revealed = v) }} class:revealed>
 *   Content
 * </div>
 * ```
 */

import type { Action } from 'svelte/action';

export interface RevealParams {
	/** IO threshold 0..1. Default 0.2 (matches EditorialTrio). */
	threshold?: number;
	/** IO rootMargin. Default '0px'. */
	rootMargin?: string;
	/** If true, disconnect after first reveal. Default true. */
	once?: boolean;
	/** Delay before firing onChange(true), in ms. For stagger sequences. */
	delay?: number;
	/** Called whenever the element's intersection state flips. */
	onChange?: (revealed: boolean) => void;
}

export const reveal: Action<HTMLElement, RevealParams | undefined> = (node, params = {}) => {
	let config = resolve(params);
	let observer: IntersectionObserver | null = null;
	let timer: ReturnType<typeof setTimeout> | null = null;

	const fire = (value: boolean) => {
		if (timer) {
			clearTimeout(timer);
			timer = null;
		}
		if (config.delay > 0 && value) {
			timer = setTimeout(() => config.onChange?.(true), config.delay);
		} else {
			config.onChange?.(value);
		}
	};

	const start = () => {
		// Reduced motion: short-circuit, mark as revealed immediately.
		if (
			typeof window !== 'undefined' &&
			window.matchMedia?.('(prefers-reduced-motion: reduce)').matches
		) {
			config.onChange?.(true);
			return;
		}

		observer = new IntersectionObserver(
			(entries) => {
				for (const entry of entries) {
					if (entry.isIntersecting) {
						fire(true);
						if (config.once) {
							observer?.disconnect();
							observer = null;
						}
					} else if (!config.once) {
						fire(false);
					}
				}
			},
			{ threshold: config.threshold, rootMargin: config.rootMargin }
		);
		observer.observe(node);
	};

	start();

	return {
		update(next: RevealParams | undefined) {
			config = resolve(next ?? {});
		},
		destroy() {
			if (timer) clearTimeout(timer);
			observer?.disconnect();
		}
	};
};

function resolve(params: RevealParams) {
	return {
		threshold: params.threshold ?? 0.2,
		rootMargin: params.rootMargin ?? '0px',
		once: params.once ?? true,
		delay: params.delay ?? 0,
		onChange: params.onChange
	};
}
