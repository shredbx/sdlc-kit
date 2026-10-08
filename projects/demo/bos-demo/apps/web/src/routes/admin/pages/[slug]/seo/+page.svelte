<script lang="ts">
	import { PageSeoScreen } from '@sbx/ui-cms';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();
</script>

<PageSeoScreen slug={data.slug} page={data.page} />
