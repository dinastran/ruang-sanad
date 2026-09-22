<script lang="ts">
	import { inertia, router } from "@inertiajs/svelte";
	import { fly } from "svelte/transition";
	import AppLayout from "@layouts/AppLayout.svelte";
	import { Toast } from "@lib/notifications/toast";
	import { getCSRFToken } from "@lib/utils/csrf";
	import type { User, GuruKelas, SantriGuru, PertemuanGuru } from "@lib/types";
	import { nomorPertemuan } from "@lib/pertemuan";
	import { ArrowLeft, BookOpen, Users, Calendar, Download, FileText } from "lucide-svelte";

	interface Props {
		user?: User;
		kelas?: GuruKelas;
		santri: SantriGuru[];
		pertemuan_list: PertemuanGuru[];
		absensi: Record<string, { id: number; status: string; catatan: string; batas_materi: string }>;
	}

	let { user, kelas, santri = [], pertemuan_list = [], absensi = {} }: Props = $props();
	let canEdit = $derived(user?.role === "guru" || user?.role === "admin_kelas" || user?.role === "super_admin");
	// The modal edits catatan and, for materi individual classes, batas materi. It
	// also carries a pending status when switching to hadir/telat needs a batas.
	let catatanModal = $state<{ key: string; santriNama: string; pertemuanKe: number; catatan: string; status: string; batasMateri: string } | null>(null);

	function butuhBatasMateri(status: string): boolean {
		return !!kelas?.materi_individual && (status === "hadir" || status === "telat");
	}
	let isSavingCatatan = $state(false);
	let updatingAbsensi = $state<Record<number, boolean>>({});

	function statusColor(status: string): string {
		const colors: Record<string, string> = {
			hadir: "bg-green-500 text-white",
			izin: "bg-yellow-500 text-white",
			sakit: "bg-blue-500 text-white",
			alpa: "bg-red-500 text-white",
			telat: "bg-orange-500 text-white",
		};
		return colors[status] || "bg-neutral-200 dark:bg-neutral-700 text-neutral-500";
	}

	function statusLabel(status: string): string {
		return status.charAt(0).toUpperCase();
	}

	function attendanceColor(pct: number): string {
		if (pct >= 85) return "text-green-600 dark:text-green-400";
		if (pct >= 70) return "text-amber-600 dark:text-amber-400";
		return "text-red-600 dark:text-red-400";
	}

	let totalPertemuan = $derived(pertemuan_list.length);

	async function updateAbsensi(id: number, status: string, catatan: string, batasMateri: string) {
		const response = await fetch(`/app/guru/kelas/${kelas?.id}/absensi/${id}`, {
			method: "PUT",
			headers: { "Content-Type": "application/json", "X-XSRF-TOKEN": getCSRFToken() },
			body: JSON.stringify({ status, catatan, batas_materi: batasMateri }),
		});
		if (!response.ok) {
			const data = await response.json().catch(() => ({}));
			throw new Error(data.error || "Gagal menyimpan absensi");
		}
	}

	function setUpdatingAbsensi(id: number, value: boolean) {
		updatingAbsensi = { ...updatingAbsensi, [id]: value };
	}

	function reloadAbsensi() {
		return new Promise<void>((resolve) => {
			router.reload({ only: ["absensi"], preserveScroll: true, onFinish: () => resolve() });
		});
	}

	async function changeStatus(key: string, status: string, select: HTMLSelectElement, santriNama: string, pertemuanKe: number) {
		const entry = absensi[key];
		if (!entry || updatingAbsensi[entry.id]) return;
		if (butuhBatasMateri(status) && !entry.batas_materi.trim()) {
			// The server requires batas materi here; ask for it before saving.
			select.value = entry.status;
			catatanModal = { key, santriNama, pertemuanKe, catatan: entry.catatan, status, batasMateri: "" };
			return;
		}
		absensi = { ...absensi, [key]: { ...entry, status } };
		setUpdatingAbsensi(entry.id, true);
		try {
			await updateAbsensi(entry.id, status, entry.catatan, entry.batas_materi);
		} catch (err) {
			Toast(err instanceof Error ? err.message : "Gagal menyimpan absensi", "error");
			await reloadAbsensi();
		} finally {
			setUpdatingAbsensi(entry.id, false);
		}
	}

	function openCatatan(key: string, santriNama: string, pertemuanKe: number) {
		const entry = absensi[key];
		if (!entry) return;
		catatanModal = { key, santriNama, pertemuanKe, catatan: entry.catatan, status: entry.status, batasMateri: entry.batas_materi };
	}

	async function saveCatatan() {
		if (!catatanModal) return;
		const entry = absensi[catatanModal.key];
		if (!entry || updatingAbsensi[entry.id]) return;
		const { key, catatan, status } = catatanModal;
		// Only replace batas materi when the form asks for it; otherwise keep what
		// was recorded (the server clears it for non hadir/telat statuses).
		const batasMateri = butuhBatasMateri(status) ? catatanModal.batasMateri.trim() : entry.batas_materi;
		if (butuhBatasMateri(status) && !batasMateri) {
			Toast("Batas materi wajib diisi untuk santri yang hadir/telat", "error");
			return;
		}
		isSavingCatatan = true;
		setUpdatingAbsensi(entry.id, true);
		try {
		absensi = { ...absensi, [key]: { ...entry, status, catatan, batas_materi: batasMateri } };
			await updateAbsensi(entry.id, status, catatan, batasMateri);
		catatanModal = null;
		Toast("Absensi tersimpan", "success");
	} catch (err) {
		Toast(err instanceof Error ? err.message : "Gagal menyimpan catatan", "error");
		await reloadAbsensi();
	} finally {
		isSavingCatatan = false;
		setUpdatingAbsensi(entry.id, false);
		}
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
				<span class="text-neutral-700 dark:text-neutral-300">Rekap Absensi</span>
			</div>

			<a href={"/app/guru/kelas/" + kelas?.id} use:inertia class="inline-flex items-center gap-1.5 text-sm text-neutral-500 hover:text-brand-600 dark:hover:text-brand-400 transition-colors mb-4">
				<ArrowLeft class="w-4 h-4" />
				Kembali
			</a>

			<div class="flex items-center justify-between gap-4 flex-wrap">
				<div class="flex items-center gap-4 min-w-0">
					<div class="w-14 h-14 rounded-2xl bg-brand-400/15 flex items-center justify-center shrink-0">
						<BookOpen class="w-7 h-7 text-brand-600 dark:text-brand-400" />
					</div>
					<div class="min-w-0">
						<h1 class="text-2xl sm:text-3xl font-bold text-neutral-900 dark:text-white tracking-tight break-words">{kelas?.nama_kelas || "Kelas"}</h1>
						<p class="text-neutral-600 dark:text-neutral-400">
							{kelas?.level || ""} &middot; {santri.length} santri &middot; {totalPertemuan} pertemuan
						</p>
					</div>
				</div>

				<button
					onclick={() => {}}
					class="inline-flex items-center gap-2 px-4 py-2.5 rounded-xl bg-neutral-100 dark:bg-neutral-800 text-neutral-700 dark:text-neutral-300 hover:bg-neutral-200 dark:hover:bg-neutral-700 font-semibold transition-colors text-sm"
					disabled
				>
					<Download class="w-4 h-4" />
					Export
				</button>
			</div>
		</div>
	</div>

	<div class="relative max-w-6xl mx-auto px-4 sm:px-6 py-8">
		{#if santri.length > 0 && pertemuan_list.length > 0}
			<div class="overflow-x-auto rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50" in:fly={{ y: 20, duration: 500 }}>
				<table class="w-full text-sm">
					<thead>
						<tr class="bg-neutral-50 dark:bg-neutral-900/50 border-b border-neutral-200/80 dark:border-white/[0.04]">
							<th class="sticky left-0 z-10 bg-neutral-50 dark:bg-neutral-900/50 text-left px-4 py-3 text-xs font-semibold text-neutral-500 dark:text-neutral-400 uppercase tracking-wider min-w-[160px]">Santri</th>
							<th class="text-center px-2 py-3 text-xs font-semibold text-neutral-500 dark:text-neutral-400 uppercase tracking-wider min-w-[36px]" title="Kehadiran %">%</th>
							{#each pertemuan_list as p, i}
								<th class="text-center px-2 py-3 text-xs font-medium text-neutral-500 dark:text-neutral-400 min-w-[36px]" title={"Pertemuan " + nomorPertemuan(p) + (p.level_nama ? " level " + p.level_nama : "") + " - " + p.tanggal}>
									<div class="flex flex-col items-center gap-0.5">
										{#if p.level_nama && (i === 0 || pertemuan_list[i - 1].level_nama !== p.level_nama)}
											<span class="rounded bg-brand-400/10 px-1 text-[9px] font-semibold text-brand-700 dark:text-brand-400">{p.level_nama}</span>
										{/if}
										<span class="font-semibold">#{nomorPertemuan(p)}</span>
										<span class="text-[9px] font-mono">{p.tanggal?.substring(5, 10) || ""}</span>
									</div>
								</th>
							{/each}
						</tr>
					</thead>
					<tbody class="divide-y divide-neutral-200/80 dark:divide-white/[0.04]">
						{#each santri as s}
							<tr class="hover:bg-neutral-50/50 dark:hover:bg-white/[0.015] transition-colors">
								<td class="sticky left-0 z-10 bg-white dark:bg-neutral-925/50 px-4 py-3">
									<div class="flex items-center gap-2 min-w-0">
										<div class="w-7 h-7 rounded-full bg-brand-600 dark:bg-brand-500 flex items-center justify-center text-white font-bold text-xs shrink-0">
											{s.nama.charAt(0).toUpperCase()}
										</div>
										<span class="text-sm font-medium text-neutral-900 dark:text-white truncate">{s.nama}</span>
									</div>
								</td>
								<td class="text-center px-2 py-3 text-sm font-mono font-bold {attendanceColor(s.persen_hadir)}">
									{s.persen_hadir}%
								</td>
								{#each pertemuan_list as p}
									{@const entry = absensi[`${s.id}:${p.id}`]}
									<td class="text-center px-2 py-3">
										{#if entry}
											{#if canEdit}
												<select value={entry.status} disabled={updatingAbsensi[entry.id]} onchange={(event) => changeStatus(`${s.id}:${p.id}`, event.currentTarget.value, event.currentTarget, s.nama, nomorPertemuan(p))} aria-label={`Absensi ${s.nama} pertemuan ${nomorPertemuan(p)}`} class="w-8 h-7 rounded-md border-0 text-xs font-bold text-center {statusColor(entry.status)} disabled:cursor-wait disabled:opacity-60">
													<option value="hadir">H</option><option value="izin">I</option><option value="sakit">S</option><option value="alpa">A</option><option value="telat">T</option>
												</select>
											{:else}
												<span class="inline-flex items-center justify-center w-7 h-7 rounded-md text-xs font-bold {statusColor(entry.status)}">{statusLabel(entry.status)}</span>
											{/if}
											<button onclick={() => openCatatan(`${s.id}:${p.id}`, s.nama, nomorPertemuan(p))} disabled={updatingAbsensi[entry.id]} class="ml-1 inline-flex h-5 w-5 items-center justify-center rounded hover:bg-brand-400/10 hover:text-brand-600 disabled:cursor-wait disabled:opacity-60 dark:hover:text-brand-400 {entry.catatan.trim().length > 0 ? 'text-brand-600 dark:text-brand-400' : 'text-neutral-400'}" title={entry.catatan.trim() ? "Lihat catatan absensi" : "Tambah catatan absensi"} aria-label={`Catatan absensi ${s.nama} pertemuan ${nomorPertemuan(p)}`}>
												<FileText class="h-3.5 w-3.5" />
											</button>
										{:else}
											<span class="text-neutral-400">-</span>
										{/if}
									</td>
								{/each}
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{:else}
			<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-12 text-center" in:fly={{ y: 20, duration: 500 }}>
				<div class="w-16 h-16 rounded-2xl bg-neutral-100 dark:bg-neutral-800 flex items-center justify-center mx-auto mb-4">
					<Calendar class="w-8 h-8 text-neutral-500" />
				</div>
				<h3 class="text-lg font-semibold text-neutral-900 dark:text-white mb-2">
					{pertemuan_list.length === 0 ? "Belum ada pertemuan" : "Belum ada santri"}
				</h3>
				<p class="text-neutral-500 dark:text-neutral-400 max-w-md mx-auto text-sm">
					{pertemuan_list.length === 0 ? "Data absensi akan muncul setelah pertemuan pertama dimulai" : "Tidak ada santri untuk ditampilkan"}
				</p>
			</div>
		{/if}
	</div>

	{#if catatanModal}
		<div class="fixed inset-0 z-50 flex items-center justify-center p-4">
			<button class="absolute inset-0 h-full w-full bg-neutral-900/50 backdrop-blur-sm" aria-label="Tutup catatan absensi" onclick={() => (catatanModal = null)}></button>
			<div class="relative w-full max-w-lg rounded-2xl border border-neutral-200/80 bg-white p-6 shadow-xl dark:border-white/[0.06] dark:bg-neutral-925" in:fly={{ y: 16, duration: 200 }} role="dialog" aria-modal="true" aria-labelledby="catatan-absensi-title" tabindex="-1">
				<div class="flex items-start gap-3">
					<div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-brand-400/10 text-brand-600 dark:text-brand-400"><FileText class="h-5 w-5" /></div>
					<div>
						<h2 id="catatan-absensi-title" class="font-semibold text-neutral-900 dark:text-white">Catatan Absensi</h2>
						<p class="mt-1 text-sm text-neutral-500 dark:text-neutral-400">{catatanModal.santriNama} · Pertemuan ke-{catatanModal.pertemuanKe}</p>
					</div>
				</div>

				<div class="mt-5 space-y-4">
					{#if catatanModal.status !== absensi[catatanModal.key]?.status}
						<p class="rounded-xl bg-brand-400/10 px-3 py-2.5 text-sm text-neutral-700 dark:text-neutral-300">Status akan diubah menjadi <span class="font-semibold capitalize">{catatanModal.status}</span>. Isi batas materi untuk menyimpan.</p>
					{/if}
					{#if kelas?.materi_individual}
						{#if canEdit && butuhBatasMateri(catatanModal.status)}
							<label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300">Batas materi
								<input bind:value={catatanModal.batasMateri} placeholder="Batas materi *" class="mt-1.5 w-full rounded-xl border border-neutral-300 bg-neutral-100/80 px-3 py-2.5 text-sm text-neutral-900 outline-none focus:border-brand-400 focus:ring-2 focus:ring-brand-400/20 dark:border-neutral-700/80 dark:bg-neutral-800/50 dark:text-white" />
							</label>
						{:else}
							<div>
								<p class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Batas materi</p>
								<p class="mt-1.5 rounded-xl bg-neutral-100/80 px-3 py-2.5 text-sm text-neutral-700 dark:bg-neutral-800/50 dark:text-neutral-300">{absensi[catatanModal.key]?.batas_materi || "Tidak ada progres baru"}</p>
							</div>
						{/if}
					{/if}
					<label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300">Catatan
						<textarea bind:value={catatanModal.catatan} readonly={!canEdit} placeholder="Catatan absensi mahasantri..." rows="4" class="mt-1.5 w-full resize-none rounded-xl border border-neutral-300 bg-neutral-100/80 px-3 py-2.5 text-sm text-neutral-900 outline-none focus:border-brand-400 focus:ring-2 focus:ring-brand-400/20 dark:border-neutral-700/80 dark:bg-neutral-800/50 dark:text-white"></textarea>
					</label>
				</div>

				<div class="mt-5 flex justify-end gap-2">
					<button onclick={() => (catatanModal = null)} class="rounded-xl border border-neutral-300 px-4 py-2.5 text-sm font-medium text-neutral-700 hover:bg-neutral-100 dark:border-neutral-700/80 dark:text-neutral-300 dark:hover:bg-neutral-800">Tutup</button>
					{#if canEdit}<button onclick={saveCatatan} disabled={isSavingCatatan} class="rounded-xl bg-brand-600 px-4 py-2.5 text-sm font-semibold text-white transition-colors hover:bg-brand-700 disabled:cursor-not-allowed disabled:opacity-50 dark:bg-brand-500 dark:hover:bg-brand-400">{isSavingCatatan ? "Menyimpan..." : "Simpan"}</button>{/if}
				</div>
			</div>
		</div>
	{/if}
</AppLayout>
