<script lang="ts">
	// Bottom sheet (mobile) / dialog tengah (desktop). Escape & klik latar menutup.
	import { fade, fly } from "svelte/transition";
	import { X } from "lucide-svelte";
	import type { Snippet } from "svelte";

	interface Props {
		open: boolean;
		title: string;
		onclose: () => void;
		children: Snippet;
	}
	let { open, title, onclose, children }: Props = $props();
	let panel = $state<HTMLDivElement | null>(null);

	$effect(() => {
		if (open && panel) panel.focus();
	});
</script>

<svelte:window onkeydown={(e) => open && e.key === "Escape" && onclose()} />

{#if open}
	<div class="fixed inset-0 z-50 flex items-end justify-center sm:items-center" role="presentation">
		<button type="button" class="absolute inset-0 bg-neutral-950/50" aria-label="Tutup" transition:fade={{ duration: 150 }} onclick={onclose}></button>
		<div
			bind:this={panel}
			tabindex="-1"
			role="dialog"
			aria-modal="true"
			aria-label={title}
			transition:fly={{ y: 40, duration: 220 }}
			class="relative flex max-h-[85vh] w-full flex-col rounded-t-3xl bg-white shadow-2xl outline-none sm:max-w-lg sm:rounded-3xl dark:bg-neutral-900"
		>
			<div class="flex items-center justify-between border-b border-neutral-100 px-5 py-4 dark:border-neutral-800">
				<h2 class="text-base font-semibold text-neutral-900 dark:text-white">{title}</h2>
				<button type="button" onclick={onclose} class="grid h-11 w-11 place-items-center rounded-full text-neutral-500 hover:bg-neutral-100 dark:hover:bg-neutral-800" aria-label="Tutup">
					<X size={20} />
				</button>
			</div>
			<div class="overflow-y-auto px-5 py-4 pb-[max(1rem,env(safe-area-inset-bottom))]">
				{@render children()}
			</div>
		</div>
	</div>
{/if}
