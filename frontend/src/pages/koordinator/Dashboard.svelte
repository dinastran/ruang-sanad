<script lang="ts">
	import { inertia } from "@inertiajs/svelte";
	import AppLayout from "@layouts/AppLayout.svelte";
	import type { KoordinatorDashboard, User } from "@lib/types";
	import { Users, BookOpen, UserRoundCheck, GraduationCap, AlertTriangle, MessageCircle, CalendarClock, ArrowRight } from "lucide-svelte";

	interface Props { user?: User; dashboard?: KoordinatorDashboard; error?: string; }
	let { user, dashboard, error }: Props = $props();
	const d = $derived(dashboard ?? { total_guru: 0, guru_tetap: 0, guru_part_time: 0, total_kelas: 0, total_santri: 0, guru_belum_absen: [] });

	function waLink(no: string): string {
		let p = no.trim();
		if (!p) return "#";
		if (p.startsWith("0")) p = "62" + p.slice(1);
		else if (!p.startsWith("+")) p = "62" + p;
		return `https://wa.me/${p.replace("+", "")}`;
	}
</script>

<AppLayout {user} group="koordinator">
	<div class="pt-8 pb-10 border-b border-neutral-200/80 dark:border-white/[0.04]">
		<div class="max-w-6xl mx-auto px-4 sm:px-6">
			<h1 class="text-2xl sm:text-3xl font-bold text-neutral-900 dark:text-white tracking-tight">Dashboard Koordinator Guru</h1>
			<p class="mt-2 text-neutral-600 dark:text-neutral-400">Ringkasan guru, kelas, dan kepatuhan absensi.</p>
		</div>
	</div>

	<div class="max-w-6xl mx-auto px-4 sm:px-6 py-8 space-y-6">
		{#if error}<div class="rounded-xl bg-red-500/10 border border-red-500/20 p-4 text-sm font-medium text-red-700 dark:text-red-400">{error}</div>{/if}

		<a href="/app/koordinator-guru/monitoring-kelas" use:inertia class="group flex flex-col gap-4 rounded-2xl border border-brand-400/30 bg-brand-400/10 p-5 transition-colors hover:border-brand-400/60 sm:flex-row sm:items-center sm:justify-between sm:p-6">
			<div class="flex items-start gap-4">
				<div class="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-brand-600 text-white dark:bg-brand-500"><CalendarClock class="h-5 w-5" /></div>
				<div><h2 class="font-semibold text-neutral-900 dark:text-white">Monitoring Kelas</h2><p class="mt-1 text-sm text-neutral-600 dark:text-neutral-300">Pantau sesi berjalan, kepatuhan absensi, dan kelas yang perlu ditindaklanjuti.</p></div>
			</div>
			<span class="inline-flex items-center gap-2 self-start whitespace-nowrap text-sm font-semibold text-brand-700 dark:text-brand-300 sm:self-auto">Buka monitoring <ArrowRight class="h-4 w-4 transition-transform group-hover:translate-x-0.5" /></span>
		</a>

		<div class="grid grid-cols-2 lg:grid-cols-4 gap-4">
			<div class="rounded-2xl bg-brand-400/10 border border-brand-400/20 p-5"><Users class="w-5 h-5 text-brand-600 dark:text-brand-300" /><p class="mt-3 text-2xl font-bold text-neutral-900 dark:text-white">{d.total_guru}</p><p class="text-sm text-neutral-600 dark:text-neutral-400">Guru aktif</p><p class="mt-1 text-xs text-neutral-500">{d.guru_tetap} tetap · {d.guru_part_time} part time</p></div>
			<div class="rounded-2xl bg-secondary-500/10 border border-secondary-500/20 p-5"><BookOpen class="w-5 h-5 text-secondary-600 dark:text-secondary-300" /><p class="mt-3 text-2xl font-bold text-neutral-900 dark:text-white">{d.total_kelas}</p><p class="text-sm text-neutral-600 dark:text-neutral-400">Kelas aktif</p></div>
			<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-5"><UserRoundCheck class="w-5 h-5 text-neutral-500" /><p class="mt-3 text-2xl font-bold text-neutral-900 dark:text-white">{d.total_santri}</p><p class="text-sm text-neutral-600 dark:text-neutral-400">Santri aktif</p></div>
			<a href="/app/koordinator-guru/guru" use:inertia class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-5 hover:border-brand-400/40 transition-colors"><GraduationCap class="w-5 h-5 text-neutral-500" /><p class="mt-3 text-sm font-semibold text-neutral-900 dark:text-white">Kelola Data Guru</p><p class="text-xs text-neutral-500 mt-1">Profil, kompetensi, riayah</p></a>
		</div>

		<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-5 sm:p-6">
			<div class="flex items-center gap-2 mb-4"><AlertTriangle class="w-5 h-5 text-amber-500" /><h2 class="font-semibold text-neutral-900 dark:text-white">Guru belum mengisi absensi pekan ini</h2></div>
			{#if d.guru_belum_absen.length === 0}
				<p class="text-sm text-neutral-500">Semua guru sudah mengisi absensi pekan ini.</p>
			{:else}
				<div class="space-y-2">
					{#each d.guru_belum_absen as g (g.id)}
						<div class="flex items-center justify-between gap-3 rounded-xl bg-amber-500/5 border border-amber-500/15 px-4 py-2.5">
							<a href={`/app/koordinator-guru/guru/${g.id}`} use:inertia class="text-sm font-medium text-neutral-900 dark:text-white hover:text-brand-600">{g.nama}</a>
							{#if g.no_wa}<a href={waLink(g.no_wa)} target="_blank" rel="noopener" class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-green-500/10 text-green-700 dark:text-green-400 text-xs font-semibold"><MessageCircle class="w-3.5 h-3.5" /> WA</a>{/if}
						</div>
					{/each}
				</div>
			{/if}
		</div>
	</div>
</AppLayout>
