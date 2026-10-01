import { describe, it, expect, beforeEach } from 'vitest';
import { initMedia, resetMedia } from './config';
import { mediaUrl } from './url';

describe('mediaUrl', () => {
	beforeEach(() => {
		resetMedia();
		initMedia({ baseUrl: 'https://cdn.example.com/website' });
	});

	it('builds thumb URL', () => {
		expect(mediaUrl('team/avatar.webp', 'thumb')).toBe(
			'https://cdn.example.com/cdn-cgi/image/width=120,format=auto/website/team/avatar.webp'
		);
	});

	it('builds hero URL', () => {
		expect(mediaUrl('backgrounds/posters/hero.jpg', 'hero')).toBe(
			'https://cdn.example.com/cdn-cgi/image/width=1280,format=auto/website/backgrounds/posters/hero.jpg'
		);
	});

	it('builds card-sm URL', () => {
		expect(mediaUrl('projects/cover.png', 'card-sm')).toBe(
			'https://cdn.example.com/cdn-cgi/image/width=320,format=auto/website/projects/cover.png'
		);
	});

	it('builds card-lg URL', () => {
		expect(mediaUrl('projects/cover.png', 'card-lg')).toBe(
			'https://cdn.example.com/cdn-cgi/image/width=640,format=auto/website/projects/cover.png'
		);
	});

	it('builds full URL', () => {
		expect(mediaUrl('hero/banner.jpg', 'full')).toBe(
			'https://cdn.example.com/cdn-cgi/image/width=1920,format=auto/website/hero/banner.jpg'
		);
	});

	it('throws without init', () => {
		resetMedia();
		expect(() => mediaUrl('any.jpg', 'thumb')).toThrow('initMedia() not called');
	});

	it('strips leading slash from path', () => {
		expect(mediaUrl('/team/avatar.webp', 'thumb')).toBe(
			'https://cdn.example.com/cdn-cgi/image/width=120,format=auto/website/team/avatar.webp'
		);
	});

	it('works when baseUrl has no path prefix (domain-only)', () => {
		resetMedia();
		initMedia({ baseUrl: 'https://media.example.com' });
		expect(mediaUrl('properties/cover.jpg', 'card-lg')).toBe(
			'https://media.example.com/cdn-cgi/image/width=640,format=auto/properties/cover.jpg'
		);
	});
});
