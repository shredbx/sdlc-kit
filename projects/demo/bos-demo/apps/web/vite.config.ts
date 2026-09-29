import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

// NOTE: no Vite `server.proxy` for /api. /api is handled by the framework's server hook (a same-origin
// pass-through, see src/hooks.server.ts) in BOTH dev and prod, so dev mirrors prod exactly.
export default defineConfig({
	plugins: [sveltekit()],
	server: {
		port: parseInt(process.env.PORT || '4010'),
		host: true,
		// The web framework is linked from the sdlc-kit checkout, five levels above this app (D24); the dev server
		// may serve its files to the browser (the components hydrate from there).
		fs: { allow: ['../../../../..'] }
	}
});
