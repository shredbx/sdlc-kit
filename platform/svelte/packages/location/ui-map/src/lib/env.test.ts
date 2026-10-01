import { describe, expect, it } from 'vitest';
import { resolveApiKey, resolveProvider } from './env.js';

describe('resolveProvider', () => {
	it('explicit prop wins over everything', () => {
		expect(resolveProvider({ provider: 'mapbox', surface: 'admin', envProvider: 'google' })).toBe(
			'mapbox'
		);
	});
	it('surface=admin defaults to google', () => {
		expect(resolveProvider({ surface: 'admin' })).toBe('google');
	});
	it('surface=public defaults to mapbox', () => {
		expect(resolveProvider({ surface: 'public' })).toBe('mapbox');
	});
	it('surface beats env when both set', () => {
		expect(resolveProvider({ surface: 'admin', envProvider: 'mapbox' })).toBe('google');
	});
	it('env is honored when no surface and no prop', () => {
		expect(resolveProvider({ envProvider: 'mapbox' })).toBe('mapbox');
	});
	it('falls back to google when nothing is set', () => {
		expect(resolveProvider({})).toBe('google');
	});
	it('ignores invalid env value', () => {
		expect(resolveProvider({ envProvider: 'bogus' })).toBe('google');
	});
});

describe('resolveApiKey', () => {
	it('returns google key for google provider', () => {
		expect(resolveApiKey({ provider: 'google', googleKey: 'AIza', mapboxToken: 'pk.' })).toBe(
			'AIza'
		);
	});
	it('returns mapbox token for mapbox provider', () => {
		expect(resolveApiKey({ provider: 'mapbox', googleKey: 'AIza', mapboxToken: 'pk.' })).toBe(
			'pk.'
		);
	});
	it('returns undefined when key for resolved provider is missing', () => {
		expect(resolveApiKey({ provider: 'google', mapboxToken: 'pk.' })).toBeUndefined();
		expect(resolveApiKey({ provider: 'mapbox', googleKey: 'AIza' })).toBeUndefined();
	});
	it('treats empty string as missing', () => {
		expect(resolveApiKey({ provider: 'google', googleKey: '' })).toBeUndefined();
	});
});
