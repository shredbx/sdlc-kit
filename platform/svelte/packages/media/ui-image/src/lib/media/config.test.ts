import { describe, it, expect, beforeEach } from 'vitest';
import { initMedia, getMediaConfig, resetMedia } from './config';

describe('media config', () => {
	beforeEach(() => {
		resetMedia();
	});

	it('throws when getMediaConfig called before init', () => {
		expect(() => getMediaConfig()).toThrow('initMedia() not called');
	});

	it('returns config after init', () => {
		initMedia({ baseUrl: 'https://example.com' });
		expect(getMediaConfig().baseUrl).toBe('https://example.com');
	});

	it('is idempotent — second init overwrites', () => {
		initMedia({ baseUrl: 'https://first.com' });
		initMedia({ baseUrl: 'https://second.com' });
		expect(getMediaConfig().baseUrl).toBe('https://second.com');
	});
});
