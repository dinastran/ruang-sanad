<script lang="ts">
	import { inertia } from "@inertiajs/svelte";
	import { fly } from "svelte/transition";
	import AppLayout from "@layouts/AppLayout.svelte";
	import GenderBadge from "@components/GenderBadge.svelte";
	import type { AbsensiGuru, Flash, User, GuruKelas, PertemuanGuru, SantriGuru, Riayah } from "@lib/types";
	import { nomorPertemuan } from "@lib/pertemuan";
	import { getCSRFToken } from "@lib/utils/csrf";
	import { Toast } from "@lib/notifications/toast";
	import {
		ArrowLeft, BookOpen, Users, Calendar, CalendarClock, Clock, Play, ClipboardList, MessageSquare,
		FileText, User as UserIcon, ChevronDown, ChevronRight, ExternalLink, Copy, Check
	} from "lucide-svelte";

	interface Props {
		user?: User;
		kelas: GuruKelas;
		santri: SantriGuru[];
		active_pertemuan?: PertemuanGuru;
		last_pertemuan?: PertemuanGuru;
		last_absensi?: AbsensiGuru[];
		success?: string;
		error?: string;
		flash?: Flash;
	}

	let { user, kelas, santri = [], active_pertemuan, last_pertemuan, last_absensi = [], success, error, flash }: Props = $props();
	let canEdit = $derived(user?.role === "guru" || user?.role === "admin_kelas" || user?.role === "super_admin");
	let isBroadcastOpen = $state(false);
	let isBroadcastCopied = $state(false);
	const attendanceLabels: Record<string, string> = { hadir: "Hadir", izin: "Izin", sakit: "Sakit", alpa: "Alpa", telat: "Telat" };
	let batasMateriBySantri = $derived(Object.fromEntries(santri.map((item) => [item.id, item.batas_materi_terakhir])));
	let broadcastMessage = $derived(`Assalamu'alaikum warahmatullahi wabarakatuh.\n\n*Laporan Kelas ${kelas.nama_kelas}* 📚\n\n*Pertemuan terakhir:* ${last_pertemuan ? `Ke-${nomorPertemuan(last_pertemuan)}${last_pertemuan.level_nama ? ` level ${last_pertemuan.level_nama}` : ""} (${last_pertemuan.tanggal})` : "Belum ada pertemuan tercatat"}\n*Materi:* ${last_pertemuan?.materi || "Belum diisi"}\n\n*Absensi Pertemuan Terakhir* 📝\n${last_absensi.length > 0 ? last_absensi.map((item, index) => `${index + 1}. *${item.santri_nama}* — ${attendanceLabels[item.status] || item.status}${kelas.materi_individual ? `\n   Batas materi: ${item.batas_materi || batasMateriBySantri[item.santri_id] || "Belum tercatat"}` : ""}${item.catatan.trim() ? `\n   Catatan: ${item.catatan.trim()}` : ""}`).join("\n\n") : "Belum ada absensi pertemuan tercatat."}\n\nJazakumullahu khairan.`);

	function openBroadcast() {
		isBroadcastCopied = false;
		isBroadcastOpen = true;
	}

	async function copyBroadcastMessage() {
		try {
			await navigator.clipboard.writeText(broadcastMessage);
			isBroadcastCopied = true;
		} catch {
			Toast("Gagal menyalin pesan", "error");
		}
	}

	function openWhatsApp() {
		window.open(`https://wa.me/?text=${encodeURIComponent(broadcastMessage)}`, "_blank", "noopener,noreferrer");
	}

	let riayahModal = $state<{ open: boolean; santriId: number; santriNama: string; notes: Riayah[]; catatan: string; loading: boolean }>({
		open: false, santriId: 0, santriNama: "", notes: [], catatan: "", loading: false
	});
	let expandedRiwayat = $state<Record<number, boolean>>({});

	function openRiayahModal(s: SantriGuru) {
		riayahModal = { open: true, santriId: s.id, santriNama: s.nama, notes: [], catatan: "", loading: true };
		fetch("/app/guru/kelas/" + kelas.id + "/santri/" + s.id + "/riayah", {
			headers: { "X-XSRF-TOKEN": getCSRFToken() },
		})
			.then((r) => r.json())
			.then((data) => {
				riayahModal = { ...riayahModal, notes: data.riayah || [], loading: false };
			})
			.catch(() => {
				riayahModal = { ...riayahModal, loading: false };
			});
	}

	function submitRiayah() {
		if (!riayahModal.catatan.trim()) return;
		riayahModal = { ...riayahModal, loading: true };
		fetch("/app/guru/kelas/" + kelas.id + "/santri/" + riayahModal.santriId + "/riayah", {
			method: "POST",
			headers: {
				"Content-Type": "application/json",
				"X-XSRF-TOKEN": getCSRFToken(),
			},
			body: JSON.stringify({ catatan: riayahModal.catatan }),
		})
			.then((r) => r.json())
			.then((data) => {
				riayahModal = { ...riayahModal, catatan: "", notes: data.riayah || [], loading: false };
			})
			.catch(() => {
				riayahModal = { ...riayahModal, loading: false };
			});
	}

	function toggleRiwayat(santriId: number) {
		expandedRiwayat[santriId] = !expandedRiwayat[santriId];
		expandedRiwayat = expandedRiwayat;
	}

	function attendanceColor(pct: number): string {
		if (pct >= 85) return "text-green-600 dark:text-green-400";
		if (pct >= 70) return "text-amber-600 dark:text-amber-400";
		return "text-red-600 dark:text-red-400";
	}

</script>

<AppLayout {user} group="guru-kelas">
	<div class="pt-8 pb-10 border-b border-neutral-200/80 dark:border-white/[0.04]">
		<div class="max-w-6xl mx-auto px-4 sm:px-6">
			<div class="flex items-center gap-2 text-sm text-neutral-500 dark:text-neutral-400 mb-2">
				<a href="/app" use:inertia class="hover:text-brand-600 dark:hover:text-brand-400 transition-colors">Dashboard</a>
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
				</svg>
				<a href="/app/guru/kelas" use:inertia class="hover:text-brand-600 dark:hover:text-brand-400 transition-colors">Kelas Saya</a>
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
				</svg>
				<span class="text-neutral-700 dark:text-neutral-300">{kelas.nama_kelas}</span>
			</div>

			<a href="/app/guru/kelas" use:inertia class="inline-flex items-center gap-1.5 text-sm text-neutral-500 hover:text-brand-600 dark:hover:text-brand-400 transition-colors mb-4">
				<ArrowLeft class="w-4 h-4" />
				Kembali
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
						</div>
						<p class="text-neutral-600 dark:text-neutral-400">
							{kelas.level} &middot; {kelas.tipe} &middot; {kelas.frekuensi} &middot; {kelas.jadwal}
						</p>
					</div>
				</div>
			</div>
		</div>
	</div>

	<div class="relative max-w-6xl mx-auto px-4 sm:px-6 py-8 space-y-6">
		{#if success || flash?.success}
			<div class="bg-green-500/10 border border-green-500/20 text-green-700 dark:text-green-400 rounded-2xl p-4 flex items-center gap-3" in:fly={{ y: 20, duration: 300 }}>
				<p class="text-sm font-medium">{success || flash?.success}</p>
			</div>
		{/if}

		{#if error || flash?.error}
			<div class="bg-red-500/10 border border-red-500/20 text-red-600 dark:text-red-400 rounded-2xl p-4 flex items-center gap-3" in:fly={{ y: 20, duration: 300 }}>
				<p class="text-sm font-medium">{error || flash?.error}</p>
			</div>
		{/if}

		{#if active_pertemuan}
			<div class="rounded-2xl border border-amber-500/25 bg-amber-500/5 p-5" in:fly={{ y: 20, duration: 400 }}>
				<div class="flex flex-wrap items-center justify-between gap-4">
					<div class="flex min-w-0 items-center gap-3">
						<div class="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-amber-500/15 text-amber-700 dark:text-amber-400"><Play class="h-5 w-5" /></div>
						<div>
							<div class="flex flex-wrap items-center gap-2">
								<h2 class="font-semibold text-neutral-900 dark:text-white">Pertemuan ke-{nomorPertemuan(active_pertemuan)}</h2>
								<span class="rounded-full bg-amber-500/15 px-2.5 py-1 text-xs font-semibold text-amber-700 dark:text-amber-400">Sedang berlangsung</span>
							</div>
							<p class="mt-1 text-sm text-neutral-600 dark:text-neutral-400">Dimulai {active_pertemuan.tanggal} pukul {active_pertemuan.jam_mulai}</p>
						</div>
					</div>
					<a href={"/app/guru/kelas/" + kelas.id + "/pertemuan/" + active_pertemuan.id + "/selesai"} use:inertia class="inline-flex items-center gap-2 rounded-xl bg-amber-600 px-4 py-2.5 text-sm font-semibold text-white transition-colors hover:bg-amber-700">
						<Play class="h-4 w-4" /> Lanjutkan
					</a>
				</div>
			</div>
		{/if}

		<div class="flex flex-wrap items-center gap-3" in:fly={{ y: 20, duration: 500 }}>
			{#if canEdit}<a href={"/app/guru/jadwal-pertemuan?kelas_id=" + kelas.id} use:inertia
				class="inline-flex items-center gap-2 px-5 py-2.5 rounded-xl bg-brand-400/10 text-brand-700 dark:text-brand-400 hover:bg-brand-400/20 font-semibold transition-colors text-sm"
			>
				<CalendarClock class="w-4 h-4" />
				Jadwalkan
			</a>{/if}
			{#if canEdit && !active_pertemuan}<a href={"/app/guru/kelas/" + kelas.id + "/pertemuan/mulai"} use:inertia
				class="inline-flex items-center gap-2 px-5 py-2.5 rounded-xl bg-brand-600 hover:bg-brand-700 text-white font-semibold transition-all dark:bg-brand-500 dark:hover:bg-brand-400 shadow-lg shadow-brand-600/25 text-sm"
			>
				<Play class="w-4 h-4" />
				Mulai Sekarang
			</a>{/if}
			<a href={"/app/guru/kelas/" + kelas.id + "/rekap"} use:inertia
				class="inline-flex items-center gap-2 px-5 py-2.5 rounded-xl bg-neutral-100 dark:bg-neutral-800 text-neutral-700 dark:text-neutral-300 hover:bg-neutral-200 dark:hover:bg-neutral-700 font-semibold transition-colors text-sm"
			>
				<ClipboardList class="w-4 h-4" />
				Rekap Absensi
			</a>
			<a href={"/app/guru/kelas/" + kelas.id + "/riwayat-pertemuan"} use:inertia
				class="inline-flex items-center gap-2 px-5 py-2.5 rounded-xl bg-neutral-100 dark:bg-neutral-800 text-neutral-700 dark:text-neutral-300 hover:bg-neutral-200 dark:hover:bg-neutral-700 font-semibold transition-colors text-sm"
			>
				<Calendar class="w-4 h-4" />
				Riwayat Pertemuan
			</a>
			<button onclick={openBroadcast}
				class="inline-flex items-center gap-2 px-5 py-2.5 rounded-xl bg-green-600 hover:bg-green-700 text-white font-semibold transition-all text-sm"
			>
				<MessageSquare class="w-4 h-4" />
				Broadcast WA
			</button>
		</div>

		<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 overflow-hidden" in:fly={{ y: 20, duration: 600, delay: 100 }}>
			<div class="flex items-center justify-between px-6 py-4 border-b border-neutral-200/80 dark:border-white/[0.04]">
				<div class="flex items-center gap-2.5">
					<Users class="w-5 h-5 text-neutral-500" />
					<h3 class="text-base font-semibold text-neutral-900 dark:text-white">Daftar Santri</h3>
					<span class="text-sm font-normal text-neutral-500">({santri.length} santri)</span>
				</div>
			</div>

			{#if santri.length > 0}
				<div class="hidden sm:block overflow-x-auto">
					<table class="w-full">
						<thead>
							<tr class="text-xs font-medium text-neutral-500 dark:text-neutral-400 uppercase tracking-wider bg-neutral-50 dark:bg-neutral-900/50">
								<th class="text-left px-4 py-3">No</th>
								<th class="text-left px-4 py-3">Nama</th>
								<th class="text-left px-4 py-3">ID</th>
								<th class="text-left px-4 py-3">Domisili</th>
								<th class="text-center px-4 py-3">Hadir</th>
								<th class="text-center px-4 py-3">I/S/A/T</th>
								<th class="text-center px-4 py-3">%</th>
								<th class="text-left px-4 py-3">Terakhir</th>
								{#if kelas.materi_individual}<th class="text-left px-4 py-3">Batas Materi</th>{/if}
								<th class="text-right px-4 py-3">Aksi</th>
							</tr>
						</thead>
						<tbody class="divide-y divide-neutral-200/80 dark:divide-white/[0.04]">
							{#each santri as s, i}
								<tr class="hover:bg-neutral-50/50 dark:hover:bg-white/[0.015] transition-colors">
									<td class="px-4 py-3 text-sm text-neutral-500 dark:text-neutral-400 font-mono">{i + 1}</td>
									<td class="px-4 py-3">
										<span class="text-sm font-medium text-neutral-900 dark:text-white">{s.nama}</span>
									</td>
									<td class="px-4 py-3 text-xs font-mono text-neutral-500">{s.id_mahasantri}</td>
									<td class="px-4 py-3 text-sm text-neutral-600 dark:text-neutral-400">{s.domisili}</td>
									<td class="px-4 py-3 text-center text-sm font-mono text-green-600 dark:text-green-400">{s.total_hadir}</td>
									<td class="px-4 py-3 text-center text-xs font-mono text-neutral-500">
										{s.total_izin}/{s.total_sakit}/{s.total_alpa}/{s.total_telat}
									</td>
									<td class="px-4 py-3 text-center text-sm font-mono {attendanceColor(s.persen_hadir)}">
										{s.persen_hadir}%
									</td>
									<td class="px-4 py-3 text-xs text-neutral-500 font-mono">{s.tanggal_hadir_terakhir || "-"}</td>
									{#if kelas.materi_individual}<td class="px-4 py-3 text-xs text-neutral-600 dark:text-neutral-300">{s.batas_materi_terakhir || "-"}</td>{/if}
									<td class="px-4 py-3 text-right">
										<div class="flex items-center justify-end gap-1">
											<button onclick={() => openRiayahModal(s)}
												class="p-1.5 rounded-lg text-neutral-400 hover:text-brand-500 hover:bg-brand-400/10 transition-colors"
												title="Catatan"
											>
												<FileText class="w-4 h-4" />
											</button>
											<a href={"https://wa.me/" + s.no_wa.replace(/[^0-9]/g, "")} target="_blank" rel="noopener noreferrer"
												class="p-1.5 rounded-lg text-neutral-400 hover:text-green-500 hover:bg-green-500/10 transition-colors"
												title="Chat WA"
											>
												<MessageSquare class="w-4 h-4" />
											</a>
											<button onclick={() => toggleRiwayat(s.id)}
												class="p-1.5 rounded-lg text-neutral-400 hover:text-brand-500 hover:bg-brand-400/10 transition-colors"
												title="Riwayat"
											>
												{#if expandedRiwayat[s.id]}
													<ChevronDown class="w-4 h-4" />
												{:else}
													<ChevronRight class="w-4 h-4" />
												{/if}
											</button>
										</div>
									</td>
								</tr>
								{#if expandedRiwayat[s.id]}
									<tr class="bg-neutral-50/50 dark:bg-white/[0.015]">
										<td colspan={kelas.materi_individual ? 10 : 9} class="px-4 py-3">
											<div class="text-xs text-neutral-500">
												<p class="mb-1">Mulai: {s.tanggal_mulai} &middot; Hadir: {s.total_hadir} &middot; Izin: {s.total_izin} &middot; Sakit: {s.total_sakit} &middot; Alpa: {s.total_alpa} &middot; Telat: {s.total_telat}</p>
												<a href={"https://wa.me/" + s.no_wa.replace(/[^0-9]/g, "")} target="_blank" rel="noopener noreferrer" class="inline-flex items-center gap-1 text-brand-600 dark:text-brand-400 hover:underline mt-1">
													<ExternalLink class="w-3 h-3" />
													Hubungi via WhatsApp
												</a>
											</div>
										</td>
									</tr>
								{/if}
							{/each}
						</tbody>
					</table>
				</div>

				<div class="sm:hidden divide-y divide-neutral-200/80 dark:divide-white/[0.04]">
					{#each santri as s, i}
						<div class="p-4 space-y-2">
							<div class="flex items-center justify-between">
								<div class="flex items-center gap-2 min-w-0">
									<div class="w-8 h-8 rounded-full bg-brand-600 dark:bg-brand-500 flex items-center justify-center text-white font-bold text-xs shrink-0">
										{s.nama.charAt(0).toUpperCase()}
									</div>
									<div class="min-w-0">
										<p class="text-sm font-medium text-neutral-900 dark:text-white truncate">{s.nama}</p>
										<p class="text-xs text-neutral-500 font-mono">{s.id_mahasantri}</p>
									</div>
								</div>
								<span class="text-sm font-mono font-bold {attendanceColor(s.persen_hadir)}">{s.persen_hadir}%</span>
							</div>
							<div class="flex items-center gap-3 text-xs text-neutral-500">
								<span>{s.domisili}</span>
								<span class="text-neutral-400">|</span>
								<span>H:{s.total_hadir} I:{s.total_izin} S:{s.total_sakit} A:{s.total_alpa} T:{s.total_telat}</span>
							</div>
							<div class="flex items-center gap-2 pt-1">
								<button onclick={() => openRiayahModal(s)}
									class="inline-flex items-center gap-1 px-2.5 py-1.5 rounded-lg text-xs font-medium bg-brand-400/10 text-brand-600 dark:text-brand-400 hover:bg-brand-400/20 transition-colors"
								>
									<FileText class="w-3 h-3" /> Catatan
								</button>
								<a href={"https://wa.me/" + s.no_wa.replace(/[^0-9]/g, "")} target="_blank" rel="noopener noreferrer"
									class="inline-flex items-center gap-1 px-2.5 py-1.5 rounded-lg text-xs font-medium bg-green-500/10 text-green-600 dark:text-green-400 hover:bg-green-500/20 transition-colors"
								>
									<MessageSquare class="w-3 h-3" /> Chat WA
								</a>
								<button onclick={() => toggleRiwayat(s.id)}
									class="inline-flex items-center gap-1 px-2.5 py-1.5 rounded-lg text-xs font-medium bg-neutral-100 dark:bg-neutral-800 text-neutral-600 dark:text-neutral-400 hover:bg-neutral-200 dark:hover:bg-neutral-700 transition-colors"
								>
									{#if expandedRiwayat[s.id]}
										<ChevronDown class="w-3 h-3" />
									{:else}
										<ChevronRight class="w-3 h-3" />
									{/if}
									Riwayat
								</button>
							</div>
							{#if expandedRiwayat[s.id]}
								<div class="text-xs text-neutral-500 bg-neutral-50 dark:bg-neutral-900/50 rounded-xl p-3 space-y-1">
									<p>Mulai: {s.tanggal_mulai}</p>
									<p>Hadir: {s.total_hadir} | Izin: {s.total_izin} | Sakit: {s.total_sakit} | Alpa: {s.total_alpa} | Telat: {s.total_telat}</p>
								<p>Terakhir hadir: {s.tanggal_hadir_terakhir || "-"}</p>
								{#if kelas.materi_individual}<p>Batas materi: {s.batas_materi_terakhir || "-"}</p>{/if}
									<a href={"https://wa.me/" + s.no_wa.replace(/[^0-9]/g, "")} target="_blank" rel="noopener noreferrer" class="inline-flex items-center gap-1 text-brand-600 dark:text-brand-400 hover:underline mt-1">
										<ExternalLink class="w-3 h-3" /> Hubungi via WhatsApp
									</a>
								</div>
							{/if}
						</div>
					{/each}
				</div>
			{:else}
				<div class="p-12 text-center">
					<div class="w-12 h-12 rounded-xl bg-neutral-100 dark:bg-neutral-800 flex items-center justify-center mx-auto mb-3">
						<Users class="w-6 h-6 text-neutral-500" />
					</div>
					<p class="text-sm text-neutral-500 dark:text-neutral-400">Belum ada santri di kelas ini</p>
				</div>
			{/if}
		</div>

		{#if isBroadcastOpen}
			<div class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" role="presentation" onclick={() => (isBroadcastOpen = false)}>
				<div class="w-full max-w-2xl rounded-2xl border border-neutral-200 dark:border-white/[0.06] bg-white p-5 shadow-xl dark:bg-neutral-925" role="dialog" aria-modal="true" aria-labelledby="broadcast-title" onclick={(event) => event.stopPropagation()}>
					<div class="flex items-center justify-between gap-3">
						<div><h2 id="broadcast-title" class="font-semibold text-neutral-900 dark:text-white">Broadcast WhatsApp</h2><p class="text-sm text-neutral-500 dark:text-neutral-400">Salin pesan atau buka WhatsApp, lalu pilih grup kelas sebagai penerima.</p></div>
						<button onclick={() => (isBroadcastOpen = false)} class="rounded-lg px-2 py-1 text-neutral-500 hover:bg-neutral-100 dark:hover:bg-neutral-800">Tutup</button>
					</div>
					<textarea readonly value={broadcastMessage} aria-label="Pesan broadcast WhatsApp" class="mt-4 h-80 w-full resize-none rounded-xl border border-neutral-200 bg-neutral-50 p-3 font-mono text-xs leading-relaxed text-neutral-700 outline-none dark:border-white/[0.06] dark:bg-neutral-900/50 dark:text-neutral-300"></textarea>
					<div class="mt-4 flex flex-wrap justify-end gap-2">
						<button onclick={copyBroadcastMessage} class="inline-flex items-center gap-2 rounded-xl bg-neutral-100 px-4 py-2.5 text-sm font-semibold text-neutral-700 hover:bg-neutral-200 dark:bg-neutral-800 dark:text-neutral-200 dark:hover:bg-neutral-700">
							{#if isBroadcastCopied}<Check class="h-4 w-4 text-green-600" /> Tersalin{:else}<Copy class="h-4 w-4" /> Salin Pesan{/if}
						</button>
						<button onclick={openWhatsApp} class="inline-flex items-center gap-2 rounded-xl bg-green-600 px-4 py-2.5 text-sm font-semibold text-white hover:bg-green-700">
							<MessageSquare class="h-4 w-4" /> Buka WhatsApp
						</button>
					</div>
				</div>
			</div>
		{/if}
	</div>
</AppLayout>

{#if riayahModal.open}
	<div class="fixed inset-0 z-50 flex items-center justify-center p-4">
		<button class="absolute inset-0 w-full h-full bg-neutral-900/50 backdrop-blur-sm" aria-label="Tutup modal" onclick={() => (riayahModal = { open: false, santriId: 0, santriNama: "", notes: [], catatan: "", loading: false })}></button>
		<div class="relative w-full max-w-lg bg-white dark:bg-neutral-925 rounded-2xl shadow-xl border border-neutral-200/80 dark:border-white/[0.06] p-6" in:fly={{ y: 20, duration: 200 }}>
			<h3 class="text-lg font-bold text-neutral-900 dark:text-white mb-1">Catatan Riayah</h3>
			<p class="text-sm text-neutral-500 dark:text-neutral-400 mb-4">{riayahModal.santriNama}</p>

			{#if riayahModal.loading && riayahModal.notes.length === 0}
				<div class="text-center py-4 text-sm text-neutral-500">Memuat...</div>
			{/if}

			{#if riayahModal.notes.length > 0}
				<div class="space-y-3 mb-4 max-h-48 overflow-y-auto">
					{#each riayahModal.notes as note}
						<div class="p-3 rounded-xl bg-neutral-50 dark:bg-neutral-900/50 border border-neutral-200/80 dark:border-white/[0.04]">
							<p class="text-sm text-neutral-700 dark:text-neutral-300">{note.catatan}</p>
							<p class="text-xs text-neutral-500 mt-1">
								{note.penulis_nama} &middot; {note.created_at?.substring(0, 10)}
							</p>
						</div>
					{/each}
				</div>
			{/if}

			{#if canEdit}<textarea
				bind:value={riayahModal.catatan}
				placeholder="Tulis catatan baru..."
				class="w-full px-4 py-3 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white outline-none text-sm resize-none"
				rows="3"
			></textarea>{/if}

			<div class="flex items-center gap-3 mt-4">
				{#if canEdit}<button
					onclick={() => (riayahModal = { open: false, santriId: 0, santriNama: "", notes: [], catatan: "", loading: false })}
					class="flex-1 px-4 py-2.5 rounded-xl border border-neutral-300 dark:border-neutral-700/80 text-sm font-medium text-neutral-700 dark:text-neutral-300 hover:bg-neutral-100 dark:hover:bg-neutral-800 transition-colors"
				>
					Tutup
				</button>{/if}
				<button
					onclick={submitRiayah}
					disabled={!riayahModal.catatan.trim() || riayahModal.loading}
					class="flex-1 px-4 py-2.5 rounded-xl bg-brand-600 hover:bg-brand-700 text-white font-semibold transition-all dark:bg-brand-500 dark:hover:bg-brand-400 disabled:opacity-50 disabled:cursor-not-allowed text-sm"
				>
					{riayahModal.loading ? "Menyimpan..." : "Simpan"}
				</button>
			</div>
		</div>
	</div>
{/if}
