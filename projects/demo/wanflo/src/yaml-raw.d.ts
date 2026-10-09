// Vite `?raw` imports return the file contents as a string. Used by the content loader
// (src/lib/content/index.ts) to bake content/*.yml — the owner's single editing surface —
// in at build time. The v1 CSV loader that used the same pattern is gone.
declare module '*.yml?raw' {
	const content: string;
	export default content;
}
