/**
 * Animation Library — Svelte action wrappers for @sbx/animations
 *
 * Provides `use:directive` actions for all animation effects.
 * Each action wraps a core TS animation function with Svelte lifecycle management.
 *
 * Categories:
 * - Text: diagonalSlice, glitchSlice, quadShatter, doubleVision, textSplit, glitch, weightShift, delayedReveal
 * - Gradient: plasmaGradient, colorTemp
 * - Reveal: accentLineReveal
 * - Background: gradientMesh, dotGrid
 *
 * @example
 * ```svelte
 * <script>
 *   import { diagonalSlice, plasmaGradient, dotGrid } from '$lib/components/animations';
 * </script>
 *
 * <h1 use:diagonalSlice={{ angle: 35 }}>Split Text</h1>
 * <h2 use:plasmaGradient>Rainbow Text</h2>
 * <div use:dotGrid={{ spacing: 24 }}>Background Grid</div>
 * ```
 */

export * from './actions';
export { default as TypewriterReveal } from './TypewriterReveal.svelte';
