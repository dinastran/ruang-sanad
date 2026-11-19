<script lang="ts">
	import Logo from "@components/Logo.svelte";
	import { CircleCheck, HandCoins, Share2, Users } from "lucide-svelte";

	let form = $state({ nama: "", no_wa: "", email: "", bank_nama: "", bank_rekening: "", bank_atas_nama: "", setuju: false, website: "" });
	let errors = $state<Record<string, string>>({});
	let pesanError = $state("");
	let mengirim = $state(false);
	let selesai = $state(false);

	const inputCls = "mt-1.5 w-full rounded-xl border border-neutral-300 bg-white px-4 py-3 text-base text-neutral-900 outline-none transition focus:border-brand-500 focus:ring-2 focus:ring-brand-500/30 aria-[invalid=true]:border-red-500 dark:border-white/10 dark:bg-neutral-900 dark:text-white";
	const labelCls = "text-sm font-medium text-neutral-800 dark:text-neutral-200";

	async function kirim(e: Event) {
		e.preventDefault();
		if (mengirim) return;
		errors = {};
		pesanError = "";
		mengirim = true;
		try {
			const res = await fetch("/affiliate/daftar", { method: "POST", headers: { "Content-Type": "application/json", Accept: "application/json" }, body: JSON.stringify(form) });
			const data = await res.json().catch(() => ({}));
			if (!res.ok) {
				if (data.field) errors = { [data.field]: data.error };
				else pesanError = data.error || "Pendaftaran belum berhasil. Silakan coba lagi.";
				return;
			}
			selesai = true;
		} catch {
			pesanError = "Koneksi terputus. Periksa internet lalu coba lagi.";
		} finally {
			mengirim = false;
		}
	}
</script>

<svelte:head><title>Daftar Affiliate · Ruang Sanad</title></svelte:head>

<main class="min-h-[100dvh] bg-neutral-50 px-4 py-8 dark:bg-neutral-950 sm:py-14">
	<div class="mx-auto w-full max-w-md">
		<div class="mb-6 flex items-center gap-2.5">
			<Logo size={32} />
			<span class="font-semibold text-neutral-900 dark:text-white">Ruang Sanad</span>
		</div>

		{#if selesai}
			<section class="rounded-2xl border border-neutral-200/80 bg-white p-6 text-center shadow-sm dark:border-white/[0.06] dark:bg-neutral-925/50" role="status">
				<CircleCheck class="mx-auto h-12 w-12 text-green-600 dark:text-green-400" />
				<h1 class="mt-3 text-lg font-semibold text-neutral-900 dark:text-white">Pendaftaran affiliate terkirim</h1>
				<p class="mt-1 text-sm text-neutral-600 dark:text-neutral-400">Tim kami akan meninjau data Anda. Setelah disetujui, link untuk membuat password akan dikirim ke WhatsApp atau email Anda.</p>
			</section>
		{:else}
			<section class="rounded-2xl border border-neutral-200/80 bg-white shadow-sm dark:border-white/[0.06] dark:bg-neutral-925/50">
				<header class="border-b border-neutral-200/80 p-6 dark:border-white/[0.05]">
					<p class="text-sm font-medium text-brand-700 dark:text-brand-400">Program Affiliate</p>
					<h1 class="mt-1 text-xl font-bold tracking-tight text-neutral-900 dark:text-white">Ajak belajar Al-Qur'an, dapatkan komisi</h1>
					<ul class="mt-4 space-y-2.5 text-sm text-neutral-700 dark:text-neutral-300">
						<li class="flex gap-2.5"><Share2 class="mt-0.5 h-4 w-4 shrink-0 text-brand-600 dark:text-brand-400" /> Dapat link & kode pribadi untuk dibagikan.</li>
						<li class="flex gap-2.5"><Users class="mt-0.5 h-4 w-4 shrink-0 text-brand-600 dark:text-brand-400" /> Setiap pendaftar yang membayar lewat link Anda tercatat otomatis.</li>
						<li class="flex gap-2.5"><HandCoins class="mt-0.5 h-4 w-4 shrink-0 text-brand-600 dark:text-brand-400" /> Komisi ditransfer ke rekening Anda setelah masa tahan.</li>
					</ul>
				</header>
				<form onsubmit={kirim} class="relative space-y-4 p-6">
					<label class="block"><span class={labelCls}>Nama lengkap</span><input bind:value={form.nama} name="nama" autocomplete="name" required maxlength="80" aria-invalid={!!errors.nama} class={inputCls} />
						{#if errors.nama}<p class="mt-1.5 text-sm text-red-600 dark:text-red-400">{errors.nama}</p>{/if}</label>
					<label class="block"><span class={labelCls}>Nomor WhatsApp</span><input bind:value={form.no_wa} name="no_wa" type="tel" inputmode="tel" autocomplete="tel" required placeholder="0812xxxxxxxx" aria-invalid={!!errors.no_wa} class={inputCls} />
						{#if errors.no_wa}<p class="mt-1.5 text-sm text-red-600 dark:text-red-400">{errors.no_wa}</p>{/if}</label>
					<label class="block"><span class={labelCls}>Email (untuk login)</span><input bind:value={form.email} name="email" type="email" autocomplete="email" required aria-invalid={!!errors.email} class={inputCls} />
						{#if errors.email}<p class="mt-1.5 text-sm text-red-600 dark:text-red-400">{errors.email}</p>{/if}</label>
					<fieldset class="space-y-3 rounded-xl bg-neutral-50 p-4 dark:bg-neutral-900/60">
						<legend class="px-1 text-sm font-semibold text-neutral-900 dark:text-white">Rekening pencairan (bisa diisi nanti)</legend>
						<div class="grid grid-cols-[110px_1fr] gap-2">
							<input bind:value={form.bank_nama} aria-label="Nama bank" placeholder="Bank" maxlength="60" class={inputCls + " !mt-0"} />
							<input bind:value={form.bank_rekening} aria-label="Nomor rekening" inputmode="numeric" placeholder="No. rekening" maxlength="40" class={inputCls + " !mt-0"} />
						</div>
						<input bind:value={form.bank_atas_nama} aria-label="Atas nama" placeholder="Atas nama" maxlength="80" class={inputCls + " !mt-0"} />
					</fieldset>
					<label class="flex items-start gap-2.5 text-sm text-neutral-700 dark:text-neutral-300">
						<input type="checkbox" bind:checked={form.setuju} class="mt-0.5 h-4 w-4 accent-brand-600" />
						<span>Saya setuju: komisi hanya untuk pendaftar yang membayar, tidak berlaku untuk diri sendiri, dan bisa dibatalkan bila santri berhenti dalam masa tahan.</span>
					</label>
					{#if errors.setuju}<p class="text-sm text-red-600 dark:text-red-400">{errors.setuju}</p>{/if}
					<div class="absolute -left-[9999px] h-px w-px overflow-hidden" aria-hidden="true"><label>Website <input bind:value={form.website} tabindex="-1" autocomplete="off" /></label></div>
					{#if pesanError}<p class="rounded-lg bg-red-500/10 px-3 py-2 text-sm text-red-700 dark:text-red-300" role="alert">{pesanError}</p>{/if}
					<button type="submit" disabled={mengirim} class="w-full min-h-12 rounded-xl bg-brand-600 text-base font-semibold text-white transition hover:bg-brand-700 active:scale-[0.98] disabled:opacity-50 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-500 dark:bg-brand-500">
						{mengirim ? "Mengirim…" : "Daftar jadi affiliate"}
					</button>
				</form>
			</section>
		{/if}
	</div>
</main>
