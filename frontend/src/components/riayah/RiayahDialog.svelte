<script lang="ts">
	import type { Snippet } from "svelte";
	import { X } from "lucide-svelte";

	interface Props {
		title: string;
		subtitle?: string;
		busy?: boolean;
		onclose: () => void;
		children: Snippet;
	}

	let { title, subtitle = "", busy = false, onclose, children }: Props = $props();
	let dialogElement = $state<HTMLDivElement>();
	const titleId = `dialog-${Math.random().toString(36).slice(2, 9)}`;

	$effect(() => {
		dialogElement?.focus();
	});

	function close() {
		if (!busy) onclose();
	}
</script>

<svelte:window onkeydown={(event) => event.key === "Escape" && close()} />

<div class="fixed inset-0 z-50 flex items-end justify-center p-0 sm:items-center sm:p-4">
	<button type="button" class="absolute inset-0 bg-neutral-950/65" aria-label="Tutup dialog" onclick={close}></button>
	<div bind:this={dialogElement} class="relative max-h-[92dvh] w-full max-w-lg overflow-y-auto rounded-t-2xl border border-neutral-200 bg-white shadow-xl focus:outline-none sm:rounded-2xl dark:border-white/[0.08] dark:bg-neutral-925" role="dialog" aria-modal="true" aria-labelledby={titleId} tabindex="-1">
		<div class="flex items-start justify-between gap-4 border-b border-neutral-200/80 px-5 py-4 dark:border-white/[0.05]">
			<div class="min-w-0">
				<h2 id={titleId} class="text-lg font-semibold text-neutral-900 dark:text-white">{title}</h2>
				{#if subtitle}<p class="mt-0.5 text-sm text-neutral-500">{subtitle}</p>{/if}
			</div>
			<button type="button" onclick={close} disabled={busy} class="rounded-lg p-1.5 text-neutral-500 hover:bg-neutral-100 disabled:opacity-50 dark:hover:bg-neutral-800" aria-label="Tutup dialog"><X class="h-5 w-5" /></button>
		</div>
		<div class="p-5">
			{@render children()}
		</div>
	</div>
</div>
