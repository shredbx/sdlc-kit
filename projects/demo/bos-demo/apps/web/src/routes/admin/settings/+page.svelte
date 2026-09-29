<script lang="ts">
	import { GeneralScreen } from '@sbx/ui-settings';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();
</script>

<GeneralScreen settings={data.settings} />
