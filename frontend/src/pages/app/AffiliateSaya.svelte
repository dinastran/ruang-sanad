<script lang="ts">
	import { router } from "@inertiajs/svelte";
	import AppLayout from "@layouts/AppLayout.svelte";
	import type { Flash, User } from "@lib/types";
	import { Check, Copy, HandCoins, Link2, MessageCircle, MousePointerClick, UserPlus, Users } from "lucide-svelte";

	interface Rujukan { nama_depan: string; program: string; tanggal: string; status: string; komisi: number }
	interface Dash {
		id: number; kode: string; nama: string; tipe: string; bank_nama: string; bank_rekening: string; bank_atas_nama: string;
		klik_30: number; daftar_30: number; santri: number; komisi_pending: number; komisi_siap: number; komisi_dibayar: number;
		masa_berlaku_hari: number; masa_tahan_hari: number;
		aturan: { label: string; pola: string; durasi: string } | null;
		rujukan: Rujukan[]; tujuan: { slug: string; judul: string }[]; tujuan_default: string; template_bagikan: string;
	}
	interface Props { user?: User; affiliate?: Dash; belum_terdaftar?: boolean; base_url: string; flash?: Flash }
	let props: Props = $props();
	let a = $derived(props.affiliate);

	let tujuan = $state(props.affiliate?.tujuan_default ?? "");
	let disalin = $state("");
	let rek = $state({ bank_nama: props.affiliate?.bank_nama ?? "", bank_rekening: props.affiliate?.bank_rekening ?? "", bank_atas_nama: props.affiliate?.bank_atas_nama ?? "" });
	let menyimpan = $state(false);

	let link = $derived(a ? `${props.base_url}/r/${a.kode}${tujuan ? `?ke=${encodeURIComponent(tujuan)}` : ""}` : "");
	let pesanBagikan = $derived(
		(a?.template_bagikan || "Assalamu'alaikum, yuk belajar Al-Qur'an (tahsin) bersama Ruang Sanad. Daftar lewat link ini: {link}").replace("{link}", link),
	);
	const rupiah = (n: number) => "Rp" + (n ?? 0).toLocaleString("id-ID");
	const STATUS: Record<string, [string, string]> = {
		daftar: ["Mendaftar", "bg-sky-500/10 text-sky-700 dark:text-sky-400"],
		terverifikasi: ["Sudah bayar", "bg-green-500/10 text-green-700 dark:text-green-400"],
		santri: ["Santri", "bg-green-500/10 text-green-700 dark:text-green-400"],
		batal: ["Tidak lanjut", "bg-neutral-500/10 text-neutral-600"],
	};

	async function salin(teks: string, apa: string) {
		try {
			await navigator.clipboard.writeText(teks);
			disalin = apa;
			setTimeout(() => (disalin = ""), 1500);
		} catch {
			/* clipboard tidak tersedia */
		}
	}
	function simpanRekening(e: Event) {
		e.preventDefault();
		menyimpan = true;
		router.put("/app/affiliate/rekening", rek, { preserveScroll: true, onFinish: () => (menyimpan = false) });
	}

	const panel = "rounded-xl border border-neutral-200/80 bg-white dark:border-white/[0.07] dark:bg-neutral-925/60";
	const field = "h-10 w-full rounded-md border border-neutral-200 bg-neutral-50 px-3 text-sm text-neutral-900 focus:border-brand-500 focus:outline-none focus:ring-2 focus:ring-brand-500/30 dark:border-white/10 dark:bg-neutral-950 dark:text-white";
</script>

<svelte:head><title>Affiliate Saya · Ruang Sanad</title></svelte:head>
<AppLayout user={props.user}>
	<div class="mx-auto max-w-4xl space-y-4 px-4 py-6 sm:px-6 lg:py-8">
		<header>
			<h1 class="text-[28px] font-semibold leading-tight tracking-[-0.02em] text-neutral-950 dark:text-white">Affiliate Saya</h1>
			{#if a}<p class="mt-1 text-sm text-neutral-500 dark:text-neutral-400">Bagikan link Anda. Pendaftar yang klik dalam {a.masa_berlaku_hari} hari tercatat sebagai rujukan Anda.</p>{/if}
		</header>
		{#if props.flash?.success}<div role="status" class="rounded-lg border border-success/20 bg-success/10 px-4 py-3 text-sm font-medium text-emerald-800 dark:text-emerald-300">{props.flash.success}</div>{/if}
		{#if props.flash?.error}<div role="alert" class="rounded-lg border border-error/20 bg-error/10 px-4 py-3 text-sm font-medium text-red-700 dark:text-red-300">{props.flash.error}</div>{/if}

		{#if !a}
			<div class={panel + " px-6 py-14 text-center"}>
				<Link2 class="mx-auto h-8 w-8 text-neutral-400" />
				<p class="mt-3 font-medium text-neutral-900 dark:text-white">Anda belum terdaftar sebagai affiliate</p>
				<p class="mx-auto mt-1 max-w-sm text-sm text-neutral-500">Guru yang punya akun otomatis mendapat link affiliate (diperbarui tiap 10 menit). Hubungi admin bila link belum muncul.</p>
			</div>
		{:else}
			<section class={panel + " p-5"} aria-labelledby="judul-link">
				<div class="flex flex-wrap items-baseline justify-between gap-2">
					<h2 id="judul-link" class="text-sm font-semibold text-neutral-900 dark:text-white">Link Anda</h2>
					<span class="text-xs text-neutral-500">Kode: <button type="button" class="font-mono font-semibold text-neutral-900 hover:underline dark:text-white" onclick={() => salin(a.kode, "kode")}>{a.kode}</button>{disalin === "kode" ? " · tersalin" : ""}</span>
				</div>
				{#if a.tujuan.length > 1}
					<label class="mt-3 block text-[11px] font-medium text-neutral-500">Arahkan ke
						<select bind:value={tujuan} class={field + " mt-1"}>{#each a.tujuan as t (t.slug)}<option value={t.slug}>{t.judul}</option>{/each}</select>
					</label>
				{/if}
				<div class="mt-3 flex items-center gap-2 rounded-lg bg-neutral-100 p-2 pl-3 dark:bg-neutral-900">
					<code class="min-w-0 flex-1 truncate text-sm text-neutral-900 dark:text-white">{link}</code>
					<button type="button" onclick={() => salin(link, "link")} class="inline-flex h-9 shrink-0 items-center gap-1.5 rounded-md bg-white px-3 text-sm font-medium text-neutral-800 shadow-sm hover:bg-neutral-50 dark:bg-neutral-800 dark:text-white">
						{#if disalin === "link"}<Check size={15} /> Tersalin{:else}<Copy size={15} /> Salin{/if}
					</button>
				</div>
				<a href={`https://wa.me/?text=${encodeURIComponent(pesanBagikan)}`} target="_blank" rel="noopener"
					class="mt-3 flex min-h-11 w-full items-center justify-center gap-2 rounded-lg bg-brand-600 text-sm font-semibold text-white transition hover:bg-brand-700 active:scale-[0.98] dark:bg-brand-500">
					<MessageCircle size={17} /> Bagikan ke WhatsApp
				</a>
				{#if a.aturan}
					<p class="mt-3 text-xs text-neutral-500">Komisi Anda: <span class="font-semibold text-neutral-800 dark:text-neutral-200">{a.aturan.label}</span>{a.aturan.pola === "bulanan" ? ` dari pembayaran pertama dan SPP bulanan (${a.aturan.durasi})` : " dari pembayaran pertama"}. Cair setelah masa tahan {a.masa_tahan_hari} hari.</p>
				{:else}
					<p class="mt-3 text-xs text-amber-700 dark:text-amber-400">Aturan komisi belum diatur admin.</p>
				{/if}
			</section>

			<section class="grid grid-cols-2 gap-3 lg:grid-cols-4" aria-label="Ringkasan 30 hari">
				{#each [[MousePointerClick, "Klik (30 hari)", String(a.klik_30)], [UserPlus, "Mendaftar (30 hari)", String(a.daftar_30)], [Users, "Sudah bayar", String(a.santri)], [HandCoins, "Siap cair", rupiah(a.komisi_siap)]] as [Icon, label, val]}
					<div class={panel + " p-4"}>
						<Icon class="h-4 w-4 text-neutral-400" />
						<p class="mt-2 text-xl font-semibold tabular-nums text-neutral-900 dark:text-white">{val}</p>
						<p class="text-xs text-neutral-500">{label}</p>
					</div>
				{/each}
			</section>
			<p class="text-xs text-neutral-500">Komisi tertahan {rupiah(a.komisi_pending)} · sudah dibayar {rupiah(a.komisi_dibayar)}{a.tipe === "guru" ? " (lewat ujroh)" : ""}.</p>

			<section class={panel} aria-labelledby="judul-rujukan">
				<h2 id="judul-rujukan" class="border-b border-neutral-100 px-5 py-3 text-sm font-semibold text-neutral-900 dark:border-white/[0.06] dark:text-white">Rujukan Anda</h2>
				{#if a.rujukan.length === 0}
					<p class="px-5 py-10 text-center text-sm text-neutral-500">Belum ada pendaftar dari link Anda. Mulai dengan membagikan link di atas.</p>
				{:else}
					<ul class="divide-y divide-neutral-100 dark:divide-white/[0.05]">
						{#each a.rujukan as r, i (i)}
							{@const st = STATUS[r.status] ?? [r.status, ""]}
							<li class="flex items-center justify-between gap-3 px-5 py-3 text-sm">
								<div class="min-w-0">
									<p class="font-medium text-neutral-900 dark:text-white">{r.nama_depan || "Pendaftar"}</p>
									<p class="text-xs text-neutral-500">{r.program || "—"} · {r.tanggal}</p>
								</div>
								<div class="flex shrink-0 items-center gap-3">
									{#if r.komisi > 0}<span class="tabular-nums text-neutral-700 dark:text-neutral-300">{rupiah(r.komisi)}</span>{/if}
									<span class="rounded-full px-2 py-0.5 text-xs font-medium {st[1]}">{st[0]}</span>
								</div>
							</li>
						{/each}
					</ul>
				{/if}
			</section>

			{#if a.tipe === "eksternal"}
				<form class={panel + " grid gap-3 p-5 sm:grid-cols-[140px_1fr_1fr_auto] sm:items-end"} onsubmit={simpanRekening}>
					<h2 class="text-sm font-semibold text-neutral-900 sm:col-span-4 dark:text-white">Rekening pencairan</h2>
					<label class="text-[11px] font-medium text-neutral-500">Bank<input class={field + " mt-1"} bind:value={rek.bank_nama} maxlength="60" /></label>
					<label class="text-[11px] font-medium text-neutral-500">No. rekening<input class={field + " mt-1"} bind:value={rek.bank_rekening} inputmode="numeric" maxlength="40" /></label>
					<label class="text-[11px] font-medium text-neutral-500">Atas nama<input class={field + " mt-1"} bind:value={rek.bank_atas_nama} maxlength="80" /></label>
					<button type="submit" disabled={menyimpan} class="h-10 rounded-md bg-neutral-900 px-4 text-sm font-medium text-white disabled:opacity-50 dark:bg-white dark:text-neutral-900">Simpan</button>
				</form>
			{/if}
		{/if}
	</div>
</AppLayout>
