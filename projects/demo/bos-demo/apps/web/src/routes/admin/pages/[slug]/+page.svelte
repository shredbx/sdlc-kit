<script lang="ts">
	import { PageContentScreen } from '@sbx/ui-cms';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();
</script>

<PageContentScreen slug={data.slug} page={data.page} />
