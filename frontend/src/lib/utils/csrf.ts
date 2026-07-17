/**
 * CSRF utilities for manual fetch() calls.
 *
 * Inertia's router (Axios) automatically reads the XSRF-TOKEN cookie
 * and sends it as the X-XSRF-TOKEN header. Only plain fetch() calls
 * to CSRF-protected routes (/app/*, /admin/*) need this explicitly.
 *
 * @example
 *   fetch("/app/upload", {
 *     method: "POST",
 *     headers: { "X-XSRF-TOKEN": getCSRFToken() },
 *     body: formData,
 *   });
 */

/**
 * Read the CSRF token from the XSRF-TOKEN cookie.
 * Returns empty string if cookie is missing.
 */
export function getCSRFToken(): string {
	const match = document.cookie.match(/(?:^|;\s*)XSRF-TOKEN=([^;]*)/);
	return match ? decodeURIComponent(match[1]) : "";
}
