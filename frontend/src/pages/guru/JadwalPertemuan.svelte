<script lang="ts">
	import { inertia, router } from "@inertiajs/svelte";
	import AppLayout from "@layouts/AppLayout.svelte";
	import type { Flash, GuruDirectory, GuruKelas, JadwalPertemuanGuru, User } from "@lib/types";
	import {
		AlertTriangle, ArrowLeft, Ban, BookOpen, CalendarClock, CalendarDays,
		MoreHorizontal, Play, Plus, RotateCcw, UserRoundCheck
	} from "lucide-svelte";

	type AgendaFilter = "attention" | "today" | "week" | "all";

	interface Props {
		user?: User;
		jadwal?: JadwalPertemuanGuru[];
		kelas?: GuruKelas[];
		guruList?: GuruDirectory[];
		selected_kelas_id?: number;
		success?: string;
		error?: string;
		flash?: Flash;
	}

	let { user, jadwal = [], kelas = [], guruList = [], selected_kelas_id = 0, success, error, flash }: Props = $props();

	const now = new Date();
	const today = localDate(now);
	const currentTime = now.toTimeString().slice(0, 5);
	const weekEnd = addDays(today, 6);

	function initialClassID() {
		return selected_kelas_id || kelas[0]?.id || 0;
	}

	function initialClassFilter() {
		return selected_kelas_id || 0;
	}

	function initialAgendaFilter(): AgendaFilter {
		if (typeof window === "undefined") return "today";
		const value = new URLSearchParams(window.location.search).get("filter");
		return value === "attention" || value === "today" || value === "week" || value === "all" ? value : "today";
	}

	let showCreate = $state(false);
	let kelasID = $state(initialClassID());
	let kelasFilter = $state(initialClassFilter());
	let agendaFilter = $state<AgendaFilter>(initialAgendaFilter());
	let tanggal = $state("");
	let jamMulai = $state("");
	let catatan = $state("");
	let submitting = $state(false);
	let openMenuID = $state<number | null>(null);
	let action = $state<{ type: "reschedule" | "badal"; item: JadwalPertemuanGuru } | null>(null);
	let tanggalBaru = $state("");
	let jamBaru = $state("");
	let guruPenggantiID = $state("");
	let alasan = $state("");

	let availableGuru = $derived(guruList.filter((guru) => guru.is_aktif && guru.user_id && guru.id !== action?.item.guru_utama_id));
	let attentionItems = $derived(jadwal.filter((item) => needsAttention(item)));
	let todayItems = $derived(jadwal.filter((item) => item.tanggal === today));
	let weekItems = $derived(jadwal.filter((item) => item.tanggal >= today && item.tanggal <= weekEnd));

	let visibleItems = $derived.by(() => {
		const byClass = kelasFilter ? jadwal.filter((item) => item.kelas_id === kelasFilter) : jadwal;
		switch (agendaFilter) {
			case "attention":
				return byClass.filter((item) => needsAttention(item));
			case "today":
				return byClass.filter((item) => item.tanggal === today);
			case "week":
				return byClass.filter((item) => item.tanggal >= today && item.tanggal <= weekEnd);
			default:
				return byClass;
		}
	});

	let groupedItems = $derived.by(() => {
		const groups = new Map<string, JadwalPertemuanGuru[]>();
		for (const item of visibleItems) {
			const current = groups.get(item.tanggal) || [];
			current.push(item);
			groups.set(item.tanggal, current);
		}
		return Array.from(groups.entries());
	});

	function localDate(date: Date) {
		const offset = date.getTimezoneOffset() * 60_000;
		return new Date(date.getTime() - offset).toISOString().slice(0, 10);
	}

	function addDays(value: string, days: number) {
		const date = new Date(value + "T00:00:00");
		date.setDate(date.getDate() + days);
		return localDate(date);
	}

	function formatDate(value: string) {
		return new Intl.DateTimeFormat("id-ID", { weekday: "long", day: "numeric", month: "long", year: "numeric" }).format(new Date(value + "T00:00:00"));
	}

	function dateGroupLabel(value: string) {
		if (value === today) return "Hari ini";
		if (value === addDays(today, 1)) return "Besok";
		return formatDate(value);
	}

	function isOverdue(item: JadwalPertemuanGuru) {
		return item.status === "dijadwalkan" && item.tanggal < today;
	}

	function isLateToday(item: JadwalPertemuanGuru) {
		return item.status === "dijadwalkan" && item.tanggal === today && item.jam_mulai <= currentTime;
	}

	function needsAttention(item: JadwalPertemuanGuru) {
		return item.status === "dimulai" || isOverdue(item) || isLateToday(item) || !!item.jadwal_kelas_berubah;
	}

	function statusInfo(item: JadwalPertemuanGuru) {
		if (item.status === "dimulai") return { label: "Sedang berlangsung", cls: "bg-amber-500/10 text-amber-700 dark:text-amber-300" };
		if (isOverdue(item)) return { label: "Terlambat", cls: "bg-red-500/10 text-red-700 dark:text-red-300" };
		if (isLateToday(item)) return { label: "Waktunya dimulai", cls: "bg-red-500/10 text-red-700 dark:text-red-300" };
		if (item.tanggal === today) return { label: "Hari ini", cls: "bg-brand-400/10 text-brand-700 dark:text-brand-300" };
		return { label: "Terjadwal", cls: "bg-neutral-500/10 text-neutral-700 dark:text-neutral-300" };
	}

	function cardClass(item: JadwalPertemuanGuru) {
		if (item.status === "dimulai") return "border-amber-500/30 bg-amber-500/[0.035]";
		if (isOverdue(item) || isLateToday(item)) return "border-red-500/25 bg-red-500/[0.025]";
		if (item.tanggal === today) return "border-brand-400/25 bg-brand-400/[0.025]";
		return "border-neutral-200/80 bg-white dark:border-white/[0.06] dark:bg-neutral-925/50";
	}

	function filterLabel() {
		if (agendaFilter === "attention") return "perlu tindakan";
		if (agendaFilter === "today") return "hari ini";
		if (agendaFilter === "week") return "7 hari ke depan";
		return "semua agenda aktif";
	}

	function openAction(type: "reschedule" | "badal", item: JadwalPertemuanGuru) {
		openMenuID = null;
		action = { type, item };
		tanggalBaru = item.tanggal;
		jamBaru = item.jam_mulai;
		guruPenggantiID = item.guru_pengganti_id ? String(item.guru_pengganti_id) : "";
		alasan = type === "reschedule" ? item.alasan_reschedule : item.alasan_badal;
	}

	function closeAction() {
		action = null;
		tanggalBaru = "";
		jamBaru = "";
		guruPenggantiID = "";
		alasan = "";
	}

	function createSchedule() {
		if (!kelasID || !tanggal || !jamMulai) return;
		submitting = true;
		router.post("/app/guru/jadwal-pertemuan", { kelas_id: kelasID, tanggal, jam_mulai: jamMulai, catatan }, {
			onSuccess: (page) => {
				const responseFlash = page.props.flash as Flash | undefined;
				if (!responseFlash?.error) {
					tanggal = "";
					jamMulai = "";
					catatan = "";
					showCreate = false;
				}
			},
			onFinish: () => (submitting = false),
		});
	}

	function submitAction() {
		if (!action) return;
		const base = `/app/guru/kelas/${action.item.kelas_id}/jadwal-pertemuan/${action.item.id}`;
		const data = action.type === "reschedule"
			? { tanggal_baru: tanggalBaru, jam_baru: jamBaru, alasan }
			: { guru_pengganti_id: Number(guruPenggantiID), alasan };
		router.post(`${base}/${action.type}`, data, {
			preserveScroll: true,
			onSuccess: (page) => {
				const responseFlash = page.props.flash as Flash | undefined;
				if (!responseFlash?.error) closeAction();
			},
		});
	}

	function startSchedule(item: JadwalPertemuanGuru) {
		openMenuID = null;
		router.post(`/app/guru/kelas/${item.kelas_id}/jadwal-pertemuan/${item.id}/mulai`, {});
	}

	function cancelSchedule(item: JadwalPertemuanGuru) {
		openMenuID = null;
		if (!window.confirm(`Batalkan jadwal ${item.kelas_nama} pada ${formatDate(item.tanggal)}?`)) return;
		router.post(`/app/guru/kelas/${item.kelas_id}/jadwal-pertemuan/${item.id}/batal`, {}, { preserveScroll: true });
	}
</script>

<svelte:head><title>Agenda Mengajar</title></svelte:head>

<AppLayout {user} group="guru-jadwal">
	<div class="border-b border-neutral-200/80 pb-7 pt-6 sm:pb-9 sm:pt-8 dark:border-white/[0.04]">
		<div class="mx-auto max-w-6xl px-3 sm:px-6">
			<a href="/app/guru" use:inertia class="mb-4 inline-flex items-center gap-1.5 text-sm text-neutral-500 hover:text-brand-600 dark:hover:text-brand-400">
				<ArrowLeft class="h-4 w-4" /> Dashboard
			</a>
			<div class="flex flex-col items-stretch gap-4 sm:flex-row sm:items-end sm:justify-between">
				<div>
					<h1 class="text-2xl font-bold tracking-tight text-neutral-900 sm:text-3xl dark:text-white">Agenda Mengajar</h1>
					<p class="mt-2 max-w-2xl text-sm leading-6 text-neutral-600 sm:text-base dark:text-neutral-400">Pusat aktivitas mengajar. Mulai kelas, lanjutkan pertemuan, reschedule, badal, dan pembatalan dilakukan dari agenda ini.</p>
				</div>
				{#if kelas.length > 0}
					<button onclick={() => (showCreate = !showCreate)} class="inline-flex w-full items-center justify-center gap-2 rounded-xl border border-neutral-300 bg-white px-3.5 py-2.5 text-sm font-medium text-neutral-600 transition-colors hover:border-brand-400/50 hover:text-brand-700 sm:w-auto sm:py-2 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-300 dark:hover:text-brand-300">
						<Plus class="h-4 w-4" /> Sesi tambahan
					</button>
				{/if}
			</div>
		</div>
	</div>

	<div class="mx-auto max-w-6xl space-y-5 px-3 py-5 sm:space-y-6 sm:px-6 sm:py-8">
		{#if success || flash?.success}<div class="rounded-xl border border-green-500/20 bg-green-500/10 p-4 text-sm font-medium text-green-700 dark:text-green-400">{success || flash?.success}</div>{/if}
		{#if error || flash?.error}<div class="rounded-xl border border-red-500/20 bg-red-500/10 p-4 text-sm font-medium text-red-600 dark:text-red-400">{error || flash?.error}</div>{/if}

		<div class="grid grid-cols-2 gap-2 sm:gap-3 lg:grid-cols-4">
			<button onclick={() => (agendaFilter = "attention")} class="min-h-[102px] rounded-2xl border p-3 text-left transition-all sm:min-h-0 sm:p-4 {agendaFilter === 'attention' ? 'border-red-500/35 bg-red-500/5 ring-1 ring-red-500/10' : 'border-neutral-200/80 bg-white hover:border-red-500/25 dark:border-white/[0.06] dark:bg-neutral-925/50'}">
				<div class="flex items-center justify-between gap-2">
					<span class="flex h-8 w-8 items-center justify-center rounded-xl sm:h-9 sm:w-9 bg-red-500/10 text-red-600 dark:text-red-400"><AlertTriangle class="h-4 w-4" /></span>
					<span class="font-mono text-xl font-bold text-neutral-900 sm:text-2xl dark:text-white">{attentionItems.length}</span>
				</div>
				<p class="mt-2 text-[11px] font-semibold leading-tight text-neutral-700 sm:text-xs dark:text-neutral-300">Perlu tindakan</p>
			</button>
			<button onclick={() => (agendaFilter = "today")} class="min-h-[102px] rounded-2xl border p-3 text-left transition-all sm:min-h-0 sm:p-4 {agendaFilter === 'today' ? 'border-brand-400/40 bg-brand-400/5 ring-1 ring-brand-400/10' : 'border-neutral-200/80 bg-white hover:border-brand-400/30 dark:border-white/[0.06] dark:bg-neutral-925/50'}">
				<div class="flex items-center justify-between gap-2">
					<span class="flex h-8 w-8 items-center justify-center rounded-xl sm:h-9 sm:w-9 bg-brand-400/10 text-brand-600 dark:text-brand-400"><CalendarClock class="h-4 w-4" /></span>
					<span class="font-mono text-xl font-bold text-neutral-900 sm:text-2xl dark:text-white">{todayItems.length}</span>
				</div>
				<p class="mt-2 text-[11px] font-semibold leading-tight text-neutral-700 sm:text-xs dark:text-neutral-300">Hari ini</p>
			</button>
			<button onclick={() => (agendaFilter = "week")} class="min-h-[102px] rounded-2xl border p-3 text-left transition-all sm:min-h-0 sm:p-4 {agendaFilter === 'week' ? 'border-sky-500/35 bg-sky-500/5 ring-1 ring-sky-500/10' : 'border-neutral-200/80 bg-white hover:border-sky-500/25 dark:border-white/[0.06] dark:bg-neutral-925/50'}">
				<div class="flex items-center justify-between gap-2">
					<span class="flex h-8 w-8 items-center justify-center rounded-xl sm:h-9 sm:w-9 bg-sky-500/10 text-sky-600 dark:text-sky-400"><CalendarDays class="h-4 w-4" /></span>
					<span class="font-mono text-xl font-bold text-neutral-900 sm:text-2xl dark:text-white">{weekItems.length}</span>
				</div>
				<p class="mt-2 text-[11px] font-semibold leading-tight text-neutral-700 sm:text-xs dark:text-neutral-300">7 hari ke depan</p>
			</button>
			<button onclick={() => (agendaFilter = "all")} class="min-h-[102px] rounded-2xl border p-3 text-left transition-all sm:min-h-0 sm:p-4 {agendaFilter === 'all' ? 'border-neutral-500/35 bg-neutral-500/5 ring-1 ring-neutral-500/10' : 'border-neutral-200/80 bg-white hover:border-neutral-400/40 dark:border-white/[0.06] dark:bg-neutral-925/50'}">
				<div class="flex items-center justify-between gap-2">
					<span class="flex h-8 w-8 items-center justify-center rounded-xl sm:h-9 sm:w-9 bg-neutral-500/10 text-neutral-600 dark:text-neutral-400"><BookOpen class="h-4 w-4" /></span>
					<span class="font-mono text-xl font-bold text-neutral-900 sm:text-2xl dark:text-white">{jadwal.length}</span>
				</div>
				<p class="mt-2 text-[11px] font-semibold leading-tight text-neutral-700 sm:text-xs dark:text-neutral-300">Semua agenda</p>
			</button>
		</div>

		<div class="flex flex-col gap-3 rounded-2xl border border-neutral-200/80 bg-white p-3.5 sm:flex-row sm:items-center sm:justify-between sm:p-4 dark:border-white/[0.06] dark:bg-neutral-925/50">
			<div>
				<p class="text-sm font-semibold text-neutral-900 dark:text-white">Menampilkan {filterLabel()}</p>
				<p class="mt-0.5 text-xs text-neutral-500">{visibleItems.length} agenda ditemukan</p>
			</div>
			<label class="flex w-full flex-col items-stretch gap-1.5 text-sm text-neutral-600 sm:w-auto sm:flex-row sm:items-center sm:gap-2 dark:text-neutral-300">
				<span class="shrink-0">Kelas</span>
				<select bind:value={kelasFilter} class="w-full min-w-0 rounded-xl border border-neutral-300 bg-white px-3 py-2.5 text-sm sm:w-auto sm:min-w-44 sm:py-2 dark:border-neutral-700 dark:bg-neutral-900 dark:text-white">
					<option value={0}>Semua kelas</option>
					{#each kelas as item}<option value={item.id}>{item.nama_kelas}</option>{/each}
				</select>
			</label>
		</div>

		{#if showCreate}
			<section class="rounded-2xl border border-neutral-300 bg-neutral-50/70 p-4 sm:p-6 dark:border-neutral-700 dark:bg-neutral-900/40">
				<div class="mb-5">
					<h2 class="font-semibold text-neutral-900 dark:text-white">Sesi tambahan</h2>
					<p class="mt-1 text-sm text-neutral-500 dark:text-neutral-400">Hanya untuk pertemuan di luar pola rutin. Agenda rutin utama dibuat otomatis oleh sistem.</p>
				</div>
				<div class="grid gap-4 md:grid-cols-3">
					<label class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Kelas
						<select bind:value={kelasID} class="mt-1.5 w-full rounded-xl border border-neutral-300 bg-white px-3 py-2.5 text-sm dark:border-neutral-700 dark:bg-neutral-900 dark:text-white">
							{#each kelas as item}<option value={item.id}>{item.nama_kelas}</option>{/each}
						</select>
					</label>
					<label class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Tanggal
						<input type="date" min={today} bind:value={tanggal} class="mt-1.5 w-full rounded-xl border border-neutral-300 bg-white px-3 py-2.5 text-sm dark:border-neutral-700 dark:bg-neutral-900 dark:text-white" />
					</label>
					<label class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Jam mulai
						<input type="time" bind:value={jamMulai} class="mt-1.5 w-full rounded-xl border border-neutral-300 bg-white px-3 py-2.5 text-sm dark:border-neutral-700 dark:bg-neutral-900 dark:text-white" />
					</label>
				</div>
				<label class="mt-4 block text-sm font-medium text-neutral-700 dark:text-neutral-300">Catatan <span class="font-normal text-neutral-400">(opsional)</span>
					<textarea bind:value={catatan} rows="2" placeholder="Topik atau informasi persiapan sesi" class="mt-1.5 w-full resize-none rounded-xl border border-neutral-300 bg-white px-3 py-2.5 text-sm dark:border-neutral-700 dark:bg-neutral-900 dark:text-white"></textarea>
				</label>
				<div class="mt-5 flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
					<button onclick={() => (showCreate = false)} class="w-full rounded-xl px-4 py-2.5 text-sm font-medium sm:w-auto text-neutral-600 hover:bg-neutral-200/70 dark:text-neutral-300 dark:hover:bg-neutral-800">Batal</button>
					<button onclick={createSchedule} disabled={submitting || !kelasID || !tanggal || !jamMulai} class="w-full rounded-xl bg-neutral-800 px-4 py-2.5 sm:w-auto text-sm font-semibold text-white hover:bg-neutral-900 disabled:opacity-50 dark:bg-neutral-200 dark:text-neutral-900 dark:hover:bg-white">{submitting ? "Menyimpan..." : "Tambah sesi"}</button>
				</div>
			</section>
		{/if}

		{#if groupedItems.length > 0}
			<div class="space-y-7">
				{#each groupedItems as [date, items] (date)}
					<section>
						<div class="mb-3 flex flex-wrap items-center gap-x-3 gap-y-1">
							<h2 class="min-w-0 text-xs font-bold uppercase tracking-wide text-neutral-700 sm:text-sm dark:text-neutral-300">{dateGroupLabel(date)}</h2>
							<span class="shrink-0 text-xs text-neutral-400">{items.length} kelas</span>
							<div class="hidden h-px flex-1 bg-neutral-200/80 sm:block dark:bg-white/[0.05]"></div>
						</div>

						<div class="grid gap-3 lg:grid-cols-2">
							{#each items as item (item.id)}
								{@const status = statusInfo(item)}
								<article class="relative rounded-2xl border p-4 sm:p-5 {cardClass(item)}">
									<div class="flex items-start justify-between gap-3 sm:gap-4">
										<div class="min-w-0">
											<div class="flex flex-wrap items-center gap-2">
												<span class="font-mono text-xl font-bold text-neutral-900 dark:text-white">{item.jam_mulai}</span>
												<span class="rounded-full px-2.5 py-1 text-[11px] font-semibold {status.cls}">{status.label}</span>
											</div>
											<h3 class="mt-2 break-words text-base font-semibold text-neutral-900 dark:text-white">{item.kelas_nama}</h3>
											<p class="mt-1 text-xs text-neutral-500">Guru utama: {item.guru_utama_nama || "Belum ditetapkan"}</p>
										</div>

										{#if item.can_manage}
											<div class="relative">
												<button onclick={() => (openMenuID = openMenuID === item.id ? null : item.id)} aria-label="Aksi lainnya" class="flex h-10 w-10 items-center justify-center rounded-xl border border-neutral-200 bg-white text-neutral-500 hover:border-neutral-300 hover:text-neutral-800 dark:border-neutral-700 dark:bg-neutral-900 dark:hover:text-white">
													<MoreHorizontal class="h-4 w-4" />
												</button>
												{#if openMenuID === item.id}
													<div class="fixed inset-x-3 bottom-3 z-40 w-auto overflow-hidden rounded-2xl border border-neutral-200 bg-white p-2 shadow-2xl sm:absolute sm:inset-auto sm:right-0 sm:bottom-auto sm:mt-2 sm:w-48 sm:rounded-xl sm:p-1.5 sm:shadow-xl dark:border-neutral-700 dark:bg-neutral-900">
														<button onclick={() => openAction("reschedule", item)} class="flex min-h-11 w-full items-center gap-2 rounded-lg px-3 py-2 text-left text-sm text-neutral-700 hover:bg-neutral-100 dark:text-neutral-200 dark:hover:bg-neutral-800"><RotateCcw class="h-4 w-4 text-blue-500" /> Reschedule</button>
														<button onclick={() => openAction("badal", item)} class="flex min-h-11 w-full items-center gap-2 rounded-lg px-3 py-2 text-left text-sm text-neutral-700 hover:bg-neutral-100 dark:text-neutral-200 dark:hover:bg-neutral-800"><UserRoundCheck class="h-4 w-4 text-secondary-500" /> Tetapkan badal</button>
														<button onclick={() => cancelSchedule(item)} class="flex min-h-11 w-full items-center gap-2 rounded-lg px-3 py-2 text-left text-sm text-red-600 hover:bg-red-500/10 dark:text-red-400"><Ban class="h-4 w-4" /> Batalkan kelas</button>
													</div>
												{/if}
											</div>
										{/if}
									</div>

									{#if item.catatan}<p class="mt-3 rounded-xl bg-neutral-500/5 px-3 py-2 text-xs text-neutral-600 dark:text-neutral-400">{item.catatan}</p>{/if}

									{#if item.is_reschedule || item.guru_pengganti_id || item.jadwal_kelas_berubah}
										<div class="mt-3 flex flex-wrap gap-2 text-xs">
											{#if item.jadwal_kelas_berubah}<span class="inline-flex items-center gap-1 rounded-full bg-amber-500/10 px-2.5 py-1 text-amber-700 dark:text-amber-400"><CalendarClock class="h-3.5 w-3.5" /> Jadwal kelas berubah</span>{/if}
											{#if item.is_reschedule}<span class="inline-flex items-center gap-1 rounded-full bg-blue-500/10 px-2.5 py-1 text-blue-700 dark:text-blue-400"><RotateCcw class="h-3.5 w-3.5" /> Reschedule dari {item.jadwal_semula}</span>{/if}
											{#if item.guru_pengganti_id}<span class="inline-flex items-center gap-1 rounded-full bg-secondary-500/10 px-2.5 py-1 text-secondary-700 dark:text-secondary-400"><UserRoundCheck class="h-3.5 w-3.5" /> Badal: {item.guru_pengganti_nama}</span>{/if}
										</div>
									{/if}

									<div class="mt-4 border-t border-neutral-200/70 pt-4 dark:border-white/[0.05]">
										{#if item.status === "dimulai" && item.pertemuan_id}
											<a href={"/app/guru/kelas/" + item.kelas_id + "/pertemuan/" + item.pertemuan_id + "/selesai"} use:inertia class="inline-flex w-full items-center justify-center gap-2 rounded-xl bg-amber-600 px-4 py-3 sm:py-2.5 text-sm font-semibold text-white hover:bg-amber-700 sm:w-auto">
												<Play class="h-4 w-4" /> Lanjutkan Kelas
											</a>
										{:else if item.can_start && item.tanggal <= today}
											<button onclick={() => startSchedule(item)} class="inline-flex w-full items-center justify-center gap-2 rounded-xl bg-brand-600 px-4 py-3 sm:py-2.5 text-sm font-semibold text-white shadow-sm hover:bg-brand-700 dark:bg-brand-500 dark:hover:bg-brand-400 sm:w-auto">
												<Play class="h-4 w-4" /> Mulai Kelas
											</button>
										{:else}
											<p class="text-xs font-medium text-neutral-400">Belum dapat dimulai sebelum tanggal jadwal.</p>
										{/if}
									</div>
								</article>
							{/each}
						</div>
					</section>
				{/each}
			</div>
		{:else if !showCreate}
			<div class="rounded-2xl border border-neutral-200/80 bg-white p-8 text-center sm:p-12 dark:border-white/[0.06] dark:bg-neutral-925/50">
				<BookOpen class="mx-auto h-8 w-8 text-neutral-400" />
				<h2 class="mt-4 font-semibold text-neutral-900 dark:text-white">Tidak ada agenda {filterLabel()}</h2>
				<p class="mx-auto mt-1 max-w-xl text-sm text-neutral-500">Pilih filter lain atau kelas lain. Jadwal rutin yang sudah diatur Admin Kelas akan muncul otomatis di sini.</p>
			</div>
		{/if}
	</div>

	{#if action}
		<div class="fixed inset-0 z-50 flex items-end justify-center sm:items-center sm:p-4">
			<button class="absolute inset-0 bg-black/50" aria-label="Tutup dialog" onclick={closeAction}></button>
			<div class="relative max-h-[90vh] w-full overflow-y-auto rounded-t-3xl bg-white p-5 shadow-xl sm:max-w-md sm:rounded-2xl dark:bg-neutral-925" role="dialog" aria-modal="true" aria-labelledby="action-title" tabindex="-1">
				<h2 id="action-title" class="text-lg font-semibold text-neutral-900 dark:text-white">{action.type === "reschedule" ? "Reschedule Pertemuan" : "Tetapkan Guru Badal"}</h2>
				<p class="mt-1 text-sm text-neutral-500">{action.item.kelas_nama}</p>
				<div class="mt-5 space-y-4">
					{#if action.type === "reschedule"}
						<label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300">Tanggal baru<input type="date" min={today} bind:value={tanggalBaru} class="mt-1.5 w-full rounded-xl border border-neutral-300 bg-neutral-50 px-3 py-2.5 text-sm dark:border-neutral-700 dark:bg-neutral-800/50 dark:text-white" /></label>
						<label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300">Jam baru<input type="time" bind:value={jamBaru} class="mt-1.5 w-full rounded-xl border border-neutral-300 bg-neutral-50 px-3 py-2.5 text-sm dark:border-neutral-700 dark:bg-neutral-800/50 dark:text-white" /></label>
					{:else}
						<label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300">Guru pengganti<select bind:value={guruPenggantiID} class="mt-1.5 w-full rounded-xl border border-neutral-300 bg-neutral-50 px-3 py-2.5 text-sm dark:border-neutral-700 dark:bg-neutral-800/50 dark:text-white"><option value="">Pilih guru aktif</option>{#each availableGuru as guru}<option value={guru.id}>{guru.nama}</option>{/each}</select></label>
					{/if}
					<label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300">Alasan <span class="font-normal text-neutral-400">(opsional)</span><textarea bind:value={alasan} rows="3" class="mt-1.5 w-full resize-none rounded-xl border border-neutral-300 bg-neutral-50 px-3 py-2.5 text-sm dark:border-neutral-700 dark:bg-neutral-800/50 dark:text-white"></textarea></label>
					<div class="flex justify-end gap-2">
						<button onclick={closeAction} class="rounded-xl px-4 py-2 text-sm text-neutral-600 dark:text-neutral-300">Batal</button>
						<button onclick={submitAction} disabled={action.type === "reschedule" ? !tanggalBaru || !jamBaru : !guruPenggantiID} class="rounded-xl bg-brand-600 px-4 py-2 text-sm font-semibold text-white disabled:opacity-50">Simpan</button>
					</div>
				</div>
			</div>
		</div>
	{/if}
</AppLayout>
