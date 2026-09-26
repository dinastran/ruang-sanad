<script lang="ts">
	import { inertia, router } from "@inertiajs/svelte";
	import { fly } from "svelte/transition";
	import AppLayout from "@layouts/AppLayout.svelte";
	import PenandaBadge from "@components/riayah/PenandaBadge.svelte";
	import SapaWADialog from "@components/riayah/SapaWADialog.svelte";
	import type { AppNotification, Flash, User, TilawahStatus, RiayahRingkasan, RiayahSantriItem, RiayahWATemplate, GuruBeranda, GuruSlotKelas } from "@lib/types";
	import { formatTanggal, labelPeriode } from "@lib/riayah";
	import { CalendarClock, ClipboardCheck, HeartHandshake, FileText, CheckCircle, AlertCircle, BookMarked, BellRing, Award, Users, CalendarDays, MessageCircle, ChevronRight, Flame } from "lucide-svelte";

	interface Props {
		user?: User;
		beranda?: GuruBeranda | null;
		tilawah?: TilawahStatus;
		notifications?: AppNotification[];
		riayah?: RiayahRingkasan | null;
		riayah_teratas?: RiayahSantriItem[];
		wa_templates?: RiayahWATemplate[];
		flash?: Flash;
		success?: string;
		error?: string;
	}

	let { user, beranda = null, tilawah, notifications = [], riayah = null, riayah_teratas = [], wa_templates = [], flash, success, error }: Props = $props();

	// Hanya guru yang memulai/mengabsen pertemuan dari dashboard.
	let canEdit = $derived(user?.role === "guru");
	let hariIni = $derived(beranda?.hari_ini ?? []);
	let tertunda = $derived(beranda?.tertunda ?? []);
	let takTerbaca = $derived(beranda?.jadwal_tak_terbaca ?? []);
	let kinerja = $derived(beranda?.kinerja ?? null);
	let agenda = $derived(beranda?.agenda ?? []);
	let tugasHariIni = $derived(hariIni.filter((s) => s.status !== "dibadalkan"));
	let raporBelum = $derived(riayah ? riayah.total_santri - riayah.rapor_terkirim : 0);
	let unread = $derived(notifications.filter((n) => !n.read).length);

	let tilawahBusy = $state(false);
	let notificationBusy = $state<number | null>(null);
	let waSantri = $state<RiayahSantriItem | null>(null);

	let tanggalPanjang = $derived.by(() => {
		const d = new Date((beranda?.tanggal ?? "") + "T00:00:00");
		return Number.isNaN(d.getTime()) ? "" : d.toLocaleDateString("id-ID", { weekday: "long", day: "numeric", month: "long", year: "numeric" });
	});

	function toggleTilawah() {
		if (!canEdit || !tilawah) return;
		tilawahBusy = true;
		if (tilawah.sudah_hari_ini) router.delete("/app/guru/tilawah", { preserveScroll: true, onFinish: () => (tilawahBusy = false) });
		else router.post("/app/guru/tilawah", {}, { preserveScroll: true, onFinish: () => (tilawahBusy = false) });
	}

	function markNotificationRead(id: number) {
		notificationBusy = id;
		router.put(`/app/notifications/${id}/read`, {}, { preserveScroll: true, onFinish: () => (notificationBusy = null) });
	}

	function aksiSlot(s: GuruSlotKelas): { href: string; label: string } | null {
		if (s.status === "dibadalkan") return null;
		if (s.status === "berlangsung" && s.pertemuan_id && canEdit) return { href: `/app/guru/kelas/${s.kelas_id}/pertemuan/${s.pertemuan_id}/selesai`, label: "Selesaikan absen" };
		if (s.status === "belum" && canEdit) return { href: `/app/guru/kelas/${s.kelas_id}/pertemuan/mulai`, label: "Mulai" };
		return { href: `/app/guru/kelas/${s.kelas_id}`, label: "Lihat" };
	}

	const statusLabel: Record<string, { label: string; cls: string }> = {
		belum: { label: "Belum mulai", cls: "bg-neutral-500/10 text-neutral-700 dark:text-neutral-300" },
		berlangsung: { label: "Berlangsung", cls: "bg-sky-500/10 text-sky-700 dark:text-sky-300" },
		selesai: { label: "Selesai", cls: "bg-green-500/10 text-green-700 dark:text-green-400" },
		dibadalkan: { label: "Dibadalkan", cls: "bg-neutral-500/10 text-neutral-500" },
	};

	const agendaLabel: Record<string, string> = { pembinaan: "Pembinaan", rapat: "Rapat guru", kalam: "Kalam Bersanad" };

	let kartu = $derived([
		{ label: "Kelas hari ini", value: tugasHariIni.length, href: "#hari-ini", icon: CalendarClock, tone: "text-brand-600 dark:text-brand-400 bg-brand-400/10", alert: false },
		{ label: "Absensi tertunda", value: tertunda.length, href: "#tertunda", icon: ClipboardCheck, tone: "text-red-600 dark:text-red-400 bg-red-500/10", alert: tertunda.length > 0 },
		{ label: "Santri perlu disapa", value: riayah?.perlu_perhatian ?? 0, href: "/app/guru/riayah", icon: HeartHandshake, tone: "text-orange-600 dark:text-orange-400 bg-orange-500/10", alert: (riayah?.perlu_perhatian ?? 0) > 0 },
		{ label: riayah?.rapor_periode ? `Rapor ${labelPeriode(riayah.rapor_periode).split(" ")[0]} belum dikirim` : "Rapor belum dikirim", value: raporBelum, href: "/app/guru/riayah", icon: FileText, tone: "text-sky-600 dark:text-sky-400 bg-sky-500/10", alert: false },
	]);
</script>

<svelte:head><title>Dashboard Guru</title></svelte:head>

<AppLayout {user} group="guru-dashboard">
	<div class="pt-8 pb-8 border-b border-neutral-200/80 dark:border-white/[0.04]">
		<div class="max-w-6xl mx-auto px-4 sm:px-6">
			<p class="text-sm text-neutral-500 dark:text-neutral-400">{tanggalPanjang}</p>
			<h1 class="mt-1 text-2xl sm:text-3xl font-bold text-neutral-900 dark:text-white tracking-tight">Assalamu'alaikum, {user?.name ?? "Ustadz"}</h1>
		</div>
	</div>

	<div class="max-w-6xl mx-auto px-4 sm:px-6 py-8 space-y-6">
		{#if success || flash?.success}
			<div class="bg-green-500/10 border border-green-500/20 text-green-700 dark:text-green-400 rounded-2xl p-4 text-sm font-medium" in:fly={{ y: 10, duration: 200 }}>{success || flash?.success}</div>
		{/if}
		{#if error || flash?.error}
			<div class="bg-red-500/10 border border-red-500/20 text-red-600 dark:text-red-400 rounded-2xl p-4 text-sm font-medium" in:fly={{ y: 10, duration: 200 }}>{error || flash?.error}</div>
		{/if}

		<div class="grid grid-cols-2 lg:grid-cols-4 gap-3">
			{#snippet isiKartu(k: (typeof kartu)[number])}
				<div class="flex items-center gap-2.5">
					<span class="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl {k.tone}"><k.icon class="h-4 w-4" /></span>
					<span class="text-2xl font-bold font-mono text-neutral-900 dark:text-white">{k.value}</span>
				</div>
				<p class="mt-2 text-xs font-medium text-neutral-600 dark:text-neutral-400">{k.label}</p>
			{/snippet}
			{#each kartu as k (k.label)}
				{@const cls = `rounded-2xl border bg-white dark:bg-neutral-925/50 p-4 transition-colors hover:border-brand-400/30 focus:outline-none focus-visible:ring-2 focus-visible:ring-brand-400/50 ${k.alert ? "border-red-500/25" : "border-neutral-200/80 dark:border-white/[0.06]"}`}
				{#if k.href.startsWith("#")}
					<a href={k.href} class={cls}>{@render isiKartu(k)}</a>
				{:else}
					<a href={k.href} use:inertia class={cls}>{@render isiKartu(k)}</a>
				{/if}
			{/each}
		</div>

		<div class="grid gap-6 lg:grid-cols-[1fr_20rem]">
			<div class="space-y-6 min-w-0">
				<section id="hari-ini" class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 overflow-hidden scroll-mt-20" aria-labelledby="hari-ini-title">
					<header class="flex items-center gap-2.5 border-b border-neutral-200/80 dark:border-white/[0.05] px-5 py-4">
						<CalendarClock class="h-5 w-5 text-brand-600 dark:text-brand-400" />
						<h2 id="hari-ini-title" class="font-semibold text-neutral-900 dark:text-white">Mengajar hari ini</h2>
					</header>
					{#if hariIni.length === 0}
						<p class="px-5 py-8 text-center text-sm text-neutral-500">Tidak ada kelas terjadwal hari ini.</p>
					{:else}
						<ul class="divide-y divide-neutral-100 dark:divide-neutral-800">
							{#each hariIni as s (`${s.kelas_id}-${s.tanggal}`)}
								{@const aksi = aksiSlot(s)}
								<li class="flex flex-wrap items-center gap-3 px-5 py-3.5 {s.status === 'dibadalkan' ? 'opacity-60' : ''}">
									<span class="w-12 shrink-0 font-mono text-sm font-semibold text-neutral-900 dark:text-white">{s.jam || "–"}</span>
									<div class="min-w-0 flex-1">
										<p class="truncate text-sm font-medium text-neutral-900 dark:text-white">{s.nama_kelas}</p>
										<p class="text-xs text-neutral-500">{s.level} · {s.jumlah_santri} santri{s.keterangan ? ` · ${s.keterangan}` : ""}</p>
									</div>
									<span class="rounded-full px-2 py-0.5 text-[11px] font-semibold {statusLabel[s.status].cls}">{statusLabel[s.status].label}</span>
									{#if aksi}
										<a href={aksi.href} use:inertia class="inline-flex min-h-10 items-center whitespace-nowrap rounded-xl px-3.5 text-sm font-semibold focus:outline-none focus-visible:ring-2 focus-visible:ring-brand-400/60 {aksi.label === 'Lihat' ? 'border border-neutral-200 text-neutral-700 hover:border-brand-400/40 dark:border-neutral-700 dark:text-neutral-300' : 'bg-brand-600 text-white hover:bg-brand-700'}">{aksi.label}</a>
									{/if}
								</li>
							{/each}
						</ul>
					{/if}
				</section>

				{#if tertunda.length > 0}
					<section id="tertunda" class="rounded-2xl border border-red-500/20 bg-red-500/5 overflow-hidden scroll-mt-20" aria-labelledby="tertunda-title">
						<header class="border-b border-red-500/15 px-5 py-4">
							<h2 id="tertunda-title" class="flex items-center gap-2.5 font-semibold text-red-800 dark:text-red-300"><AlertCircle class="h-5 w-5" /> Absensi tertunda <span class="text-sm font-normal">({tertunda.length})</span></h2>
							<p class="mt-1 text-xs text-red-700/80 dark:text-red-300/80">Jadwal 7 hari terakhir yang belum ada pertemuannya. Pertemuan yang diisi sekarang tercatat bertanggal hari ini.</p>
						</header>
						<ul class="divide-y divide-red-500/10">
							{#each tertunda as s (`${s.kelas_id}-${s.tanggal}`)}
								{@const aksi = aksiSlot(s)}
								<li class="flex flex-wrap items-center gap-3 px-5 py-3.5">
									<span class="w-24 shrink-0 text-xs font-medium text-red-700 dark:text-red-300">{formatTanggal(s.tanggal)}</span>
									<div class="min-w-0 flex-1">
										<p class="truncate text-sm font-medium text-neutral-900 dark:text-white">{s.nama_kelas}</p>
										<p class="text-xs text-neutral-500">{s.jam || "–"}{s.keterangan ? ` · ${s.keterangan}` : ""}</p>
									</div>
									{#if aksi}<a href={aksi.href} use:inertia class="inline-flex min-h-10 items-center whitespace-nowrap rounded-xl border border-red-500/30 px-3.5 text-sm font-semibold text-red-700 hover:bg-red-500/10 dark:text-red-300">{aksi.label === "Mulai" ? "Isi absensi" : aksi.label}</a>{/if}
								</li>
							{/each}
						</ul>
					</section>
				{/if}

				{#if takTerbaca.length > 0}
					<div class="rounded-2xl border border-amber-500/25 bg-amber-500/5 px-5 py-4 text-sm text-amber-900 dark:text-amber-200">
						<p class="font-medium">Jadwal kelas berikut tidak bisa dibaca otomatis, sehingga tidak muncul di "Mengajar hari ini":</p>
						<ul class="mt-2 space-y-1">
							{#each takTerbaca as k (k.kelas_id)}<li><a href={`/app/guru/kelas/${k.kelas_id}`} use:inertia class="underline">{k.nama_kelas}</a> — "{k.jadwal}"</li>{/each}
						</ul>
						<p class="mt-2 text-xs">Minta admin kelas memilih jadwal dengan nama hari (mis. "Senin, jam 20.30 WIB").</p>
					</div>
				{/if}

				{#if riayah}
					<section class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 overflow-hidden" aria-labelledby="riayah-title">
						<header class="flex items-center justify-between gap-3 border-b border-neutral-200/80 dark:border-white/[0.05] px-5 py-4">
							<h2 id="riayah-title" class="flex items-center gap-2.5 font-semibold text-neutral-900 dark:text-white"><HeartHandshake class="h-5 w-5 text-brand-600 dark:text-brand-400" /> Riayah pekan ini</h2>
							<a href="/app/guru/riayah" use:inertia class="inline-flex min-h-9 items-center gap-1 text-sm font-medium text-brand-700 hover:underline dark:text-brand-300">Semua santri <ChevronRight class="h-4 w-4" /></a>
						</header>
						{#if riayah_teratas.length === 0}
							<p class="px-5 py-8 text-center text-sm text-neutral-500">Tidak ada santri yang ditandai. Tetap sapa santri secara berkala.</p>
						{:else}
							<ul class="divide-y divide-neutral-100 dark:divide-neutral-800">
								{#each riayah_teratas as s (s.id)}
									<li class="flex items-start gap-3 px-5 py-3.5">
										<a href={`/app/guru/santri/${s.id}`} use:inertia class="min-w-0 flex-1">
											<p class="text-sm font-medium text-neutral-900 dark:text-white">{s.nama} <span class="font-normal text-neutral-500">· {s.nama_kelas}</span></p>
											<div class="mt-1.5 flex flex-wrap gap-1.5">{#each s.penanda as p (p.kode)}<PenandaBadge penanda={p} showAlasan />{/each}</div>
										</a>
										{#if canEdit && s.no_wa}
											<button type="button" onclick={() => (waSantri = s)} class="inline-flex min-h-10 shrink-0 items-center gap-1.5 whitespace-nowrap rounded-xl bg-brand-600 px-3 text-xs font-semibold text-white hover:bg-brand-700" aria-label={`Sapa ${s.nama} via WhatsApp`}><MessageCircle class="h-3.5 w-3.5" /><span class="hidden sm:inline">WA</span></button>
										{/if}
									</li>
								{/each}
							</ul>
						{/if}
					</section>
				{/if}
			</div>

			<aside class="space-y-6">
				{#if kinerja || tilawah}
					<section class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-5" aria-labelledby="kinerja-title">
						<h2 id="kinerja-title" class="font-semibold text-neutral-900 dark:text-white">Kinerja saya</h2>
						{#if tilawah}
							<div class="mt-4 flex items-center justify-between gap-3">
								<div class="flex items-center gap-2.5">
									<span class="flex h-9 w-9 items-center justify-center rounded-xl bg-secondary-500/10"><BookMarked class="h-4 w-4 text-secondary-600 dark:text-secondary-400" /></span>
									<div>
										<p class="text-sm font-medium text-neutral-900 dark:text-white">Tilawah harian</p>
										<p class="flex items-center gap-1 text-xs text-neutral-500">
											{#if kinerja && kinerja.tilawah_streak > 0}<Flame class="h-3 w-3 text-orange-500" />{kinerja.tilawah_streak} hari berturut · {/if}{tilawah.bulan_ini} hari bulan ini
										</p>
									</div>
								</div>
								{#if canEdit}
									<button type="button" onclick={toggleTilawah} disabled={tilawahBusy} class="inline-flex min-h-10 items-center gap-1.5 whitespace-nowrap rounded-xl px-3 text-xs font-semibold disabled:opacity-50 {tilawah.sudah_hari_ini ? 'bg-green-500/10 text-green-700 dark:text-green-400' : 'bg-secondary-600 text-white hover:bg-secondary-700'}">
										<CheckCircle class="h-3.5 w-3.5" />{tilawah.sudah_hari_ini ? "Sudah" : "Tandai"}
									</button>
								{/if}
							</div>
						{/if}
						{#if kinerja}
							<a href="/app/guru/tsi" use:inertia class="mt-4 -mx-2 flex items-center justify-between gap-3 rounded-xl px-2 py-1.5 text-sm hover:bg-neutral-50 dark:hover:bg-white/[0.03]">
								<span class="flex items-center gap-2 text-neutral-600 dark:text-neutral-400"><Award class="h-4 w-4" /> Nilai TSI{kinerja.tsi_bulan ? ` (${labelPeriode(kinerja.tsi_bulan)})` : ""}</span>
								<span class="font-semibold text-neutral-900 dark:text-white">{kinerja.tsi_total !== null ? `${kinerja.tsi_total.toFixed(1)}${kinerja.tsi_predikat ? ` · ${kinerja.tsi_predikat}` : ""}` : "Belum final"}</span>
							</a>
							<dl class="mt-3 space-y-3 text-sm">
								<div class="flex items-center justify-between gap-3">
									<dt class="flex items-center gap-2 text-neutral-600 dark:text-neutral-400"><Users class="h-4 w-4" /> Pembinaan bulan ini</dt>
									<dd class="font-mono font-semibold text-neutral-900 dark:text-white">{kinerja.pembinaan_hadir}/{kinerja.pembinaan_total}</dd>
								</div>
								<div class="flex items-center justify-between gap-3">
									<dt class="flex items-center gap-2 text-neutral-600 dark:text-neutral-400"><Users class="h-4 w-4" /> Hadir rapat bulan ini</dt>
									<dd class="font-mono font-semibold text-neutral-900 dark:text-white">{kinerja.rapat_hadir}</dd>
								</div>
							</dl>
						{/if}
					</section>
				{/if}

				<section class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 overflow-hidden" aria-labelledby="info-title">
					<header class="flex items-center justify-between gap-3 border-b border-neutral-200/80 dark:border-white/[0.05] px-5 py-4">
						<h2 id="info-title" class="flex items-center gap-2 font-semibold text-neutral-900 dark:text-white"><BellRing class="h-4 w-4 text-brand-600 dark:text-brand-400" /> Info koordinator</h2>
						{#if unread > 0}<span class="rounded-full bg-brand-600 px-2 py-0.5 text-[10px] font-bold text-white">{unread} baru</span>{/if}
					</header>
					{#if notifications.length === 0 && agenda.length === 0}
						<p class="px-5 py-6 text-center text-sm text-neutral-500">Belum ada pengingat atau agenda.</p>
					{/if}
					{#if notifications.length > 0}
						<ul class="divide-y divide-neutral-100 dark:divide-neutral-800">
							{#each notifications as item (item.id)}
								<li class="px-5 py-3.5 {item.read ? 'opacity-65' : ''}">
									<p class="text-sm font-semibold text-neutral-900 dark:text-white">{item.title}</p>
									<p class="mt-0.5 text-sm text-neutral-600 dark:text-neutral-300">{item.message}</p>
									<div class="mt-2 flex flex-wrap items-center gap-2">
										<span class="text-xs text-neutral-500">{item.created_at}</span>
										{#if item.action_url}<a href={item.action_url} use:inertia class="inline-flex min-h-9 items-center rounded-lg px-2 text-xs font-semibold text-brand-700 hover:bg-brand-400/10 dark:text-brand-300">Buka</a>{/if}
										{#if !item.read}<button type="button" onclick={() => markNotificationRead(item.id)} disabled={notificationBusy === item.id} class="min-h-9 rounded-lg px-2 text-xs font-semibold text-neutral-600 hover:bg-neutral-100 disabled:opacity-50 dark:text-neutral-300 dark:hover:bg-neutral-800">{notificationBusy === item.id ? "Memproses..." : "Tandai dibaca"}</button>{/if}
									</div>
								</li>
							{/each}
						</ul>
					{/if}
					{#if agenda.length > 0}
						<div class="border-t border-neutral-100 px-5 py-4 dark:border-neutral-800">
							<p class="flex items-center gap-2 text-xs font-semibold uppercase tracking-wide text-neutral-500"><CalendarDays class="h-3.5 w-3.5" /> Agenda terdekat</p>
							<ul class="mt-2 space-y-2">
								{#each agenda as a, i (i)}
									<li class="text-sm">
										<span class="font-medium text-neutral-900 dark:text-white">{agendaLabel[a.jenis]}</span>
										<span class="text-neutral-500"> · {formatTanggal(a.tanggal)}</span>
										{#if a.judul}<p class="text-xs text-neutral-600 dark:text-neutral-400">{a.judul}</p>{/if}
									</li>
								{/each}
							</ul>
						</div>
					{/if}
				</section>
			</aside>
		</div>
	</div>

	{#if waSantri}
		<SapaWADialog santri={waSantri} templates={wa_templates} pengirim={user?.name ?? ""} kembali="/app/guru" onclose={() => (waSantri = null)} />
	{/if}
</AppLayout>
