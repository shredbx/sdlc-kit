<script lang="ts">
	import { PageSettingsScreen } from '@sbx/ui-cms';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();
</script>

<PageSettingsScreen slug={data.slug} page={data.page} />
