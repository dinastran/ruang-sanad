<script lang="ts">
	import TrenKehadiran from "./TrenKehadiran.svelte";
	import GuruAlfaTerbanyak from "./GuruAlfaTerbanyak.svelte";
	import RiwayatTable from "./RiwayatTable.svelte";
	import {
		LABEL_KEGIATAN,
		LABEL_PERIODE,
		LABEL_STATUS,
		STATUS_KEHADIRAN,
		dalamRentang,
		formatAngka,
		formatPersen,
		guruAlfaTerbanyak,
		jumlahSesi,
		rekapKehadiran,
		rentangPeriode,
		rentangSebelumnya,
		statusRiwayat,
		trenKehadiran,
		trenPerTanggal,
		type Kegiatan,
		type Periode,
		type RiwayatAbsensi,
		type StatusKehadiran,
	} from "@lib/riwayatAbsensi";
	import { ArrowDownRight, ArrowUpRight, Search, X } from "lucide-svelte";

	interface Props {
		riwayat: RiwayatAbsensi[];
		guruList: { id: number; nama: string }[];
		/** Guru awal dari ?guru=<id>. */
		guruAwal?: string;
		/** Tanggal acuan periode; default hari ini. */
		today?: Date;
	}

	let { riwayat, guruList, guruAwal = "", today = new Date() }: Props = $props();

	const awal = () => guruAwal;
	let periode = $state<Periode>("3b");
	let kegiatan = $state<"" | Kegiatan>("");
	let guruID = $state(awal());
	let search = $state("");
	let statusTabel = $state<"" | StatusKehadiran>("");

	const rentang = $derived(rentangPeriode(periode, today));
	const rentangLalu = $derived(rentangSebelumnya(rentang));

	// Filter non-periode dipakai bersama periode sekarang dan periode sebelumnya.
	function cocok(r: RiwayatAbsensi, q: string): boolean {
		if (kegiatan && r.kegiatan !== kegiatan) return false;
		if (guruID && r.guru_id !== Number(guruID)) return false;
		if (q && !`${r.guru_nama} ${r.judul}`.toLowerCase().includes(q)) return false;
		return true;
	}

	const q = $derived(search.trim().toLowerCase());
	const terfilter = $derived(riwayat.filter((r) => dalamRentang(r.tanggal, rentang) && cocok(r, q)));
	const terfilterLalu = $derived(rentangLalu ? riwayat.filter((r) => dalamRentang(r.tanggal, rentangLalu) && cocok(r, q)) : null);

	const rekap = $derived(rekapKehadiran(terfilter));
	const rekapLalu = $derived(terfilterLalu ? rekapKehadiran(terfilterLalu) : null);
	const sesi = $derived(jumlahSesi(terfilter));
	const sesiPembinaan = $derived(jumlahSesi(terfilter.filter((r) => r.kegiatan === "pembinaan")));
	const kunjungan = $derived(terfilter.filter((r) => r.kegiatan === "kunjungan"));
	const kunjunganTerlaksana = $derived(kunjungan.filter((r) => r.hadir).length);
	const tren = $derived(trenKehadiran(terfilter, rentang));
	const perTanggal = $derived(trenPerTanggal(rentang));
	const guruAlfa = $derived(guruAlfaTerbanyak(terfilter));
	const barisTabel = $derived(statusTabel ? terfilter.filter((r) => statusRiwayat(r) === statusTabel) : terfilter);

	// Delta hanya bila periode sebelumnya punya catatan; tanpa data pembanding, delta disembunyikan.
	const deltaPersen = $derived(rekap.persen !== null && rekapLalu?.persen != null ? rekap.persen - rekapLalu.persen : null);
	const deltaAlfa = $derived(rekapLalu && rekapLalu.total > 0 ? rekap.alfa - rekapLalu.alfa : null);

	const namaGuru = $derived(guruList.find((g) => String(g.id) === guruID)?.nama ?? "");
	const chips = $derived(
		[
			kegiatan && { key: "kegiatan", label: LABEL_KEGIATAN[kegiatan], hapus: () => (kegiatan = "") },
			guruID && { key: "guru", label: namaGuru || "Guru", hapus: () => (guruID = "") },
			q && { key: "search", label: `"${search.trim()}"`, hapus: () => (search = "") },
			statusTabel && { key: "status", label: `Status: ${LABEL_STATUS[statusTabel]}`, hapus: () => (statusTabel = "") },
		].filter(Boolean) as { key: string; label: string; hapus: () => void }[],
	);

	function reset() {
		kegiatan = "";
		guruID = "";
		search = "";
		statusTabel = "";
	}

	function pilihGuru(id: number) {
		guruID = guruID === String(id) ? "" : String(id);
	}

	const PERIODE: Periode[] = ["30h", "3b", "6b", "12b", "semua"];
	const KEGIATAN: ("" | Kegiatan)[] = ["", "pembinaan", "rapat", "kunjungan"];
	const labelLalu = $derived(periode === "30h" ? "30 hari sebelumnya" : `${LABEL_PERIODE[periode]} sebelumnya`);

	const segmen = "inline-flex rounded-xl bg-neutral-100 p-1 text-xs font-semibold dark:bg-neutral-800/70";
	const segmenItem = (aktif: boolean) =>
		`min-h-8 rounded-lg px-3 transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-brand-400/60 ${aktif ? "bg-white text-neutral-900 shadow-sm dark:bg-neutral-925 dark:text-white" : "text-neutral-600 hover:text-neutral-900 dark:text-neutral-400 dark:hover:text-white"}`;
	const kartu = "rounded-2xl border border-neutral-200/80 bg-white p-4 sm:p-5 dark:border-white/[0.06] dark:bg-neutral-925/50";
</script>

<div class="space-y-5">
	<!-- Filter -->
	<div class="space-y-3">
		<div class="flex flex-col gap-3 xl:flex-row xl:items-center xl:justify-between">
			<div class="flex flex-wrap gap-2">
				<div class={segmen} role="radiogroup" aria-label="Periode">
					{#each PERIODE as p (p)}
						<button type="button" role="radio" aria-checked={periode === p} onclick={() => (periode = p)} class={segmenItem(periode === p)}>{LABEL_PERIODE[p]}</button>
					{/each}
				</div>
				<div class={segmen} role="radiogroup" aria-label="Kegiatan">
					{#each KEGIATAN as k (k)}
						<button type="button" role="radio" aria-checked={kegiatan === k} onclick={() => (kegiatan = k)} class={segmenItem(kegiatan === k)}>{k ? LABEL_KEGIATAN[k] : "Semua"}</button>
					{/each}
				</div>
			</div>
			<div class="grid gap-2 sm:grid-cols-2 lg:w-[26rem] lg:shrink-0">
				<label class="relative">
					<Search class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-neutral-400" />
					<input bind:value={search} placeholder="Cari guru atau judul" aria-label="Cari guru atau judul kegiatan" class="w-full rounded-xl border border-neutral-200 bg-white py-2 pl-9 pr-3 text-sm outline-none focus:border-brand-400 dark:border-neutral-800 dark:bg-neutral-925 dark:text-white" />
				</label>
				<select bind:value={guruID} aria-label="Filter guru" class="rounded-xl border border-neutral-200 bg-white px-3 py-2 text-sm outline-none focus:border-brand-400 dark:border-neutral-800 dark:bg-neutral-925 dark:text-white">
					<option value="">Semua guru</option>
					{#each guruList as g (g.id)}<option value={String(g.id)}>{g.nama}</option>{/each}
				</select>
			</div>
		</div>
		{#if chips.length > 0}
			<div class="flex flex-wrap items-center gap-2">
				{#each chips as c (c.key)}
					<button type="button" onclick={c.hapus} class="inline-flex min-h-8 items-center gap-1 rounded-full border border-brand-500/30 bg-brand-400/10 pl-3 pr-2 text-xs font-medium text-brand-800 hover:bg-brand-400/20 dark:text-brand-200" aria-label="Hapus filter {c.label}">
						{c.label}<X class="h-3.5 w-3.5" />
					</button>
				{/each}
				<button type="button" onclick={reset} class="text-xs font-semibold text-neutral-500 underline hover:text-neutral-800 dark:hover:text-neutral-200">Reset semua</button>
			</div>
		{/if}
	</div>

	<!-- KPI -->
	<div class="grid grid-cols-2 gap-3 lg:grid-cols-5">
		<div class="{kartu} col-span-2 lg:col-span-1">
			<p class="text-xs font-medium text-neutral-500">Tingkat kehadiran</p>
			<p class="mt-1 text-3xl font-bold tabular-nums tracking-tight text-neutral-900 dark:text-white">{formatPersen(rekap.persen)}</p>
			<p class="mt-1 text-xs text-neutral-500">
				{#if rekap.total > 0}{formatAngka(rekap.hadir)} hadir dari {formatAngka(rekap.total)} catatan{:else}Belum ada catatan absen{/if}
			</p>
			{#if deltaPersen !== null}
				<p class="mt-2 flex items-center gap-1 whitespace-nowrap text-xs font-semibold tabular-nums {deltaPersen >= 0 ? 'text-emerald-700 dark:text-emerald-400' : 'text-red-700 dark:text-red-400'}">
					{#if deltaPersen >= 0}<ArrowUpRight class="h-3.5 w-3.5 shrink-0" />{:else}<ArrowDownRight class="h-3.5 w-3.5 shrink-0" />{/if}
					{deltaPersen >= 0 ? "+" : ""}{deltaPersen.toLocaleString("id-ID", { maximumFractionDigits: 1 })} poin
				</p>
				<p class="text-xs text-neutral-500">vs {labelLalu}</p>
			{/if}
		</div>
		<div class={kartu}>
			<p class="text-xs font-medium text-neutral-500">Alfa</p>
			<p class="mt-1 text-3xl font-bold tabular-nums tracking-tight {rekap.alfa > 0 ? 'text-red-600 dark:text-red-400' : 'text-neutral-900 dark:text-white'}">{rekap.total > 0 ? formatAngka(rekap.alfa) : "–"}</p>
			<p class="mt-1 text-xs text-neutral-500">tanpa keterangan</p>
			{#if deltaAlfa !== null}
				<p class="mt-2 flex items-center gap-1 whitespace-nowrap text-xs font-semibold tabular-nums {deltaAlfa <= 0 ? 'text-emerald-700 dark:text-emerald-400' : 'text-red-700 dark:text-red-400'}">
					{#if deltaAlfa <= 0}<ArrowDownRight class="h-3.5 w-3.5 shrink-0" />{:else}<ArrowUpRight class="h-3.5 w-3.5 shrink-0" />{/if}
					{deltaAlfa > 0 ? "+" : ""}{formatAngka(deltaAlfa)}
				</p>
				<p class="text-xs text-neutral-500">vs {labelLalu}</p>
			{/if}
		</div>
		<div class={kartu}>
			<p class="text-xs font-medium text-neutral-500">Izin &amp; sakit</p>
			<p class="mt-1 text-3xl font-bold tabular-nums tracking-tight text-neutral-900 dark:text-white">{rekap.total > 0 ? formatAngka(rekap.izin + rekap.sakit) : "–"}</p>
			<p class="mt-1 text-xs tabular-nums text-neutral-500">izin {formatAngka(rekap.izin)} · sakit {formatAngka(rekap.sakit)}</p>
		</div>
		<div class={kartu}>
			<p class="text-xs font-medium text-neutral-500">Sesi tercatat</p>
			<p class="mt-1 text-3xl font-bold tabular-nums tracking-tight text-neutral-900 dark:text-white">{formatAngka(sesi)}</p>
			<p class="mt-1 text-xs tabular-nums text-neutral-500">pembinaan {formatAngka(sesiPembinaan)} · rapat {formatAngka(sesi - sesiPembinaan)}</p>
		</div>
		<div class={kartu}>
			<p class="text-xs font-medium text-neutral-500">Kunjungan kelas</p>
			<p class="mt-1 text-3xl font-bold tabular-nums tracking-tight text-neutral-900 dark:text-white">{formatAngka(kunjunganTerlaksana)}<span class="ml-1 text-base font-semibold text-neutral-400">/ {formatAngka(kunjungan.length)}</span></p>
			<p class="mt-1 text-xs text-neutral-500">terlaksana dari jadwal</p>
		</div>
	</div>

	<!-- Grafik -->
	<div class="grid gap-3 lg:grid-cols-5">
		<div class="min-w-0 lg:col-span-3"><TrenKehadiran buckets={tren} {perTanggal} /></div>
		<div class="min-w-0 lg:col-span-2"><GuruAlfaTerbanyak items={guruAlfa} aktif={guruID} onpilih={pilihGuru} /></div>
	</div>

	<!-- Detail -->
	<div class="space-y-3">
		<div class="flex flex-wrap items-center gap-2">
			<span class="text-xs font-medium text-neutral-500">Status di tabel:</span>
			<div class={segmen} role="radiogroup" aria-label="Status di tabel">
				<button type="button" role="radio" aria-checked={statusTabel === ""} onclick={() => (statusTabel = "")} class={segmenItem(statusTabel === "")}>Semua</button>
				{#each STATUS_KEHADIRAN as s (s)}
					<button type="button" role="radio" aria-checked={statusTabel === s} onclick={() => (statusTabel = s)} class={segmenItem(statusTabel === s)}>{LABEL_STATUS[s]}</button>
				{/each}
			</div>
		</div>
		<RiwayatTable rows={barisTabel} adaFilter={riwayat.length > 0 && (chips.length > 0 || periode !== "semua")} onreset={() => { reset(); periode = "semua"; }} />
	</div>
</div>
