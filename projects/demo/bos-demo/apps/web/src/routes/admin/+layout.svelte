<script lang="ts">
	import { AdminShell } from '@sbx/bos-svelte/admin';
	import { registeredAdminModule as cmsModule } from '@sbx/ui-cms';
	import { registeredAdminModule as settingsModule } from '@sbx/ui-settings';
	import { invalidateAll } from '$app/navigation';
	import type { Snippet } from 'svelte';
	import type { LayoutData } from './$types';

	let { data, children }: { data: LayoutData; children: Snippet } = $props();

	// This is bos-demo's own composition root for the admin — the enabled-kit list
	// lives here as plain imports, the same way main.go composes the API by calling
	// each enabled kit's RegisterAdmin directly rather than reflecting over a config
	// file. Each kit declares its own nav entry (docs/proposals/bos-admin-modules.md
	// §6); this file only decides which kits are enabled, never what they're called
	// or where they link.
	const enabledModules = [cmsModule, settingsModule];
	const navItems = enabledModules.map((m) => m.navItem);

	// Dev-only: there is no real login yet, so acting as a different fake role (see
	// platform/go/frameworks/bos-go/authfake's RoleCookie) is how content-manager vs admin gets
	// exercised at all. A real login would replace this with an actual account switch.
	const ROLES = ['admin', 'content-manager'];

	function actAs(role: string) {
		document.cookie = `bosdemo_role=${role}; path=/`;
		invalidateAll();
	}
</script>

<AdminShell {navItems} role={data.role}>
	{@render children()}
</AdminShell>

<div class="dev-role-switcher">
	Acting as:
	<select value={data.role ?? 'admin'} onchange={(e) => actAs(e.currentTarget.value)}>
		{#each ROLES as r (r)}
			<option value={r}>{r}</option>
		{/each}
	</select>
	<span class="hint">dev only — no real login yet</span>
</div>

<style>
	.dev-role-switcher {
		position: fixed;
		bottom: 0.75rem;
		right: 0.75rem;
		background: #222;
		color: #fff;
		padding: 0.4rem 0.75rem;
		border-radius: 0.375rem;
		font-family: sans-serif;
		font-size: 0.8rem;
		display: flex;
		align-items: center;
		gap: 0.4rem;
		z-index: 50;
	}
	.hint {
		opacity: 0.6;
	}
</style>
