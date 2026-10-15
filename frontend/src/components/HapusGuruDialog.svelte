<script lang="ts">
	import { onMount } from "svelte";
	import { router } from "@inertiajs/svelte";
	import { fly } from "svelte/transition";
	import { getCSRFToken } from "@lib/utils/csrf";
	import type { GuruHapusInfo } from "@lib/types";
	import { AlertTriangle, BookOpen, Loader2, Trash2, UserX } from "lucide-svelte";

	// Hapus permanen hanya untuk guru tanpa kelas & riwayat (mis. data dobel).
	// Guru yang punya riwayat diarahkan ke Nonaktifkan agar datanya tetap utuh.
	interface Props { guruId: number; guruNama: string; onclose: () => void; }
	let { guruId, guruNama, onclose }: Props = $props();

	let info = $state<GuruHapusInfo | null>(null);
	let loadError = $state("");
	let saving = $state(false);

	onMount(async () => {
		try {
			const res = await fetch(`/app/koordinator-guru/guru/${guruId}/hapus`, {
				headers: { Accept: "application/json", "X-XSRF-TOKEN": getCSRFToken() },
			});
			const body = await res.json();
			if (!res.ok) throw new Error(body?.error ?? "Gagal memeriksa riwayat guru");
			info = body as GuruHapusInfo;
		} catch (e) {
			loadError = e instanceof Error ? e.message : "Gagal memeriksa riwayat guru";
		}
	});

	function hapus() {
		saving = true;
		router.delete(`/app/koordinator-guru/guru/${guruId}`, { onSuccess: onclose, onFinish: () => (saving = false) });
	}

	function nonaktifkan() {
		saving = true;
		router.post(`/app/koordinator-guru/guru/${guruId}/nonaktif`, {}, { preserveScroll: true, onSuccess: onclose, onFinish: () => (saving = false) });
	}

	const btnSecondary = "flex-1 rounded-xl border border-neutral-300 px-4 py-2.5 text-sm font-medium text-neutral-700 hover:bg-neutral-100 dark:border-neutral-700 dark:text-neutral-300 dark:hover:bg-neutral-800";
</script>

<div class="fixed inset-0 z-50 flex items-center justify-center p-4">
	<button class="absolute inset-0 h-full w-full bg-neutral-900/50 backdrop-blur-sm" aria-label="Tutup" onclick={onclose}></button>
	<div role="dialog" aria-modal="true" aria-labelledby="hapus-guru-title" class="relative w-full max-w-md rounded-2xl border border-neutral-200/80 bg-white p-6 shadow-xl dark:border-white/[0.06] dark:bg-neutral-925" in:fly={{ y: 20, duration: 200 }}>
		<h3 id="hapus-guru-title" class="text-lg font-bold text-neutral-900 dark:text-white">Hapus {guruNama}?</h3>

		{#if loadError}
			<p class="mt-3 rounded-xl border border-red-500/20 bg-red-500/10 p-3 text-sm text-red-700 dark:text-red-400">{loadError}</p>
			<div class="mt-5 flex"><button onclick={onclose} class={btnSecondary}>Tutup</button></div>
		{:else if !info}
			<p class="mt-4 flex items-center gap-2 text-sm text-neutral-500"><Loader2 class="h-4 w-4 animate-spin" /> Memeriksa riwayat guru...</p>
		{:else if info.kelas_aktif.length > 0}
			<div class="mt-3 flex gap-3 rounded-xl border border-amber-500/30 bg-amber-500/[0.06] p-3">
				<BookOpen class="mt-0.5 h-4 w-4 shrink-0 text-amber-600 dark:text-amber-400" />
				<div class="text-sm text-neutral-700 dark:text-neutral-300">
					<p>Guru ini masih menjadi guru utama di {info.kelas_aktif.length} kelas aktif. Ganti guru kelasnya terlebih dahulu, baru guru ini bisa dinonaktifkan atau dihapus.</p>
					<ul class="mt-2 list-disc pl-4 text-neutral-600 dark:text-neutral-400">{#each info.kelas_aktif as k (k)}<li>{k}</li>{/each}</ul>
				</div>
			</div>
			<div class="mt-5 flex"><button onclick={onclose} class={btnSecondary}>Mengerti</button></div>
		{:else if !info.bisa_dihapus}
			<p class="mt-2 text-sm leading-6 text-neutral-600 dark:text-neutral-400">Guru ini sudah punya riwayat, jadi tidak bisa dihapus permanen. Menghapusnya akan ikut menghapus data di bawah ini.</p>
			<ul class="mt-3 divide-y divide-neutral-200/80 rounded-xl border border-neutral-200/80 text-sm dark:divide-white/[0.06] dark:border-white/[0.06]">
				{#each info.riwayat as r (r.label)}
					<li class="flex justify-between gap-3 px-3 py-2"><span class="text-neutral-700 dark:text-neutral-300">{r.label}</span><span class="font-semibold tabular-nums text-neutral-900 dark:text-white">{r.jumlah}</span></li>
				{/each}
			</ul>
			{#if info.is_aktif}
				<p class="mt-3 text-sm leading-6 text-neutral-600 dark:text-neutral-400">Sebagai gantinya, <strong>nonaktifkan</strong> guru ini: ia hilang dari daftar guru aktif dan pilihan guru, tetapi seluruh riwayatnya tetap tersimpan dan bisa diaktifkan kembali kapan saja.</p>
				<div class="mt-5 flex gap-3">
					<button onclick={onclose} class={btnSecondary}>Batal</button>
					<button onclick={nonaktifkan} disabled={saving} class="inline-flex flex-1 items-center justify-center gap-2 rounded-xl bg-amber-600 px-4 py-2.5 text-sm font-semibold text-white hover:bg-amber-700 disabled:opacity-50"><UserX class="h-4 w-4" /> {saving ? "Memproses..." : "Nonaktifkan"}</button>
				</div>
			{:else}
				<p class="mt-3 text-sm text-neutral-600 dark:text-neutral-400">Guru ini sudah nonaktif, jadi tidak ada lagi yang perlu dilakukan.</p>
				<div class="mt-5 flex"><button onclick={onclose} class={btnSecondary}>Tutup</button></div>
			{/if}
		{:else}
			<div class="mt-3 flex gap-3 rounded-xl border border-red-500/20 bg-red-500/[0.06] p-3">
				<AlertTriangle class="mt-0.5 h-4 w-4 shrink-0 text-red-600 dark:text-red-400" />
				<p class="text-sm text-neutral-700 dark:text-neutral-300">Guru ini belum punya kelas maupun riwayat apa pun. Data guru akan dihapus permanen dan tidak bisa dikembalikan. Akun login yang terhubung (bila ada) tidak ikut dihapus.</p>
			</div>
			<div class="mt-5 flex gap-3">
				<button onclick={onclose} class={btnSecondary}>Batal</button>
				<button onclick={hapus} disabled={saving} class="inline-flex flex-1 items-center justify-center gap-2 rounded-xl bg-red-600 px-4 py-2.5 text-sm font-semibold text-white hover:bg-red-700 disabled:opacity-50"><Trash2 class="h-4 w-4" /> {saving ? "Menghapus..." : "Hapus permanen"}</button>
			</div>
		{/if}
	</div>
</div>
