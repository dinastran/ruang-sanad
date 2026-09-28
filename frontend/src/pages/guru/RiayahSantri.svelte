<script lang="ts">
	import { inertia } from "@inertiajs/svelte";
	import { fly } from "svelte/transition";
	import AppLayout from "@layouts/AppLayout.svelte";
	import PenandaBadge from "@components/riayah/PenandaBadge.svelte";
	import CatatKontakDialog from "@components/riayah/CatatKontakDialog.svelte";
	import SapaWADialog from "@components/riayah/SapaWADialog.svelte";
	import { hariLalu, labelPeriode } from "@lib/riayah";
	import type { Flash, User, RiayahSantriItem, RiayahRingkasan, RiayahPenandaKode, RiayahWATemplate, RiayahGuruMonitoring } from "@lib/types";
	import { HeartHandshake, Search, ChevronRight, Users, Eye, NotebookPen, MessageCircle, FileText } from "lucide-svelte";

	interface Props {
		user?: User;
		santri?: RiayahSantriItem[];
		ringkasan?: RiayahRingkasan;
		can_write?: boolean;
		is_semua?: boolean;
		is_koordinator?: boolean;
		guru_monitoring?: RiayahGuruMonitoring[];
		monitoring_error?: string;
		wa_templates?: RiayahWATemplate[];
		flash?: Flash;
		success?: string;
		error?: string;
	}

	let {
		user,
		santri = [],
		ringkasan,
		can_write = false,
		is_semua = false,
		is_koordinator = false,
		guru_monitoring = [],
		monitoring_error = "",
		wa_templates = [],
		flash,
		success,
		error,
	}: Props = $props();

	type Filter = "semua" | "perhatian" | "rapor" | RiayahPenandaKode;
	type ViewMode = "guru" | "santri";
	let mode = $state<ViewMode>(is_koordinator ? "guru" : "santri");
	let filter = $state<Filter>("perhatian");
	let search = $state("");
	let guruSearch = $state("");
	let guruID = $state("");
	let kelasID = $state("");

	let kelasOptions = $derived(
		Array.from(new Map(santri.filter((s) => !guruID || String(s.guru_id) === guruID).map((s) => [s.kelas_id, s.nama_kelas])).entries()).sort((a, b) => a[1].localeCompare(b[1])),
	);

	let guruOptions = $derived(guru_monitoring.slice().sort((a, b) => a.guru_nama.localeCompare(b.guru_nama)));

	function needsCoordinatorAttention(s: RiayahSantriItem): boolean {
		return s.guru_belum_disapa || s.penanda.some((p) => p.kode !== "kontak");
	}

	let scopedSantri = $derived.by(() => santri.filter((s) => !guruID || String(s.guru_id) === guruID));

	let filtered = $derived.by(() => {
		const q = search.trim().toLowerCase();
		return scopedSantri.filter((s) => {
			if (is_koordinator) {
				if (filter === "perhatian" && !needsCoordinatorAttention(s)) return false;
				if (filter === "rapor" && s.guru_rapor_terkirim) return false;
				if (filter === "kontak" && !s.guru_belum_disapa) return false;
				if (filter !== "semua" && filter !== "perhatian" && filter !== "rapor" && filter !== "kontak" && !s.penanda.some((p) => p.kode === filter)) return false;
			} else {
				if (filter === "perhatian" && s.penanda.length === 0) return false;
				if (filter === "rapor" && s.rapor_terkirim) return false;
				if (filter !== "semua" && filter !== "perhatian" && filter !== "rapor" && !s.penanda.some((p) => p.kode === filter)) return false;
			}
			if (kelasID && String(s.kelas_id) !== kelasID) return false;
			if (q && !s.nama.toLowerCase().includes(q) && !s.id_mahasantri.toLowerCase().includes(q) && !s.nama_kelas.toLowerCase().includes(q)) return false;
			return true;
		});
	});

	let chips = $derived.by(() => {
		const base = scopedSantri;
		const perhatian = is_koordinator ? base.filter(needsCoordinatorAttention).length : (ringkasan?.perlu_perhatian ?? 0);
		const kontak = is_koordinator ? base.filter((s) => s.guru_belum_disapa).length : (ringkasan?.kontak ?? 0);
		const raporBelum = is_koordinator ? base.filter((s) => !s.guru_rapor_terkirim).length : (ringkasan?.total_santri ?? santri.length) - (ringkasan?.rapor_terkirim ?? 0);
		return [
			{ key: "perhatian" as Filter, label: "Perlu perhatian", count: perhatian, dot: "bg-brand-500" },
			{ key: "kehadiran" as Filter, label: "Kehadiran", count: base.filter((s) => s.penanda.some((p) => p.kode === "kehadiran")).length, dot: "bg-red-500" },
			{ key: "kontak" as Filter, label: is_koordinator ? "Belum disapa guru" : "Belum disapa", count: kontak, dot: "bg-orange-500" },
			{ key: "progres" as Filter, label: "Progres macet", count: base.filter((s) => s.penanda.some((p) => p.kode === "progres")).length, dot: "bg-amber-500" },
			{ key: "rapor" as Filter, label: `Rapor ${ringkasan?.rapor_periode ? labelPeriode(ringkasan.rapor_periode).split(" ")[0] : ""} belum dikirim`, count: raporBelum, dot: "bg-sky-500" },
			{ key: "semua" as Filter, label: "Semua santri", count: base.length, dot: "bg-neutral-400" },
		];
	});

	let filteredGuru = $derived.by(() => {
		const q = guruSearch.trim().toLowerCase();
		return guru_monitoring.filter((item) => !q || item.guru_nama.toLowerCase().includes(q));
	});

	let guruSummary = $derived({
		totalGuru: guru_monitoring.length,
		totalSantri: guru_monitoring.reduce((sum, item) => sum + item.total_santri, 0),
		perluPerhatian: guru_monitoring.reduce((sum, item) => sum + item.perlu_perhatian, 0),
		belumDisapa: guru_monitoring.reduce((sum, item) => sum + item.belum_disapa, 0),
		belumPernah: guru_monitoring.reduce((sum, item) => sum + item.belum_pernah_disapa, 0),
		raporBelum: guru_monitoring.reduce((sum, item) => sum + item.rapor_belum, 0),
	});

	let kontakSantri = $state<RiayahSantriItem | null>(null);
	let waSantri = $state<RiayahSantriItem | null>(null);

	function persen(v: number | null): string {
		return v === null ? "–" : `${Math.round(v)}%`;
	}

	function pilihGuru(item: RiayahGuruMonitoring) {
		guruID = String(item.guru_id);
		kelasID = "";
		search = "";
		filter = "semua";
		mode = "santri";
	}

	function resetGuruFilter() {
		guruID = "";
		kelasID = "";
		search = "";
		filter = "perhatian";
	}

	function aktivitasLabel(value: string): string {
		if (!value) return "Belum ada aktivitas";
		return hariLalu(value.slice(0, 10));
	}
</script>

<svelte:head><title>Riayah Santri</title></svelte:head>

<AppLayout {user} group="guru-riayah">
	<div class="pt-8 pb-8 border-b border-neutral-200/80 dark:border-white/[0.04]">
		<div class="max-w-6xl mx-auto px-4 sm:px-6">
			<h1 class="text-2xl sm:text-3xl font-bold text-neutral-900 dark:text-white mb-2 tracking-tight">Riayah Santri</h1>
			<p class="text-neutral-600 dark:text-neutral-400">{is_koordinator ? "Pantau pelaksanaan riayah per guru, lalu telusuri santri yang belum terjangkau." : "Santri yang perlu disapa dan dibimbing lebih dulu ada di urutan teratas."}</p>
		</div>
	</div>

	<div class="max-w-6xl mx-auto px-4 sm:px-6 py-8 space-y-6">
		{#if success || flash?.success}
			<div class="bg-green-500/10 border border-green-500/20 text-green-700 dark:text-green-400 rounded-2xl p-4 text-sm font-medium" in:fly={{ y: 10, duration: 200 }}>{success || flash?.success}</div>
		{/if}
		{#if error || flash?.error}
			<div class="bg-red-500/10 border border-red-500/20 text-red-600 dark:text-red-400 rounded-2xl p-4 text-sm font-medium" in:fly={{ y: 10, duration: 200 }}>{error || flash?.error}</div>
		{/if}

		{#if !can_write}
			<div class="flex items-center gap-2.5 rounded-2xl border border-neutral-200 dark:border-neutral-800 bg-neutral-50 dark:bg-neutral-900/40 px-4 py-3 text-sm text-neutral-600 dark:text-neutral-400">
				<Eye class="w-4 h-4 shrink-0" />
				{is_koordinator ? "Mode monitoring: Koordinator dapat melihat aktivitas riayah guru, tetapi pencatatan tetap dilakukan oleh guru." : "Mode baca: Anda dapat memantau semua santri, pencatatan dilakukan oleh guru."}
			</div>
		{/if}

		{#if monitoring_error}
			<div class="rounded-2xl border border-amber-500/20 bg-amber-500/10 px-4 py-3 text-sm text-amber-800 dark:text-amber-300">{monitoring_error}</div>
		{/if}

		{#if is_koordinator}
			<div class="inline-flex rounded-xl border border-neutral-200 bg-white p-1 dark:border-neutral-800 dark:bg-neutral-925/50" role="group" aria-label="Mode monitoring riayah">
				<button type="button" onclick={() => (mode = "guru")} class="rounded-lg px-4 py-2 text-sm font-semibold transition {mode === 'guru' ? 'bg-brand-600 text-white' : 'text-neutral-600 hover:bg-neutral-100 dark:text-neutral-300 dark:hover:bg-neutral-800'}">Per Guru</button>
				<button type="button" onclick={() => (mode = "santri")} class="rounded-lg px-4 py-2 text-sm font-semibold transition {mode === 'santri' ? 'bg-brand-600 text-white' : 'text-neutral-600 hover:bg-neutral-100 dark:text-neutral-300 dark:hover:bg-neutral-800'}">Per Santri</button>
			</div>
		{/if}

		{#if is_koordinator && mode === "guru"}
			<div class="grid grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-6">
				<div class="rounded-2xl border border-neutral-200 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-925/50"><p class="text-xs text-neutral-500">Guru dipantau</p><p class="mt-2 text-2xl font-bold text-neutral-900 dark:text-white">{guruSummary.totalGuru}</p></div>
				<div class="rounded-2xl border border-neutral-200 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-925/50"><p class="text-xs text-neutral-500">Santri aktif</p><p class="mt-2 text-2xl font-bold text-neutral-900 dark:text-white">{guruSummary.totalSantri}</p></div>
				<div class="rounded-2xl border border-brand-500/20 bg-brand-500/5 p-4"><p class="text-xs text-neutral-500">Perlu perhatian</p><p class="mt-2 text-2xl font-bold text-brand-700 dark:text-brand-300">{guruSummary.perluPerhatian}</p></div>
				<div class="rounded-2xl border border-orange-500/20 bg-orange-500/5 p-4"><p class="text-xs text-neutral-500">Disapa &gt;14 hari lalu</p><p class="mt-2 text-2xl font-bold text-orange-700 dark:text-orange-300">{guruSummary.belumDisapa}</p></div>
				<div class="rounded-2xl border border-neutral-200 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-925/50"><p class="text-xs text-neutral-500">Belum pernah disapa</p><p class="mt-2 text-2xl font-bold text-neutral-900 dark:text-white">{guruSummary.belumPernah}</p></div>
				<div class="rounded-2xl border border-sky-500/20 bg-sky-500/5 p-4"><p class="text-xs text-neutral-500">Rapor belum</p><p class="mt-2 text-2xl font-bold text-sky-700 dark:text-sky-300">{guruSummary.raporBelum}</p></div>
			</div>

			<label class="relative block">
				<span class="sr-only">Cari guru</span>
				<Search class="absolute left-3 top-3 w-4 h-4 text-neutral-400" />
				<input bind:value={guruSearch} placeholder="Cari nama guru" class="w-full pl-9 pr-3 py-2.5 rounded-xl bg-white dark:bg-neutral-925 border border-neutral-200 dark:border-neutral-800 text-sm focus:outline-none focus:ring-2 focus:ring-brand-400/40" />
			</label>

			<section class="space-y-3" aria-label="Monitoring riayah per guru">
				{#if guru_monitoring.length === 0}
					<div class="rounded-2xl border border-dashed border-neutral-300 p-10 text-center text-sm text-neutral-500 dark:border-neutral-700">Belum ada guru dengan santri aktif untuk dimonitor.</div>
				{:else if filteredGuru.length === 0}
					<div class="rounded-2xl border border-dashed border-neutral-300 p-10 text-center text-sm text-neutral-500 dark:border-neutral-700">Tidak ada guru yang cocok dengan pencarian.</div>
				{:else}
					{#each filteredGuru as row (row.guru_id)}
						<article class="rounded-2xl border border-neutral-200 bg-white p-5 dark:border-neutral-800 dark:bg-neutral-925/50">
							<div class="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
								<div class="min-w-0">
									<h2 class="truncate text-base font-semibold text-neutral-900 dark:text-white">{row.guru_nama}</h2>
									<p class="mt-1 text-xs text-neutral-500">{row.total_santri} santri aktif · aktivitas terakhir {aktivitasLabel(row.aktivitas_terakhir)}</p>
								</div>
								<button type="button" onclick={() => pilihGuru(row)} class="inline-flex shrink-0 items-center justify-center gap-1.5 rounded-xl border border-brand-500/30 px-3.5 py-2 text-xs font-semibold text-brand-700 hover:bg-brand-500/5 dark:text-brand-300">
									Lihat santri <ChevronRight class="h-3.5 w-3.5" />
								</button>
							</div>

							<div class="mt-4 grid gap-3 sm:grid-cols-2 xl:grid-cols-5">
								<div class="rounded-xl bg-neutral-50 p-3 dark:bg-neutral-900/50">
									<div class="flex items-center justify-between gap-2"><p class="text-xs text-neutral-500">Coverage 14 hari</p><p class="font-mono text-sm font-semibold text-neutral-900 dark:text-white">{Math.round(row.coverage_riayah)}%</p></div>
									<div class="mt-2 h-1.5 overflow-hidden rounded-full bg-neutral-200 dark:bg-neutral-800"><div class="h-full rounded-full bg-brand-500" style={"width: " + Math.max(0, Math.min(100, row.coverage_riayah)) + "%"}></div></div>
								</div>
								<div class="rounded-xl bg-neutral-50 p-3 dark:bg-neutral-900/50"><p class="text-xs text-neutral-500">Perlu perhatian</p><p class="mt-1 text-lg font-bold text-neutral-900 dark:text-white">{row.perlu_perhatian}</p></div>
								<div class="rounded-xl bg-neutral-50 p-3 dark:bg-neutral-900/50"><p class="text-xs text-neutral-500">Disapa &gt;14 hari lalu</p><p class="mt-1 text-lg font-bold text-neutral-900 dark:text-white">{row.belum_disapa}</p></div>
								<div class="rounded-xl bg-neutral-50 p-3 dark:bg-neutral-900/50"><p class="text-xs text-neutral-500">Belum pernah disapa</p><p class="mt-1 text-lg font-bold text-neutral-900 dark:text-white">{row.belum_pernah_disapa}</p></div>
								<div class="rounded-xl bg-neutral-50 p-3 dark:bg-neutral-900/50"><p class="text-xs text-neutral-500">Rapor belum</p><p class="mt-1 text-lg font-bold text-neutral-900 dark:text-white">{row.rapor_belum}</p></div>
							</div>
						</article>
					{/each}
				{/if}
			</section>

			<p class="text-xs leading-5 text-neutral-500">Coverage 14 hari menghitung santri aktif yang benar-benar disapa oleh guru penanggung jawab dalam 14 hari terakhir. Aktivitas Admin atau Super Admin tidak menaikkan coverage guru.</p>
		{:else}
			{#if is_koordinator && guruID}
				<div class="flex flex-wrap items-center justify-between gap-3 rounded-2xl border border-brand-500/20 bg-brand-500/5 px-4 py-3">
					<p class="text-sm text-neutral-700 dark:text-neutral-300">Menampilkan santri <span class="font-semibold">{guruOptions.find((g) => String(g.guru_id) === guruID)?.guru_nama ?? "guru terpilih"}</span>.</p>
					<button type="button" onclick={resetGuruFilter} class="text-xs font-semibold text-brand-700 hover:underline dark:text-brand-300">Tampilkan semua guru</button>
				</div>
			{/if}

			<div class="flex flex-wrap gap-2" role="group" aria-label="Filter penanda">
				{#each chips as chip (chip.key)}
					<button
						type="button"
						onclick={() => (filter = chip.key)}
						aria-pressed={filter === chip.key}
						class="inline-flex min-h-10 items-center gap-2 whitespace-nowrap rounded-xl border px-3.5 text-sm font-medium transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-brand-400/50 {filter === chip.key
							? 'border-brand-500/40 bg-brand-400/10 text-brand-800 dark:text-brand-200'
							: 'border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-925/50 text-neutral-700 dark:text-neutral-300 hover:border-brand-400/30'}"
					>
						<span class="h-2 w-2 rounded-full {chip.dot}" aria-hidden="true"></span>
						{chip.label}
						<span class="font-mono text-xs text-neutral-500">{chip.count}</span>
					</button>
				{/each}
			</div>

			<div class="grid gap-3 {is_koordinator ? 'sm:grid-cols-2 lg:grid-cols-[1fr_15rem_15rem]' : 'sm:grid-cols-[1fr_16rem]'}">
				<label class="relative">
					<span class="sr-only">Cari santri</span>
					<Search class="absolute left-3 top-3 w-4 h-4 text-neutral-400" />
					<input bind:value={search} placeholder="Cari nama, ID, atau kelas" class="w-full pl-9 pr-3 py-2.5 rounded-xl bg-white dark:bg-neutral-925 border border-neutral-200 dark:border-neutral-800 text-sm focus:outline-none focus:ring-2 focus:ring-brand-400/40" />
				</label>
				{#if is_koordinator}
					<label>
						<span class="sr-only">Filter guru</span>
						<select bind:value={guruID} onchange={() => (kelasID = "")} class="w-full px-3 py-2.5 rounded-xl bg-white dark:bg-neutral-925 border border-neutral-200 dark:border-neutral-800 text-sm focus:outline-none focus:ring-2 focus:ring-brand-400/40">
							<option value="">Semua guru</option>
							{#each guruOptions as item (item.guru_id)}<option value={String(item.guru_id)}>{item.guru_nama}</option>{/each}
						</select>
					</label>
				{/if}
				<label>
					<span class="sr-only">Filter kelas</span>
					<select bind:value={kelasID} class="w-full px-3 py-2.5 rounded-xl bg-white dark:bg-neutral-925 border border-neutral-200 dark:border-neutral-800 text-sm focus:outline-none focus:ring-2 focus:ring-brand-400/40">
						<option value="">Semua kelas</option>
						{#each kelasOptions as [id, nama] (id)}<option value={String(id)}>{nama}</option>{/each}
					</select>
				</label>
			</div>

			<section class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 overflow-hidden" aria-label="Daftar santri">
				{#if santri.length === 0}
					<div class="p-12 text-center text-neutral-500">
						<Users class="w-8 h-8 mx-auto mb-3 opacity-50" />
						<p class="font-medium text-neutral-700 dark:text-neutral-300">Belum ada santri aktif</p>
						<p class="mt-1 text-sm">Santri akan muncul di sini setelah ditempatkan ke kelas.</p>
					</div>
				{:else if filtered.length === 0}
					<div class="p-12 text-center text-neutral-500">
						<HeartHandshake class="w-8 h-8 mx-auto mb-3 opacity-50" />
						{#if filter === "perhatian" && !search && !kelasID}
							<p class="font-medium text-neutral-700 dark:text-neutral-300">Tidak ada santri yang ditandai saat ini</p>
							<p class="mt-1 text-sm">Lihat <button type="button" class="underline text-brand-700 dark:text-brand-300" onclick={() => (filter = "semua")}>semua santri</button>.</p>
						{:else}
							<p class="font-medium text-neutral-700 dark:text-neutral-300">Tidak ada santri yang cocok dengan filter</p>
						{/if}
					</div>
				{:else}
					<ul class="divide-y divide-neutral-100 dark:divide-neutral-800">
						{#each filtered as s (s.id)}
							<li class="flex items-stretch hover:bg-neutral-50 dark:hover:bg-white/[0.02]">
								<a href={"/app/guru/santri/" + s.id} use:inertia class="flex min-w-0 flex-1 items-start gap-4 px-5 py-4 focus:outline-none focus-visible:bg-brand-400/5">
									<div class="min-w-0 flex-1">
										<div class="flex flex-wrap items-baseline gap-x-2">
											<p class="font-semibold text-neutral-900 dark:text-white">{s.nama}</p>
											<p class="text-xs text-neutral-500">{s.nama_kelas} · {s.level}{is_semua && s.guru_nama ? " · " + s.guru_nama : ""}</p>
										</div>

										<div class="mt-2 flex flex-wrap gap-1.5">
											{#each s.penanda.filter((p) => !is_koordinator || p.kode !== "kontak") as p (p.kode)}<PenandaBadge penanda={p} showAlasan />{/each}
											{#if is_koordinator && s.guru_belum_pernah_disapa}
												<span class="rounded-full bg-orange-500/10 px-2.5 py-1 text-xs font-medium text-orange-700 dark:text-orange-300">Belum pernah disapa guru</span>
											{:else if is_koordinator && s.guru_belum_disapa}
												<span class="rounded-full bg-orange-500/10 px-2.5 py-1 text-xs font-medium text-orange-700 dark:text-orange-300">Guru belum menyapa &gt;14 hari</span>
											{/if}
										</div>

										<dl class="mt-2 flex flex-wrap gap-x-5 gap-y-1 text-xs text-neutral-500">
											<div class="flex gap-1"><dt>Hadir 30 hari:</dt><dd class="font-mono text-neutral-700 dark:text-neutral-300">{persen(s.persen_hadir_30)}{s.total_pertemuan_30 ? " (" + s.total_pertemuan_30 + "x)" : ""}</dd></div>
											<div class="flex gap-1 min-w-0"><dt>Batas materi:</dt><dd class="truncate text-neutral-700 dark:text-neutral-300">{s.batas_materi_terakhir || "–"}</dd></div>
											<div class="flex gap-1"><dt>{is_koordinator ? "Disapa guru:" : "Disapa:"}</dt><dd class="text-neutral-700 dark:text-neutral-300">{(is_koordinator ? s.kontak_guru_terakhir : s.kontak_terakhir) ? hariLalu(is_koordinator ? s.kontak_guru_terakhir : s.kontak_terakhir) : "belum pernah"}</dd></div>
											{#if is_koordinator ? s.guru_rapor_terkirim : s.rapor_terkirim}<div class="flex items-center gap-1 text-sky-700 dark:text-sky-400"><FileText class="w-3 h-3" /><dt class="sr-only">Rapor</dt><dd>Rapor {labelPeriode(ringkasan?.rapor_periode ?? "").split(" ")[0]} terkirim</dd></div>{/if}
										</dl>
									</div>
									<ChevronRight class="mt-1 w-4 h-4 shrink-0 text-neutral-400" />
								</a>
								{#if can_write}
									<div class="flex shrink-0 items-center gap-2 pr-4">
										{#if s.no_wa}
											<button type="button" onclick={() => (waSantri = s)} class="inline-flex min-h-10 items-center gap-1.5 whitespace-nowrap rounded-xl bg-brand-600 px-3 text-xs font-semibold text-white hover:bg-brand-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-brand-400/60" aria-label={"Sapa " + s.nama + " via WhatsApp"}>
												<MessageCircle class="w-3.5 h-3.5" /><span class="hidden sm:inline">WA</span>
											</button>
										{/if}
										<button type="button" onclick={() => (kontakSantri = s)} class="inline-flex min-h-10 items-center gap-1.5 whitespace-nowrap rounded-xl border border-neutral-200 px-3 text-xs font-semibold text-neutral-700 hover:border-brand-400/40 hover:text-brand-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-brand-400/50 dark:border-neutral-700 dark:text-neutral-300 dark:hover:text-brand-300" aria-label={"Catat sapaan untuk " + s.nama}>
											<NotebookPen class="w-3.5 h-3.5" /><span class="hidden sm:inline">Catat sapaan</span>
										</button>
									</div>
								{/if}
							</li>
						{/each}
					</ul>
				{/if}
			</section>

			<p class="text-xs leading-5 text-neutral-500">
				{#if is_koordinator}
					Monitoring sapaan menggunakan aktivitas yang dicatat oleh guru penanggung jawab saat ini. Penanda kehadiran dan progres tetap mengikuti kondisi santri.
				{:else}
					Penanda: <span class="text-red-600 dark:text-red-400">merah</span> alpa 2x berturut-turut atau kehadiran di bawah 70% (minimal 3 pertemuan dalam 30 hari);
					<span class="text-orange-700 dark:text-orange-400">oranye</span> belum disapa lebih dari 14 hari (dihitung dari mulai belajar jika belum pernah);
					<span class="text-amber-700 dark:text-amber-400">kuning</span> batas materi tidak berubah dalam 4 pertemuan hadir terakhir.
				{/if}
			</p>
		{/if}
	</div>

	{#if kontakSantri}
		<CatatKontakDialog santri={kontakSantri} kembali="/app/guru/riayah" onclose={() => (kontakSantri = null)} />
	{/if}
	{#if waSantri}
		<SapaWADialog santri={waSantri} templates={wa_templates} pengirim={user?.name ?? ""} kembali="/app/guru/riayah" onclose={() => (waSantri = null)} />
	{/if}
</AppLayout>
