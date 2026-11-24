<script lang="ts">
	import Logo from "@components/Logo.svelte";
	import DaftarFields from "@components/registration/DaftarFields.svelte";
	import { formKosong, rupiah, type DaftarForm, type JadwalOpsi, type PublicLink } from "@components/registration/types";
	import { firePixelEvents, initPixel, setPixelUser, trackPixel } from "@lib/metaPixel";
	import { onMount } from "svelte";
	import { ArrowLeft, ArrowRight, CircleAlert, CircleCheck, Clock, LoaderCircle, LockKeyhole, MessageCircle, ShieldCheck } from "lucide-svelte";

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

	// Draft survives an accidental close/refresh (per device, this link only).
	const kunciDraft = () => `rs-daftar-${link?.slug ?? ""}`;
	function muatDraft(): DaftarForm {
		const kosong = formKosong();
		try {
			const raw = link ? localStorage.getItem(kunciDraft()) : null;
			if (raw) return { ...kosong, ...JSON.parse(raw), website: "" };
		} catch {
			/* storage diblokir: mulai kosong */
		}
		return kosong;
	}
	function hapusDraft() {
		try {
			localStorage.removeItem(kunciDraft());
		} catch {
			/* abaikan */
		}
	}

	let form = $state(muatDraft());
	let errors = $state<Record<string, string>>({});
	let pesanError = $state("");
	let mengirim = $state(false);
	let sukses = $state<{ wa_url: string; lead_ref: string } | null>(null);
	let hitungMundur = $state(0);

	$effect(() => {
		if (!link || sukses) return;
		const { website: _, ...simpan } = form;
		try {
			localStorage.setItem(kunciDraft(), JSON.stringify(simpan));
		} catch {
			/* abaikan */
		}
	});

	// Meta Pixel: PageView + ViewContent (eventID shared with the CAPI copy).
	// onMount, not $effect: Inertia re-sends the page props right after the
	// first render, which re-ran an effect and fired ViewContent twice.
	onMount(() => {
		if (!link?.meta || galat) return;
		if (initPixel(link.meta.pixel_ids)) {
			trackPixel("ViewContent", { content_name: link.program_nama, content_category: "pendaftaran" }, link.meta.view_event_id);
		}
	});

	let hargaLabel = $derived.by(() => {
		if (!link) return "";
		if (!link.varian.length) return link.harga > 0 ? rupiah(link.harga) : "";
		const hs = link.varian.map((v) => v.harga);
		const lo = Math.min(...hs), hi = Math.max(...hs);
		return lo === hi ? rupiah(lo) : `mulai ${rupiah(lo)}`;
	});
	let hargaTerpilih = $derived.by(() => {
		if (!link) return "";
		const v = link.varian.find((x) => x.id === Number(form.varian_id));
		return v ? rupiah(v.harga) : hargaLabel;
	});

	// Description lines starting with "-", "•", "*" or "✓" become a benefit list.
	let isi = $derived.by(() => {
		const baris = (link?.deskripsi ?? "").split("\n").map((b) => b.trim()).filter(Boolean);
		const poin = baris.filter((b) => /^[-•*✓✔]\s*/.test(b)).map((b) => b.replace(/^[-•*✓✔]\s*/, ""));
		const teks = baris.filter((b) => !/^[-•*✓✔]\s*/.test(b));
		return { teks, poin };
	});

	let adaLangkah2 = $derived(
		!!link && (link.mode === "full" || link.butuh_jadwal || link.varian.length > 0 ||
			Object.values(link.field_config ?? {}).filter((r) => r.tampil).length > 2),
	);
	let langkah = $state<1 | 2>(1);
	let estimasi = $derived(link?.mode === "full" ? "± 2 menit" : "± 1 menit");
	let ctaUtama = $derived(link?.after_submit === "bayar" ? "Lanjut ke pembayaran" : "Kirim & lanjut ke WhatsApp");

	const reWA = /^(?:\+?62|0)8\d{7,12}$/;
	function cekField(f: string): string {
		if (f === "nama") {
			const n = form.nama.trim().replace(/\s+/g, " ");
			return n.length < 2 ? "Tulis nama lengkap Anda" : "";
		}
		if (f === "no_wa") {
			const n = form.no_wa.replace(/[\s\-().]/g, "");
			if (!n) return "Nomor WhatsApp wajib diisi";
			return reWA.test(n) ? "" : "Nomor belum sesuai. Contoh: 0812 3456 7890";
		}
		if (f === "jenis_kelamin") return form.jenis_kelamin ? "" : "Pilih ikhwan atau akhwat";
		return "";
	}
	function blurField(f: string) {
		// Validate on blur only once something was typed (no error on first focus-out of an empty form).
		if (f === "nama" && !form.nama) return;
		if (f === "no_wa" && !form.no_wa) return;
		const pesan = cekField(f);
		const { [f]: _, ...rest } = errors;
		errors = pesan ? { ...rest, [f]: pesan } : rest;
	}
	function fokus(field: string) {
		requestAnimationFrame(() => document.querySelector<HTMLElement>(`[name="${field}"]`)?.focus());
	}
	function cekLangkah1(): boolean {
		const e: Record<string, string> = {};
		for (const f of ["nama", "no_wa", "jenis_kelamin"]) {
			const p = cekField(f);
			if (p) e[f] = p;
		}
		errors = e;
		const pertama = Object.keys(e)[0];
		if (pertama) fokus(pertama);
		return !pertama;
	}
	// Step-1 events have no server event_id: fire them once per page view.
	let langkah1Terkirim = false;
	function lanjut() {
		if (!cekLangkah1()) return;
		langkah = 2;
		if (!langkah1Terkirim) {
			langkah1Terkirim = true;
			firePixelEvents(link?.meta?.langkah1, { content_name: link?.program_nama });
		}
		document.getElementById("form-daftar")?.scrollIntoView({ behavior: matchMedia("(prefers-reduced-motion: reduce)").matches ? "auto" : "smooth", block: "start" });
	}

	const langkah1Field = new Set(["nama", "no_wa", "jenis_kelamin"]);

	async function kirim(e: Event) {
		e.preventDefault();
		if (!link || mengirim) return;
		if (adaLangkah2 && langkah === 1) return lanjut();
		if (!cekLangkah1()) {
			langkah = 1;
			return;
		}
		errors = {};
		pesanError = "";
		mengirim = true;
		try {
			const res = await fetch(`/daftar/${encodeURIComponent(link.slug)}`, {
				method: "POST",
				headers: { "Content-Type": "application/json", Accept: "application/json" },
				body: JSON.stringify({ ...form, usia: Number(form.usia) || 0, form_token, page_url: window.location.href }),
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
					if (adaLangkah2) langkah = langkah1Field.has(data.field) ? 1 : 2;
					fokus(data.field);
				} else {
					pesanError = data.error || "Pendaftaran belum berhasil. Silakan coba lagi.";
				}
				return;
			}
			hapusDraft();
			if (data.meta) {
				setPixelUser(data.meta.user);
				firePixelEvents(data.meta.events, { content_name: link.program_nama });
			}
			if (data.next === "bayar" && data.bayar_url) {
				// Give the Pixel a moment to send before leaving the page.
				setTimeout(() => (window.location.href = data.bayar_url), data.meta ? 400 : 0);
				return;
			}
			sukses = { wa_url: data.wa_url, lead_ref: data.lead_ref };
			window.scrollTo({ top: 0 });
			if (data.wa_url) {
				hitungMundur = 3;
				const t = setInterval(() => {
					hitungMundur -= 1;
					if (hitungMundur <= 0) {
						clearInterval(t);
						window.location.href = data.wa_url;
					}
				}, 1000);
			}
		} catch {
			pesanError = "Koneksi terputus. Periksa internet lalu coba lagi.";
		} finally {
			mengirim = false;
		}
	}

	const tombolUtama =
		"inline-flex min-h-13 w-full cursor-pointer items-center justify-center gap-2 rounded-xl bg-brand-700 px-5 text-base font-semibold text-white shadow-[0_1px_2px_rgb(0_0_0/0.08),0_6px_16px_-6px_var(--color-brand-700)] transition-[background-color,transform] duration-150 hover:bg-brand-800 active:scale-[0.98] disabled:cursor-wait disabled:opacity-70 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-600 dark:bg-brand-600 dark:hover:bg-brand-700";
	const kartu = "rounded-2xl border border-neutral-200/80 bg-white shadow-sm dark:border-white/[0.07] dark:bg-neutral-925/60";
</script>

<svelte:head>
	<title>{link?.judul ?? "Pendaftaran"} · Ruang Sanad</title>
	{#if link?.deskripsi}<meta name="description" content={link.deskripsi.slice(0, 155)} />{/if}
	<meta name="theme-color" content="#0e7490" />
</svelte:head>

{#snippet bantuanCS()}
	{#if wa_cs_url}
		<a href={wa_cs_url} class="inline-flex min-h-11 items-center gap-1.5 rounded-lg px-1 text-sm font-medium text-neutral-600 underline-offset-4 transition-colors hover:text-brand-700 hover:underline dark:text-neutral-400 dark:hover:text-brand-400">
			<MessageCircle size={16} aria-hidden="true" /> Ada pertanyaan dulu? Tanya CS
		</a>
	{/if}
{/snippet}

<main class="relative min-h-[100dvh] bg-neutral-50 dark:bg-neutral-950">
	{#if ditutup || galat || !link}
		<div class="mx-auto w-full max-w-md px-4 py-10">
			<div class="mb-6 flex items-center gap-2.5"><Logo size={32} /><span class="font-semibold text-neutral-900 dark:text-white">Ruang Sanad</span></div>
			<section class="{kartu} p-6" role="alert">
				<div class="flex gap-3">
					<CircleAlert class="mt-0.5 h-5 w-5 shrink-0 text-amber-600 dark:text-amber-400" aria-hidden="true" />
					<div>
						<p class="font-medium text-neutral-900 dark:text-white">{ditutup?.pesan || galat || "Link pendaftaran tidak dikenal."}</p>
						{#if !ditutup}<p class="mt-1 text-sm text-neutral-600 dark:text-neutral-400">Anda tetap bisa mendaftar dengan menghubungi CS kami.</p>{/if}
					</div>
				</div>
				{#if ditutup?.wa || wa_cs_url}
					<a href={ditutup?.wa || wa_cs_url} class="{tombolUtama} mt-5"><MessageCircle size={18} aria-hidden="true" /> Chat CS di WhatsApp</a>
				{/if}
			</section>
		</div>
	{:else if sukses}
		<div class="mx-auto w-full max-w-md px-4 py-10">
			<div class="mb-6 flex items-center gap-2.5"><Logo size={32} /><span class="font-semibold text-neutral-900 dark:text-white">Ruang Sanad</span></div>
			<section class="{kartu} p-6 text-center sm:p-8" role="status" aria-live="polite">
				<span class="mx-auto flex h-16 w-16 items-center justify-center rounded-full bg-green-500/10">
					<CircleCheck class="h-9 w-9 text-green-600 dark:text-green-400" aria-hidden="true" />
				</span>
				<h1 class="mt-4 text-xl font-bold tracking-tight text-neutral-900 text-balance dark:text-white">Alhamdulillah, pendaftaran Anda sudah kami terima</h1>
				<p class="mt-2 text-sm text-neutral-600 dark:text-neutral-400">Satu langkah lagi: kirim pesan WhatsApp yang sudah kami siapkan agar CS bisa membantu langkah berikutnya.</p>
				<p class="mt-4 inline-flex items-center gap-2 rounded-lg bg-neutral-100 px-3 py-1.5 text-sm dark:bg-white/5">
					<span class="text-neutral-500">Nomor pendaftaran</span>
					<span class="font-mono font-semibold text-neutral-900 dark:text-white">{sukses.lead_ref}</span>
				</p>
				{#if sukses.wa_url}
					<a href={sukses.wa_url} class="{tombolUtama} mt-6"><MessageCircle size={18} aria-hidden="true" /> Buka WhatsApp sekarang</a>
					<p class="mt-3 text-xs text-neutral-500" aria-live="polite">
						{hitungMundur > 0 ? `Membuka WhatsApp otomatis dalam ${hitungMundur} detik…` : "Jika WhatsApp tidak terbuka, ketuk tombol di atas."}
					</p>
				{/if}
			</section>
		</div>
	{:else}
		<!-- Decorative brand wash behind the hero -->
		<div class="pointer-events-none absolute inset-x-0 top-0 h-[420px] bg-gradient-to-b from-brand-100/70 via-brand-50/40 to-transparent dark:from-brand-950/50 dark:via-brand-950/10" aria-hidden="true"></div>

		<div class="relative mx-auto w-full max-w-5xl px-4 pt-5 sm:px-6">
			<div class="flex items-center gap-2.5">
				<Logo size={30} />
				<span class="font-semibold text-neutral-900 dark:text-white">Ruang Sanad</span>
			</div>
		</div>

		<div class="relative mx-auto grid w-full max-w-5xl gap-6 px-4 pb-40 pt-5 sm:px-6 lg:grid-cols-[minmax(0,1fr)_minmax(0,440px)] lg:gap-12 lg:pb-16 lg:pt-10">
			<!-- Offer -->
			<section class="lg:sticky lg:top-8 lg:self-start" aria-labelledby="judul-daftar">
				<p class="inline-flex items-center rounded-full bg-brand-700/10 px-3 py-1 text-xs font-semibold text-brand-800 dark:bg-brand-400/10 dark:text-brand-300">{link.program_nama}</p>
				<h1 id="judul-daftar" class="mt-2.5 text-[1.625rem] font-bold leading-[1.15] tracking-tight text-neutral-900 text-balance sm:text-4xl dark:text-white">{link.judul}</h1>
				{#each isi.teks as p}<p class="mt-2 max-w-prose lg:mt-3 text-[15px] leading-relaxed text-neutral-600 dark:text-neutral-400">{p}</p>{/each}
				{#if isi.poin.length}
					<ul class="mt-4 space-y-2 lg:mt-5 lg:space-y-2.5">
						{#each isi.poin as p}
							<li class="flex gap-2.5 text-[15px] leading-snug text-neutral-800 dark:text-neutral-200">
								<CircleCheck class="mt-px h-5 w-5 shrink-0 text-brand-700 dark:text-brand-400" aria-hidden="true" />{p}
							</li>
						{/each}
					</ul>
				{/if}
				{#if link.after_submit === "bayar" && hargaLabel}
					<p class="mt-4 flex items-baseline gap-2 lg:mt-6">
						<span class="text-sm text-neutral-500 dark:text-neutral-400">Biaya</span>
						<span class="text-xl font-bold tabular-nums tracking-tight text-neutral-900 lg:text-2xl dark:text-white">{hargaLabel}</span>
					</p>
				{/if}
				<ul class="mt-4 flex flex-wrap gap-x-4 gap-y-1.5 text-[13px] text-neutral-600 dark:text-neutral-400 lg:mt-6 lg:gap-x-5" aria-label="Keterangan">
					<li class="flex items-center gap-1.5"><Clock size={15} class="text-brand-700 dark:text-brand-400" aria-hidden="true" />Isi {estimasi}</li>
					<li class="flex items-center gap-1.5"><MessageCircle size={15} class="text-brand-700 dark:text-brand-400" aria-hidden="true" />Dibantu CS via WhatsApp</li>
					<li class="flex items-center gap-1.5"><ShieldCheck size={15} class="text-brand-700 dark:text-brand-400" aria-hidden="true" />Data aman</li>
				</ul>
				<div class="mt-4 hidden lg:block">{@render bantuanCS()}</div>
			</section>

			<!-- Form -->
			<section id="form-daftar" class="{kartu} scroll-mt-4" aria-label="Formulir pendaftaran">
				<form onsubmit={kirim} novalidate class="relative">
					<div class="border-b border-neutral-100 px-5 pb-4 pt-5 dark:border-white/[0.06] sm:px-6">
						{#if adaLangkah2}
							<div class="flex items-center justify-between text-xs font-medium">
								<span class="text-brand-800 dark:text-brand-300">Langkah {langkah} dari 2</span>
								<span class="text-neutral-500">{langkah === 1 ? "Data diri" : "Pilihan kelas"}</span>
							</div>
							<div class="mt-2 h-1.5 overflow-hidden rounded-full bg-neutral-100 dark:bg-white/5" role="progressbar" aria-valuemin={1} aria-valuemax={2} aria-valuenow={langkah} aria-label="Kemajuan pengisian">
								<div class="h-full rounded-full bg-brand-600 transition-[width] duration-300 ease-out motion-reduce:transition-none" style="width: {langkah === 1 ? 50 : 100}%"></div>
							</div>
						{/if}
						<h2 class="{adaLangkah2 ? 'mt-4' : ''} text-lg font-semibold tracking-tight text-neutral-900 dark:text-white">
							{#if !adaLangkah2 || langkah === 1}Daftar sekarang{:else}Hampir selesai{/if}
						</h2>
						<p class="mt-0.5 text-sm text-neutral-500 dark:text-neutral-400">
							{#if !adaLangkah2}Isi data singkat di bawah, lalu CS kami akan menghubungi Anda.
							{:else if langkah === 1}Mulai dengan data singkat — gratis, tanpa komitmen di langkah ini.
							{:else}Pilih yang paling sesuai, lalu kirim.{/if}
						</p>
					</div>

					<div class="px-5 py-5 sm:px-6">
						{#if adaLangkah2 && langkah === 2}
							<button type="button" class="mb-4 inline-flex min-h-11 cursor-pointer items-center gap-1.5 rounded-lg pr-2 text-sm font-medium text-neutral-600 transition-colors hover:text-neutral-900 dark:text-neutral-400 dark:hover:text-white" onclick={() => (langkah = 1)}>
								<ArrowLeft size={16} aria-hidden="true" /> Ubah data diri
								<span class="ml-1 max-w-40 truncate font-normal text-neutral-400">· {form.nama}</span>
							</button>
						{/if}
						<DaftarFields {link} {jadwal} bind:form {errors} bagian={!adaLangkah2 ? "semua" : langkah === 1 ? "diri" : "pilihan"} onblurfield={blurField} />
						{#if pesanError}<p class="mt-5 flex gap-2 rounded-xl bg-red-500/10 px-4 py-3 text-sm text-red-700 dark:text-red-300" role="alert"><CircleAlert size={18} class="mt-px shrink-0" aria-hidden="true" />{pesanError}</p>{/if}
					</div>

					<!-- Sticky action bar on mobile, inline on desktop -->
					<div class="fixed inset-x-0 bottom-0 z-20 border-t border-neutral-200/80 bg-white/95 px-4 pb-[max(0.75rem,env(safe-area-inset-bottom))] pt-3 backdrop-blur supports-[backdrop-filter]:bg-white/85 dark:border-white/10 dark:bg-neutral-950/90 lg:static lg:rounded-b-2xl lg:border-neutral-100 lg:bg-neutral-50/60 lg:px-6 lg:pb-6 lg:pt-4 lg:backdrop-blur-none dark:lg:border-white/[0.06] dark:lg:bg-white/[0.02]">
						<div class="mx-auto w-full max-w-md lg:max-w-none">
							{#if link.after_submit === "bayar" && hargaTerpilih && (!adaLangkah2 || langkah === 2)}
								<p class="mb-2 flex items-baseline justify-between text-sm">
									<span class="text-neutral-500 dark:text-neutral-400">{form.varian_id ? "Total" : "Biaya"}</span>
									<span class="font-semibold tabular-nums text-neutral-900 dark:text-white">{hargaTerpilih}</span>
								</p>
							{/if}
							{#if adaLangkah2 && langkah === 1}
								<button type="submit" class={tombolUtama}>Lanjut <ArrowRight size={18} aria-hidden="true" /></button>
							{:else}
								<button type="submit" disabled={mengirim} class={tombolUtama} aria-live="polite">
									{#if mengirim}<LoaderCircle size={18} class="animate-spin motion-reduce:animate-none" aria-hidden="true" /> Mengirim…{:else}{ctaUtama} <ArrowRight size={18} aria-hidden="true" />{/if}
								</button>
							{/if}
							<p class="mt-2 flex items-center justify-center gap-1.5 text-center text-xs text-neutral-500 dark:text-neutral-400">
								<LockKeyhole size={12} aria-hidden="true" /> Data Anda hanya dipakai untuk proses pendaftaran di Ruang Sanad.
							</p>
						</div>
					</div>
				</form>
			</section>

			<div class="-mt-2 text-center lg:hidden">{@render bantuanCS()}</div>
		</div>
	{/if}
</main>
