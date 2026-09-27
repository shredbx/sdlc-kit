<script lang="ts">
	import { FooterScreen } from '@sbx/ui-settings';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();
</script>

<FooterScreen settings={data.settings} />
