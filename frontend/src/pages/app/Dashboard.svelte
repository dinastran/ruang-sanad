<script lang="ts">
	import { router } from "@inertiajs/svelte";
	import AppLayout from "@layouts/AppLayout.svelte";
	import type { User } from "@lib/types";
	import type { Overview } from "@lib/overview";
	import InstitutionChain, { type ChainLink } from "@components/overview/InstitutionChain.svelte";
	import PriorityList from "@components/overview/PriorityList.svelte";
	import DecisionNotes, { type Note } from "@components/overview/DecisionNotes.svelte";
	import EnrollmentPanel from "@components/overview/EnrollmentPanel.svelte";
	import AttendancePanel from "@components/overview/AttendancePanel.svelte";
	import ProgramTable from "@components/overview/ProgramTable.svelte";
	import OperationsLedger from "@components/overview/OperationsLedger.svelte";
	import FinancePanel from "@components/overview/FinancePanel.svelte";
	import MonthControl from "@components/overview/MonthControl.svelte";
	import { attendanceRate, count, money, moneyShort, percent } from "@components/overview/ui";

	let { user, overview }: { user?: User; overview?: Overview } = $props();
	let loading = $state(false);
	let loadError = $state("");
	let data = $derived(overview);
	let management = $derived(user?.role === "super_admin");
	let financeAllowed = $derived(["super_admin", "keuangan", "admin"].includes(user?.role ?? ""));
	let finance = $derived(financeAllowed ? data?.finance : undefined);
	let summary = $derived(management ? data?.santri : undefined);
	let ops = $derived(management ? data?.operations : undefined);
	let riayah = $derived(management ? data?.riayah : undefined);
	let products = $derived(management ? data?.products : undefined);
	let attendance = $derived(data?.attendance ?? []);
	let sourceErrors = $derived(data?.errors ?? []);
	let bulan = $derived(data?.bulan ?? "");
	let monthLabel = $derived(data?.bulan_label ?? "periode terpilih");
	let maxMonth = $derived(data?.updated_at?.slice(0, 7) ?? "");

	let seats = $derived((data?.programs ?? []).reduce((t, p) => ({ seats: t.seats + p.seats, occupied: t.occupied + p.occupied }), { seats: 0, occupied: 0 }));
	let latestRate = $derived(attendanceRate(attendance.at(-1)));
	let previousRate = $derived(attendanceRate(attendance.at(-2)));

	const joinNonZero = (parts: [number | undefined, string][]) =>
		parts
			.filter(([n]) => (n ?? 0) > 0)
			.map(([n, label]) => `${count(n)} ${label}`)
			.join(" · ");

	let chain = $derived<ChainLink[]>([
		{
			key: "santri",
			label: "Santri",
			value: count(summary?.aktif),
			caption: summary ? `aktif dari ${count(summary.total)} tercatat · ${count(summary.cuti)} cuti` : "Status santri belum tersedia",
			attention: joinNonZero([
				[riayah?.perlu_perhatian, "perlu perhatian riayah"],
				[ops?.without_class, "belum punya kelas"],
			]),
			available: !!summary,
			href: riayah && riayah.perlu_perhatian > 0 ? "/app/guru/riayah" : "/app/analitik-santri",
		},
		{
			key: "kelas",
			label: "Kelas",
			value: count(ops?.classes),
			caption: seats.seats > 0 ? `kelas aktif · kursi terisi ${percent((seats.occupied / seats.seats) * 100)}` : "kelas aktif",
			attention: joinNonZero([[ops?.without_teacher, "kelas tanpa guru aktif"]]),
			available: !!ops,
			href: "/app/kelas",
		},
		{
			key: "guru",
			label: "Guru",
			value: count(ops?.teachers),
			caption: "guru aktif mengajar",
			attention: joinNonZero([
				[ops?.follow_ups, "tindak lanjut jatuh tempo"],
				[ops?.todos, "tugas koordinator jatuh tempo"],
			]),
			available: !!ops,
			href: "/app/koordinator-guru",
		},
		{
			key: "pertemuan",
			label: "Pertemuan",
			value: count(ops?.finished),
			caption: `selesai ${monthLabel}${latestRate !== null ? ` · hadir ${percent(latestRate)}` : ""}`,
			attention: joinNonZero([[ops?.open_meetings, "pertemuan lama belum ditutup"]]),
			available: !!ops,
			href: "/app/koordinator-guru",
		},
		{
			key: "keuangan",
			label: "Keuangan",
			value: moneyShort(finance?.sekarang.kas_masuk),
			caption: `penerimaan ${monthLabel}`,
			attention: joinNonZero([[finance?.jumlah_terlambat, "tagihan lewat jatuh tempo"]]),
			available: !!finance,
			href: `/app/keuangan?bulan=${bulan}`,
		},
	]);

	let notes = $derived.by<Note[]>(() => {
		const out: Note[] = [];
		if (summary && summary.total > 0) {
			out.push({
				topic: "Status santri",
				text: `${percent(summary.active_rate)} dari seluruh santri tercatat berstatus aktif.${summary.cuti > 0 ? ` ${count(summary.cuti)} santri cuti perlu dipantau rencana kembalinya.` : ""}`,
				caveat: "Proporsi status saat ini, bukan angka retensi.",
			});
		}
		const growth = data?.growth ?? [];
		const reg = growth.at(-1)?.total;
		const prevReg = growth.at(-2)?.total;
		if (management && reg !== undefined && prevReg !== undefined) {
			out.push({
				topic: "Pendaftaran",
				text: `${count(reg)} santri terdaftar ${monthLabel}, ${reg === prevReg ? "sama dengan" : `${reg > prevReg ? "naik" : "turun"} ${count(Math.abs(reg - prevReg))} dari`} bulan sebelumnya.`,
				caveat: data?.current_month ? "Bulan berjalan belum selesai; pembanding adalah satu bulan penuh." : "Berdasarkan tanggal daftar, bukan tanggal pembayaran.",
			});
		}
		if (management && latestRate !== null) {
			out.push({
				topic: "Kehadiran",
				text: `${percent(latestRate)} absensi tercatat hadir${previousRate !== null ? `, ${latestRate >= previousRate ? "naik" : "turun"} ${percent(Math.abs(latestRate - previousRate)).replace("%", "")} poin dari bulan lalu` : ""}.`,
				caveat: "Hanya pertemuan berstatus selesai yang dihitung.",
			});
		}
		if (finance) {
			out.push({
				topic: "Arus kas tercatat",
				text: `Selisih ${finance.sekarang.kas_bersih >= 0 ? "positif" : "negatif"} ${money(Math.abs(finance.sekarang.kas_bersih))} setelah penerimaan dikurangi ujroh dibayar.`,
				caveat: "Belum mencakup seluruh biaya lembaga; ini bukan laba bersih.",
			});
		}
		return out;
	});

	let leadsHref = $derived.by(() => {
		if (!bulan) return "/app/leads/dashboard";
		const [y, m] = bulan.split("-").map(Number);
		const lastDay = new Date(Date.UTC(y, m, 0)).toISOString().slice(0, 10);
		const today = data?.updated_at?.slice(0, 10) ?? lastDay;
		return "/app/leads/dashboard?" + new URLSearchParams({ from: `${bulan}-01`, to: lastDay < today ? lastDay : today }).toString();
	});

	const refreshed = (iso: string) =>
		new Intl.DateTimeFormat("id-ID", { day: "numeric", month: "short", hour: "2-digit", minute: "2-digit", timeZone: "Asia/Jakarta" }).format(new Date(iso)) + " WIB";

	function visit(month: string) {
		if (loading) return;
		loadError = "";
		router.get("/app", month ? { bulan: month } : {}, {
			preserveScroll: true,
			preserveState: true,
			onStart: () => (loading = true),
			onFinish: () => (loading = false),
			onError: () => {
				loadError = "Dashboard belum berhasil diperbarui. Coba muat ulang.";
			},
			onNetworkError: () => {
				loadError = "Koneksi terputus. Data sebelumnya tetap ditampilkan. Coba perbarui lagi.";
				return false;
			},
			onHttpException: () => {
				loadError = "Server belum berhasil memuat dashboard. Coba perbarui lagi.";
				return false;
			},
		});
	}
</script>

<svelte:head><title>Ringkasan · Ruang Sanad</title></svelte:head>
<AppLayout {user}>
	<div class="mx-auto max-w-7xl space-y-4 px-4 py-6 sm:px-6 lg:py-8">
		<header class="flex flex-col gap-4 pb-2 sm:flex-row sm:items-end sm:justify-between">
			<div class="min-w-0">
				<p class="text-[11px] font-semibold uppercase tracking-[0.12em] text-brand-700 dark:text-brand-400">Ruang Sanad · Ringkasan</p>
				<h1 class="mt-1 text-[28px] font-semibold leading-tight tracking-[-0.02em] text-neutral-950 [text-wrap:balance] dark:text-white">{management ? "Kondisi lembaga" : "Ringkasan keuangan"}</h1>
				<p class="mt-1 max-w-2xl text-sm text-neutral-500 [text-wrap:pretty] dark:text-neutral-400">
					Status, kelas, riayah, stok, dan piutang adalah kondisi terkini. Aktivitas dan tren mengikuti <span class="font-medium text-neutral-700 dark:text-neutral-200">{monthLabel}</span>{data?.current_month ? " (bulan berjalan)" : ""}.
				</p>
			</div>
			{#if data}
				<MonthControl {bulan} {maxMonth} {loading} updated={refreshed(data.updated_at)} onvisit={visit} />
			{/if}
		</header>

		{#if loadError}<div role="alert" class="rounded-lg border border-red-500/30 bg-red-500/[0.06] px-4 py-3 text-sm text-red-800 dark:text-red-300">{loadError}</div>{/if}
		{#if sourceErrors.length}
			<div role="alert" class="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-amber-500/30 bg-amber-500/[0.06] px-4 py-3 text-sm text-amber-900 dark:text-amber-200">
				<p><strong class="font-semibold">Sebagian data belum tersedia:</strong> {sourceErrors.join(", ")}. Angka di bagian itu ditandai "—", bukan nol.</p>
				<button type="button" class="rounded-md border border-amber-600/30 px-3 py-1.5 text-xs font-medium hover:bg-amber-500/10 disabled:opacity-50" onclick={() => visit(bulan)} disabled={loading}>Coba lagi</button>
			</div>
		{/if}

		{#if !data}
			<div role="alert" class="rounded-xl border border-dashed border-neutral-300 px-6 py-14 text-center dark:border-neutral-700">
				<p class="text-sm font-medium text-neutral-800 dark:text-neutral-200">Ringkasan belum tersedia.</p>
				<button type="button" class="mt-3 rounded-lg bg-brand-600 px-4 py-2 text-sm font-semibold text-white hover:bg-brand-700" onclick={() => visit("")}>Muat dashboard</button>
			</div>
		{:else}
			{#if loading}<p role="status" aria-live="polite" class="sr-only">Memuat data dashboard…</p>{/if}
			<div class={"space-y-4 transition-opacity duration-150 " + (loading ? "pointer-events-none opacity-60" : "")} aria-busy={loading}>
				{#if management}
					<InstitutionChain links={chain} />

					<div class="grid grid-cols-[minmax(0,1fr)] gap-4 lg:grid-cols-[minmax(0,1.6fr)_minmax(0,1fr)]">
						<PriorityList priorities={data.priorities} incomplete={sourceErrors.length > 0} />
						<DecisionNotes {notes} />
					</div>

					<div class="grid grid-cols-[minmax(0,1fr)] gap-4 lg:grid-cols-2">
						<EnrollmentPanel growth={data.growth} leads={data.leads} {leadsHref} currentMonth={data.current_month} />
						<AttendancePanel {attendance} available={!sourceErrors.includes("Kehadiran santri")} />
					</div>

					<ProgramTable programs={data.programs} failed={sourceErrors.includes("Performa program")} />
					<OperationsLedger {ops} {riayah} {products} {monthLabel} />
				{:else}
					<div class="grid grid-cols-[minmax(0,1fr)] gap-4 lg:grid-cols-[minmax(0,1.6fr)_minmax(0,1fr)]">
						<PriorityList priorities={data.priorities} incomplete={sourceErrors.length > 0} />
						<DecisionNotes {notes} />
					</div>
				{/if}

				{#if financeAllowed}
					<div class={management ? "pt-4" : ""}>
						<FinancePanel {finance} {bulan} {monthLabel} primary={!management} />
					</div>
				{/if}

				<details class="rounded-xl border border-neutral-200/80 px-5 py-3 text-sm text-neutral-600 sm:px-6 dark:border-white/[0.07] dark:text-neutral-300">
					<summary class="w-fit py-1 text-xs font-medium text-neutral-500 dark:text-neutral-400">Cakupan data dan cara membaca dashboard</summary>
					<div class="mt-2 max-w-3xl space-y-2 text-xs leading-relaxed text-neutral-500 dark:text-neutral-400">
						<p>Dashboard membaca data tersimpan saat halaman dimuat. Gunakan tombol perbarui untuk mengambil kondisi terbaru. Pergantian bulan hanya mengubah aktivitas dan tren, bukan kondisi saat ini.</p>
						{#if management}
							<p>Rantai lembaga menandai mata rantai "perlu perhatian" hanya dari penanda yang tercatat: riayah, santri tanpa kelas, kelas tanpa guru aktif, tindak lanjut dan tugas koordinator yang jatuh tempo, pertemuan lama yang belum ditutup, dan tagihan lewat jatuh tempo. Penanda bukan bukti santri pasti berhenti.</p>
							<p>Kehadiran dihitung dari absensi pertemuan berstatus selesai. Lead dan closing berasal dari CRM WhatsApp; closing memakai tanggal konversi menjadi santri.</p>
						{/if}
						<p>Penerimaan mencakup SPP lunas berdasarkan tanggal bayar serta order produk aktif. Tagihan batal dan order dibatalkan tidak dihitung. Persentase tagihan lunas memakai nominal tagihan yang terbit pada bulan terpilih dan kini lunas.</p>
					</div>
				</details>
			</div>
		{/if}
	</div>
</AppLayout>
