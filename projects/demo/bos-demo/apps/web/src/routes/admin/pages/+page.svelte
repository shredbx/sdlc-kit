<script lang="ts">
	import { goto } from '$app/navigation';
	import { base } from '$app/paths';
	import { PagesListScreen } from '@sbx/ui-cms';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	function open(slug: string) {
		goto(`${base}/admin/pages/${slug}`);
	}
</script>

<PagesListScreen pages={data.pages} onOpen={open} />
