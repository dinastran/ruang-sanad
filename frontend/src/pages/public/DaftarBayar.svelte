<script lang="ts">
	import Logo from "@components/Logo.svelte";
	import { rupiah } from "@components/registration/types";
	import { onMount } from "svelte";
	import { initPixel, setPixelUser, trackPixel } from "@lib/metaPixel";
	import { ChevronDown, CircleAlert, CircleCheck, Clock, Copy, FileText, MessageCircle, QrCode, Upload } from "lucide-svelte";

	interface BayarInfo {
		lead_ref: string;
		nama_depan: string;
		program_nama: string;
		nominal: number;
		bank_manual_info: string;
		izinkan_manual: boolean;
		status: "belum" | "menunggu_verifikasi" | "ditolak" | "lunas";
		catatan_tolak?: string;
		wa_url: string;
		gateway_aktif: boolean;
		sesi_aktif: boolean;
		meta?: { pixel_ids: string[]; add_payment_info_id: string; value: number; user: Record<string, string> };
	}
	interface Props { info?: BayarInfo; ref?: string; galat?: string }
	let { info, ref = "", galat = "" }: Props = $props();

	let status = $state(info?.status ?? "belum");
	let file = $state<File | null>(null);
	let previewURL = $state("");
	let mengirim = $state(false);
	let pesanError = $state("");
	let tersalin = $state(false);
	let sesiAktif = $state(info?.sesi_aktif ?? false);
	let membuka = $state(false);
	let manualTerbuka = $state(!(info?.gateway_aktif ?? false) || info?.status === "ditolak");

	// While an online session is open, poll so the page flips to "lunas" by
	// itself after the registrant returns from the Xendit checkout.
	onMount(() => {
		if (initPixel(info?.meta?.pixel_ids)) setPixelUser(info?.meta?.user);
	});

	onMount(() => {
		if (!sesiAktif || status === "lunas") return;
		const mulai = Date.now();
		const t = setInterval(async () => {
			if (Date.now() - mulai > 10 * 60 * 1000) return clearInterval(t);
			try {
				const res = await fetch(`/daftar/bayar/${encodeURIComponent(ref)}/status`, { headers: { Accept: "application/json" } });
				if (!res.ok) return;
				const d = await res.json();
				sesiAktif = d.sesi_aktif;
				if (d.status === "lunas") {
					status = "lunas";
					clearInterval(t);
				}
			} catch {
				/* coba lagi di putaran berikutnya */
			}
		}, 5000);
		return () => clearInterval(t);
	});

	async function bayarOnline() {
		if (membuka) return;
		pesanError = "";
		membuka = true;
		try {
			const res = await fetch(`/daftar/bayar/${encodeURIComponent(ref)}/gateway`, { method: "POST", headers: { Accept: "application/json" } });
			const data = await res.json().catch(() => ({}));
			if (!res.ok || !data.url) {
				pesanError = data.error || "Pembayaran online belum bisa dibuka. Coba lagi atau gunakan transfer manual.";
				membuka = false;
				return;
			}
			window.location.href = data.url;
		} catch {
			pesanError = "Koneksi terputus. Periksa internet lalu coba lagi.";
			membuka = false;
		}
	}

	const MAKS = 2 * 1024 * 1024;

	function pilih(e: Event) {
		pesanError = "";
		const f = (e.currentTarget as HTMLInputElement).files?.[0] ?? null;
		if (previewURL) URL.revokeObjectURL(previewURL);
		previewURL = "";
		file = null;
		if (!f) return;
		if (f.size > MAKS) {
			pesanError = "Ukuran file maksimal 2 MB. Coba kirim tangkapan layar.";
			return;
		}
		file = f;
		if (f.type.startsWith("image/")) previewURL = URL.createObjectURL(f);
	}

	async function salin() {
		if (!info) return;
		try {
			await navigator.clipboard.writeText(info.bank_manual_info);
			tersalin = true;
			setTimeout(() => (tersalin = false), 2000);
		} catch {
			/* clipboard tidak tersedia */
		}
	}

	async function kirim(e: Event) {
		e.preventDefault();
		if (!file || mengirim) return;
		pesanError = "";
		mengirim = true;
		try {
			const body = new FormData();
			body.append("bukti", file);
			body.append("page_url", window.location.href);
			const res = await fetch(`/daftar/bayar/${encodeURIComponent(ref)}/bukti`, { method: "POST", body, headers: { Accept: "application/json" } });
			const data = await res.json().catch(() => ({}));
			if (!res.ok) {
				pesanError = data.error || "Unggahan gagal. Silakan coba lagi.";
				return;
			}
			status = "menunggu_verifikasi";
			if (info?.meta) trackPixel("AddPaymentInfo", { value: info.meta.value, currency: "IDR", content_name: info.program_nama }, info.meta.add_payment_info_id);
		} catch {
			pesanError = "Koneksi terputus. Periksa internet lalu coba lagi.";
		} finally {
			mengirim = false;
		}
	}
</script>

<svelte:head><title>Pembayaran pendaftaran · Ruang Sanad</title></svelte:head>

<main class="min-h-[100dvh] bg-neutral-50 px-4 py-8 dark:bg-neutral-950 sm:py-14">
	<div class="mx-auto w-full max-w-md">
		<div class="mb-6 flex items-center gap-2.5">
			<Logo size={32} />
			<span class="font-semibold text-neutral-900 dark:text-white">Ruang Sanad</span>
		</div>

		{#if galat || !info}
			<section class="rounded-2xl border border-neutral-200/80 bg-white p-6 dark:border-white/[0.06] dark:bg-neutral-925/50" role="alert">
				<div class="flex gap-3">
					<CircleAlert class="mt-0.5 h-5 w-5 shrink-0 text-amber-600 dark:text-amber-400" />
					<p class="text-sm text-neutral-700 dark:text-neutral-300">{galat || "Tautan pembayaran tidak dikenal."} Pastikan tautan disalin lengkap, atau hubungi CS kami.</p>
				</div>
			</section>
		{:else}
			<section class="rounded-2xl border border-neutral-200/80 bg-white shadow-sm dark:border-white/[0.06] dark:bg-neutral-925/50" aria-labelledby="judul-bayar">
				<header class="border-b border-neutral-200/80 p-6 dark:border-white/[0.05]">
					<p class="text-sm text-neutral-600 dark:text-neutral-400">{info.nama_depan ? `Assalamu'alaikum, ${info.nama_depan}` : "Pembayaran pendaftaran"} · {info.lead_ref}</p>
					<h1 id="judul-bayar" class="mt-1 text-lg font-semibold text-neutral-900 dark:text-white">{info.program_nama}</h1>
					<p class="mt-4 text-sm text-neutral-600 dark:text-neutral-400">Total pembayaran</p>
					<p class="text-3xl font-bold tabular-nums tracking-tight text-neutral-900 dark:text-white">{rupiah(info.nominal)}</p>
				</header>

				<div class="p-6">
					{#if status === "lunas"}
						<div class="text-center" role="status">
							<CircleCheck class="mx-auto h-12 w-12 text-green-600 dark:text-green-400" />
							<p class="mt-3 text-lg font-semibold text-neutral-900 dark:text-white">Pembayaran sudah terverifikasi</p>
							<p class="mt-1 text-sm text-neutral-600 dark:text-neutral-400">Jazakumullahu khairan. CS kami akan menghubungi Anda untuk penempatan kelas.</p>
						</div>
					{:else if status === "menunggu_verifikasi"}
						<div class="flex gap-3 rounded-xl bg-sky-500/10 p-4 text-sm text-sky-800 dark:text-sky-300" role="status">
							<Clock class="h-5 w-5 shrink-0" />
							<p>Bukti transfer sudah kami terima dan sedang diperiksa tim keuangan. Biasanya selesai dalam 1×24 jam pada hari kerja.</p>
						</div>
					{:else if !info.izinkan_manual && !info.gateway_aktif}
						<p class="text-sm text-neutral-700 dark:text-neutral-300">Silakan hubungi CS kami untuk menyelesaikan pembayaran.</p>
					{:else}
						{#if info.gateway_aktif}
							{#if sesiAktif}
								<div class="mb-4 flex gap-3 rounded-xl bg-sky-500/10 p-4 text-sm text-sky-800 dark:text-sky-300" role="status">
									<Clock class="h-5 w-5 shrink-0" />
									<p>Menunggu pembayaran Anda. Halaman ini akan berubah otomatis setelah pembayaran diterima.</p>
								</div>
							{/if}
							<button type="button" onclick={bayarOnline} disabled={membuka}
								class="flex min-h-12 w-full items-center justify-center gap-2 rounded-xl bg-brand-600 text-base font-semibold text-white transition hover:bg-brand-700 active:scale-[0.98] disabled:opacity-50 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-500 dark:bg-brand-500">
								<QrCode size="18" /> {membuka ? "Membuka…" : sesiAktif ? "Lanjutkan pembayaran" : "Bayar sekarang"}
							</button>
							<p class="mt-2 text-center text-xs text-neutral-500">QRIS, GoPay, DANA, OVO, ShopeePay, atau virtual account bank. Lunas langsung terverifikasi.</p>
							{#if pesanError && !manualTerbuka}<p class="mt-3 rounded-lg bg-red-500/10 px-3 py-2 text-sm text-red-700 dark:text-red-300" role="alert">{pesanError}</p>{/if}
						{/if}
						{#if info.izinkan_manual}
							{#if info.gateway_aktif}
								<button type="button" onclick={() => (manualTerbuka = !manualTerbuka)} aria-expanded={manualTerbuka}
									class="mt-6 flex w-full items-center justify-between border-t border-neutral-200/80 pt-4 text-sm font-medium text-neutral-700 dark:border-white/[0.06] dark:text-neutral-300">
									Atau transfer manual <ChevronDown size="16" class="transition {manualTerbuka ? 'rotate-180' : ''}" />
								</button>
							{/if}
							{#if manualTerbuka}
								<div class={info.gateway_aktif ? "mt-4" : ""}>
						{#if status === "ditolak"}
							<div class="mb-5 flex gap-3 rounded-xl bg-red-500/10 p-4 text-sm text-red-800 dark:text-red-300" role="alert">
								<CircleAlert class="h-5 w-5 shrink-0" />
								<p>Bukti sebelumnya belum bisa kami terima{info.catatan_tolak ? `: ${info.catatan_tolak}` : ""}. Silakan unggah bukti yang benar.</p>
							</div>
						{/if}
						<h2 class="text-base font-semibold text-neutral-900 dark:text-white">1. Transfer ke rekening</h2>
						<div class="mt-2 flex items-start justify-between gap-3 rounded-xl bg-neutral-100 p-4 dark:bg-neutral-900">
							<p class="whitespace-pre-line text-sm font-medium text-neutral-900 dark:text-white">{info.bank_manual_info || "Info rekening belum tersedia, silakan hubungi CS."}</p>
							{#if info.bank_manual_info}
								<button type="button" onclick={salin} class="inline-flex shrink-0 items-center gap-1 rounded-lg px-2 py-1 text-xs font-semibold text-brand-700 hover:bg-brand-50 dark:text-brand-400 dark:hover:bg-brand-500/10">
									<Copy size="14" /> {tersalin ? "Tersalin" : "Salin"}
								</button>
							{/if}
						</div>
						<p class="mt-2 text-xs text-neutral-500">Transfer tepat {rupiah(info.nominal)} agar mudah dicocokkan.</p>

						<h2 class="mt-7 text-base font-semibold text-neutral-900 dark:text-white">2. Unggah bukti transfer</h2>
						<form onsubmit={kirim} class="mt-2 space-y-3">
							<label class="flex cursor-pointer flex-col items-center justify-center gap-2 rounded-xl border-2 border-dashed border-neutral-300 px-4 py-6 text-center transition hover:border-brand-500 has-[:focus-visible]:ring-2 has-[:focus-visible]:ring-brand-500 dark:border-white/15">
								{#if previewURL}
									<img src={previewURL} alt="Pratinjau bukti transfer" class="max-h-56 rounded-lg object-contain" />
								{:else if file}
									<FileText class="h-8 w-8 text-neutral-500" />
								{:else}
									<Upload class="h-8 w-8 text-neutral-400" />
								{/if}
								<span class="text-sm font-medium text-neutral-800 dark:text-neutral-200">{file ? file.name : "Pilih foto atau PDF"}</span>
								<span class="text-xs text-neutral-500">JPG, PNG, WebP, atau PDF · maks 2 MB</span>
								<input type="file" accept="image/jpeg,image/png,image/webp,application/pdf" onchange={pilih} class="sr-only" />
							</label>
							{#if pesanError}<p class="rounded-lg bg-red-500/10 px-3 py-2 text-sm text-red-700 dark:text-red-300" role="alert">{pesanError}</p>{/if}
							<button type="submit" disabled={!file || mengirim}
								class="w-full min-h-12 rounded-xl bg-brand-600 text-base font-semibold text-white transition hover:bg-brand-700 active:scale-[0.98] disabled:opacity-50 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-500 dark:bg-brand-500">
								{mengirim ? "Mengunggah…" : "Kirim bukti transfer"}
							</button>
						</form>
								</div>
							{/if}
						{/if}
					{/if}

					<a href={info.wa_url} class="mt-6 flex items-center justify-center gap-2 text-sm font-medium text-neutral-600 hover:text-neutral-900 dark:text-neutral-400 dark:hover:text-white">
						<MessageCircle size="16" /> Ada kendala? Chat CS
					</a>
				</div>
			</section>
		{/if}
	</div>
</main>
