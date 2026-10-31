<script lang="ts">
	import { router } from "@inertiajs/svelte";
	import { BookOpenCheck, Save } from "lucide-svelte";

	type MasterLevel = { id: number; kode: string; nama: string; urutan: number };
	type ApprovedLevel = { guru_id: number; level_kode: string; approved_by: number; approved_at: string };
	let {guruId, data}: {guruId:number; data?:{available:MasterLevel[]; approved:ApprovedLevel[]}} = $props();
	let chosen = $state<string[]>(data?.approved?.map(x=>x.level_kode) ?? []);
	let saving = $state(false);
	const levels = $derived(data?.available ?? []);
	function toggle(code:string,checked:boolean) {
		chosen = checked ? [...new Set([...chosen,code])] : chosen.filter(x=>x!==code);
	}
	function save() {
		if(saving)return;
		saving=true;
		router.put("/app/koordinator-guru/guru/"+guruId+"/teaching-levels",
			{level_codes:chosen},{preserveScroll:true,onFinish:()=>{saving=false;}});
	}
</script>

<section class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-5 sm:p-6 space-y-4">
	<div>
		<h2 class="flex items-center gap-2 font-semibold text-neutral-900 dark:text-white"><BookOpenCheck class="w-5 h-5 text-brand-600"/> Izin Mengajar per Level</h2>
		<p class="mt-1 text-sm text-neutral-600 dark:text-neutral-400">Koordinator Guru menyetujui level yang boleh diajar. Ini terpisah dari matriks kompetensi umum dan dipakai engine rekomendasi jadwal.</p>
	</div>
	{#if levels.length===0}
		<p class="rounded-lg bg-amber-500/10 p-3 text-sm text-amber-700 dark:text-amber-300">Master level belum tersedia. Tambahkan level di Master Data terlebih dahulu.</p>
	{:else}
		<div class="grid gap-2 sm:grid-cols-2">
			{#each levels as level (level.kode)}
				<label class="flex items-center gap-3 rounded-lg border border-neutral-200 dark:border-white/10 px-3 py-2 text-sm cursor-pointer">
					<input type="checkbox" checked={chosen.includes(level.kode)} onchange={e=>toggle(level.kode,e.currentTarget.checked)} class="accent-brand-600 w-4 h-4"/>
					<span class="text-neutral-800 dark:text-neutral-200">{level.nama} <span class="text-neutral-500">({level.kode})</span></span>
				</label>
			{/each}
		</div>
	{/if}
	<div class="flex flex-wrap items-center justify-between gap-2">
		<p class="text-xs text-neutral-500">Kosongkan semua pilihan untuk mencabut seluruh izin mengajar level. Tidak mengubah kelas aktif.</p>
		<button type="button" disabled={saving || levels.length===0} onclick={save} class="inline-flex items-center gap-2 rounded-lg bg-brand-600 px-4 py-2 text-sm font-semibold text-white disabled:opacity-50"><Save class="w-4 h-4"/> {saving?"Menyimpan...":"Simpan persetujuan"}</button>
	</div>
</section>
