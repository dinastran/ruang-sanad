<script lang="ts">
	import { router } from "@inertiajs/svelte";
	import AppLayout from "@layouts/AppLayout.svelte";
	import StatusBadge from "@components/StatusBadge.svelte";
	import LaporanPager from "@components/keuangan/LaporanPager.svelte";
	import type { Flash, LaporanKeuangan, LaporanSantri, User } from "@lib/types";
	import { ArrowDownCircle, ArrowUpCircle, Download, Info, Printer, Scale, Search, TriangleAlert } from "lucide-svelte";

	interface Props {
		user?: User;
		laporan?: LaporanKeuangan;
		flash?: Flash;
	}

	let props: Props = $props();
	let l = $derived(props.laporan);

	const rupiah = (n: number) => new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 0 }).format(n);
	const tanggal = (t: string) => (t ? new Date(t + "T00:00:00").toLocaleDateString("id-ID", { day: "numeric", month: "short", year: "numeric" }) : "-");
	const inputClass = "rounded-xl border border-neutral-300 bg-neutral-50 px-3 py-2 text-sm text-neutral-900 outline-none focus:border-brand-400 focus:ring-2 focus:ring-brand-400/15 dark:border-neutral-700 dark:bg-neutral-800/60 dark:text-white";

	// --- Periode ---
	// svelte-ignore state_referenced_locally
	let dari = $state(props.laporan?.filter.dari ?? "");
	// svelte-ignore state_referenced_locally
	let sampai = $state(props.laporan?.filter.sampai ?? "");
	let statusSantri = $derived(l?.filter.status === "" ? "semua" : (l?.filter.status ?? "aktif"));
	let angkatanKelas = $derived(l?.filter.angkatan_kelas ?? "");

	function iso(d: Date): string {
		return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
	}

	function query(over: Record<string, string> = {}): Record<string, string> {
		const q: Record<string, string> = { dari, sampai, status: statusSantri, angkatan_kelas: angkatanKelas, ...over };
		for (const k of Object.keys(q)) if (q[k] === "") delete q[k];
		return q;
	}

	function terapkan(over: Record<string, string> = {}) {
		router.get("/app/laporan/keuangan", query(over), { preserveScroll: true });
	}

	function preset(jenis: "bulan_ini" | "bulan_lalu" | "3_bulan" | "tahun_ini") {
		const now = new Date();
		let a: Date, b: Date;
		if (jenis === "bulan_ini") { a = new Date(now.getFullYear(), now.getMonth(), 1); b = new Date(now.getFullYear(), now.getMonth() + 1, 0); }
		else if (jenis === "bulan_lalu") { a = new Date(now.getFullYear(), now.getMonth() - 1, 1); b = new Date(now.getFullYear(), now.getMonth(), 0); }
		else if (jenis === "3_bulan") { a = new Date(now.getFullYear(), now.getMonth() - 2, 1); b = new Date(now.getFullYear(), now.getMonth() + 1, 0); }
		else { a = new Date(now.getFullYear(), 0, 1); b = new Date(now.getFullYear(), 11, 31); }
		dari = iso(a);
		sampai = iso(b);
		terapkan();
	}

	let exportHref = $derived(`/app/laporan/keuangan/export?${new URLSearchParams(query({ dari: l?.filter.dari ?? "", sampai: l?.filter.sampai ?? "" })).toString()}`);
	let labelPeriode = $derived(l ? `${tanggal(l.filter.dari)} – ${tanggal(l.filter.sampai)}` : "");

	// --- Tab & tabel ---
	type Tab = "pembayaran" | "santri" | "pengeluaran" | "tidak_lanjut";
	let tab = $state<Tab>("santri");
	const pageSize = 25;
	let pages = $state<Record<Tab, number>>({ pembayaran: 1, santri: 1, pengeluaran: 1, tidak_lanjut: 1 });
	let cari = $state("");
	let hanyaMenunggak = $state(false);

	const cocok = (...vals: string[]) => {
		const q = cari.trim().toLowerCase();
		return q === "" || vals.some((v) => (v || "").toLowerCase().includes(q));
	};

	let pembayaran = $derived((l?.pembayaran ?? []).filter((p) => cocok(p.santri_nama, p.id_mahasantri, p.metode, p.kelas_nama)));
	let santri = $derived((l?.santri ?? []).filter((s) => cocok(s.nama, s.id_mahasantri, s.kelas_nama) && (!hanyaMenunggak || s.belum_bayar > 0)));
	let pengeluaran = $derived([
		...(l?.ujroh ?? []).map((u) => ({ tanggal: u.dibayar_at, jenis: "Ujroh", nama: u.guru_nama, rincian: `Ujroh ${u.bulan} · pokok ${rupiah(u.pokok)}${u.komponen ? ` + komponen ${rupiah(u.komponen)}` : ""}${u.kasbon ? ` − kasbon ${rupiah(u.kasbon)}` : ""}`, nominal: u.diterima, catatan: u.catatan })),
		...(l?.kasbon ?? []).map((k) => ({ tanggal: k.tanggal, jenis: "Kasbon", nama: k.guru_nama, rincian: "Kasbon dicairkan", nominal: k.nominal, catatan: k.keterangan })),
	].filter((p) => cocok(p.nama, p.jenis, p.catatan)).sort((a, b) => a.tanggal.localeCompare(b.tanggal)));
	let tidakLanjut = $derived((l?.tidak_lanjut ?? []).filter((s) => cocok(s.nama, s.id_mahasantri, s.keterangan_tidak_lanjut)));

	const halaman = <T,>(rows: T[], t: Tab) => rows.slice((pages[t] - 1) * pageSize, pages[t] * pageSize);
	function gantiTab(t: Tab) {
		tab = t;
		cari = "";
		resetHalaman();
	}
	// Pencarian/filter baru selalu mulai dari halaman pertama.
	function resetHalaman() {
		pages = { pembayaran: 1, santri: 1, pengeluaran: 1, tidak_lanjut: 1 };
	}

	let totalSantriTabel = $derived(santri.reduce((a, s) => ({ terbit: a.terbit + s.terbit_periode, dibayar: a.dibayar + s.dibayar_periode, belum: a.belum + s.belum_bayar }), { terbit: 0, dibayar: 0, belum: 0 }));

	function pembayaranTerakhir(s: LaporanSantri): string {
		if (s.pembayaran_terakhir) return `${tanggal(s.pembayaran_terakhir.tanggal)} · bulan ke-${s.pembayaran_terakhir.bulan_ke}`;
		return "";
	}

	const tabs: { key: Tab; label: string; count: () => number }[] = [
		{ key: "santri", label: "Per santri", count: () => l?.santri.length ?? 0 },
		{ key: "pembayaran", label: "Pembayaran SPP", count: () => l?.pembayaran.length ?? 0 },
		{ key: "pengeluaran", label: "Pengeluaran", count: () => (l?.ujroh.length ?? 0) + (l?.kasbon.length ?? 0) },
		{ key: "tidak_lanjut", label: "Tidak lanjut", count: () => l?.tidak_lanjut.length ?? 0 },
	];
</script>

<AppLayout user={props.user}>
	<main class="max-w-7xl mx-auto px-4 sm:px-6 py-8 space-y-6">
		<section class="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
			<div>
				<p class="text-xs font-bold tracking-[0.16em] uppercase text-brand-600 dark:text-brand-400">Keuangan</p>
				<h1 class="mt-2 text-3xl font-bold text-neutral-900 dark:text-white">Laporan Keuangan</h1>
				<p class="mt-2 text-neutral-600 dark:text-neutral-400">Periode {labelPeriode}</p>
			</div>
			<div class="flex flex-wrap items-end gap-2 print:hidden">
				<label class="text-xs font-medium text-neutral-600 dark:text-neutral-400">Dari<input type="date" bind:value={dari} class="{inputClass} mt-1 block" /></label>
				<label class="text-xs font-medium text-neutral-600 dark:text-neutral-400">Sampai<input type="date" bind:value={sampai} class="{inputClass} mt-1 block" /></label>
				<button onclick={() => terapkan()} disabled={!dari || !sampai || sampai < dari} class="rounded-xl bg-brand-600 px-4 py-2 text-sm font-semibold text-white hover:bg-brand-700 disabled:opacity-50">Terapkan</button>
				<a href={exportHref} class="inline-flex items-center gap-2 rounded-xl border border-neutral-300 px-3.5 py-2 text-sm font-semibold text-neutral-700 hover:bg-neutral-100 dark:border-neutral-700 dark:text-neutral-200 dark:hover:bg-neutral-800"><Download size="16" /> Excel</a>
				<button onclick={() => window.print()} class="inline-flex items-center gap-2 rounded-xl border border-neutral-300 px-3.5 py-2 text-sm font-semibold text-neutral-700 hover:bg-neutral-100 dark:border-neutral-700 dark:text-neutral-200 dark:hover:bg-neutral-800"><Printer size="16" /> Cetak</button>
			</div>
		</section>

		<div class="flex flex-wrap gap-2 print:hidden">
			{#each [["bulan_ini", "Bulan ini"], ["bulan_lalu", "Bulan lalu"], ["3_bulan", "3 bulan terakhir"], ["tahun_ini", "Tahun ini"]] as [k, label] (k)}
				<button onclick={() => preset(k as "bulan_ini")} class="rounded-full border border-neutral-300 px-3 py-1 text-xs font-medium text-neutral-700 hover:bg-neutral-100 dark:border-neutral-700 dark:text-neutral-300 dark:hover:bg-neutral-800">{label}</button>
			{/each}
		</div>

		{#if props.flash?.error}
			<div role="alert" class="rounded-2xl border border-error/20 bg-error/10 p-4 text-sm font-medium text-error">{props.flash.error}</div>
		{/if}

		{#if l}
			<div class="flex items-start gap-3 rounded-2xl border border-neutral-200 bg-neutral-50 p-4 text-sm text-neutral-600 dark:border-neutral-800 dark:bg-neutral-900/40 dark:text-neutral-400">
				<Info size="18" class="mt-0.5 shrink-0" />
				<p>Basis kas: SPP dihitung dari <strong>tanggal bayar</strong>, produk dari tanggal order, ujroh dari tanggal dibayar (nilai bersih setelah potong kasbon), dan kasbon dari tanggal diberikan. <strong>Belum termasuk biaya operasional di luar sistem.</strong></p>
			</div>

			<section class="grid grid-cols-2 lg:grid-cols-4 gap-3">
				<div class="rounded-2xl border border-success/20 bg-success/10 p-5">
					<p class="flex items-center gap-1.5 text-xs text-neutral-600 dark:text-neutral-300"><ArrowDownCircle size="14" class="text-success" /> Pemasukan</p>
					<p class="mt-3 text-2xl font-bold text-success font-mono">{rupiah(l.ringkasan.total_pemasukan)}</p>
				</div>
				<div class="rounded-2xl border border-error/20 bg-error/10 p-5">
					<p class="flex items-center gap-1.5 text-xs text-neutral-600 dark:text-neutral-300"><ArrowUpCircle size="14" class="text-error" /> Pengeluaran</p>
					<p class="mt-3 text-2xl font-bold text-error font-mono">{rupiah(l.ringkasan.total_pengeluaran)}</p>
				</div>
				<div class="rounded-2xl bg-neutral-900 p-5 text-white">
					<p class="flex items-center gap-1.5 text-xs text-neutral-400"><Scale size="14" /> Saldo periode</p>
					<p class="mt-3 text-2xl font-bold font-mono {l.ringkasan.saldo < 0 ? 'text-red-300' : ''}">{rupiah(l.ringkasan.saldo)}</p>
				</div>
				<div class="rounded-2xl border border-warning/20 bg-warning/10 p-5">
					<p class="flex items-center gap-1.5 text-xs text-neutral-600 dark:text-neutral-300"><TriangleAlert size="14" class="text-warning" /> Piutang hari ini</p>
					<p class="mt-3 text-2xl font-bold text-warning font-mono">{rupiah(l.piutang.belum_bayar)}</p>
					<p class="mt-1 text-xs text-neutral-600 dark:text-neutral-400">{l.piutang.jumlah_santri} santri · {rupiah(l.piutang.terlambat)} lewat jatuh tempo</p>
				</div>
			</section>

			<section class="grid gap-4 lg:grid-cols-2">
				<div class="rounded-2xl border border-neutral-200 bg-white p-5 dark:border-neutral-800 dark:bg-neutral-925">
					<h2 class="font-semibold text-neutral-900 dark:text-white">Arus kas</h2>
					<dl class="mt-4 space-y-2 text-sm">
						<div class="flex justify-between gap-3"><dt class="text-neutral-600 dark:text-neutral-400">SPP / infaq ({l.ringkasan.jumlah_bayar_spp} pembayaran)</dt><dd class="font-mono">{rupiah(l.ringkasan.pemasukan_spp)}</dd></div>
						<div class="flex justify-between gap-3"><dt class="text-neutral-600 dark:text-neutral-400">Produk ({l.ringkasan.jumlah_order_produk} order)</dt><dd class="font-mono">{rupiah(l.ringkasan.pemasukan_produk)}</dd></div>
						<div class="flex justify-between gap-3 border-t border-neutral-200 pt-2 font-semibold dark:border-neutral-800"><dt>Total pemasukan</dt><dd class="font-mono text-success">{rupiah(l.ringkasan.total_pemasukan)}</dd></div>
						<div class="flex justify-between gap-3 pt-3"><dt class="text-neutral-600 dark:text-neutral-400">Ujroh guru dibayar ({l.ringkasan.jumlah_ujroh})</dt><dd class="font-mono">{rupiah(l.ringkasan.pengeluaran_ujroh)}</dd></div>
						<div class="flex justify-between gap-3"><dt class="text-neutral-600 dark:text-neutral-400">Kasbon dicairkan ({l.ringkasan.jumlah_kasbon})</dt><dd class="font-mono">{rupiah(l.ringkasan.pengeluaran_kasbon)}</dd></div>
						<div class="flex justify-between gap-3 border-t border-neutral-200 pt-2 font-semibold dark:border-neutral-800"><dt>Total pengeluaran</dt><dd class="font-mono text-error">{rupiah(l.ringkasan.total_pengeluaran)}</dd></div>
						<div class="flex justify-between gap-3 border-t-2 border-neutral-900 pt-2 text-base font-bold dark:border-white"><dt>Saldo periode</dt><dd class="font-mono">{rupiah(l.ringkasan.saldo)}</dd></div>
					</dl>
				</div>

				<div class="rounded-2xl border border-neutral-200 bg-white p-5 dark:border-neutral-800 dark:bg-neutral-925">
					<h2 class="font-semibold text-neutral-900 dark:text-white">Tagihan terbit di periode</h2>
					<p class="mt-1 text-xs text-neutral-500">Berdasarkan tanggal tagih; pembayaran bisa terjadi di periode lain.</p>
					<div class="mt-4 flex items-end justify-between gap-3">
						<div>
							<p class="text-2xl font-bold font-mono text-neutral-900 dark:text-white">{rupiah(l.tagihan.nominal)}</p>
							<p class="text-xs text-neutral-500">{l.tagihan.jumlah} tagihan{l.tagihan.jumlah_batal ? ` · ${l.tagihan.jumlah_batal} dibatalkan (tidak dihitung)` : ""}</p>
						</div>
						<p class="text-right text-sm"><span class="font-semibold text-success">{l.tagihan.persen_lunas.toFixed(0)}%</span> lunas<br /><span class="text-xs text-neutral-500">{rupiah(l.tagihan.nominal_lunas)}</span></p>
					</div>
					<div class="mt-3 h-2 overflow-hidden rounded-full bg-neutral-100 dark:bg-neutral-800"><div class="h-full rounded-full bg-success" style={`width: ${Math.min(100, l.tagihan.persen_lunas)}%`}></div></div>

					<div class="mt-5 grid gap-4 sm:grid-cols-2">
						<div>
							<p class="text-xs font-semibold uppercase tracking-wide text-neutral-500">SPP masuk per metode</p>
							<ul class="mt-2 space-y-1 text-sm">
								{#each l.spp_per_metode as g (g.nama)}<li class="flex justify-between gap-2"><span class="truncate text-neutral-600 dark:text-neutral-400">{g.nama} ({g.jumlah})</span><span class="font-mono">{rupiah(g.nominal)}</span></li>{:else}<li class="text-neutral-500">-</li>{/each}
							</ul>
						</div>
						<div>
							<p class="text-xs font-semibold uppercase tracking-wide text-neutral-500">SPP masuk per angkatan kelas</p>
							<ul class="mt-2 space-y-1 text-sm">
								{#each l.spp_per_angkatan as g (g.nama)}<li class="flex justify-between gap-2"><span class="truncate text-neutral-600 dark:text-neutral-400">{g.nama} ({g.jumlah})</span><span class="font-mono">{rupiah(g.nominal)}</span></li>{:else}<li class="text-neutral-500">-</li>{/each}
							</ul>
						</div>
					</div>
				</div>
			</section>

			<section class="rounded-2xl border border-neutral-200 bg-white dark:border-neutral-800 dark:bg-neutral-925">
				<div class="flex flex-col gap-3 border-b border-neutral-200 px-4 pt-3 dark:border-neutral-800 lg:flex-row lg:items-end lg:justify-between">
					<div class="flex gap-1 overflow-x-auto" role="tablist">
						{#each tabs as t (t.key)}
							<button role="tab" aria-selected={tab === t.key} onclick={() => gantiTab(t.key)} class="whitespace-nowrap border-b-2 px-3 py-2.5 text-sm font-semibold transition-colors {tab === t.key ? 'border-brand-500 text-brand-700 dark:text-brand-300' : 'border-transparent text-neutral-500 hover:text-neutral-800 dark:hover:text-neutral-200'}">
								{t.label} <span class="ml-1 rounded-full bg-neutral-100 px-1.5 text-xs dark:bg-neutral-800">{t.count()}</span>
							</button>
						{/each}
					</div>
					<div class="flex flex-wrap items-center gap-2 pb-3 print:hidden">
						{#if tab === "santri"}
							<select value={statusSantri} onchange={(e) => terapkan({ status: e.currentTarget.value })} aria-label="Status santri" class={inputClass}>
								<option value="aktif">Aktif</option>
								<option value="cuti">Cuti</option>
								<option value="nonaktif">Nonaktif</option>
								<option value="tidak_lanjut">Tidak lanjut</option>
								<option value="semua">Semua status</option>
							</select>
							<select value={angkatanKelas} onchange={(e) => terapkan({ angkatan_kelas: e.currentTarget.value })} aria-label="Angkatan kelas" class={inputClass}>
								<option value="">Semua angkatan kelas</option>
								{#each l.opsi_angkatan as a (a)}<option value={a}>Angkatan {a}</option>{/each}
							</select>
							<label class="flex items-center gap-1.5 text-xs text-neutral-600 dark:text-neutral-400"><input type="checkbox" bind:checked={hanyaMenunggak} onchange={resetHalaman} /> Hanya yang menunggak</label>
						{/if}
						<label class="relative">
							<Search size="15" class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-neutral-400" />
							<input bind:value={cari} oninput={resetHalaman} placeholder="Cari nama / ID" aria-label="Cari" class="{inputClass} w-52 pl-9" />
						</label>
					</div>
				</div>

				<div class="overflow-x-auto">
					{#if tab === "santri"}
						<table class="w-full min-w-[980px] text-sm">
							<thead>
								<tr class="border-b border-neutral-200 bg-neutral-50 text-left text-xs font-semibold uppercase tracking-wide text-neutral-500 dark:border-neutral-800 dark:bg-neutral-900/60">
									<th class="px-4 py-3">Santri</th>
									<th class="px-4 py-3">Kelas</th>
									<th class="px-4 py-3 text-right">SPP</th>
									<th class="px-4 py-3 text-right">Terbit periode</th>
									<th class="px-4 py-3 text-right">Dibayar periode</th>
									<th class="px-4 py-3 text-right">Belum dibayar</th>
									<th class="px-4 py-3">Pembayaran terakhir</th>
								</tr>
							</thead>
							<tbody class="divide-y divide-neutral-100 dark:divide-neutral-800/70">
								{#each halaman(santri, "santri") as s (s.santri_id)}
									<tr class="align-top">
										<td class="px-4 py-3">
											<p class="font-medium text-neutral-900 dark:text-white">{s.nama}</p>
											<p class="flex flex-wrap items-center gap-1.5 text-xs text-neutral-500">{s.id_mahasantri || "-"}{#if s.status !== "aktif"}<StatusBadge status={s.status} />{/if}</p>
										</td>
										<td class="px-4 py-3 text-neutral-600 dark:text-neutral-400">{s.kelas_nama || "-"}<span class="block text-xs">Angkatan {s.angkatan_kelas || "-"}</span></td>
										<td class="px-4 py-3 text-right font-mono">{rupiah(s.nominal_spp)}</td>
										<td class="px-4 py-3 text-right font-mono">{s.terbit_periode ? rupiah(s.terbit_periode) : "-"}</td>
										<td class="px-4 py-3 text-right font-mono {s.dibayar_periode ? 'text-success' : ''}">{s.dibayar_periode ? rupiah(s.dibayar_periode) : "-"}</td>
										<td class="px-4 py-3 text-right">
											{#if s.belum_bayar > 0}
												<p class="font-mono font-semibold text-warning">{rupiah(s.belum_bayar)}</p>
												<p class="text-xs text-neutral-500">{s.jumlah_belum_bayar} tagihan{#if s.jumlah_terlambat > 0} · <span class="text-error">{s.jumlah_terlambat} lewat tempo</span>{/if}</p>
											{:else}
												<span class="text-neutral-400">-</span>
											{/if}
										</td>
										<td class="px-4 py-3">
											{#if s.pembayaran_terakhir}
												<p class="text-neutral-900 dark:text-white">{pembayaranTerakhir(s)}</p>
												<p class="text-xs text-neutral-500">{rupiah(s.pembayaran_terakhir.nominal)}{s.pembayaran_terakhir.metode ? ` · ${s.pembayaran_terakhir.metode}` : ""}</p>
											{:else if s.catatan_infaq_lama}
												<p class="text-neutral-700 dark:text-neutral-300">{s.catatan_infaq_lama}</p>
												<p class="text-xs text-neutral-500">catatan lama (sebelum sistem tagihan)</p>
											{:else}
												<span class="text-xs text-neutral-500">Belum ada pembayaran</span>
											{/if}
										</td>
									</tr>
								{:else}
									<tr><td colspan="7" class="px-4 py-10 text-center text-neutral-500">Tidak ada santri sesuai filter.</td></tr>
								{/each}
							</tbody>
							{#if santri.length > 0}
								<tfoot>
									<tr class="border-t border-neutral-200 bg-neutral-50 font-semibold dark:border-neutral-800 dark:bg-neutral-900/60">
										<td class="px-4 py-3" colspan="3">Total {santri.length} santri</td>
										<td class="px-4 py-3 text-right font-mono">{rupiah(totalSantriTabel.terbit)}</td>
										<td class="px-4 py-3 text-right font-mono">{rupiah(totalSantriTabel.dibayar)}</td>
										<td class="px-4 py-3 text-right font-mono">{rupiah(totalSantriTabel.belum)}</td>
										<td></td>
									</tr>
								</tfoot>
							{/if}
						</table>
						<LaporanPager page={pages.santri} total={santri.length} {pageSize} onchange={(p) => (pages.santri = p)} />
					{:else if tab === "pembayaran"}
						<table class="w-full min-w-[860px] text-sm">
							<thead>
								<tr class="border-b border-neutral-200 bg-neutral-50 text-left text-xs font-semibold uppercase tracking-wide text-neutral-500 dark:border-neutral-800 dark:bg-neutral-900/60">
									<th class="px-4 py-3">Tanggal bayar</th>
									<th class="px-4 py-3">Santri</th>
									<th class="px-4 py-3">Kelas</th>
									<th class="px-4 py-3 text-right">Bulan ke</th>
									<th class="px-4 py-3">Metode</th>
									<th class="px-4 py-3 text-right">Nominal</th>
								</tr>
							</thead>
							<tbody class="divide-y divide-neutral-100 dark:divide-neutral-800/70">
								{#each halaman(pembayaran, "pembayaran") as p (p.tagihan_id)}
									<tr class="align-top">
										<td class="px-4 py-3 whitespace-nowrap">{tanggal(p.tanggal_bayar)}</td>
										<td class="px-4 py-3"><p class="font-medium text-neutral-900 dark:text-white">{p.santri_nama}</p><p class="text-xs text-neutral-500">{p.id_mahasantri || "-"}</p></td>
										<td class="px-4 py-3 text-neutral-600 dark:text-neutral-400">{p.kelas_nama || "-"}<span class="block text-xs">Angkatan {p.angkatan_kelas || "-"}</span></td>
										<td class="px-4 py-3 text-right font-mono">{p.bulan_ke}</td>
										<td class="px-4 py-3 text-neutral-600 dark:text-neutral-400">{p.metode || "-"}{#if p.catatan}<span class="block text-xs">{p.catatan}</span>{/if}{#if p.dicatat_oleh}<span class="block text-xs text-neutral-500">dicatat {p.dicatat_oleh}</span>{/if}</td>
										<td class="px-4 py-3 text-right font-mono font-semibold">{rupiah(p.nominal)}</td>
									</tr>
								{:else}
									<tr><td colspan="6" class="px-4 py-10 text-center text-neutral-500">Tidak ada pembayaran SPP di periode ini.</td></tr>
								{/each}
							</tbody>
						</table>
						<LaporanPager page={pages.pembayaran} total={pembayaran.length} {pageSize} onchange={(p) => (pages.pembayaran = p)} />
					{:else if tab === "pengeluaran"}
						<table class="w-full min-w-[760px] text-sm">
							<thead>
								<tr class="border-b border-neutral-200 bg-neutral-50 text-left text-xs font-semibold uppercase tracking-wide text-neutral-500 dark:border-neutral-800 dark:bg-neutral-900/60">
									<th class="px-4 py-3">Tanggal</th>
									<th class="px-4 py-3">Jenis</th>
									<th class="px-4 py-3">Guru</th>
									<th class="px-4 py-3">Rincian</th>
									<th class="px-4 py-3 text-right">Nominal keluar</th>
								</tr>
							</thead>
							<tbody class="divide-y divide-neutral-100 dark:divide-neutral-800/70">
								{#each halaman(pengeluaran, "pengeluaran") as p, i (i)}
									<tr class="align-top">
										<td class="px-4 py-3 whitespace-nowrap">{tanggal(p.tanggal)}</td>
										<td class="px-4 py-3"><span class="rounded-md px-1.5 py-0.5 text-xs font-medium {p.jenis === 'Ujroh' ? 'bg-neutral-100 text-neutral-700 dark:bg-neutral-800 dark:text-neutral-300' : 'bg-warning/10 text-warning'}">{p.jenis}</span></td>
										<td class="px-4 py-3 font-medium text-neutral-900 dark:text-white">{p.nama}</td>
										<td class="px-4 py-3 text-xs text-neutral-600 dark:text-neutral-400">{p.rincian}{#if p.catatan}<span class="block">{p.catatan}</span>{/if}</td>
										<td class="px-4 py-3 text-right font-mono font-semibold">{rupiah(p.nominal)}</td>
									</tr>
								{:else}
									<tr><td colspan="5" class="px-4 py-10 text-center text-neutral-500">Tidak ada pengeluaran tercatat di periode ini.</td></tr>
								{/each}
							</tbody>
						</table>
						<LaporanPager page={pages.pengeluaran} total={pengeluaran.length} {pageSize} onchange={(p) => (pages.pengeluaran = p)} />
					{:else}
						<table class="w-full min-w-[760px] text-sm">
							<thead>
								<tr class="border-b border-neutral-200 bg-neutral-50 text-left text-xs font-semibold uppercase tracking-wide text-neutral-500 dark:border-neutral-800 dark:bg-neutral-900/60">
									<th class="px-4 py-3">Santri</th>
									<th class="px-4 py-3">Angkatan</th>
									<th class="px-4 py-3">Keterangan</th>
									<th class="px-4 py-3 text-right">Belum dibayar</th>
								</tr>
							</thead>
							<tbody class="divide-y divide-neutral-100 dark:divide-neutral-800/70">
								{#each halaman(tidakLanjut, "tidak_lanjut") as s (s.santri_id)}
									<tr class="align-top">
										<td class="px-4 py-3"><p class="font-medium text-neutral-900 dark:text-white">{s.nama}</p><p class="text-xs text-neutral-500">{s.id_mahasantri || "-"}</p></td>
										<td class="px-4 py-3 text-neutral-600 dark:text-neutral-400">Daftar {s.angkatan || "-"}<span class="block text-xs">Kelas {s.angkatan_kelas || "-"}</span></td>
										<td class="px-4 py-3 text-neutral-700 dark:text-neutral-300">{s.keterangan_tidak_lanjut || "-"}</td>
										<td class="px-4 py-3 text-right font-mono {s.belum_bayar ? 'text-warning font-semibold' : 'text-neutral-400'}">{s.belum_bayar ? rupiah(s.belum_bayar) : "-"}</td>
									</tr>
								{:else}
									<tr><td colspan="4" class="px-4 py-10 text-center text-neutral-500">Tidak ada santri tidak lanjut.</td></tr>
								{/each}
							</tbody>
						</table>
						<LaporanPager page={pages.tidak_lanjut} total={tidakLanjut.length} {pageSize} onchange={(p) => (pages.tidak_lanjut = p)} />
					{/if}
				</div>
			</section>
		{/if}
	</main>
</AppLayout>
