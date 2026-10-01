<script lang="ts">
	import { Field, Input, Textarea } from '@sbx/core-ui/components/primitives';
	import { resolveSeo } from './resolve';
	import { fromSeoDraft, type SeoDraft } from './draft';
	import type { SeoDefaults, SeoContext } from './types';
	import SeoSearchPreview from './SeoSearchPreview.svelte';
	import SeoSocialPreview from './SeoSocialPreview.svelte';

	/**
	 * SeoEditor — the reusable per-entity SEO override form + live SERP/social previews.
	 * Generic across offerings / content / static pages: the consumer supplies the
	 * entity-derived `defaults` (title/description/cover) + request `context`, and binds a
	 * `draft` (see ./draft). The previews render from resolveSeo(draft, defaults, context)
	 * — the SAME resolution SeoHead emits, so a preview can't drift from the live <head>.
	 *
	 * Primary fields: meta title + description, with SERP-length counters that fall back to
	 * the derived defaults when blank. Search visibility (noindex) is the one v1 control the
	 * legacy field-set lacked. Open Graph + canonical are demoted into an advanced group —
	 * they inherit from the primaries/cover unless explicitly overridden. meta_keywords is
	 * intentionally absent (dead since 2009; carried through unchanged on save).
	 */
	interface Props {
		/** The editor working copy — seed with toSeoDraft(stored), bind for live edits. */
		draft: SeoDraft;
		/** Entity-derived fallbacks (property title/cover, article title/hero, …). */
		defaults: SeoDefaults;
		/** Request context for the previews' absolute canonical URL. */
		context: SeoContext;
		/** SERP-length guidance — counters warn past these (also the backend hard caps). */
		titleMax?: number;
		descMax?: number;
	}
	let { draft = $bindable(), defaults, context, titleMax = 60, descMax = 160 }: Props = $props();

	// Live resolution — drives BOTH previews. fromSeoDraft drops empties so each preview
	// shows the derived default until the agent overrides it.
	const resolved = $derived(resolveSeo(fromSeoDraft(draft), defaults, context));

	const titleLen = $derived(draft.meta_title.trim().length);
	const descLen = $derived(draft.meta_description.trim().length);

	let advancedOpen = $state(false);
</script>

<div class="seo-editor">
	<div class="seo-editor__form">
		<div class="seo-pageurl">
			<span class="seo-pageurl__label">Page URL</span>
			<span class="seo-pageurl__value">{resolved.canonical}</span>
		</div>

		<Field label="Meta title" for="seo-meta-title">
			<Input
				id="seo-meta-title"
				name="seo_meta_title"
				type="text"
				autocomplete="off"
				placeholder={defaults.title}
				bind:value={draft.meta_title}
			/>
			<div class="seo-foot">
				<span class="seo-count" class:seo-count--over={titleLen > titleMax}>{titleLen}/{titleMax}</span>
				{#if titleLen === 0}
					<span class="seo-inherit">Inherits “{defaults.title}”</span>
				{/if}
			</div>
		</Field>

		<Field label="Meta description" for="seo-meta-description">
			<Textarea
				id="seo-meta-description"
				name="seo_meta_description"
				rows={3}
				autocomplete="off"
				placeholder={defaults.description}
				bind:value={draft.meta_description}
			/>
			<div class="seo-foot">
				<span class="seo-count" class:seo-count--over={descLen > descMax}>{descLen}/{descMax}</span>
				{#if descLen === 0}
					<span class="seo-inherit">Inherits the listing summary</span>
				{/if}
			</div>
		</Field>

		<div class="seo-visibility">
			<label class="seo-check">
				<input type="checkbox" name="seo_noindex" bind:checked={draft.noindex} />
				<span class="seo-check__text">
					<span class="seo-check__label">Hide from search engines</span>
					<span class="seo-check__help">
						Adds a <code>noindex</code> tag — the page won’t appear in Google or Bing
						results. Leave off for normal listings.
					</span>
				</span>
			</label>
		</div>

		<details class="seo-advanced" bind:open={advancedOpen}>
			<summary>
				Social &amp; advanced
				<span class="seo-advanced__hint">— Open Graph + canonical overrides (optional)</span>
			</summary>
			<div class="seo-advanced__body">
				<Field
					label="Social title (og:title)"
					for="seo-og-title"
					help="Shown on Facebook / LinkedIn / Slack shares. Falls back to the meta title."
				>
					<Input
						id="seo-og-title"
						name="seo_og_title"
						type="text"
						autocomplete="off"
						placeholder={resolved.finalTitle}
						bind:value={draft.og_title}
					/>
				</Field>
				<Field
					label="Social description (og:description)"
					for="seo-og-description"
					help="Falls back to the meta description."
				>
					<Textarea
						id="seo-og-description"
						name="seo_og_description"
						rows={2}
						autocomplete="off"
						placeholder={resolved.finalDesc}
						bind:value={draft.og_description}
					/>
				</Field>
				<Field
					label="Social image (og:image)"
					for="seo-og-image"
					help="Absolute URL of a 1200×630 share image. Falls back to the cover image."
				>
					<Input
						id="seo-og-image"
						name="seo_og_image"
						type="url"
						autocomplete="off"
						placeholder={defaults.image ?? defaults.defaultImage}
						bind:value={draft.og_image}
					/>
				</Field>
				<Field
					label="Canonical URL"
					for="seo-canonical"
					help="Overrides the self-referencing canonical. Leave blank unless this content also lives at another URL."
				>
					<Input
						id="seo-canonical"
						name="seo_canonical_url"
						type="url"
						autocomplete="off"
						placeholder={resolved.canonical}
						bind:value={draft.canonical_url}
					/>
				</Field>
			</div>
		</details>
	</div>

	<aside class="seo-editor__previews" aria-label="Live previews">
		<div class="seo-preview-group">
			<h4 class="seo-preview-title">Search result</h4>
			<SeoSearchPreview seo={resolved} />
		</div>
		<div class="seo-preview-group">
			<h4 class="seo-preview-title">Social share</h4>
			<SeoSocialPreview seo={resolved} />
		</div>
	</aside>
</div>

<style>
	.seo-editor {
		display: grid;
		grid-template-columns: minmax(0, 2fr) minmax(0, 1fr);
		gap: 2rem;
		align-items: start;
	}
	@media (max-width: 1024px) {
		.seo-editor {
			grid-template-columns: 1fr;
		}
	}

	.seo-editor__form {
		display: flex;
		flex-direction: column;
		gap: 1.25rem;
		min-width: 0;
	}

	/* Read-only "Page URL" — informational page address (resolved.canonical), the same URL
	   the previews' breadcrumb resolves. NOT editable: the canonical OVERRIDE lives under
	   Social & advanced. Quiet muted field; the URL wraps (never clips). */
	.seo-pageurl {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
	}
	.seo-pageurl__label {
		font-size: 0.75rem;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.04em;
		color: var(--seo-muted, #6b7280);
	}
	.seo-pageurl__value {
		font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
		font-size: 0.8rem;
		line-height: 1.45;
		color: var(--seo-muted, #6b7280);
		word-break: break-word;
	}

	.seo-foot {
		display: flex;
		justify-content: space-between;
		gap: 1rem;
		margin-top: 0.35rem;
		font-size: 0.75rem;
		color: var(--seo-muted, #6b7280);
	}
	.seo-count {
		font-variant-numeric: tabular-nums;
	}
	.seo-count--over {
		color: var(--seo-warn, #b91c1c);
		font-weight: 600;
	}
	.seo-inherit {
		font-style: italic;
	}

	.seo-visibility {
		padding: 0.25rem 0;
	}
	.seo-check {
		display: flex;
		gap: 0.6rem;
		align-items: flex-start;
		cursor: pointer;
	}
	.seo-check input {
		margin-top: 0.2rem;
		flex: none;
		width: 1rem;
		height: 1rem;
	}
	.seo-check__text {
		display: flex;
		flex-direction: column;
		gap: 0.15rem;
	}
	.seo-check__label {
		font-weight: 600;
		font-size: 0.9rem;
	}
	.seo-check__help {
		font-size: 0.8rem;
		color: var(--seo-muted, #6b7280);
		line-height: 1.5;
	}
	.seo-check code {
		font-family: ui-monospace, monospace;
		font-size: 0.85em;
	}

	.seo-advanced {
		border: 1px solid var(--seo-preview-border, #e5e7eb);
		border-radius: 8px;
		padding: 0.5rem 0.85rem;
	}
	.seo-advanced summary {
		cursor: pointer;
		font-weight: 600;
		font-size: 0.9rem;
		padding: 0.35rem 0;
	}
	.seo-advanced__hint {
		font-weight: 400;
		color: var(--seo-muted, #6b7280);
	}
	.seo-advanced__body {
		display: flex;
		flex-direction: column;
		gap: 1rem;
		padding-top: 0.85rem;
	}

	.seo-editor__previews {
		display: flex;
		flex-direction: column;
		gap: 1.5rem;
		max-width: 320px;
		position: sticky;
		/* top:0 (not 1rem): the drawer body only becomes scrollable once the form expands
		   (e.g. opening Social & advanced). At top:1rem the sticky position jumped 16px the
		   moment sticky activated vs. its inert natural-flow position; at top:0 the inert and
		   active positions coincide, so the previews stay visually stable on expand/collapse
		   while still pinning to the viewport top on real scroll. */
		top: 0;
	}
	.seo-preview-title {
		font-size: 0.72rem;
		text-transform: uppercase;
		letter-spacing: 0.06em;
		color: var(--seo-muted, #6b7280);
		margin: 0 0 0.6rem;
		font-weight: 600;
	}
</style>
