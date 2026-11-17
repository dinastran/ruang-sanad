<script lang="ts">
	import { CloudUpload } from "lucide-svelte";
	import { getCSRFToken } from "@lib/utils/csrf";
	import { angka, durasi, tglPendek } from "./format";
	import type { Dzikir, RiwayatItem, SesiTertutup } from "@lib/tasbih/types";
	import { tanggalWIB } from "@lib/tasbih/types";

	interface Props {
		awal: RiwayatItem[];
		adaLagiAwal: boolean;
		dzikir: Dzikir[];
		lokal: SesiTertutup[];
	}
	let { awal, adaLagiAwal, dzikir, lokal }: Props = $props();

	let items = $state<RiwayatItem[]>(awal);
	let adaLagi = $state(adaLagiAwal);
	let halaman = $state(1);
	let filter = $state<number | "">("");
	let memuat = $state(false);
	let galat = $state("");

	const lokalTampil = $derived(
		lokal
			.filter((s) => filter === "" || s.dzikir_id === filter)
			.map((s) => ({
				id: s.id,
				tanggal: tanggalWIB(s.mulai_at),
				jam: new Date(s.mulai_at).toLocaleTimeString("id-ID", { hour: "2-digit", minute: "2-digit", timeZone: "Asia/Jakarta" }),
				latin: dzikir.find((d) => d.id === s.dzikir_id)?.latin ?? "Dzikir",
				jumlah: s.jumlah,
				target: s.target,
			}))
			.reverse(),
	);

	async function muat(reset: boolean) {
		memuat = true;
		galat = "";
		const hlm = reset ? 1 : halaman + 1;
		const q = new URLSearchParams({ halaman: String(hlm) });
		if (filter !== "") q.set("dzikir_id", String(filter));
		try {
			const res = await fetch(`/app/tasbih/riwayat?${q}`, { headers: { Accept: "application/json", "X-XSRF-TOKEN": getCSRFToken() }, credentials: "same-origin" });
			if (!res.ok) throw new Error(String(res.status));
			const data = (await res.json()) as { items: RiwayatItem[]; ada_lagi: boolean };
			items = reset ? data.items : [...items, ...data.items];
			adaLagi = data.ada_lagi;
			halaman = hlm;
		} catch {
			galat = "Gagal memuat riwayat. Periksa koneksi lalu coba lagi.";
		} finally {
			memuat = false;
		}
	}
</script>

<div class="mb-3 flex items-center justify-between gap-3">
	<label for="filter-dzikir" class="sr-only">Filter dzikir</label>
	<select id="filter-dzikir" bind:value={filter} onchange={() => muat(true)} class="h-10 w-full max-w-xs rounded-lg border border-neutral-200 bg-white px-3 text-sm dark:border-neutral-700 dark:bg-neutral-950 dark:text-white">
		<option value="">Semua dzikir</option>
		{#each dzikir as d (d.id)}<option value={d.id}>{d.latin}</option>{/each}
	</select>
</div>

{#if lokalTampil.length === 0 && items.length === 0}
	<p class="rounded-xl bg-neutral-50 px-4 py-8 text-center text-sm text-neutral-500 dark:bg-neutral-800/40 dark:text-neutral-400">Belum ada sesi{filter !== "" ? " untuk dzikir ini" : ""}.</p>
{:else}
	<ul class="divide-y divide-neutral-100 dark:divide-neutral-800" data-testid="riwayat">
		{#each lokalTampil as s (s.id)}
			<li class="flex items-center gap-3 py-3">
				<span class="w-14 shrink-0 text-xs tabular-nums text-neutral-500"><span class="block">{tglPendek(s.tanggal)}</span>{s.jam}</span>
				<span class="min-w-0 flex-1">
					<span class="block truncate text-sm font-medium text-neutral-900 dark:text-white">{s.latin}</span>
					<span class="inline-flex items-center gap-1 text-[11px] font-medium text-warning"><CloudUpload size={12} aria-hidden="true" /> belum tersinkron</span>
				</span>
				<span class="shrink-0 text-sm font-semibold tabular-nums text-neutral-900 dark:text-white">{angka(s.jumlah)}<span class="font-normal text-neutral-400">/{s.target}</span></span>
			</li>
		{/each}
		{#each items as s (s.id)}
			<li class="flex items-center gap-3 py-3">
				<span class="w-14 shrink-0 text-xs tabular-nums text-neutral-500"><span class="block">{tglPendek(s.tanggal)}</span>{s.jam}</span>
				<span class="min-w-0 flex-1">
					<span class="block truncate text-sm font-medium text-neutral-900 dark:text-white">{s.dzikir_latin}</span>
					<span class="block truncate text-[11px] text-neutral-500 dark:text-neutral-400">{durasi(s.durasi_detik)}{s.paket_nama ? ` · ${s.paket_nama}` : ""}</span>
				</span>
				<span class="shrink-0 text-sm font-semibold tabular-nums {s.jumlah >= s.target ? 'text-success' : 'text-neutral-900 dark:text-white'}">{angka(s.jumlah)}<span class="font-normal text-neutral-400">/{s.target}</span></span>
			</li>
		{/each}
	</ul>
{/if}
{#if galat}<p class="mt-2 text-sm text-red-600 dark:text-red-400" role="alert">{galat}</p>{/if}
{#if adaLagi}
	<button type="button" onclick={() => muat(false)} disabled={memuat} class="mt-3 h-11 w-full rounded-xl bg-neutral-100 text-sm font-medium text-neutral-700 disabled:opacity-60 dark:bg-neutral-800 dark:text-neutral-200">
		{memuat ? "Memuat…" : "Muat lagi"}
	</button>
{/if}
