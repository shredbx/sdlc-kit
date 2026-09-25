export interface MediaConfig {
	baseUrl: string;
}

let config: MediaConfig | null = null;

export function initMedia(c: MediaConfig): void {
	config = c;
}

export function getMediaConfig(): MediaConfig {
	if (!config) throw new Error('initMedia() not called — add to hooks.server.ts');
	return config;
}

export function resetMedia(): void {
	config = null;
}
