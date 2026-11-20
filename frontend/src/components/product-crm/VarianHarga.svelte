<script lang="ts" module>
	export interface Varian {
		id: number;
		produk_id: number;
		nama: string;
		harga: number;
		paket_kode: string;
		urutan: number;
		is_aktif: boolean;
	}
</script>

<script lang="ts">
	import { router } from "@inertiajs/svelte";
	import { Pencil, Plus, Tag } from "lucide-svelte";

	interface Props { produkID: number; varian: Varian[] }
	let { produkID, varian }: Props = $props();

	const kosong = () => ({ nama: "", harga: 0, paket_kode: "", urutan: varian.length + 1, is_aktif: true });
	let terbuka = $state(false);
	let editID = $state<number | null>(null);
	let form = $state(kosong());
	let menyimpan = $state(false);

	const inputCls = "w-full rounded-lg border border-neutral-300 bg-neutral-50 px-3 py-2 text-sm text-neutral-900 outline-none focus:border-brand-400 focus:ring-2 focus:ring-brand-400/15 dark:border-neutral-700 dark:bg-neutral-900 dark:text-white";
	const rupiah = (n: number) => "Rp" + (n ?? 0).toLocaleString("id-ID");

	function tambah() {
		form = kosong();
		editID = null;
		terbuka = true;
	}
	function ubah(v: Varian) {
		form = { nama: v.nama, harga: v.harga, paket_kode: v.paket_kode, urutan: v.urutan, is_aktif: v.is_aktif };
		editID = v.id;
		terbuka = true;
	}
	function simpan(e: Event) {
		e.preventDefault();
		menyimpan = true;
		const body = { ...form, harga: Number(form.harga), urutan: Number(form.urutan) };
		const opts = { preserveScroll: true, onSuccess: () => (terbuka = false), onFinish: () => (menyimpan = false) };
		if (editID) router.put(`/app/produk-crm/varian/${editID}`, body, opts);
		else router.post(`/app/produk-crm/produk/${produkID}/varian`, body, opts);
	}
</script>

<div class="mt-4 border-t border-neutral-100 pt-4 dark:border-white/[0.06]">
	<div class="flex items-center justify-between">
		<p class="flex items-center gap-1.5 text-xs font-semibold uppercase tracking-wide text-neutral-500"><Tag size="13" /> Varian harga</p>
		<button type="button" onclick={tambah} class="inline-flex items-center gap-1 text-xs font-semibold text-brand-600 dark:text-brand-400"><Plus size="13" /> Tambah varian</button>
	</div>
	{#if varian.length}
		<ul class="mt-2 divide-y divide-neutral-100 dark:divide-white/[0.06]">
			{#each varian as v (v.id)}
				<li class="flex items-center justify-between gap-3 py-2 text-sm">
					<div class="min-w-0">
						<span class="font-medium text-neutral-800 dark:text-neutral-200 {v.is_aktif ? '' : 'line-through opacity-60'}">{v.nama}</span>
						{#if v.paket_kode}<span class="ml-1.5 rounded bg-neutral-100 px-1.5 py-0.5 text-[11px] text-neutral-500 dark:bg-neutral-800">paket {v.paket_kode}</span>{/if}
						{#if !v.is_aktif}<span class="ml-1.5 text-[11px] text-neutral-500">nonaktif</span>{/if}
					</div>
					<div class="flex shrink-0 items-center gap-3">
						<span class="tabular-nums font-semibold text-neutral-900 dark:text-white">{rupiah(v.harga)}</span>
						<button type="button" onclick={() => ubah(v)} aria-label="Ubah varian {v.nama}" class="text-neutral-400 hover:text-neutral-700 dark:hover:text-neutral-200"><Pencil size="14" /></button>
					</div>
				</li>
			{/each}
		</ul>
	{:else}
		<p class="mt-2 text-xs text-neutral-500">Belum ada varian. Form pendaftaran memakai harga produk.</p>
	{/if}

	{#if terbuka}
		<form onsubmit={simpan} class="mt-3 grid gap-2 rounded-xl bg-neutral-50 p-3 sm:grid-cols-[1fr_140px_100px] dark:bg-neutral-900/60">
			<input bind:value={form.nama} required maxlength="80" placeholder="Mis. 2x / pekan" class={inputCls} />
			<input type="number" min="0" step="1000" bind:value={form.harga} required placeholder="Harga" class={inputCls} />
			<input bind:value={form.paket_kode} placeholder="Paket (1x/2x)" class={inputCls} />
			<div class="flex flex-wrap items-center gap-3 sm:col-span-3">
				<label class="flex items-center gap-1.5 text-xs text-neutral-600 dark:text-neutral-400">Urutan <input type="number" min="0" bind:value={form.urutan} class="{inputCls} !w-16 !py-1" /></label>
				<label class="flex items-center gap-1.5 text-xs text-neutral-600 dark:text-neutral-400"><input type="checkbox" bind:checked={form.is_aktif} /> Aktif</label>
				<div class="ml-auto flex gap-2">
					<button type="button" onclick={() => (terbuka = false)} class="rounded-lg px-3 py-1.5 text-xs text-neutral-500">Batal</button>
					<button disabled={menyimpan} class="rounded-lg bg-brand-600 px-3 py-1.5 text-xs font-semibold text-white disabled:opacity-50">{editID ? "Simpan" : "Tambah"}</button>
				</div>
			</div>
		</form>
	{/if}
</div>
