<script lang="ts">
  import SiteHeader from '../../lib/components/SiteHeader.svelte';
  import MenuGrid from '../../lib/components/MenuGrid.svelte';
  import ProcessSteps from '../../lib/components/ProcessSteps.svelte';
  import SiteFooter from '../../lib/components/SiteFooter.svelte';
  import { restaurant } from '../../config/restaurant';
  import { site } from '../../config/site';
  import SeoHead from '../../lib/seo/SeoHead.svelte';
  import { restaurantJsonLd } from './structuredData';
  import { homeSeo } from './seo';
  import type { homePageSchema } from '../../lib/server/data-modules';
  import type { z } from 'zod';

  type HomePageData = z.infer<typeof homePageSchema>;

  let { homePage }: { homePage: HomePageData } = $props();
</script>

<svelte:head>
  <link rel="preconnect" href="https://fonts.googleapis.com" />
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin="anonymous" />
  <link
    href="https://fonts.googleapis.com/css2?family=Anton&family=DM+Sans:wght@400;500;600;700&display=swap"
    rel="stylesheet"
  />
</svelte:head>
<SeoHead seo={homeSeo} structuredData={restaurantJsonLd} />

<SiteHeader nav={homePage.nav} orderLink={homePage.links.grab} />

<main>
  <section class="hero">
    <div class="container hero-grid">
      <div>
        <div class="eyebrow">Hua Hin · Baked fresh</div>
        <h1>BIG<br /><span>POTATO.</span><br />BIG FLAVOUR.</h1>
        <p class="hero-copy">
          A hot, fluffy baked potato loaded with the good stuff. Choose your favourite topping,
          grab a drink, and make it a meal.
        </p>
        <div class="actions">
          <a class="btn btn-primary" href={homePage.links.grab.href} target="_blank" rel="noopener noreferrer">
            ORDER ON GRAB ↗
          </a>
          <a class="btn btn-secondary" href="#menu">SEE THE POTATOES</a>
        </div>
      </div>

      <div class="hero-photo">
        <img src="/assets/promo-dish.jpg" alt="Loaded baked potato with melted cheese and bacon" />
        <div class="heat">HOT · LOADED · READY</div>
      </div>
    </div>
  </section>

  <section class="intro" id="story">
    <div class="container intro-grid">
      <div>
        <div class="big-number">01</div>
      </div>
      <div>
        <div class="section-kicker">The idea</div>
        <h2>One big potato.<br />So many ways.</h2>
        <p class="lead">
          We take a proper baked potato, open it while it's hot, and load it with flavours that turn
          a simple potato into a full meal.
        </p>
        <div class="intro-note">
          Cheesy, spicy, creamy, meaty or veggie — the potato is the starting point.
        </div>
      </div>
    </div>
  </section>

  <section id="menu">
    <div class="container">
      <div class="menu-head">
        <div>
          <div class="section-kicker">From the current menu</div>
          <h2>LOAD IT UP.</h2>
          <p class="lead">
            A few favourites from the menu. Prices and the full current selection are kept live on Grab.
          </p>
        </div>
        <a class="btn btn-primary" href={homePage.links.grab.href} target="_blank" rel="noopener noreferrer">
          FULL MENU ON GRAB ↗
        </a>
      </div>

      <MenuGrid items={homePage.menuItems} orderLink={homePage.links.grab} />
    </div>
  </section>

  <section class="process">
    <div class="container">
      <div class="section-kicker">The hot potato ritual</div>
      <h2>BAKE. OPEN. LOAD. EAT.</h2>
      <ProcessSteps steps={homePage.processSteps} />
    </div>
  </section>

  <section class="reviews">
    <div class="container">
      <div class="section-kicker">Real people, real reviews</div>
      <h2>WHAT GUESTS SAY</h2>
      <div class="review-card">
        <div>
          <div class="stars">★★★★★</div>
          <strong>5.0 on Google</strong>
        </div>
        <p class="review-note">
          The hot potato is very yummy!! Very friendly atmosphere, we'll be coming here more for
          sure. Thank you 🙏🏼
        </p>
        <a class="btn btn-secondary" href={homePage.links.maps.href} target="_blank" rel="noopener noreferrer">
          READ ON GOOGLE ↗
        </a>
      </div>
    </div>
  </section>

  <section class="location" id="find-us">
    <div class="container location-grid">
      <div>
        <div class="section-kicker">Come hungry</div>
        <h2>FIND US<br />IN HUA HIN.</h2>
        <p class="address">
          <strong>{site.name}</strong><br />{restaurant.streetAddress}<br />{restaurant.locality},
          {restaurant.region} {restaurant.postalCode}
        </p>
        <p><strong>{restaurant.openingHoursLabel}</strong><br />{restaurant.telephoneDisplay}</p>
        <div class="actions">
          <a class="btn btn-primary" href={homePage.links.maps.href} target="_blank" rel="noopener noreferrer">
            GET DIRECTIONS ↗
          </a>
          <a class="btn btn-secondary" href={homePage.links.phone.href}>CALL</a>
        </div>
      </div>
      <a class="map" href={homePage.links.maps.href} target="_blank" rel="noopener noreferrer" aria-label="Open Google Maps directions">
        <span class="map-cta">OPEN GOOGLE MAPS ↗</span>
      </a>
    </div>
  </section>
</main>

<SiteFooter />

<a class="mobile-order" href={homePage.links.grab.href} target="_blank" rel="noopener noreferrer">🥔 ORDER ON GRAB ↗</a>
