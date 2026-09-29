<script lang="ts">
	import { HeaderScreen } from '@sbx/ui-settings';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();
</script>

<HeaderScreen settings={data.settings} />
