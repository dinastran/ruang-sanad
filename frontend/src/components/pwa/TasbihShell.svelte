<script lang="ts">
	// Layout "aplikasi" untuk Tasbih (role user / mode terpasang): header ringkas,
	// navigasi bawah, safe-area iPhone, tanpa sidebar dashboard.
	import type { Snippet } from "svelte";
	import { fly } from "svelte/transition";
	import { inertia } from "@inertiajs/svelte";
	import { BarChart3, CircleDot, History, LogOut, RefreshCw, UserRound } from "lucide-svelte";
	import DarkModeToggle from "@components/DarkModeToggle.svelte";
	import { muatVersiBaru, pwa } from "@lib/pwa.svelte";

	interface Props {
		user?: { id: number; name: string; email: string; avatar: string; role: string };
		onlogout: () => void;
		children: Snippet;
	}
	let { user, onlogout, children }: Props = $props();

	const path = typeof window !== "undefined" ? window.location.pathname : "";
	const NAV = [
		{ href: "/app/tasbih", label: "Tasbih", icon: CircleDot, aktif: path === "/app/tasbih" || path === "/app/tasbih/" },
		{ href: "/app/tasbih/dashboard", label: "Progres", icon: BarChart3, aktif: path.startsWith("/app/tasbih/dashboard") },
		{ href: "/app/tasbih/riwayat", label: "Riwayat", icon: History, aktif: path.startsWith("/app/tasbih/riwayat") },
	];
	// Profil dipindah ke menu akun; tab bawah khusus alur tasbih.
	const judul = path.startsWith("/app/profile") ? "Profil" : (NAV.find((n) => n.aktif)?.label ?? "Tasbih");
	let menuAkun = $state(false);

	// Mode aplikasi: matikan efek tarik-untuk-refresh/bounce (side effect DOM).
	$effect(() => {
		document.documentElement.classList.add("app-mode");
		return () => document.documentElement.classList.remove("app-mode");
	});
</script>

<div class="min-h-[100dvh] bg-neutral-50 dark:bg-neutral-950" data-testid="app-shell">
	<header class="fixed inset-x-0 top-0 z-30 border-b border-neutral-200/70 bg-white/90 pt-[env(safe-area-inset-top)] backdrop-blur-xl dark:border-white/[0.06] dark:bg-neutral-950/90">
		<div class="mx-auto flex h-14 max-w-xl items-center gap-3 px-4">
			<img src="/public/icons/tasbih-192.png" alt="" class="h-8 w-8 rounded-[10px]" width="32" height="32" />
			<h1 class="flex-1 truncate text-base font-semibold tracking-tight text-neutral-900 dark:text-white">{judul}</h1>
			<DarkModeToggle />
			<div class="relative">
				<button
					type="button"
					onclick={() => (menuAkun = !menuAkun)}
					class="grid h-11 w-11 place-items-center rounded-full"
					aria-label="Menu akun"
					aria-expanded={menuAkun}
				>
					{#if user?.avatar}
						<img src={user.avatar} alt="" class="h-8 w-8 rounded-full object-cover" />
					{:else}
						<span class="grid h-8 w-8 place-items-center rounded-full bg-brand-600 text-sm font-semibold text-white dark:bg-brand-500 dark:text-neutral-950">{user?.name?.charAt(0).toUpperCase() ?? "?"}</span>
					{/if}
				</button>
				{#if menuAkun}
					<button type="button" class="fixed inset-0 z-10 cursor-default" aria-label="Tutup menu" onclick={() => (menuAkun = false)}></button>
					<div class="absolute right-0 top-12 z-20 w-60 rounded-2xl border border-neutral-200 bg-white p-2 shadow-xl dark:border-neutral-800 dark:bg-neutral-900" transition:fly={{ y: -6, duration: 150 }}>
						<div class="px-3 py-2">
							<p class="truncate text-sm font-semibold text-neutral-900 dark:text-white">{user?.name}</p>
							<p class="truncate text-xs text-neutral-500">{user?.email}</p>
						</div>
						<a href="/app/profile" use:inertia onclick={() => (menuAkun = false)} class="flex min-h-11 w-full items-center gap-2 rounded-xl px-3 text-sm font-medium text-neutral-700 hover:bg-neutral-100 dark:text-neutral-200 dark:hover:bg-neutral-800">
							<UserRound size={16} aria-hidden="true" /> Profil
						</a>
						<button type="button" onclick={() => { menuAkun = false; onlogout(); }} class="flex min-h-11 w-full items-center gap-2 rounded-xl px-3 text-sm font-medium text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-950/40">
							<LogOut size={16} aria-hidden="true" /> Keluar
						</button>
					</div>
				{/if}
			</div>
		</div>
	</header>

	<main class="mx-auto max-w-xl px-4 pb-[calc(5.5rem+env(safe-area-inset-bottom))] pt-[calc(4.5rem+env(safe-area-inset-top))]">
		{@render children()}
	</main>

	{#if pwa.updateSiap}
		<div class="fixed inset-x-0 bottom-[calc(4.75rem+env(safe-area-inset-bottom))] z-30 flex justify-center px-4" transition:fly={{ y: 16, duration: 200 }} role="status">
			<div class="flex w-full max-w-sm items-center gap-3 rounded-2xl bg-neutral-900 px-4 py-3 text-sm text-white shadow-xl dark:bg-white dark:text-neutral-900">
				<span class="flex-1">Versi baru tersedia</span>
				<button type="button" onclick={muatVersiBaru} class="inline-flex min-h-9 items-center gap-1.5 rounded-lg bg-brand-500 px-3 font-semibold text-neutral-950">
					<RefreshCw size={14} aria-hidden="true" /> Muat ulang
				</button>
			</div>
		</div>
	{/if}

	<nav class="fixed inset-x-0 bottom-0 z-30 border-t border-neutral-200/70 bg-white/95 pb-[env(safe-area-inset-bottom)] backdrop-blur-xl dark:border-white/[0.06] dark:bg-neutral-950/95" aria-label="Navigasi utama">
		<ul class="mx-auto grid h-16 max-w-xl grid-cols-3">
			{#each NAV as n (n.href)}
				<li>
					<a
						href={n.href}
						use:inertia
						aria-current={n.aktif ? "page" : undefined}
						class="flex h-full flex-col items-center justify-center gap-1 text-[11px] font-medium transition-colors {n.aktif ? 'text-brand-700 dark:text-brand-300' : 'text-neutral-500 dark:text-brand-100/45'}"
					>
						<span class="grid h-7 w-14 place-items-center rounded-full transition-colors {n.aktif ? 'bg-brand-100 dark:bg-brand-300/10' : ''}">
							<n.icon size={20} aria-hidden="true" />
						</span>
						{n.label}
					</a>
				</li>
			{/each}
		</ul>
	</nav>
</div>
