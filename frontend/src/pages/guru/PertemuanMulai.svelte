<script lang="ts">
	import { inertia, router } from "@inertiajs/svelte";
	import { fly } from "svelte/transition";
	import AppLayout from "@layouts/AppLayout.svelte";
	import type { Flash, User, GuruKelas, JadwalPertemuanGuru } from "@lib/types";
	import { BookOpen, Clock, Calendar, Play, RotateCcw, UserRoundCheck } from "lucide-svelte";

	interface Props {
		user?: User;
		kelas: GuruKelas;
		next_pertemuan_ke: number;
		jadwal_tersedia?: JadwalPertemuanGuru[];
		flash?: Flash;
	}

	let { user, kelas, next_pertemuan_ke, jadwal_tersedia = [], flash }: Props = $props();
	let canEdit = $derived(user?.role === "guru" || user?.role === "admin_kelas" || user?.role === "super_admin");
	function initialScheduleID() { return jadwal_tersedia.length === 1 ? jadwal_tersedia[0].id : 0; }
	let selectedJadwalID = $state(initialScheduleID());
	let selectedJadwal = $derived(jadwal_tersedia.find((item) => item.id === selectedJadwalID));
	let hasDueSchedule = $derived(jadwal_tersedia.length > 0);

	let jamMulai = $state(new Date().toTimeString().substring(0, 5));
	let catatan = $state("");
	let isSubmitting = $state(false);
	let errorMsg = $state("");

	let today = $derived(new Date().toLocaleDateString("id-ID", { year: "numeric", month: "long", day: "numeric" }));
	function formatDate(value: string) {
		return new Intl.DateTimeFormat("id-ID", { weekday: "long", day: "numeric", month: "long", year: "numeric" }).format(new Date(value + "T00:00:00"));
	}

	function handleMulai() {
		if (hasDueSchedule && !selectedJadwalID) {
			errorMsg = "Pilih jadwal yang akan dimulai";
			return;
		}
		if (!hasDueSchedule && !jamMulai) return;
		isSubmitting = true;
		errorMsg = "";
		router.post("/app/guru/kelas/" + kelas.id + "/pertemuan/mulai", {
			jadwal_id: selectedJadwalID,
			jam_mulai: hasDueSchedule ? "" : jamMulai,
			catatan: hasDueSchedule ? "" : catatan,
		}, {
			onError: (err) => {
				errorMsg = Object.values(err).join(", ");
				isSubmitting = false;
			},
			onSuccess: (page) => {
				const responseFlash = page.props.flash as Flash | undefined;
				if (responseFlash?.error) errorMsg = responseFlash.error;
			},
			onFinish: () => {
				isSubmitting = false;
			},
		});
	}
</script>

<AppLayout {user} group="guru-kelas">
	<div class="pt-8 pb-10 border-b border-neutral-200/80 dark:border-white/[0.04]">
		<div class="max-w-3xl mx-auto px-4 sm:px-6">
			<div class="flex items-center gap-2 text-sm text-neutral-500 dark:text-neutral-400 mb-4">
				<a href="/app/guru/kelas" use:inertia class="hover:text-brand-600 dark:hover:text-brand-400 transition-colors">Kelas Saya</a>
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
				</svg>
				<a href={"/app/guru/kelas/" + kelas.id} use:inertia class="hover:text-brand-600 dark:hover:text-brand-400 transition-colors">{kelas.nama_kelas}</a>
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
				</svg>
				<span class="text-neutral-700 dark:text-neutral-300">{hasDueSchedule ? "Mulai Jadwal" : "Mulai Pertemuan"}</span>
			</div>
			<h1 class="text-2xl sm:text-3xl font-bold text-neutral-900 dark:text-white mb-2 tracking-tight">
				{hasDueSchedule ? "Mulai Pertemuan Terjadwal" : "Mulai Pertemuan"}
			</h1>
		</div>
	</div>

	<div class="relative max-w-3xl mx-auto px-4 sm:px-6 py-8">
		{#if errorMsg || flash?.error}
			<div class="bg-red-500/10 border border-red-500/20 text-red-600 dark:text-red-400 rounded-2xl p-4 flex items-center gap-3 mb-6" in:fly={{ y: 20, duration: 300 }}>
				<p class="text-sm font-medium">{errorMsg || flash?.error}</p>
			</div>
		{/if}

		<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-6" in:fly={{ y: 20, duration: 500 }}>
			<div class="flex items-center gap-4 mb-6 pb-4 border-b border-neutral-200/80 dark:border-white/[0.04]">
				<div class="w-12 h-12 rounded-xl bg-brand-400/10 flex items-center justify-center shrink-0">
					<BookOpen class="w-6 h-6 text-brand-600 dark:text-brand-400" />
				</div>
				<div>
					<h2 class="text-lg font-semibold text-neutral-900 dark:text-white">{kelas.nama_kelas}</h2>
					<p class="text-sm text-neutral-500 dark:text-neutral-400">{kelas.level} &middot; {kelas.jadwal}</p>
				</div>
			</div>

			<div class="grid sm:grid-cols-2 gap-4 mb-6">
				<div class="p-4 rounded-xl bg-neutral-50 dark:bg-neutral-900/50 border border-neutral-200/80 dark:border-white/[0.04]">
					<span class="text-xs font-medium text-neutral-500 dark:text-neutral-400 uppercase tracking-wider">Pertemuan Ke-</span>
					<p class="text-2xl font-bold text-neutral-900 dark:text-white mt-1">{next_pertemuan_ke}</p>
				</div>
				<div class="p-4 rounded-xl bg-neutral-50 dark:bg-neutral-900/50 border border-neutral-200/80 dark:border-white/[0.04]">
					<span class="text-xs font-medium text-neutral-500 dark:text-neutral-400 uppercase tracking-wider">Tanggal pelaksanaan</span>
					<p class="text-lg font-semibold text-neutral-900 dark:text-white mt-1">{selectedJadwal ? formatDate(selectedJadwal.tanggal) : today}</p>
				</div>
			</div>

			{#if hasDueSchedule}
				<div class="mb-6 rounded-xl border border-brand-400/25 bg-brand-400/5 p-4">
					<p class="text-sm font-semibold text-brand-800 dark:text-brand-300">Jadwal ditemukan</p>
					<p class="mt-1 text-xs text-brand-700/80 dark:text-brand-400">Pertemuan ini akan memakai record jadwal yang sudah dibuat, termasuk informasi reschedule dan guru badal.</p>
				</div>
				{#if jadwal_tersedia.length > 1}
					<label for="jadwal_id" class="mb-4 block text-sm font-medium text-neutral-700 dark:text-neutral-300">Pilih jadwal
						<select id="jadwal_id" bind:value={selectedJadwalID} class="mt-1.5 w-full rounded-xl border border-neutral-300 bg-neutral-100/80 px-4 py-3 text-sm text-neutral-900 outline-none focus:border-brand-400 dark:border-neutral-700/80 dark:bg-neutral-800/50 dark:text-white">
							<option value={0}>Pilih jadwal yang dijalankan</option>
							{#each jadwal_tersedia as jadwal}<option value={jadwal.id}>{formatDate(jadwal.tanggal)} · {jadwal.jam_mulai}</option>{/each}
						</select>
					</label>
				{/if}
				{#if selectedJadwal}
					<div class="space-y-3 rounded-xl border border-neutral-200/80 bg-neutral-50 p-4 dark:border-white/[0.05] dark:bg-neutral-900/50">
						<div class="flex flex-wrap items-center gap-x-4 gap-y-2 text-sm text-neutral-700 dark:text-neutral-300">
							<span class="inline-flex items-center gap-1.5"><Calendar class="h-4 w-4 text-brand-500" /> {formatDate(selectedJadwal.tanggal)}</span>
							<span class="inline-flex items-center gap-1.5"><Clock class="h-4 w-4 text-brand-500" /> {selectedJadwal.jam_mulai}</span>
						</div>
						{#if selectedJadwal.catatan}<p class="text-sm text-neutral-600 dark:text-neutral-400">{selectedJadwal.catatan}</p>{/if}
						<div class="flex flex-wrap gap-2 text-xs">
							{#if selectedJadwal.is_reschedule}<span class="inline-flex items-center gap-1 rounded-full bg-blue-500/10 px-2.5 py-1 text-blue-700 dark:text-blue-400"><RotateCcw class="h-3.5 w-3.5" /> Dijadwalkan ulang</span>{/if}
							{#if selectedJadwal.guru_pengganti_id}<span class="inline-flex items-center gap-1 rounded-full bg-secondary-500/10 px-2.5 py-1 text-secondary-700 dark:text-secondary-400"><UserRoundCheck class="h-3.5 w-3.5" /> Badal: {selectedJadwal.guru_pengganti_nama}</span>{/if}
						</div>
					</div>
				{/if}
			{:else}
			<div class="space-y-4">
				<div>
					<label for="jam_mulai" class="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-1.5">
						<Clock class="w-4 h-4 inline mr-1" />
						Jadwal Mulai
					</label>
					<input
						id="jam_mulai"
						type="time"
						bind:value={jamMulai}
						disabled={!canEdit}
						class="w-full px-4 py-3 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white outline-none text-sm"
					/>
				</div>
				<div>
					<label for="catatan" class="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-1.5">Catatan (opsional)</label>
					<textarea
						id="catatan"
						bind:value={catatan}
						readonly={!canEdit}
						placeholder="Catatan awal pertemuan..."
						class="w-full px-4 py-3 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white outline-none text-sm resize-none"
						rows="3"
					></textarea>
				</div>
			</div>
			{/if}

			<div class="mt-6 pt-4 border-t border-neutral-200/80 dark:border-white/[0.04]">
				<button
					onclick={handleMulai}
					disabled={!canEdit || isSubmitting || (hasDueSchedule ? !selectedJadwalID : !jamMulai)}
					class="w-full inline-flex items-center justify-center gap-2 px-6 py-3 rounded-xl bg-brand-600 hover:bg-brand-700 text-white font-semibold transition-all dark:bg-brand-500 dark:hover:bg-brand-400 disabled:opacity-50 disabled:cursor-not-allowed shadow-lg shadow-brand-600/25 text-base"
				>
					{#if isSubmitting}
						<svg class="animate-spin h-5 w-5" fill="none" viewBox="0 0 24 24">
							<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
							<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
						</svg>
						Memulai...
					{:else if !canEdit}
						Mode lihat saja
					{:else}
						<Play class="w-5 h-5" />
						{hasDueSchedule ? "Mulai Jadwal Ini" : "Mulai Pertemuan"}
					{/if}
				</button>
			</div>
		</div>
	</div>
</AppLayout>
