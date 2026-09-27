<script lang="ts">
	import type { ApiHealth } from './health';

	let { siteName = 'bos', api }: { siteName?: string; api: ApiHealth } = $props();
</script>

<main>
	<h1>{siteName}: it works.</h1>
	<p data-testid="api-status">API: {api.status}</p>
	{#if api.status === 'healthy'}
		<dl>
			<dt>service</dt>
			<dd>{api.service}</dd>
			<dt>version</dt>
			<dd>{api.version}</dd>
			<dt>environment</dt>
			<dd>{api.environment}</dd>
			<dt>database</dt>
			<dd>{api.database}</dd>
		</dl>
	{/if}
</main>

<style>
	main {
		font-family: system-ui, sans-serif;
		max-width: 40rem;
		margin: 4rem auto;
		padding: 0 1rem;
	}
	dl {
		display: grid;
		grid-template-columns: max-content 1fr;
		gap: 0.25rem 1rem;
	}
	dt {
		color: #555;
	}
	dd {
		margin: 0;
	}
</style>
