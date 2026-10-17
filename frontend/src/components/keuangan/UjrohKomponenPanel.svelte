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
	let penerimaDraft = $state<Record<number, string | number | null>>({});

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
		penerimaDraft = Object.fromEntries(guruAktif.map((g) => [g.guru_id, k.penerima[g.guru_id] !== undefined ? String(k.penerima[g.guru_id]) : ""]));
	}

	function simpanPenerima(k: UjrohKomponen, guruID: number) {
		const raw = String(penerimaDraft[guruID] ?? "").trim();
		const nominal = raw === "" ? null : Number(raw);
		if (nominal !== null && (!Number.isFinite(nominal) || nominal < 0)) return;
		router.put(`/app/keuangan/ujroh/komponen/${k.id}/guru/${guruID}${q}`, { nominal }, opts);
	}

	function isiSemua(k: UjrohKomponen) {
		for (const g of guruAktif) if (String(penerimaDraft[g.guru_id] ?? "").trim() === "") penerimaDraft[g.guru_id] = k.nominal_default;
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
								<div class="mb-2 flex flex-wrap items-center justify-between gap-2">
									<p class="text-xs text-neutral-500">Kosongkan nominal lalu Simpan untuk menghapus guru dari penerima. Berlaku untuk bulan yang belum dikunci.</p>
									<button onclick={() => isiSemua(k)} class="text-xs font-semibold text-brand-700 hover:underline dark:text-brand-300">Isi kosong dengan default</button>
								</div>
								<div class="divide-y divide-neutral-100 dark:divide-neutral-800/70">
									{#each guruAktif as g (g.guru_id)}
										<div class="flex items-center gap-2 py-1.5">
											<p class="min-w-0 flex-1 truncate text-sm text-neutral-900 dark:text-white">{g.nama}</p>
											<label class="sr-only" for="penerima-{k.id}-{g.guru_id}">Nominal {k.nama} untuk {g.nama}</label>
											<input id="penerima-{k.id}-{g.guru_id}" type="number" min="0" step="1000" placeholder="Tidak menerima" bind:value={penerimaDraft[g.guru_id]} class="{inputClass} w-40" />
											<button onclick={() => simpanPenerima(k, g.guru_id)} disabled={busy} class="rounded-lg border border-neutral-300 px-3 py-2 text-xs font-semibold text-neutral-700 hover:bg-neutral-100 disabled:opacity-50 dark:border-neutral-700 dark:text-neutral-200 dark:hover:bg-neutral-800">Simpan</button>
										</div>
									{/each}
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
