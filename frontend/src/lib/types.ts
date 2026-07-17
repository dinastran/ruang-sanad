/**
 * Shared frontend types.
 *
 * Must stay in sync with Go backend `app/models/dto.go` UserResponse.
 */

export interface User {
	id: number;
	email: string;
	name: string;
	avatar: string;
	role: string;
	email_verified: boolean;
}

/**
 * Common flash message shape from Inertia.
 * Backend sets via `s.store.Flash(c, "error", ...)` and `inertiaService.Render`
 * merges into `props.flash` automatically.
 */
export interface Flash {
	error?: string;
	success?: string;
	warning?: string;
	info?: string;
}
