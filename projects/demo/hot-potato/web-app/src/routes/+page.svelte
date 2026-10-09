<script lang="ts">
  import HomePage from '../pages/home/HomePage.svelte';
  import type { PageData } from './$types';

  let { data }: { data: PageData } = $props();
</script>

<HomePage homePage={data.homePage} />
