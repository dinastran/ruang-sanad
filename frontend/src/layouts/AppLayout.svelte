<script lang="ts">
	import type { Snippet } from "svelte";
	import { fly, fade } from "svelte/transition";
	import { inertia, router } from "@inertiajs/svelte";
	import {
		LayoutDashboard,
		Users,
		BookOpen,
		Wallet,
		Upload,
		Settings,
		Shield,
		LogOut,
		Menu,
		X,
		User,
		AlertCircle,
		FileSpreadsheet,
		Database,
		Receipt,
		PanelLeftClose,
		PanelLeft,
	} from "lucide-svelte";
	import DarkModeToggle from "@components/DarkModeToggle.svelte";
	import Logo from "@components/Logo.svelte";

	interface Props {
		user?: { id: number; name: string; email: string; avatar: string; role: string };
		children: Snippet;
	}

	let { user, children }: Props = $props();

	let sidebarCollapsed = $state(false);
	let isMenuOpen = $state(false);
	let isDesktopUserMenuOpen = $state(false);
	let isUserMenuOpen = $state(false);

	let role = $derived(user?.role || "");
	let isSuperAdmin = $derived(role === "super_admin");
	let isCS = $derived(role === "cs" || isSuperAdmin);
	let isAdminKelas = $derived(role === "admin_kelas" || isSuperAdmin);
	let isKeuangan = $derived(role === "keuangan" || isSuperAdmin);

	let sidebarW = $derived(sidebarCollapsed ? "w-16" : "w-72");
	let headerLeft = $derived(sidebarCollapsed ? "left-16" : "left-72");
	let showLabel = $derived(!sidebarCollapsed);

	let menuLinks = $derived([
		{ href: "/app", label: "Dashboard", group: "dashboard", show: true, icon: LayoutDashboard },
		{ href: "/app/santri", label: "Data Santri", group: "santri", show: isCS || isAdminKelas, icon: Users },
		{ href: "/app/perlu-dilengkapi", label: "Perlu Dilengkapi", group: "perlu-dilengkapi", show: isAdminKelas, icon: AlertCircle },
		{ href: "/app/kelas", label: "Kelas", group: "kelas", show: isAdminKelas, icon: BookOpen },
		{ href: "/app/keuangan", label: "Keuangan", group: "keuangan", show: isKeuangan, icon: Wallet },
		{ href: "/app/laporan/keuangan", label: "Laporan Keuangan", group: "laporan-keuangan", show: isKeuangan, icon: Receipt },
		{ href: "/app/master", label: "Data Master", group: "master", show: isAdminKelas, icon: Database },
		{ href: "/admin/import", label: "Import CSV", group: "import", show: isSuperAdmin, icon: FileSpreadsheet },
		{ href: "/admin/users", label: "Kelola User", group: "users", show: isSuperAdmin, icon: Shield },
		{ href: "/app/profile", label: "Profil", group: "profile", show: true, icon: Settings },
	].filter((item) => item.show));

	let activeGroup = $derived(activeGroupFromPath());
	function activeGroupFromPath(): string {
		if (typeof window === "undefined") return "";
		const path = window.location.pathname;
		if (path === "/app" || path === "/app/") return "dashboard";
		if (path.startsWith("/app/santri")) return "santri";
		if (path.startsWith("/app/perlu-dilengkapi")) return "perlu-dilengkapi";
		if (path.startsWith("/app/kelas")) return "kelas";
		if (path.startsWith("/app/laporan/keuangan")) return "laporan-keuangan";
		if (path.startsWith("/app/keuangan")) return "keuangan";
		if (path.startsWith("/app/master")) return "master";
		if (path.startsWith("/admin/import")) return "import";
		if (path.startsWith("/admin/users")) return "users";
		if (path.startsWith("/app/profile")) return "profile";
		return "";
	}

	$effect(() => {
		if (typeof localStorage === "undefined") return;
		const saved = localStorage.getItem("sidebarCollapsed");
		if (saved === "true") sidebarCollapsed = true;
	});

	function toggleSidebar() {
		sidebarCollapsed = !sidebarCollapsed;
		localStorage.setItem("sidebarCollapsed", String(sidebarCollapsed));
	}

	let desktopMenuEl = $state<HTMLDivElement>();

	function handleLogout() {
		router.post("/logout");
	}

	$effect(() => {
		if (!isDesktopUserMenuOpen || typeof document === "undefined") return;
		const timer = setTimeout(() => {
			document.addEventListener("click", onDocumentClick);
		}, 0);
		return () => {
			clearTimeout(timer);
			document.removeEventListener("click", onDocumentClick);
		};
	});

	function onDocumentClick(e: MouseEvent) {
		if (desktopMenuEl && !desktopMenuEl.contains(e.target as Node)) {
			isDesktopUserMenuOpen = false;
		}
	}

	$effect(() => {
		if (typeof document !== "undefined") {
			document.body.style.overflow = isMenuOpen ? "hidden" : "unset";
		}
	});
</script>

<header
	class="hidden lg:flex fixed top-0 right-0 h-16 z-40 bg-white/95 dark:bg-neutral-950/95 backdrop-blur-xl border-b border-neutral-200/80 dark:border-white/[0.04] items-center justify-between px-4 gap-3 transition-all duration-300"
	style="left: {sidebarCollapsed ? '4rem' : '18rem'}"
>
	<button onclick={toggleSidebar}
		class="p-2 rounded-lg hover:bg-neutral-100 dark:hover:bg-neutral-800 text-neutral-500 dark:text-neutral-400 transition-colors"
		aria-label={sidebarCollapsed ? "Buka sidebar" : "Tutup sidebar"}
	>
		{#if sidebarCollapsed}
			<PanelLeft class="w-5 h-5" />
		{:else}
			<PanelLeftClose class="w-5 h-5" />
		{/if}
	</button>

	<div class="flex items-center gap-3 ml-auto">
		<DarkModeToggle />
		{#if user && user.id}
			<div bind:this={desktopMenuEl} class="relative" role="menu">
				<button
					onclick={() => (isDesktopUserMenuOpen = !isDesktopUserMenuOpen)}
					class="flex items-center gap-2.5 px-3 py-1.5 rounded-lg hover:bg-neutral-100 dark:hover:bg-neutral-800 transition-colors"
				>
					{#if user.avatar}
						<img src={user.avatar} alt={user.name} class="w-8 h-8 rounded-full object-cover ring-2 ring-neutral-300 dark:ring-neutral-700 shrink-0" />
					{:else}
						<div class="w-8 h-8 rounded-full bg-brand-600 dark:bg-brand-500 flex items-center justify-center text-white font-bold text-sm ring-2 ring-neutral-300 dark:ring-neutral-700 shrink-0">
							{user.name.charAt(0).toUpperCase()}
						</div>
					{/if}
					<span class="text-sm font-semibold text-neutral-900 dark:text-white">{user.name}</span>
				</button>
				{#if isDesktopUserMenuOpen}
					<div
						class="absolute right-0 mt-2 w-56 bg-white dark:bg-neutral-925 rounded-xl shadow-xl border border-neutral-200/80 dark:border-white/[0.06] overflow-hidden ring-1 ring-black/10 dark:ring-white/10"
						transition:fly={{ y: 10, duration: 200 }}
					>
						<div class="px-4 py-3 border-b border-neutral-200/80 dark:border-white/[0.04]">
							<p class="text-xs font-medium text-neutral-500 dark:text-neutral-400 uppercase tracking-wider">Masuk sebagai</p>
							<p class="text-sm font-semibold text-neutral-900 dark:text-white mt-0.5">{user.name}</p>
							<p class="text-xs text-neutral-500 dark:text-neutral-400 truncate">{user.email}</p>
							<p class="text-xs text-neutral-500 dark:text-neutral-400 mt-1 capitalize">{user.role?.replace('_', ' ')}</p>
						</div>
						<div class="p-2">
							<a href="/app/profile" use:inertia onclick={() => (isDesktopUserMenuOpen = false)}
								class="flex items-center gap-2 px-3 py-2 rounded-lg text-sm text-neutral-700 dark:text-neutral-300 hover:bg-neutral-100 dark:hover:bg-neutral-800 transition-colors"
							>
								<User size="16" />
								Profil
							</a>
						</div>
						<div class="p-2 border-t border-neutral-200/80 dark:border-white/[0.04]">
							<button onclick={() => { isDesktopUserMenuOpen = false; handleLogout(); }}
								class="w-full flex items-center gap-2 px-3 py-2 rounded-lg text-sm text-red-500 dark:text-red-400 hover:bg-red-500/10 transition-colors"
							>
								<LogOut size="16" />
								Keluar
							</button>
						</div>
					</div>
				{/if}
			</div>
		{/if}
	</div>
</header>

<aside
	class="hidden lg:flex flex-col fixed left-0 top-0 h-full bg-white/95 dark:bg-neutral-950/95 backdrop-blur-xl border-r border-neutral-200/80 dark:border-white/[0.04] z-30 transition-all duration-300"
	style="width: {sidebarCollapsed ? '4rem' : '18rem'}"
>
	<a href="/app" use:inertia class="flex items-center gap-3 px-4 py-6 hover:opacity-80 transition-opacity no-underline min-h-[88px]">
		<Logo size={36} />
		{#if showLabel}
			<div class="overflow-hidden">
				<h1 class="text-xl font-black italic text-neutral-900 dark:text-white whitespace-nowrap">
					Ruang<span class="text-brand-400">Sanad</span>
				</h1>
				<p class="text-xs text-neutral-500 dark:text-neutral-400 truncate">Dashboard {user?.role?.replace('_', ' ') || ''}</p>
			</div>
		{/if}
	</a>

	<nav class="flex-1 px-2 py-4 space-y-1 overflow-y-auto">
		{#each menuLinks as item}
			{@const Icon = item.icon}
			<a
				href={item.href}
				use:inertia
				class="flex items-center gap-3 px-3 py-2.5 rounded-xl text-sm font-medium transition-all duration-200 group {item.group ===
				activeGroup
					? 'bg-brand-400/10 text-brand-600 dark:text-brand-400 border border-brand-400/20'
					: 'text-neutral-600 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-white hover:bg-neutral-100 dark:hover:bg-neutral-800/50 border border-transparent'}"
				title={!showLabel ? item.label : ''}
			>
				<Icon
					size="20"
					class={item.group === activeGroup
						? 'text-brand-400 shrink-0'
						: 'text-neutral-500 dark:text-neutral-400 group-hover:text-neutral-900 dark:group-hover:text-white shrink-0'}
				/>
				{#if showLabel}
					<span class="truncate">{item.label}</span>
					{#if item.group === activeGroup}
						<div class="ml-auto w-1.5 h-1.5 rounded-full bg-brand-400 shrink-0"></div>
					{/if}
				{/if}
			</a>
		{/each}
	</nav>

	{#if user && user.id}
		<div class="p-2 border-t border-neutral-200/80 dark:border-white/[0.04]">
			<button onclick={handleLogout}
				class="w-full flex items-center justify-center gap-1.5 px-2 py-2 rounded-lg text-[11px] font-medium text-red-500 dark:text-red-400 hover:bg-red-500/10 transition-colors"
				title="Keluar"
			>
				<LogOut size="14" class="shrink-0" />
				{#if showLabel}<span>Keluar</span>{/if}
			</button>
		</div>
	{/if}
</aside>

<header class="lg:hidden fixed top-0 left-0 right-0 z-50 bg-white dark:bg-neutral-950 backdrop-blur-xl border-b border-neutral-200/80 dark:border-white/[0.04]">
	<div class="flex items-center justify-between px-4 h-16">
		<a href="/app" use:inertia class="flex items-center gap-2">
			<Logo size={28} />
			<span class="text-lg font-black italic text-neutral-900 dark:text-white">
				Ruang<span class="text-brand-400">Sanad</span>
			</span>
		</a>
		<div class="flex items-center gap-2">
			{#if user && user.id}
				<div class="relative" role="menu">
					<button onclick={() => (isUserMenuOpen = !isUserMenuOpen)}
						class="w-9 h-9 rounded-full ring-2 ring-neutral-300 dark:ring-neutral-700 overflow-hidden"
					>
						{#if user.avatar}
							<img src={user.avatar} alt={user.name} class="w-full h-full object-cover" />
						{:else}
							<div class="w-full h-full bg-brand-600 dark:bg-brand-500 flex items-center justify-center text-white font-bold text-sm">
								{user.name.charAt(0).toUpperCase()}
							</div>
						{/if}
					</button>
					{#if isUserMenuOpen}
						<div class="fixed inset-0 z-10" role="presentation" onclick={() => (isUserMenuOpen = false)}></div>
						<div class="absolute right-0 mt-2 w-48 bg-white dark:bg-neutral-925 rounded-xl shadow-xl border border-neutral-200/80 dark:border-white/[0.06] overflow-hidden"
							transition:fly={{ y: 10, duration: 200 }}
						>
							<div class="p-3 border-b border-neutral-200/80 dark:border-white/[0.04]">
								<p class="text-xs font-medium text-neutral-500 dark:text-neutral-400 uppercase">Masuk sebagai</p>
								<p class="text-sm font-semibold text-neutral-900 dark:text-white truncate">{user.name}</p>
							</div>
							<div class="p-2">
								<a href="/app/profile" use:inertia
									class="flex items-center gap-2 px-3 py-2 rounded-lg text-sm text-neutral-700 dark:text-neutral-300 hover:bg-neutral-100 dark:hover:bg-neutral-800 transition-colors"
								><User size="16" /> Profil</a>
							</div>
							<div class="p-2 border-t border-neutral-200/80 dark:border-white/[0.04]">
								<button onclick={handleLogout}
									class="w-full flex items-center gap-2 px-3 py-2 rounded-lg text-sm text-red-500 dark:text-red-400 hover:bg-red-500/10 transition-colors"
								><LogOut size="16" /> Keluar</button>
							</div>
						</div>
					{/if}
				</div>
			{/if}
			<button onclick={() => (isMenuOpen = !isMenuOpen)}
				class="p-2 rounded-lg bg-neutral-200/80 dark:bg-neutral-800 text-neutral-600 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-white transition-colors"
				aria-label="Menu"
			>
				{#if isMenuOpen}<X size="20" />{:else}<Menu size="20" />{/if}
			</button>
		</div>
	</div>
</header>

{#if isMenuOpen}
	<div class="lg:hidden fixed inset-0 z-50">
		<button class="absolute inset-0 w-full h-full bg-neutral-900/50 backdrop-blur-sm"
			transition:fade={{ duration: 200 }} onclick={() => (isMenuOpen = false)}
		></button>
		<div class="absolute right-0 top-0 h-full w-[85%] max-w-[320px] bg-white dark:bg-neutral-925 shadow-2xl border-l border-neutral-200/80 dark:border-white/[0.04] flex flex-col"
			transition:fly={{ x: 300, duration: 400, opacity: 1 }}
		>
			<div class="flex items-center justify-between p-4 border-b border-neutral-200/80 dark:border-white/[0.04]">
				<span class="text-base font-bold text-neutral-900 dark:text-white">Menu</span>
				<button onclick={() => (isMenuOpen = false)}
					class="p-2 rounded-lg hover:bg-neutral-200/80 dark:hover:bg-neutral-800 text-neutral-600 dark:text-neutral-400 transition-colors"
				><X size="20" /></button>
			</div>
			<div class="flex-1 overflow-y-auto p-4 space-y-2">
				{#each menuLinks as item}
					{@const Icon = item.icon}
					<a href={item.href} use:inertia
						class="flex items-center gap-3 px-4 py-3 rounded-xl text-sm font-medium transition-all {item.group ===
						activeGroup
							? 'bg-brand-400/10 text-brand-600 dark:text-brand-400 border border-brand-400/20'
							: 'text-neutral-700 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-white hover:bg-neutral-100 dark:hover:bg-neutral-800/50 border border-transparent'}"
						onclick={() => (isMenuOpen = false)}
					>
						<Icon size="20" />
						{item.label}
					</a>
				{/each}
			</div>
			<div class="p-4 border-t border-neutral-200/80 dark:border-white/[0.04]">
				<DarkModeToggle />
			</div>
		</div>
	</div>
{/if}

<div class="hidden lg:block h-16"></div>
<div class="lg:hidden h-16"></div>

<div class="min-h-screen bg-neutral-50 dark:bg-neutral-950 transition-all duration-300 {sidebarCollapsed ? 'lg:ml-16' : 'lg:ml-72'}">
	{@render children()}
</div>
