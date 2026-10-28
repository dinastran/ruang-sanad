<script lang="ts">
	import { inertia } from "@inertiajs/svelte";
	import { ArrowDownRight, ArrowUpRight } from "lucide-svelte";
	import OverviewChart from "@components/OverviewChart.svelte";
	import type { Overview } from "@lib/overview";
	import { change, count, focusRing, kicker, money, moneyShort, panel, percent, signed } from "./ui";

	let { finance, bulan, monthLabel, primary }: {
		finance: Overview["finance"];
		bulan: string;
		monthLabel: string;
		/** Finance-only roles: this panel is the whole dashboard. */
		primary: boolean;
	} = $props();

	let now = $derived(finance?.sekarang);
	let before = $derived(finance?.sebelumnya);
	let hasIssued = $derived((now?.terbit_nominal ?? 0) > 0);
	let kpis = $derived(
		finance && now && before
			? [
					{
						label: "Penerimaan tercatat",
						value: money(now.kas_masuk),
						delta: change(now.kas_masuk, before.kas_masuk),
						deltaText: null as string | null,
						note: `SPP ${moneyShort(now.kas_masuk_spp)} · produk ${moneyShort(now.kas_masuk_produk)}`,
						href: `/app/keuangan?bulan=${bulan}`,
					},
					{
						label: "Tagihan terbit yang lunas",
						value: hasIssued ? percent(now.kolektibilitas) : "Belum ada tagihan",
						delta: null,
						deltaText: hasIssued && before.terbit_nominal > 0 ? `${signed(now.kolektibilitas - before.kolektibilitas, " poin")} dari bulan lalu` : null,
						deltaSign: now.kolektibilitas - before.kolektibilitas,
						note: `${moneyShort(now.terbit_lunas_nominal)} dari ${moneyShort(now.terbit_nominal)} terbit`,
						href: `/app/keuangan?bulan=${bulan}`,
					},
					{
						label: "Piutang saat ini",
						value: money(finance.total_piutang),
						delta: null,
						deltaText: null,
						note: `${count(finance.jumlah_piutang)} tagihan, ${count(finance.jumlah_terlambat)} lewat jatuh tempo`,
						warn: finance.jumlah_terlambat > 0,
						href: "/app/keuangan/tagihan",
					},
					{
						label: "Penerimaan − ujroh dibayar",
						value: money(now.kas_bersih),
						delta: null,
						deltaText: null,
						note: "Belum mencakup biaya lain; bukan laba",
						href: `/app/keuangan?bulan=${bulan}`,
					},
				]
			: [],
	);
	let overdue = $derived(finance ? finance.aging.filter((b) => b.key !== "belum_tempo").reduce((s, b) => s + b.nominal, 0) : 0);
</script>

<section class="min-w-0 space-y-4" aria-labelledby="keuangan-title">
	<div class="flex flex-wrap items-end justify-between gap-3">
		<div>
			<h2 id="keuangan-title" class={primary ? "text-lg font-semibold tracking-[-0.01em] text-neutral-950 dark:text-white" : kicker}>Keuangan yang tercatat</h2>
			<p class="mt-1 text-sm text-neutral-600 dark:text-neutral-300">Aktivitas {monthLabel}. Piutang adalah kondisi saat ini.</p>
		</div>
		<a href={`/app/keuangan?bulan=${bulan}`} use:inertia class={"rounded text-xs font-medium text-brand-700 hover:underline dark:text-brand-400 " + focusRing}>Dashboard keuangan</a>
	</div>

	{#if !finance}
		<div class={panel + " px-5 py-10 text-sm text-neutral-500 sm:px-6 dark:text-neutral-400"}>Keuangan belum tersedia. Coba perbarui data atau buka menu Keuangan.</div>
	{:else}
		<div class="grid grid-cols-1 gap-px overflow-hidden rounded-xl border border-neutral-200/80 bg-neutral-100 sm:grid-cols-2 xl:grid-cols-4 dark:border-white/[0.07] dark:bg-white/[0.05]">
			{#each kpis as k, i (k.label)}
				<a href={k.href} use:inertia class={"block min-w-0 bg-white px-5 py-4 transition-colors duration-150 hover:bg-neutral-50 sm:px-6 dark:bg-neutral-925 dark:hover:bg-neutral-900 " + focusRing + " focus-visible:ring-inset"}>
					<span class="block text-xs font-medium text-neutral-600 dark:text-neutral-300">{k.label}</span>
					<span class={"mt-1 block font-semibold tracking-[-0.02em] tabular-nums " + (i === 0 ? "text-[28px] leading-9 text-neutral-950 dark:text-white" : "text-[22px] leading-8 " + ("warn" in k && k.warn ? "text-amber-700 dark:text-amber-400" : "text-neutral-950 dark:text-white"))}>{k.value}</span>
					{#if k.delta !== null}
						<span class={"mt-1 inline-flex items-center gap-0.5 text-xs font-medium tabular-nums " + (k.delta >= 0 ? "text-emerald-700 dark:text-emerald-400" : "text-red-700 dark:text-red-400")}>
							{#if k.delta > 0}<ArrowUpRight size={12} />{:else if k.delta < 0}<ArrowDownRight size={12} />{/if}{signed(k.delta)} dari bulan lalu
						</span>
					{:else if k.deltaText}
						<span class={"mt-1 block text-xs font-medium tabular-nums " + (("deltaSign" in k ? k.deltaSign ?? 0 : 0) >= 0 ? "text-emerald-700 dark:text-emerald-400" : "text-red-700 dark:text-red-400")}>{k.deltaText}</span>
					{/if}
					<span class="mt-1 block text-xs text-neutral-500 dark:text-neutral-400">{k.note}</span>
				</a>
			{/each}
		</div>

		<div class="grid grid-cols-[minmax(0,1fr)] gap-4 lg:grid-cols-[minmax(0,1.4fr)_minmax(0,1fr)]">
			<div class={panel + " min-w-0 px-5 py-5 sm:px-6"}>
				<h3 class="text-sm font-semibold text-neutral-900 dark:text-white">Penerimaan dan ujroh dibayar</h3>
				<p class="mt-0.5 text-xs text-neutral-500 dark:text-neutral-400">Enam bulan, berdasarkan tanggal transaksi.</p>
				<OverviewChart
					title="Tren penerimaan dan ujroh dibayar"
					money
					labels={finance.tren.map((t) => t.label)}
					series={[
						{ name: "Penerimaan", values: finance.tren.map((t) => t.kas_masuk) },
						{ name: "Ujroh dibayar", values: finance.tren.map((t) => t.kas_keluar) },
					]}
				/>
			</div>
			<div class={panel + " min-w-0 px-5 py-5 sm:px-6"}>
				<h3 class="text-sm font-semibold text-neutral-900 dark:text-white">Umur piutang</h3>
				<p class="mt-0.5 text-xs text-neutral-500 dark:text-neutral-400">
					{finance.total_piutang > 0 ? `${percent((overdue / finance.total_piutang) * 100)} dari piutang sudah lewat jatuh tempo.` : "Tidak ada piutang tercatat."}
				</p>
				<ul class="mt-4 space-y-3.5">
					{#each finance.aging as b (b.key)}
						{@const share = finance.total_piutang > 0 ? (b.nominal / finance.total_piutang) * 100 : 0}
						<li>
							<div class="flex items-baseline justify-between gap-3 text-sm">
								<span class="text-neutral-700 dark:text-neutral-300">{b.label}</span>
								<span class="font-semibold tabular-nums text-neutral-900 dark:text-white">{money(b.nominal)}</span>
							</div>
							<div class="mt-1.5 h-1.5 overflow-hidden rounded-full bg-neutral-100 dark:bg-white/[0.06]">
								<div class={"h-full rounded-full " + (b.key === "belum_tempo" ? "bg-neutral-400 dark:bg-neutral-500" : b.key === "telat_lebih" ? "bg-red-500" : "bg-amber-500")} style:width={`${share}%`}></div>
							</div>
							<span class="mt-1 block text-[11px] tabular-nums text-neutral-500 dark:text-neutral-400">{count(b.jumlah)} tagihan</span>
						</li>
					{/each}
				</ul>
			</div>
		</div>

		<dl class={panel + " grid grid-cols-2 divide-neutral-100 sm:grid-cols-3 lg:grid-cols-5 dark:divide-white/[0.05]"}>
			{#each [
				{ label: "SPP diterima", value: now?.kas_masuk_spp },
				{ label: "Penjualan produk", value: now?.kas_masuk_produk },
				{ label: "Ujroh dibayar", value: now?.kas_keluar },
				{ label: "Kewajiban ujroh terkunci", value: finance.ujroh.kewajiban_terkunci },
				{ label: "Estimasi ujroh berjalan", value: finance.ujroh.estimasi_berjalan },
			] as row (row.label)}
				<div class="min-w-0 px-5 py-3 sm:px-6">
					<dt class="text-xs text-neutral-500 dark:text-neutral-400">{row.label}</dt>
					<dd class="mt-0.5 text-sm font-semibold tabular-nums text-neutral-900 dark:text-white">{money(row.value)}</dd>
				</div>
			{/each}
		</dl>
	{/if}
</section>
