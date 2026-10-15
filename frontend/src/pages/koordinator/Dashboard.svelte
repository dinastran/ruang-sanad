<script lang="ts">
	import { inertia, router } from "@inertiajs/svelte";
	import AppLayout from "@layouts/AppLayout.svelte";
	import PertemuanBelumDitutupList from "@components/PertemuanBelumDitutup.svelte";
	import { formatJadwalSemula } from "@lib/utils/jadwal";
	import type { Flash, KoordinatorDashboard, MonitoringKelasItem, PertemuanBelumDitutup, User } from "@lib/types";
	import { AlertTriangle, ArrowRight, BellRing, CalendarDays, CheckCircle2, ClipboardCheck, ListTodo, Users, BookOpen, UserRoundCheck, Award, RotateCcw } from "lucide-svelte";

	interface Props {
		user?: User;
		flash?: Flash;
		error?: string;
		dashboard?: KoordinatorDashboard;
		kelas_hari_ini?: MonitoringKelasItem[];
		kelas_error?: string;
		belum_ditutup?: PertemuanBelumDitutup[];
		belum_ditutup_error?: string;
	}
	let { user, flash, error, dashboard, kelas_hari_ini = [], kelas_error, belum_ditutup = [], belum_ditutup_error }: Props = $props();

	const KELAS_TAMPIL = 8;
	const statusOrder: Record<string, number> = { terlambat: 0, tidak_terlaksana: 1, berlangsung: 2, belum_mulai: 3, selesai: 4, dibatalkan: 5 };
	const statusLabel: Record<string, string> = {
		terlambat: "Belum mulai",
		tidak_terlaksana: "Tidak terlaksana",
		berlangsung: "Berlangsung",
		belum_mulai: "Nanti",
		selesai: "Selesai",
		dibatalkan: "Dibatalkan",
	};
	const jenisLabel: Record<string, string> = { apresiasi: "Apresiasi", monitoring: "Monitoring", koordinasi: "Koordinasi", coaching: "Coaching", pembinaan: "Pembinaan" };

	const d = $derived(dashboard);
	const tindakLanjut = $derived(d?.tindak_lanjut ?? []);
	const todo = $derived(d?.todo ?? []);
	const agenda = $derived(d?.agenda ?? []);

	// Problems first, then by start time, so the koordinator sees what needs action without scrolling.
	const kelasUrut = $derived(
		[...kelas_hari_ini].sort((a, b) => (statusOrder[a.status] ?? 9) - (statusOrder[b.status] ?? 9) || a.jam_mulai.localeCompare(b.jam_mulai)),
	);
	const kelasAktif = $derived(kelas_hari_ini.filter((k) => k.status !== "dibatalkan"));
	const kelasSelesai = $derived(kelasAktif.filter((k) => k.status === "selesai").length);
	const progres = $derived(kelasAktif.length ? Math.round((kelasSelesai / kelasAktif.length) * 100) : 0);
	const kelasTelat = $derived(kelas_hari_ini.filter((k) => k.status === "terlambat").length);
	const totalAksi = $derived(kelasTelat + belum_ditutup.length + tindakLanjut.length + todo.length);

	let mengirim = $state<number | null>(null);

	function tanggalPanjang(iso: string): string {
		if (!iso) return "";
		return new Date(`${iso}T00:00:00`).toLocaleDateString("id-ID", { weekday: "long", day: "numeric", month: "long", year: "numeric" });
	}

	function tanggalPendek(iso: string): string {
		if (!iso) return "";
		return new Date(`${iso}T00:00:00`).toLocaleDateString("id-ID", { weekday: "short", day: "numeric", month: "short" });
	}

	function bulanLabel(ym: string): string {
		if (!ym) return "";
		return new Date(`${ym}-01T00:00:00`).toLocaleDateString("id-ID", { month: "long", year: "numeric" });
	}

	function namaGuru(item: MonitoringKelasItem): string {
		if (item.guru_pengganti_nama) return `${item.guru_pengganti_nama} (badal)`;
		return item.guru_utama_nama || "Belum ada guru";
	}

	// Jadwal tersimpan dalam WIB; hitung selisih dengan offset eksplisit agar benar di zona waktu mana pun.
	function telatMenit(item: MonitoringKelasItem): number {
		const mulai = new Date(`${item.tanggal}T${item.jam_mulai}:00+07:00`);
		return Math.max(0, Math.floor((Date.now() - mulai.getTime()) / 60000));
	}

	function badgeClass(status: string): string {
		if (status === "selesai") return "bg-success/10 text-emerald-700 dark:text-emerald-400";
		if (status === "berlangsung") return "bg-info/10 text-blue-700 dark:text-blue-400";
		if (status === "terlambat" || status === "tidak_terlaksana") return "bg-error/10 text-red-700 dark:text-red-400";
		return "bg-neutral-200/70 text-neutral-700 dark:bg-neutral-800 dark:text-neutral-300";
	}

	function bisaDiingatkan(item: MonitoringKelasItem): boolean {
		return item.status === "terlambat" && item.schedule_id > 0 && !!item.assigned_user_id;
	}

	function ingatkan(item: MonitoringKelasItem) {
		if (!confirm(`Kirim pengingat presensi ke ${namaGuru(item)} untuk kelas ${item.nama_kelas}?`)) return;
		mengirim = item.schedule_id;
		router.post(
			`/app/koordinator-guru/monitoring-kelas/${item.kelas_id}/schedules/${item.schedule_id}/reminder`,
			{ kind: "presensi_guru", message: "" },
			{ preserveScroll: true, onFinish: () => (mengirim = null) },
		);
	}

	const card = "rounded-2xl border border-neutral-200/80 bg-white dark:border-white/[0.06] dark:bg-neutral-925/50";
	const sectionTitle = "text-sm font-semibold text-neutral-900 dark:text-white";
</script>

<AppLayout {user} group="koordinator">
	<div class="mx-auto max-w-3xl space-y-6 px-4 pt-6 pb-10 sm:px-6 sm:pt-8">
		<header>
			<p class="text-sm text-neutral-500 dark:text-neutral-400">{tanggalPanjang(d?.tanggal ?? "")}</p>
			<h1 class="mt-1 text-2xl font-bold tracking-tight text-neutral-900 dark:text-white">Assalamu'alaikum{user?.name ? `, ${user.name}` : ""}</h1>
		</header>

		{#if flash?.success}<div role="status" class="rounded-xl border border-success/20 bg-success/10 p-4 text-sm font-medium text-emerald-700 dark:text-emerald-400">{flash.success}</div>{/if}
		{#if flash?.error}<div role="alert" class="rounded-xl border border-red-500/20 bg-red-500/10 p-4 text-sm font-medium text-red-700 dark:text-red-400">{flash.error}</div>{/if}
		{#if error}<div role="alert" class="rounded-xl border border-red-500/20 bg-red-500/10 p-4 text-sm font-medium text-red-700 dark:text-red-400">{error}</div>{/if}

		<!-- Ringkasan aksi hari ini -->
		{#if !d}
			<!-- Ringkasan gagal dimuat: pesan error di atas sudah cukup, jangan tampilkan hitungan yang tidak lengkap. -->
		{:else if totalAksi === 0 && !kelas_error && !belum_ditutup_error}
			<section class="flex items-center gap-3 rounded-2xl border border-success/20 bg-success/10 p-4 sm:p-5">
				<CheckCircle2 class="h-6 w-6 shrink-0 text-emerald-600 dark:text-emerald-400" />
				<div>
					<p class="font-semibold text-emerald-800 dark:text-emerald-300">Semua aman hari ini</p>
					<p class="text-sm text-emerald-700/80 dark:text-emerald-400/80">Tidak ada kelas telat, pertemuan menggantung, atau tindak lanjut yang jatuh tempo.</p>
				</div>
			</section>
		{:else}
			<section class="rounded-2xl border border-amber-500/30 bg-amber-500/[0.06] p-4 sm:p-5">
				<p class="flex items-center gap-2 text-sm font-semibold text-amber-800 dark:text-amber-300"><AlertTriangle class="h-4 w-4" /> {totalAksi} hal perlu tindakan hari ini</p>
				<div class="mt-3 grid grid-cols-2 gap-2 sm:grid-cols-4">
					{#each [
						{ href: "#kelas-hari-ini", n: kelasTelat, label: "Kelas belum mulai", tone: "text-red-600 dark:text-red-400" },
						{ href: "#belum-ditutup", n: belum_ditutup.length, label: "Belum ditutup", tone: "text-amber-700 dark:text-amber-400" },
						{ href: "#tindak-lanjut", n: tindakLanjut.length, label: "Tindak lanjut", tone: "text-neutral-900 dark:text-white" },
						{ href: "#tindak-lanjut", n: todo.length, label: "To-do jatuh tempo", tone: "text-neutral-900 dark:text-white" },
					] as s (s.label)}
						<a href={s.href} class="rounded-xl bg-white px-3 py-2.5 text-center transition-colors hover:bg-neutral-50 dark:bg-neutral-925/60 dark:hover:bg-neutral-900">
							<p class="text-xl font-bold {s.n > 0 ? s.tone : 'text-neutral-400 dark:text-neutral-600'}">{s.n}</p>
							<p class="text-xs text-neutral-600 dark:text-neutral-400">{s.label}</p>
						</a>
					{/each}
				</div>
			</section>
		{/if}

		<!-- Kelas hari ini -->
		<section id="kelas-hari-ini" class="scroll-mt-20">
			<div class="mb-2 flex items-baseline justify-between gap-3">
				<h2 class={sectionTitle}>Kelas hari ini {#if kelasAktif.length}<span class="font-normal text-neutral-500">· {kelasSelesai} dari {kelasAktif.length} selesai</span>{/if}</h2>
				<a href="/app/koordinator-guru/monitoring-kelas" use:inertia class="inline-flex shrink-0 items-center gap-1 text-xs font-semibold text-brand-700 hover:text-brand-800 dark:text-brand-300">Monitoring <ArrowRight class="h-3.5 w-3.5" /></a>
			</div>
			{#if kelas_error}
				<div class="rounded-xl border border-red-500/20 bg-red-500/10 p-4 text-sm text-red-700 dark:text-red-400">{kelas_error}</div>
			{:else if kelas_hari_ini.length === 0}
				<div class="{card} p-5 text-sm text-neutral-500">Tidak ada jadwal kelas hari ini.</div>
			{:else}
				<div class="mb-2 h-1.5 overflow-hidden rounded-full bg-neutral-200 dark:bg-neutral-800" role="progressbar" aria-valuenow={progres} aria-valuemin={0} aria-valuemax={100} aria-label="Kelas selesai hari ini">
					<div class="h-full rounded-full bg-emerald-500 transition-all" style="width: {progres}%"></div>
				</div>
				<ul class="{card} divide-y divide-neutral-200/80 dark:divide-white/[0.06]">
					{#each kelasUrut.slice(0, KELAS_TAMPIL) as item (`${item.schedule_id}-${item.pertemuan_id ?? 0}-${item.kelas_id}`)}
						<li class="flex items-center gap-3 px-4 py-3">
							<span class="w-11 shrink-0 text-xs tabular-nums text-neutral-500">{item.jam_mulai}</span>
							<div class="min-w-0 flex-1">
								<p class="truncate text-sm font-medium text-neutral-900 dark:text-white">{item.nama_kelas}</p>
								<p class="truncate text-xs text-neutral-500">
									{namaGuru(item)}
									{#if item.status === "terlambat"}<span class="text-red-600 dark:text-red-400"> · telat {telatMenit(item)} mnt</span>{/if}
								</p>
								{#if item.is_reschedule || item.guru_pengganti_id}
									<p class="mt-1 flex flex-wrap gap-1">
										{#if item.is_reschedule}<span class="inline-flex items-center gap-1 rounded-full bg-amber-500/10 px-2 py-0.5 text-[11px] font-semibold text-amber-700 dark:text-amber-400" title={item.alasan_reschedule ? `Alasan: ${item.alasan_reschedule}` : "Jadwal dipindah"}><RotateCcw class="h-3 w-3" /> Reschedule{#if item.jadwal_semula} · dari {formatJadwalSemula(item.jadwal_semula)}{/if}</span>{/if}
										{#if item.guru_pengganti_id}<span class="inline-flex items-center gap-1 rounded-full bg-secondary-500/10 px-2 py-0.5 text-[11px] font-semibold text-secondary-700 dark:text-secondary-300" title={item.alasan_badal ? `Alasan: ${item.alasan_badal}` : "Diajar guru pengganti"}><UserRoundCheck class="h-3 w-3" /> Badal · {item.guru_pengganti_nama}</span>{/if}
									</p>
								{/if}
							</div>
							<span class="shrink-0 rounded-full px-2 py-0.5 text-xs font-semibold {badgeClass(item.status)}">{statusLabel[item.status] ?? item.status}</span>
							{#if bisaDiingatkan(item)}
								<button type="button" onclick={() => ingatkan(item)} disabled={mengirim === item.schedule_id} aria-label="Kirim pengingat presensi ke guru" title="Kirim pengingat presensi" class="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-neutral-200 text-neutral-600 hover:border-brand-400/50 hover:text-brand-700 disabled:opacity-50 dark:border-neutral-700 dark:text-neutral-300 dark:hover:text-brand-300"><BellRing class="h-4 w-4" /></button>
							{/if}
						</li>
					{/each}
				</ul>
				{#if kelasUrut.length > KELAS_TAMPIL}
					<a href="/app/koordinator-guru/monitoring-kelas" use:inertia class="mt-2 block text-center text-xs font-semibold text-brand-700 dark:text-brand-300">Lihat {kelasUrut.length - KELAS_TAMPIL} kelas lainnya di monitoring</a>
				{/if}
			{/if}
		</section>

		<!-- Pertemuan belum ditutup -->
		<section id="belum-ditutup" class="scroll-mt-20">
			{#if belum_ditutup_error}
				<h2 class="mb-2 {sectionTitle}">Pertemuan belum ditutup</h2>
				<div class="rounded-xl border border-red-500/20 bg-red-500/10 p-4 text-sm text-red-700 dark:text-red-400">{belum_ditutup_error}</div>
			{:else if belum_ditutup.length > 0}
				<PertemuanBelumDitutupList items={belum_ditutup} showGuru />
			{/if}
		</section>

		<!-- Tindak lanjut dan to-do -->
		{#if d}
			<section id="tindak-lanjut" class="scroll-mt-20">
				<h2 class="mb-2 {sectionTitle}">Tindak lanjut dan to-do</h2>
				{#if tindakLanjut.length === 0 && todo.length === 0}
					<div class="{card} p-5 text-sm text-neutral-500">Tidak ada tindak lanjut atau to-do yang jatuh tempo.</div>
				{:else}
					<ul class="{card} divide-y divide-neutral-200/80 dark:divide-white/[0.06]">
						{#each tindakLanjut as tl (`tl-${tl.id}`)}
							<li>
								<a href={`/app/koordinator-guru/kunjungan/${tl.kunjungan_id}`} use:inertia class="flex gap-3 px-4 py-3 hover:bg-neutral-50 dark:hover:bg-white/[0.02]">
									<ClipboardCheck class="mt-0.5 h-4 w-4 shrink-0 text-neutral-400" />
									<div class="min-w-0">
										<p class="text-sm font-medium text-neutral-900 dark:text-white">{jenisLabel[tl.jenis] ?? tl.jenis} · {tl.guru_nama}</p>
										{#if tl.catatan}<p class="truncate text-xs text-neutral-500">{tl.catatan}</p>{/if}
										<p class="text-xs {tl.terlambat ? 'text-red-600 dark:text-red-400' : 'text-amber-700 dark:text-amber-400'}">{tl.terlambat ? `Lewat target ${tanggalPendek(tl.target_tanggal)}` : "Target hari ini"}</p>
									</div>
								</a>
							</li>
						{/each}
						{#each todo as t (`todo-${t.id}`)}
							<li>
								<a href="/app/koordinator-guru/todo" use:inertia class="flex gap-3 px-4 py-3 hover:bg-neutral-50 dark:hover:bg-white/[0.02]">
									<ListTodo class="mt-0.5 h-4 w-4 shrink-0 text-neutral-400" />
									<div class="min-w-0">
										<p class="text-sm font-medium text-neutral-900 dark:text-white">{t.judul}</p>
										<p class="text-xs {t.terlambat ? 'text-red-600 dark:text-red-400' : 'text-amber-700 dark:text-amber-400'}">
											{t.terlambat ? `Lewat deadline ${tanggalPendek(t.deadline)}` : "Deadline hari ini"}{#if t.pic}<span class="text-neutral-500"> · PIC {t.pic}</span>{/if}
										</p>
									</div>
								</a>
							</li>
						{/each}
					</ul>
				{/if}
			</section>

			<!-- Agenda 7 hari -->
			<section>
				<h2 class="mb-2 {sectionTitle}">Agenda 7 hari ke depan</h2>
				<ul class="{card} divide-y divide-neutral-200/80 dark:divide-white/[0.06]">
					{#each agenda as a (`${a.jenis}-${a.id}`)}
						<li>
							<a href={a.href} use:inertia class="flex items-center gap-3 px-4 py-3 hover:bg-neutral-50 dark:hover:bg-white/[0.02]">
								<span class="w-20 shrink-0 text-xs text-neutral-500">{tanggalPendek(a.tanggal)}{#if a.jam}<br />{a.jam}{/if}</span>
								<p class="min-w-0 flex-1 truncate text-sm text-neutral-900 dark:text-white">{a.judul}</p>
								<CalendarDays class="h-4 w-4 shrink-0 text-neutral-400" />
							</a>
						</li>
					{/each}
					{#if d.tsi_belum_final > 0}
						<li>
							<a href="/app/koordinator-guru/tsi" use:inertia class="flex items-center gap-3 px-4 py-3 hover:bg-neutral-50 dark:hover:bg-white/[0.02]">
								<span class="w-20 shrink-0 text-xs text-neutral-500">TSI</span>
								<p class="min-w-0 flex-1 text-sm text-neutral-900 dark:text-white">{bulanLabel(d.tsi_bulan)}: {d.tsi_belum_final} guru belum difinalisasi</p>
								<Award class="h-4 w-4 shrink-0 text-neutral-400" />
							</a>
						</li>
					{/if}
					{#if agenda.length === 0 && d.tsi_belum_final === 0}
						<li class="px-4 py-4 text-sm text-neutral-500">Belum ada pembinaan, rapat, atau kunjungan terjadwal.</li>
					{/if}
				</ul>
			</section>

			<!-- Statistik ringkas -->
			<section class="grid grid-cols-3 gap-2 border-t border-neutral-200/80 pt-5 text-center dark:border-white/[0.06]">
				<a href="/app/koordinator-guru/guru" use:inertia class="rounded-xl p-2 hover:bg-neutral-100 dark:hover:bg-white/[0.03]">
					<Users class="mx-auto h-4 w-4 text-neutral-400" />
					<p class="mt-1 text-lg font-bold text-neutral-900 dark:text-white">{d.total_guru}</p>
					<p class="text-xs text-neutral-500">Guru aktif</p>
					<p class="text-[11px] text-neutral-400">{d.guru_tetap} tetap · {d.guru_part_time} PT</p>
				</a>
				<div class="p-2">
					<BookOpen class="mx-auto h-4 w-4 text-neutral-400" />
					<p class="mt-1 text-lg font-bold text-neutral-900 dark:text-white">{d.total_kelas}</p>
					<p class="text-xs text-neutral-500">Kelas aktif</p>
				</div>
				<div class="p-2">
					<UserRoundCheck class="mx-auto h-4 w-4 text-neutral-400" />
					<p class="mt-1 text-lg font-bold text-neutral-900 dark:text-white">{d.total_santri}</p>
					<p class="text-xs text-neutral-500">Santri aktif</p>
				</div>
			</section>
		{/if}
	</div>
</AppLayout>
