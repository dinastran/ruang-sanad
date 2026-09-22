<script lang="ts">
	import { inertia, router } from "@inertiajs/svelte";
	import AppLayout from "@layouts/AppLayout.svelte";
	import type { Flash, GuruDirectory, GuruKelas, JadwalPertemuanGuru, User } from "@lib/types";
	import { ArrowLeft, CalendarClock, Clock, Plus, RotateCcw, UserRoundCheck, Play, Ban, BookOpen } from "lucide-svelte";

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
	function initialClassID() { return selected_kelas_id || kelas[0]?.id || 0; }
	function shouldOpenCreate() { return selected_kelas_id > 0; }
	let showCreate = $state(shouldOpenCreate());
	let kelasID = $state(initialClassID());
	let tanggal = $state("");
	let jamMulai = $state("");
	let catatan = $state("");
	let submitting = $state(false);
	let action = $state<{ type: "reschedule" | "badal"; item: JadwalPertemuanGuru } | null>(null);
	let tanggalBaru = $state("");
	let jamBaru = $state("");
	let guruPenggantiID = $state("");
	let alasan = $state("");

	let today = localDate(new Date());
	let availableGuru = $derived(guruList.filter((guru) => guru.is_aktif && guru.user_id && guru.id !== action?.item.guru_utama_id));

	function localDate(date: Date) {
		const offset = date.getTimezoneOffset() * 60_000;
		return new Date(date.getTime() - offset).toISOString().slice(0, 10);
	}

	function formatDate(value: string) {
		return new Intl.DateTimeFormat("id-ID", { weekday: "long", day: "numeric", month: "long", year: "numeric" }).format(new Date(value + "T00:00:00"));
	}

	function openAction(type: "reschedule" | "badal", item: JadwalPertemuanGuru) {
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
				if (!responseFlash?.error) { tanggal = ""; jamMulai = ""; catatan = ""; showCreate = false; }
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
		router.post(`/app/guru/kelas/${item.kelas_id}/jadwal-pertemuan/${item.id}/mulai`, {});
	}

	function cancelSchedule(item: JadwalPertemuanGuru) {
		if (!window.confirm(`Batalkan jadwal ${item.kelas_nama} pada ${formatDate(item.tanggal)}?`)) return;
		router.post(`/app/guru/kelas/${item.kelas_id}/jadwal-pertemuan/${item.id}/batal`, {}, { preserveScroll: true });
	}
</script>

<AppLayout {user} group="guru-jadwal">
	<div class="border-b border-neutral-200/80 pt-8 pb-10 dark:border-white/[0.04]">
		<div class="mx-auto max-w-6xl px-4 sm:px-6">
			<a href="/app/guru" use:inertia class="mb-4 inline-flex items-center gap-1.5 text-sm text-neutral-500 hover:text-brand-600 dark:hover:text-brand-400">
				<ArrowLeft class="h-4 w-4" /> Dashboard guru
			</a>
			<div class="flex flex-wrap items-end justify-between gap-4">
				<div>
					<h1 class="text-2xl font-bold tracking-tight text-neutral-900 sm:text-3xl dark:text-white">Jadwal Pertemuan</h1>
					<p class="mt-2 max-w-2xl text-neutral-600 dark:text-neutral-400">Kelola sesi yang belum berjalan. Nomor pertemuan baru diberikan saat sesi dimulai.</p>
				</div>
				{#if kelas.length > 0}
					<button onclick={() => (showCreate = !showCreate)} class="inline-flex items-center gap-2 rounded-xl bg-brand-600 px-4 py-2.5 text-sm font-semibold text-white transition-colors hover:bg-brand-700 dark:bg-brand-500 dark:hover:bg-brand-400">
						<Plus class="h-4 w-4" /> Buat jadwal
					</button>
				{/if}
			</div>
		</div>
	</div>

	<div class="mx-auto max-w-6xl space-y-6 px-4 py-8 sm:px-6">
		{#if success || flash?.success}<div class="rounded-xl border border-green-500/20 bg-green-500/10 p-4 text-sm font-medium text-green-700 dark:text-green-400">{success || flash?.success}</div>{/if}
		{#if error || flash?.error}<div class="rounded-xl border border-red-500/20 bg-red-500/10 p-4 text-sm font-medium text-red-600 dark:text-red-400">{error || flash?.error}</div>{/if}

		{#if showCreate}
			<section class="rounded-2xl border border-brand-400/25 bg-brand-400/5 p-5 sm:p-6">
				<div class="mb-5">
					<h2 class="font-semibold text-neutral-900 dark:text-white">Jadwalkan sesi baru</h2>
					<p class="mt-1 text-sm text-neutral-500 dark:text-neutral-400">Jadwal dapat diubah atau diberi guru badal sebelum dimulai.</p>
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
				<div class="mt-5 flex justify-end gap-2">
					<button onclick={() => (showCreate = false)} class="rounded-xl px-4 py-2.5 text-sm font-medium text-neutral-600 hover:bg-neutral-200/70 dark:text-neutral-300 dark:hover:bg-neutral-800">Batal</button>
					<button onclick={createSchedule} disabled={submitting || !kelasID || !tanggal || !jamMulai} class="rounded-xl bg-brand-600 px-4 py-2.5 text-sm font-semibold text-white hover:bg-brand-700 disabled:opacity-50 dark:bg-brand-500 dark:hover:bg-brand-400">{submitting ? "Menyimpan..." : "Simpan jadwal"}</button>
				</div>
			</section>
		{/if}

		{#if jadwal.length > 0}
			<div class="space-y-3">
				{#each jadwal as item}
					<article class="rounded-2xl border border-neutral-200/80 bg-white p-5 dark:border-white/[0.06] dark:bg-neutral-925/50">
						<div class="flex flex-wrap justify-between gap-4">
							<div class="flex min-w-0 gap-3">
								<div class="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-brand-400/10 text-brand-600 dark:text-brand-400"><CalendarClock class="h-5 w-5" /></div>
								<div>
									<h2 class="font-semibold text-neutral-900 dark:text-white">{item.kelas_nama}</h2>
									<p class="mt-1 text-sm text-neutral-600 dark:text-neutral-300">{formatDate(item.tanggal)} · {item.jam_mulai}</p>
									<p class="mt-1 text-xs text-neutral-500">Guru utama: {item.guru_utama_nama || "Belum ditetapkan"}</p>
								</div>
							</div>
							<span class="h-fit rounded-full px-2.5 py-1 text-xs font-semibold {item.status === 'dimulai' ? 'bg-amber-500/10 text-amber-700 dark:text-amber-400' : 'bg-brand-400/10 text-brand-700 dark:text-brand-400'}">{item.status === "dimulai" ? "Berlangsung" : "Dijadwalkan"}</span>
						</div>

						{#if item.catatan}<p class="mt-4 text-sm text-neutral-600 dark:text-neutral-400">{item.catatan}</p>{/if}
						{#if item.is_reschedule || item.guru_pengganti_id || item.jadwal_kelas_berubah}
							<div class="mt-3 flex flex-wrap gap-2 text-xs">
								{#if item.jadwal_kelas_berubah}<span class="inline-flex items-center gap-1 rounded-full bg-amber-500/10 px-2.5 py-1 text-amber-700 dark:text-amber-400" title="Jadwal rutin kelas sudah diganti Admin Kelas setelah sesi ini dibuat. Jadwalkan ulang bila perlu."><CalendarClock class="h-3.5 w-3.5" /> Jadwal kelas berubah</span>{/if}
								{#if item.is_reschedule}<span class="inline-flex items-center gap-1 rounded-full bg-blue-500/10 px-2.5 py-1 text-blue-700 dark:text-blue-400"><RotateCcw class="h-3.5 w-3.5" /> Dari {item.jadwal_semula}</span>{/if}
								{#if item.guru_pengganti_id}<span class="inline-flex items-center gap-1 rounded-full bg-secondary-500/10 px-2.5 py-1 text-secondary-700 dark:text-secondary-400"><UserRoundCheck class="h-3.5 w-3.5" /> Badal: {item.guru_pengganti_nama}</span>{/if}
							</div>
						{/if}

						<div class="mt-4 flex flex-wrap gap-2 border-t border-neutral-200/70 pt-4 dark:border-white/[0.05]">
							{#if item.status === "dimulai" && item.pertemuan_id}
								<a href={"/app/guru/kelas/" + item.kelas_id + "/pertemuan/" + item.pertemuan_id + "/selesai"} use:inertia class="inline-flex items-center gap-1.5 rounded-lg bg-amber-600 px-3 py-2 text-xs font-semibold text-white hover:bg-amber-700"><Play class="h-3.5 w-3.5" /> Lanjutkan</a>
							{:else}
								<button onclick={() => startSchedule(item)} disabled={!item.can_start || item.tanggal > today} title={item.tanggal > today ? "Pertemuan belum dapat dimulai sebelum tanggal jadwal" : "Mulai pertemuan"} class="inline-flex items-center gap-1.5 rounded-lg bg-brand-600 px-3 py-2 text-xs font-semibold text-white hover:bg-brand-700 disabled:opacity-40 dark:bg-brand-500 dark:hover:bg-brand-400"><Play class="h-3.5 w-3.5" /> Mulai</button>
							{/if}
							{#if item.can_manage}
								<button onclick={() => openAction("reschedule", item)} class="inline-flex items-center gap-1.5 rounded-lg bg-blue-500/10 px-3 py-2 text-xs font-medium text-blue-700 hover:bg-blue-500/20 dark:text-blue-400"><RotateCcw class="h-3.5 w-3.5" /> Jadwalkan ulang</button>
								<button onclick={() => openAction("badal", item)} class="inline-flex items-center gap-1.5 rounded-lg bg-secondary-500/10 px-3 py-2 text-xs font-medium text-secondary-700 hover:bg-secondary-500/20 dark:text-secondary-400"><UserRoundCheck class="h-3.5 w-3.5" /> Tetapkan badal</button>
								<button onclick={() => cancelSchedule(item)} class="inline-flex items-center gap-1.5 rounded-lg px-3 py-2 text-xs font-medium text-red-600 hover:bg-red-500/10 dark:text-red-400"><Ban class="h-3.5 w-3.5" /> Batalkan</button>
							{/if}
						</div>
					</article>
				{/each}
			</div>
		{:else if !showCreate}
			<div class="rounded-2xl border border-neutral-200/80 bg-white p-12 text-center dark:border-white/[0.06] dark:bg-neutral-925/50">
				<BookOpen class="mx-auto h-8 w-8 text-neutral-400" />
				<h2 class="mt-4 font-semibold text-neutral-900 dark:text-white">Belum ada jadwal mendatang</h2>
				<p class="mt-1 text-sm text-neutral-500">Buat jadwal untuk mengatur reschedule dan guru badal sebelum kelas berjalan.</p>
			</div>
		{/if}
	</div>

	{#if action}
		<div class="fixed inset-0 z-50 flex items-center justify-center p-4">
			<button class="absolute inset-0 bg-black/50" aria-label="Tutup dialog" onclick={closeAction}></button>
			<div class="relative w-full max-w-md rounded-2xl bg-white p-5 shadow-xl dark:bg-neutral-925" role="dialog" aria-modal="true" aria-labelledby="action-title" tabindex="-1">
				<h2 id="action-title" class="text-lg font-semibold text-neutral-900 dark:text-white">{action.type === "reschedule" ? "Jadwalkan Ulang" : "Tetapkan Guru Badal"}</h2>
				<p class="mt-1 text-sm text-neutral-500">{action.item.kelas_nama}</p>
				<div class="mt-5 space-y-4">
					{#if action.type === "reschedule"}
						<label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300">Tanggal baru<input type="date" min={today} bind:value={tanggalBaru} class="mt-1.5 w-full rounded-xl border border-neutral-300 bg-neutral-50 px-3 py-2.5 text-sm dark:border-neutral-700 dark:bg-neutral-800/50 dark:text-white" /></label>
						<label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300">Jam baru<input type="time" bind:value={jamBaru} class="mt-1.5 w-full rounded-xl border border-neutral-300 bg-neutral-50 px-3 py-2.5 text-sm dark:border-neutral-700 dark:bg-neutral-800/50 dark:text-white" /></label>
					{:else}
						<label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300">Guru pengganti<select bind:value={guruPenggantiID} class="mt-1.5 w-full rounded-xl border border-neutral-300 bg-neutral-50 px-3 py-2.5 text-sm dark:border-neutral-700 dark:bg-neutral-800/50 dark:text-white"><option value="">Pilih guru aktif</option>{#each availableGuru as guru}<option value={guru.id}>{guru.nama}</option>{/each}</select></label>
					{/if}
					<label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300">Alasan <span class="font-normal text-neutral-400">(opsional)</span><textarea bind:value={alasan} rows="3" class="mt-1.5 w-full resize-none rounded-xl border border-neutral-300 bg-neutral-50 px-3 py-2.5 text-sm dark:border-neutral-700 dark:bg-neutral-800/50 dark:text-white"></textarea></label>
					<div class="flex justify-end gap-2"><button onclick={closeAction} class="rounded-xl px-4 py-2 text-sm text-neutral-600 dark:text-neutral-300">Batal</button><button onclick={submitAction} disabled={action.type === "reschedule" ? !tanggalBaru || !jamBaru : !guruPenggantiID} class="rounded-xl bg-brand-600 px-4 py-2 text-sm font-semibold text-white disabled:opacity-50">Simpan</button></div>
				</div>
			</div>
		</div>
	{/if}
</AppLayout>
