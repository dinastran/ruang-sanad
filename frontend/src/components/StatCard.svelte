<script lang="ts">
	import { fly } from "svelte/transition";

	interface Props {
		title: string;
		value: number | string;
		icon?: string;
		color?: string;
		delay?: number;
	}

	let { title, value, icon = "", color = "brand", delay = 0 }: Props = $props();

	let colorClasses: Record<string, { bg: string; text: string; ring: string }> = {
		brand: { bg: "bg-brand-400/10", text: "text-brand-700 dark:text-brand-400", ring: "ring-brand-400/20" },
		green: { bg: "bg-green-500/10", text: "text-green-700 dark:text-green-400", ring: "ring-green-500/20" },
		blue: { bg: "bg-blue-500/10", text: "text-blue-700 dark:text-blue-400", ring: "ring-blue-500/20" },
		pink: { bg: "bg-pink-500/10", text: "text-pink-700 dark:text-pink-400", ring: "ring-pink-500/20" },
		amber: { bg: "bg-amber-500/10", text: "text-amber-700 dark:text-amber-400", ring: "ring-amber-500/20" },
	};
	let c = $derived(colorClasses[color] || colorClasses.brand);
</script>

<div
	class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-5 transition-all hover:border-brand-400/30 hover:shadow-lg hover:shadow-brand-400/5"
	in:fly={{ y: 20, duration: 500, delay }}
>
	<div class="flex items-center gap-3 mb-3">
		<div class="w-10 h-10 rounded-xl {c.bg} {c.ring} ring-1 flex items-center justify-center">
			{#if icon}
				<span class="text-lg {c.text}">{icon}</span>
			{/if}
		</div>
		<span class="text-sm font-medium text-neutral-500 dark:text-neutral-400">{title}</span>
	</div>
	<div class="text-3xl font-bold text-neutral-900 dark:text-white tracking-tight font-mono">{value}</div>
</div>
