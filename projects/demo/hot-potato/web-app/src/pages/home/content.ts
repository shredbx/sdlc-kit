import type { Link, MenuItem, Step } from '../../lib/content/types.ts';
import { restaurant } from '../../config/restaurant.ts';

export const links = {
  grab: {
    label: 'ORDER ON GRAB',
    href: 'https://app.grab.com/s/tTK0lZHp',
    external: true
  },
  maps: {
    label: 'GET DIRECTIONS',
    href: 'https://www.google.com/maps/search/?api=1&query=The%20Hot%20Potato%20Hua%20Hin',
    external: true
  },
  phone: {
    label: 'CALL',
    href: restaurant.phoneHref
  }
};

export const nav: Link[] = [
  { label: 'Menu', href: '#menu' },
  { label: 'Our potato', href: '#story' },
  { label: 'Find us', href: '#find-us' }
];

export const menuItems: MenuItem[] = [
  {
    name: 'Baked Potato with Chili Con Carne',
    image: { src: '/assets/menu/chili-potato.jpg', alt: 'Baked potato with chili con carne' },
    priceLabel: 'SEE CURRENT PRICE ON GRAB'
  },
  {
    name: 'Baked Potato with Beans & Cheese',
    image: { src: '/assets/menu/beans-cheese.jpg', alt: 'Baked potato with beans and cheese' },
    priceLabel: 'SEE CURRENT PRICE ON GRAB'
  },
  {
    name: 'Chili Con Carne & Rice',
    image: { src: '/assets/menu/chili-rice.jpg', alt: 'Chili con carne with rice' },
    priceLabel: 'SEE CURRENT PRICE ON GRAB'
  },
  {
    name: 'Baked Potato with BBQ Pulled Pork',
    image: { src: '/assets/menu/pulled-pork.jpg', alt: 'Baked potato with BBQ pulled pork' },
    priceLabel: 'SEE CURRENT PRICE ON GRAB'
  },
  {
    name: 'Baked Potato with Curry Chicken',
    image: { src: '/assets/menu/curry-chicken.jpg', alt: 'Baked potato with curry chicken' },
    priceLabel: 'SEE CURRENT PRICE ON GRAB'
  },
  {
    name: 'Baked Potato with Tuna Mayonnaise',
    image: { src: '/assets/menu/tuna-potato.jpg', alt: 'Baked potato with tuna mayonnaise' },
    priceLabel: 'SEE CURRENT PRICE ON GRAB'
  },
  {
    name: 'Vegetarian Chilli Con Carn Beans & Soya',
    image: { src: '/assets/menu/chili-con-vegetarian.jpg', alt: 'Vegetarian chilli with beans and soya' },
    priceLabel: 'SEE CURRENT PRICE ON GRAB'
  },
  {
    name: 'Baked Potato with Cheese & Bacon',
    image: { src: '/assets/menu/crispy-onion-rings.jpg', alt: 'Baked potato with cheese and bacon' },
    priceLabel: 'SEE CURRENT PRICE ON GRAB'
  }
];

export const processSteps: Step[] = [
  { number: '01', title: 'BAKE', body: 'Get that big potato hot and fluffy.' },
  { number: '02', title: 'OPEN', body: 'Split it open and let the heat out.' },
  {
    number: '03',
    title: 'LOAD',
    body: 'Add cheese, meat, curry, beans or your favourite combination.'
  },
  { number: '04', title: 'EAT', body: 'Stay, take away, or order it to your door.' }
];
