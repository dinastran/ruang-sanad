<script lang="ts">
	import type { Snippet } from "svelte";
	import Logo from "@components/Logo.svelte";
	import type { Flash } from "@lib/types";

	/**
	 * AuthLayout wraps every authentication page (login, register, forgot-password,
	 * reset-password) with the two-column split:
	 *
	 *   [ Desktop: logo + headline + stats ]
	 *   [ Mobile:  centered logo            ]
	 *   [           form card               ]
	 *
	 * Pages provide a headline/subtitle + the form as children.
	 *
	 *   <AuthLayout
	 *     title="Sign in"
	 *     subtitle="Enter your credentials to continue"
	 *     flash={flash}
	 *   >
	 *     <form>...</form>
	 *     <p>Don't have an account? <a href="/register">Create one</a></p>
	 *   </AuthLayout>
	 */
	interface Props {
		/** Form title (e.g. "Sign in", "Create account"). */
		title: string;
		/** Form subtitle shown under title. */
		subtitle?: string;
		/** Document <title> shown by browser tab. Defaults to title. */
		pageTitle?: string;
		/** Branding headline shown in desktop left panel. */
		headline: string;
		/** Branding subheadline shown under headline. */
		subheadline: string;
		/** Optional branding stats — array of { value, label }. */
		stats?: { value: string; label: string }[];
		/** Flash messages from server (set via Flash() cookies on server). */
		flash?: Flash;
		children: Snippet;
	}

	let {
		title,
		subtitle,
		pageTitle,
		headline,
		subheadline,
		stats,
		flash,
		children,
	}: Props = $props();
</script>

<svelte:head>
	<title>{pageTitle ?? title} - Laju Go</title>
</svelte:head>

<section class="min-h-screen bg-white dark:bg-neutral-950 flex">
	<!-- Left Side - Branding (Desktop only) -->
	<div class="hidden lg:flex lg:w-1/2 relative items-center justify-center p-12">
		<div class="relative z-10 max-w-lg">
			<div class="mb-8">
				<Logo size={80} />
			</div>
			<h1 class="text-4xl font-bold text-neutral-900 dark:text-white mb-4">
				{headline}
			</h1>
			<p class="text-neutral-600 dark:text-neutral-400 text-lg leading-relaxed">
				{subheadline}
			</p>

			{#if stats && stats.length > 0}
				<div class="mt-12 grid grid-cols-3 gap-6">
					{#each stats as stat}
						<div class="text-center">
							<div class="text-3xl font-bold text-brand-600 dark:text-brand-400">
								{stat.value}
							</div>
							<div class="text-sm text-neutral-500 dark:text-neutral-400 mt-1">
								{stat.label}
							</div>
						</div>
					{/each}
				</div>
			{/if}
		</div>
	</div>

	<!-- Right Side - Form -->
	<div class="w-full lg:w-1/2 flex items-center justify-center p-6 lg:p-12">
		<div class="w-full max-w-md">
			<!-- Mobile Logo -->
			<div class="lg:hidden mb-8 flex justify-center">
				<Logo size={64} />
			</div>

			<div
				class="bg-white dark:bg-neutral-925/80 backdrop-blur-xl rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] p-8 shadow-xl shadow-black/5 dark:shadow-black/20"
			>
				<div class="text-center mb-8">
					<h2 class="text-2xl font-bold text-neutral-900 dark:text-white">
						{title}
					</h2>
					{#if subtitle}
						<p class="text-neutral-600 dark:text-neutral-400 mt-2">{subtitle}</p>
					{/if}
				</div>

				{#if flash?.error}
					<div
						class="mb-6 p-4 rounded-xl bg-red-500/10 border border-red-500/20 flex items-start gap-3"
					>
						<svg
							class="w-5 h-5 text-red-400 shrink-0 mt-0.5"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="2"
								d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
							/>
						</svg>
						<span class="text-red-400 text-sm">{flash.error}</span>
					</div>
				{/if}

				{@render children()}
			</div>
		</div>
	</div>
</section>