<script lang="ts" module>
	export interface AffiliatePanelData {
		current: { affiliate_id: number; kode: string; nama: string; tipe: string; sumber: string; dikunci_at: string } | null;
		options: { id: number; nama: string }[];
	}
</script>

<script lang="ts">
	import { router } from "@inertiajs/svelte";
	import { Share2 } from "lucide-svelte";

	interface Props { leadID: number; panel: AffiliatePanelData; canEdit: boolean }
	let { leadID, panel, canEdit }: Props = $props();
	let ubah = $state(false);
	let pilihan = $state(panel.current?.affiliate_id ?? 0);
	const SUMBER: Record<string, string> = { cookie: "klik link", kode_form: "kode di form", kode_wa: "kode di WhatsApp", manual: "diatur manual" };

	function simpan(e: Event) {
		e.preventDefault();
		router.put(`/app/leads/${leadID}/affiliate`, { affiliate_id: Number(pilihan) }, { preserveScroll: true, onSuccess: () => (ubah = false) });
	}
</script>

{#if panel.current || canEdit}
	<section class="rounded-xl border border-neutral-200/80 bg-white p-4 dark:border-white/[0.06] dark:bg-neutral-925/50">
		<div class="flex items-center justify-between gap-2">
			<h2 class="flex items-center gap-1.5 text-sm font-semibold text-neutral-900 dark:text-white"><Share2 size={14} /> Affiliate</h2>
			{#if canEdit && !ubah}<button type="button" class="text-xs font-semibold text-brand-700 hover:underline dark:text-brand-400" onclick={() => (ubah = true)}>{panel.current ? "Ubah" : "Atur"}</button>{/if}
		</div>
		{#if ubah}
			<form class="mt-2 flex gap-2" onsubmit={simpan}>
				<select bind:value={pilihan} aria-label="Pilih affiliate" class="h-8 min-w-0 flex-1 rounded-md border border-neutral-200 bg-neutral-50 px-2 text-xs dark:border-white/10 dark:bg-neutral-950 dark:text-white">
					<option value={0}>— Tanpa affiliate —</option>
					{#each panel.options as o (o.id)}<option value={o.id}>{o.nama}</option>{/each}
				</select>
				<button type="submit" class="h-8 rounded-md bg-brand-600 px-2.5 text-xs font-semibold text-white">Simpan</button>
				<button type="button" class="h-8 px-1 text-xs text-neutral-500" onclick={() => (ubah = false)}>Batal</button>
			</form>
		{:else if panel.current}
			<p class="mt-2 text-sm text-neutral-800 dark:text-neutral-200">{panel.current.nama} <span class="font-mono text-xs text-neutral-500">{panel.current.kode}</span></p>
			<p class="text-xs text-neutral-500">{panel.current.tipe === "guru" ? "Guru" : "Eksternal"} · {SUMBER[panel.current.sumber] ?? panel.current.sumber}</p>
		{:else}
			<p class="mt-2 text-xs text-neutral-500">Lead ini tidak berasal dari affiliate.</p>
		{/if}
	</section>
{/if}
