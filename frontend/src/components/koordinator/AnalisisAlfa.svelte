<script lang="ts">
	import { inertia } from "@inertiajs/svelte";
	import CatatRiayahGuruDialog from "./CatatRiayahGuruDialog.svelte";
	import { waLink } from "@lib/riayah";
	import type { AnalisisKehadiran, AnalisisKehadiranGuru, FilterAlfa, Kehadiran } from "@lib/types";
	import { Search, MessageCircle, NotebookPen, History, UserCheck } from "lucide-svelte";

	interface Props {
		analisis: AnalisisKehadiran;
		/** "pembinaan" atau "rapat" — dipakai untuk teks pesan & label. */
		kegiatan: "pembinaan" | "rapat";
		filter?: FilterAlfa;
	}

	let { analisis, kegiatan, filter = $bindable("merah") }: Props = $props();

	const namaKegiatan = $derived(kegiatan === "pembinaan" ? "pembinaan guru" : "rapat guru");
	const DOT: Record<Kehadiran | "", { kelas: string; label: string }> = {
		hadir: { kelas: "bg-emerald-500", label: "Hadir" },
		izin: { kelas: "bg-sky-500", label: "Izin" },
		sakit: { kelas: "bg-amber-400", label: "Sakit" },
		alfa: { kelas: "bg-red-500", label: "Alfa" },
		"": { kelas: "border border-dashed border-neutral-300 dark:border-neutral-600", label: "Tidak tercatat" },
	};

	let search = $state("");
	let riayahGuru = $state<AnalisisKehadiranGuru | null>(null);

	// Strip ditampilkan dari sesi terlama ke terbaru supaya rangkaian alfa terbaca kiri → kanan.
	const sesiUrut = $derived([...analisis.sesi].reverse());

	const chips = $derived([
		{ key: "merah" as FilterAlfa, label: `Alfa ≥${analisis.ambang_alfa}×`, count: analisis.merah, dot: "bg-red-500" },
		{ key: "waspada" as FilterAlfa, label: "Waspada", count: analisis.waspada, dot: "bg-orange-500" },
		{ key: "semua" as FilterAlfa, label: "Semua guru", count: analisis.guru.length, dot: "bg-neutral-400" },
	]);

	const daftar = $derived.by(() => {
		const q = search.trim().toLowerCase();
		return analisis.guru.filter((g) => {
			if (filter === "merah" && g.level !== "merah") return false;
			if (filter === "waspada" && g.level !== "oranye") return false;
			return !q || g.nama.toLowerCase().includes(q);
		});
	});

	function tanggalPendek(iso: string): string {
		if (!iso) return "";
		return new Date(`${iso}T00:00:00`).toLocaleDateString("id-ID", { day: "numeric", month: "short", year: "numeric" });
	}

	function pesanWA(g: AnalisisKehadiranGuru): string {
		return [
			`Assalamu'alaikum ${g.nama},`,
			"",
			`Semoga senantiasa dalam lindungan Allah. Kami perhatikan antum belum hadir di ${g.alfa_beruntun} ${namaKegiatan} terakhir berturut-turut.`,
			"Apakah ada kendala yang bisa kami bantu? Kami tunggu kabarnya.",
			"",
			"Jazakumullahu khairan.",
		].join("\n");
	}

	function catatanAwal(g: AnalisisKehadiranGuru): string {
		return g.alfa_beruntun > 0 ? `Alfa ${g.alfa_beruntun}× berturut-turut di ${namaKegiatan}. ` : "";
	}
</script>

<div class="space-y-4">
	<div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
		<div class="flex flex-wrap gap-2" role="tablist" aria-label="Filter analisis">
			{#each chips as c (c.key)}
				<button
					type="button"
					role="tab"
					aria-selected={filter === c.key}
					onclick={() => (filter = c.key)}
					class="inline-flex min-h-9 items-center gap-2 rounded-full border px-3 text-sm font-medium transition-colors {filter === c.key ? 'border-brand-500/50 bg-brand-400/10 text-brand-800 dark:text-brand-200' : 'border-neutral-200 text-neutral-600 hover:bg-neutral-50 dark:border-neutral-800 dark:text-neutral-300 dark:hover:bg-neutral-900'}"
				>
					<span class="h-2 w-2 rounded-full {c.dot}"></span>
					{c.label}
					<span class="tabular-nums text-xs text-neutral-500">{c.count}</span>
				</button>
			{/each}
		</div>
		<label class="relative sm:w-64">
			<Search class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-neutral-400" />
			<input bind:value={search} placeholder="Cari nama guru" aria-label="Cari nama guru" class="w-full rounded-xl border border-neutral-200 bg-white py-2 pl-9 pr-3 text-sm outline-none focus:border-brand-400 dark:border-neutral-800 dark:bg-neutral-925" />
		</label>
	</div>

	{#if analisis.sesi.length > 0}
		<div class="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-neutral-500">
			<span>{analisis.sesi.length} sesi terlaksana terakhir, lama → baru:</span>
			{#each ["hadir", "izin", "sakit", "alfa", ""] as k (k)}
				<span class="inline-flex items-center gap-1.5"><span class="h-2.5 w-2.5 rounded-full {DOT[k as Kehadiran | ''].kelas}"></span>{DOT[k as Kehadiran | ""].label}</span>
			{/each}
		</div>
	{/if}

	<div class="space-y-3">
		{#each daftar as g (g.guru_id)}
			<article class="rounded-2xl border bg-white p-4 sm:p-5 dark:bg-neutral-925/50 {g.level === 'merah' ? 'border-red-500/30' : g.level === 'oranye' ? 'border-orange-500/30' : 'border-neutral-200/80 dark:border-white/[0.06]'}">
				<div class="flex flex-wrap items-start justify-between gap-3">
					<div class="min-w-0">
						<div class="flex flex-wrap items-center gap-2">
							<h3 class="font-semibold text-neutral-900 dark:text-white">{g.nama}</h3>
							{#if g.level === "merah"}
								<span class="rounded-full bg-red-500/10 px-2 py-0.5 text-xs font-semibold text-red-700 dark:text-red-400">Alfa {g.alfa_beruntun}× berturut-turut</span>
							{:else if g.level === "oranye"}
								<span class="rounded-full bg-orange-500/10 px-2 py-0.5 text-xs font-semibold text-orange-700 dark:text-orange-400">Waspada · alfa {g.alfa_beruntun}×</span>
							{/if}
						</div>
						<p class="mt-1 text-xs text-neutral-500">
							{#if g.dicatat_terakhir > 0}Hadir {g.hadir_terakhir} dari {g.dicatat_terakhir} sesi tercatat{:else}Belum ada catatan absen{/if}
							· {g.terakhir_hadir ? `Terakhir hadir ${tanggalPendek(g.terakhir_hadir)}` : "Belum pernah tercatat hadir"}
						</p>
					</div>
					<div class="flex items-center gap-1.5" aria-label="Riwayat kehadiran {sesiUrut.length} sesi terakhir">
						{#each sesiUrut as sesi, i (sesi.id)}
							{@const status = g.riwayat[sesiUrut.length - 1 - i] ?? ""}
							<span class="h-3.5 w-3.5 rounded-full {DOT[status].kelas}" title="{tanggalPendek(sesi.tanggal)} · {sesi.judul || '(tanpa judul)'} — {DOT[status].label}"></span>
						{/each}
					</div>
				</div>
				<div class="mt-3 flex flex-wrap gap-2">
					{#if g.no_wa}
						<a href={waLink(g.no_wa, pesanWA(g))} target="_blank" rel="noopener noreferrer" class="inline-flex min-h-9 items-center gap-1.5 rounded-lg border border-green-600/30 px-3 text-xs font-semibold text-green-700 hover:bg-green-500/10 dark:text-green-400"><MessageCircle class="h-4 w-4" /> Sapa WA</a>
					{:else}
						<span class="inline-flex min-h-9 items-center gap-1.5 rounded-lg border border-dashed border-neutral-300 px-3 text-xs text-neutral-400 dark:border-neutral-700" title="Nomor WA guru belum diisi"><MessageCircle class="h-4 w-4" /> Tanpa no. WA</span>
					{/if}
					<button type="button" onclick={() => (riayahGuru = g)} class="inline-flex min-h-9 items-center gap-1.5 rounded-lg bg-brand-400/10 px-3 text-xs font-semibold text-brand-700 hover:bg-brand-400/20 dark:text-brand-300"><NotebookPen class="h-4 w-4" /> Catat riayah</button>
					<a href={`/app/koordinator-guru/riwayat-absensi?guru=${g.guru_id}`} use:inertia class="inline-flex min-h-9 items-center gap-1.5 rounded-lg px-3 text-xs font-semibold text-neutral-600 hover:bg-neutral-100 dark:text-neutral-300 dark:hover:bg-neutral-800"><History class="h-4 w-4" /> Riwayat absen</a>
				</div>
			</article>
		{:else}
			<div class="rounded-2xl border border-dashed border-neutral-300 p-10 text-center text-sm text-neutral-500 dark:border-neutral-700">
				<UserCheck class="mx-auto mb-2 h-8 w-8 opacity-40" />
				{#if filter === "merah"}Tidak ada guru yang alfa {analisis.ambang_alfa}× berturut-turut. Alhamdulillah.{:else if filter === "waspada"}Tidak ada guru di level waspada.{:else}Tidak ada guru sesuai pencarian.{/if}
			</div>
		{/each}
	</div>
</div>

{#if riayahGuru}
	<CatatRiayahGuruDialog guru={riayahGuru} awal={catatanAwal(riayahGuru)} onclose={() => (riayahGuru = null)} />
{/if}
