// SC-C1b: every new Prompt Studio component ships a *.presenter.ts so it auto-registers into the
// SAME presenter gallery (componentRegistry's import.meta.glob discovery) — no bespoke /gallery
// route. RED until each *.presenter.ts exists with variants covering both themes.
//
// Imports each presenter module DIRECTLY (not via componentRegistry's eager glob of every presenter
// in the package) — the eager glob drags in unrelated presenters like DotNav.presenter.ts, which
// imports $app/stores and only resolves inside a SvelteKit app, not this package's plain vitest
// environment. A direct, single-module import isolates the assertion to the file under test.
import { describe, expect, it } from 'vitest';
import type { IPresenter } from './IPresenter';

const EXPECTED: Array<{ slug: string; modulePath: string }> = [
	{ slug: 'section-block', modulePath: '../components/sections/SectionBlock.presenter.ts' },
	{ slug: 'preset-pill', modulePath: '../components/sections/PresetPill.presenter.ts' },
	{ slug: 'arg-token', modulePath: '../components/sections/ArgToken.presenter.ts' },
	{ slug: 'model-controls', modulePath: '../components/sections/ModelControls.presenter.ts' },
	{ slug: 'response-view', modulePath: '../components/sections/ResponseView.presenter.ts' },
	{ slug: 'debug-panel', modulePath: '../components/sections/DebugPanel.presenter.ts' },
	{ slug: 'log-row', modulePath: '../components/sections/LogRow.presenter.ts' },
	{ slug: 'panel', modulePath: '../components/primitives/Panel.presenter.ts' }
];

describe('SC-C1b — new prompt components register in the presenter gallery', () => {
	for (const { slug, modulePath } of EXPECTED) {
		// 20s timeout: the first cold import of a heavy .svelte component (SectionBlock is recursive and
		// pulls in 3 children) can exceed vitest's 5s default on the vite-plugin-svelte transform alone.
		it(
			`${slug} has a *.presenter.ts exporting a conformant IPresenter with ≥1 variant`,
			async () => {
				const mod: { default: IPresenter } = await import(/* @vite-ignore */ modulePath);
				expect(mod.default.slug, `presenter default export slug`).toBe(slug);
				expect(mod.default.variants?.length ?? 0).toBeGreaterThan(0);
			},
			20000
		);
	}
});
