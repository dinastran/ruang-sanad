<script lang="ts">
	// Riwayat sesi dikelompokkan per hari. Filter dzikir lewat chip; pagination via fetch JSON.
	import { CloudUpload } from "lucide-svelte";
	import { getCSRFToken } from "@lib/utils/csrf";
	import { angka, durasi } from "./format";
	import type { Dzikir, RiwayatItem, SesiTertutup } from "@lib/tasbih/types";
	import { tanggalWIB } from "@lib/tasbih/types";

	interface Props {
		awal: RiwayatItem[];
		adaLagiAwal: boolean;
		dzikir: Dzikir[];
		lokal: SesiTertutup[];
	}
	let { awal, adaLagiAwal, dzikir, lokal }: Props = $props();

	type Baris = { id: string; tanggal: string; jam: string; latin: string; jumlah: number; target: number; detail: string; lokal: boolean };

	let items = $state<RiwayatItem[]>(awal);
	let adaLagi = $state(adaLagiAwal);
	let halaman = $state(1);
	let filter = $state<number | null>(null);
	let memuat = $state(false);
	let galat = $state("");

	const jamWIB = (iso: string) => new Date(iso).toLocaleTimeString("id-ID", { hour: "2-digit", minute: "2-digit", timeZone: "Asia/Jakarta" });
	const hariIni = tanggalWIB(new Date());

	const baris = $derived.by<Baris[]>(() => {
		const dariLokal = lokal
			.filter((s) => filter === null || s.dzikir_id === filter)
			.map((s) => ({
				id: s.id,
				tanggal: tanggalWIB(s.mulai_at),
				jam: jamWIB(s.mulai_at),
				latin: dzikir.find((d) => d.id === s.dzikir_id)?.latin ?? "Dzikir",
				jumlah: s.jumlah,
				target: s.target,
				detail: "",
				lokal: true,
			}))
			.reverse();
		const dariServer = items.map((s) => ({
			id: s.id,
			tanggal: s.tanggal,
			jam: s.jam,
			latin: s.dzikir_latin,
			jumlah: s.jumlah,
			target: s.target,
			detail: [s.paket_nama, durasi(s.durasi_detik)].filter(Boolean).join(" · "),
			lokal: false,
		}));
		return [...dariLokal, ...dariServer];
	});

	const grup = $derived.by(() => {
		const out: { tanggal: string; sesi: Baris[] }[] = [];
		for (const b of baris) {
			const last = out.at(-1);
			if (last && last.tanggal === b.tanggal) last.sesi.push(b);
			else out.push({ tanggal: b.tanggal, sesi: [b] });
		}
		return out;
	});

	function labelHari(iso: string) {
		const [y, m, d] = iso.split("-").map(Number);
		const tgl = new Intl.DateTimeFormat("id-ID", { weekday: "long", day: "numeric", month: "short" }).format(new Date(y, m - 1, d));
		return iso === hariIni ? `Hari ini · ${tgl}` : tgl;
	}

	async function muat(reset: boolean) {
		memuat = true;
		galat = "";
		const hlm = reset ? 1 : halaman + 1;
		const q = new URLSearchParams({ halaman: String(hlm) });
		if (filter !== null) q.set("dzikir_id", String(filter));
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

	function pilihFilter(id: number | null) {
		if (filter === id) return;
		filter = id;
		muat(true);
	}

	const chip = "h-9 shrink-0 rounded-full border px-3.5 text-[13px] font-semibold transition-colors";
	const chipOn = "border-neutral-900 bg-neutral-900 text-white dark:border-white dark:bg-white dark:text-neutral-950";
	const chipOff = "border-neutral-200 bg-white text-neutral-600 hover:border-neutral-300 dark:border-brand-200/10 dark:bg-brand-950/40 dark:text-brand-100/70";
</script>

<div class="-mx-4 overflow-x-auto px-4 pb-1 [scrollbar-width:none]" role="group" aria-label="Filter dzikir">
	<div class="flex gap-2">
		<button type="button" class="{chip} {filter === null ? chipOn : chipOff}" aria-pressed={filter === null} onclick={() => pilihFilter(null)}>Semua</button>
		{#each dzikir.filter((x) => !x.hanya_paket) as d (d.id)}
			<button type="button" class="{chip} {filter === d.id ? chipOn : chipOff}" aria-pressed={filter === d.id} onclick={() => pilihFilter(d.id)}>{d.latin}</button>
		{/each}
	</div>
</div>

<div aria-busy={memuat}>
	{#if grup.length === 0}
		<p class="mt-6 rounded-2xl bg-neutral-100 px-4 py-10 text-center text-sm text-neutral-500 dark:bg-brand-950/40 dark:text-brand-100/50">
			{memuat ? "Memuat…" : `Belum ada sesi${filter !== null ? " untuk dzikir ini" : ""}.`}
		</p>
	{:else}
		{#each grup as g (g.tanggal)}
			<section class="mt-5" aria-label={labelHari(g.tanggal)}>
				<h2 class="flex items-baseline justify-between text-xs font-semibold tracking-[0.04em] text-neutral-500 uppercase dark:text-brand-100/45">
					<span>{labelHari(g.tanggal)}</span>
					<span class="font-medium normal-case tracking-normal tabular-nums">{g.sesi.length} sesi</span>
				</h2>
				<ul class="mt-1 divide-y divide-neutral-200/70 dark:divide-brand-200/10" data-testid="riwayat">
					{#each g.sesi as s (s.id)}
						{@const tuntas = s.target > 0 && s.jumlah >= s.target}
						<li class="grid grid-cols-[3rem_minmax(0,1fr)_auto] items-center gap-3 py-3">
							<time class="text-[13px] tabular-nums text-neutral-400 dark:text-brand-100/40">{s.jam}</time>
							<span class="min-w-0">
								<span class="block truncate text-[14.5px] font-semibold text-neutral-900 dark:text-white">{s.latin}</span>
								{#if s.lokal}
									<span class="inline-flex items-center gap-1 text-xs font-medium text-warning"><CloudUpload size={12} aria-hidden="true" /> belum tersinkron</span>
								{:else if s.detail}
									<span class="block truncate text-xs text-neutral-500 dark:text-brand-100/45">{s.detail}</span>
								{/if}
							</span>
							<span class="text-right">
								<b class="block text-[15px] font-semibold tabular-nums {tuntas ? 'text-success' : 'text-neutral-900 dark:text-white'}">{angka(s.jumlah)}</b>
								<span class="block text-[11.5px] tabular-nums text-neutral-400 dark:text-brand-100/40">{tuntas ? "tuntas" : `/ ${angka(s.target)}`}</span>
							</span>
						</li>
					{/each}
				</ul>
			</section>
		{/each}
	{/if}
</div>

{#if galat}<p class="mt-3 text-sm text-red-600 dark:text-red-400" role="alert">{galat}</p>{/if}
{#if adaLagi}
	<button type="button" onclick={() => muat(false)} disabled={memuat} class="mt-4 h-11 w-full rounded-xl border border-neutral-200 bg-white text-sm font-medium text-neutral-700 disabled:opacity-60 dark:border-brand-200/10 dark:bg-brand-950/40 dark:text-brand-100/80">
		{memuat ? "Memuat…" : "Muat lagi"}
	</button>
{/if}
