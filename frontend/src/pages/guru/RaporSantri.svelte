<script lang="ts">
	import { inertia, router } from "@inertiajs/svelte";
	import { fly } from "svelte/transition";
	import AppLayout from "@layouts/AppLayout.svelte";
	import RiayahDialog from "@components/riayah/RiayahDialog.svelte";
	import type { Flash, User, RiayahRapor } from "@lib/types";
	import { adaFlashError, formatTanggal, labelPeriode, salinTeks, waLink } from "@lib/riayah";
	import { ArrowLeft, MessageCircle, Copy, CheckCircle, Eye, NotebookPen } from "lucide-svelte";

	interface Props {
		user?: User;
		rapor: RiayahRapor;
		can_write?: boolean;
		flash?: Flash;
		success?: string;
		error?: string;
	}

	let { user, rapor, can_write = false, flash, success, error }: Props = $props();

	let s = $derived(rapor.santri);
	let r = $derived(rapor.rekap);

	let pesanPribadi = $state("");
	let konfirmasi = $state(false);
	let busy = $state(false);
	let disalin = $state(false);

	let panjangPesan = $derived([...pesanPribadi.trim()].length);
	let pesanCukup = $derived(panjangPesan >= rapor.pesan_minimal);

	let baris = $derived.by(() => {
		const out: string[] = [];
		if (r.total === 0) {
			out.push("• Belum ada pertemuan tercatat pada bulan ini.");
		} else {
			const lain = [r.izin && `izin ${r.izin}`, r.sakit && `sakit ${r.sakit}`, r.alpa && `alpa ${r.alpa}`].filter(Boolean).join(", ");
			out.push(`• Kehadiran: ${r.hadir + r.telat} dari ${r.total} pertemuan${lain ? ` (${lain})` : ""}`);
		}
		if (rapor.batas_awal && rapor.batas_akhir && rapor.batas_awal !== rapor.batas_akhir) {
			out.push(`• Materi: ${rapor.batas_awal} → ${rapor.batas_akhir}`);
		} else if (rapor.batas_akhir) {
			out.push(`• Materi terakhir: ${rapor.batas_akhir}`);
		}
		return out;
	});

	let teksRapor = $derived(
		[
			`Assalamu'alaikum ${s.nama},`,
			"",
			`Berikut ringkasan belajar ananda bulan ${labelPeriode(rapor.periode)}${s.nama_kelas ? ` di kelas ${s.nama_kelas}` : ""}:`,
			...baris,
			"",
			pesanPribadi.trim() || "[pesan pribadi dari guru]",
			"",
			"Barakallahu fiik.",
			user?.name ?? "",
		].join("\n").trim(),
	);

	let link = $derived(waLink(s.no_wa, teksRapor));

	function gantiPeriode(periode: string) {
		router.get(`/app/guru/santri/${s.id}/rapor`, { periode });
	}

	function bukaWA() {
		if (!pesanCukup || !link) return;
		window.open(link, "_blank", "noopener");
		konfirmasi = true;
	}

	async function salin() {
		if (!pesanCukup) return;
		disalin = await salinTeks(teksRapor);
		if (disalin) setTimeout(() => (disalin = false), 2000);
	}

	function catatTerkirim() {
		if (busy) return;
		busy = true;
		router.post(`/app/guru/santri/${s.id}/kontak`, { media: "wa", jenis: "rapor", periode: rapor.periode, catatan: pesanPribadi.trim() }, {
			// Jika ditolak, server kembali ke halaman rapor; state dipertahankan
			// agar pesan pribadi guru tidak hilang.
			preserveState: true,
			onSuccess: (page) => {
				if (adaFlashError(page)) konfirmasi = false;
			},
			onFinish: () => (busy = false),
		});
	}

	const statusStyle: Record<string, string> = {
		hadir: "bg-green-500/10 text-green-700 dark:text-green-400",
		telat: "bg-sky-500/10 text-sky-700 dark:text-sky-400",
		izin: "bg-blue-500/10 text-blue-700 dark:text-blue-400",
		sakit: "bg-purple-500/10 text-purple-700 dark:text-purple-400",
		alpa: "bg-red-500/10 text-red-700 dark:text-red-400",
	};
</script>

<svelte:head><title>Rapor {s.nama}</title></svelte:head>

<AppLayout {user} group="guru-riayah">
	<div class="pt-6 pb-8 border-b border-neutral-200/80 dark:border-white/[0.04]">
		<div class="max-w-5xl mx-auto px-4 sm:px-6">
			<a href={`/app/guru/santri/${s.id}`} use:inertia class="inline-flex min-h-10 items-center gap-1.5 text-sm font-medium text-neutral-600 dark:text-neutral-400 hover:text-brand-700 dark:hover:text-brand-300">
				<ArrowLeft class="w-4 h-4" /> {s.nama}
			</a>
			<div class="mt-2 flex flex-wrap items-end justify-between gap-3">
				<div>
					<h1 class="text-2xl sm:text-3xl font-bold text-neutral-900 dark:text-white tracking-tight">Rapor bulanan</h1>
					<p class="mt-1 text-sm text-neutral-600 dark:text-neutral-400">{s.nama}{s.nama_kelas ? ` · ${s.nama_kelas}` : ""}</p>
				</div>
				<label class="flex items-center gap-2 text-sm">
					<span class="text-neutral-500">Periode</span>
					<select value={rapor.periode} onchange={(e) => gantiPeriode(e.currentTarget.value)} class="rounded-xl border border-neutral-200 bg-white px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-400/40 dark:border-neutral-800 dark:bg-neutral-925">
						{#each rapor.periode_opsi as p (p)}<option value={p}>{labelPeriode(p)}</option>{/each}
					</select>
				</label>
			</div>
		</div>
	</div>

	<div class="max-w-5xl mx-auto px-4 sm:px-6 py-8 space-y-6">
		{#if success || flash?.success}
			<div class="bg-green-500/10 border border-green-500/20 text-green-700 dark:text-green-400 rounded-2xl p-4 text-sm font-medium" in:fly={{ y: 10, duration: 200 }}>{success || flash?.success}</div>
		{/if}
		{#if error || flash?.error}
			<div class="bg-red-500/10 border border-red-500/20 text-red-600 dark:text-red-400 rounded-2xl p-4 text-sm font-medium" in:fly={{ y: 10, duration: 200 }}>{error || flash?.error}</div>
		{/if}
		{#if rapor.terkirim_pada.length > 0}
			<div class="flex items-center gap-2.5 rounded-2xl border border-green-500/20 bg-green-500/5 px-4 py-3 text-sm text-green-800 dark:text-green-300">
				<CheckCircle class="w-4 h-4 shrink-0" /> Rapor {labelPeriode(rapor.periode)} sudah dikirim pada {rapor.terkirim_pada.map(formatTanggal).join(", ")}.
			</div>
		{/if}

		<div class="grid gap-6 lg:grid-cols-[1fr_22rem]">
			<div class="space-y-6 min-w-0">
				{#if can_write}
					<section class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-5" aria-labelledby="pesan-title">
						<h2 id="pesan-title" class="flex items-center gap-2 text-sm font-semibold text-neutral-900 dark:text-white"><NotebookPen class="w-4 h-4 text-brand-600 dark:text-brand-400" /> Pesan pribadi dari guru</h2>
						<p class="mt-1 text-xs text-neutral-500">Wajib diisi. Tulis hal spesifik tentang ananda bulan ini: kemajuan, hal yang perlu diperbaiki, atau doa.</p>
						<label for="pesan" class="sr-only">Pesan pribadi</label>
						<textarea id="pesan" bind:value={pesanPribadi} rows="5" class="mt-3 w-full rounded-xl border border-neutral-200 bg-white px-3 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-brand-400/40 dark:border-neutral-800 dark:bg-neutral-925" placeholder="Mis. Bacaan mad ananda sudah jauh lebih rapi. Bulan depan kita fokus di makharijul huruf ya."></textarea>
						<p class="mt-1 text-xs {pesanCukup ? 'text-green-700 dark:text-green-400' : 'text-neutral-500'}">{panjangPesan}/{rapor.pesan_minimal} karakter minimal</p>

						{#if !s.no_wa}
							<p class="mt-3 rounded-xl bg-amber-500/10 px-3 py-2 text-sm text-amber-800 dark:text-amber-300">Nomor WA santri belum tercatat. Salin teks rapor lalu kirim manual.</p>
						{/if}
						<div class="mt-4 flex flex-wrap justify-end gap-2">
							<button type="button" onclick={salin} disabled={!pesanCukup} class="inline-flex min-h-10 items-center gap-1.5 whitespace-nowrap rounded-xl border border-neutral-200 px-4 text-sm font-medium text-neutral-700 hover:border-brand-400/40 disabled:opacity-50 dark:border-neutral-700 dark:text-neutral-300">
								{#if disalin}<CheckCircle class="h-4 w-4 text-green-600" /> Tersalin{:else}<Copy class="h-4 w-4" /> Salin teks{/if}
							</button>
							{#if s.no_wa}
								<button type="button" onclick={bukaWA} disabled={!pesanCukup} class="inline-flex min-h-10 items-center gap-1.5 whitespace-nowrap rounded-xl bg-brand-600 px-4 text-sm font-semibold text-white hover:bg-brand-700 disabled:opacity-50 focus:outline-none focus-visible:ring-2 focus-visible:ring-brand-400/60">
									<MessageCircle class="h-4 w-4" /> Kirim via WhatsApp
								</button>
							{:else}
								<button type="button" onclick={() => (konfirmasi = true)} disabled={!pesanCukup} class="min-h-10 whitespace-nowrap rounded-xl bg-brand-600 px-4 text-sm font-semibold text-white hover:bg-brand-700 disabled:opacity-50">Sudah dikirim manual</button>
							{/if}
						</div>
					</section>
				{:else}
					<div class="flex items-center gap-2.5 rounded-2xl border border-neutral-200 dark:border-neutral-800 bg-neutral-50 dark:bg-neutral-900/40 px-4 py-3 text-sm text-neutral-600 dark:text-neutral-400">
						<Eye class="w-4 h-4 shrink-0" /> Mode baca: rapor dikirim oleh guru pengampu.
					</div>
				{/if}

				<section aria-labelledby="bahan-title">
					<h2 id="bahan-title" class="text-base font-semibold text-neutral-900 dark:text-white">Bahan dari pertemuan bulan ini</h2>
					<p class="mt-0.5 text-xs text-neutral-500">Catatan per pertemuan hanya untuk referensi guru dan tidak ikut terkirim.</p>
					{#if rapor.pertemuan.length === 0}
						<p class="mt-3 rounded-2xl border border-dashed border-neutral-300 dark:border-neutral-700 p-8 text-center text-sm text-neutral-500">Tidak ada pertemuan selesai pada {labelPeriode(rapor.periode)}.</p>
					{:else}
						<ul class="mt-3 divide-y divide-neutral-100 dark:divide-neutral-800 rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50">
							{#each rapor.pertemuan as p, i (i)}
								<li class="px-4 py-3 text-sm">
									<div class="flex flex-wrap items-center gap-2">
										<span class="text-xs font-medium text-neutral-500">{formatTanggal(p.tanggal)}</span>
										<span class="font-medium text-neutral-900 dark:text-white">{p.judul}</span>
										<span class="rounded-full px-2 py-0.5 text-[11px] font-semibold capitalize {statusStyle[p.status ?? ''] ?? 'bg-neutral-500/10 text-neutral-600'}">{p.status}</span>
									</div>
									{#if p.batas_materi}<p class="mt-1 text-neutral-600 dark:text-neutral-400">Batas materi: {p.batas_materi}</p>{/if}
									{#if p.isi}<p class="mt-1 text-neutral-700 dark:text-neutral-300 whitespace-pre-line">“{p.isi}”</p>{/if}
								</li>
							{/each}
						</ul>
					{/if}
				</section>
			</div>

			<aside class="lg:sticky lg:top-20 self-start" aria-labelledby="preview-title">
				<h2 id="preview-title" class="text-sm font-semibold text-neutral-900 dark:text-white">Pratinjau pesan</h2>
				<pre class="mt-2 whitespace-pre-wrap break-words rounded-2xl border border-green-500/20 bg-green-500/5 p-4 font-sans text-sm leading-relaxed text-neutral-800 dark:text-neutral-200">{teksRapor}</pre>
			</aside>
		</div>
	</div>

	{#if konfirmasi}
		<RiayahDialog title="Rapor sudah terkirim?" subtitle={`${s.nama} · ${labelPeriode(rapor.periode)}`} {busy} onclose={() => (konfirmasi = false)}>
			<p class="text-sm text-neutral-700 dark:text-neutral-300">Pengiriman rapor hanya dicatat setelah Anda konfirmasi. Pesan pribadi Anda ikut tersimpan di riwayat santri.</p>
			<div class="mt-4 flex flex-wrap justify-end gap-2">
				<button type="button" onclick={() => (konfirmasi = false)} disabled={busy} class="min-h-10 whitespace-nowrap rounded-xl px-4 text-sm font-medium text-neutral-600 hover:bg-neutral-100 disabled:opacity-50 dark:text-neutral-300 dark:hover:bg-neutral-800">Belum</button>
				<button type="button" onclick={catatTerkirim} disabled={busy} class="min-h-10 whitespace-nowrap rounded-xl bg-brand-600 px-4 text-sm font-semibold text-white hover:bg-brand-700 disabled:opacity-50">{busy ? "Mencatat..." : "Sudah, catat terkirim"}</button>
			</div>
		</RiayahDialog>
	{/if}
</AppLayout>
