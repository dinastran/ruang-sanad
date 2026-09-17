<script lang="ts">
	import { inertia, router } from "@inertiajs/svelte";
	import { fly } from "svelte/transition";
	import AppLayout from "@layouts/AppLayout.svelte";
	import type { User, GuruKelas, SantriGuru, PertemuanGuru } from "@lib/types";
	import { ArrowLeft, BookOpen, Users, Calendar, Download } from "lucide-svelte";

	interface Props {
		user?: User;
		kelas?: GuruKelas;
		santri: SantriGuru[];
		pertemuan_list: PertemuanGuru[];
		absensi: Record<string, { id: number; status: string; catatan: string }>;
	}

	let { user, kelas, santri = [], pertemuan_list = [], absensi = {} }: Props = $props();
	let canEdit = $derived(user?.role === "guru" || user?.role === "admin_kelas" || user?.role === "super_admin");

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

	function updateAbsensi(id: number, status: string, catatan: string) {
		router.put(`/app/guru/kelas/${kelas?.id}/absensi/${id}`, { status, catatan }, { preserveScroll: true });
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
							{#each pertemuan_list as p}
								<th class="text-center px-2 py-3 text-xs font-medium text-neutral-500 dark:text-neutral-400 min-w-[36px]" title={"Pertemuan " + p.pertemuan_ke + " - " + p.tanggal}>
									<div class="flex flex-col items-center gap-0.5">
										<span class="font-semibold">#{p.pertemuan_ke}</span>
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
												<select value={entry.status} onchange={(event) => updateAbsensi(entry.id, event.currentTarget.value, entry.catatan)} aria-label={`Absensi ${s.nama} pertemuan ${p.pertemuan_ke}`} class="w-8 h-7 rounded-md border-0 text-xs font-bold text-center {statusColor(entry.status)}">
													<option value="hadir">H</option><option value="izin">I</option><option value="sakit">S</option><option value="alpa">A</option><option value="telat">T</option>
												</select>
											{:else}
												<span class="inline-flex items-center justify-center w-7 h-7 rounded-md text-xs font-bold {statusColor(entry.status)}">{statusLabel(entry.status)}</span>
											{/if}
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
</AppLayout>
