<script lang="ts">
  import { onMount } from 'svelte';
  import { inject } from '@vercel/analytics';
  import '../app.css';

  onMount(() => inject());
</script>

<slot />
