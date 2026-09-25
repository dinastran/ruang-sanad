<script lang="ts">
	import { inertia, router } from "@inertiajs/svelte";
	import { fly } from "svelte/transition";
	import AppLayout from "@layouts/AppLayout.svelte";
	import type { AppNotification, Flash, User, GuruDashboard, TilawahStatus, RiayahRingkasan } from "@lib/types";
	import { BookOpen, Users, Clock, CheckCircle, AlertCircle, ArrowRight, Play, BookMarked, BellRing, HeartHandshake } from "lucide-svelte";

	interface Props {
		user?: User;
		dashboard?: GuruDashboard;
		tilawah?: TilawahStatus;
		notifications?: AppNotification[];
		riayah?: RiayahRingkasan | null;
		flash?: Flash;
		success?: string;
		error?: string;
	}

	let { user, dashboard, tilawah, notifications = [], riayah = null, flash, success, error }: Props = $props();

	let canEdit = $derived(user?.role === "guru");
	let d = $derived(dashboard as GuruDashboard);
	let jadwalHariIni = $derived(d?.jadwal_hari_ini ?? []);
	let kelasBelumAbsen = $derived(d?.kelas_belum_absen ?? []);
	let tilawahBusy = $state(false);
	let notificationBusy = $state<number | null>(null);
	function toggleTilawah() {
		if (!canEdit || !tilawah) return;
		tilawahBusy = true;
		if (tilawah.sudah_hari_ini) router.delete("/app/guru/tilawah", { preserveScroll: true, onFinish: () => (tilawahBusy = false) });
		else router.post("/app/guru/tilawah", {}, { preserveScroll: true, onFinish: () => (tilawahBusy = false) });
	}

	function markNotificationRead(id: number) {
		notificationBusy = id;
		router.put(`/app/notifications/${id}/read`, {}, {
			preserveScroll: true,
			onFinish: () => (notificationBusy = null),
		});
	}
</script>

<AppLayout {user} group="guru-dashboard">
	<div class="pt-8 pb-10 border-b border-neutral-200/80 dark:border-white/[0.04]">
		<div class="max-w-6xl mx-auto px-4 sm:px-6">
			<h1 class="text-2xl sm:text-3xl font-bold text-neutral-900 dark:text-white mb-2 tracking-tight">
				Dashboard Guru
			</h1>
			<p class="text-neutral-600 dark:text-neutral-400">Ringkasan kelas dan aktivitas mengajar</p>
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

		{#if notifications.length > 0}
			<section class="overflow-hidden rounded-2xl border border-brand-400/20 bg-brand-400/5" aria-labelledby="notification-title">
				<div class="flex items-center justify-between gap-3 border-b border-brand-400/15 px-5 py-4">
					<div class="flex items-center gap-2.5">
						<BellRing class="h-5 w-5 text-brand-600 dark:text-brand-400" />
						<h2 id="notification-title" class="font-semibold text-neutral-900 dark:text-white">Pengingat Koordinator</h2>
					</div>
					<span class="text-xs text-neutral-500">{notifications.filter((item) => !item.read).length} belum dibaca</span>
				</div>
				<div class="divide-y divide-brand-400/10">
					{#each notifications as item (item.id)}
						<article class="flex flex-col gap-3 px-5 py-4 sm:flex-row sm:items-start sm:justify-between {item.read ? 'opacity-65' : ''}">
							<div class="min-w-0">
								<div class="flex flex-wrap items-center gap-2">
									<h3 class="text-sm font-semibold text-neutral-900 dark:text-white">{item.title}</h3>
									{#if !item.read}<span class="rounded-full bg-brand-600 px-2 py-0.5 text-[10px] font-bold uppercase tracking-wide text-white">Baru</span>{/if}
								</div>
								<p class="mt-1 text-sm text-neutral-600 dark:text-neutral-300">{item.message}</p>
								<p class="mt-1.5 text-xs text-neutral-500">{item.created_at}</p>
							</div>
							<div class="flex shrink-0 items-center gap-2">
								{#if item.action_url}<a href={item.action_url} use:inertia class="inline-flex min-h-11 items-center rounded-lg px-3 text-xs font-semibold text-brand-700 hover:bg-brand-400/10 dark:text-brand-300">Buka jadwal</a>{/if}
								{#if !item.read}<button onclick={() => markNotificationRead(item.id)} disabled={notificationBusy === item.id} class="min-h-11 rounded-lg border border-neutral-200 px-3 text-xs font-semibold text-neutral-700 hover:border-brand-400/40 disabled:opacity-50 dark:border-neutral-700 dark:text-neutral-300">{notificationBusy === item.id ? "Memproses..." : "Tandai dibaca"}</button>{/if}
							</div>
						</article>
					{/each}
				</div>
			</section>
		{/if}

		<div class="grid md:grid-cols-4 gap-5" in:fly={{ y: 20, duration: 600 }}>
			<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-5 transition-all hover:border-brand-400/30">
				<div class="flex items-center gap-3 mb-2">
					<div class="w-10 h-10 rounded-xl bg-brand-400/10 flex items-center justify-center">
						<BookOpen class="w-5 h-5 text-brand-600 dark:text-brand-400" />
					</div>
					<span class="text-sm font-medium text-neutral-500 dark:text-neutral-400">Total Kelas</span>
				</div>
				<div class="text-2xl sm:text-3xl font-bold text-neutral-900 dark:text-white tracking-tight font-mono">{d?.total_kelas || 0}</div>
			</div>
			<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-5 transition-all hover:border-blue-500/30">
				<div class="flex items-center gap-3 mb-2">
					<div class="w-10 h-10 rounded-xl bg-blue-500/10 flex items-center justify-center">
						<Users class="w-5 h-5 text-blue-600 dark:text-blue-400" />
					</div>
					<span class="text-sm font-medium text-neutral-500 dark:text-neutral-400">Total Santri</span>
				</div>
				<div class="text-2xl sm:text-3xl font-bold text-neutral-900 dark:text-white tracking-tight font-mono">{d?.total_santri || 0}</div>
			</div>
			<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-5 transition-all hover:border-green-500/30">
				<div class="flex items-center gap-3 mb-2">
					<div class="w-10 h-10 rounded-xl bg-green-500/10 flex items-center justify-center">
						<CheckCircle class="w-5 h-5 text-green-600 dark:text-green-400" />
					</div>
					<span class="text-sm font-medium text-neutral-500 dark:text-neutral-400">Santri Aktif</span>
				</div>
				<div class="text-2xl sm:text-3xl font-bold text-neutral-900 dark:text-white tracking-tight font-mono">{d?.santri_aktif || 0}</div>
			</div>
			<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-5 transition-all hover:border-amber-500/30">
				<div class="flex items-center gap-3 mb-2">
					<div class="w-10 h-10 rounded-xl bg-amber-500/10 flex items-center justify-center">
						<Clock class="w-5 h-5 text-amber-600 dark:text-amber-400" />
					</div>
					<span class="text-sm font-medium text-neutral-500 dark:text-neutral-400">Total Pertemuan</span>
				</div>
				<div class="text-2xl sm:text-3xl font-bold text-neutral-900 dark:text-white tracking-tight font-mono">{d?.total_pertemuan || 0}</div>
			</div>
		</div>

		{#if tilawah}
			<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-5 flex items-center justify-between gap-4 flex-wrap" in:fly={{ y: 20, duration: 600, delay: 80 }}>
				<div class="flex items-center gap-3">
					<div class="w-11 h-11 rounded-xl bg-secondary-500/10 flex items-center justify-center"><BookMarked class="w-5 h-5 text-secondary-600 dark:text-secondary-400" /></div>
					<div>
						<p class="text-sm font-semibold text-neutral-900 dark:text-white">Tilawah Harian</p>
						<p class="text-xs text-neutral-500 dark:text-neutral-400">{tilawah.bulan_ini} hari bulan ini{tilawah.sudah_hari_ini ? " · sudah hari ini ✓" : ""}</p>
					</div>
				</div>
				{#if canEdit}
					<button onclick={toggleTilawah} disabled={tilawahBusy} class="inline-flex items-center gap-2 px-4 py-2.5 rounded-xl text-sm font-semibold disabled:opacity-50 {tilawah.sudah_hari_ini ? 'bg-neutral-500/10 text-neutral-600 dark:text-neutral-300' : 'bg-secondary-600 hover:bg-secondary-700 text-white'}">
						<CheckCircle class="w-4 h-4" /> {tilawah.sudah_hari_ini ? "Batalkan" : "Tandai sudah tilawah"}
					</button>
				{/if}
			</div>
		{/if}

		{#if riayah}
			<a href="/app/guru/riayah" use:inertia class="block rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-5 transition-all hover:border-brand-400/30 focus:outline-none focus-visible:ring-2 focus-visible:ring-brand-400/50" in:fly={{ y: 20, duration: 600, delay: 90 }}>
				<div class="flex items-center justify-between gap-4 flex-wrap">
					<div class="flex items-center gap-3">
						<div class="w-11 h-11 rounded-xl bg-brand-400/10 flex items-center justify-center"><HeartHandshake class="w-5 h-5 text-brand-600 dark:text-brand-400" /></div>
						<div>
							<p class="text-sm font-semibold text-neutral-900 dark:text-white">Riayah Santri</p>
							<p class="text-xs text-neutral-500 dark:text-neutral-400">
								{#if riayah.perlu_perhatian > 0}{riayah.perlu_perhatian} dari {riayah.total_santri} santri perlu diperhatikan{:else}Tidak ada santri yang ditandai dari {riayah.total_santri} santri{/if}
							</p>
						</div>
					</div>
					<div class="flex flex-wrap items-center gap-2 text-xs">
						{#if riayah.kehadiran > 0}<span class="inline-flex items-center gap-1.5 rounded-lg bg-red-500/10 px-2 py-1 font-medium text-red-700 dark:text-red-300"><span class="h-1.5 w-1.5 rounded-full bg-red-500"></span>Kehadiran {riayah.kehadiran}</span>{/if}
						{#if riayah.kontak > 0}<span class="inline-flex items-center gap-1.5 rounded-lg bg-orange-500/10 px-2 py-1 font-medium text-orange-700 dark:text-orange-300"><span class="h-1.5 w-1.5 rounded-full bg-orange-500"></span>Belum disapa {riayah.kontak}</span>{/if}
						{#if riayah.progres > 0}<span class="inline-flex items-center gap-1.5 rounded-lg bg-amber-500/10 px-2 py-1 font-medium text-amber-800 dark:text-amber-300"><span class="h-1.5 w-1.5 rounded-full bg-amber-500"></span>Progres macet {riayah.progres}</span>{/if}
						<ArrowRight class="w-4 h-4 text-brand-500" />
					</div>
				</div>
			</a>
		{/if}

		{#if jadwalHariIni.length > 0}
			<div class="rounded-2xl border border-brand-400/20 bg-brand-400/5 overflow-hidden" in:fly={{ y: 20, duration: 600, delay: 100 }}>
				<div class="flex items-center gap-2.5 px-6 py-4 border-b border-brand-400/15">
					<Clock class="w-5 h-5 text-brand-600 dark:text-brand-400" />
					<h3 class="text-base font-semibold text-brand-800 dark:text-brand-300">Jadwal Hari Ini</h3>
					<span class="text-sm font-normal text-brand-600 dark:text-brand-400">({jadwalHariIni.length} kelas)</span>
				</div>
				<div class="divide-y divide-brand-400/10">
					{#each jadwalHariIni as k}
						<a href={"/app/guru/kelas/" + k.id} use:inertia class="flex items-center justify-between px-6 py-3.5 hover:bg-brand-400/5 transition-colors">
							<div class="flex items-center gap-3 min-w-0">
								<div class="w-8 h-8 rounded-full bg-brand-400/15 flex items-center justify-center text-brand-700 dark:text-brand-400 text-xs font-bold shrink-0">
									<BookOpen class="w-4 h-4" />
								</div>
								<div class="min-w-0">
									<p class="text-sm font-medium text-neutral-900 dark:text-white truncate">{k.nama_kelas}</p>
									<p class="text-xs text-neutral-500 dark:text-neutral-400">{k.level} &middot; {k.jadwal}</p>
								</div>
							</div>
							<ArrowRight class="w-4 h-4 text-brand-500 shrink-0" />
						</a>
					{/each}
				</div>
			</div>
		{/if}

		{#if kelasBelumAbsen.length > 0}
			<div class="rounded-2xl border border-red-500/20 bg-red-500/5 overflow-hidden" in:fly={{ y: 20, duration: 600, delay: 150 }}>
				<div class="flex items-center gap-2.5 px-6 py-4 border-b border-red-500/15">
					<AlertCircle class="w-5 h-5 text-red-600 dark:text-red-400" />
					<h3 class="text-base font-semibold text-red-800 dark:text-red-300">Kelas Belum Diabsen</h3>
					<span class="text-sm font-normal text-red-600 dark:text-red-400">({kelasBelumAbsen.length} kelas)</span>
				</div>
				<div class="divide-y divide-red-500/10">
					{#each kelasBelumAbsen as k}
						<a href={"/app/guru/kelas/" + k.id + (canEdit ? "/pertemuan/mulai" : "")} use:inertia class="flex items-center justify-between px-6 py-3.5 hover:bg-red-500/5 transition-colors">
							<div class="flex items-center gap-3 min-w-0">
								<div class="w-8 h-8 rounded-full bg-red-500/15 flex items-center justify-center text-red-700 dark:text-red-400 text-xs font-bold shrink-0">
									<AlertCircle class="w-4 h-4" />
								</div>
								<div class="min-w-0">
									<p class="text-sm font-medium text-neutral-900 dark:text-white truncate">{k.nama_kelas}</p>
									<p class="text-xs text-red-600 dark:text-red-400">{k.level} &middot; {k.jadwal}</p>
								</div>
							</div>
							<Play class="w-4 h-4 text-red-500 shrink-0" />
						</a>
					{/each}
				</div>
			</div>
		{/if}

		{#if jadwalHariIni.length === 0 && kelasBelumAbsen.length === 0}
			<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-12 text-center" in:fly={{ y: 20, duration: 500, delay: 100 }}>
				<div class="w-16 h-16 rounded-2xl bg-neutral-100 dark:bg-neutral-800 flex items-center justify-center mx-auto mb-4">
					<BookOpen class="w-8 h-8 text-neutral-500" />
				</div>
				<h3 class="text-lg font-semibold text-neutral-900 dark:text-white mb-2">Tidak ada aktivitas hari ini</h3>
				<p class="text-neutral-500 dark:text-neutral-400 max-w-md mx-auto text-sm">
					Kelas akan muncul di sini ketika ada jadwal mengajar atau pertemuan yang perlu diabsen
				</p>
			</div>
		{/if}
	</div>
</AppLayout>
