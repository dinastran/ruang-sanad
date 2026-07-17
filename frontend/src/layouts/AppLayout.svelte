<script lang="ts">
	import type { Snippet } from "svelte";
	import { fly, fade } from "svelte/transition";
	import { inertia, router } from "@inertiajs/svelte";
	import {
		LayoutDashboard,
		Settings,
		LogOut,
		Menu,
		X,
		User,
		Upload,
	} from "lucide-svelte";
	import DarkModeToggle from "@components/DarkModeToggle.svelte";
	import Logo from "@components/Logo.svelte";
	import type { User as UserType } from "@lib/types";

	/**
	 * AppLayout wraps every authenticated page with:
	 * - Desktop sidebar (logo, nav, user card)
	 * - Desktop top header with breadcrumbs & user menu
	 * - Mobile top bar + drawer
	 * - Logout + dark-mode toggles
	 *
	 * Pages render their content as children:
	 *   <AppLayout user={user} group="dashboard">
	 *     <h1>Page content...</h1>
	 *   </AppLayout>
	 */
	interface Props {
		user?: UserType;
		/** Active nav group: "dashboard" | "profile" | "" */
		group?: string;
		children: Snippet;
	}

	let { user, group = "", children }: Props = $props();

	let isMenuOpen = $state(false);
	let isDesktopUserMenuOpen = $state(false);
	let isUserMenuOpen = $state(false);



	const menuLinks = [
		{
			href: "/app",
			label: "Dashboard",
			group: "dashboard",
			show: true,
			icon: LayoutDashboard,
		},
		{
			href: "/app/upload",
			label: "Upload Test",
			group: "upload",
			show: true,
			icon: Upload,
		},
		{
			href: "/app/profile",
			label: "Settings",
			group: "profile",
			show: true,
			icon: Settings,
		},
	];

	let desktopMenuEl = $state<HTMLDivElement>();

	function handleLogout() {
		router.post("/logout");
	}

	// Close desktop dropdown on click outside
	$effect(() => {
		if (!isDesktopUserMenuOpen || typeof document === "undefined") return;

		function onDocumentClick(e: MouseEvent) {
			if (desktopMenuEl && !desktopMenuEl.contains(e.target as Node)) {
				isDesktopUserMenuOpen = false;
			}
		}

		// Use setTimeout to avoid the same click that opened it from closing it
		const timer = setTimeout(() => {
			document.addEventListener("click", onDocumentClick);
		}, 0);

		return () => {
			clearTimeout(timer);
			document.removeEventListener("click", onDocumentClick);
		};
	});

	// Prevent body scroll when mobile menu is open
	$effect(() => {
		if (typeof document !== "undefined") {
			document.body.style.overflow = isMenuOpen ? "hidden" : "unset";
		}
	});
</script>

<!-- Desktop Header -->
<header
	class="hidden lg:flex fixed top-0 left-72 right-0 h-16 z-40 bg-white/95 dark:bg-neutral-950/95 backdrop-blur-xl border-b border-neutral-200/80 dark:border-white/[0.04] items-center justify-end px-6 gap-3"
>
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
					<div
						class="w-8 h-8 rounded-full bg-brand-600 dark:bg-brand-500 flex items-center justify-center text-white font-bold text-sm ring-2 ring-neutral-300 dark:ring-neutral-700 shrink-0"
					>
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
						<p class="text-xs font-medium text-neutral-500 dark:text-neutral-400 uppercase tracking-wider">Signed in as</p>
						<p class="text-sm font-semibold text-neutral-900 dark:text-white mt-0.5">{user.name}</p>
						<p class="text-xs text-neutral-500 dark:text-neutral-400 truncate">{user.email}</p>
					</div>
					<div class="p-2">
						<a
							href="/app/profile"
							use:inertia
							onclick={() => (isDesktopUserMenuOpen = false)}
							class="flex items-center gap-2 px-3 py-2 rounded-lg text-sm text-neutral-700 dark:text-neutral-300 hover:bg-neutral-100 dark:hover:bg-neutral-800 hover:text-neutral-900 dark:hover:text-white transition-colors"
						>
							<User size="16" />
							Profile
						</a>
					</div>
					<div class="p-2 border-t border-neutral-200/80 dark:border-white/[0.04]">
						<button
							onclick={() => { isDesktopUserMenuOpen = false; handleLogout(); }}
							class="w-full flex items-center gap-2 px-3 py-2 rounded-lg text-sm text-red-500 dark:text-red-400 hover:bg-red-500/10 transition-colors"
						>
							<LogOut size="16" />
							Logout
						</button>
					</div>
				</div>
			{/if}
		</div>
	{/if}
</header>

<!-- Desktop Sidebar -->
<aside
	class="hidden lg:flex flex-col fixed left-0 top-0 h-full w-72 bg-white/95 dark:bg-neutral-950/95 backdrop-blur-xl border-r border-neutral-200/80 dark:border-white/[0.04] z-30 transition-all duration-300"
>
	<!-- Logo -->
	<a
		href="/app"
		use:inertia
		class="flex items-center gap-3 px-6 py-6 hover:opacity-80 transition-opacity no-underline"
	>
		<Logo size={36} />
		<div>
			<h1 class="text-xl font-black italic text-neutral-900 dark:text-white">
				Laju<span class="text-brand-400">Go</span>
			</h1>
			{#if group === "dashboard"}
				<p class="text-xs text-neutral-500 dark:text-neutral-400">Dashboard</p>
			{:else if group === "profile"}
				<p class="text-xs text-neutral-500 dark:text-neutral-400">Settings</p>
			{:else}
				<p class="text-xs text-neutral-500 dark:text-neutral-400">App</p>
			{/if}
		</div>
	</a>

	<!-- Navigation -->
	<nav class="flex-1 px-4 py-6 space-y-2">
		{#each menuLinks.filter((item) => item.show) as item}
			{@const Icon = item.icon}
			<a
				href={item.href}
				use:inertia
				class="flex items-center gap-3 px-4 py-3 rounded-xl text-sm font-medium transition-all duration-200 group {item.group ===
				group
					? 'bg-brand-400/10 text-brand-600 dark:text-brand-400 border border-brand-400/20'
					: 'text-neutral-600 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-white hover:bg-neutral-100 dark:hover:bg-neutral-800/50 border border-transparent'}"
			>
				<Icon
					size="20"
					class={item.group === group
						? 'text-brand-400'
						: 'text-neutral-500 dark:text-neutral-400 group-hover:text-neutral-900 dark:group-hover:text-white'}
					stroke-width="2"
				/>
				{item.label}
				{#if item.group === group}
					<div class="ml-auto w-1.5 h-1.5 rounded-full bg-brand-400"></div>
				{/if}
			</a>
		{/each}
	</nav>

	{#if user && user.id}
		<div class="p-3 border-t border-neutral-200/80 dark:border-white/[0.04] space-y-2">
			<div class="flex items-center gap-2.5">
				{#if user.avatar}
					<img src={user.avatar} alt={user.name} class="w-7 h-7 rounded-full object-cover ring-2 ring-neutral-300 dark:ring-neutral-700 shrink-0" />
				{:else}
					<div
						class="w-7 h-7 rounded-full bg-brand-600 dark:bg-brand-500 flex items-center justify-center text-white font-bold text-xs ring-2 ring-neutral-300 dark:ring-neutral-700 shrink-0"
					>
						{user.name.charAt(0).toUpperCase()}
					</div>
				{/if}
				<div class="flex-1 min-w-0">
					<p class="text-xs font-semibold text-neutral-900 dark:text-white truncate leading-normal">
						{user.name}
					</p>
					<p class="text-[11px] text-neutral-500 dark:text-neutral-400 truncate leading-normal">
						{user.email || "Member"}
					</p>
				</div>
			</div>

			<button
				onclick={handleLogout}
				class="w-full flex items-center justify-center gap-1.5 px-2 py-1.5 rounded-lg text-[11px] font-medium text-red-500 dark:text-red-400 hover:bg-red-500/10 transition-colors"
			>
				<LogOut size="14" />
				Logout
			</button>
		</div>
	{/if}
</aside>

<!-- Mobile Header -->
<header
	class="lg:hidden fixed top-0 left-0 right-0 z-50 bg-white dark:bg-neutral-950 backdrop-blur-xl border-b border-neutral-200/80 dark:border-white/[0.04]"
>
	<div class="flex items-center justify-between px-4 h-16">
		<a href="/app" use:inertia class="flex items-center gap-2">
			<Logo size={28} />
			<span class="text-lg font-black italic text-neutral-900 dark:text-white">
				Laju<span class="text-brand-400">Go</span>
			</span>
		</a>

		<div class="flex items-center gap-2">
			{#if user && user.id}
				<div class="relative" role="menu">
					<button
						onclick={() => (isUserMenuOpen = !isUserMenuOpen)}
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
						<div
							class="fixed inset-0 z-10"
							role="presentation"
							onclick={() => (isUserMenuOpen = false)}
						></div>
						<div
							class="absolute right-0 mt-2 w-48 bg-white dark:bg-neutral-925 rounded-xl shadow-xl border border-neutral-200/80 dark:border-white/[0.06] overflow-hidden ring-1 ring-black/10 dark:ring-white/10"
							transition:fly={{ y: 10, duration: 200 }}
						>
							<div
								class="p-3 border-b border-neutral-200/80 dark:border-white/[0.04]"
							>
								<p
									class="text-xs font-medium text-neutral-500 dark:text-neutral-400 uppercase"
								>
									Signed in as
								</p>
								<p
									class="text-sm font-semibold text-neutral-900 dark:text-white truncate"
								>
									{user.name}
								</p>
							</div>
							<div class="p-2">
								<a
									href="/app/profile"
									use:inertia
									class="flex items-center gap-2 px-3 py-2 rounded-lg text-sm text-neutral-700 dark:text-neutral-300 hover:bg-neutral-100 dark:hover:bg-neutral-800 hover:text-neutral-900 dark:hover:text-white transition-colors"
								>
									<User size="16" />
									Profile
								</a>
							</div>
							<div
								class="p-2 border-t border-neutral-200/80 dark:border-white/[0.04]"
							>
								<button
									onclick={handleLogout}
									class="w-full flex items-center gap-2 px-3 py-2 rounded-lg text-sm text-red-500 dark:text-red-400 hover:bg-red-500/10 transition-colors"
								>
									<LogOut size="16" />
									Logout
								</button>
							</div>
						</div>
					{/if}
				</div>
			{:else}
				<a
					href="/login"
					use:inertia
					class="px-4 py-2 rounded-lg bg-neutral-200/80 dark:bg-neutral-800 hover:bg-neutral-300/80 dark:hover:bg-neutral-700 text-neutral-700 dark:text-neutral-300 text-sm font-medium transition-colors"
				>
					Sign In
				</a>
			{/if}

			<button
				onclick={() => (isMenuOpen = !isMenuOpen)}
				class="p-2 rounded-lg bg-neutral-200/80 dark:bg-neutral-800 text-neutral-600 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-white transition-colors"
				aria-label="Menu"
			>
				{#if isMenuOpen}
					<X size="20" />
				{:else}
					<Menu size="20" />
				{/if}
			</button>
		</div>
	</div>
</header>

<!-- Mobile Menu Drawer -->
{#if isMenuOpen}
	<div class="lg:hidden fixed inset-0 z-50">
		<button
			class="absolute inset-0 w-full h-full bg-neutral-900/50 backdrop-blur-sm"
			transition:fade={{ duration: 200 }}
			onclick={() => (isMenuOpen = false)}
			aria-label="Close menu"
		></button>

		<div
			class="absolute right-0 top-0 h-full w-[85%] max-w-[320px] bg-white dark:bg-neutral-925 shadow-2xl border-l border-neutral-200/80 dark:border-white/[0.04] flex flex-col"
			transition:fly={{ x: 300, duration: 400, opacity: 1 }}
		>
			<!-- Header -->
			<div
				class="flex items-center justify-between p-4 border-b border-neutral-200/80 dark:border-white/[0.04] bg-neutral-50 dark:bg-neutral-925/50"
			>
				<span class="text-base font-bold text-neutral-900 dark:text-white">Menu</span>
				<button
					onclick={() => (isMenuOpen = false)}
					class="p-2 rounded-lg hover:bg-neutral-200/80 dark:hover:bg-neutral-800 text-neutral-600 dark:text-neutral-400 transition-colors"
				>
					<X size="20" />
				</button>
			</div>

			<!-- Navigation -->
			<div
				class="flex-1 overflow-y-auto p-4 space-y-2 bg-white dark:bg-neutral-925"
			>
				{#each menuLinks.filter((item) => item.show) as item}
					{@const Icon = item.icon}
					<a
						href={item.href}
						use:inertia
						class="flex items-center gap-3 px-4 py-3 rounded-xl text-sm font-medium transition-all {item.group ===
						group
							? 'bg-brand-400/10 text-brand-600 dark:text-brand-400 border border-brand-400/20'
							: 'text-neutral-700 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-white hover:bg-neutral-100 dark:hover:bg-neutral-800/50 border border-transparent'}"
					>
						<Icon size="20" stroke-width="2" />
						{item.label}
					</a>
				{/each}
			</div>

			<!-- Footer -->
			{#if user}
				<div
					class="p-4 border-t border-neutral-200/80 dark:border-white/[0.04] bg-neutral-50 dark:bg-neutral-925/50"
				>
					<div
						class="bg-neutral-100/50 dark:bg-neutral-800/50 rounded-xl p-4 border border-neutral-200/80 dark:border-white/[0.06] mb-3"
					>
						<div class="flex items-center gap-3">
							{#if user.avatar}
								<img src={user.avatar} alt={user.name} class="w-10 h-10 rounded-full object-cover" />
							{:else}
								<div
									class="w-10 h-10 rounded-full bg-brand-600 dark:bg-brand-500 flex items-center justify-center text-white font-bold text-sm"
								>
									{user.name.charAt(0).toUpperCase()}
								</div>
							{/if}
							<div class="flex-1 min-w-0">
								<p
									class="text-sm font-semibold text-neutral-900 dark:text-white truncate"
								>
									{user.name}
								</p>
								<p
									class="text-xs text-neutral-500 dark:text-neutral-400 truncate"
								>
									{user.email || "Member"}
								</p>
							</div>
						</div>
					</div>
					<div class="flex items-center gap-2">
						<DarkModeToggle />
						<button
							onclick={handleLogout}
							class="flex-1 flex items-center justify-center gap-2 px-4 py-2.5 rounded-lg bg-red-500/10 hover:bg-red-500/20 text-red-500 dark:text-red-400 font-medium transition-colors"
						>
							<LogOut size="18" />
							Logout
						</button>
					</div>
				</div>
			{:else}
				<div
					class="p-4 border-t border-neutral-200/80 dark:border-white/[0.04] bg-neutral-50 dark:bg-neutral-925/50 space-y-2"
				>
					<a
						href="/login"
						use:inertia
						class="block w-full px-4 py-3 rounded-lg bg-neutral-200/80 dark:bg-neutral-800 hover:bg-neutral-300/80 dark:hover:bg-neutral-700 text-neutral-700 dark:text-neutral-300 font-medium transition-colors text-center"
					>
						Sign In
					</a>
					<a
						href="/register"
						use:inertia
						class="block w-full px-4 py-3 rounded-lg bg-brand-600 hover:bg-brand-700 text-white font-semibold transition-all text-center dark:bg-brand-500 dark:hover:bg-brand-400 shadow-lg shadow-brand-600/25"
					>
						Get Started
					</a>
				</div>
			{/if}
		</div>
	</div>
{/if}

<!-- Desktop spacer for header -->
<div class="hidden lg:block h-16"></div>

<!-- Mobile spacer -->
<div class="lg:hidden h-16"></div>

<!-- Page Content -->
<div class="min-h-screen bg-neutral-50 dark:bg-neutral-950">
	{@render children()}
</div>