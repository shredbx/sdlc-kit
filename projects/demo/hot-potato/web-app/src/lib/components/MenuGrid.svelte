<script lang="ts">
  import type { Link, MenuItem } from '../content/types';

  let { items, orderLink }: { items: MenuItem[]; orderLink: Link } = $props();
</script>

<div class="menu-grid" aria-label="Menu listing">
  {#each items as item}
    <article class="food-card">
      <a href={orderLink.href} target="_blank" rel="noopener noreferrer">
        <img src={item.image.src} alt={item.image.alt} />
        <div class="food-info">
          <div class="food-name">{item.name}</div>
          <div class="price">{item.priceLabel} ↗</div>
        </div>
      </a>
    </article>
  {/each}
</div>
