import { describe, it, expect } from 'vitest';
import {
	escapeXml,
	renderPromptDocument,
	type PromptBlock
} from './PromptDocument';

describe('escapeXml', () => {
	it('escapes the three structural characters and leaves quotes alone', () => {
		expect(escapeXml('a & b < c > d "e" \'f\'')).toBe('a &amp; b &lt; c &gt; d "e" \'f\'');
	});

	it('escapes & first so introduced entities are not double-escaped', () => {
		expect(escapeXml('<tag>')).toBe('&lt;tag&gt;');
		expect(escapeXml('&amp;')).toBe('&amp;amp;');
	});
});

describe('renderPromptDocument — xml', () => {
	it('wraps each section as <name>text</name> with the body escaped', () => {
		const blocks: PromptBlock[] = [{ name: 'role', text: 'You are a helpful agent.' }];
		expect(renderPromptDocument(blocks, 'xml')).toBe('<role>You are a helpful agent.</role>');
	});

	it('escapes & < > in the body but not the tag', () => {
		const blocks: PromptBlock[] = [{ name: 'rules', text: 'use < and > and & here' }];
		expect(renderPromptDocument(blocks, 'xml')).toBe(
			'<rules>use &lt; and &gt; and &amp; here</rules>'
		);
	});

	it('separates multiple sections with a blank line', () => {
		const blocks: PromptBlock[] = [
			{ name: 'role', text: 'agent' },
			{ name: 'tone', text: 'friendly' }
		];
		expect(renderPromptDocument(blocks, 'xml')).toBe('<role>agent</role>\n\n<tone>friendly</tone>');
	});
});

describe('renderPromptDocument — markdown', () => {
	it('renders ## name then a blank line then the verbatim body', () => {
		const blocks: PromptBlock[] = [{ name: 'Role', text: 'You are a helpful agent.' }];
		expect(renderPromptDocument(blocks, 'markdown')).toBe('## Role\n\nYou are a helpful agent.');
	});

	it('passes the body through verbatim (fidelity-preserving — no escaping)', () => {
		const blocks: PromptBlock[] = [{ name: 'Rules', text: 'use < and > and & here' }];
		expect(renderPromptDocument(blocks, 'markdown')).toBe('## Rules\n\nuse < and > and & here');
	});

	it('emits only the heading when the body is empty', () => {
		const blocks: PromptBlock[] = [{ name: 'Notes', text: '   ' }];
		expect(renderPromptDocument(blocks, 'markdown')).toBe('## Notes');
	});

	it('separates multiple sections with a blank line', () => {
		const blocks: PromptBlock[] = [
			{ name: 'Role', text: 'agent' },
			{ name: 'Tone', text: 'friendly' }
		];
		expect(renderPromptDocument(blocks, 'markdown')).toBe(
			'## Role\n\nagent\n\n## Tone\n\nfriendly'
		);
	});
});

describe('renderPromptDocument — skipping', () => {
	it('skips hidden sections in both formats', () => {
		const blocks: PromptBlock[] = [
			{ name: 'role', text: 'agent' },
			{ name: 'secret', text: 'do not ship', hidden: true }
		];
		expect(renderPromptDocument(blocks, 'xml')).toBe('<role>agent</role>');
		expect(renderPromptDocument(blocks, 'markdown')).toBe('## role\n\nagent');
	});

	it('skips a section whose title AND body are both blank', () => {
		const blocks: PromptBlock[] = [
			{ name: '  ', text: '  ' },
			{ name: 'role', text: 'agent' }
		];
		expect(renderPromptDocument(blocks, 'xml')).toBe('<role>agent</role>');
	});

	it('returns an empty string for an empty or all-skipped document', () => {
		expect(renderPromptDocument([], 'xml')).toBe('');
		expect(renderPromptDocument([{ name: '', text: '' }], 'markdown')).toBe('');
	});
});

describe('renderPromptDocument — xml tag-name safety', () => {
	it('slugifies a free-form title so it cannot forge an attribute', () => {
		const blocks: PromptBlock[] = [{ name: 'role onload="x"', text: 'body' }];
		expect(renderPromptDocument(blocks, 'xml')).toBe('<role-onload-x>body</role-onload-x>');
	});

	it('slugifies a structural title so the markup stays balanced', () => {
		const blocks: PromptBlock[] = [{ name: 'a><b', text: 'c' }];
		expect(renderPromptDocument(blocks, 'xml')).toBe('<a-b>c</a-b>');
	});

	it('falls back to <section> for a body-only section (blank title)', () => {
		const blocks: PromptBlock[] = [{ name: '', text: 'orphan body' }];
		expect(renderPromptDocument(blocks, 'xml')).toBe('<section>orphan body</section>');
	});

	it('leaves a clean identifier-like name unchanged (python parity)', () => {
		const blocks: PromptBlock[] = [{ name: 'system_prompt-1', text: 'x' }];
		expect(renderPromptDocument(blocks, 'xml')).toBe('<system_prompt-1>x</system_prompt-1>');
	});

	it('keeps the free-form title verbatim in markdown (prose, not structure)', () => {
		const blocks: PromptBlock[] = [{ name: 'Role & Context', text: 'x' }];
		expect(renderPromptDocument(blocks, 'markdown')).toBe('## Role & Context\n\nx');
	});
});

describe('renderPromptDocument — body whitespace', () => {
	it('never trims the body (author whitespace is intentional)', () => {
		const blocks: PromptBlock[] = [{ name: 'code', text: '  keep \n  me  ' }];
		expect(renderPromptDocument(blocks, 'xml')).toBe('<code>  keep \n  me  </code>');
	});
});
