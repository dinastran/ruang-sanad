<script lang="ts">
	// Pilih dzikir atau paket wirid; kelola dzikir custom.
	import { router } from "@inertiajs/svelte";
	import { Check, ListOrdered, Pencil, Plus, Archive } from "lucide-svelte";
	import Sheet from "./Sheet.svelte";
	import type { Dzikir, Paket } from "@lib/tasbih/types";

	interface Props {
		open: boolean;
		dzikir: Dzikir[];
		paket: Paket[];
		aktifDzikir: number | null;
		aktifPaket: number | null;
		onclose: () => void;
		onpilihdzikir: (id: number) => void;
		onpilihpaket: (id: number) => void;
		ontambah: () => void;
		onubah: (d: Dzikir) => void;
	}
	let { open, dzikir, paket, aktifDzikir, aktifPaket, onclose, onpilihdzikir, onpilihpaket, ontambah, onubah }: Props = $props();
	let tab = $state<"dzikir" | "paket">("dzikir");

	const namaDzikir = (id: number) => dzikir.find((d) => d.id === id)?.latin ?? "";

	function arsipkan(d: Dzikir) {
		if (!confirm(`Arsipkan dzikir "${d.latin}"? Riwayatnya tetap tersimpan.`)) return;
		router.delete(`/app/tasbih/dzikir/${d.id}`, { preserveScroll: true });
	}
</script>

<Sheet {open} title="Pilih dzikir" {onclose}>
	<div class="mb-4 grid grid-cols-2 gap-1 rounded-xl bg-neutral-100 p-1 dark:bg-neutral-800" role="tablist">
		{#each [["dzikir", "Dzikir"], ["paket", "Paket wirid"]] as [k, l]}
			<button
				type="button"
				role="tab"
				aria-selected={tab === k}
				onclick={() => (tab = k as "dzikir" | "paket")}
				class="h-10 rounded-lg text-sm font-medium transition {tab === k ? 'bg-white text-neutral-900 shadow-sm dark:bg-neutral-950 dark:text-white' : 'text-neutral-500 dark:text-neutral-400'}"
			>{l}</button>
		{/each}
	</div>

	{#if tab === "dzikir"}
		<ul class="space-y-1.5">
			{#each dzikir as d (d.id)}
				<li class="flex items-center gap-1">
					<button
						type="button"
						onclick={() => onpilihdzikir(d.id)}
						class="flex min-h-14 flex-1 items-center gap-3 rounded-xl px-3 py-2 text-left transition hover:bg-neutral-50 dark:hover:bg-neutral-800 {aktifDzikir === d.id && !aktifPaket ? 'bg-brand-50 ring-1 ring-brand-200 dark:bg-brand-950/40 dark:ring-brand-900' : ''}"
					>
						<span class="min-w-0 flex-1">
							<span class="block truncate text-sm font-semibold text-neutral-900 dark:text-white">{d.latin}</span>
							{#if d.arab}<span class="block truncate font-arab text-lg leading-snug text-neutral-600 dark:text-neutral-300" dir="rtl" lang="ar">{d.arab}</span>{/if}
						</span>
						<span class="shrink-0 text-xs tabular-nums text-neutral-500">×{d.target}</span>
						{#if aktifDzikir === d.id && !aktifPaket}<Check size={16} class="shrink-0 text-brand-600 dark:text-brand-400" aria-label="Aktif" />{/if}
					</button>
					{#if d.custom}
						<button type="button" class="grid h-11 w-11 place-items-center rounded-lg text-neutral-500 hover:bg-neutral-100 dark:hover:bg-neutral-800" aria-label="Ubah {d.latin}" onclick={() => onubah(d)}><Pencil size={16} /></button>
						<button type="button" class="grid h-11 w-11 place-items-center rounded-lg text-neutral-500 hover:bg-neutral-100 dark:hover:bg-neutral-800" aria-label="Arsipkan {d.latin}" onclick={() => arsipkan(d)}><Archive size={16} /></button>
					{/if}
				</li>
			{/each}
		</ul>
		<button type="button" onclick={ontambah} class="mt-3 flex h-12 w-full items-center justify-center gap-2 rounded-xl border border-dashed border-neutral-300 text-sm font-medium text-neutral-600 hover:border-brand-400 hover:text-brand-700 dark:border-neutral-700 dark:text-neutral-300">
			<Plus size={16} /> Tambah dzikir sendiri
		</button>
	{:else}
		<ul class="space-y-2">
			{#each paket as p (p.id)}
				<li>
					<button
						type="button"
						onclick={() => onpilihpaket(p.id)}
						class="w-full rounded-2xl border px-4 py-3 text-left transition hover:border-brand-300 {aktifPaket === p.id ? 'border-brand-400 bg-brand-50 dark:bg-brand-950/40' : 'border-neutral-200 dark:border-neutral-800'}"
					>
						<span class="flex items-center gap-2 text-sm font-semibold text-neutral-900 dark:text-white"><ListOrdered size={16} class="text-brand-600 dark:text-brand-400" />{p.nama}</span>
						<span class="mt-1 block text-xs text-neutral-500 dark:text-neutral-400">{p.items.map((it) => `${namaDzikir(it.dzikir_id)} ${it.target}`).join(" → ")}</span>
						<span class="mt-1 block text-[11px] text-neutral-400">{p.rujukan}</span>
					</button>
				</li>
			{/each}
		</ul>
	{/if}
</Sheet>
