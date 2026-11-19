<script lang="ts">
	import Logo from "@components/Logo.svelte";
	import { CircleCheck } from "lucide-svelte";

	interface Props { token: string }
	let { token }: Props = $props();
	let password = $state("");
	let ulang = $state("");
	let pesanError = $state("");
	let mengirim = $state(false);
	let selesai = $state(false);
	let cocok = $derived(password.length >= 8 && password === ulang);

	const inputCls = "mt-1.5 w-full rounded-xl border border-neutral-300 bg-white px-4 py-3 text-base text-neutral-900 outline-none focus:border-brand-500 focus:ring-2 focus:ring-brand-500/30 dark:border-white/10 dark:bg-neutral-900 dark:text-white";

	async function kirim(e: Event) {
		e.preventDefault();
		if (!cocok || mengirim) return;
		pesanError = "";
		mengirim = true;
		try {
			const res = await fetch(`/affiliate/aktivasi/${encodeURIComponent(token)}`, { method: "POST", headers: { "Content-Type": "application/json", Accept: "application/json" }, body: JSON.stringify({ password }) });
			const data = await res.json().catch(() => ({}));
			if (!res.ok) {
				pesanError = data.error || "Aktivasi gagal. Minta link baru ke admin.";
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

<svelte:head><title>Aktivasi Affiliate · Ruang Sanad</title></svelte:head>

<main class="min-h-[100dvh] bg-neutral-50 px-4 py-10 dark:bg-neutral-950 sm:py-16">
	<div class="mx-auto w-full max-w-sm">
		<div class="mb-6 flex items-center gap-2.5"><Logo size={32} /><span class="font-semibold text-neutral-900 dark:text-white">Ruang Sanad</span></div>
		<section class="rounded-2xl border border-neutral-200/80 bg-white p-6 shadow-sm dark:border-white/[0.06] dark:bg-neutral-925/50">
			{#if selesai}
				<div class="text-center" role="status">
					<CircleCheck class="mx-auto h-12 w-12 text-green-600 dark:text-green-400" />
					<h1 class="mt-3 text-lg font-semibold text-neutral-900 dark:text-white">Akun Anda aktif</h1>
					<p class="mt-1 text-sm text-neutral-600 dark:text-neutral-400">Masuk dengan email pendaftaran dan password yang baru dibuat.</p>
					<a href="/login" class="mt-5 flex min-h-12 w-full items-center justify-center rounded-xl bg-brand-600 text-base font-semibold text-white hover:bg-brand-700 dark:bg-brand-500">Masuk</a>
				</div>
			{:else}
				<h1 class="text-xl font-bold tracking-tight text-neutral-900 dark:text-white">Buat password</h1>
				<p class="mt-1 text-sm text-neutral-600 dark:text-neutral-400">Untuk masuk ke halaman Affiliate Saya.</p>
				<form onsubmit={kirim} class="mt-5 space-y-4">
					<label class="block"><span class="text-sm font-medium text-neutral-800 dark:text-neutral-200">Password (min. 8 karakter)</span>
						<input type="password" bind:value={password} autocomplete="new-password" minlength="8" required class={inputCls} /></label>
					<label class="block"><span class="text-sm font-medium text-neutral-800 dark:text-neutral-200">Ulangi password</span>
						<input type="password" bind:value={ulang} autocomplete="new-password" required class={inputCls} /></label>
					{#if ulang && password !== ulang}<p class="text-sm text-red-600 dark:text-red-400">Password belum sama.</p>{/if}
					{#if pesanError}<p class="rounded-lg bg-red-500/10 px-3 py-2 text-sm text-red-700 dark:text-red-300" role="alert">{pesanError}</p>{/if}
					<button type="submit" disabled={!cocok || mengirim} class="w-full min-h-12 rounded-xl bg-brand-600 text-base font-semibold text-white transition hover:bg-brand-700 disabled:opacity-50 dark:bg-brand-500">{mengirim ? "Menyimpan…" : "Simpan & aktifkan"}</button>
				</form>
			{/if}
		</section>
	</div>
</main>
