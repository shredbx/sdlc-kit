<script lang="ts">
  import type { Step } from '../content/types';

  let { steps }: { steps: Step[] } = $props();
</script>

<div class="process-grid" aria-label="How it works">
  {#each steps as step}
    <div class="step">
      <div class="step-no">{step.number}</div>
      <h3>{step.title}</h3>
      <p>{step.body}</p>
    </div>
  {/each}
</div>
