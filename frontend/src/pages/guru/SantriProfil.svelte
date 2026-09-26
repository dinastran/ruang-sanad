<script lang="ts">
	import { inertia, router } from "@inertiajs/svelte";
	import { fly } from "svelte/transition";
	import AppLayout from "@layouts/AppLayout.svelte";
	import PenandaBadge from "@components/riayah/PenandaBadge.svelte";
	import CatatKontakDialog from "@components/riayah/CatatKontakDialog.svelte";
	import SapaWADialog from "@components/riayah/SapaWADialog.svelte";
	import { formatTanggal, hariLalu, mediaKontak } from "@lib/riayah";
	import type { Flash, User, RiayahSantriProfil, RiayahTimelineItem, RiayahWATemplate } from "@lib/types";
	import { ArrowLeft, BookOpen, NotebookPen, Trash2, CalendarCheck, MessageCircle, FileText, PhoneCall } from "lucide-svelte";

	interface Props {
		user?: User;
		profil: RiayahSantriProfil;
		can_write?: boolean;
		wa_templates?: RiayahWATemplate[];
		flash?: Flash;
		success?: string;
		error?: string;
	}

	let { user, profil, can_write = false, wa_templates = [], flash, success, error }: Props = $props();

	let s = $derived(profil.santri);
	let rekap = $derived(profil.rekap);

	type JenisFilter = "semua" | RiayahTimelineItem["jenis"];
	let jenisFilter = $state<JenisFilter>("semua");
	let timeline = $derived(jenisFilter === "semua" ? profil.timeline : profil.timeline.filter((t) => t.jenis === jenisFilter));

	let catatan = $state("");
	let saving = $state(false);
	let deleting = $state<string | null>(null);
	let kontakOpen = $state(false);
	let waOpen = $state(false);

	function simpanCatatan() {
		if (!catatan.trim() || saving) return;
		saving = true;
		router.post(`/app/guru/santri/${s.id}/catatan`, { catatan }, {
			preserveScroll: true,
			onSuccess: () => (catatan = ""),
			onFinish: () => (saving = false),
		});
	}

	function hapus(item: RiayahTimelineItem) {
		const kontak = item.jenis === "kontak";
		if (!confirm(kontak ? "Hapus log kontak ini?" : "Hapus catatan riayah ini?")) return;
		deleting = `${item.jenis}-${item.id}`;
		router.delete(`/app/guru/santri/${s.id}/${kontak ? "kontak" : "catatan"}/${item.id}`, { preserveScroll: true, onFinish: () => (deleting = null) });
	}

	function labelMedia(value?: string): string {
		return mediaKontak.find((m) => m.value === value)?.label ?? "";
	}

	const statusStyle: Record<string, string> = {
		hadir: "bg-green-500/10 text-green-700 dark:text-green-400",
		telat: "bg-sky-500/10 text-sky-700 dark:text-sky-400",
		izin: "bg-blue-500/10 text-blue-700 dark:text-blue-400",
		sakit: "bg-purple-500/10 text-purple-700 dark:text-purple-400",
		alpa: "bg-red-500/10 text-red-700 dark:text-red-400",
	};

	let rekapItems = $derived([
		{ label: "Hadir", value: rekap.hadir, cls: "text-green-700 dark:text-green-400" },
		{ label: "Telat", value: rekap.telat, cls: "text-sky-700 dark:text-sky-400" },
		{ label: "Izin", value: rekap.izin, cls: "text-blue-700 dark:text-blue-400" },
		{ label: "Sakit", value: rekap.sakit, cls: "text-purple-700 dark:text-purple-400" },
		{ label: "Alpa", value: rekap.alpa, cls: "text-red-700 dark:text-red-400" },
	]);

	let jenisChips = $derived([
		{ key: "semua" as JenisFilter, label: "Semua", count: profil.timeline.length },
		{ key: "pertemuan" as JenisFilter, label: "Pertemuan", count: profil.timeline.filter((t) => t.jenis === "pertemuan").length },
		{ key: "catatan" as JenisFilter, label: "Catatan", count: profil.timeline.filter((t) => t.jenis === "catatan").length },
		{ key: "kontak" as JenisFilter, label: "Sapaan", count: profil.timeline.filter((t) => t.jenis === "kontak").length },
	]);
</script>

<svelte:head><title>{s.nama} · Riayah</title></svelte:head>

<AppLayout {user} group="guru-riayah">
	<div class="pt-6 pb-8 border-b border-neutral-200/80 dark:border-white/[0.04]">
		<div class="max-w-5xl mx-auto px-4 sm:px-6">
			<a href="/app/guru/riayah" use:inertia class="inline-flex min-h-10 items-center gap-1.5 text-sm font-medium text-neutral-600 dark:text-neutral-400 hover:text-brand-700 dark:hover:text-brand-300">
				<ArrowLeft class="w-4 h-4" /> Riayah Santri
			</a>
			<h1 class="mt-2 text-2xl sm:text-3xl font-bold text-neutral-900 dark:text-white tracking-tight">{s.nama}</h1>
			<p class="mt-1 text-sm text-neutral-600 dark:text-neutral-400">
				{[s.id_mahasantri, s.nama_kelas && `${s.nama_kelas} · ${s.level}`, s.jadwal, s.guru_nama && `Guru: ${s.guru_nama}`].filter(Boolean).join(" · ") || "Belum ditempatkan di kelas"}
			</p>
			{#if s.penanda.length > 0}
				<div class="mt-3 flex flex-wrap gap-1.5">
					{#each s.penanda as p (p.kode)}<PenandaBadge penanda={p} showAlasan />{/each}
				</div>
			{/if}
		</div>
	</div>

	<div class="max-w-5xl mx-auto px-4 sm:px-6 py-8 space-y-6">
		{#if success || flash?.success}
			<div class="bg-green-500/10 border border-green-500/20 text-green-700 dark:text-green-400 rounded-2xl p-4 text-sm font-medium" in:fly={{ y: 10, duration: 200 }}>{success || flash?.success}</div>
		{/if}
		{#if error || flash?.error}
			<div class="bg-red-500/10 border border-red-500/20 text-red-600 dark:text-red-400 rounded-2xl p-4 text-sm font-medium" in:fly={{ y: 10, duration: 200 }}>{error || flash?.error}</div>
		{/if}

		<div class="grid gap-4 lg:grid-cols-[1fr_20rem]">
			<section class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-5" aria-labelledby="rekap-title">
				<h2 id="rekap-title" class="text-sm font-semibold text-neutral-900 dark:text-white">Kehadiran sepanjang belajar</h2>
				<p class="mt-0.5 text-xs text-neutral-500">
					{rekap.total} pertemuan tercatat{s.persen_hadir_30 !== null ? ` · ${Math.round(s.persen_hadir_30)}% hadir dalam 30 hari terakhir` : ""}
				</p>
				<dl class="mt-4 grid grid-cols-5 gap-2">
					{#each rekapItems as r (r.label)}
						<div class="rounded-xl bg-neutral-50 dark:bg-neutral-900/50 px-2 py-2.5 text-center">
							<dd class="font-mono text-xl font-bold {r.cls}">{r.value}</dd>
							<dt class="text-[11px] text-neutral-500">{r.label}</dt>
						</div>
					{/each}
				</dl>
				<div class="mt-4 flex items-start gap-2 rounded-xl border border-neutral-200 dark:border-neutral-800 px-3 py-2.5">
					<BookOpen class="mt-0.5 w-4 h-4 shrink-0 text-brand-600 dark:text-brand-400" />
					<div class="min-w-0">
						<p class="text-xs text-neutral-500">Batas materi terakhir</p>
						<p class="text-sm font-medium text-neutral-900 dark:text-white break-words">{s.batas_materi_terakhir || "Belum dicatat"}</p>
					</div>
				</div>
			</section>

			<section class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-5 text-sm" aria-labelledby="data-title">
				<h2 id="data-title" class="font-semibold text-neutral-900 dark:text-white">Data santri</h2>
				<dl class="mt-3 space-y-2">
					<div class="flex justify-between gap-3"><dt class="text-neutral-500">Mulai belajar</dt><dd class="text-right text-neutral-800 dark:text-neutral-200">{formatTanggal(profil.mulai_belajar)}</dd></div>
					<div class="flex justify-between gap-3"><dt class="text-neutral-500">Usia</dt><dd class="text-right text-neutral-800 dark:text-neutral-200">{profil.usia ? `${profil.usia} tahun` : "–"}</dd></div>
					<div class="flex justify-between gap-3"><dt class="text-neutral-500">Domisili</dt><dd class="text-right text-neutral-800 dark:text-neutral-200">{profil.domisili || "–"}</dd></div>
					<div class="flex justify-between gap-3"><dt class="text-neutral-500">No. WA</dt><dd class="text-right font-mono text-neutral-800 dark:text-neutral-200">{s.no_wa || "–"}</dd></div>
					<div class="flex justify-between gap-3"><dt class="text-neutral-500">Terakhir disapa</dt><dd class="text-right text-neutral-800 dark:text-neutral-200">{s.kontak_terakhir ? `${formatTanggal(s.kontak_terakhir)} (${hariLalu(s.kontak_terakhir)})` : "Belum pernah"}</dd></div>
				</dl>
				<div class="mt-4 grid gap-2">
					{#if can_write}
						<button type="button" onclick={() => (waOpen = true)} class="inline-flex min-h-10 w-full items-center justify-center gap-2 whitespace-nowrap rounded-xl bg-brand-600 px-4 text-sm font-semibold text-white hover:bg-brand-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-brand-400/60">
							<MessageCircle class="w-4 h-4" /> Sapa via WhatsApp
						</button>
						<button type="button" onclick={() => (kontakOpen = true)} class="inline-flex min-h-10 w-full items-center justify-center gap-2 whitespace-nowrap rounded-xl border border-neutral-200 px-4 text-sm font-medium text-neutral-700 hover:border-brand-400/40 focus:outline-none focus-visible:ring-2 focus-visible:ring-brand-400/50 dark:border-neutral-700 dark:text-neutral-300">
							<PhoneCall class="w-4 h-4" /> Catat sapaan lain
						</button>
					{/if}
					<a href={`/app/guru/santri/${s.id}/rapor`} use:inertia class="inline-flex min-h-10 w-full items-center justify-center gap-2 whitespace-nowrap rounded-xl border border-neutral-200 px-4 text-sm font-medium text-neutral-700 hover:border-brand-400/40 focus:outline-none focus-visible:ring-2 focus-visible:ring-brand-400/50 dark:border-neutral-700 dark:text-neutral-300">
						<FileText class="w-4 h-4" /> Rapor bulanan
					</a>
				</div>
			</section>
		</div>

		{#if can_write}
			<section class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-5" aria-labelledby="catatan-title">
				<h2 id="catatan-title" class="flex items-center gap-2 text-sm font-semibold text-neutral-900 dark:text-white"><NotebookPen class="w-4 h-4 text-brand-600 dark:text-brand-400" /> Tulis catatan riayah</h2>
				<label for="catatan" class="sr-only">Catatan riayah</label>
				<textarea
					id="catatan"
					bind:value={catatan}
					rows="3"
					placeholder="Mis. kondisi, kendala, perkembangan adab, atau hal yang perlu ditindaklanjuti"
					class="mt-3 w-full rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-925 px-3 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-brand-400/40"
				></textarea>
				<div class="mt-3 flex justify-end">
					<button type="button" onclick={simpanCatatan} disabled={!catatan.trim() || saving} class="inline-flex min-h-10 items-center gap-2 whitespace-nowrap rounded-xl bg-brand-600 px-4 text-sm font-semibold text-white hover:bg-brand-700 disabled:opacity-50 focus:outline-none focus-visible:ring-2 focus-visible:ring-brand-400/60">
						{saving ? "Menyimpan..." : "Simpan catatan"}
					</button>
				</div>
			</section>
		{/if}

		<section aria-labelledby="timeline-title">
			<div class="flex flex-wrap items-center justify-between gap-3">
				<h2 id="timeline-title" class="text-base font-semibold text-neutral-900 dark:text-white">Perjalanan santri</h2>
				<div class="flex flex-wrap gap-1.5" role="group" aria-label="Filter timeline">
					{#each jenisChips as chip (chip.key)}
						<button type="button" onclick={() => (jenisFilter = chip.key)} aria-pressed={jenisFilter === chip.key}
							class="min-h-9 whitespace-nowrap rounded-lg px-3 text-xs font-medium focus:outline-none focus-visible:ring-2 focus-visible:ring-brand-400/50 {jenisFilter === chip.key ? 'bg-brand-400/15 text-brand-800 dark:text-brand-200' : 'text-neutral-600 dark:text-neutral-400 hover:bg-neutral-100 dark:hover:bg-neutral-800'}">
							{chip.label} <span class="font-mono text-neutral-500">{chip.count}</span>
						</button>
					{/each}
				</div>
			</div>

			{#if timeline.length === 0}
				<div class="mt-4 rounded-2xl border border-dashed border-neutral-300 dark:border-neutral-700 p-10 text-center text-sm text-neutral-500">
					<CalendarCheck class="w-7 h-7 mx-auto mb-2 opacity-50" />
					Belum ada catatan perjalanan untuk ditampilkan.
				</div>
			{:else}
				<ol class="mt-4 border-l border-neutral-200 dark:border-neutral-800 ml-2 space-y-4">
					{#each timeline as item (`${item.jenis}-${item.id}`)}
						<li class="relative ml-5">
							<span class="absolute -left-[25px] top-4 h-2.5 w-2.5 rounded-full ring-4 ring-white dark:ring-neutral-950 {item.jenis === 'catatan' ? 'bg-brand-500' : item.jenis === 'kontak' ? 'bg-green-500' : item.status === 'alpa' ? 'bg-red-500' : 'bg-neutral-400'}" aria-hidden="true"></span>
							<article class="rounded-2xl border bg-white dark:bg-neutral-925/50 p-4 {item.jenis === 'catatan' ? 'border-brand-400/30' : item.jenis === 'kontak' ? 'border-green-500/25' : 'border-neutral-200/80 dark:border-white/[0.06]'}">
								<header class="flex flex-wrap items-center justify-between gap-2">
									<div class="flex flex-wrap items-center gap-2">
										<time class="text-xs font-medium text-neutral-500" datetime={item.tanggal}>{formatTanggal(item.tanggal)}{item.waktu ? ` · ${item.waktu}` : ""}</time>
										<h3 class="text-sm font-semibold text-neutral-900 dark:text-white">{item.judul}</h3>
										{#if item.status}<span class="rounded-full px-2 py-0.5 text-[11px] font-semibold capitalize {statusStyle[item.status] ?? 'bg-neutral-500/10 text-neutral-600'}">{item.status}</span>{/if}
									</div>
									{#if item.bisa_hapus}
										<button type="button" onclick={() => hapus(item)} disabled={deleting === `${item.jenis}-${item.id}`} class="inline-flex min-h-9 items-center gap-1 rounded-lg px-2 text-xs text-neutral-500 hover:text-red-600 disabled:opacity-50" aria-label={item.jenis === "kontak" ? "Hapus log kontak" : "Hapus catatan"}>
											<Trash2 class="w-3.5 h-3.5" /> Hapus
										</button>
									{/if}
								</header>
								{#if item.jenis === "pertemuan"}
									<dl class="mt-2 grid gap-1 text-sm sm:grid-cols-2">
										{#if item.materi}<div><dt class="inline text-neutral-500">Materi kelas: </dt><dd class="inline text-neutral-800 dark:text-neutral-200">{item.materi}</dd></div>{/if}
										{#if item.batas_materi}<div><dt class="inline text-neutral-500">Batas materi: </dt><dd class="inline text-neutral-800 dark:text-neutral-200">{item.batas_materi}</dd></div>{/if}
									</dl>
									{#if item.isi}<p class="mt-2 rounded-lg bg-neutral-50 dark:bg-neutral-900/50 px-3 py-2 text-sm text-neutral-700 dark:text-neutral-300 whitespace-pre-line">{item.isi}</p>{/if}
								{:else}
									{#if item.isi}<p class="mt-2 text-sm text-neutral-800 dark:text-neutral-200 whitespace-pre-line">{item.isi}</p>{/if}
									{#if item.penulis || item.media}<p class="mt-2 text-xs text-neutral-500">{[item.jenis === "kontak" ? labelMedia(item.media) : "", item.penulis && `oleh ${item.penulis}`].filter(Boolean).join(" · ")}</p>{/if}
								{/if}
							</article>
						</li>
					{/each}
				</ol>
			{/if}
		</section>
	</div>

	{#if kontakOpen}
		<CatatKontakDialog santri={s} onclose={() => (kontakOpen = false)} />
	{/if}
	{#if waOpen}
		<SapaWADialog santri={s} templates={wa_templates} pengirim={user?.name ?? ""} onclose={() => (waOpen = false)} />
	{/if}
</AppLayout>
