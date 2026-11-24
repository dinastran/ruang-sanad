<script lang="ts">
	import Logo from "@components/Logo.svelte";
	import { CircleAlert, CircleCheck, Clock } from "lucide-svelte";

	interface SesiInfo {
		program: string;
		ke: number;
		judul: string;
		tanggal: string;
		jam: string;
		status: "buka" | "belum" | "tutup";
		buka_pukul: string;
	}
	interface Props { token: string; sesi?: SesiInfo; galat?: string }

	let { token, sesi, galat = "" }: Props = $props();

	const STORAGE_KEY = "hadir-no-wa";
	function nomorTersimpan(): string {
		try {
			return localStorage.getItem(STORAGE_KEY) ?? "";
		} catch {
			return "";
		}
	}
	let noWA = $state(nomorTersimpan());
	let mengirim = $state(false);
	let pesanError = $state("");
	let hasil = $state<{ nama: string; sudah_tercatat: boolean; kelas?: string; zoom_url?: string } | null>(null);

	let tanggalPanjang = $derived.by(() => {
		if (!sesi) return "";
		const d = new Date(sesi.tanggal + "T00:00:00");
		return Number.isNaN(d.getTime()) ? sesi.tanggal : d.toLocaleDateString("id-ID", { weekday: "long", day: "numeric", month: "long", year: "numeric" });
	});

	async function kirim(e: Event) {
		e.preventDefault();
		if (mengirim) return;
		pesanError = "";
		mengirim = true;
		try {
			const res = await fetch(`/hadir/${encodeURIComponent(token)}`, {
				method: "POST",
				headers: { "Content-Type": "application/json", Accept: "application/json" },
				body: JSON.stringify({ no_wa: noWA }),
			});
			const data = await res.json().catch(() => ({}));
			if (!res.ok) {
				pesanError = data.error || data.message || "Gagal mencatat kehadiran. Coba lagi.";
				return;
			}
			hasil = data;
			try {
				localStorage.setItem(STORAGE_KEY, noWA.trim());
			} catch {
				/* storage unavailable */
			}
		} catch {
			pesanError = "Koneksi terputus. Periksa internet lalu coba lagi.";
		} finally {
			mengirim = false;
		}
	}
</script>

<svelte:head><title>Absensi {sesi?.program ?? "program"} · Ruang Sanad</title></svelte:head>

<main class="min-h-screen bg-neutral-50 px-4 py-10 dark:bg-neutral-950 sm:py-16">
	<div class="mx-auto w-full max-w-sm">
		<div class="mb-6 flex items-center gap-2.5">
			<Logo size={32} />
			<span class="font-semibold text-neutral-900 dark:text-white">Ruang Sanad</span>
		</div>

		{#if galat || !sesi}
			<section class="rounded-2xl border border-red-500/25 bg-white p-6 dark:bg-neutral-925/50" role="alert">
				<div class="flex gap-3">
					<CircleAlert class="mt-0.5 h-5 w-5 shrink-0 text-red-600 dark:text-red-400" />
					<p class="text-sm text-neutral-700 dark:text-neutral-300">{galat || "Link absensi tidak dikenal."}</p>
				</div>
			</section>
		{:else}
			<section class="rounded-2xl border border-neutral-200/80 bg-white shadow-sm dark:border-white/[0.06] dark:bg-neutral-925/50" aria-labelledby="judul-absen">
				<div class="border-b border-neutral-200/80 p-6 dark:border-white/[0.05]">
					<p class="text-sm font-medium text-brand-700 dark:text-brand-400">Absensi sesi {sesi.ke}</p>
					<h1 id="judul-absen" class="mt-1 text-xl font-bold tracking-tight text-neutral-900 dark:text-white">{sesi.program}</h1>
					<p class="mt-1 text-sm text-neutral-600 dark:text-neutral-400">{tanggalPanjang}{sesi.jam ? ` · ${sesi.jam} WIB` : ""}</p>
					{#if sesi.judul}<p class="mt-0.5 text-sm text-neutral-500">{sesi.judul}</p>{/if}
				</div>

				<div class="p-6">
					{#if hasil}
						<div class="text-center" role="status">
							<CircleCheck class="mx-auto h-12 w-12 text-green-600 dark:text-green-400" />
							<p class="mt-3 text-lg font-semibold text-neutral-900 dark:text-white">Jazakumullahu khairan</p>
							<p class="mt-1 text-sm text-neutral-600 dark:text-neutral-400">
								Kehadiran <span class="font-medium text-neutral-900 dark:text-white">{hasil.nama}</span> di sesi {sesi.ke} {hasil.sudah_tercatat ? "sudah tercatat sebelumnya" : "tercatat"}.
							</p>
							{#if hasil.kelas}
								<p class="mt-3 text-sm text-neutral-600 dark:text-neutral-400">Kelas Anda: <span class="font-semibold text-neutral-900 dark:text-white">{hasil.kelas}</span></p>
								{#if hasil.zoom_url}
									<a href={hasil.zoom_url} target="_blank" rel="noopener noreferrer" class="mt-3 inline-flex min-h-11 items-center justify-center rounded-xl bg-brand-700 px-5 text-sm font-semibold text-white transition hover:bg-brand-800">Buka Zoom kelas</a>
								{/if}
							{/if}
						</div>
					{:else if sesi.status === "belum"}
						<div class="flex gap-3 rounded-xl bg-sky-500/10 p-4 text-sm text-sky-800 dark:text-sky-300">
							<Clock class="h-5 w-5 shrink-0" />
							<p>Absensi dibuka pukul <strong>{sesi.buka_pukul} WIB</strong>. Muat ulang halaman ini saat sesi akan dimulai.</p>
						</div>
					{:else if sesi.status === "tutup"}
						<div class="flex gap-3 rounded-xl bg-amber-500/10 p-4 text-sm text-amber-800 dark:text-amber-300">
							<CircleAlert class="h-5 w-5 shrink-0" />
							<p>Absensi sesi ini sudah ditutup. Bila Anda hadir, hubungi admin Ruang Sanad.</p>
						</div>
					{:else}
						<form onsubmit={kirim} class="space-y-4">
							<label class="block">
								<span class="text-sm font-medium text-neutral-800 dark:text-neutral-200">Nomor WhatsApp saat mendaftar</span>
								<input type="tel" inputmode="tel" autocomplete="tel" required bind:value={noWA} placeholder="0812xxxxxxxx"
									class="mt-1.5 w-full rounded-xl border border-neutral-300 bg-white px-4 py-3 text-base text-neutral-900 outline-none focus:border-brand-500 focus:ring-2 focus:ring-brand-500/30 dark:border-white/10 dark:bg-neutral-900 dark:text-white" />
							</label>
							{#if pesanError}<p class="rounded-lg bg-red-500/10 px-3 py-2 text-sm text-red-700 dark:text-red-300" role="alert">{pesanError}</p>{/if}
							<button type="submit" disabled={mengirim || noWA.trim().length < 9}
								class="w-full min-h-12 rounded-xl bg-brand-600 text-base font-semibold text-white transition hover:bg-brand-700 active:scale-[0.98] disabled:opacity-50 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-500 dark:bg-brand-500">
								{mengirim ? "Mencatat…" : "Saya hadir"}
							</button>
							<p class="text-center text-xs text-neutral-500">Nomor hanya dipakai untuk mencocokkan data pendaftaran.</p>
						</form>
					{/if}
				</div>
			</section>
		{/if}
	</div>
</main>
