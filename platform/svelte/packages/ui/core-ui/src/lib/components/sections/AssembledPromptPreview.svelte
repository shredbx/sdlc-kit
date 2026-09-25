<script lang="ts">
	/**
	 * AssembledPromptPreview — read-only view of the composed prompt (dumb). Markdown heading lines
	 * (`# ` at any depth) are highlighted as section keys (mockup `.sk`); everything else is mono
	 * prose. Imports no store/API.
	 */
	interface Props {
		text?: string;
	}
	let { text = '' }: Props = $props();

	const lines = $derived(text.split('\n'));
	const isKey = (line: string) => /^#{1,6} /.test(line);
</script>

<div class="assembled" data-testid="assembled-preview">{#each lines as line, i}{#if isKey(line)}<span class="sk">{line}</span>{:else}{line}{/if}{i < lines.length - 1 ? '\n' : ''}{/each}</div>

<style>
	.assembled {
		font-family: var(--font-family-mono, ui-monospace, monospace);
		font-size: 12.5px;
		line-height: 1.7;
		color: var(--color-code);
		white-space: pre-wrap;
		max-height: 240px;
		overflow: auto;
	}
	.assembled .sk {
		color: var(--color-accent-ink);
		font-weight: 600;
	}
</style>
