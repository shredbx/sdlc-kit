/**
 * Admin Types — Shared type definitions for admin control panels.
 *
 * Used by AdminTemplate, UserProfileCard, AdminNavItem, and any
 * project-specific control center pages.
 */

/** Authenticated admin user for sidebar display and role gating. */
export interface AdminUser {
	id: string;
	email: string;
	role: string;
	name?: string;
	avatarUrl?: string;
}

/** A navigable section in the admin sidebar. */
export interface AdminSection {
	id: string;
	label: string;
	/** Emoji or icon name displayed before the label. */
	icon?: string;
	/** Roles allowed to see this section. Empty array = visible to all. */
	requiredRoles: string[];
	/** Notification count pill. Omit or 0 to hide. */
	badge?: number;
}

/** Top-level configuration for the AdminTemplate layout. */
export interface AdminConfig {
	projectName: string;
	/** URL the project name links to (e.g. "/" or "/dashboard"). */
	logoHref?: string;
	sections: AdminSection[];
}
