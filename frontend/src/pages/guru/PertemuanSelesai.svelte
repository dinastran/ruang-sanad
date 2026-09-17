<script lang="ts">
	import { inertia, router } from "@inertiajs/svelte";
	import { fly } from "svelte/transition";
	import AppLayout from "@layouts/AppLayout.svelte";
	import type { Flash, User, GuruKelas, PertemuanGuru, SantriGuru } from "@lib/types";
	import { BookOpen, Clock, Calendar, CheckSquare } from "lucide-svelte";

	interface Props {
		user?: User;
		kelas?: GuruKelas;
		pertemuan: PertemuanGuru;
		santri: SantriGuru[];
		can_manage_class?: boolean;
		success?: string;
		error?: string;
		flash?: Flash;
	}

	let { user, kelas, pertemuan, santri = [], can_manage_class = true, success, error, flash }: Props = $props();
	let canEdit = $derived(user?.role === "guru" || user?.role === "admin_kelas" || user?.role === "super_admin");
	let backHref = $derived(can_manage_class ? "/app/guru/kelas/" + (kelas?.id || pertemuan.kelas_id) : "/app/guru/jadwal-pertemuan");
	let backLabel = $derived(can_manage_class ? (kelas?.nama_kelas || "Kelas") : "Jadwal Pertemuan");

	let materi = $state("");
	let catatan = $state("");
	let isSubmitting = $state(false);
	let errorMsg = $state("");

	type Kehadiran = "hadir" | "izin" | "sakit" | "alpa" | "telat";
	function initialAttendance() {
		return Object.fromEntries(santri.map((item) => [item.id, { status: "hadir" as Kehadiran, catatan: "" }]));
	}
	let absensi = $state<Record<number, { status: Kehadiran; catatan: string }>>(initialAttendance());

	const statusOptions: { value: Kehadiran; label: string; color: string }[] = [
		{ value: "hadir", label: "Hadir", color: "bg-green-500/10 text-green-700 dark:text-green-400 border-green-500/20" },
		{ value: "izin", label: "Izin", color: "bg-yellow-500/10 text-yellow-700 dark:text-yellow-400 border-yellow-500/20" },
		{ value: "sakit", label: "Sakit", color: "bg-blue-500/10 text-blue-700 dark:text-blue-400 border-blue-500/20" },
		{ value: "alpa", label: "Alpa", color: "bg-red-500/10 text-red-700 dark:text-red-400 border-red-500/20" },
		{ value: "telat", label: "Telat", color: "bg-orange-500/10 text-orange-700 dark:text-orange-400 border-orange-500/20" },
	];

	let semuaHadir = $derived(santri.every((s) => absensi[s.id]?.status === "hadir"));

	function handleSelesai() {
		if (!materi.trim()) {
			errorMsg = "Materi wajib diisi";
			return;
		}
		isSubmitting = true;
		errorMsg = "";

		const payload = {
			materi: materi,
			catatan: catatan,
			absensi: santri.map((s) => ({
				santri_id: s.id,
				status: absensi[s.id]?.status || "hadir",
				catatan: absensi[s.id]?.catatan || "",
			})),
		};

		router.post("/app/guru/kelas/" + (kelas?.id || pertemuan.kelas_id) + "/pertemuan/" + pertemuan.id + "/selesai", payload, {
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

	function setSemuaHadir() {
		for (const s of santri) {
			absensi[s.id] = { status: "hadir", catatan: absensi[s.id]?.catatan || "" };
		}
		absensi = absensi;
	}
</script>

<AppLayout {user} group="guru-kelas">
	<div class="pt-8 pb-10 border-b border-neutral-200/80 dark:border-white/[0.04]">
		<div class="max-w-4xl mx-auto px-4 sm:px-6">
			<div class="flex items-center gap-2 text-sm text-neutral-500 dark:text-neutral-400 mb-4">
				<a href={can_manage_class ? "/app/guru/kelas" : "/app/guru/jadwal-pertemuan"} use:inertia class="hover:text-brand-600 dark:hover:text-brand-400 transition-colors">{can_manage_class ? "Kelas Saya" : "Jadwal"}</a>
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
				</svg>
				<a href={backHref} use:inertia class="hover:text-brand-600 dark:hover:text-brand-400 transition-colors">{backLabel}</a>
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
				</svg>
				<span class="text-neutral-700 dark:text-neutral-300">Pertemuan {pertemuan.pertemuan_ke}</span>
			</div>
			<h1 class="text-2xl sm:text-3xl font-bold text-neutral-900 dark:text-white mb-2 tracking-tight">
				Selesai Pertemuan
			</h1>
		</div>
	</div>

	<div class="relative max-w-4xl mx-auto px-4 sm:px-6 py-8 space-y-6">
		{#if success || flash?.success}
			<div class="bg-green-500/10 border border-green-500/20 text-green-700 dark:text-green-400 rounded-2xl p-4 flex items-center gap-3" in:fly={{ y: 20, duration: 300 }}>
				<p class="text-sm font-medium">{success || flash?.success}</p>
			</div>
		{/if}

		{#if errorMsg || error || flash?.error}
			<div class="bg-red-500/10 border border-red-500/20 text-red-600 dark:text-red-400 rounded-2xl p-4 flex items-center gap-3" in:fly={{ y: 20, duration: 300 }}>
				<p class="text-sm font-medium">{errorMsg || error || flash?.error}</p>
			</div>
		{/if}

		<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-6" in:fly={{ y: 20, duration: 500 }}>
			<div class="flex items-center gap-4 mb-6 pb-4 border-b border-neutral-200/80 dark:border-white/[0.04]">
				<div class="w-12 h-12 rounded-xl bg-brand-400/10 flex items-center justify-center shrink-0">
					<BookOpen class="w-6 h-6 text-brand-600 dark:text-brand-400" />
				</div>
				<div>
					<h2 class="text-lg font-semibold text-neutral-900 dark:text-white">{kelas?.nama_kelas || "Kelas"}</h2>
					<p class="text-sm text-neutral-500 dark:text-neutral-400">
						Pertemuan {pertemuan.pertemuan_ke} &middot; {pertemuan.tanggal} &middot; {pertemuan.jam_mulai}
					</p>
				</div>
			</div>

			<div class="space-y-4 mb-6">
				<div>
					<label for="materi" class="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-1.5">
						Materi / Batas Materi <span class="text-red-500">*</span>
					</label>
					<input
						id="materi"
						type="text"
						bind:value={materi}
						readonly={!canEdit}
						placeholder="Contoh: Bab 3 - Fi'il Madhi"
						class="w-full px-4 py-3 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white outline-none text-sm"
					/>
				</div>
				<div>
					<label for="catatan" class="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-1.5">Catatan Pertemuan (opsional)</label>
					<textarea
						id="catatan"
						bind:value={catatan}
						readonly={!canEdit}
						placeholder="Catatan selama pertemuan..."
						class="w-full px-4 py-3 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white outline-none text-sm resize-none"
						rows="2"
					></textarea>
				</div>
			</div>

			<div class="flex items-center justify-between pt-4 border-t border-neutral-200/80 dark:border-white/[0.04]">
				<h3 class="text-base font-semibold text-neutral-900 dark:text-white">Absensi Santri</h3>
				<button onclick={setSemuaHadir} disabled={!canEdit}
					class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium bg-green-500/10 text-green-700 dark:text-green-400 hover:bg-green-500/20 transition-colors"
				>
					<CheckSquare class="w-3.5 h-3.5" />
					Semua Hadir
				</button>
			</div>
		</div>

		<div class="space-y-3" in:fly={{ y: 20, duration: 600, delay: 100 }}>
			{#each santri as s}
				<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-4 sm:p-5">
					<div class="flex flex-col sm:flex-row sm:items-center gap-3 sm:gap-4">
						<div class="flex items-center gap-3 min-w-0 sm:w-48 shrink-0">
							<div class="w-9 h-9 rounded-full bg-brand-600 dark:bg-brand-500 flex items-center justify-center text-white font-bold text-sm shrink-0">
								{s.nama.charAt(0).toUpperCase()}
							</div>
							<div class="min-w-0">
								<p class="text-sm font-medium text-neutral-900 dark:text-white truncate">{s.nama}</p>
								<p class="text-xs text-neutral-500 font-mono">{s.id_mahasantri}</p>
							</div>
						</div>

						<div class="flex-1 grid sm:grid-cols-2 gap-2">
							<select
								value={absensi[s.id]?.status || "hadir"}
								onchange={(event) => (absensi[s.id] = { status: event.currentTarget.value as Kehadiran, catatan: absensi[s.id]?.catatan || "" })}
								disabled={!canEdit}
								class="w-full px-3 py-2 rounded-lg text-sm font-medium border focus:outline-none focus:ring-2 focus:ring-brand-400/40 appearance-none cursor-pointer {statusOptions.find((o) => o.value === absensi[s.id]?.status)?.color || 'bg-neutral-100/80 dark:bg-neutral-800/50 border-neutral-300 dark:border-neutral-700/80 text-neutral-700 dark:text-neutral-300'}"
							>
								{#each statusOptions as opt}
									<option value={opt.value}>{opt.label}</option>
								{/each}
							</select>
							<input
								type="text"
								value={absensi[s.id]?.catatan || ""}
								oninput={(event) => (absensi[s.id] = { status: absensi[s.id]?.status || "hadir", catatan: event.currentTarget.value })}
								readonly={!canEdit}
								placeholder="Catatan (opsional)"
								class="w-full px-3 py-2 rounded-lg bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white outline-none text-xs"
							/>
						</div>
					</div>
				</div>
			{/each}
		</div>

		<div class="pt-2" in:fly={{ y: 20, duration: 600, delay: 200 }}>
			<button
				onclick={handleSelesai}
				disabled={!canEdit || isSubmitting || !materi.trim()}
				class="w-full inline-flex items-center justify-center gap-2 px-6 py-3.5 rounded-xl bg-brand-600 hover:bg-brand-700 text-white font-semibold transition-all dark:bg-brand-500 dark:hover:bg-brand-400 disabled:opacity-50 disabled:cursor-not-allowed shadow-lg shadow-brand-600/25 text-base"
			>
				{#if isSubmitting}
					<svg class="animate-spin h-5 w-5" fill="none" viewBox="0 0 24 24">
						<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
						<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
					</svg>
					Menyimpan...
				{:else if !canEdit}
					Mode lihat saja
				{:else}
					<CheckSquare class="w-5 h-5" />
					Selesai
				{/if}
			</button>
		</div>
	</div>
</AppLayout>
