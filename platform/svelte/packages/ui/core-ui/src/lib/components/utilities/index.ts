/**
 * Utilities Module - Cross-cutting UI concerns
 *
 * Utility components handle cross-cutting concerns like transitions,
 * animations, loading states, and other UI infrastructure.
 *
 * @layer utilities (cross-cutting in modified atomic design)
 *
 * Categorization Rule:
 * - Utilities are cross-cutting concerns
 * - They provide infrastructure (transitions, loading, errors)
 * - They can be used at any level of the component hierarchy
 */

// Loading state utilities
export { default as Skeleton } from './Skeleton.svelte';
export { default as SkeletonPresenter } from './Skeleton.presenter';

