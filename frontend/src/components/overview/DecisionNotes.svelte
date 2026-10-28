<script lang="ts" module>
	export interface Note {
		topic: string;
		text: string;
		caveat: string;
	}
</script>

<script lang="ts">
	import { kicker, panel } from "./ui";

	let { notes }: { notes: Note[] } = $props();
</script>

<aside class={panel + " min-w-0 px-5 py-5 sm:px-6"} aria-labelledby="catatan-title">
	<h2 id="catatan-title" class={kicker}>Catatan untuk keputusan</h2>
	<p class="mt-1 text-sm text-neutral-600 dark:text-neutral-300">Dibaca otomatis dari data yang tersedia.</p>
	{#if notes.length}
		<ul class="mt-4 space-y-4">
			{#each notes as note (note.topic)}
				<li class="border-l-2 border-brand-500/40 pl-3">
					<p class="text-[11px] font-medium text-neutral-500 dark:text-neutral-400">{note.topic}</p>
					<p class="mt-0.5 text-sm leading-relaxed text-neutral-800 [text-wrap:pretty] dark:text-neutral-200">{note.text}</p>
					{#if note.caveat}<p class="mt-1 text-xs leading-relaxed text-neutral-500 dark:text-neutral-400">{note.caveat}</p>{/if}
				</li>
			{/each}
		</ul>
	{:else}
		<p class="mt-6 text-sm text-neutral-500 dark:text-neutral-400">Catatan menunggu sumber data berhasil dimuat.</p>
	{/if}
</aside>
