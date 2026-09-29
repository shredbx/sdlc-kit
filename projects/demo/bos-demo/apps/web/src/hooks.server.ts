import { createHandle } from '@sbx/bos-svelte/server';

export const handle = createHandle({
	apiUrl: process.env.PUBLIC_API_URL || 'http://localhost:5010'
});
