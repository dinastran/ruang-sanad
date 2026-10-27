<script lang="ts">
	import { LABEL_KEGIATAN, LABEL_STATUS, WARNA_STATUS, formatAngka, formatTanggal, statusRiwayat, type Kegiatan, type RiwayatAbsensi } from "@lib/riwayatAbsensi";
	import { ArrowDown, ArrowUp, ChevronsUpDown, ClipboardList } from "lucide-svelte";

	interface Props {
		rows: RiwayatAbsensi[];
		adaFilter: boolean;
		onreset: () => void;
	}

	let { rows, adaFilter, onreset }: Props = $props();

	type Kolom = "tanggal" | "guru" | "kegiatan" | "status";
	const PER_HALAMAN = 25;

	let sortKolom = $state<Kolom>("tanggal");
	let sortNaik = $state(false);
	let halaman = $state(1);

	function urutkan(kolom: Kolom) {
		if (sortKolom === kolom) sortNaik = !sortNaik;
		else {
			sortKolom = kolom;
			sortNaik = kolom !== "tanggal";
		}
		halaman = 1;
	}

	function nilai(r: RiwayatAbsensi, kolom: Kolom): string {
		switch (kolom) {
			case "tanggal":
				return r.tanggal;
			case "guru":
				return r.guru_nama.toLowerCase();
			case "kegiatan":
				return r.kegiatan;
			case "status":
				return statusRiwayat(r);
		}
	}

	const urut = $derived.by(() => {
		const arah = sortNaik ? 1 : -1;
		return [...rows].sort((a, b) => {
			const x = nilai(a, sortKolom);
			const y = nilai(b, sortKolom);
			if (x !== y) return x < y ? -arah : arah;
			// Urutan kedua: terbaru dulu, lalu nama guru.
			if (a.tanggal !== b.tanggal) return a.tanggal < b.tanggal ? 1 : -1;
			return a.guru_nama.localeCompare(b.guru_nama);
		});
	});

	const totalHalaman = $derived(Math.max(1, Math.ceil(urut.length / PER_HALAMAN)));
	const halamanAktif = $derived(Math.min(halaman, totalHalaman));
	const tampil = $derived(urut.slice((halamanAktif - 1) * PER_HALAMAN, halamanAktif * PER_HALAMAN));

	const KOLOM: { key: Kolom; label: string }[] = [
		{ key: "tanggal", label: "Tanggal" },
		{ key: "guru", label: "Guru" },
		{ key: "kegiatan", label: "Kegiatan" },
		{ key: "status", label: "Status" },
	];
</script>

<section class="overflow-hidden rounded-2xl border border-neutral-200/80 bg-white dark:border-white/[0.06] dark:bg-neutral-925/50">
	<div class="flex items-center justify-between gap-3 border-b border-neutral-200/80 px-5 py-4 dark:border-white/[0.06]">
		<h2 class="text-sm font-semibold text-neutral-900 dark:text-white">Detail riwayat</h2>
		<p class="text-xs tabular-nums text-neutral-500">{formatAngka(rows.length)} catatan</p>
	</div>
	<!-- Mobile: kartu, supaya status tidak tersembunyi di balik scroll horizontal. -->
	<ul class="divide-y divide-neutral-100 sm:hidden dark:divide-neutral-800">
		{#each tampil as r (`${r.kegiatan}-${r.kegiatan_id}-${r.guru_id}`)}
			{@const status = statusRiwayat(r)}
			<li class="px-4 py-3">
				<div class="flex items-start justify-between gap-3">
					<p class="min-w-0 font-medium text-neutral-900 dark:text-white">{r.guru_nama}</p>
					<span class="shrink-0 whitespace-nowrap rounded-full px-2 py-0.5 text-xs font-semibold {WARNA_STATUS[status].pill}">{LABEL_STATUS[status]}</span>
				</div>
				<p class="mt-0.5 truncate text-xs text-neutral-500">{LABEL_KEGIATAN[r.kegiatan as Kegiatan] ?? r.kegiatan} · {r.judul || "–"}</p>
				<p class="mt-0.5 text-xs tabular-nums text-neutral-500">{formatTanggal(r.tanggal)}{r.jam_masuk ? ` · masuk ${r.jam_masuk}` : ""}</p>
				{#if r.alasan && r.kegiatan !== "kunjungan"}<p class="mt-1 text-xs text-neutral-600 dark:text-neutral-400">Alasan: {r.alasan}</p>{/if}
				{#if r.keterangan}<p class="mt-1 text-xs text-neutral-600 dark:text-neutral-400">{r.keterangan}</p>{/if}
			</li>
		{:else}
			<li class="p-10 text-center text-sm text-neutral-500">
				{#if adaFilter}Tidak ada riwayat yang cocok dengan filter. <button type="button" onclick={onreset} class="font-semibold text-brand-700 underline dark:text-brand-300">Reset filter</button>{:else}Belum ada riwayat absensi.{/if}
			</li>
		{/each}
	</ul>
	<div class="hidden overflow-x-auto sm:block">
		<table class="w-full min-w-[760px] text-sm">
			<thead class="bg-neutral-50 text-xs text-neutral-500 dark:bg-neutral-900/50">
				<tr>
					{#each KOLOM as k (k.key)}
						<th class="p-0 text-left font-semibold" aria-sort={sortKolom === k.key ? (sortNaik ? "ascending" : "descending") : "none"}>
							<button type="button" onclick={() => urutkan(k.key)} class="inline-flex w-full items-center gap-1 px-4 py-3 uppercase tracking-wide hover:text-neutral-800 focus:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-brand-400/60 dark:hover:text-neutral-200">
								{k.label}
								{#if sortKolom === k.key}
									{#if sortNaik}<ArrowUp class="h-3.5 w-3.5" />{:else}<ArrowDown class="h-3.5 w-3.5" />{/if}
								{:else}
									<ChevronsUpDown class="h-3.5 w-3.5 opacity-40" />
								{/if}
							</button>
						</th>
					{/each}
					<th class="px-4 py-3 text-left font-semibold uppercase tracking-wide">Jam masuk</th>
					<th class="px-4 py-3 text-left font-semibold uppercase tracking-wide">Keterangan</th>
				</tr>
			</thead>
			<tbody class="divide-y divide-neutral-100 dark:divide-neutral-800">
				{#each tampil as r (`${r.kegiatan}-${r.kegiatan_id}-${r.guru_id}`)}
					{@const status = statusRiwayat(r)}
					<tr class="hover:bg-neutral-50/60 dark:hover:bg-white/[0.02]">
						<td class="whitespace-nowrap px-4 py-3 tabular-nums text-neutral-600 dark:text-neutral-400">{formatTanggal(r.tanggal)}</td>
						<td class="px-4 py-3 font-medium text-neutral-900 dark:text-white">{r.guru_nama}</td>
						<td class="px-4 py-3">
							<p class="text-neutral-800 dark:text-neutral-200">{LABEL_KEGIATAN[r.kegiatan as Kegiatan] ?? r.kegiatan}</p>
							<p class="max-w-56 truncate text-xs text-neutral-500">{r.judul || "–"}</p>
						</td>
						<td class="px-4 py-3"><span class="whitespace-nowrap rounded-full px-2 py-0.5 text-xs font-semibold {WARNA_STATUS[status].pill}">{LABEL_STATUS[status]}</span></td>
						<td class="px-4 py-3 font-mono text-xs tabular-nums text-neutral-600 dark:text-neutral-400">{r.jam_masuk || "–"}</td>
						<td class="px-4 py-3 text-neutral-700 dark:text-neutral-300">
							<p class="max-w-72">{r.keterangan || "–"}</p>
							{#if r.alasan && r.kegiatan !== "kunjungan"}<p class="mt-0.5 text-xs text-neutral-500">Alasan: {r.alasan}</p>{/if}
						</td>
					</tr>
				{:else}
					<tr>
						<td colspan="6" class="p-12 text-center text-sm text-neutral-500">
							<ClipboardList class="mx-auto mb-2 h-7 w-7 opacity-40" />
							{#if adaFilter}
								Tidak ada riwayat yang cocok dengan filter.
								<button type="button" onclick={onreset} class="ml-1 font-semibold text-brand-700 underline dark:text-brand-300">Reset filter</button>
							{:else}
								Belum ada riwayat absensi.
							{/if}
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
	{#if urut.length > PER_HALAMAN}
		<div class="flex items-center justify-between gap-3 border-t border-neutral-200/80 px-5 py-3 text-xs text-neutral-500 dark:border-white/[0.06]">
			<p class="tabular-nums">{formatAngka((halamanAktif - 1) * PER_HALAMAN + 1)}–{formatAngka(Math.min(halamanAktif * PER_HALAMAN, urut.length))} dari {formatAngka(urut.length)}</p>
			<div class="flex gap-2">
				<button type="button" disabled={halamanAktif <= 1} onclick={() => (halaman = halamanAktif - 1)} class="min-h-9 rounded-lg border border-neutral-200 px-3 font-semibold text-neutral-700 hover:bg-neutral-50 disabled:opacity-40 dark:border-neutral-700 dark:text-neutral-300 dark:hover:bg-neutral-800">Sebelumnya</button>
				<button type="button" disabled={halamanAktif >= totalHalaman} onclick={() => (halaman = halamanAktif + 1)} class="min-h-9 rounded-lg border border-neutral-200 px-3 font-semibold text-neutral-700 hover:bg-neutral-50 disabled:opacity-40 dark:border-neutral-700 dark:text-neutral-300 dark:hover:bg-neutral-800">Berikutnya</button>
			</div>
		</div>
	{/if}
</section>
