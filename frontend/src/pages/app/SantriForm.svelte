<script lang="ts">
	import { inertia, router } from "@inertiajs/svelte";
	import { tick } from "svelte";
	import { fly } from "svelte/transition";
	import AppLayout from "@layouts/AppLayout.svelte";
	import { Toast } from "@lib/notifications/toast";
	import { getCSRFToken } from "@lib/utils/csrf";
	import StatusBadge from "@components/StatusBadge.svelte";
	import GenderBadge from "@components/GenderBadge.svelte";
	import type { Flash, User } from "@lib/types";
	import { Save, User as UserIcon, BookOpen, Calendar, MapPin, DollarSign, Hash, GraduationCap, Users, Phone, Mail, ClipboardList, Plus, Trash2, Upload, AlertTriangle } from "lucide-svelte";

	interface MasterItem {
		id: number;
		kode?: string;
		nama?: string;
		keterangan?: string;
	}

	interface KodeKelasItem {
		id: number;
		kode: string;
		tipe: string;
		frekuensi: string;
		urutan: number;
	}

	interface SantriItem {
		id: number;
		id_mahasantri: string;
		id_mahasantri_bermasalah: boolean;
		id_mahasantri_terbit: boolean;
		kelas_kode: string;
		nama: string;
		jenis_kelamin: string;
		nominal: number;
		tanggal_daftar: string;
		angkatan: string;
		angkatan_kelas: string;
		usia: number;
		domisili: string;
		no_wa: string;
		email: string;
		fu: string;
		tanggal_vn: string;
		hasil_vn: string;
		voice_note_url: string;
		keterangan_vn: string;
		masuk_grup: string;
		mulai_belajar: string;
		jumlah: number;
		level: string;
		jadwal: string;
		guru: string;
		tipe: string;
		frekuensi: string;
		is_lengkap: boolean;
		status: string;
	}

	interface Props {
		user?: User;
		santri?: SantriItem | null;
		angkatan?: MasterItem[];
		levels?: MasterItem[];
		jadwals?: MasterItem[];
		gurus?: MasterItem[];
		kode_kelas?: KodeKelasItem[];
		success?: string;
		error?: string;
		flash?: Flash;
	}

	let { user, santri = null, angkatan = [], levels = [], jadwals = [], gurus = [], kode_kelas = [], success, error, flash }: Props = $props();
	let successMessage = $derived(flash?.success ?? success);
	let errorMessage = $derived(flash?.error ?? error);

	let isEdit = $derived(santri !== null);
	let isAdminKelas = $derived(user?.role === "admin_kelas" || user?.role === "super_admin");
	// A pure admin_kelas may view Data Santri detail but not edit it. cs &
	// super_admin keep full edit access.
	let readonly = $derived(user?.role === "admin_kelas");
	let identityLocked = $derived(santri?.id_mahasantri_terbit === true);
	// The Data Kelas section is admin_kelas's own domain. Only admin_kelas and
	// super_admin can see it, and both are allowed to edit it (backend route
	// /santri/:id/kelas-data is gated to those roles) — so it stays editable even
	// while the CS/Data Santri section above is read-only for admin_kelas.
	let kelasReadonly = false;
	let pageTitle = $derived(readonly ? "Detail Santri" : isEdit ? "Edit Santri" : "Tambah Santri");

	let csForm = $state({
		kelas_kode: santri?.kelas_kode ?? "",
		nama: santri?.nama ?? "",
		jenis_kelamin: santri?.jenis_kelamin ?? "L",
		nominal: santri?.nominal ?? 0,
		tanggal_daftar: santri?.tanggal_daftar ?? "",
		angkatan: santri?.angkatan ?? "",
		usia: santri?.usia ?? 0,
		domisili: santri?.domisili ?? "",
		no_wa: santri?.no_wa ?? "",
		email: santri?.email ?? "",
	});

	let kelasForm = $state({
		fu: santri?.fu ?? "",
		tanggal_vn: santri?.tanggal_vn ?? "",
		hasil_vn: santri?.hasil_vn ?? "",
		masuk_grup: santri?.masuk_grup ?? "",
		mulai_belajar: santri?.mulai_belajar ?? "",
		jumlah: santri?.jumlah ?? 0,
		angkatan_kelas: santri?.angkatan_kelas ?? "",
		level: santri?.level ?? "",
		jadwal: santri?.jadwal ?? "",
		guru: santri?.guru ?? "",
		guru_id: gurus.find((g) => g.nama === santri?.guru)?.id ?? 0,
	});
	let hasLegacyAngkatanKelas = $derived(
		kelasForm.angkatan_kelas !== "" &&
		!angkatan.some((a) => (a.kode || String(a.id)) === kelasForm.angkatan_kelas),
	);

	// Multi-jadwal: a class can meet more than once per pekan (2x/pekan, private
	// 4x/16x). The sessions are stored joined by " & " in the single jadwal field.
	let jadwalRows = $state<string[]>(
		santri?.jadwal ? santri.jadwal.split(" & ").map((s) => s.trim()) : [""]
	);

	// The chosen Kode Kelas tells how often the class meets — a hint for how many
	// jadwal sessions to add.
	let selectedKodeKelas = $derived(kode_kelas.find((k) => k.kode === csForm.kelas_kode));
	let frekuensiLabel = $derived(selectedKodeKelas?.frekuensi ?? "");

	function addJadwalRow() {
		jadwalRows = [...jadwalRows, ""];
	}
	function removeJadwalRow(i: number) {
		jadwalRows = jadwalRows.length > 1 ? jadwalRows.filter((_, idx) => idx !== i) : [""];
	}

	let isCsLoading = $state(false);
	let isKelasLoading = $state(false);
	let voiceNoteDescription = $state(santri?.keterangan_vn ?? "");
	let voiceNoteFile = $state<File | null>(null);
	let isVoiceNoteLoading = $state(false);
	let deleteDialog = $state<HTMLDialogElement>();
	let deleteNameInput = $state<HTMLInputElement>();
	let deleteName = $state("");
	let isDeleteLoading = $state(false);
	let correctionDialog = $state<HTMLDialogElement>();
	let isCorrectionLoading = $state(false);
	let correctionForm = $state({ angkatan: santri?.angkatan ?? "", tanggal_daftar: santri?.tanggal_daftar ?? "", konfirmasi: false });
	let canDelete = $derived(santri !== null && deleteName === santri.nama && !isDeleteLoading);

	function handleCsSubmit(e: Event) {
		e.preventDefault();
		isCsLoading = true;
		if (isEdit) {
			router.put(`/app/santri/${santri!.id}/cs`, csForm, {
				onFinish: () => { isCsLoading = false; },
				onError: () => { isCsLoading = false; },
			});
		} else {
			router.post("/app/santri", csForm, {
				onFinish: () => { isCsLoading = false; },
				onError: () => { isCsLoading = false; },
			});
		}
	}

	function handleKelasSubmit(e: Event) {
		e.preventDefault();
		if (!santri?.id) return;
		kelasForm.jadwal = jadwalRows.map((s) => s.trim()).filter(Boolean).join(" & ");
		kelasForm.guru = gurus.find((g) => g.id === kelasForm.guru_id)?.nama ?? "";
		isKelasLoading = true;
		router.put(`/app/santri/${santri.id}/kelas-data`, kelasForm, {
			onFinish: () => { isKelasLoading = false; },
			onError: () => { isKelasLoading = false; },
		});
	}

	function handleVoiceNoteFile(event: Event) {
		const input = event.currentTarget as HTMLInputElement;
		voiceNoteFile = input.files?.[0] ?? null;
	}

	async function saveVoiceNote() {
		if (!santri?.id) return;
		const formData = new FormData();
		if (voiceNoteFile) formData.append("file", voiceNoteFile);
		formData.append("keterangan_vn", voiceNoteDescription);
		isVoiceNoteLoading = true;

		try {
			const response = await fetch(`/app/santri/${santri.id}/voice-note`, {
				method: "POST",
				headers: { "X-XSRF-TOKEN": getCSRFToken() },
				body: formData,
			});
			const data = await response.json();
			if (!response.ok || !data.success) throw new Error(data.error || "Gagal menyimpan VN");
			Toast("Voice note berhasil disimpan", "success");
			voiceNoteFile = null;
			router.reload({ only: ["santri"] });
		} catch (error) {
			Toast(error instanceof Error ? error.message : "Gagal menyimpan VN", "error");
		} finally {
			isVoiceNoteLoading = false;
		}
	}

	async function openDeleteDialog() {
		deleteName = "";
		deleteDialog?.showModal();
		await tick();
		deleteNameInput?.focus();
	}

	function closeDeleteDialog() {
		if (!isDeleteLoading) deleteDialog?.close();
	}

	function handleDeleteBackdrop(event: MouseEvent) {
		if (event.target === deleteDialog) closeDeleteDialog();
	}

	function handleDeleteSubmit(event: SubmitEvent) {
		event.preventDefault();
		if (!santri || !canDelete) return;
		isDeleteLoading = true;
		router.delete(`/app/santri/${santri.id}`, {
			onFinish: () => { isDeleteLoading = false; },
		});
	}

	function handleCorrectionSubmit(event: SubmitEvent) {
		event.preventDefault();
		if (!santri || !correctionForm.konfirmasi) return;
		isCorrectionLoading = true;
		router.put(`/app/santri/${santri.id}/registration-identity`, correctionForm, {
			onSuccess: () => correctionDialog?.close(),
			onFinish: () => { isCorrectionLoading = false; },
		});
	}

	function formatDate(d: string): string {
		if (!d) return "";
		return d.substring(0, 10);
	}
</script>

<AppLayout {user} group="santri">
	<div class="pt-8 pb-10 border-b border-neutral-200/80 dark:border-white/[0.04]">
		<div class="max-w-4xl mx-auto px-4 sm:px-6">
			<div class="flex items-center gap-2 text-sm text-neutral-500 dark:text-neutral-400 mb-4">
				<a href="/app" use:inertia class="hover:text-brand-600 dark:hover:text-brand-400 transition-colors">Dashboard</a>
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
				</svg>
				<a href="/app/santri" use:inertia class="hover:text-brand-600 dark:hover:text-brand-400 transition-colors">Data Santri</a>
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
				</svg>
				<span class="text-neutral-700 dark:text-neutral-300">{pageTitle}</span>
			</div>
			<h1 class="text-2xl sm:text-3xl font-bold text-neutral-900 dark:text-white mb-2 tracking-tight">
				{pageTitle}
			</h1>
			<p class="text-neutral-600 dark:text-neutral-400">
				{readonly ? 'Lihat data santri (hanya baca)' : isEdit ? 'Perbarui data santri' : 'Masukkan data santri baru'}
			</p>
		</div>
	</div>

	<div class="relative max-w-4xl mx-auto px-4 sm:px-6 py-8 space-y-8">
		{#if successMessage}
			<div class="bg-green-500/10 border border-green-500/20 text-green-700 dark:text-green-400 rounded-2xl p-4 flex items-center gap-3" in:fly={{ y: 20, duration: 300 }}>
				<p class="text-sm font-medium">{successMessage}</p>
			</div>
		{/if}

		{#if errorMessage}
			<div class="bg-red-500/10 border border-red-500/20 text-red-600 dark:text-red-400 rounded-2xl p-4 flex items-center gap-3" in:fly={{ y: 20, duration: 300 }}>
				<p class="text-sm font-medium">{errorMessage}</p>
			</div>
		{/if}

		<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-6" in:fly={{ y: 20, duration: 500 }}>
			<div class="flex items-center gap-3 mb-6">
				<div class="w-10 h-10 rounded-xl bg-brand-400/10 flex items-center justify-center">
					<UserIcon class="w-5 h-5 text-brand-600 dark:text-brand-400" />
				</div>
				<div>
					<h3 class="text-lg font-semibold text-neutral-900 dark:text-white">Data Santri</h3>
					<p class="text-sm text-neutral-600 dark:text-neutral-500">Informasi dasar pendaftaran</p>
				</div>
			</div>

			{#if isEdit}
				<div class="mb-6 rounded-xl border border-neutral-200/80 bg-neutral-50 p-4 dark:border-white/[0.06] dark:bg-neutral-900/40">
					<p class="text-xs font-medium uppercase tracking-wide text-neutral-500 dark:text-neutral-400">ID Mahasantri</p>
					<p class="mt-1 font-mono text-sm font-semibold text-neutral-900 dark:text-white">{santri?.id_mahasantri || "Belum diterbitkan"}</p>
					{#if santri?.id_mahasantri_bermasalah}
						<div class="mt-3 flex items-start gap-2 border-t border-warning/20 pt-3 text-sm text-neutral-700 dark:text-neutral-300" role="status">
							<AlertTriangle class="mt-0.5 h-4 w-4 shrink-0 text-warning" aria-hidden="true" />
							<p>ID Mahasantri kosong, tidak sesuai format, atau terduplikasi. Tanggal pendaftaran lama yang tidak diketahui bukan masalah ID.</p>
						</div>
					{/if}
				</div>
			{/if}

			<form onsubmit={handleCsSubmit}>
				<fieldset disabled={readonly} class="space-y-5 border-0 p-0 m-0 min-w-0">
				<div class="grid md:grid-cols-2 gap-5">
					<div>
						<label for="kelas_kode" class="block text-sm font-medium text-neutral-700 dark:text-neutral-400 mb-2">Kode Kelas</label>
						<div class="relative">
							<div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
								<Hash class="w-4 h-4 text-neutral-500" />
							</div>
							<select
								id="kelas_kode"
								bind:value={csForm.kelas_kode}
								class="w-full pl-12 pr-4 py-3 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white transition-all outline-none"
							>
								<option value="">Pilih Kode Kelas</option>
								{#each kode_kelas as k}
									<option value={k.kode}>{k.kode} — {k.tipe} · {k.frekuensi}</option>
								{/each}
							</select>
						</div>
					</div>
					<div>
						<label for="nama" class="block text-sm font-medium text-neutral-700 dark:text-neutral-400 mb-2">Nama Lengkap <span class="text-error">(wajib)</span></label>
						<div class="relative">
							<div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
								<UserIcon class="w-4 h-4 text-neutral-500" />
							</div>
							<input
								id="nama"
								type="text"
								bind:value={csForm.nama}
								required
								class="w-full pl-12 pr-4 py-3 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white placeholder-neutral-500 transition-all outline-none"
								placeholder="Nama santri"
							/>
						</div>
					</div>
				</div>

				<div class="grid md:grid-cols-2 gap-5">
					<div>
						<label for="no_wa" class="block text-sm font-medium text-neutral-700 dark:text-neutral-400 mb-2">No. WhatsApp Aktif</label>
						<div class="relative">
							<div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
								<Phone class="w-4 h-4 text-neutral-500" />
							</div>
							<input
								id="no_wa"
								type="tel"
								bind:value={csForm.no_wa}
								class="w-full pl-12 pr-4 py-3 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white placeholder-neutral-500 transition-all outline-none"
								placeholder="Contoh: 081234567890"
							/>
						</div>
					</div>
					<div>
						<label for="email" class="block text-sm font-medium text-neutral-700 dark:text-neutral-400 mb-2">Email</label>
						<div class="relative">
							<div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
								<Mail class="w-4 h-4 text-neutral-500" />
							</div>
							<input
								id="email"
								type="email"
								bind:value={csForm.email}
								class="w-full pl-12 pr-4 py-3 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white placeholder-neutral-500 transition-all outline-none"
								placeholder="Contoh: santri@example.com"
							/>
						</div>
					</div>
				</div>

				<div class="grid md:grid-cols-2 gap-5">
					<fieldset class="border-0 p-0 m-0 min-w-0">
						<legend class="block text-sm font-medium text-neutral-700 dark:text-neutral-400 mb-2">Jenis Kelamin <span class="text-error">(wajib)</span></legend>
						<div class="flex gap-4 pt-1">
							<label class="flex items-center gap-2 cursor-pointer">
								<input
									type="radio"
									name="jenis_kelamin"
								value="L"
								bind:group={csForm.jenis_kelamin}
								required
									class="w-4 h-4 text-brand-600 focus:ring-brand-400"
								/>
								<span class="text-sm text-neutral-700 dark:text-neutral-300">Laki-laki</span>
							</label>
							<label class="flex items-center gap-2 cursor-pointer">
								<input
									type="radio"
									name="jenis_kelamin"
									value="P"
									bind:group={csForm.jenis_kelamin}
									class="w-4 h-4 text-brand-600 focus:ring-brand-400"
								/>
								<span class="text-sm text-neutral-700 dark:text-neutral-300">Perempuan</span>
							</label>
						</div>
					</fieldset>
					<div>
						<label for="angkatan" class="block text-sm font-medium text-neutral-700 dark:text-neutral-400 mb-2">Angkatan Pendaftaran <span class="text-error">(wajib)</span></label>
						<div class="relative">
							<div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
								<GraduationCap class="w-4 h-4 text-neutral-500" />
							</div>
							<select
								id="angkatan"
								bind:value={csForm.angkatan}
								required
								disabled={identityLocked}
								class="w-full pl-12 pr-4 py-3 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white transition-all outline-none"
							>
								<option value="">Pilih Angkatan Pendaftaran</option>
								{#each angkatan as a}
									<option value={a.kode || String(a.id)}>{a.keterangan || a.kode || String(a.id)}</option>
								{/each}
							</select>
						</div>
					</div>
				</div>

				<div class="grid md:grid-cols-2 gap-5">
					<div>
						<label for="nominal" class="block text-sm font-medium text-neutral-700 dark:text-neutral-400 mb-2">Nominal (Rp)</label>
						<div class="relative">
							<div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
								<DollarSign class="w-4 h-4 text-neutral-500" />
							</div>
							<input
								id="nominal"
								type="number"
								bind:value={csForm.nominal}
								class="w-full pl-12 pr-4 py-3 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white placeholder-neutral-500 transition-all outline-none"
								placeholder="0"
							/>
						</div>
					</div>
					<div>
						<label for="tanggal_daftar" class="block text-sm font-medium text-neutral-700 dark:text-neutral-400 mb-2">Tanggal Daftar <span class="text-error">(wajib)</span></label>
						<div class="relative">
							<div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
								<Calendar class="w-4 h-4 text-neutral-500" />
							</div>
							<input
								id="tanggal_daftar"
								type="date"
								bind:value={csForm.tanggal_daftar}
								required
								disabled={identityLocked}
								class="w-full pl-12 pr-4 py-3 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white transition-all outline-none"
							/>
						</div>
					</div>
				</div>

				<div class="grid md:grid-cols-2 gap-5">
					<div>
						<label for="usia" class="block text-sm font-medium text-neutral-700 dark:text-neutral-400 mb-2">Usia</label>
						<div class="relative">
							<div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
								<Users class="w-4 h-4 text-neutral-500" />
							</div>
							<input
								id="usia"
								type="number"
								bind:value={csForm.usia}
								class="w-full pl-12 pr-4 py-3 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white placeholder-neutral-500 transition-all outline-none"
								placeholder="0"
							/>
						</div>
					</div>
					<div>
						<label for="domisili" class="block text-sm font-medium text-neutral-700 dark:text-neutral-400 mb-2">Domisili</label>
						<div class="relative">
							<div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
								<MapPin class="w-4 h-4 text-neutral-500" />
							</div>
							<input
								id="domisili"
								type="text"
								bind:value={csForm.domisili}
								class="w-full pl-12 pr-4 py-3 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white placeholder-neutral-500 transition-all outline-none"
								placeholder="Kota domisili"
							/>
						</div>
					</div>
				</div>

				{#if !readonly}
					<div class="pt-4">
						<button
							type="submit"
							disabled={isCsLoading}
							class="w-full md:w-auto px-6 py-3 rounded-xl bg-brand-600 hover:bg-brand-700 text-white font-semibold transition-all dark:bg-brand-500 dark:hover:bg-brand-400 shadow-lg shadow-brand-600/25 hover:shadow-brand-600/40 disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
						>
							<Save class="w-4 h-4" />
							{isCsLoading ? 'Menyimpan...' : 'Simpan'}
						</button>
					</div>
				{/if}
				</fieldset>
			</form>
		</div>

		{#if isAdminKelas}
			<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-6" in:fly={{ y: 20, duration: 600 }}>
				<div class="flex items-center gap-3 mb-6">
					<div class="w-10 h-10 rounded-xl bg-secondary-500/15 flex items-center justify-center">
						<BookOpen class="w-5 h-5 text-secondary-500" />
					</div>
					<div>
						<h3 class="text-lg font-semibold text-neutral-900 dark:text-white">Data Kelas</h3>
						<p class="text-sm text-neutral-600 dark:text-neutral-500">Informasi kelas, level, dan pengajaran</p>
					</div>
				</div>

				<form onsubmit={handleKelasSubmit}>
					<fieldset disabled={kelasReadonly} class="space-y-5 border-0 p-0 m-0 min-w-0">
					<div class="grid md:grid-cols-2 gap-5">
						<div>
							<label for="angkatan_kelas" class="block text-sm font-medium text-neutral-700 dark:text-neutral-400 mb-2">Angkatan Kelas <span class="text-error">(wajib)</span></label>
							<select id="angkatan_kelas" bind:value={kelasForm.angkatan_kelas} required class="w-full px-4 py-3 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white transition-all outline-none">
								<option value="">Pilih Angkatan Kelas</option>
								{#if hasLegacyAngkatanKelas}
									<option value={kelasForm.angkatan_kelas}>{kelasForm.angkatan_kelas} (data lama)</option>
								{/if}
								{#each angkatan as a}
									<option value={a.kode || String(a.id)}>{a.keterangan || a.kode || String(a.id)}</option>
								{/each}
							</select>
							<p class="mt-1 text-xs text-neutral-500">Terpisah dari Angkatan Pendaftaran ({santri?.angkatan || "-"}).</p>
						</div>
						<div>
							<label for="fu" class="block text-sm font-medium text-neutral-700 dark:text-neutral-400 mb-2">FU</label>
							<div class="relative">
								<div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
									<ClipboardList class="w-4 h-4 text-neutral-500" />
								</div>
								<input
									id="fu"
									type="text"
									bind:value={kelasForm.fu}
									class="w-full pl-12 pr-4 py-3 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white placeholder-neutral-500 transition-all outline-none"
									placeholder="FU"
								/>
							</div>
						</div>
						<div>
							<label for="tanggal_vn" class="block text-sm font-medium text-neutral-700 dark:text-neutral-400 mb-2">Tanggal VN</label>
							<div class="relative">
								<div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
									<Calendar class="w-4 h-4 text-neutral-500" />
								</div>
								<input
									id="tanggal_vn"
									type="date"
									bind:value={kelasForm.tanggal_vn}
									class="w-full pl-12 pr-4 py-3 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white transition-all outline-none"
								/>
							</div>
						</div>
					</div>

					<div class="rounded-xl border border-secondary-500/20 bg-secondary-500/5 p-4 space-y-4">
						<div>
							<h4 class="text-sm font-semibold text-neutral-900 dark:text-white">Voice Note</h4>
							<p class="text-xs text-neutral-600 dark:text-neutral-400 mt-1">Unggah rekaman VN dan tambahkan keterangannya.</p>
						</div>
						{#if santri?.voice_note_url}
							<audio controls src={santri.voice_note_url} class="w-full h-10">Browser tidak mendukung pemutar audio.</audio>
						{/if}
						<div>
							<label for="keterangan_vn" class="block text-sm font-medium text-neutral-700 dark:text-neutral-400 mb-2">Keterangan VN</label>
							<textarea
								id="keterangan_vn"
								bind:value={voiceNoteDescription}
								rows="3"
								placeholder="Tulis keterangan voice note"
								class="w-full px-4 py-3 rounded-xl bg-white dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white placeholder-neutral-500 transition-all outline-none"
							></textarea>
						</div>
						<div class="flex flex-col sm:flex-row sm:items-center gap-3">
							<label class="inline-flex items-center justify-center gap-2 px-4 py-2.5 rounded-xl border border-neutral-300 dark:border-neutral-700/80 text-sm font-medium text-neutral-700 dark:text-neutral-300 cursor-pointer hover:bg-neutral-100 dark:hover:bg-neutral-800 transition-colors">
								<Upload class="w-4 h-4" />
								<span>{voiceNoteFile?.name || "Pilih file audio"}</span>
								<input type="file" accept="audio/*" class="sr-only" onchange={handleVoiceNoteFile} />
							</label>
							<button
								type="button"
								onclick={saveVoiceNote}
								disabled={isVoiceNoteLoading || !santri?.id}
								class="inline-flex items-center justify-center gap-2 px-4 py-2.5 rounded-xl bg-secondary-500 hover:bg-secondary-600 text-white text-sm font-semibold disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
							>
								<Save class="w-4 h-4" />
								{isVoiceNoteLoading ? "Menyimpan VN..." : "Simpan VN"}
							</button>
						</div>
					</div>

					<div class="grid md:grid-cols-2 gap-5">
						<div>
							<label for="hasil_vn" class="block text-sm font-medium text-neutral-700 dark:text-neutral-400 mb-2">Hasil VN</label>
							<div class="relative">
								<div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
									<ClipboardList class="w-4 h-4 text-neutral-500" />
								</div>
								<input
									id="hasil_vn"
									type="text"
									bind:value={kelasForm.hasil_vn}
									class="w-full pl-12 pr-4 py-3 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white placeholder-neutral-500 transition-all outline-none"
									placeholder="Hasil VN"
								/>
							</div>
						</div>
						<div>
							<label for="masuk_grup" class="block text-sm font-medium text-neutral-700 dark:text-neutral-400 mb-2">Masuk Grup</label>
							<div class="relative">
								<div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
									<Users class="w-4 h-4 text-neutral-500" />
								</div>
								<input
									id="masuk_grup"
									type="text"
									bind:value={kelasForm.masuk_grup}
									class="w-full pl-12 pr-4 py-3 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white placeholder-neutral-500 transition-all outline-none"
									placeholder="Nama grup"
								/>
							</div>
						</div>
					</div>

					<div class="grid md:grid-cols-2 gap-5">
						<div>
							<label for="mulai_belajar" class="block text-sm font-medium text-neutral-700 dark:text-neutral-400 mb-2">Mulai Belajar</label>
							<div class="relative">
								<div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
									<Calendar class="w-4 h-4 text-neutral-500" />
								</div>
								<input
									id="mulai_belajar"
									type="date"
									bind:value={kelasForm.mulai_belajar}
									class="w-full pl-12 pr-4 py-3 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white transition-all outline-none"
								/>
							</div>
						</div>
						<div>
							<label for="jumlah" class="block text-sm font-medium text-neutral-700 dark:text-neutral-400 mb-2">Jumlah</label>
							<div class="relative">
								<div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
									<Hash class="w-4 h-4 text-neutral-500" />
								</div>
								<input
									id="jumlah"
									type="number"
									bind:value={kelasForm.jumlah}
									class="w-full pl-12 pr-4 py-3 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white placeholder-neutral-500 transition-all outline-none"
									placeholder="0"
								/>
							</div>
						</div>
					</div>

					<div class="grid md:grid-cols-2 gap-5">
						<div>
							<label for="level" class="block text-sm font-medium text-neutral-700 dark:text-neutral-400 mb-2">Level</label>
							<div class="relative">
								<div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
									<BookOpen class="w-4 h-4 text-neutral-500" />
								</div>
								<select
									id="level"
									bind:value={kelasForm.level}
									class="w-full pl-12 pr-4 py-3 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white transition-all outline-none"
								>
									<option value="">Pilih Level</option>
									{#each levels as l}
										<option value={l.kode || String(l.id)}>{l.nama || l.kode || String(l.id)}</option>
									{/each}
								</select>
							</div>
						</div>
						<div>
							<div class="flex items-center justify-between mb-2">
								<label class="block text-sm font-medium text-neutral-700 dark:text-neutral-400">
									Jadwal{#if frekuensiLabel}<span class="ml-1 text-xs font-normal text-neutral-400">· {frekuensiLabel}</span>{/if}
								</label>
								{#if !kelasReadonly}
									<button type="button" onclick={addJadwalRow}
										class="inline-flex items-center gap-1 text-xs font-medium text-brand-600 dark:text-brand-400 hover:underline">
										<Plus class="w-3.5 h-3.5" /> Tambah sesi
									</button>
								{/if}
							</div>
							<div class="space-y-2">
								{#each jadwalRows as _row, i}
									<div class="flex items-center gap-2">
										<div class="relative flex-1">
											<div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
												<Calendar class="w-4 h-4 text-neutral-500" />
											</div>
											<select
												bind:value={jadwalRows[i]}
												class="w-full pl-12 pr-4 py-3 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white transition-all outline-none"
											>
												<option value="">Pilih Jadwal</option>
												{#each jadwals as j}
													<option value={j.nama || String(j.id)}>{j.nama || String(j.id)}</option>
												{/each}
											</select>
										</div>
										{#if jadwalRows.length > 1}
											<button type="button" onclick={() => removeJadwalRow(i)} aria-label="Hapus sesi"
												class="shrink-0 p-2.5 rounded-xl text-neutral-400 hover:text-red-500 hover:bg-red-500/10 transition-colors">
												<Trash2 class="w-4 h-4" />
											</button>
										{/if}
									</div>
								{/each}
							</div>
						</div>
					</div>

					<div>
						<label for="guru" class="block text-sm font-medium text-neutral-700 dark:text-neutral-400 mb-2">Guru</label>
						<div class="relative">
							<div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
								<Users class="w-4 h-4 text-neutral-500" />
							</div>
							<select
								id="guru"
								bind:value={kelasForm.guru_id}
								class="w-full pl-12 pr-4 py-3 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white transition-all outline-none"
							>
								<option value="">Pilih Guru</option>
								{#each gurus as g}
									<option value={g.id}>{g.nama || String(g.id)}</option>
								{/each}
							</select>
						</div>
					</div>

					{#if !kelasReadonly}
						<div class="pt-4">
							<button
								type="submit"
								disabled={isKelasLoading || !santri?.id}
								class="w-full md:w-auto px-6 py-3 rounded-xl bg-secondary-500 hover:bg-secondary-600 text-white font-semibold transition-all shadow-lg shadow-secondary-500/25 hover:shadow-secondary-500/40 disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
							>
								<Save class="w-4 h-4" />
								{isKelasLoading ? 'Menyimpan...' : 'Simpan Data Kelas'}
							</button>
						</div>
					{/if}
					</fieldset>
				</form>
			</div>
		{/if}

		{#if isEdit && user?.role === "super_admin"}
			<section class="rounded-2xl border border-warning/30 bg-warning/5 p-6" aria-labelledby="identity-correction-title">
				<div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
					<div>
						<h2 id="identity-correction-title" class="text-base font-semibold text-neutral-900 dark:text-white">Koreksi Identitas Pendaftaran</h2>
						<p class="mt-1 text-sm text-neutral-600 dark:text-neutral-400">Tindakan khusus ini mengubah metadata pendaftaran dan menerbitkan ulang ID Mahasantri.</p>
					</div>
					<button type="button" onclick={() => correctionDialog?.showModal()} class="shrink-0 rounded-xl border border-warning/40 px-4 py-2.5 text-sm font-semibold text-warning hover:bg-warning/10">Koreksi Identitas</button>
				</div>
			</section>
			<section class="rounded-2xl border border-error/25 bg-error/5 p-6" aria-labelledby="danger-zone-title">
				<div class="flex flex-col gap-5 sm:flex-row sm:items-center sm:justify-between">
					<div class="max-w-2xl">
						<h2 id="danger-zone-title" class="text-lg font-semibold text-neutral-900 dark:text-white">Zona Berbahaya</h2>
						<p class="mt-1 text-sm leading-6 text-neutral-600 dark:text-neutral-400">
							Hapus santri beserta absensi, tagihan, catatan riayah, dan file voice note secara permanen. Kelas tetap disimpan.
						</p>
					</div>
					<button
						type="button"
						onclick={openDeleteDialog}
						class="inline-flex w-full shrink-0 items-center justify-center gap-2 rounded-xl border border-error/40 px-4 py-2.5 text-sm font-semibold text-error transition-colors hover:bg-error hover:text-white sm:w-auto"
					>
						<Trash2 class="h-4 w-4" />
						Hapus Permanen
					</button>
				</div>
			</section>
		{/if}
	</div>

	{#if isEdit && user?.role === "super_admin"}
		<dialog bind:this={correctionDialog} class="m-auto w-[calc(100%-2rem)] max-w-lg rounded-2xl border border-neutral-200 bg-white p-0 shadow-xl backdrop:bg-neutral-950/70 dark:border-white/[0.06] dark:bg-neutral-925">
			<form onsubmit={handleCorrectionSubmit} class="space-y-4 p-6">
				<div><h2 class="text-lg font-semibold text-neutral-900 dark:text-white">Terbitkan Ulang ID</h2><p class="mt-1 text-sm text-neutral-600 dark:text-neutral-400">ID saat ini: <span class="font-mono">{santri?.id_mahasantri}</span>. Pastikan data koreksi sudah diverifikasi.</p></div>
				<label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300">Angkatan Pendaftaran<select bind:value={correctionForm.angkatan} required class="mt-2 w-full rounded-xl border border-neutral-300 bg-neutral-50 px-4 py-3 dark:border-neutral-700 dark:bg-neutral-900">{#each angkatan as a}<option value={a.kode || String(a.id)}>{a.keterangan || a.kode || String(a.id)}</option>{/each}</select></label>
				<label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300">Tanggal Pendaftaran<input type="date" bind:value={correctionForm.tanggal_daftar} required class="mt-2 w-full rounded-xl border border-neutral-300 bg-neutral-50 px-4 py-3 dark:border-neutral-700 dark:bg-neutral-900" /></label>
				<label class="flex items-start gap-2 text-sm text-neutral-700 dark:text-neutral-300"><input type="checkbox" bind:checked={correctionForm.konfirmasi} class="mt-1" />Saya memahami ID Mahasantri akan diterbitkan ulang.</label>
				<div class="flex justify-end gap-3"><button type="button" onclick={() => correctionDialog?.close()} class="rounded-xl border border-neutral-300 px-4 py-2.5 text-sm font-semibold dark:border-neutral-700">Batal</button><button type="submit" disabled={!correctionForm.konfirmasi || isCorrectionLoading} class="rounded-xl bg-warning px-4 py-2.5 text-sm font-semibold text-white disabled:opacity-50">{isCorrectionLoading ? "Menyimpan..." : "Terbitkan Ulang"}</button></div>
			</form>
		</dialog>
		<dialog
			bind:this={deleteDialog}
			aria-labelledby="delete-santri-title"
			aria-describedby="delete-santri-description"
			onclick={handleDeleteBackdrop}
			onclose={() => { deleteName = ""; }}
			oncancel={(event) => { if (isDeleteLoading) event.preventDefault(); }}
			class="m-auto w-[calc(100%-2rem)] max-w-lg rounded-2xl border border-neutral-200 bg-white p-0 text-left shadow-xl backdrop:bg-neutral-950/70 dark:border-white/[0.06] dark:bg-neutral-925"
		>
			<form onsubmit={handleDeleteSubmit} class="p-5 sm:p-6">
				<div class="flex items-start gap-3">
					<div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-error/10 text-error">
						<Trash2 class="h-5 w-5" />
					</div>
					<div>
						<h2 id="delete-santri-title" class="text-lg font-semibold text-neutral-900 dark:text-white">Hapus {santri?.nama}?</h2>
						<p id="delete-santri-description" class="mt-1 text-sm leading-6 text-neutral-600 dark:text-neutral-400">
							Tindakan ini tidak dapat dibatalkan. Data santri, absensi, tagihan, catatan riayah, dan file voice note akan dihapus permanen.
						</p>
					</div>
				</div>

				<div class="mt-5">
					<label for="delete-santri-name" class="block text-sm font-medium text-neutral-700 dark:text-neutral-300">
						Ketik <span class="font-semibold text-neutral-900 dark:text-white">{santri?.nama}</span> untuk konfirmasi
					</label>
					<input
						bind:this={deleteNameInput}
						id="delete-santri-name"
						type="text"
						autocomplete="off"
						bind:value={deleteName}
						class="mt-2 w-full rounded-xl border border-neutral-300 bg-neutral-50 px-4 py-3 text-neutral-900 outline-none transition-colors focus:border-error focus:ring-2 focus:ring-error/15 dark:border-neutral-700 dark:bg-neutral-900 dark:text-white"
					/>
				</div>

				<div class="mt-6 flex flex-col-reverse gap-3 sm:flex-row sm:justify-end">
					<button type="button" onclick={closeDeleteDialog} disabled={isDeleteLoading} class="rounded-xl border border-neutral-300 px-4 py-2.5 text-sm font-semibold text-neutral-700 transition-colors hover:bg-neutral-100 disabled:opacity-50 dark:border-neutral-700 dark:text-neutral-300 dark:hover:bg-neutral-800">
						Batal
					</button>
					<button type="submit" disabled={!canDelete} class="inline-flex items-center justify-center gap-2 rounded-xl bg-error px-4 py-2.5 text-sm font-semibold text-white transition-colors hover:bg-error/90 disabled:cursor-not-allowed disabled:opacity-50">
						<Trash2 class="h-4 w-4" />
						{isDeleteLoading ? "Menghapus..." : "Hapus Permanen"}
					</button>
				</div>
			</form>
		</dialog>
	{/if}
</AppLayout>
