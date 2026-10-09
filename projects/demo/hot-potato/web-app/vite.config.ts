import { svelteTesting } from "@testing-library/svelte/vite";
import { sveltekit } from "@sveltejs/kit/vite";
import { defineConfig } from "vitest/config";
import { readFileSync } from "node:fs";

const version = (JSON.parse(
  readFileSync(new URL("./package.json", import.meta.url), "utf8")
) as { version: string }).version;

export default defineConfig({
  plugins: [sveltekit(), svelteTesting()],
  define: { __APP_VERSION__: JSON.stringify(version) },
  server: {
    port: 3400,
    host: true,
  },
  build: { sourcemap: false },
  test: { environment: "jsdom", include: ["src/**/*.test.ts"] },
});
