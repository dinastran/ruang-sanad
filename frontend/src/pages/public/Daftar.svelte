<script lang="ts">
	import Logo from "@components/Logo.svelte";
	import DaftarFields from "@components/registration/DaftarFields.svelte";
	import { formKosong, rupiah, type JadwalOpsi, type PublicLink } from "@components/registration/types";
	import { CircleAlert, CircleCheck, MessageCircle } from "lucide-svelte";

	interface Props {
		link?: PublicLink;
		jadwal?: JadwalOpsi[];
		form_token?: string;
		galat?: string;
		wa_cs_url?: string;
	}
	let { link, jadwal: jadwalAwal = [], form_token = "", galat = "", wa_cs_url = "" }: Props = $props();

	// Seat counts can be refreshed by a 422 "jadwal penuh" response.
	let jadwal = $state(jadwalAwal);
	// Set when the link was closed while the visitor was filling the form.
	let ditutup = $state<{ pesan: string; wa: string } | null>(null);

	let form = $state(formKosong());
	let errors = $state<Record<string, string>>({});
	let pesanError = $state("");
	let mengirim = $state(false);
	let sukses = $state<{ wa_url: string; lead_ref: string } | null>(null);

	let hargaLabel = $derived.by(() => {
		if (!link) return "";
		if (!link.varian.length) return rupiah(link.harga);
		const hs = link.varian.map((v) => v.harga);
		const lo = Math.min(...hs), hi = Math.max(...hs);
		return lo === hi ? rupiah(lo) : `${rupiah(lo)} – ${rupiah(hi)}`;
	});

	async function kirim(e: Event) {
		e.preventDefault();
		if (!link || mengirim) return;
		errors = {};
		pesanError = "";
		mengirim = true;
		try {
			const res = await fetch(`/daftar/${encodeURIComponent(link.slug)}`, {
				method: "POST",
				headers: { "Content-Type": "application/json", Accept: "application/json" },
				body: JSON.stringify({ ...form, usia: Number(form.usia) || 0, form_token }),
			});
			const data = await res.json().catch(() => ({}));
			if (!res.ok) {
				if (Array.isArray(data.jadwal)) {
					jadwal = data.jadwal;
					form.jadwal_kode = "";
				}
				if (res.status === 404 && data.wa_cs_url) {
					ditutup = { pesan: data.error, wa: data.wa_cs_url };
					return;
				}
				if (data.field) {
					errors = { [data.field]: data.error };
					document.querySelector<HTMLElement>(`[name="${data.field}"]`)?.focus();
				} else {
					pesanError = data.error || "Pendaftaran belum berhasil. Silakan coba lagi.";
				}
				return;
			}
			if (data.next === "bayar" && data.bayar_url) {
				window.location.href = data.bayar_url;
				return;
			}
			sukses = { wa_url: data.wa_url, lead_ref: data.lead_ref };
			setTimeout(() => { if (data.wa_url) window.location.href = data.wa_url; }, 1500);
		} catch {
			pesanError = "Koneksi terputus. Periksa internet lalu coba lagi.";
		} finally {
			mengirim = false;
		}
	}
</script>

<svelte:head><title>{link?.judul ?? "Pendaftaran"} · Ruang Sanad</title></svelte:head>

<main class="min-h-[100dvh] bg-neutral-50 px-4 py-8 dark:bg-neutral-950 sm:py-14">
	<div class="mx-auto w-full max-w-md">
		<div class="mb-6 flex items-center gap-2.5">
			<Logo size={32} />
			<span class="font-semibold text-neutral-900 dark:text-white">Ruang Sanad</span>
		</div>

		{#if ditutup}
			<section class="rounded-2xl border border-neutral-200/80 bg-white p-6 dark:border-white/[0.06] dark:bg-neutral-925/50" role="alert">
				<div class="flex gap-3">
					<CircleAlert class="mt-0.5 h-5 w-5 shrink-0 text-amber-600 dark:text-amber-400" />
					<p class="font-medium text-neutral-900 dark:text-white">{ditutup.pesan}</p>
				</div>
				<a href={ditutup.wa} class="mt-5 flex min-h-12 w-full items-center justify-center gap-2 rounded-xl bg-brand-600 text-base font-semibold text-white transition hover:bg-brand-700 active:scale-[0.98] dark:bg-brand-500">
					<MessageCircle size="18" /> Chat CS
				</a>
			</section>
		{:else if galat || !link}
			<section class="rounded-2xl border border-neutral-200/80 bg-white p-6 dark:border-white/[0.06] dark:bg-neutral-925/50" role="alert">
				<div class="flex gap-3">
					<CircleAlert class="mt-0.5 h-5 w-5 shrink-0 text-amber-600 dark:text-amber-400" />
					<div>
						<p class="font-medium text-neutral-900 dark:text-white">{galat || "Link pendaftaran tidak dikenal."}</p>
						<p class="mt-1 text-sm text-neutral-600 dark:text-neutral-400">Anda tetap bisa mendaftar dengan menghubungi CS kami.</p>
					</div>
				</div>
				{#if wa_cs_url}
					<a href={wa_cs_url} class="mt-5 flex min-h-12 w-full items-center justify-center gap-2 rounded-xl bg-brand-600 text-base font-semibold text-white transition hover:bg-brand-700 active:scale-[0.98] dark:bg-brand-500">
						<MessageCircle size="18" /> Chat CS
					</a>
				{/if}
			</section>
		{:else if sukses}
			<section class="rounded-2xl border border-neutral-200/80 bg-white p-6 text-center shadow-sm dark:border-white/[0.06] dark:bg-neutral-925/50" role="status">
				<CircleCheck class="mx-auto h-12 w-12 text-green-600 dark:text-green-400" />
				<h1 class="mt-3 text-lg font-semibold text-neutral-900 dark:text-white">Data Anda sudah kami terima</h1>
				<p class="mt-1 text-sm text-neutral-600 dark:text-neutral-400">Lanjutkan ke WhatsApp agar CS kami bisa membantu langkah berikutnya. Nomor pendaftaran: <span class="font-medium text-neutral-900 dark:text-white">{sukses.lead_ref}</span></p>
				<a href={sukses.wa_url} class="mt-5 flex min-h-12 w-full items-center justify-center gap-2 rounded-xl bg-brand-600 text-base font-semibold text-white transition hover:bg-brand-700 active:scale-[0.98] dark:bg-brand-500">
					<MessageCircle size="18" /> Lanjut ke WhatsApp
				</a>
				<p class="mt-3 text-xs text-neutral-500">Anda akan diarahkan otomatis…</p>
			</section>
		{:else}
			<section class="rounded-2xl border border-neutral-200/80 bg-white shadow-sm dark:border-white/[0.06] dark:bg-neutral-925/50" aria-labelledby="judul-daftar">
				<header class="border-b border-neutral-200/80 p-6 dark:border-white/[0.05]">
					<p class="text-sm font-medium text-brand-700 dark:text-brand-400">{link.program_nama}</p>
					<h1 id="judul-daftar" class="mt-1 text-xl font-bold tracking-tight text-neutral-900 dark:text-white">{link.judul}</h1>
					{#if link.deskripsi}<p class="mt-2 whitespace-pre-line text-sm text-neutral-600 dark:text-neutral-400">{link.deskripsi}</p>{/if}
					{#if link.after_submit === "bayar"}<p class="mt-3 text-sm text-neutral-700 dark:text-neutral-300">Biaya: <span class="font-semibold tabular-nums text-neutral-900 dark:text-white">{hargaLabel}</span></p>{/if}
				</header>
				<form onsubmit={kirim} class="relative p-6" novalidate={false}>
					<DaftarFields {link} {jadwal} bind:form {errors} />
					{#if pesanError}<p class="mt-5 rounded-lg bg-red-500/10 px-3 py-2 text-sm text-red-700 dark:text-red-300" role="alert">{pesanError}</p>{/if}
					<button type="submit" disabled={mengirim}
						class="mt-7 w-full min-h-12 rounded-xl bg-brand-600 text-base font-semibold text-white transition hover:bg-brand-700 active:scale-[0.98] disabled:opacity-50 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-500 dark:bg-brand-500">
						{mengirim ? "Mengirim…" : link.after_submit === "bayar" ? "Lanjut ke pembayaran" : "Kirim pendaftaran"}
					</button>
					<p class="mt-3 text-center text-xs text-neutral-500">Data Anda hanya dipakai untuk proses pendaftaran di Ruang Sanad.</p>
				</form>
			</section>
		{/if}
	</div>
</main>
