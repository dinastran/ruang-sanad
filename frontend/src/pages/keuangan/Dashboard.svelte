<script lang="ts">
	import { inertia, router } from "@inertiajs/svelte";
	import AppLayout from "@layouts/AppLayout.svelte";
	import CashflowChart from "@components/keuangan/CashflowChart.svelte";
	import FollowUpTagihanDialog from "@components/keuangan/FollowUpTagihanDialog.svelte";
	import type { KeuanganDashboard, Tagihan, User, WaTemplate } from "@lib/types";
	import { ArrowDownRight, ArrowRight, ArrowUpRight, ChevronLeft, ChevronRight, Eye, HandCoins, MessageCircle, Package, WalletCards } from "lucide-svelte";

	interface Props { user?: User; dashboard: KeuanganDashboard; templates?: WaTemplate[] }
	let props: Props = $props();
	let d = $derived(props.dashboard);
	let templates = $derived(props.templates ?? []);
	let hasActiveTemplate = $derived(templates.some((t) => t.is_aktif));

	let piutangTab = $state<"kelas" | "angkatan">("kelas");
	let piutangRows = $derived(piutangTab === "kelas" ? d.piutang_kelas : d.piutang_angkatan);
	let piutangMax = $derived(Math.max(1, ...piutangRows.map((r) => r.nominal)));
	let followUpTarget = $state<Tagihan | null>(null);

	const rupiah = (n: number) => new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 0 }).format(n);
	const angka = (n: number) => n.toLocaleString("id-ID");

	function gotoBulan(bulan: string) {
		if (bulan) router.get("/app/keuangan", { bulan }, { preserveScroll: true });
	}

	// Change vs previous month; null when there is nothing to compare against.
	function delta(cur: number, prev: number): number | null {
		if (!prev) return null;
		return ((cur - prev) / Math.abs(prev)) * 100;
	}

	const agingColor: Record<string, string> = {
		belum_tempo: "bg-neutral-300 dark:bg-neutral-600",
		telat_30: "bg-warning",
		telat_60: "bg-orange-500",
		telat_lebih: "bg-error",
	};

	let kartu = $derived([
		{ label: "Kas masuk", hint: `SPP ${rupiah(d.sekarang.kas_masuk_spp)} · produk ${rupiah(d.sekarang.kas_masuk_produk)}`, value: rupiah(d.sekarang.kas_masuk), change: delta(d.sekarang.kas_masuk, d.sebelumnya.kas_masuk), naikBaik: true, tone: "text-success" },
		{ label: "Kas keluar", hint: `${angka(d.sekarang.jumlah_ujroh_dibayar)} ujroh guru dibayar`, value: rupiah(d.sekarang.kas_keluar), change: delta(d.sekarang.kas_keluar, d.sebelumnya.kas_keluar), naikBaik: false, tone: "text-error" },
		{ label: "Kas bersih", hint: "Kas masuk − kas keluar", value: rupiah(d.sekarang.kas_bersih), change: delta(d.sekarang.kas_bersih, d.sebelumnya.kas_bersih), naikBaik: true, tone: d.sekarang.kas_bersih < 0 ? "text-error" : "text-brand-600 dark:text-brand-400" },
		{ label: "Kolektibilitas", hint: `${rupiah(d.sekarang.terbit_lunas_nominal)} dari ${rupiah(d.sekarang.terbit_nominal)} terbit`, value: `${d.sekarang.kolektibilitas.toFixed(1)}%`, change: d.sebelumnya.terbit_nominal ? d.sekarang.kolektibilitas - d.sebelumnya.kolektibilitas : null, naikBaik: true, tone: "text-neutral-900 dark:text-white", poin: true },
	]);
</script>

<AppLayout user={props.user}>
	<main class="max-w-7xl mx-auto px-4 sm:px-6 py-8 space-y-6">
		<header class="flex flex-col lg:flex-row lg:items-end justify-between gap-4">
			<div>
				<p class="text-xs font-bold tracking-[0.16em] uppercase text-brand-600 dark:text-brand-400">Keuangan</p>
				<h1 class="mt-2 text-3xl font-bold text-neutral-900 dark:text-white">Dashboard keuangan</h1>
				<p class="mt-2 text-sm text-neutral-600 dark:text-neutral-400">Arus kas, penagihan SPP, dan ujroh guru dalam satu tempat.</p>
			</div>
			<div class="flex flex-wrap items-center gap-2">
				<div class="inline-flex items-center rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-925">
					<button onclick={() => gotoBulan(d.bulan_sebelum)} class="p-2.5 text-neutral-500 hover:text-neutral-900 dark:hover:text-white" title="Bulan sebelumnya"><ChevronLeft size="18" /></button>
					<input type="month" value={d.bulan} onchange={(e) => gotoBulan(e.currentTarget.value)} aria-label="Pilih bulan" class="bg-transparent px-1 py-2 text-sm font-semibold text-neutral-900 dark:text-white outline-none" />
					<button onclick={() => gotoBulan(d.bulan_berikut)} disabled={!d.bulan_berikut} class="p-2.5 text-neutral-500 hover:text-neutral-900 dark:hover:text-white disabled:opacity-30" title="Bulan berikutnya"><ChevronRight size="18" /></button>
				</div>
				<a href="/app/keuangan/ujroh?bulan={d.bulan}" use:inertia class="inline-flex items-center gap-2 px-4 py-2.5 rounded-xl border border-neutral-300 dark:border-neutral-700 text-neutral-700 dark:text-neutral-200 font-semibold text-sm"><HandCoins size="16" /> Ujroh</a>
				<a href="/app/keuangan/tagihan" use:inertia class="inline-flex items-center gap-2 px-4 py-2.5 rounded-xl bg-brand-600 hover:bg-brand-700 text-white font-semibold text-sm"><WalletCards size="16" /> Kelola tagihan</a>
			</div>
		</header>

		<section>
			<p class="mb-3 text-xs font-semibold uppercase tracking-wider text-neutral-500">{d.bulan_label} · dibanding bulan lalu</p>
			<div class="grid grid-cols-2 lg:grid-cols-4 gap-3">
				{#each kartu as k}
					<div class="rounded-2xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-925 p-5">
						<div class="flex items-start justify-between gap-2">
							<p class="text-xs text-neutral-600 dark:text-neutral-300">{k.label}</p>
							{#if k.change !== null}
								{@const baik = k.change === 0 ? null : (k.change > 0) === k.naikBaik}
								<span class="inline-flex items-center gap-0.5 rounded-full px-1.5 py-0.5 text-[11px] font-semibold {baik === null ? 'bg-neutral-100 text-neutral-500 dark:bg-neutral-800' : baik ? 'bg-success/10 text-success' : 'bg-error/10 text-error'}">
									{#if k.change >= 0}<ArrowUpRight size="12" />{:else}<ArrowDownRight size="12" />{/if}
									{Math.abs(k.change).toFixed(1)}{k.poin ? " poin" : "%"}
								</span>
							{/if}
						</div>
						<p class="mt-3 text-xl sm:text-2xl font-bold font-mono tracking-tight {k.tone}">{k.value}</p>
						<p class="mt-1 text-xs text-neutral-500">{k.hint}</p>
					</div>
				{/each}
			</div>
		</section>

		<section class="grid lg:grid-cols-3 gap-3">
			<div class="rounded-2xl border border-warning/25 bg-warning/5 p-5">
				<p class="text-xs text-neutral-600 dark:text-neutral-300">Total piutang SPP · per hari ini</p>
				<p class="mt-3 text-2xl font-bold font-mono text-warning">{rupiah(d.total_piutang)}</p>
				<p class="mt-1 text-xs text-neutral-500">{angka(d.jumlah_piutang)} tagihan belum bayar · <span class={d.jumlah_terlambat ? "text-error font-semibold" : ""}>{angka(d.jumlah_terlambat)} terlambat</span></p>
			</div>
			<div class="rounded-2xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-925 p-5">
				<p class="text-xs text-neutral-600 dark:text-neutral-300">Kewajiban ujroh belum dibayar</p>
				<p class="mt-3 text-2xl font-bold font-mono text-neutral-900 dark:text-white">{rupiah(d.ujroh.kewajiban_terkunci + d.ujroh.estimasi_berjalan)}</p>
				<p class="mt-1 text-xs text-neutral-500">{rupiah(d.ujroh.kewajiban_terkunci)} terkunci ({d.ujroh.kewajiban_terkunci_guru} guru){#if d.ujroh.estimasi_berjalan} + {rupiah(d.ujroh.estimasi_berjalan)} estimasi bulan berjalan{/if}</p>
			</div>
			<div class="rounded-2xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-925 p-5">
				<p class="text-xs text-neutral-600 dark:text-neutral-300">Tagihan terbit {d.bulan_label}</p>
				<p class="mt-3 text-2xl font-bold font-mono text-neutral-900 dark:text-white">{rupiah(d.sekarang.terbit_nominal)}</p>
				<p class="mt-1 text-xs text-neutral-500">{angka(d.sekarang.terbit_jumlah)} tagihan · sisa {rupiah(d.sekarang.terbit_nominal - d.sekarang.terbit_lunas_nominal)} belum lunas</p>
			</div>
		</section>

		<section class="grid lg:grid-cols-3 gap-4">
			<div class="lg:col-span-2 rounded-2xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-925 p-5">
				<div class="flex items-center justify-between gap-3"><h2 class="font-semibold text-neutral-900 dark:text-white">Arus kas 6 bulan</h2><p class="text-xs text-neutral-500">Klik bulan untuk membuka</p></div>
				<div class="mt-4"><CashflowChart items={d.tren} selected={d.bulan} onselect={gotoBulan} /></div>
			</div>
			<div class="rounded-2xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-925 p-5">
				<h2 class="font-semibold text-neutral-900 dark:text-white">Kolektibilitas per bulan</h2>
				<p class="mt-1 text-xs text-neutral-500">% nominal tagihan terbit yang sudah lunas</p>
				<div class="mt-4 space-y-3">
					{#each d.tren as b}
						<button type="button" onclick={() => gotoBulan(b.bulan)} class="group block w-full text-left">
							<div class="mb-1 flex justify-between text-xs"><span class="{b.bulan === d.bulan ? 'font-semibold text-brand-600 dark:text-brand-400' : 'text-neutral-600 dark:text-neutral-300'}">{b.label}</span><span class="font-mono text-neutral-500">{b.terbit_nominal ? `${b.kolektibilitas.toFixed(0)}%` : "–"}</span></div>
							<div class="h-2 overflow-hidden rounded-full bg-neutral-100 dark:bg-neutral-800"><div class="h-full rounded-full bg-brand-500 group-hover:bg-brand-600" style={`width: ${Math.min(b.kolektibilitas, 100)}%`}></div></div>
						</button>
					{/each}
				</div>
			</div>
		</section>

		<section class="rounded-2xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-925 p-5">
			<div class="flex flex-wrap items-center justify-between gap-3">
				<div><h2 class="font-semibold text-neutral-900 dark:text-white">Umur piutang</h2><p class="text-xs text-neutral-500">Dihitung dari jatuh tempo, per {d.hari_ini}</p></div>
				<a href="/app/keuangan/tagihan?status=terlambat" use:inertia class="inline-flex items-center gap-1 text-sm font-semibold text-brand-600 dark:text-brand-400">Lihat tagihan terlambat <ArrowRight size="15" /></a>
			</div>
			{#if d.total_piutang > 0}
				<div class="mt-4 flex h-3 overflow-hidden rounded-full bg-neutral-100 dark:bg-neutral-800">
					{#each d.aging as a}{#if a.nominal > 0}<div class={agingColor[a.key]} style={`width: ${(a.nominal / d.total_piutang) * 100}%`} title={`${a.label}: ${rupiah(a.nominal)}`}></div>{/if}{/each}
				</div>
			{/if}
			<div class="mt-4 grid grid-cols-2 lg:grid-cols-4 gap-3">
				{#each d.aging as a}
					<div class="rounded-xl bg-neutral-50 dark:bg-white/[0.03] p-4">
						<p class="flex items-center gap-2 text-xs text-neutral-600 dark:text-neutral-300"><span class="h-2.5 w-2.5 rounded-full {agingColor[a.key]}"></span>{a.label}</p>
						<p class="mt-2 font-mono font-bold text-neutral-900 dark:text-white">{rupiah(a.nominal)}</p>
						<p class="text-xs text-neutral-500">{angka(a.jumlah)} tagihan{#if d.total_piutang} · {((a.nominal / d.total_piutang) * 100).toFixed(0)}%{/if}</p>
					</div>
				{/each}
			</div>
		</section>

		<section class="grid lg:grid-cols-5 gap-4">
			<div class="lg:col-span-3 rounded-2xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-925 overflow-hidden">
				<div class="px-5 py-4 border-b border-neutral-200 dark:border-neutral-800"><h2 class="font-semibold text-neutral-900 dark:text-white">Prioritas follow-up</h2><p class="text-xs text-neutral-500">Telat terlama, nominal terbesar, lalu yang paling lama tidak di-FU</p></div>
				<div class="divide-y divide-neutral-100 dark:divide-neutral-800/70">
					{#each d.prioritas as t (t.id)}
						<div class="flex items-center gap-3 px-5 py-3">
							<div class="w-14 shrink-0 text-center"><p class="font-mono text-lg font-bold {t.hari_telat > 60 ? 'text-error' : t.hari_telat > 30 ? 'text-orange-500' : 'text-warning'}">{t.hari_telat}</p><p class="text-[10px] uppercase tracking-wide text-neutral-500">hari</p></div>
							<div class="min-w-0 flex-1">
								<p class="truncate font-semibold text-neutral-900 dark:text-white">{t.santri_nama}</p>
								<p class="truncate text-xs text-neutral-500">{t.kelas_nama || "Tanpa kelas"} · Bulan ke-{t.bulan_ke} · {t.fu_count ? `FU ${t.fu_count}x, terakhir ${t.fu_terakhir.slice(0, 10)}` : "Belum pernah FU"}</p>
							</div>
							<p class="hidden sm:block font-mono text-sm text-neutral-700 dark:text-neutral-300">{rupiah(t.nominal)}</p>
							<div class="flex shrink-0 gap-1">
								<a href={`/app/keuangan/tagihan/${t.id}`} use:inertia title="Detail" class="p-2 text-neutral-500 hover:bg-neutral-100 dark:hover:bg-neutral-800 rounded-lg"><Eye size="17" /></a>
								<button onclick={() => (followUpTarget = t)} disabled={!t.no_wa || !hasActiveTemplate} title={t.no_wa ? "Follow-up WhatsApp" : "Nomor WA belum diisi"} class="p-2 text-brand-600 hover:bg-brand-500/10 rounded-lg disabled:opacity-30"><MessageCircle size="17" /></button>
							</div>
						</div>
					{:else}
						<div class="p-12 text-center text-sm text-neutral-500">Tidak ada tagihan terlambat. Alhamdulillah.</div>
					{/each}
				</div>
			</div>

			<div class="lg:col-span-2 rounded-2xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-925 p-5">
				<div class="flex items-center justify-between gap-3">
					<h2 class="font-semibold text-neutral-900 dark:text-white">Piutang terbesar</h2>
					<div class="flex rounded-lg bg-neutral-100 dark:bg-neutral-800 p-0.5 text-xs font-semibold">
						<button onclick={() => (piutangTab = "kelas")} class="px-2.5 py-1 rounded-md {piutangTab === 'kelas' ? 'bg-white dark:bg-neutral-925 text-neutral-900 dark:text-white shadow-sm' : 'text-neutral-500'}">Kelas</button>
						<button onclick={() => (piutangTab = "angkatan")} class="px-2.5 py-1 rounded-md {piutangTab === 'angkatan' ? 'bg-white dark:bg-neutral-925 text-neutral-900 dark:text-white shadow-sm' : 'text-neutral-500'}">Angkatan</button>
					</div>
				</div>
				<div class="mt-4 space-y-3">
					{#each piutangRows as r}
						{#snippet bar()}
							<div class="mb-1 flex items-center justify-between gap-3 text-sm"><span class="truncate font-medium text-neutral-700 dark:text-neutral-300 group-hover:text-brand-600">{r.label}</span><span class="shrink-0 font-mono text-xs text-neutral-500">{rupiah(r.nominal)}</span></div>
							<div class="h-2 overflow-hidden rounded-full bg-neutral-100 dark:bg-neutral-800"><div class="h-full rounded-full bg-warning" style={`width: ${(r.nominal / piutangMax) * 100}%`}></div></div>
							<p class="mt-1 text-[11px] text-neutral-500">{angka(r.jumlah_santri)} santri · {angka(r.jumlah_tagihan)} tagihan</p>
						{/snippet}
						{#if r.href}<a href={r.href} use:inertia class="group block">{@render bar()}</a>{:else}<div class="group block">{@render bar()}</div>{/if}
					{:else}
						<p class="py-8 text-center text-sm text-neutral-500">Tidak ada piutang.</p>
					{/each}
				</div>
			</div>
		</section>

		<section class="grid md:grid-cols-2 gap-4">
			<div class="rounded-2xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-925 p-5">
				<div class="flex items-center justify-between gap-3">
					<h2 class="flex items-center gap-2 font-semibold text-neutral-900 dark:text-white"><HandCoins size="17" /> Ujroh {d.bulan_label}</h2>
					<span class="rounded-full px-2 py-0.5 text-xs font-semibold {d.ujroh.terkunci ? 'bg-success/10 text-success' : 'bg-warning/10 text-warning'}">{d.ujroh.terkunci ? "Terkunci" : "Belum dikunci"}</span>
				</div>
				<p class="mt-4 text-2xl font-bold font-mono text-neutral-900 dark:text-white">{rupiah(d.ujroh.total_ujroh)}</p>
				<p class="text-xs text-neutral-500">{d.ujroh.terkunci ? "Total terkunci" : "Estimasi dari pertemuan selesai"} · {d.ujroh.jumlah_guru} guru</p>
				<div class="mt-4 h-2 overflow-hidden rounded-full bg-neutral-100 dark:bg-neutral-800"><div class="h-full rounded-full bg-success" style={`width: ${d.ujroh.total_ujroh ? (d.ujroh.total_dibayar / d.ujroh.total_ujroh) * 100 : 0}%`}></div></div>
				<div class="mt-2 flex justify-between text-xs text-neutral-600 dark:text-neutral-400"><span>Dibayar {rupiah(d.ujroh.total_dibayar)} ({d.ujroh.guru_dibayar}/{d.ujroh.jumlah_guru} guru)</span><span>Sisa {rupiah(d.ujroh.total_belum)}</span></div>
				<a href="/app/keuangan/ujroh?bulan={d.bulan}" use:inertia class="mt-4 inline-flex items-center gap-1 text-sm font-semibold text-brand-600 dark:text-brand-400">Buka rekap ujroh <ArrowRight size="15" /></a>
			</div>
			<div class="rounded-2xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-925 p-5">
				<h2 class="flex items-center gap-2 font-semibold text-neutral-900 dark:text-white"><Package size="17" /> Penjualan produk {d.bulan_label}</h2>
				<div class="mt-4 grid grid-cols-3 gap-3">
					<div><p class="text-2xl font-bold font-mono text-neutral-900 dark:text-white">{angka(d.produk.jumlah_order)}</p><p class="text-xs text-neutral-500">order aktif</p></div>
					<div><p class="text-2xl font-bold font-mono text-neutral-900 dark:text-white">{angka(d.produk.jumlah_qty)}</p><p class="text-xs text-neutral-500">qty terjual</p></div>
					<div><p class="text-2xl font-bold font-mono text-neutral-900 dark:text-white">{rupiah(d.produk.omzet)}</p><p class="text-xs text-neutral-500">omzet · masuk kas</p></div>
				</div>
				{#if d.produk.jumlah_qty > 0 && d.produk.omzet === 0}<p class="mt-4 rounded-xl bg-neutral-50 dark:bg-white/[0.03] p-3 text-xs text-neutral-500">Order bulan ini belum memiliki harga (order dibuat sebelum fitur harga), sehingga omzetnya Rp0.</p>{/if}
				{#if props.user?.role === "admin_kelas" || props.user?.role === "super_admin"}<a href="/app/produk-crm/orders" use:inertia class="mt-4 inline-flex items-center gap-1 text-sm font-semibold text-brand-600 dark:text-brand-400">Buka order produk <ArrowRight size="15" /></a>{/if}
			</div>
		</section>
	</main>

	{#if followUpTarget}
		<FollowUpTagihanDialog target={followUpTarget} {templates} onclose={() => (followUpTarget = null)} onsent={() => router.reload({ only: ["dashboard"] })} />
	{/if}
</AppLayout>
