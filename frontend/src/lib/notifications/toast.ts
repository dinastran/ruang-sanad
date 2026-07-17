/**
 * Toast notification helper.
 *
 * Imperative DOM-based API — call Toast("message", "success") from anywhere.
 * Creates a fixed container on first call and reuses it.
 *
 * For Svelte-idiomatic usage, prefer subscribing to a toast store from a
 * `<ToastContainer />` component. Kept imperative here for simplicity in
 * handlers and non-component modules.
 */

export type ToastType = "success" | "error" | "warning" | "info";

const ICONS: Record<ToastType, string> = {
	success: `<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"></path><polyline points="22 4 12 14.01 9 11.01"></polyline></svg>`,
	error: `<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"></circle><line x1="15" y1="9" x2="9" y2="15"></line><line x1="9" y1="9" x2="15" y2="15"></line></svg>`,
	warning: `<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"></path><line x1="12" y1="9" x2="12" y2="13"></line><line x1="12" y1="17" x2="12.01" y2="17"></line></svg>`,
	info: `<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"></circle><line x1="12" y1="16" x2="12" y2="12"></line><line x1="12" y1="8" x2="12.01" y2="8"></line></svg>`,
};

const BG_COLORS: Record<ToastType, string> = {
	success: "rgba(34, 197, 94, 0.95)",
	error: "rgba(239, 68, 68, 0.95)",
	warning: "rgba(245, 158, 11, 0.95)",
	info: "rgba(59, 130, 246, 0.95)",
};

/**
 * Display a toast notification.
 * Creates a floating notification that auto-dismisses after `duration` ms.
 */
export function Toast(
	text: string,
	type: ToastType = "success",
	duration: number = 3000,
): void {
	let container = document.getElementById("toast-container");
	if (!container) {
		container = document.createElement("div");
		container.id = "toast-container";
		container.style.cssText = `
            position: fixed;
            bottom: 24px;
            left: 50%;
            transform: translateX(-50%);
            z-index: 9999;
            display: flex;
            flex-direction: column;
            align-items: center;
            gap: 8px;
        `;
		document.body.appendChild(container);
	}

	const toast = document.createElement("div");
	toast.style.cssText = `
        min-width: 300px;
        max-width: 90vw;
        margin: 0;
        padding: 12px 16px;
        border-radius: 8px;
        color: white;
        font-size: 14px;
        font-weight: 500;
        opacity: 0;
        transform: translateY(20px) scale(0.95);
        transition: all 0.2s cubic-bezier(0.68, -0.55, 0.265, 1.55);
        box-shadow: 0 4px 12px rgba(0,0,0,0.15), 0 0 1px rgba(0,0,0,0.1);
        display: flex;
        align-items: center;
        gap: 12px;
        backdrop-filter: blur(8px);
        background: ${BG_COLORS[type]};
    `;

	const iconWrapper = document.createElement("div");
	iconWrapper.style.cssText = `
        display: flex;
        align-items: center;
        justify-content: center;
        flex-shrink: 0;
    `;
	const parser = new DOMParser();
	const svg = parser.parseFromString(
		ICONS[type],
		"image/svg+xml",
	).documentElement;
	iconWrapper.appendChild(svg);

	const textWrapper = document.createElement("div");
	textWrapper.style.cssText = `
        flex-grow: 1;
        line-height: 1.4;
    `;
	textWrapper.textContent = text;

	toast.appendChild(iconWrapper);
	toast.appendChild(textWrapper);
	container.appendChild(toast);

	requestAnimationFrame(() => {
		toast.style.opacity = "1";
		toast.style.transform = "translateY(0) scale(1)";
	});

	setTimeout(() => {
		toast.style.opacity = "0";
		toast.style.transform = "translateY(-20px) scale(0.95)";
		setTimeout(() => {
			if (container && toast.parentNode === container) {
				container.removeChild(toast);
				if (container.children.length === 0) {
					document.body.removeChild(container);
				}
			}
		}, 200);
	}, duration);
}
