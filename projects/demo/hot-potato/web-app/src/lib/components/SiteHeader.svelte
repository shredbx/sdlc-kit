<script lang="ts">
  import type { Link } from '../content/types';

  let { nav, orderLink }: { nav: Link[]; orderLink: Link } = $props();

  const target = (link: Link) => (link.external ? '_blank' : undefined);
  const rel = (link: Link) => (link.external ? 'noopener' : undefined);
</script>

<header class="topbar">
  <div class="container">
    <nav aria-label="Main navigation">
      <a class="brand" href="#top" aria-label="The Hot Potato home">
        <img src="/assets/hot-potato-logo.png" alt="The Hot Potato logo" />
        THE HOT <span>POTATO</span>
      </a>

      <div class="navlinks">
        {#each nav as item}
          <a href={item.href}>{item.label}</a>
        {/each}
      </div>

      <a class="nav-cta" href={orderLink.href} target={target(orderLink)} rel={rel(orderLink)}>
        {orderLink.label} ↗
      </a>
    </nav>
  </div>
</header>
