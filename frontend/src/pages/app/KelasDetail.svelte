<script lang="ts">
	import { inertia, router } from "@inertiajs/svelte";
	import { fly } from "svelte/transition";
	import AppLayout from "@layouts/AppLayout.svelte";
	import StatusBadge from "@components/StatusBadge.svelte";
	import GenderBadge from "@components/GenderBadge.svelte";
	import type { Flash, User } from "@lib/types";
	import { Toast } from "@lib/notifications/toast";
	import {
		ArrowLeft, Users, BookOpen, UserCheck, MoveRight, Calendar, Clock, GraduationCap, ChevronDown, UserX, Power, Trash2,
		Layers, CalendarClock, History, ArrowUpCircle, Pencil, UserCog
	} from "lucide-svelte";

	interface KelasResponse {
		id: number;
		kunci_kelas: string;
		angkatan: string;
		tipe: string;
		jenis_kelamin: string;
		level: string;
		frekuensi: string;
		jadwal: string;
		sub_index: number;
		nama_kelas: string;
		guru_id: number | null;
		guru_nama?: string;
		kapasitas: number;
		jumlah_santri: number;
		pertemuan_terakhir: number;
		tanggal_pertemuan_terakhir: string;
		materi_terakhir: string;
		materi_individual: boolean;
		is_aktif: boolean;
		created_at: string;
	}

	interface SantriResponse {
		id: number;
		id_mahasantri: string;
		nama: string;
		status: string;
		status_alasan: string;
		cuti_mulai: string;
		cuti_selesai: string;
		jenis_kelamin: string;
		kelas_id?: number;
		total_hadir: number;
		total_izin: number;
		total_sakit: number;
		total_alpa: number;
		total_telat: number;
		persen_hadir: number;
	}

	interface Guru {
		id: number;
		nama: string;
		jenis_kelamin: string;
	}

	interface LevelOption {
		id: number;
		kode: string;
		nama: string;
		urutan: number;
	}

	interface JadwalOption {
		id: number;
		nama: string;
	}

	interface KelasPerubahan {
		id: number;
		jenis: "level_kelas" | "jadwal_kelas" | "level_santri";
		nilai_lama: string;
		nilai_baru: string;
		pertemuan_ke: number;
		santri_nama: string;
		kelas_asal_id: number | null;
		kelas_asal_nama: string;
		kelas_tujuan_id: number | null;
		kelas_tujuan_nama: string;
		dibuat_oleh_nama: string;
		created_at: string;
	}

	interface SantriStatusLog {
		id: number;
		santri_id: number;
		santri_nama: string;
		status_lama: string;
		status_baru: string;
		alasan: string;
		cuti_mulai: string;
		cuti_selesai: string;
		dibuat_oleh_nama: string;
		created_at: string;
	}

	interface Props {
		user?: User;
		kelas: KelasResponse;
		santri: SantriResponse[];
		gurus: Guru[];
		kelas_lain?: KelasResponse[];
		has_pertemuan?: boolean;
		levels?: LevelOption[];
		jadwals?: JadwalOption[];
		riwayat_perubahan?: KelasPerubahan[];
		riwayat_status?: SantriStatusLog[];
		return_to?: string;
		flash?: Flash;
		success?: string;
		error?: string;
	}

	let { user, kelas, santri = [], gurus = [], kelas_lain = [], has_pertemuan = false, levels = [], jadwals = [], riwayat_perubahan = [], riwayat_status = [], return_to = "/app/kelas", flash, success, error }: Props = $props();

	// Kelas handlers report results via the session flash (props.flash); a failed
	// action still redirects, so Inertia calls onSuccess and we must inspect it.
	let pesanSukses = $derived(flash?.success ?? success);
	let pesanError = $derived(flash?.error ?? error);

	function flashError(p: { props: Record<string, unknown> }): string | undefined {
		return (p.props.flash as Flash | undefined)?.error;
	}

	let selectedGuruId = $state<number | null>(null);
	let isAssignLoading = $state(false);
	let showGuruDropdown = $state(false);
	let pertemuanTerakhir = $state(kelas.pertemuan_terakhir);
	let isPertemuanLoading = $state(false);
	let isMateriIndividualLoading = $state(false);

	let pindahModal = $state<{ open: boolean; santriId: number; santriNama: string }>({ open: false, santriId: 0, santriNama: "" });
	let pindahKelasId = $state<number | null>(null);
	let isPindahLoading = $state(false);

	let sortedLevels = $derived([...levels].sort((a, b) => a.urutan - b.urutan));
	let levelKelasNama = $derived(namaLevel(kelas.level));

	function namaLevel(kode: string): string {
		return levels.find((l) => l.kode === kode)?.nama || kode;
	}

	// Level setelah level kelas saat ini (sesuai urutan master), untuk default pilihan.
	function levelBerikutnya(): string {
		const idx = sortedLevels.findIndex((l) => l.kode === kelas.level);
		const next = sortedLevels[idx + 1] ?? sortedLevels.find((l) => l.kode !== kelas.level);
		return next?.kode ?? "";
	}

	function withReturnTo(path: string): string {
		return `${path}?return_to=${encodeURIComponent(return_to)}`;
	}

	// --- Ganti level kelas ---
	let levelModalOpen = $state(false);
	let levelBaru = $state("");
	let isLevelLoading = $state(false);

	function openLevelModal() {
		levelBaru = levelBerikutnya();
		levelModalOpen = true;
	}

	function submitGantiLevel() {
		if (!levelBaru || levelBaru === kelas.level) return;
		isLevelLoading = true;
		router.put(withReturnTo(`/app/kelas/${kelas.id}/level`), { level: levelBaru }, {
			preserveScroll: true,
			preserveState: true,
			onSuccess: (p) => {
				const err = flashError(p);
				if (err) { Toast(err, "error"); return; }
				levelModalOpen = false;
			},
			onFinish: () => { isLevelLoading = false; },
		});
	}

	// --- Ganti jadwal kelas ---
	let jadwalModalOpen = $state(false);
	let jadwalBaru = $state("");
	let isJadwalLoading = $state(false);

	function openJadwalModal() {
		jadwalBaru = "";
		jadwalModalOpen = true;
	}

	function submitGantiJadwal() {
		if (!jadwalBaru || jadwalBaru === kelas.jadwal) return;
		isJadwalLoading = true;
		router.put(withReturnTo(`/app/kelas/${kelas.id}/jadwal`), { jadwal: jadwalBaru }, {
			preserveScroll: true,
			preserveState: true,
			onSuccess: (p) => {
				const err = flashError(p);
				if (err) { Toast(err, "error"); return; }
				jadwalModalOpen = false;
			},
			onFinish: () => { isJadwalLoading = false; },
		});
	}

	// --- Ganti level per santri ---
	let selectedSantri = $state<number[]>([]);
	let levelSantriModal = $state<{ open: boolean; santriIds: number[] }>({ open: false, santriIds: [] });
	let levelSantriTujuan = $state("");
	let levelSantriMode = $state<"kelas" | "baru">("kelas");
	let levelSantriKelasId = $state<number | null>(null);
	let isLevelSantriLoading = $state(false);

	let kelasTujuanLevel = $derived(
		kelas_lain.filter((k) => k.level === levelSantriTujuan && k.jenis_kelamin === kelas.jenis_kelamin)
	);
	let namaSantriTerpilih = $derived(
		levelSantriModal.santriIds.map((id) => santri.find((s) => s.id === id)?.nama).filter(Boolean).join(", ")
	);

	function toggleSantri(id: number) {
		selectedSantri = selectedSantri.includes(id) ? selectedSantri.filter((x) => x !== id) : [...selectedSantri, id];
	}

	function toggleSemuaSantri() {
		selectedSantri = selectedSantri.length === santriRoster.length ? [] : santriRoster.map((s) => s.id);
	}

	function openLevelSantriModal(ids: number[]) {
		levelSantriModal = { open: true, santriIds: ids };
		levelSantriTujuan = levelBerikutnya();
		levelSantriKelasId = null;
		levelSantriMode = "kelas";
	}

	function pilihLevelSantri(kode: string) {
		levelSantriTujuan = kode;
		levelSantriKelasId = null;
		levelSantriMode = kelas_lain.some((k) => k.level === kode && k.jenis_kelamin === kelas.jenis_kelamin) ? "kelas" : "baru";
	}

	function submitLevelSantri() {
		const buatBaru = levelSantriMode === "baru";
		if (!levelSantriTujuan || (!buatBaru && !levelSantriKelasId)) return;
		isLevelSantriLoading = true;
		router.post(withReturnTo(`/app/kelas/${kelas.id}/ganti-level-santri`), {
			santri_ids: levelSantriModal.santriIds,
			kelas_tujuan_id: buatBaru ? 0 : levelSantriKelasId,
			buat_kelas_baru: buatBaru,
			level: levelSantriTujuan,
		}, {
			preserveScroll: true,
			preserveState: true,
			onSuccess: (p) => {
				const err = flashError(p);
				if (err) { Toast(err, "error"); return; }
				levelSantriModal = { open: false, santriIds: [] };
				selectedSantri = [];
			},
			onFinish: () => { isLevelSantriLoading = false; },
		});
	}

	function labelPerubahan(item: KelasPerubahan): string {
		if (item.jenis === "level_kelas") return "Ganti level kelas";
		if (item.jenis === "jadwal_kelas") return "Ganti jadwal kelas";
		return "Santri naik level";
	}

	let santriAktif = $derived(santri.filter((s) => s.status === "aktif"));
	let santriCuti = $derived(santri.filter((s) => s.status === "cuti"));
	let santriNonaktif = $derived(santri.filter((s) => s.status === "nonaktif"));
	let santriTidakLanjut = $derived(santri.filter((s) => s.status === "tidak_lanjut"));
	// Roster = santri holding a seat (aktif + cuti); nonaktif & tidak lanjut are
	// shown separately because they no longer count toward occupancy.
	let santriRoster = $derived(santri.filter((s) => s.status === "aktif" || s.status === "cuti"));
	let santriKeluar = $derived(santri.filter((s) => s.status === "nonaktif" || s.status === "tidak_lanjut"));

	const labelStatus: Record<string, string> = { aktif: "Aktif", cuti: "Cuti", nonaktif: "Nonaktif", tidak_lanjut: "Tidak Lanjut" };

	function formatTanggal(iso: string): string {
		if (!iso) return "";
		const [y, m, d] = iso.split("-").map(Number);
		return new Date(y, m - 1, d).toLocaleDateString("id-ID", { day: "numeric", month: "short", year: "numeric" });
	}

	function hariIni(): string {
		const d = new Date();
		return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
	}

	// --- Ubah status santri (aktif / cuti / nonaktif) ---
	let statusModal = $state<{ open: boolean; santri: SantriResponse | null }>({ open: false, santri: null });
	let statusBaru = $state<"aktif" | "cuti" | "nonaktif">("cuti");
	let statusAlasan = $state("");
	let cutiMulai = $state("");
	let cutiSelesai = $state("");
	let isUbahStatusLoading = $state(false);

	function openStatusModal(s: SantriResponse) {
		statusModal = { open: true, santri: s };
		statusBaru = s.status === "aktif" ? "cuti" : s.status === "cuti" ? "cuti" : "aktif";
		statusAlasan = s.status === "cuti" || s.cuti_mulai ? s.status_alasan : "";
		// An aktif santri with cuti dates has a scheduled cuti; prefill it for editing.
		cutiMulai = s.cuti_mulai ? s.cuti_mulai : hariIni();
		cutiSelesai = s.cuti_selesai;
	}

	function closeStatusModal() {
		statusModal = { open: false, santri: null };
	}

	let statusFormValid = $derived.by(() => {
		const current = statusModal.santri;
		if (!current) return false;
		// aktif -> aktif is allowed only to cancel a scheduled cuti.
		if (statusBaru === current.status && statusBaru !== "cuti" && !(statusBaru === "aktif" && current.cuti_mulai)) return false;
		if (statusBaru === "aktif") return true;
		if (!statusAlasan.trim()) return false;
		if (statusBaru === "cuti") return !!cutiMulai && !!cutiSelesai && cutiSelesai >= cutiMulai;
		return true;
	});

	function submitUbahStatus() {
		const current = statusModal.santri;
		if (!current || !statusFormValid) return;
		isUbahStatusLoading = true;
		router.put(withReturnTo(`/app/kelas/${kelas.id}/santri/${current.id}/status`), {
			status: statusBaru,
			alasan: statusBaru === "aktif" ? "" : statusAlasan.trim(),
			cuti_mulai: statusBaru === "cuti" ? cutiMulai : "",
			cuti_selesai: statusBaru === "cuti" ? cutiSelesai : "",
		}, {
			preserveScroll: true,
			preserveState: true,
			onSuccess: (p) => {
				const err = flashError(p);
				if (err) { Toast(err, "error"); return; }
				// preserveState keeps the selection; drop santri that left the roster.
				selectedSantri = selectedSantri.filter((id) => santriRoster.some((s) => s.id === id));
				closeStatusModal();
			},
			onFinish: () => { isUbahStatusLoading = false; },
		});
	}

	let capacityPercent = $derived(kelas.kapasitas > 0 ? Math.round((kelas.jumlah_santri / kelas.kapasitas) * 100) : 0);

	function capacityColor(pct: number): string {
		if (pct >= 90) return "bg-red-500";
		if (pct >= 75) return "bg-amber-500";
		return "bg-brand-500";
	}

	function handleAssignGuru() {
		if (!selectedGuruId) return;
		isAssignLoading = true;
		router.put(`/app/kelas/${kelas.id}/guru?return_to=${encodeURIComponent(return_to)}`, { guru_id: selectedGuruId }, {
			preserveScroll: true,
			onSuccess: (p) => {
				const err = flashError(p);
				if (err) { Toast(err, "error"); return; }
				Toast("Guru berhasil di-assign", "success");
				showGuruDropdown = false;
			},
			onError: () => {
				Toast("Gagal meng-assign guru", "error");
			},
			onFinish: () => {
				isAssignLoading = false;
			},
		});
	}

	function simpanPertemuanTerakhir() {
		if (pertemuanTerakhir < 0) return;
		isPertemuanLoading = true;
		router.put(`/app/kelas/${kelas.id}/pertemuan-terakhir?return_to=${encodeURIComponent(return_to)}`, { pertemuan_terakhir: pertemuanTerakhir }, {
			preserveScroll: true,
			onFinish: () => { isPertemuanLoading = false; },
		});
	}

	function toggleMateriIndividual() {
		isMateriIndividualLoading = true;
		router.put(`/app/kelas/${kelas.id}/materi-individual?return_to=${encodeURIComponent(return_to)}`, { materi_individual: !kelas.materi_individual }, {
			preserveScroll: true,
			onSuccess: (p) => {
				const err = flashError(p);
				Toast(err ?? "Pengaturan progres materi tersimpan", err ? "error" : "success");
			},
			onError: () => Toast("Gagal mengubah pengaturan progres materi", "error"),
			onFinish: () => { isMateriIndividualLoading = false; },
		});
	}

	function openPindahModal(s: SantriResponse) {
		pindahModal = { open: true, santriId: s.id, santriNama: s.nama };
		pindahKelasId = null;
	}

	let isStatusLoading = $state(false);

	function toggleAktif() {
		isStatusLoading = true;
		const menjadiAktif = !kelas.is_aktif;
		router.put(`/app/kelas/${kelas.id}/status?from_detail=1&return_to=${encodeURIComponent(return_to)}`, { is_aktif: menjadiAktif }, {
			preserveScroll: true,
			onSuccess: (p) => {
				const err = flashError(p);
				Toast(err ?? (menjadiAktif ? "Kelas diaktifkan" : "Kelas dinonaktifkan"), err ? "error" : "success");
			},
			onError: () => Toast("Gagal mengubah status kelas", "error"),
			onFinish: () => { isStatusLoading = false; },
		});
	}

	function hapusKelas() {
		if (!confirm(`Hapus kelas "${kelas.nama_kelas}"?\nSantri di kelas ini akan dilepas (kelasnya dikosongkan).`)) return;
		router.delete(`/app/kelas/${kelas.id}?return_to=${encodeURIComponent(return_to)}`, {
			onError: () => Toast("Gagal menghapus kelas", "error"),
		});
	}

	function handlePindahkanSantri() {
		if (!pindahKelasId) return;
		isPindahLoading = true;
		router.post("/app/santri/" + pindahModal.santriId + "/pindah", { kelas_tujuan_id: pindahKelasId }, {
			preserveScroll: true,
			onSuccess: (p) => {
				const err = flashError(p);
				if (err) { Toast(err, "error"); return; }
				Toast("Santri berhasil dipindahkan", "success");
				pindahModal = { open: false, santriId: 0, santriNama: "" };
			},
			onError: () => {
				Toast("Gagal memindahkan santri", "error");
			},
			onFinish: () => {
				isPindahLoading = false;
			},
		});
	}

</script>

<AppLayout {user} group="dashboard">
	<div class="pt-8 pb-10 border-b border-neutral-200/80 dark:border-white/[0.04]">
		<div class="max-w-6xl mx-auto px-4 sm:px-6">
			<div class="flex items-center gap-2 text-sm text-neutral-500 dark:text-neutral-400 mb-4">
				<a href="/app" use:inertia class="hover:text-brand-600 dark:hover:text-brand-400 transition-colors">Dashboard</a>
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
				</svg>
				<a href={return_to} use:inertia class="hover:text-brand-600 dark:hover:text-brand-400 transition-colors">Kelas</a>
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
				</svg>
				<span class="text-neutral-700 dark:text-neutral-300">{kelas.nama_kelas}</span>
			</div>

			<a href={return_to} use:inertia class="inline-flex items-center gap-1.5 text-sm text-neutral-500 hover:text-brand-600 dark:hover:text-brand-400 transition-colors mb-4">
				<ArrowLeft class="w-4 h-4" />
				Kembali ke daftar kelas
			</a>

			<div class="flex items-start justify-between gap-4 flex-wrap">
				<div class="flex items-center gap-4 min-w-0">
					<div class="w-14 h-14 rounded-2xl bg-brand-400/15 flex items-center justify-center shrink-0">
						<BookOpen class="w-7 h-7 text-brand-600 dark:text-brand-400" />
					</div>
					<div class="min-w-0">
						<div class="flex items-center gap-2.5 flex-wrap mb-1">
							<h1 class="text-2xl sm:text-3xl font-bold text-neutral-900 dark:text-white tracking-tight break-words">{kelas.nama_kelas}</h1>
							<GenderBadge gender={kelas.jenis_kelamin} />
							{#if kelas.is_aktif}
								<span class="px-2 py-0.5 rounded-md bg-green-500/10 text-green-700 dark:text-green-400 text-xs font-semibold">Aktif</span>
							{:else}
								<span class="px-2 py-0.5 rounded-md bg-neutral-200 dark:bg-neutral-800 text-neutral-500 dark:text-neutral-400 text-xs font-semibold">Nonaktif</span>
							{/if}
						</div>
						<p class="text-neutral-600 dark:text-neutral-400">
						Angkatan Kelas {kelas.angkatan} · {kelas.tipe}{kelas.sub_index > 0 ? ` · Sub ${kelas.sub_index}` : ""} · {levelKelasNama} · {kelas.frekuensi}
						</p>
					</div>
				</div>

				<div class="flex items-center gap-2 shrink-0">
					<button
						onclick={toggleAktif}
						disabled={isStatusLoading}
						class="inline-flex items-center gap-1.5 px-3.5 py-2 rounded-xl text-sm font-semibold transition-colors disabled:opacity-50 {kelas.is_aktif ? 'bg-neutral-100 dark:bg-neutral-800 text-neutral-700 dark:text-neutral-300 hover:bg-neutral-200 dark:hover:bg-neutral-700' : 'bg-green-600 hover:bg-green-700 text-white'}"
					>
						<Power class="w-4 h-4" />
						{kelas.is_aktif ? "Nonaktifkan" : "Aktifkan"}
					</button>
					<button
						onclick={hapusKelas}
						aria-label="Hapus kelas"
						class="inline-flex items-center gap-1.5 px-3.5 py-2 rounded-xl text-sm font-semibold bg-red-500/10 text-red-600 dark:text-red-400 hover:bg-red-500/20 transition-colors"
					>
						<Trash2 class="w-4 h-4" />
						<span class="hidden sm:inline">Hapus</span>
					</button>
				</div>
			</div>
		</div>
	</div>

	<div class="relative max-w-6xl mx-auto px-4 sm:px-6 py-8 space-y-6">
		{#if pesanSukses}
			<div class="bg-green-500/10 border border-green-500/20 text-green-700 dark:text-green-400 rounded-2xl p-4 flex items-center gap-3" in:fly={{ y: 20, duration: 300 }}>
				<p class="text-sm font-medium">{pesanSukses}</p>
			</div>
		{/if}

		{#if pesanError}
			<div class="bg-red-500/10 border border-red-500/20 text-red-600 dark:text-red-400 rounded-2xl p-4 flex items-center gap-3" in:fly={{ y: 20, duration: 300 }}>
				<p class="text-sm font-medium">{pesanError}</p>
			</div>
		{/if}

		<div class="grid lg:grid-cols-3 gap-6" in:fly={{ y: 20, duration: 600 }}>
			<div class="lg:col-span-2 rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-6">
				<div class="grid grid-cols-2 gap-6">
					<div>
						<div class="flex items-center gap-2">
							<span class="text-xs font-medium text-neutral-500 dark:text-neutral-400 uppercase tracking-wider">Level</span>
							<button onclick={openLevelModal} class="inline-flex items-center gap-1 rounded-md px-1.5 py-0.5 text-xs font-semibold text-brand-600 hover:bg-brand-400/10 dark:text-brand-400">
								<Pencil class="w-3 h-3" /> Ganti
							</button>
						</div>
						<p class="text-lg font-semibold text-neutral-900 dark:text-white mt-1">{levelKelasNama}</p>
					</div>
					<div>
						<span class="text-xs font-medium text-neutral-500 dark:text-neutral-400 uppercase tracking-wider">Frekuensi</span>
						<p class="text-lg font-semibold text-neutral-900 dark:text-white mt-1">{kelas.frekuensi}</p>
					</div>
					<div>
						<div class="flex items-center gap-2">
							<span class="text-xs font-medium text-neutral-500 dark:text-neutral-400 uppercase tracking-wider">Jadwal</span>
							<button onclick={openJadwalModal} class="inline-flex items-center gap-1 rounded-md px-1.5 py-0.5 text-xs font-semibold text-brand-600 hover:bg-brand-400/10 dark:text-brand-400">
								<Pencil class="w-3 h-3" /> Ganti
							</button>
						</div>
						<p class="text-lg font-semibold text-neutral-900 dark:text-white mt-1">{kelas.jadwal}</p>
					</div>
					<div>
						<span class="text-xs font-medium text-neutral-500 dark:text-neutral-400 uppercase tracking-wider">Jenis Kelamin</span>
						<p class="mt-1"><GenderBadge gender={kelas.jenis_kelamin} /></p>
					</div>
				</div>

				<div class="mt-6 pt-6 border-t border-neutral-200/80 dark:border-white/[0.04]">
					<div class="rounded-xl border border-brand-400/20 bg-brand-400/5 p-4">
						<div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
							<div>
								<h3 class="text-sm font-semibold text-neutral-900 dark:text-white">Progres materi per mahasantri</h3>
								<p class="mt-1 text-xs text-neutral-600 dark:text-neutral-400">Gunakan untuk kelas seperti Talaqqi ketika batas materi setiap mahasantri berbeda.</p>
							</div>
							<button onclick={toggleMateriIndividual} disabled={isMateriIndividualLoading} class="inline-flex items-center justify-center rounded-xl px-3.5 py-2 text-sm font-semibold transition-colors disabled:cursor-wait disabled:opacity-50 {kelas.materi_individual ? 'bg-brand-600 text-white hover:bg-brand-700 dark:bg-brand-500 dark:hover:bg-brand-400' : 'bg-neutral-100 text-neutral-700 hover:bg-neutral-200 dark:bg-neutral-800 dark:text-neutral-300 dark:hover:bg-neutral-700'}">
								{kelas.materi_individual ? "Aktif" : "Nonaktif"}
							</button>
						</div>
					</div>
				</div>

				<div class="mt-6 pt-6 border-t border-neutral-200/80 dark:border-white/[0.04]">
					<div class="rounded-xl border border-brand-400/20 bg-brand-400/5 p-4">
						<div class="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
							<div>
								<h3 class="text-sm font-semibold text-neutral-900 dark:text-white">Pertemuan terakhir sebelum sistem</h3>
								<p class="mt-1 text-xs text-neutral-600 dark:text-neutral-400">Isi nomor pertemuan riil kelas lama. Pertemuan berikutnya akan menjadi {pertemuanTerakhir + 1}, dan periode tagihan tetap mengikuti nomor riil.</p>
							</div>
							<div class="flex items-center gap-2">
								<input type="number" min="0" bind:value={pertemuanTerakhir} disabled={has_pertemuan} aria-label="Pertemuan terakhir sebelum sistem" class="w-24 rounded-xl border border-neutral-300 bg-white px-3 py-2 text-sm font-mono text-neutral-900 outline-none focus:border-brand-400 disabled:cursor-not-allowed disabled:opacity-50 dark:border-neutral-700 dark:bg-neutral-800 dark:text-white" />
								<button onclick={simpanPertemuanTerakhir} disabled={has_pertemuan || isPertemuanLoading || pertemuanTerakhir < 0} class="rounded-xl bg-brand-600 px-3.5 py-2 text-sm font-semibold text-white hover:bg-brand-700 disabled:opacity-50 dark:bg-brand-500 dark:hover:bg-brand-400">Simpan</button>
							</div>
						</div>
						<p class="mt-3 text-xs text-amber-700 dark:text-amber-400">{has_pertemuan ? "Pengaturan terkunci karena kelas sudah memiliki pertemuan di aplikasi." : "Pengaturan ini akan terkunci setelah pertemuan pertama dicatat di aplikasi."}</p>
					</div>
				</div>

				<div class="mt-6 pt-6 border-t border-neutral-200/80 dark:border-white/[0.04]">
					<h3 class="text-sm font-semibold text-neutral-900 dark:text-white">Pertemuan Terakhir</h3>
					{#if kelas.tanggal_pertemuan_terakhir}
						<div class="mt-3 grid gap-3 sm:grid-cols-2">
							<div class="rounded-xl bg-neutral-50 p-3 dark:bg-neutral-900/50">
								<p class="text-xs font-medium uppercase tracking-wider text-neutral-500">Pertemuan</p>
								<p class="mt-1 text-sm font-semibold text-neutral-900 dark:text-white">Ke-{kelas.pertemuan_terakhir} · {kelas.tanggal_pertemuan_terakhir}</p>
								<p class="mt-0.5 text-xs text-neutral-500">Nomor dihitung sejak awal level {levelKelasNama}</p>
							</div>
							<div class="rounded-xl bg-neutral-50 p-3 dark:bg-neutral-900/50">
								<p class="text-xs font-medium uppercase tracking-wider text-neutral-500">Materi</p>
								<p class="mt-1 text-sm font-semibold text-neutral-900 dark:text-white">{kelas.materi_terakhir || "Belum diisi"}</p>
							</div>
						</div>
					{:else}
						<p class="mt-2 text-sm text-neutral-500 dark:text-neutral-400">Belum ada pertemuan selesai yang tercatat di aplikasi.</p>
					{/if}
				</div>

				<div class="mt-6 pt-6 border-t border-neutral-200/80 dark:border-white/[0.04]">
					<div class="flex items-center justify-between mb-2">
						<span class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Kapasitas</span>
						<span class="text-sm font-mono text-neutral-700 dark:text-neutral-300">
							{kelas.jumlah_santri} / {kelas.kapasitas}
							<span class="text-neutral-500 ml-1">({capacityPercent}%)</span>
						</span>
					</div>
					<div class="h-2 rounded-full bg-neutral-200/80 dark:bg-neutral-800 overflow-hidden">
						<div
							class="h-full rounded-full transition-all duration-500 {capacityColor(capacityPercent)}"
							style="width: {Math.min(capacityPercent, 100)}%"
						></div>
					</div>
				</div>
			</div>

			<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-6">
				<div class="flex items-center gap-3 mb-4">
					<div class="w-10 h-10 rounded-xl bg-brand-400/10 flex items-center justify-center">
						<GraduationCap class="w-5 h-5 text-brand-600 dark:text-brand-400" />
					</div>
					<div>
						<h3 class="text-sm font-semibold text-neutral-900 dark:text-white">Guru</h3>
						<p class="text-xs text-neutral-500 dark:text-neutral-400">Pengajar kelas ini</p>
					</div>
				</div>

				{#if kelas.guru_nama}
					<div class="flex items-center gap-3 p-3 rounded-xl bg-brand-400/5 border border-brand-400/15 mb-4">
						<div class="w-9 h-9 rounded-full bg-brand-600 dark:bg-brand-500 flex items-center justify-center text-white font-bold text-sm shrink-0">
							{kelas.guru_nama.charAt(0).toUpperCase()}
						</div>
						<div class="min-w-0">
							<p class="text-sm font-semibold text-neutral-900 dark:text-white truncate">{kelas.guru_nama}</p>
							<p class="text-xs text-neutral-500">Guru</p>
						</div>
					</div>
				{:else}
					<div class="flex items-center gap-3 p-3 rounded-xl bg-amber-500/5 border border-amber-500/15 mb-4">
						<div class="w-9 h-9 rounded-full bg-amber-500/20 flex items-center justify-center text-amber-600 dark:text-amber-400 font-bold text-sm shrink-0">
							?
						</div>
						<p class="text-sm text-amber-700 dark:text-amber-400 font-medium">Belum ada guru</p>
					</div>
				{/if}

				<div class="relative">
					<button
						onclick={() => (showGuruDropdown = !showGuruDropdown)}
						class="w-full flex items-center justify-center gap-2 px-4 py-2.5 rounded-xl bg-brand-600 hover:bg-brand-700 text-white font-semibold transition-all dark:bg-brand-500 dark:hover:bg-brand-400 shadow-lg shadow-brand-600/25 text-sm"
					>
						<UserCheck class="w-4 h-4" />
						Assign Guru
						<ChevronDown class="w-4 h-4" />
					</button>

					{#if showGuruDropdown}
						<div class="absolute top-full left-0 right-0 mt-2 bg-white dark:bg-neutral-925 rounded-xl shadow-xl border border-neutral-200/80 dark:border-white/[0.06] overflow-hidden z-10 ring-1 ring-black/10 dark:ring-white/10">
							<div class="max-h-48 overflow-y-auto p-2 space-y-1">
								{#each gurus as guru}
									<button
										onclick={() => { selectedGuruId = guru.id; showGuruDropdown = false; handleAssignGuru(); }}
										class="w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm text-left hover:bg-brand-400/10 transition-colors text-neutral-700 dark:text-neutral-300 hover:text-neutral-900 dark:hover:text-white"
									>
										<div class="w-7 h-7 rounded-full bg-neutral-200 dark:bg-neutral-800 flex items-center justify-center text-xs font-bold text-neutral-600 dark:text-neutral-400 shrink-0">
											{guru.nama.charAt(0).toUpperCase()}
										</div>
										<span>{guru.nama}</span>
									</button>
								{/each}
							</div>
							{#if gurus.length === 0}
								<div class="p-4 text-center text-sm text-neutral-500">Tidak ada guru tersedia</div>
							{/if}
						</div>
					{/if}
				</div>
			</div>
		</div>

		<div class="grid grid-cols-2 sm:grid-cols-4 gap-4" in:fly={{ y: 20, duration: 600, delay: 100 }}>
			<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-5">
				<p class="text-sm text-neutral-500 dark:text-neutral-400 mb-1">Aktif</p>
				<p class="text-3xl font-bold text-green-600 dark:text-green-400 font-mono">{santriAktif.length}</p>
			</div>
			<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-5">
				<p class="text-sm text-neutral-500 dark:text-neutral-400 mb-1">Cuti</p>
				<p class="text-3xl font-bold text-amber-600 dark:text-amber-400 font-mono">{santriCuti.length}</p>
			</div>
			<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-5">
				<p class="text-sm text-neutral-500 dark:text-neutral-400 mb-1">Nonaktif</p>
				<p class="text-3xl font-bold text-neutral-600 dark:text-neutral-300 font-mono">{santriNonaktif.length}</p>
			</div>
			<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-5">
				<p class="text-sm text-neutral-500 dark:text-neutral-400 mb-1">Tidak Lanjut</p>
				<p class="text-3xl font-bold text-red-600 dark:text-red-400 font-mono">{santriTidakLanjut.length}</p>
			</div>
		</div>

		<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 overflow-hidden" in:fly={{ y: 20, duration: 600, delay: 150 }}>
			<div class="flex items-center justify-between px-6 py-4 border-b border-neutral-200/80 dark:border-white/[0.04]">
				<div class="flex items-center gap-2.5">
					<Users class="w-5 h-5 text-neutral-500" />
					<h3 class="text-base font-semibold text-neutral-900 dark:text-white">Daftar Santri</h3>
					<span class="text-sm font-normal text-neutral-500">({santriAktif.length} aktif{santriCuti.length > 0 ? `, ${santriCuti.length} cuti` : ""})</span>
				</div>
				{#if selectedSantri.length > 0}
					<button onclick={() => openLevelSantriModal(selectedSantri)} class="inline-flex items-center gap-1.5 rounded-xl bg-brand-600 px-3.5 py-2 text-sm font-semibold text-white hover:bg-brand-700 dark:bg-brand-500 dark:hover:bg-brand-400">
						<ArrowUpCircle class="w-4 h-4" />
						Ganti level ({selectedSantri.length} santri)
					</button>
				{/if}
			</div>

			{#if santriRoster.length > 0}
				<div class="overflow-x-auto">
					<table class="w-full">
						<thead>
							<tr class="text-xs font-medium text-neutral-500 dark:text-neutral-400 uppercase tracking-wider bg-neutral-50 dark:bg-neutral-900/50">
								<th class="pl-6 py-3 w-10">
									<input type="checkbox" checked={santriRoster.length > 0 && selectedSantri.length === santriRoster.length} onchange={toggleSemuaSantri} aria-label="Pilih semua santri" class="h-4 w-4 rounded border-neutral-300 text-brand-600 focus:ring-brand-400 dark:border-neutral-600" />
								</th>
								<th class="text-left px-6 py-3 w-12">No</th>
								<th class="text-left px-6 py-3">ID Mahasantri</th>
								<th class="text-left px-6 py-3">Nama</th>
								<th class="text-left px-6 py-3">Status</th>
								<th class="text-center px-6 py-3">Hadir</th>
								<th class="text-center px-6 py-3">I/S/A/T</th>
								<th class="text-center px-6 py-3">%</th>
								<th class="text-right px-6 py-3">Aksi</th>
							</tr>
						</thead>
						<tbody class="divide-y divide-neutral-200/80 dark:divide-white/[0.04]">
							{#each santriRoster as s, i}
								<tr class="hover:bg-neutral-50/50 dark:hover:bg-white/[0.015] transition-colors">
									<td class="pl-6 py-3.5">
										<input type="checkbox" checked={selectedSantri.includes(s.id)} onchange={() => toggleSantri(s.id)} aria-label={`Pilih ${s.nama}`} class="h-4 w-4 rounded border-neutral-300 text-brand-600 focus:ring-brand-400 dark:border-neutral-600" />
									</td>
									<td class="px-6 py-3.5 text-sm text-neutral-500 dark:text-neutral-400 font-mono">{i + 1}</td>
									<td class="px-6 py-3.5 text-sm font-mono text-neutral-700 dark:text-neutral-300">{s.id_mahasantri}</td>
									<td class="px-6 py-3.5">
										<a href={"/app/santri/" + s.id} use:inertia class="text-sm font-medium text-neutral-900 dark:text-white hover:text-brand-600 dark:hover:text-brand-400 transition-colors">
											{s.nama}
										</a>
									</td>
									<td class="px-6 py-3.5">
										<StatusBadge status={s.status} />
										{#if s.status === "cuti" && s.cuti_selesai}
											<p class="mt-1 text-xs text-neutral-500 dark:text-neutral-400 whitespace-nowrap" title={s.status_alasan}>s/d {formatTanggal(s.cuti_selesai)}</p>
										{:else if s.status === "aktif" && s.cuti_mulai}
											<p class="mt-1 text-xs text-amber-600 dark:text-amber-400 whitespace-nowrap" title={s.status_alasan}>Cuti mulai {formatTanggal(s.cuti_mulai)}</p>
										{/if}
									</td>
									<td class="px-6 py-3.5 text-center text-sm font-mono text-green-600 dark:text-green-400">{s.total_hadir}</td>
									<td class="px-6 py-3.5 text-center text-xs font-mono text-neutral-500 dark:text-neutral-400">{s.total_izin}/{s.total_sakit}/{s.total_alpa}/{s.total_telat}</td>
									<td class="px-6 py-3.5 text-center text-sm font-mono text-neutral-700 dark:text-neutral-300">{s.persen_hadir.toFixed(0)}%</td>
									<td class="px-6 py-3.5 text-right whitespace-nowrap">
										<button
											onclick={() => openStatusModal(s)}
											class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium text-neutral-600 dark:text-neutral-400 hover:bg-brand-400/10 hover:text-brand-600 dark:hover:text-brand-400 transition-colors"
										>
											<UserCog class="w-3.5 h-3.5" />
											Ubah Status
										</button>
										<button
											onclick={() => openLevelSantriModal([s.id])}
											class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium text-neutral-600 dark:text-neutral-400 hover:bg-brand-400/10 hover:text-brand-600 dark:hover:text-brand-400 transition-colors"
										>
											<ArrowUpCircle class="w-3.5 h-3.5" />
											Ganti Level
										</button>
										<button
											onclick={() => openPindahModal(s)}
											class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium text-neutral-600 dark:text-neutral-400 hover:bg-brand-400/10 hover:text-brand-600 dark:hover:text-brand-400 transition-colors"
										>
											<MoveRight class="w-3.5 h-3.5" />
											Pindahkan
										</button>
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{:else}
				<div class="p-12 text-center">
					<div class="w-12 h-12 rounded-xl bg-neutral-100 dark:bg-neutral-800 flex items-center justify-center mx-auto mb-3">
						<Users class="w-6 h-6 text-neutral-500" />
					</div>
					<p class="text-sm text-neutral-500 dark:text-neutral-400">Belum ada santri aktif di kelas ini</p>
				</div>
			{/if}
		</div>

		{#if santriKeluar.length > 0}
			<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 overflow-hidden" in:fly={{ y: 20, duration: 600, delay: 200 }}>
				<div class="flex items-center gap-2.5 px-6 py-4 border-b border-neutral-200/80 dark:border-white/[0.04]">
					<UserX class="w-5 h-5 text-red-500" />
					<h3 class="text-base font-semibold text-neutral-900 dark:text-white">Nonaktif & Tidak Lanjut</h3>
					<span class="text-sm font-normal text-neutral-500">({santriKeluar.length} santri · tidak mengisi kursi)</span>
				</div>
				<div class="overflow-x-auto">
					<table class="w-full">
						<thead>
							<tr class="text-xs font-medium text-neutral-500 dark:text-neutral-400 uppercase tracking-wider bg-neutral-50 dark:bg-neutral-900/50">
								<th class="text-left px-6 py-3 w-12">No</th>
								<th class="text-left px-6 py-3">ID Mahasantri</th>
								<th class="text-left px-6 py-3">Nama</th>
								<th class="text-left px-6 py-3">Status</th>
								<th class="text-left px-6 py-3">Alasan</th>
								<th class="text-right px-6 py-3">Aksi</th>
							</tr>
						</thead>
						<tbody class="divide-y divide-neutral-200/80 dark:divide-white/[0.04]">
							{#each santriKeluar as s, i}
								<tr class="opacity-70 hover:opacity-100 hover:bg-neutral-50/50 dark:hover:bg-white/[0.015] transition-all">
									<td class="px-6 py-3.5 text-sm text-neutral-500 dark:text-neutral-400 font-mono">{i + 1}</td>
									<td class="px-6 py-3.5 text-sm font-mono text-neutral-700 dark:text-neutral-300">{s.id_mahasantri}</td>
									<td class="px-6 py-3.5">
										<a href={"/app/santri/" + s.id} use:inertia class="text-sm font-medium text-neutral-700 dark:text-neutral-300 hover:text-brand-600 dark:hover:text-brand-400 transition-colors line-through decoration-neutral-400/60">
											{s.nama}
										</a>
									</td>
									<td class="px-6 py-3.5">
										<StatusBadge status={s.status} />
									</td>
									<td class="px-6 py-3.5 text-sm text-neutral-600 dark:text-neutral-400">{s.status_alasan || "-"}</td>
									<td class="px-6 py-3.5 text-right whitespace-nowrap">
										{#if s.status === "nonaktif"}
											<button
												onclick={() => openStatusModal(s)}
												class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium text-neutral-600 dark:text-neutral-400 hover:bg-brand-400/10 hover:text-brand-600 dark:hover:text-brand-400 transition-colors"
											>
												<UserCog class="w-3.5 h-3.5" />
												Ubah Status
											</button>
										{:else}
											<span class="text-xs text-neutral-400" title="Status Tidak Lanjut diatur oleh Keuangan">Diatur Keuangan</span>
										{/if}
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			</div>
		{/if}

		<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 overflow-hidden" in:fly={{ y: 20, duration: 600, delay: 225 }}>
			<div class="flex items-center gap-2.5 px-6 py-4 border-b border-neutral-200/80 dark:border-white/[0.04]">
				<UserCog class="w-5 h-5 text-neutral-500" />
				<h3 class="text-base font-semibold text-neutral-900 dark:text-white">Riwayat Status Santri</h3>
			</div>
			{#if riwayat_status.length > 0}
				<ul class="divide-y divide-neutral-200/80 dark:divide-white/[0.04] max-h-96 overflow-y-auto">
					{#each riwayat_status as log (log.id)}
						<li class="px-6 py-4">
							<p class="text-sm font-semibold text-neutral-900 dark:text-white">{log.santri_nama}</p>
							<p class="mt-0.5 flex flex-wrap items-center gap-1.5 text-sm">
								<StatusBadge status={log.status_lama} />
								<MoveRight class="w-3.5 h-3.5 text-neutral-400" />
								<StatusBadge status={log.status_baru} />
								{#if (log.status_baru === "cuti" || log.status_baru === "cuti_terjadwal") && log.cuti_mulai}
									<span class="text-xs text-neutral-500 dark:text-neutral-400">{formatTanggal(log.cuti_mulai)} – {formatTanggal(log.cuti_selesai)}</span>
								{/if}
							</p>
							{#if log.alasan}
								<p class="mt-1 text-sm text-neutral-600 dark:text-neutral-400">{log.alasan}</p>
							{/if}
							<p class="mt-1 text-xs text-neutral-400">{log.created_at} · oleh {log.dibuat_oleh_nama || "Sistem"}</p>
						</li>
					{/each}
				</ul>
			{:else}
				<p class="px-6 py-8 text-center text-sm text-neutral-500 dark:text-neutral-400">Belum ada perubahan status santri di kelas ini.</p>
			{/if}
		</div>

		<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 overflow-hidden" in:fly={{ y: 20, duration: 600, delay: 250 }}>
			<div class="flex items-center gap-2.5 px-6 py-4 border-b border-neutral-200/80 dark:border-white/[0.04]">
				<History class="w-5 h-5 text-neutral-500" />
				<h3 class="text-base font-semibold text-neutral-900 dark:text-white">Riwayat Perubahan Level & Jadwal</h3>
			</div>
			{#if riwayat_perubahan.length > 0}
				<ul class="divide-y divide-neutral-200/80 dark:divide-white/[0.04]">
					{#each riwayat_perubahan as item (item.id)}
						<li class="flex gap-3 px-6 py-4">
							<div class="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg {item.jenis === 'jadwal_kelas' ? 'bg-blue-500/10 text-blue-600 dark:text-blue-400' : 'bg-brand-400/10 text-brand-600 dark:text-brand-400'}">
								{#if item.jenis === "jadwal_kelas"}<CalendarClock class="w-4 h-4" />{:else}<Layers class="w-4 h-4" />{/if}
							</div>
							<div class="min-w-0 flex-1">
								<p class="text-sm font-semibold text-neutral-900 dark:text-white">
									{labelPerubahan(item)}{item.santri_nama ? ` · ${item.santri_nama}` : ""}
								</p>
								<p class="mt-0.5 text-sm text-neutral-600 dark:text-neutral-400">
									{item.nilai_lama} <MoveRight class="inline w-3.5 h-3.5 mx-0.5" /> <span class="font-medium text-neutral-800 dark:text-neutral-200">{item.nilai_baru}</span>
								</p>
								{#if item.jenis === "level_kelas" && item.pertemuan_ke > 0}
									<p class="mt-0.5 text-xs text-neutral-500">Level lama selesai setelah {item.pertemuan_ke} pertemuan tercatat</p>
								{/if}
								{#if item.jenis === "level_santri"}
									<p class="mt-0.5 text-xs text-neutral-500">
										{#if item.kelas_tujuan_id && item.kelas_tujuan_id !== kelas.id}
											Ke kelas <a href={"/app/kelas/" + item.kelas_tujuan_id} use:inertia class="font-medium text-brand-600 hover:underline dark:text-brand-400">{item.kelas_tujuan_nama}</a>
										{:else if item.kelas_asal_id && item.kelas_asal_id !== kelas.id}
											Dari kelas <a href={"/app/kelas/" + item.kelas_asal_id} use:inertia class="font-medium text-brand-600 hover:underline dark:text-brand-400">{item.kelas_asal_nama}</a>
										{/if}
									</p>
								{/if}
								<p class="mt-1 text-xs text-neutral-400">{item.created_at} · oleh {item.dibuat_oleh_nama}</p>
							</div>
						</li>
					{/each}
				</ul>
			{:else}
				<p class="px-6 py-8 text-center text-sm text-neutral-500 dark:text-neutral-400">Belum ada perubahan level atau jadwal untuk kelas ini.</p>
			{/if}
		</div>
	</div>
</AppLayout>

{#if statusModal.open && statusModal.santri}
	{@const current = statusModal.santri}
	<div class="fixed inset-0 z-50 flex items-center justify-center p-4">
		<button class="absolute inset-0 w-full h-full bg-neutral-900/50 backdrop-blur-sm" aria-label="Tutup modal" onclick={closeStatusModal}></button>
		<div class="relative w-full max-w-md max-h-[90vh] overflow-y-auto bg-white dark:bg-neutral-925 rounded-2xl shadow-xl border border-neutral-200/80 dark:border-white/[0.06] p-6" in:fly={{ y: 20, duration: 200 }}>
			<h3 class="text-lg font-bold text-neutral-900 dark:text-white mb-1">Ubah Status Santri</h3>
			<p class="text-sm text-neutral-600 dark:text-neutral-400 mb-4">
				<strong>{current.nama}</strong> · saat ini <StatusBadge status={current.status} />
			</p>

			<fieldset class="mb-4">
				<legend class="text-xs font-semibold uppercase tracking-wider text-neutral-500 mb-2">Status baru</legend>
				<div class="grid grid-cols-3 gap-2">
					{#each ["aktif", "cuti", "nonaktif"] as opsi}
						{@const disabled = opsi === current.status && opsi !== "cuti" && !(opsi === "aktif" && current.cuti_mulai)}
						<label class="flex cursor-pointer items-center justify-center rounded-xl border px-3 py-2.5 text-sm font-medium transition-colors {statusBaru === opsi ? 'border-brand-500 bg-brand-400/10 text-brand-700 dark:text-brand-300' : 'border-neutral-300 dark:border-neutral-700/80 text-neutral-700 dark:text-neutral-300 hover:bg-neutral-100 dark:hover:bg-neutral-800'} {disabled ? 'opacity-40 cursor-not-allowed' : ''}">
							<input type="radio" class="sr-only" name="status_baru" value={opsi} bind:group={statusBaru} {disabled} />
							{labelStatus[opsi]}
						</label>
					{/each}
				</div>
				<p class="mt-2 text-xs text-neutral-500 dark:text-neutral-400">
					{#if statusBaru === "cuti" && cutiMulai > hariIni()}
						Cuti terjadwal: santri tetap Aktif (diabsen dan ditagih) sampai tanggal mulai, lalu otomatis menjadi Cuti dan kembali Aktif setelah tanggal selesai.
					{:else if statusBaru === "cuti"}
						Cuti tetap memegang kursi kelas, tidak diabsen, dan tidak ditagih infaq. Status kembali Aktif otomatis setelah tanggal selesai.
					{:else if statusBaru === "aktif" && current.status === "aktif" && current.cuti_mulai}
						Membatalkan cuti terjadwal {formatTanggal(current.cuti_mulai)} – {formatTanggal(current.cuti_selesai)}.
					{:else if statusBaru === "nonaktif"}
						Nonaktif melepas kursi kelas, tidak diabsen, dan tidak ditagih infaq.
					{:else}
						Santri kembali diabsen dan ditagih seperti biasa.
					{/if}
				</p>
			</fieldset>

			{#if statusBaru === "cuti"}
				<div class="grid grid-cols-2 gap-3 mb-4">
					<label class="block">
						<span class="text-xs font-semibold uppercase tracking-wider text-neutral-500">Mulai cuti</span>
						<input type="date" bind:value={cutiMulai} class="mt-1 w-full px-3 py-2.5 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white outline-none text-sm" />
					</label>
					<label class="block">
						<span class="text-xs font-semibold uppercase tracking-wider text-neutral-500">Sampai (hari terakhir)</span>
						<input type="date" bind:value={cutiSelesai} min={cutiMulai > hariIni() ? cutiMulai : hariIni()} class="mt-1 w-full px-3 py-2.5 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white outline-none text-sm" />
					</label>
				</div>
			{/if}

			{#if statusBaru !== "aktif"}
				<label class="block mb-4">
					<span class="text-xs font-semibold uppercase tracking-wider text-neutral-500">Alasan <span class="text-red-500">*</span></span>
					<textarea bind:value={statusAlasan} rows="3" placeholder={statusBaru === "cuti" ? "Contoh: sakit, safar, melahirkan" : "Contoh: mengundurkan diri, pindah domisili"} class="mt-1 w-full px-3 py-2.5 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white outline-none text-sm"></textarea>
				</label>
			{/if}

			<div class="flex items-center gap-3">
				<button onclick={closeStatusModal} class="flex-1 px-4 py-2.5 rounded-xl border border-neutral-300 dark:border-neutral-700/80 text-sm font-medium text-neutral-700 dark:text-neutral-300 hover:bg-neutral-100 dark:hover:bg-neutral-800 transition-colors">
					Batal
				</button>
				<button
					onclick={submitUbahStatus}
					disabled={!statusFormValid || isUbahStatusLoading}
					class="flex-1 px-4 py-2.5 rounded-xl bg-brand-600 hover:bg-brand-700 text-white font-semibold transition-all dark:bg-brand-500 dark:hover:bg-brand-400 disabled:opacity-50 disabled:cursor-not-allowed text-sm"
				>
					{isUbahStatusLoading ? "Menyimpan..." : "Simpan Status"}
				</button>
			</div>
		</div>
	</div>
{/if}

{#if pindahModal.open}
	<div class="fixed inset-0 z-50 flex items-center justify-center p-4">
		<button class="absolute inset-0 w-full h-full bg-neutral-900/50 backdrop-blur-sm" aria-label="Tutup modal" onclick={() => (pindahModal = { open: false, santriId: 0, santriNama: "" })}></button>
		<div class="relative w-full max-w-md bg-white dark:bg-neutral-925 rounded-2xl shadow-xl border border-neutral-200/80 dark:border-white/[0.06] p-6" in:fly={{ y: 20, duration: 200 }}>
			<h3 class="text-lg font-bold text-neutral-900 dark:text-white mb-2">Pindahkan Santri</h3>
			<p class="text-sm text-neutral-600 dark:text-neutral-400 mb-4">Pindahkan <strong>{pindahModal.santriNama}</strong> ke kelas lain</p>

			<select
				bind:value={pindahKelasId}
				class="w-full px-4 py-3 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white outline-none text-sm mb-4"
			>
				<option value={null} disabled>Pilih kelas tujuan</option>
				{#each kelas_lain as kl}
					<option value={kl.id}>Angkatan {kl.angkatan} · {kl.nama_kelas} - {namaLevel(kl.level)} ({kl.jadwal})</option>
				{/each}
			</select>

			<div class="flex items-center gap-3">
				<button
					onclick={() => (pindahModal = { open: false, santriId: 0, santriNama: "" })}
					class="flex-1 px-4 py-2.5 rounded-xl border border-neutral-300 dark:border-neutral-700/80 text-sm font-medium text-neutral-700 dark:text-neutral-300 hover:bg-neutral-100 dark:hover:bg-neutral-800 transition-colors"
				>
					Batal
				</button>
				<button
					onclick={handlePindahkanSantri}
					disabled={!pindahKelasId || isPindahLoading}
					class="flex-1 px-4 py-2.5 rounded-xl bg-brand-600 hover:bg-brand-700 text-white font-semibold transition-all dark:bg-brand-500 dark:hover:bg-brand-400 disabled:opacity-50 disabled:cursor-not-allowed text-sm flex items-center justify-center gap-2"
				>
					{#if isPindahLoading}
						<svg class="animate-spin h-4 w-4" fill="none" viewBox="0 0 24 24">
							<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
							<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
						</svg>
						Memindahkan...
					{:else}
						Pindahkan
					{/if}
				</button>
			</div>
		</div>
	</div>
{/if}

{#if levelModalOpen}
	<div class="fixed inset-0 z-50 flex items-center justify-center p-4">
		<button class="absolute inset-0 w-full h-full bg-neutral-900/50 backdrop-blur-sm" aria-label="Tutup modal" onclick={() => (levelModalOpen = false)}></button>
		<div class="relative w-full max-w-md bg-white dark:bg-neutral-925 rounded-2xl shadow-xl border border-neutral-200/80 dark:border-white/[0.06] p-6" in:fly={{ y: 20, duration: 200 }}>
			<h3 class="text-lg font-bold text-neutral-900 dark:text-white mb-1">Ganti Level Kelas</h3>
			<p class="text-sm text-neutral-600 dark:text-neutral-400 mb-4">Seluruh kelas (guru dan {santriRoster.length} santri) pindah dari level <strong>{levelKelasNama}</strong> ke level baru.</p>
			<label for="level-baru" class="block text-xs font-medium uppercase tracking-wider text-neutral-500 mb-1.5">Level baru</label>
			<select id="level-baru" bind:value={levelBaru} class="w-full px-4 py-3 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white outline-none text-sm">
				<option value="" disabled>Pilih level</option>
				{#each sortedLevels as lv}
					<option value={lv.kode} disabled={lv.kode === kelas.level}>{lv.nama}{lv.kode === kelas.level ? " (saat ini)" : ""}</option>
				{/each}
			</select>
			<ul class="mt-4 space-y-1.5 rounded-xl bg-brand-400/5 border border-brand-400/15 p-3 text-xs text-neutral-600 dark:text-neutral-400">
				<li>• Pertemuan berikutnya dimulai lagi dari <strong>ke-1</strong> di level baru.</li>
				<li>• Riwayat absensi dan materi level lama tetap tersimpan.</li>
				<li>• Periode tagihan SPP tidak berubah.</li>
				<li>• Guru kelas mendapat notifikasi.</li>
			</ul>
			<div class="mt-5 flex items-center gap-3">
				<button onclick={() => (levelModalOpen = false)} class="flex-1 px-4 py-2.5 rounded-xl border border-neutral-300 dark:border-neutral-700/80 text-sm font-medium text-neutral-700 dark:text-neutral-300 hover:bg-neutral-100 dark:hover:bg-neutral-800 transition-colors">Batal</button>
				<button onclick={submitGantiLevel} disabled={!levelBaru || levelBaru === kelas.level || isLevelLoading} class="flex-1 px-4 py-2.5 rounded-xl bg-brand-600 hover:bg-brand-700 text-white font-semibold transition-all dark:bg-brand-500 dark:hover:bg-brand-400 disabled:opacity-50 disabled:cursor-not-allowed text-sm">
					{isLevelLoading ? "Menyimpan..." : "Ganti Level"}
				</button>
			</div>
		</div>
	</div>
{/if}

{#if jadwalModalOpen}
	<div class="fixed inset-0 z-50 flex items-center justify-center p-4">
		<button class="absolute inset-0 w-full h-full bg-neutral-900/50 backdrop-blur-sm" aria-label="Tutup modal" onclick={() => (jadwalModalOpen = false)}></button>
		<div class="relative w-full max-w-md bg-white dark:bg-neutral-925 rounded-2xl shadow-xl border border-neutral-200/80 dark:border-white/[0.06] p-6" in:fly={{ y: 20, duration: 200 }}>
			<h3 class="text-lg font-bold text-neutral-900 dark:text-white mb-1">Ganti Jadwal Kelas</h3>
			<p class="text-sm text-neutral-600 dark:text-neutral-400 mb-4">Jadwal rutin saat ini: <strong>{kelas.jadwal || "-"}</strong>. Perubahan berlaku permanen untuk pertemuan berikutnya.</p>
			<label for="jadwal-baru" class="block text-xs font-medium uppercase tracking-wider text-neutral-500 mb-1.5">Jadwal baru</label>
			<select id="jadwal-baru" bind:value={jadwalBaru} class="w-full px-4 py-3 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white outline-none text-sm">
				<option value="" disabled>Pilih jadwal dari master</option>
				{#each jadwals as j}
					<option value={j.nama} disabled={j.nama === kelas.jadwal}>{j.nama}{j.nama === kelas.jadwal ? " (saat ini)" : ""}</option>
				{/each}
			</select>
			<p class="mt-2 text-xs text-neutral-500">Jadwal belum ada? Tambahkan dulu di <a href="/app/master" use:inertia class="font-medium text-brand-600 hover:underline dark:text-brand-400">Master Data</a>.</p>
			<p class="mt-3 rounded-xl bg-amber-500/5 border border-amber-500/15 p-3 text-xs text-amber-700 dark:text-amber-400">Sesi yang sudah dijadwalkan tidak diubah otomatis, tetapi diberi tanda "Jadwal kelas berubah" agar bisa dijadwalkan ulang. Guru kelas mendapat notifikasi.</p>
			<div class="mt-5 flex items-center gap-3">
				<button onclick={() => (jadwalModalOpen = false)} class="flex-1 px-4 py-2.5 rounded-xl border border-neutral-300 dark:border-neutral-700/80 text-sm font-medium text-neutral-700 dark:text-neutral-300 hover:bg-neutral-100 dark:hover:bg-neutral-800 transition-colors">Batal</button>
				<button onclick={submitGantiJadwal} disabled={!jadwalBaru || jadwalBaru === kelas.jadwal || isJadwalLoading} class="flex-1 px-4 py-2.5 rounded-xl bg-brand-600 hover:bg-brand-700 text-white font-semibold transition-all dark:bg-brand-500 dark:hover:bg-brand-400 disabled:opacity-50 disabled:cursor-not-allowed text-sm">
					{isJadwalLoading ? "Menyimpan..." : "Ganti Jadwal"}
				</button>
			</div>
		</div>
	</div>
{/if}

{#if levelSantriModal.open}
	<div class="fixed inset-0 z-50 flex items-center justify-center p-4">
		<button class="absolute inset-0 w-full h-full bg-neutral-900/50 backdrop-blur-sm" aria-label="Tutup modal" onclick={() => (levelSantriModal = { open: false, santriIds: [] })}></button>
		<div class="relative w-full max-w-lg bg-white dark:bg-neutral-925 rounded-2xl shadow-xl border border-neutral-200/80 dark:border-white/[0.06] p-6" in:fly={{ y: 20, duration: 200 }}>
			<h3 class="text-lg font-bold text-neutral-900 dark:text-white mb-1">Ganti Level Santri</h3>
			<p class="text-sm text-neutral-600 dark:text-neutral-400 mb-4">Pindahkan <strong>{namaSantriTerpilih}</strong> dari level {levelKelasNama} ke kelas level lain. Santri lain tetap di kelas ini.</p>

			<label for="level-santri" class="block text-xs font-medium uppercase tracking-wider text-neutral-500 mb-1.5">Level tujuan</label>
			<select id="level-santri" value={levelSantriTujuan} onchange={(e) => pilihLevelSantri(e.currentTarget.value)} class="w-full px-4 py-3 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white outline-none text-sm">
				<option value="" disabled>Pilih level</option>
				{#each sortedLevels as lv}
					<option value={lv.kode} disabled={lv.kode === kelas.level}>{lv.nama}{lv.kode === kelas.level ? " (saat ini)" : ""}</option>
				{/each}
			</select>

			{#if levelSantriTujuan}
				<p class="mt-4 mb-2 text-xs font-medium uppercase tracking-wider text-neutral-500">Kelas tujuan</p>
				<div class="max-h-60 space-y-2 overflow-y-auto">
					{#each kelasTujuanLevel as kl (kl.id)}
						{@const sisa = kl.kapasitas - kl.jumlah_santri}
						<label class="flex cursor-pointer items-start gap-3 rounded-xl border p-3 text-sm transition-colors {levelSantriMode === 'kelas' && levelSantriKelasId === kl.id ? 'border-brand-400 bg-brand-400/5' : 'border-neutral-200 dark:border-neutral-700/80'} {sisa < levelSantriModal.santriIds.length ? 'opacity-50' : ''}">
							<input type="radio" name="kelas-tujuan" checked={levelSantriMode === "kelas" && levelSantriKelasId === kl.id} disabled={sisa < levelSantriModal.santriIds.length} onchange={() => { levelSantriMode = "kelas"; levelSantriKelasId = kl.id; }} class="mt-0.5 text-brand-600 focus:ring-brand-400" />
							<span class="min-w-0">
								<span class="block font-medium text-neutral-900 dark:text-white">{kl.nama_kelas}</span>
								<span class="block text-xs text-neutral-500">{kl.jadwal} · {kl.guru_nama || "Belum ada guru"} · {kl.jumlah_santri}/{kl.kapasitas} santri</span>
							</span>
						</label>
					{/each}
					<label class="flex cursor-pointer items-start gap-3 rounded-xl border border-dashed p-3 text-sm transition-colors {levelSantriMode === 'baru' ? 'border-brand-400 bg-brand-400/5' : 'border-neutral-300 dark:border-neutral-700/80'}">
						<input type="radio" name="kelas-tujuan" checked={levelSantriMode === "baru"} onchange={() => { levelSantriMode = "baru"; levelSantriKelasId = null; }} class="mt-0.5 text-brand-600 focus:ring-brand-400" />
						<span class="min-w-0">
							<span class="block font-medium text-neutral-900 dark:text-white">Buat kelas baru level {namaLevel(levelSantriTujuan)}</span>
							<span class="block text-xs text-neutral-500">Jadwal, jenis kelamin, frekuensi, dan angkatan sama dengan kelas ini. Guru di-assign setelahnya.</span>
						</span>
					</label>
				</div>
				{#if kelasTujuanLevel.length === 0}
					<p class="mt-2 text-xs text-neutral-500">Belum ada kelas aktif level {namaLevel(levelSantriTujuan)} untuk jenis kelamin ini.</p>
				{/if}
			{/if}

			<div class="mt-5 flex items-center gap-3">
				<button onclick={() => (levelSantriModal = { open: false, santriIds: [] })} class="flex-1 px-4 py-2.5 rounded-xl border border-neutral-300 dark:border-neutral-700/80 text-sm font-medium text-neutral-700 dark:text-neutral-300 hover:bg-neutral-100 dark:hover:bg-neutral-800 transition-colors">Batal</button>
				<button onclick={submitLevelSantri} disabled={!levelSantriTujuan || (levelSantriMode === "kelas" && !levelSantriKelasId) || isLevelSantriLoading} class="flex-1 px-4 py-2.5 rounded-xl bg-brand-600 hover:bg-brand-700 text-white font-semibold transition-all dark:bg-brand-500 dark:hover:bg-brand-400 disabled:opacity-50 disabled:cursor-not-allowed text-sm">
					{isLevelSantriLoading ? "Memproses..." : "Ganti Level"}
				</button>
			</div>
		</div>
	</div>
{/if}
