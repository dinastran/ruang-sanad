<script lang="ts">
	import { inertia } from "@inertiajs/svelte";
	import AppLayout from "@layouts/AppLayout.svelte";
	import SapaWADialog from "@components/riayah/SapaWADialog.svelte";
	import CatatKontakDialog from "@components/riayah/CatatKontakDialog.svelte";
	import { hariLalu, mediaKontak } from "@lib/riayah";
	import type { Flash, RiayahSantriItem, RiayahWATemplate, User } from "@lib/types";
	import { PhoneCall, Search, MessageCircle, NotebookPen, ChevronRight, CheckCircle2 } from "lucide-svelte";

	interface Props {
		user?: User;
		perlu?: RiayahSantriItem[];
		sudah?: RiayahSantriItem[];
		wa_templates?: RiayahWATemplate[];
		flash?: Flash;
	}

	let { user, perlu = [], sudah = [], wa_templates = [], flash }: Props = $props();

	type Tab = "perlu" | "sudah";
	let tab = $state<Tab>("perlu");
	let search = $state("");
	let waSantri = $state<RiayahSantriItem | null>(null);
	let kontakSantri = $state<RiayahSantriItem | null>(null);

	const KEMBALI = "/app/kelas/tindak-lanjut-alpa" as const;

	const daftar = $derived.by(() => {
		const q = search.trim().toLowerCase();
		const sumber = tab === "perlu" ? perlu : sudah;
		return sumber.filter((s) => !q || `${s.nama} ${s.id_mahasantri} ${s.nama_kelas} ${s.guru_nama}`.toLowerCase().includes(q));
	});

	const DOT: Record<string, { kelas: string; label: string }> = {
		hadir: { kelas: "bg-emerald-500", label: "Hadir" },
		telat: { kelas: "bg-emerald-300", label: "Telat" },
		izin: { kelas: "bg-sky-500", label: "Izin" },
		sakit: { kelas: "bg-amber-400", label: "Sakit" },
		alpa: { kelas: "bg-red-500", label: "Alpa" },
	};

	function labelMedia(value: string): string {
		return mediaKontak.find((m) => m.value === value)?.label ?? value;
	}

	function persen(v: number | null): string {
		return v === null ? "–" : `${Math.round(v)}%`;
	}
</script>

<svelte:head><title>Tindak Lanjut Alpa</title></svelte:head>

<AppLayout {user} group="tindak-lanjut-alpa">
	<div class="pt-8 pb-8 border-b border-neutral-200/80 dark:border-white/[0.04]">
		<div class="max-w-4xl mx-auto px-4 sm:px-6">
			<h1 class="text-2xl sm:text-3xl font-bold tracking-tight text-neutral-900 dark:text-white">Tindak Lanjut Alpa</h1>
			<p class="mt-2 text-neutral-600 dark:text-neutral-400">Santri yang alpa 2 pertemuan atau lebih berturut-turut. Hubungi santri lalu catat kontaknya; penanda hilang sampai santri alpa lagi.</p>
		</div>
	</div>

	<div class="max-w-4xl mx-auto px-4 sm:px-6 py-8 space-y-4">
		{#if flash?.success}<div role="status" class="rounded-xl border border-green-500/20 bg-green-500/10 p-4 text-sm font-medium text-green-700 dark:text-green-400">{flash.success}</div>{/if}
		{#if flash?.error}<div role="alert" class="rounded-xl border border-red-500/20 bg-red-500/10 p-4 text-sm font-medium text-red-700 dark:text-red-400">{flash.error}</div>{/if}

		<div class="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
			<div class="flex gap-1 border-b border-neutral-200/80 dark:border-white/[0.06]" role="tablist">
				{#each [{ key: "perlu" as Tab, label: "Perlu dihubungi", n: perlu.length }, { key: "sudah" as Tab, label: "Sudah dihubungi", n: sudah.length }] as t (t.key)}
					<button type="button" role="tab" aria-selected={tab === t.key} onclick={() => (tab = t.key)} class="-mb-px inline-flex items-center gap-2 border-b-2 px-3 py-2.5 text-sm font-semibold transition-colors {tab === t.key ? 'border-brand-500 text-brand-700 dark:text-brand-300' : 'border-transparent text-neutral-500 hover:text-neutral-800 dark:hover:text-neutral-200'}">
						{t.label}
						<span class="rounded-full px-1.5 text-xs tabular-nums {t.key === 'perlu' && t.n > 0 ? 'bg-red-500/10 text-red-700 dark:text-red-400' : 'bg-neutral-100 text-neutral-600 dark:bg-neutral-800 dark:text-neutral-400'}">{t.n}</span>
					</button>
				{/each}
			</div>
			<label class="relative sm:w-64">
				<Search class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-neutral-400" />
				<input bind:value={search} placeholder="Cari santri, kelas, guru" aria-label="Cari santri" class="w-full rounded-xl border border-neutral-200 bg-white py-2 pl-9 pr-3 text-sm outline-none focus:border-brand-400 dark:border-neutral-800 dark:bg-neutral-925" />
			</label>
		</div>

		{#if tab === "sudah"}<p class="text-xs text-neutral-500">Santri yang dihubungi dalam 14 hari terakhir dan belum alpa lagi sesudahnya.</p>{/if}

		<div class="space-y-3">
			{#each daftar as s (s.id)}
				{@const tl = s.tindak_lanjut_alpa}
				<article class="rounded-2xl border bg-white p-4 sm:p-5 dark:bg-neutral-925/50 {tab === 'perlu' ? 'border-red-500/25' : 'border-neutral-200/80 dark:border-white/[0.06]'}">
					<div class="flex flex-wrap items-start justify-between gap-3">
						<div class="min-w-0">
							<div class="flex flex-wrap items-center gap-2">
								<h2 class="font-semibold text-neutral-900 dark:text-white">{s.nama}</h2>
								{#if tl}
									<span class="rounded-full bg-red-500/10 px-2 py-0.5 text-xs font-semibold text-red-700 dark:text-red-400">Alpa {tl.alpa_beruntun}× berturut-turut</span>
								{/if}
							</div>
							<p class="mt-1 text-xs text-neutral-500">{[s.id_mahasantri, s.nama_kelas, s.jadwal, s.guru_nama && `Guru: ${s.guru_nama}`].filter(Boolean).join(" · ")}</p>
							<p class="mt-1 text-xs text-neutral-500">
								{#if tl}Alpa terakhir {hariLalu(tl.alpa_terakhir)}{/if} · Hadir 30 hari {persen(s.persen_hadir_30)}
							</p>
							{#if tab === "sudah" && tl}
								<p class="mt-2 inline-flex items-center gap-1.5 text-xs font-medium text-emerald-700 dark:text-emerald-400"><CheckCircle2 class="h-3.5 w-3.5" /> Dihubungi {hariLalu(tl.kontak_tanggal)} via {labelMedia(tl.kontak_media)}{tl.kontak_oleh ? ` oleh ${tl.kontak_oleh}` : ""}</p>
							{/if}
						</div>
						{#if tl}
							<div class="flex items-center gap-1.5" aria-label="Kehadiran {tl.absen_terakhir.length} pertemuan terakhir, lama ke baru">
								{#each [...tl.absen_terakhir].reverse() as status, i (i)}
									<span class="h-3.5 w-3.5 rounded-full {DOT[status]?.kelas ?? 'bg-neutral-300'}" title={DOT[status]?.label ?? status}></span>
								{/each}
							</div>
						{/if}
					</div>
					<div class="mt-3 flex flex-wrap gap-2">
						{#if s.no_wa}
							<button type="button" onclick={() => (waSantri = s)} class="inline-flex min-h-9 items-center gap-1.5 rounded-lg bg-brand-600 px-3 text-xs font-semibold text-white hover:bg-brand-700"><MessageCircle class="h-4 w-4" /> Sapa WA</button>
						{:else}
							<span class="inline-flex min-h-9 items-center gap-1.5 rounded-lg border border-dashed border-neutral-300 px-3 text-xs text-neutral-400 dark:border-neutral-700" title="Nomor WA santri belum diisi"><MessageCircle class="h-4 w-4" /> Tanpa no. WA</span>
						{/if}
						<button type="button" onclick={() => (kontakSantri = s)} class="inline-flex min-h-9 items-center gap-1.5 rounded-lg border border-neutral-200 px-3 text-xs font-semibold text-neutral-700 hover:border-brand-400/40 hover:text-brand-700 dark:border-neutral-700 dark:text-neutral-300 dark:hover:text-brand-300"><NotebookPen class="h-4 w-4" /> Catat kontak</button>
						<a href={`/app/guru/santri/${s.id}`} use:inertia class="inline-flex min-h-9 items-center gap-1 rounded-lg px-3 text-xs font-semibold text-neutral-600 hover:bg-neutral-100 dark:text-neutral-300 dark:hover:bg-neutral-800">Profil santri <ChevronRight class="h-4 w-4" /></a>
					</div>
				</article>
			{:else}
				<div class="rounded-2xl border border-dashed border-neutral-300 p-12 text-center text-sm text-neutral-500 dark:border-neutral-700">
					<PhoneCall class="mx-auto mb-2 h-8 w-8 opacity-40" />
					{#if search}Tidak ada santri yang cocok dengan pencarian.{:else if tab === "perlu"}Tidak ada santri yang perlu dihubungi. Alhamdulillah.{:else}Belum ada santri yang dihubungi dalam 14 hari terakhir.{/if}
				</div>
			{/each}
		</div>
	</div>

	{#if kontakSantri}
		<CatatKontakDialog santri={kontakSantri} kembali={KEMBALI} onclose={() => (kontakSantri = null)} />
	{/if}
	{#if waSantri}
		<SapaWADialog santri={waSantri} templates={wa_templates} pengirim={user?.name ?? ""} kembali={KEMBALI} onclose={() => (waSantri = null)} />
	{/if}
</AppLayout>
