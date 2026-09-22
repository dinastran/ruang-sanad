<script lang="ts">
	import { tick } from "svelte";
	import { inertia, router } from "@inertiajs/svelte";
	import AppLayout from "@layouts/AppLayout.svelte";
	import type {
		Flash,
		GuruRingkas,
		KelasSimple,
		MonitoringKelasData,
		MonitoringKelasFilters,
		MonitoringKelasItem,
		User,
	} from "@lib/types";
	import {
		AlertTriangle,
		ArrowLeft,
		Ban,
		BellRing,
		BookOpen,
		CalendarClock,
		CheckCircle2,
		ChevronDown,
		Clock3,
		Filter,
		History,
		MessageSquarePlus,
		PlayCircle,
		RotateCcw,
		Send,
		UserRoundCheck,
		UsersRound,
		X,
	} from "lucide-svelte";

	type ActionType = "reminder_teacher" | "reminder_students" | "note" | "reschedule" | "substitute" | "cancel";
	type ActionModal = { type: ActionType; item: MonitoringKelasItem };

	interface Props {
		user?: User;
		flash?: Flash;
		error?: string;
		monitoring?: MonitoringKelasData;
		guru?: GuruRingkas[];
		kelas?: KelasSimple[];
		filters?: MonitoringKelasFilters;
	}

	let { user, flash, error, monitoring, guru = [], kelas = [], filters }: Props = $props();

	const emptySummary = { total: 0, belum_mulai: 0, berlangsung: 0, selesai: 0, dibatalkan: 0, perlu_tindakan: 0 };
	const inputClass = "mt-1.5 w-full rounded-xl border border-neutral-300 bg-neutral-50 px-3 py-2.5 text-sm text-neutral-900 dark:border-neutral-700 dark:bg-neutral-800/60 dark:text-white";
	const statusLabel: Record<string, string> = {
		belum_mulai: "Belum mulai",
		berlangsung: "Berlangsung",
		selesai: "Selesai",
		dibatalkan: "Dibatalkan",
		terlambat: "Terlambat",
	};
	const attendanceLabel: Record<string, string> = {
		belum_hadir: "Belum hadir",
		hadir: "Hadir",
		terlambat: "Terlambat",
		tidak_berlaku: "Tidak berlaku",
		belum_tersedia: "Belum tersedia",
		belum_lengkap: "Belum lengkap",
		lengkap: "Lengkap",
		izin: "Izin",
		sakit: "Sakit",
		alpa: "Alpa",
		telat: "Terlambat",
	};

	function initialFilters() {
		return {
			startDate: filters?.StartDate ?? "",
			endDate: filters?.EndDate ?? "",
			guruID: filters?.GuruID ? String(filters.GuruID) : "",
			kelasID: filters?.KelasID ? String(filters.KelasID) : "",
			status: filters?.Status ?? "",
		};
	}

	const initial = initialFilters();
	let data = $derived(monitoring ?? { summary: emptySummary, items: [] });
	let startDate = $state(initial.startDate);
	let endDate = $state(initial.endDate);
	let guruID = $state(initial.guruID);
	let kelasID = $state(initial.kelasID);
	let status = $state(initial.status);
	let filtering = $state(false);
	let action = $state<ActionModal | null>(null);
	let actionBusy = $state(false);
	let dialogElement = $state<HTMLDivElement>();
	let message = $state("");
	let note = $state("");
	let tanggalBaru = $state("");
	let jamBaru = $state("");
	let alasan = $state("");
	let guruPenggantiID = $state("");
	let reason = $state("");

	let availableGuru = $derived(guru.filter((item) => item.id !== action?.item.guru_utama_id));

	function formatDate(value: string): string {
		if (!value) return "Tanggal belum tersedia";
		const date = new Date(`${value.slice(0, 10)}T00:00:00`);
		if (Number.isNaN(date.getTime())) return value;
		return new Intl.DateTimeFormat("id-ID", { weekday: "short", day: "numeric", month: "short", year: "numeric" }).format(date);
	}

	function formatDateTime(value: string): string {
		if (!value) return "-";
		const date = new Date(value.replace(" ", "T"));
		if (Number.isNaN(date.getTime())) return value;
		return new Intl.DateTimeFormat("id-ID", { day: "numeric", month: "short", year: "numeric", hour: "2-digit", minute: "2-digit" }).format(date);
	}

	function statusClass(value: string): string {
		if (value === "selesai") return "bg-success/10 text-emerald-700 dark:text-emerald-400";
		if (value === "berlangsung") return "bg-info/10 text-blue-700 dark:text-blue-400";
		if (value === "terlambat") return "bg-warning/10 text-amber-700 dark:text-amber-400";
		if (value === "dibatalkan") return "bg-error/10 text-red-700 dark:text-red-400";
		return "bg-neutral-200/70 text-neutral-700 dark:bg-neutral-800 dark:text-neutral-300";
	}

	function attendanceClass(value: string): string {
		if (value === "hadir" || value === "lengkap") return "bg-success/10 text-emerald-700 dark:text-emerald-400";
		if (value === "terlambat" || value === "telat" || value === "belum_lengkap") return "bg-warning/10 text-amber-700 dark:text-amber-400";
		if (value === "alpa" || value === "belum_hadir") return "bg-error/10 text-red-700 dark:text-red-400";
		return "bg-neutral-200/70 text-neutral-700 dark:bg-neutral-800 dark:text-neutral-300";
	}

	function teacherName(item: MonitoringKelasItem): string {
		return item.guru_pengganti_nama || item.guru_utama_nama || "Belum ditetapkan";
	}

	function runFilter(reset = false) {
		if (reset) {
			startDate = "";
			endDate = "";
			guruID = "";
			kelasID = "";
			status = "";
		}
		const params = new URLSearchParams();
		if (!reset && startDate) params.set("start_date", startDate);
		if (!reset && endDate) params.set("end_date", endDate);
		if (!reset && guruID) params.set("guru_id", guruID);
		if (!reset && kelasID) params.set("kelas_id", kelasID);
		if (!reset && status) params.set("status", status);
		const query = params.toString();
		filtering = true;
		router.get(`/app/koordinator-guru/monitoring-kelas${query ? `?${query}` : ""}`, {}, {
			replace: true,
			onFinish: () => (filtering = false),
		});
	}

	async function openAction(type: ActionType, item: MonitoringKelasItem) {
		action = { type, item };
		message = "";
		note = "";
		tanggalBaru = item.tanggal;
		jamBaru = item.jam_mulai;
		alasan = type === "reschedule" ? item.alasan_reschedule : type === "substitute" ? item.alasan_badal : "";
		guruPenggantiID = item.guru_pengganti_id ? String(item.guru_pengganti_id) : "";
		reason = "";
		await tick();
		dialogElement?.focus();
	}

	function closeAction() {
		if (!actionBusy) action = null;
	}

	function actionTitle(type: ActionType): string {
		return {
			reminder_teacher: "Ingatkan Presensi Guru",
			reminder_students: "Ingatkan Absensi Peserta",
			note: "Tambah Catatan Koordinator",
			reschedule: "Jadwalkan Ulang Sesi",
			substitute: "Tetapkan Guru Pengganti",
			cancel: "Batalkan Jadwal Kelas",
		}[type];
	}

	function canSubmit(): boolean {
		if (!action) return false;
		if ((action.type === "reminder_teacher" || action.type === "reminder_students") && !action.item.assigned_user_id) return false;
		if (action.type === "note") return note.trim().length > 0;
		if (action.type === "reschedule") return Boolean(tanggalBaru && jamBaru);
		if (action.type === "substitute") return Boolean(guruPenggantiID);
		if (action.type === "cancel") return reason.trim().length > 0;
		return true;
	}

	function submitAction() {
		if (!action || !canSubmit()) return;
		const base = `/app/koordinator-guru/monitoring-kelas/${action.item.kelas_id}/schedules/${action.item.schedule_id}`;
		let endpoint = "";
		let payload: Record<string, string | number> = {};
		if (action.type === "reminder_teacher" || action.type === "reminder_students") {
			endpoint = "reminder";
			payload = { kind: action.type === "reminder_teacher" ? "presensi_guru" : "absensi_peserta", message };
		} else if (action.type === "note") {
			endpoint = "notes";
			payload = { note: note.trim() };
		} else if (action.type === "reschedule") {
			endpoint = "reschedule";
			payload = { tanggal_baru: tanggalBaru, jam_baru: jamBaru, alasan };
		} else if (action.type === "substitute") {
			endpoint = "substitute";
			payload = { guru_pengganti_id: Number(guruPenggantiID), alasan };
		} else {
			endpoint = "cancel";
			payload = { reason: reason.trim() };
		}

		actionBusy = true;
		router.post(`${base}/${endpoint}`, payload, {
			preserveScroll: true,
			// Keep the modal (and typed input) mounted so a server error can be fixed and resubmitted.
			preserveState: true,
			onSuccess: (page) => {
				const responseFlash = page.props.flash as Flash | undefined;
				if (!responseFlash?.error) action = null;
			},
			onFinish: () => (actionBusy = false),
		});
	}
</script>

<svelte:window onkeydown={(event) => event.key === "Escape" && closeAction()} />

<AppLayout {user} group="koordinator-monitoring">
	<header class="border-b border-neutral-200/80 pt-8 pb-9 dark:border-white/[0.04]">
		<div class="mx-auto max-w-7xl px-4 sm:px-6">
			<a href="/app/koordinator-guru" use:inertia class="mb-4 inline-flex items-center gap-1.5 text-sm text-neutral-500 hover:text-brand-600 dark:hover:text-brand-400">
				<ArrowLeft class="h-4 w-4" /> Dashboard koordinator
			</a>
			<h1 class="text-2xl font-bold tracking-tight text-neutral-900 sm:text-3xl dark:text-white">Monitoring Kelas</h1>
			<p class="mt-2 max-w-2xl text-neutral-600 dark:text-neutral-400">Pantau kesiapan guru, pelaksanaan sesi, dan kelengkapan absensi dalam satu tampilan operasional.</p>
		</div>
	</header>

	<main class="mx-auto max-w-7xl space-y-6 px-4 py-7 sm:px-6">
		{#if flash?.success}
			<div role="status" class="rounded-xl border border-success/20 bg-success/10 p-4 text-sm font-medium text-emerald-700 dark:text-emerald-400">{flash.success}</div>
		{/if}
		{#if error || flash?.error}
			<div role="alert" class="rounded-xl border border-error/20 bg-error/10 p-4 text-sm font-medium text-red-700 dark:text-red-400">{error || flash?.error}</div>
		{/if}

		<section aria-labelledby="summary-title">
			<div class="mb-3 flex items-center justify-between gap-3">
				<h2 id="summary-title" class="text-sm font-semibold text-neutral-900 dark:text-white">Ringkasan operasional</h2>
				<p class="text-xs text-neutral-500">{startDate || "Hari ini"}{#if endDate && endDate !== startDate} sampai {endDate}{/if}</p>
			</div>
			<div class="grid grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-6">
				<div class="rounded-xl border border-neutral-200/80 bg-white p-4 dark:border-white/[0.06] dark:bg-neutral-925/50"><CalendarClock class="h-4 w-4 text-neutral-500" /><p class="mt-3 text-2xl font-bold text-neutral-900 dark:text-white">{data.summary.total}</p><p class="text-xs text-neutral-500">Total sesi</p></div>
				<div class="rounded-xl border border-error/25 bg-error/5 p-4"><AlertTriangle class="h-4 w-4 text-red-600 dark:text-red-400" /><p class="mt-3 text-2xl font-bold text-red-700 dark:text-red-300">{data.summary.perlu_tindakan}</p><p class="text-xs font-medium text-red-700 dark:text-red-300">Perlu tindakan</p></div>
				<div class="rounded-xl border border-neutral-200/80 bg-white p-4 dark:border-white/[0.06] dark:bg-neutral-925/50"><Clock3 class="h-4 w-4 text-neutral-500" /><p class="mt-3 text-2xl font-bold text-neutral-900 dark:text-white">{data.summary.belum_mulai}</p><p class="text-xs text-neutral-500">Belum mulai</p></div>
				<div class="rounded-xl border border-info/20 bg-info/5 p-4"><PlayCircle class="h-4 w-4 text-blue-600 dark:text-blue-400" /><p class="mt-3 text-2xl font-bold text-neutral-900 dark:text-white">{data.summary.berlangsung}</p><p class="text-xs text-neutral-500">Berlangsung</p></div>
				<div class="rounded-xl border border-success/20 bg-success/5 p-4"><CheckCircle2 class="h-4 w-4 text-emerald-600 dark:text-emerald-400" /><p class="mt-3 text-2xl font-bold text-neutral-900 dark:text-white">{data.summary.selesai}</p><p class="text-xs text-neutral-500">Selesai</p></div>
				<div class="rounded-xl border border-neutral-200/80 bg-white p-4 dark:border-white/[0.06] dark:bg-neutral-925/50"><Ban class="h-4 w-4 text-neutral-500" /><p class="mt-3 text-2xl font-bold text-neutral-900 dark:text-white">{data.summary.dibatalkan}</p><p class="text-xs text-neutral-500">Dibatalkan</p></div>
			</div>
		</section>

		<form onsubmit={(event) => { event.preventDefault(); runFilter(); }} class="rounded-xl border border-neutral-200/80 bg-white p-4 dark:border-white/[0.06] dark:bg-neutral-925/50" aria-label="Filter monitoring kelas">
			<div class="mb-4 flex items-center gap-2"><Filter class="h-4 w-4 text-brand-600 dark:text-brand-400" /><h2 class="text-sm font-semibold text-neutral-900 dark:text-white">Filter sesi</h2></div>
			<div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-5">
				<label class="text-xs font-medium text-neutral-600 dark:text-neutral-300">Tanggal mulai<input type="date" bind:value={startDate} class={inputClass} /></label>
				<label class="text-xs font-medium text-neutral-600 dark:text-neutral-300">Tanggal selesai<input type="date" min={startDate || undefined} bind:value={endDate} class={inputClass} /></label>
				<label class="text-xs font-medium text-neutral-600 dark:text-neutral-300">Guru<select bind:value={guruID} class={inputClass}><option value="">Semua guru</option>{#each guru as item (item.id)}<option value={String(item.id)}>{item.nama}</option>{/each}</select></label>
				<label class="text-xs font-medium text-neutral-600 dark:text-neutral-300">Kelas<select bind:value={kelasID} class={inputClass}><option value="">Semua kelas</option>{#each kelas as item (item.id)}<option value={String(item.id)}>{item.nama_kelas}</option>{/each}</select></label>
				<label class="text-xs font-medium text-neutral-600 dark:text-neutral-300">Status<select bind:value={status} class={inputClass}><option value="">Semua status</option><option value="belum_mulai">Belum mulai</option><option value="berlangsung">Berlangsung</option><option value="selesai">Selesai</option><option value="dibatalkan">Dibatalkan</option><option value="terlambat">Terlambat</option></select></label>
			</div>
			<div class="mt-4 flex flex-wrap justify-end gap-2 border-t border-neutral-200/70 pt-4 dark:border-white/[0.05]">
				<button type="button" onclick={() => runFilter(true)} disabled={filtering} class="rounded-lg px-4 py-2 text-sm font-medium text-neutral-600 hover:bg-neutral-100 disabled:opacity-50 dark:text-neutral-300 dark:hover:bg-neutral-800">Reset</button>
				<button type="submit" disabled={filtering} class="rounded-lg bg-brand-600 px-4 py-2 text-sm font-semibold text-white hover:bg-brand-700 disabled:opacity-50 dark:bg-brand-500 dark:hover:bg-brand-400">{filtering ? "Memuat..." : "Terapkan filter"}</button>
			</div>
		</form>

		<section aria-labelledby="session-list-title" aria-busy={filtering}>
			<div class="mb-3 flex items-center justify-between gap-3">
				<h2 id="session-list-title" class="text-sm font-semibold text-neutral-900 dark:text-white">Daftar sesi</h2>
				<span class="text-xs text-neutral-500">{data.items.length} sesi ditampilkan</span>
			</div>

			{#if filtering}
				<div class="space-y-3" aria-label="Memuat sesi">
					{#each [1, 2, 3] as row}
						<div class="animate-pulse rounded-xl border border-neutral-200/80 bg-white p-5 dark:border-white/[0.06] dark:bg-neutral-925/50"><div class="h-4 w-1/3 rounded bg-neutral-200 dark:bg-neutral-800"></div><div class="mt-3 h-3 w-2/3 rounded bg-neutral-100 dark:bg-neutral-800/70"></div><div class="mt-5 h-9 rounded bg-neutral-100 dark:bg-neutral-800/70"></div></div>
					{/each}
				</div>
			{:else if data.items.length === 0}
				<div class="rounded-xl border border-dashed border-neutral-300 bg-white p-10 text-center dark:border-neutral-700 dark:bg-neutral-925/40">
					<CalendarClock class="mx-auto h-8 w-8 text-neutral-400" />
					<h3 class="mt-3 font-semibold text-neutral-900 dark:text-white">Tidak ada sesi pada rentang ini</h3>
					<p class="mt-1 text-sm text-neutral-500">Ubah tanggal atau longgarkan filter guru, kelas, dan status.</p>
				</div>
			{:else}
				<div class="space-y-3">
					{#each data.items as item (item.tanpa_jadwal ? `p-${item.pertemuan_id}` : `j-${item.schedule_id}`)}
						<article class="overflow-hidden rounded-xl border bg-white dark:bg-neutral-925/50 {item.needs_action ? 'border-error/35' : 'border-neutral-200/80 dark:border-white/[0.06]'}">
							<div class="p-4 sm:p-5">
								<div class="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
									<div class="min-w-0">
										<div class="flex flex-wrap items-center gap-2">
											<h3 class="font-semibold text-neutral-900 dark:text-white">{item.nama_kelas}</h3>
											<span class="rounded-full px-2.5 py-1 text-xs font-semibold {statusClass(item.status)}">{statusLabel[item.status] ?? item.status}</span>
											{#if item.tanpa_jadwal}<span class="rounded-full bg-neutral-200/70 px-2.5 py-1 text-xs font-semibold text-neutral-700 dark:bg-neutral-800 dark:text-neutral-300" title="Pertemuan dimulai langsung oleh guru tanpa jadwal pertemuan">Tanpa jadwal</span>{/if}
											{#if item.needs_action}<span class="inline-flex items-center gap-1 rounded-full bg-error/10 px-2.5 py-1 text-xs font-semibold text-red-700 dark:text-red-400"><AlertTriangle class="h-3.5 w-3.5" /> Perlu tindakan</span>{/if}
										</div>
										<p class="mt-1.5 text-sm font-medium text-neutral-700 dark:text-neutral-200">{formatDate(item.tanggal)} / {item.jam_mulai || "Jam belum tersedia"}</p>
										<p class="mt-1 text-xs text-neutral-500">{item.angkatan || "Angkatan belum tersedia"} / {item.level || "Level belum tersedia"} / {item.frekuensi || "Frekuensi belum tersedia"}</p>
									</div>
									<div class="grid shrink-0 grid-cols-2 gap-x-6 gap-y-2 text-xs sm:grid-cols-3 xl:min-w-[470px]">
										<div><p class="text-neutral-500">Guru bertugas</p><p class="mt-0.5 font-medium text-neutral-800 dark:text-neutral-200">{teacherName(item)}</p></div>
										<div><p class="text-neutral-500">Presensi guru</p><span class="mt-0.5 inline-flex rounded-full px-2 py-0.5 font-medium {attendanceClass(item.teacher_attendance)}">{attendanceLabel[item.teacher_attendance] ?? item.teacher_attendance}</span></div>
										<div><p class="text-neutral-500">Absensi peserta</p><span class="mt-0.5 inline-flex rounded-full px-2 py-0.5 font-medium {attendanceClass(item.student_attendance)}">{attendanceLabel[item.student_attendance] ?? item.student_attendance}</span></div>
										<div><p class="text-neutral-500">Peserta tercatat</p><p class="mt-0.5 font-medium text-neutral-800 dark:text-neutral-200">{item.attendance_count} / {item.active_student_count}</p></div>
										<div class="col-span-2"><p class="text-neutral-500">Jadwal kelas</p><p class="mt-0.5 font-medium text-neutral-800 dark:text-neutral-200">{item.jadwal_kelas || "-"}</p></div>
									</div>
								</div>

								{#if item.tanpa_jadwal}
									<p class="mt-4 border-t border-neutral-200/70 pt-4 text-xs text-neutral-500 dark:border-white/[0.05]">Pertemuan ini dimulai langsung oleh guru tanpa jadwal, sehingga pengingat, catatan, dan perubahan jadwal tidak tersedia.</p>
								{:else}
								<div class="mt-4 flex flex-wrap gap-2 border-t border-neutral-200/70 pt-4 dark:border-white/[0.05]">
									<button onclick={() => openAction("reminder_teacher", item)} disabled={!item.assigned_user_id || (item.status !== "belum_mulai" && item.status !== "terlambat")} title={!item.assigned_user_id ? "Akun guru belum terhubung" : item.status !== "belum_mulai" && item.status !== "terlambat" ? "Pertemuan sudah dimulai atau dibatalkan" : "Kirim pengingat presensi guru"} class="inline-flex items-center gap-1.5 rounded-lg border border-neutral-200 px-3 py-2 text-xs font-medium text-neutral-700 hover:border-brand-400/40 hover:text-brand-700 disabled:opacity-40 dark:border-neutral-700 dark:text-neutral-300 dark:hover:text-brand-300"><BellRing class="h-3.5 w-3.5" /> Presensi guru</button>
									<button onclick={() => openAction("reminder_students", item)} disabled={!item.assigned_user_id || item.status === "dibatalkan"} title={!item.assigned_user_id ? "Akun guru belum terhubung" : item.status === "dibatalkan" ? "Jadwal sudah dibatalkan" : "Kirim pengingat absensi peserta"} class="inline-flex items-center gap-1.5 rounded-lg border border-neutral-200 px-3 py-2 text-xs font-medium text-neutral-700 hover:border-brand-400/40 hover:text-brand-700 disabled:opacity-40 dark:border-neutral-700 dark:text-neutral-300 dark:hover:text-brand-300"><UsersRound class="h-3.5 w-3.5" /> Absensi peserta</button>
									<button onclick={() => openAction("note", item)} class="inline-flex items-center gap-1.5 rounded-lg border border-neutral-200 px-3 py-2 text-xs font-medium text-neutral-700 hover:border-brand-400/40 hover:text-brand-700 dark:border-neutral-700 dark:text-neutral-300 dark:hover:text-brand-300"><MessageSquarePlus class="h-3.5 w-3.5" /> Tambah catatan</button>
									{#if item.schedule_status === "dijadwalkan"}
										<button onclick={() => openAction("reschedule", item)} class="inline-flex items-center gap-1.5 rounded-lg border border-neutral-200 px-3 py-2 text-xs font-medium text-neutral-700 hover:border-brand-400/40 hover:text-brand-700 dark:border-neutral-700 dark:text-neutral-300 dark:hover:text-brand-300"><RotateCcw class="h-3.5 w-3.5" /> Jadwal ulang</button>
										<button onclick={() => openAction("substitute", item)} class="inline-flex items-center gap-1.5 rounded-lg border border-neutral-200 px-3 py-2 text-xs font-medium text-neutral-700 hover:border-brand-400/40 hover:text-brand-700 dark:border-neutral-700 dark:text-neutral-300 dark:hover:text-brand-300"><UserRoundCheck class="h-3.5 w-3.5" /> Guru pengganti</button>
										<button onclick={() => openAction("cancel", item)} class="inline-flex items-center gap-1.5 rounded-lg px-3 py-2 text-xs font-medium text-red-700 hover:bg-error/10 dark:text-red-400"><Ban class="h-3.5 w-3.5" /> Batalkan</button>
									{/if}
								</div>
								{#if !item.assigned_user_id}<p class="mt-2 text-xs text-amber-700 dark:text-amber-400">Pengingat tidak tersedia karena guru belum memiliki akun yang terhubung.</p>{/if}
								{/if}
							</div>

							<details class="group border-t border-neutral-200/70 dark:border-white/[0.05]">
								<summary class="flex min-h-11 items-center justify-between px-4 py-3 text-sm font-medium text-neutral-700 hover:bg-neutral-50 dark:text-neutral-300 dark:hover:bg-neutral-900/40 sm:px-5">
									Lihat detail sesi
									<ChevronDown class="h-4 w-4 transition-transform group-open:rotate-180" />
								</summary>
								<div class="grid grid-cols-[minmax(0,1fr)] gap-6 border-t border-neutral-200/70 px-4 py-5 dark:border-white/[0.05] sm:px-5 xl:grid-cols-[repeat(2,minmax(0,1fr))]">
									<section aria-label="Waktu pelaksanaan">
										<h4 class="text-xs font-bold uppercase tracking-wide text-neutral-500">Pelaksanaan</h4>
										<dl class="mt-3 grid grid-cols-2 gap-3 text-sm">
											<div><dt class="text-xs text-neutral-500">Mulai aktual</dt><dd class="mt-0.5 text-neutral-800 dark:text-neutral-200">{formatDateTime(item.actual_start_time)}</dd></div>
											<div><dt class="text-xs text-neutral-500">Selesai aktual</dt><dd class="mt-0.5 text-neutral-800 dark:text-neutral-200">{formatDateTime(item.actual_end_time)}</dd></div>
											<div><dt class="text-xs text-neutral-500">Check-in guru</dt><dd class="mt-0.5 text-neutral-800 dark:text-neutral-200">{formatDateTime(item.teacher_checked_in_at)}</dd></div>
											<div><dt class="text-xs text-neutral-500">Mode materi</dt><dd class="mt-0.5 text-neutral-800 dark:text-neutral-200">{item.materi_individual ? "Individual" : "Kelas"}</dd></div>
										</dl>
									</section>
									<section aria-label="Materi dan catatan pertemuan">
										<h4 class="flex items-center gap-1.5 text-xs font-bold uppercase tracking-wide text-neutral-500"><BookOpen class="h-3.5 w-3.5" /> Materi dan catatan</h4>
										<div class="mt-3 space-y-3 text-sm text-neutral-700 dark:text-neutral-300">
											<div><p class="text-xs text-neutral-500">Materi</p><p class="mt-0.5 whitespace-pre-wrap">{item.materi || "Belum dicatat"}</p></div>
											<div><p class="text-xs text-neutral-500">Catatan jadwal</p><p class="mt-0.5 whitespace-pre-wrap">{item.schedule_note || "Tidak ada"}</p></div>
											<div><p class="text-xs text-neutral-500">Catatan pertemuan</p><p class="mt-0.5 whitespace-pre-wrap">{item.meeting_note || "Tidak ada"}</p></div>
											{#if item.is_reschedule}<div><p class="text-xs text-neutral-500">Perubahan jadwal</p><p class="mt-0.5">Dari {item.jadwal_semula || "jadwal sebelumnya"}{#if item.alasan_reschedule}: {item.alasan_reschedule}{/if}</p></div>{/if}
											{#if item.guru_pengganti_id}<div><p class="text-xs text-neutral-500">Guru pengganti</p><p class="mt-0.5">{item.guru_pengganti_nama}{#if item.alasan_badal}: {item.alasan_badal}{/if}</p></div>{/if}
										</div>
									</section>

									<section class="min-w-0 xl:col-span-2" aria-label="Absensi peserta">
										<div class="flex items-center justify-between gap-3"><h4 class="flex items-center gap-1.5 text-xs font-bold uppercase tracking-wide text-neutral-500"><UsersRound class="h-3.5 w-3.5" /> Absensi peserta</h4><span class="text-xs text-neutral-500">{item.attendance_count} dari {item.active_student_count} tercatat</span></div>
										{#if item.attendance.length > 0}
											<div class="mt-3 max-w-full overflow-x-auto overscroll-x-contain rounded-lg border border-neutral-200/80 [-webkit-overflow-scrolling:touch] dark:border-white/[0.06]">
												<table class="w-max min-w-full text-left text-sm">
													<thead class="bg-neutral-50 text-xs text-neutral-500 dark:bg-neutral-900/60"><tr><th scope="col" class="px-3 py-2.5 font-medium">Peserta</th><th scope="col" class="px-3 py-2.5 font-medium">Status</th><th scope="col" class="px-3 py-2.5 font-medium">Batas materi</th><th scope="col" class="px-3 py-2.5 font-medium">Catatan</th></tr></thead>
													<tbody class="divide-y divide-neutral-200/70 dark:divide-white/[0.05]">
														{#each item.attendance as row (row.santri_id)}<tr><td class="min-w-40 px-3 py-2.5"><p class="font-medium text-neutral-800 dark:text-neutral-200">{row.santri_nama}</p><p class="text-xs text-neutral-500">{row.id_mahasantri || "-"}</p></td><td class="whitespace-nowrap px-3 py-2.5"><span class="rounded-full px-2 py-1 text-xs font-medium {attendanceClass(row.attendance_status)}">{attendanceLabel[row.attendance_status] ?? row.attendance_status}</span></td><td class="min-w-40 px-3 py-2.5 text-neutral-600 dark:text-neutral-300">{row.batas_materi || "-"}</td><td class="min-w-48 px-3 py-2.5 text-neutral-600 dark:text-neutral-300">{row.attendance_note || "-"}</td></tr>{/each}
													</tbody>
												</table>
											</div>
										{:else}<p class="mt-3 rounded-lg bg-neutral-100/70 p-3 text-sm text-neutral-500 dark:bg-neutral-800/40">Absensi peserta belum tersedia untuk sesi ini.</p>{/if}
									</section>

									{#if !item.tanpa_jadwal}
									<section aria-label="Catatan koordinator">
										<div class="flex items-center justify-between gap-3"><h4 class="flex items-center gap-1.5 text-xs font-bold uppercase tracking-wide text-neutral-500"><MessageSquarePlus class="h-3.5 w-3.5" /> Catatan koordinator</h4><button onclick={() => openAction("note", item)} class="text-xs font-semibold text-brand-700 hover:text-brand-600 dark:text-brand-300">Tambah catatan</button></div>
										<div class="mt-3 space-y-3">{#each item.notes as entry (entry.id)}<article class="border-l-2 border-brand-400/40 pl-3"><p class="whitespace-pre-wrap text-sm text-neutral-700 dark:text-neutral-300">{entry.note}</p><p class="mt-1 text-xs text-neutral-500">{entry.author_name || "Koordinator"} / {formatDateTime(entry.created_at)}</p></article>{:else}<p class="rounded-lg bg-neutral-100/70 p-3 text-sm text-neutral-500 dark:bg-neutral-800/40">Belum ada catatan tindak lanjut.</p>{/each}</div>
									</section>

									<section aria-label="Riwayat perubahan">
										<h4 class="flex items-center gap-1.5 text-xs font-bold uppercase tracking-wide text-neutral-500"><History class="h-3.5 w-3.5" /> Riwayat perubahan</h4>
										<div class="mt-3 space-y-3">{#each item.activities as entry (entry.id)}<article class="border-l border-neutral-300 pl-3 dark:border-neutral-700"><p class="text-sm font-medium text-neutral-800 dark:text-neutral-200">{entry.action}</p>{#if entry.details}<p class="mt-0.5 text-sm text-neutral-600 dark:text-neutral-400">{entry.details}</p>{/if}<p class="mt-1 text-xs text-neutral-500">{entry.actor_name || "Sistem"} / {formatDateTime(entry.created_at)}</p></article>{:else}<p class="rounded-lg bg-neutral-100/70 p-3 text-sm text-neutral-500 dark:bg-neutral-800/40">Belum ada perubahan pada jadwal ini.</p>{/each}</div>
									</section>
									{/if}
								</div>
							</details>
						</article>
					{/each}
				</div>
			{/if}
		</section>
	</main>

	{#if action}
		<div class="fixed inset-0 z-50 flex items-center justify-center p-4">
			<button class="absolute inset-0 bg-neutral-950/65" aria-label="Tutup dialog" onclick={closeAction}></button>
			<div bind:this={dialogElement} class="relative max-h-[90dvh] w-full max-w-md overflow-y-auto rounded-2xl border border-neutral-200 bg-white shadow-xl dark:border-white/[0.08] dark:bg-neutral-925" role="dialog" aria-modal="true" aria-labelledby="monitoring-action-title" tabindex="-1">
				<div class="flex items-start justify-between gap-4 border-b border-neutral-200/80 px-5 py-4 dark:border-white/[0.05]">
					<div><h2 id="monitoring-action-title" class="text-lg font-semibold text-neutral-900 dark:text-white">{actionTitle(action.type)}</h2><p class="mt-1 text-sm text-neutral-500">{action.item.nama_kelas} / {formatDate(action.item.tanggal)} / {action.item.jam_mulai}</p></div>
					<button onclick={closeAction} disabled={actionBusy} class="rounded-lg p-1.5 text-neutral-500 hover:bg-neutral-100 disabled:opacity-50 dark:hover:bg-neutral-800" aria-label="Tutup dialog"><X class="h-5 w-5" /></button>
				</div>

				<form onsubmit={(event) => { event.preventDefault(); submitAction(); }} class="space-y-4 p-5">
					{#if action.type === "reminder_teacher" || action.type === "reminder_students"}
						<div class="rounded-lg bg-brand-400/10 p-3 text-sm text-neutral-700 dark:text-neutral-300">Pengingat akan dikirim ke akun {teacherName(action.item)} untuk {action.type === "reminder_teacher" ? "melakukan presensi guru" : "melengkapi absensi peserta"}.</div>
						<label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300">Pesan tambahan <span class="font-normal text-neutral-400">(opsional)</span><textarea bind:value={message} rows="3" placeholder="Tambahkan konteks jika diperlukan" class={`${inputClass} resize-none`}></textarea></label>
					{:else if action.type === "note"}
						<label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300">Catatan tindak lanjut<textarea bind:value={note} required rows="4" placeholder="Tuliskan temuan atau tindak lanjut yang diperlukan" class={`${inputClass} resize-none`}></textarea></label>
					{:else if action.type === "reschedule"}
						<div class="grid grid-cols-2 gap-3"><label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300">Tanggal baru<input type="date" bind:value={tanggalBaru} required class={inputClass} /></label><label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300">Jam baru<input type="time" bind:value={jamBaru} required class={inputClass} /></label></div>
						<label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300">Alasan perubahan <span class="font-normal text-neutral-400">(opsional)</span><textarea bind:value={alasan} rows="3" class={`${inputClass} resize-none`}></textarea></label>
					{:else if action.type === "substitute"}
						<label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300">Guru pengganti<select bind:value={guruPenggantiID} required class={inputClass}><option value="">Pilih guru</option>{#each availableGuru as item (item.id)}<option value={String(item.id)}>{item.nama}</option>{/each}</select></label>
						<label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300">Alasan penugasan <span class="font-normal text-neutral-400">(opsional)</span><textarea bind:value={alasan} rows="3" class={`${inputClass} resize-none`}></textarea></label>
					{:else}
						<div class="rounded-lg border border-error/20 bg-error/5 p-3 text-sm text-red-700 dark:text-red-300">Jadwal yang dibatalkan tidak dapat dimulai oleh guru. Pastikan alasan sudah dikonfirmasi.</div>
						<label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300">Alasan pembatalan<textarea bind:value={reason} required rows="3" placeholder="Jelaskan alasan pembatalan" class={`${inputClass} resize-none`}></textarea></label>
					{/if}

					<div class="flex justify-end gap-2 border-t border-neutral-200/80 pt-4 dark:border-white/[0.05]">
						<button type="button" onclick={closeAction} disabled={actionBusy} class="rounded-lg px-4 py-2.5 text-sm font-medium text-neutral-600 hover:bg-neutral-100 disabled:opacity-50 dark:text-neutral-300 dark:hover:bg-neutral-800">Batal</button>
						<button type="submit" disabled={actionBusy || !canSubmit()} class="inline-flex items-center gap-2 rounded-lg px-4 py-2.5 text-sm font-semibold text-white disabled:opacity-50 {action.type === 'cancel' ? 'bg-error hover:bg-red-600' : 'bg-brand-600 hover:bg-brand-700 dark:bg-brand-500 dark:hover:bg-brand-400'}"><Send class="h-4 w-4" /> {actionBusy ? "Memproses..." : action.type === "cancel" ? "Batalkan jadwal" : action.type === "note" ? "Simpan catatan" : action.type.startsWith("reminder") ? "Kirim pengingat" : "Simpan perubahan"}</button>
					</div>
				</form>
			</div>
		</div>
	{/if}
</AppLayout>
