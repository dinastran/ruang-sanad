<script lang="ts">
	import { router } from "@inertiajs/svelte";
	import type { UjrohKomponen, UjrohTarifGuru } from "@lib/types";
	import { ChevronDown, Layers, Plus } from "lucide-svelte";

	interface Props {
		komponen: UjrohKomponen[];
		guru: UjrohTarifGuru[];
		bulan: string;
		canEdit: boolean;
	}

	let { komponen, guru, bulan, canEdit }: Props = $props();

	const rupiah = (n: number) => new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 0 }).format(n);
	const inputClass = "w-full rounded-xl border border-neutral-300 bg-neutral-50 px-3 py-2 text-sm text-neutral-900 outline-none focus:border-brand-400 focus:ring-2 focus:ring-brand-400/15 dark:border-neutral-700 dark:bg-neutral-800/60 dark:text-white";
	let q = $derived(`?bulan=${encodeURIComponent(bulan)}`);

	let open = $state(false);
	let busy = $state(false);
	const opts = { preserveScroll: true, preserveState: true, onStart: () => (busy = true), onFinish: () => (busy = false) };

	let baru = $state({ nama: "", jenis: "tetap", nominal_default: 0 });
	let editID = $state<number | null>(null);
	let edit = $state({ nama: "", nominal_default: 0 });
	let penerimaID = $state<number | null>(null);
	// Draft penerima per guru: dicentang = menerima dengan nominal tsb.
	let penerimaDraft = $state<Record<number, { terima: boolean; nominal: number }>>({});
	let cariGuru = $state("");

	let guruAktif = $derived(guru.filter((g) => g.is_aktif));
	let jumlahTetap = $derived(komponen.filter((k) => k.is_aktif && k.jenis === "tetap").length);
	let jumlahInsidental = $derived(komponen.filter((k) => k.is_aktif && k.jenis === "insidental").length);

	function tambah() {
		if (!baru.nama.trim()) return;
		router.post(`/app/keuangan/ujroh/komponen${q}`, { ...baru, nominal_default: Number(baru.nominal_default) }, {
			...opts,
			onSuccess: () => (baru = { nama: "", jenis: baru.jenis, nominal_default: 0 }),
		});
	}

	function mulaiEdit(k: UjrohKomponen) {
		editID = k.id;
		edit = { nama: k.nama, nominal_default: k.nominal_default };
	}

	function simpanEdit(k: UjrohKomponen) {
		router.put(`/app/keuangan/ujroh/komponen/${k.id}${q}`, { nama: edit.nama, nominal_default: Number(edit.nominal_default), is_aktif: k.is_aktif }, {
			...opts,
			onSuccess: () => (editID = null),
		});
	}

	function toggleAktif(k: UjrohKomponen) {
		const pesan = k.is_aktif
			? `Nonaktifkan ${k.nama}? Komponen tidak lagi otomatis/terpilih di bulan yang belum dikunci. Bulan terkunci tidak berubah.`
			: `Aktifkan kembali ${k.nama}?`;
		if (!confirm(pesan)) return;
		router.put(`/app/keuangan/ujroh/komponen/${k.id}${q}`, { nama: k.nama, nominal_default: k.nominal_default, is_aktif: !k.is_aktif }, opts);
	}

	function bukaPenerima(k: UjrohKomponen) {
		if (penerimaID === k.id) {
			penerimaID = null;
			return;
		}
		penerimaID = k.id;
		cariGuru = "";
		penerimaDraft = Object.fromEntries(guruAktif.map((g) => {
			const cur = k.penerima[g.guru_id];
			return [g.guru_id, { terima: cur !== undefined, nominal: cur ?? k.nominal_default }];
		}));
	}

	let komponenPenerima = $derived(komponen.find((k) => k.id === penerimaID));
	let guruTampil = $derived(guruAktif.filter((g) => g.nama.toLowerCase().includes(cariGuru.trim().toLowerCase())));
	let ringkasPenerima = $derived.by(() => {
		let jumlah = 0, total = 0;
		for (const d of Object.values(penerimaDraft)) if (d.terima) { jumlah++; total += Number(d.nominal) || 0; }
		return { jumlah, total };
	});
	// Hanya baris yang berubah dikirim ke server.
	let perubahan = $derived.by(() => {
		const k = komponenPenerima;
		if (!k) return [];
		return guruAktif.flatMap((g) => {
			const d = penerimaDraft[g.guru_id];
			if (!d) return [];
			const cur = k.penerima[g.guru_id];
			const baru = d.terima ? Number(d.nominal) : null;
			if (baru === (cur ?? null)) return [];
			return [{ guru_id: g.guru_id, nominal: baru }];
		});
	});
	let draftValid = $derived(Object.values(penerimaDraft).every((d) => !d.terima || (Number.isFinite(Number(d.nominal)) && Number(d.nominal) >= 0)));

	function centangSemua(terima: boolean) {
		for (const g of guruTampil) penerimaDraft[g.guru_id].terima = terima;
	}

	function simpanPenerima(k: UjrohKomponen) {
		if (perubahan.length === 0 || !draftValid) return;
		router.put(`/app/keuangan/ujroh/komponen/${k.id}/penerima${q}`, { penerima: perubahan }, opts);
	}
</script>

<section class="rounded-2xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-925">
	<button onclick={() => (open = !open)} aria-expanded={open} class="flex w-full items-center justify-between gap-3 px-5 py-4 text-left">
		<span class="flex items-center gap-2 font-semibold text-neutral-900 dark:text-white"><Layers size="18" /> Komponen ujroh</span>
		<span class="flex items-center gap-2 text-sm text-neutral-500">
			{jumlahTetap} tetap · {jumlahInsidental} insidental
			<ChevronDown size="16" class="transition-transform {open ? 'rotate-180' : ''}" />
		</span>
	</button>
	{#if open}
		<div class="space-y-5 border-t border-neutral-200 dark:border-neutral-800 px-5 py-5">
			<div class="grid gap-3 text-sm text-neutral-600 dark:text-neutral-400 sm:grid-cols-2">
				<p><strong class="text-neutral-900 dark:text-white">Tetap</strong> (kuota, tunjangan kesehatan): atur nominal per guru sekali, otomatis masuk setiap bulan. Bisa diubah untuk bulan tertentu di rincian guru.</p>
				<p><strong class="text-neutral-900 dark:text-white">Insidental</strong> (MC, Qiro'ah): ditambahkan per bulan di rincian guru sebagai jumlah × nominal. Nominal default hanya isian awal.</p>
			</div>

			{#if canEdit}
				<div class="grid gap-3 rounded-xl border border-dashed border-neutral-300 p-3 dark:border-neutral-700 sm:grid-cols-[1.5fr_1fr_1fr_auto] sm:items-end">
					<label class="text-xs font-medium text-neutral-600 dark:text-neutral-400">Nama komponen<input bind:value={baru.nama} maxlength="60" placeholder="mis. Kuota internet" class="{inputClass} mt-1" /></label>
					<label class="text-xs font-medium text-neutral-600 dark:text-neutral-400">Jenis
						<select bind:value={baru.jenis} class="{inputClass} mt-1">
							<option value="tetap">Tetap (bulanan)</option>
							<option value="insidental">Insidental</option>
						</select>
					</label>
					<label class="text-xs font-medium text-neutral-600 dark:text-neutral-400">Nominal default (Rp)<input type="number" min="0" step="1000" bind:value={baru.nominal_default} class="{inputClass} mt-1" /></label>
					<button onclick={tambah} disabled={busy || !baru.nama.trim()} class="inline-flex items-center justify-center gap-1.5 rounded-xl bg-brand-600 px-4 py-2 text-sm font-semibold text-white hover:bg-brand-700 disabled:opacity-50"><Plus size="16" /> Tambah</button>
				</div>
			{/if}

			<div class="divide-y divide-neutral-100 rounded-xl border border-neutral-200 dark:divide-neutral-800/70 dark:border-neutral-800">
				{#each komponen as k (k.id)}
					{@const jumlahPenerima = Object.keys(k.penerima).length}
					<div class="px-3 py-3 {k.is_aktif ? '' : 'opacity-60'}">
						{#if editID === k.id}
							<div class="grid gap-2 sm:grid-cols-[1.5fr_1fr_auto] sm:items-end">
								<label class="text-xs text-neutral-600 dark:text-neutral-400">Nama<input bind:value={edit.nama} maxlength="60" class="{inputClass} mt-1" /></label>
								<label class="text-xs text-neutral-600 dark:text-neutral-400">Nominal default (Rp)<input type="number" min="0" step="1000" bind:value={edit.nominal_default} class="{inputClass} mt-1" /></label>
								<div class="flex gap-2">
									<button onclick={() => (editID = null)} class="rounded-lg px-3 py-2 text-xs font-medium text-neutral-600 hover:bg-neutral-100 dark:text-neutral-300 dark:hover:bg-neutral-800">Batal</button>
									<button onclick={() => simpanEdit(k)} disabled={busy || !edit.nama.trim()} class="rounded-lg bg-brand-600 px-3 py-2 text-xs font-semibold text-white hover:bg-brand-700 disabled:opacity-50">Simpan</button>
								</div>
							</div>
						{:else}
							<div class="flex flex-col gap-2 sm:flex-row sm:items-center">
								<div class="min-w-0 flex-1">
									<p class="text-sm font-medium text-neutral-900 dark:text-white">
										{k.nama}
										<span class="ml-1 rounded-md px-1.5 text-[11px] font-medium {k.jenis === 'tetap' ? 'bg-neutral-100 text-neutral-600 dark:bg-neutral-800 dark:text-neutral-300' : 'bg-brand-400/10 text-brand-700 dark:text-brand-300'}">{k.jenis}</span>
										{#if !k.is_aktif}<span class="ml-1 text-xs text-neutral-500">(nonaktif)</span>{/if}
									</p>
									<p class="text-xs text-neutral-500">
										Default {rupiah(k.nominal_default)}{k.jenis === "tetap" ? ` · ${jumlahPenerima} guru penerima` : " per kali"}
									</p>
								</div>
								{#if canEdit}
									<div class="flex flex-wrap gap-1.5">
										{#if k.jenis === "tetap" && k.is_aktif}
											<button onclick={() => bukaPenerima(k)} class="rounded-lg border border-neutral-300 px-3 py-1.5 text-xs font-semibold text-neutral-700 hover:bg-neutral-100 dark:border-neutral-700 dark:text-neutral-200 dark:hover:bg-neutral-800">{penerimaID === k.id ? "Tutup penerima" : "Atur penerima"}</button>
										{/if}
										<button onclick={() => mulaiEdit(k)} class="rounded-lg px-3 py-1.5 text-xs font-medium text-neutral-600 hover:bg-neutral-100 dark:text-neutral-300 dark:hover:bg-neutral-800">Ubah</button>
										<button onclick={() => toggleAktif(k)} disabled={busy} class="rounded-lg px-3 py-1.5 text-xs font-medium text-neutral-600 hover:bg-neutral-100 dark:text-neutral-300 dark:hover:bg-neutral-800">{k.is_aktif ? "Nonaktifkan" : "Aktifkan"}</button>
									</div>
								{/if}
							</div>
						{/if}

						{#if penerimaID === k.id}
							<div class="mt-3 rounded-xl bg-neutral-50 p-3 dark:bg-neutral-900/40">
								<p class="text-sm text-neutral-700 dark:text-neutral-300">
									Centang guru yang menerima <strong>{k.nama}</strong> setiap bulan, isi nominalnya, lalu tekan <strong>Simpan penerima</strong>. Berlaku untuk bulan yang belum dikunci.
								</p>
								<div class="mt-3 flex flex-wrap items-center gap-2">
									<input bind:value={cariGuru} placeholder="Cari guru" aria-label="Cari guru" class="{inputClass} sm:max-w-56" />
									<button onclick={() => centangSemua(true)} class="rounded-lg px-2.5 py-1.5 text-xs font-semibold text-brand-700 hover:bg-brand-400/10 dark:text-brand-300">Centang semua</button>
									<button onclick={() => centangSemua(false)} class="rounded-lg px-2.5 py-1.5 text-xs font-semibold text-neutral-600 hover:bg-neutral-100 dark:text-neutral-300 dark:hover:bg-neutral-800">Hapus centang</button>
								</div>
								<ul class="mt-3 divide-y divide-neutral-200 rounded-xl border border-neutral-200 bg-white dark:divide-neutral-800 dark:border-neutral-800 dark:bg-neutral-925">
									{#each guruTampil as g (g.guru_id)}
										{@const d = penerimaDraft[g.guru_id]}
										{#if d}
											<li class="flex flex-col gap-2 px-3 py-2.5 sm:flex-row sm:items-center">
												<label class="flex min-w-0 flex-1 cursor-pointer items-center gap-3">
													<input type="checkbox" bind:checked={d.terima} class="h-5 w-5 shrink-0 rounded accent-brand-600" />
													<span class="min-w-0">
														<span class="block font-medium text-neutral-900 dark:text-white">{g.nama}</span>
														<span class="block text-xs text-neutral-500">{g.status === "tetap" ? "Guru tetap" : g.status === "part_time" ? "Part time" : g.status}{k.penerima[g.guru_id] !== undefined ? ` · saat ini ${rupiah(k.penerima[g.guru_id])}` : ""}</span>
													</span>
												</label>
												{#if d.terima}
													<label class="flex items-center gap-2 pl-8 text-xs text-neutral-500 sm:pl-0">
														Rp
														<input type="number" min="0" step="1000" bind:value={d.nominal} aria-label="Nominal {k.nama} untuk {g.nama}" class="w-full rounded-xl border border-neutral-300 bg-neutral-50 px-3 py-2 text-sm text-neutral-900 outline-none focus:border-brand-400 focus:ring-2 focus:ring-brand-400/15 dark:border-neutral-700 dark:bg-neutral-800/60 dark:text-white sm:w-36" />
													</label>
												{:else}
													<span class="pl-8 text-xs text-neutral-400 sm:pl-0">Tidak menerima</span>
												{/if}
											</li>
										{/if}
									{:else}
										<li class="px-3 py-4 text-center text-sm text-neutral-500">Guru tidak ditemukan.</li>
									{/each}
								</ul>
								<div class="sticky bottom-0 mt-3 flex flex-col gap-2 rounded-xl border border-neutral-200 bg-white p-3 shadow-sm dark:border-neutral-800 dark:bg-neutral-925 sm:flex-row sm:items-center sm:justify-between">
									<p class="text-sm text-neutral-700 dark:text-neutral-300">
										<strong>{ringkasPenerima.jumlah}</strong> guru menerima · {rupiah(ringkasPenerima.total)}/bulan
										{#if perubahan.length > 0}<span class="block text-xs text-warning">{perubahan.length} perubahan belum disimpan</span>{/if}
									</p>
									<button onclick={() => simpanPenerima(k)} disabled={busy || perubahan.length === 0 || !draftValid} class="rounded-xl bg-brand-600 px-4 py-2.5 text-sm font-semibold text-white hover:bg-brand-700 disabled:opacity-50">Simpan penerima</button>
								</div>
							</div>
						{/if}
					</div>
				{:else}
					<p class="px-3 py-6 text-center text-sm text-neutral-500">Belum ada komponen. Tambahkan misalnya Kuota, Tunjangan Kesehatan, MC, Qiro'ah.</p>
				{/each}
			</div>
		</div>
	{/if}
</section>
