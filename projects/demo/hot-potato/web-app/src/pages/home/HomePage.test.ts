import { render, screen } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import { homePageModule } from '../../lib/server/data-modules';
import HomePage from './HomePage.svelte';

describe('HomePage', () => {
  it('renders the main restaurant message from loaded page data', async () => {
    const homePage = await homePageModule.repository.findById('home');
    if (!homePage) {
      throw new Error('Home page content was not found');
    }
    render(HomePage, { props: { homePage } });
    expect(screen.getByRole('heading', { name: /BIG.*FLAVOUR/i })).toBeTruthy();
    expect(screen.getByText('+66 63 137 7619')).toBeTruthy();
    expect(document.head.querySelector('meta[name="description"]')?.getAttribute('content')).toBe(
      'Big baked potatoes, loaded with your favourite toppings, plus salads and drinks in Hua Hin. Order takeaway from The Hot Potato on Grab.'
    );
    const structuredData = document.head.querySelector('script[type="application/ld+json"]')?.textContent;
    expect(JSON.parse(structuredData ?? '{}')).toMatchObject({
      '@type': 'Restaurant',
      openingHours: 'Mo-Su 12:00-23:00'
    });
  });
});
