<script lang="ts">
	import { SettingsLayout } from '@sbx/ui-settings';
	import type { Snippet } from 'svelte';

	let { children }: { children: Snippet } = $props();
</script>

<SettingsLayout>
	{@render children()}
</SettingsLayout>
