// SC-C1a red test: every new Prompt Studio component must be fully controlled — no store or API
// import, styled only via var(--token/--color-*). Enforced statically (source-text scan) rather than
// by convention, per 2607-054's test-plan.yml risk note: pilot on an already-dumb component
// (PromptDocumentEditor) before trusting the check against new ones.
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';

const HERE = fileURLToPath(new URL('.', import.meta.url));

// Value imports that would make a component non-dumb. `import type` is exempt (type-only, erased at
// build time — carries no runtime store/API coupling).
const FORBIDDEN_IMPORT_PATTERNS = [/from ['"]\$app\//, /from ['"]\$lib\/storage/, /from ['"]\$lib\/stores/, /\bfetch\(/];

function assertControlled(relativePath: string) {
	const source = readFileSync(`${HERE}${relativePath}`, 'utf-8');
	const valueImportLines = source
		.split('\n')
		.filter((line) => /^\s*import\b/.test(line) && !/^\s*import type\b/.test(line));
	for (const pattern of FORBIDDEN_IMPORT_PATTERNS) {
		const hit = valueImportLines.find((line) => pattern.test(line)) ?? (pattern.test(source) && source.match(pattern)?.[0]);
		expect(hit, `${relativePath} must not import ${pattern} as a value`).toBeFalsy();
	}
}

describe('SC-C1a — controlled-component import boundary (pilot: known-dumb component)', () => {
	it('PromptDocumentEditor (existing, already dumb) has no store/API import', () => {
		assertControlled('sections/PromptDocumentEditor.svelte');
	});
});

describe('SC-C1a — new prompt components stay controlled (RED until each file exists)', () => {
	const NEW_COMPONENTS = [
		'sections/SectionBlock.svelte',
		'sections/SectionAddRow.svelte',
		'sections/PresetPill.svelte',
		'sections/ArgToken.svelte',
		'sections/ModelControls.svelte',
		'sections/VariablesForm.svelte',
		'sections/AssembledPromptPreview.svelte',
		'sections/ResponseView.svelte',
		'sections/DebugPanel.svelte',
		'sections/LogRow.svelte',
		'primitives/Panel.svelte'
	];

	for (const path of NEW_COMPONENTS) {
		it(`${path} is controlled — no store/API import`, () => {
			assertControlled(path);
		});
	}
});
