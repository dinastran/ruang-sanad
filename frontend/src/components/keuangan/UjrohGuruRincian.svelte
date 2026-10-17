<script lang="ts">
	import { router } from "@inertiajs/svelte";
	import type { UjrohGuru, UjrohKomponen, UjrohKomponenLine, UjrohPertemuan } from "@lib/types";
	import { Pencil, Plus, RotateCcw, Trash2 } from "lucide-svelte";

	interface Props {
		guru: UjrohGuru;
		bulan: string;
		terkunci: boolean;
		canEdit: boolean;
		komponen: UjrohKomponen[];
	}

	let { guru, bulan, terkunci, canEdit, komponen }: Props = $props();

	const rupiah = (n: number) => new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 0 }).format(n);
	const tanggalPendek = (t: string) => (t ? new Date(t + "T00:00:00").toLocaleDateString("id-ID", { day: "numeric", month: "short" }) : "");
	const tanggalPanjang = (t: string) => (t ? new Date(t + "T00:00:00").toLocaleDateString("id-ID", { day: "numeric", month: "short", year: "numeric" }) : "");
	const inputClass = "w-full rounded-lg border border-neutral-300 bg-neutral-50 px-2.5 py-1.5 text-sm text-neutral-900 outline-none focus:border-brand-400 focus:ring-2 focus:ring-brand-400/15 dark:border-neutral-700 dark:bg-neutral-800/60 dark:text-white";

	let bisaUbah = $derived(canEdit && !terkunci);
	let komponenAktif = $derived(komponen.filter((k) => k.is_aktif));
	let belumDipakai = $derived(komponenAktif.filter((k) => !guru.komponen.some((l) => l.komponen_id === k.id)));

	function perKelas(items: UjrohPertemuan[]) {
		const groups = new Map<number, { nama: string; items: UjrohPertemuan[] }>();
		for (const p of items) {
			const g = groups.get(p.kelas_id) ?? { nama: p.kelas_nama, items: [] };
			g.items.push(p);
			groups.set(p.kelas_id, g);
		}
		return Array.from(groups.entries()).map(([id, g]) => ({ id, ...g }));
	}

	// Satu form dipakai untuk menambah atau mengubah komponen bulan ini.
	let form = $state<{ komponen_id: number; jumlah: number; tarif: number; catatan: string } | null>(null);
	let busy = $state(false);
	const opts = { preserveScroll: true, preserveState: true, onStart: () => (busy = true), onFinish: () => (busy = false) };

	function mulaiTambah() {
		const k = belumDipakai[0];
		if (!k) return;
		form = { komponen_id: k.id, jumlah: 1, tarif: k.nominal_default, catatan: "" };
	}

	function pilihKomponen(id: number) {
		const k = komponen.find((x) => x.id === id);
		if (form && k) form = { ...form, komponen_id: id, tarif: k.nominal_default };
	}

	function mulaiUbah(l: UjrohKomponenLine) {
		form = { komponen_id: l.komponen_id, jumlah: l.jumlah, tarif: l.tarif, catatan: l.catatan };
	}

	let formKomponen = $derived(form ? komponen.find((k) => k.id === form!.komponen_id) : undefined);
	let formValid = $derived(
		!!form && Number(form.tarif) >= 0 && Number(form.jumlah) >= (formKomponen?.jenis === "insidental" ? 1 : 0),
	);

	function simpan() {
		if (!form || !formValid) return;
		router.put(
			`/app/keuangan/ujroh/${bulan}/guru/${guru.guru_id}/komponen/${form.komponen_id}`,
			{ jumlah: Number(form.jumlah), tarif: Number(form.tarif), catatan: form.catatan },
			{ ...opts, onSuccess: () => (form = null) },
		);
	}

	function hapus(l: UjrohKomponenLine) {
		const pesan = l.jenis === "tetap"
			? `Kembalikan ${l.komponen_nama} ke nominal tetap ${rupiah(l.nominal_tetap)}?`
			: `Hapus ${l.komponen_nama} dari ujroh ${guru.guru_nama} bulan ini?`;
		if (!confirm(pesan)) return;
		router.delete(`/app/keuangan/ujroh/${bulan}/guru/${guru.guru_id}/komponen/${l.komponen_id}`, opts);
	}

	function tiadakan(l: UjrohKomponenLine) {
		if (!confirm(`${l.komponen_nama} tidak diberikan ke ${guru.guru_nama} bulan ini?`)) return;
		router.put(`/app/keuangan/ujroh/${bulan}/guru/${guru.guru_id}/komponen/${l.komponen_id}`, { jumlah: 0, tarif: l.tarif, catatan: l.catatan }, opts);
	}
</script>

<div class="grid gap-4 lg:grid-cols-[1fr_minmax(0,22rem)]">
	<div class="space-y-4">
		{#if guru.pertemuan.length > 0}
			<div>
				<p class="mb-2 text-xs font-semibold uppercase tracking-wide text-neutral-500">Pertemuan</p>
				<div class="grid gap-3 md:grid-cols-2">
					{#each perKelas(guru.pertemuan) as k (k.id)}
						<div class="rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-925 p-3">
							<p class="text-sm font-semibold text-neutral-900 dark:text-white">{k.nama}</p>
							<p class="text-xs text-neutral-500">{k.items.length} pertemuan · {rupiah(k.items.reduce((a, p) => a + p.tarif, 0))}</p>
							<ul class="mt-2 flex flex-wrap gap-1.5">
								{#each k.items as p (p.pertemuan_id)}
									<li class="rounded-md px-2 py-0.5 text-xs {p.is_badal ? 'bg-brand-400/10 text-brand-700 dark:text-brand-300' : 'bg-neutral-100 text-neutral-700 dark:bg-neutral-800 dark:text-neutral-300'}" title="Pertemuan ke-{p.pertemuan_ke}">
										{tanggalPendek(p.tanggal)} · ke-{p.pertemuan_ke}{p.is_badal ? " · badal" : ""}
									</li>
								{/each}
							</ul>
						</div>
					{/each}
				</div>
			</div>
		{/if}

		<div>
			<div class="mb-2 flex items-center justify-between gap-2">
				<p class="text-xs font-semibold uppercase tracking-wide text-neutral-500">Komponen</p>
				{#if bisaUbah && belumDipakai.length > 0 && !form}
					<button onclick={mulaiTambah} class="inline-flex items-center gap-1 rounded-lg border border-neutral-300 px-2.5 py-1 text-xs font-semibold text-neutral-700 hover:bg-neutral-100 dark:border-neutral-700 dark:text-neutral-200 dark:hover:bg-neutral-800">
						<Plus size="14" /> Tambah komponen
					</button>
				{/if}
			</div>
			<div class="divide-y divide-neutral-100 rounded-xl border border-neutral-200 bg-white dark:divide-neutral-800/70 dark:border-neutral-800 dark:bg-neutral-925">
				{#each guru.komponen as l (l.komponen_id)}
					<div class="flex flex-wrap items-center gap-x-3 gap-y-1 px-3 py-2 text-sm {l.total === 0 ? 'opacity-60' : ''}">
						<div class="min-w-0 flex-1">
							<p class="font-medium text-neutral-900 dark:text-white">
								{l.komponen_nama}
								<span class="ml-1 rounded-md px-1.5 text-[11px] font-medium {l.jenis === 'tetap' ? 'bg-neutral-100 text-neutral-600 dark:bg-neutral-800 dark:text-neutral-300' : 'bg-brand-400/10 text-brand-700 dark:text-brand-300'}">{l.jenis === "tetap" ? "tetap" : "insidental"}</span>
								{#if l.jenis === "tetap" && l.diubah_bulan}<span class="ml-1 rounded-md bg-warning/10 px-1.5 text-[11px] font-medium text-warning">diubah bulan ini</span>{/if}
							</p>
							<p class="text-xs text-neutral-500">
								{#if l.total === 0}Tidak diberikan bulan ini{:else if l.jumlah !== 1}{l.jumlah} × {rupiah(l.tarif)}{:else}{rupiah(l.tarif)}{/if}{l.catatan ? ` · ${l.catatan}` : ""}
							</p>
						</div>
						<p class="font-mono font-semibold text-neutral-900 dark:text-white">{rupiah(l.total)}</p>
						{#if bisaUbah}
							<div class="flex items-center gap-0.5">
								<button onclick={() => mulaiUbah(l)} title="Ubah untuk bulan ini" class="rounded-md p-1.5 text-neutral-500 hover:bg-neutral-100 dark:hover:bg-neutral-800"><Pencil size="14" /></button>
								{#if l.jenis === "tetap" && l.diubah_bulan}
									<button onclick={() => hapus(l)} disabled={busy} title="Kembalikan ke nominal tetap" class="rounded-md p-1.5 text-neutral-500 hover:bg-neutral-100 dark:hover:bg-neutral-800"><RotateCcw size="14" /></button>
								{:else if l.jenis === "tetap"}
									<button onclick={() => tiadakan(l)} disabled={busy} title="Tidak diberikan bulan ini" class="rounded-md p-1.5 text-neutral-500 hover:bg-error/10 hover:text-error"><Trash2 size="14" /></button>
								{:else}
									<button onclick={() => hapus(l)} disabled={busy} title="Hapus dari bulan ini" class="rounded-md p-1.5 text-neutral-500 hover:bg-error/10 hover:text-error"><Trash2 size="14" /></button>
								{/if}
							</div>
						{/if}
					</div>
				{:else}
					<p class="px-3 py-3 text-xs text-neutral-500">Tidak ada komponen bulan ini.</p>
				{/each}

				{#if form}
					<div class="space-y-2 bg-neutral-50 px-3 py-3 dark:bg-neutral-900/40">
						<div class="grid gap-2 sm:grid-cols-[1.4fr_0.6fr_1fr]">
							<label class="text-xs text-neutral-600 dark:text-neutral-400">Komponen
								<select value={form.komponen_id} onchange={(e) => pilihKomponen(Number(e.currentTarget.value))} disabled={guru.komponen.some((l) => l.komponen_id === form?.komponen_id)} class="{inputClass} mt-1">
									{#each komponen.filter((k) => k.is_aktif || k.id === form?.komponen_id) as k (k.id)}
										{#if k.id === form.komponen_id || !guru.komponen.some((l) => l.komponen_id === k.id)}
											<option value={k.id}>{k.nama} ({k.jenis})</option>
										{/if}
									{/each}
								</select>
							</label>
							<label class="text-xs text-neutral-600 dark:text-neutral-400">Jumlah<input type="number" min={formKomponen?.jenis === "insidental" ? 1 : 0} step="1" bind:value={form.jumlah} class="{inputClass} mt-1" /></label>
							<label class="text-xs text-neutral-600 dark:text-neutral-400">Nominal (Rp)<input type="number" min="0" step="1000" bind:value={form.tarif} class="{inputClass} mt-1" /></label>
						</div>
						<label class="block text-xs text-neutral-600 dark:text-neutral-400">Catatan<input bind:value={form.catatan} placeholder="mis. MC walimah 12 Sep" class="{inputClass} mt-1" /></label>
						<div class="flex items-center justify-between gap-2">
							<p class="text-xs text-neutral-500">Total {rupiah(Number(form.jumlah || 0) * Number(form.tarif || 0))} · hanya untuk bulan ini</p>
							<div class="flex gap-2">
								<button onclick={() => (form = null)} class="rounded-lg px-3 py-1.5 text-xs font-medium text-neutral-600 hover:bg-neutral-100 dark:text-neutral-300 dark:hover:bg-neutral-800">Batal</button>
								<button onclick={simpan} disabled={busy || !formValid} class="rounded-lg bg-brand-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-brand-700 disabled:opacity-50">Simpan</button>
							</div>
						</div>
					</div>
				{/if}
			</div>
			{#if bisaUbah && komponenAktif.length === 0}
				<p class="mt-1 text-xs text-neutral-500">Buat komponen dulu di bagian "Komponen ujroh" di bawah.</p>
			{/if}
		</div>

		{#if guru.kasbon.length > 0}
			<div>
				<p class="mb-2 text-xs font-semibold uppercase tracking-wide text-neutral-500">Potongan kasbon</p>
				<div class="divide-y divide-neutral-100 rounded-xl border border-neutral-200 bg-white dark:divide-neutral-800/70 dark:border-neutral-800 dark:bg-neutral-925">
					{#each guru.kasbon as k (k.kasbon_id)}
						<div class="flex flex-wrap items-center gap-x-3 gap-y-1 px-3 py-2 text-sm">
							<div class="min-w-0 flex-1">
								<p class="font-medium text-neutral-900 dark:text-white">Kasbon {tanggalPanjang(k.tanggal)} · {rupiah(k.total_kasbon)}</p>
								<p class="text-xs text-neutral-500">
									Sisa {rupiah(k.sisa_sebelum)} → {rupiah(k.sisa_setelah)}{k.keterangan ? ` · ${k.keterangan}` : ""}
								</p>
								{#if k.potongan === 0}
									<p class="text-xs text-warning">Ujroh bulan ini tidak cukup; dipotong bulan berikutnya.</p>
								{/if}
							</div>
							<p class="font-mono font-semibold text-error">−{rupiah(k.potongan)}</p>
						</div>
					{/each}
				</div>
			</div>
		{/if}
	</div>

	<aside class="h-fit rounded-xl border border-neutral-200 bg-white p-4 text-sm dark:border-neutral-800 dark:bg-neutral-925">
		<p class="text-xs font-semibold uppercase tracking-wide text-neutral-500">Slip {guru.guru_nama}</p>
		<dl class="mt-3 space-y-1.5">
			<div class="flex justify-between gap-3"><dt class="text-neutral-600 dark:text-neutral-400">Pokok ({guru.jumlah_pertemuan} pertemuan)</dt><dd class="font-mono">{rupiah(guru.total)}</dd></div>
			{#each guru.komponen.filter((l) => l.total > 0) as l (l.komponen_id)}
				<div class="flex justify-between gap-3"><dt class="text-neutral-600 dark:text-neutral-400">{l.komponen_nama}</dt><dd class="font-mono">{rupiah(l.total)}</dd></div>
			{/each}
			{#if guru.total_kasbon > 0}
				<div class="flex justify-between gap-3"><dt class="text-neutral-600 dark:text-neutral-400">Potongan kasbon</dt><dd class="font-mono text-error">−{rupiah(guru.total_kasbon)}</dd></div>
			{/if}
			<div class="mt-2 flex justify-between gap-3 border-t border-neutral-200 pt-2 font-semibold dark:border-neutral-800"><dt class="text-neutral-900 dark:text-white">Diterima</dt><dd class="font-mono text-neutral-900 dark:text-white">{rupiah(guru.diterima)}</dd></div>
		</dl>
	</aside>
</div>
