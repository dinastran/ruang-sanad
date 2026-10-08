<script lang="ts">
	import { inertia, router } from "@inertiajs/svelte";
	import { fly } from "svelte/transition";
	import AppLayout from "@layouts/AppLayout.svelte";
	import type { Kunjungan, User } from "@lib/types";

	interface Props { user?: User; kunjungan: Kunjungan; success?: string; error?: string; }
	let { user, kunjungan: k, success, error }: Props = $props();

	let isGuru = $derived(user?.role === "guru");
	const fmt = (n: number) => n.toFixed(2).replace(".", ",");
	let tanggapan = $state(k.tanggapan_guru);
	let saving = $state(false);
	let editing = $state(!k.tanggapan_guru);

	function simpan() {
		saving = true;
		router.post(`/app/guru/kunjungan/${k.id}/tanggapan`, { tanggapan }, { preserveScroll: true, onFinish: () => { saving = false; } });
	}
</script>

<AppLayout {user} group="guru-kunjungan">
	<div class="max-w-3xl mx-auto px-4 sm:px-6 py-8 space-y-6">
		<div class="flex items-center gap-2 text-sm text-neutral-500"><a href="/app/guru/kunjungan" use:inertia class="hover:text-brand-600">Hasil Kunjungan</a><span>/</span><span class="text-neutral-700 dark:text-neutral-300">{k.tanggal}</span></div>
		{#if success}<div class="rounded-xl bg-green-500/10 border border-green-500/20 p-4 text-sm font-medium text-green-700 dark:text-green-400" in:fly={{ y: 10, duration: 200 }}>{success}</div>{/if}
		{#if error}<div class="rounded-xl bg-red-500/10 border border-red-500/20 p-4 text-sm font-medium text-red-700 dark:text-red-400">{error}</div>{/if}

		<header class="flex items-end justify-between gap-4 flex-wrap">
			<div>
				<h1 class="text-2xl sm:text-3xl font-bold text-neutral-900 dark:text-white tracking-tight">{k.kelas_nama || "Kunjungan kelas"}</h1>
				<p class="mt-1 text-neutral-600 dark:text-neutral-400">Kunjungan {k.tanggal} {k.jam}</p>
			</div>
			<div class="text-right">
				<p class="text-xs uppercase tracking-wider text-neutral-500">Rata-rata</p>
				<p class="text-3xl font-bold tabular-nums text-neutral-900 dark:text-white">{fmt(k.nilai_rata_rata)}</p>
				<p class="text-sm text-neutral-600 dark:text-neutral-400">{k.predikat}</p>
			</div>
		</header>

		<section class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-5 sm:p-6">
			<h2 class="text-base font-semibold text-neutral-900 dark:text-white">Penilaian</h2>
			<ul class="mt-3 divide-y divide-neutral-200/80 dark:divide-white/[0.05]">
				{#each k.aspek as a (a.kode)}
					<li class="py-3 flex items-center justify-between gap-4">
						<span class="text-sm text-neutral-800 dark:text-neutral-200">{a.label}</span>
						<span class="flex items-center gap-3">
							<span class="flex gap-1" aria-hidden="true">{#each [1, 2, 3, 4] as i (i)}<span class="h-2 w-6 rounded-full {i <= a.nilai ? 'bg-brand-600 dark:bg-brand-400' : 'bg-neutral-200 dark:bg-neutral-800'}"></span>{/each}</span>
							<span class="w-32 text-right text-sm"><span class="font-semibold tabular-nums text-neutral-900 dark:text-white">{a.nilai}</span> <span class="text-neutral-500">{a.predikat}</span></span>
						</span>
					</li>
				{/each}
			</ul>
		</section>

		<section class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-5 sm:p-6">
			<h2 class="text-base font-semibold text-neutral-900 dark:text-white">Catatan Koordinator</h2>
			<p class="mt-2 text-sm leading-6 text-neutral-700 dark:text-neutral-300 whitespace-pre-line">{k.catatan}</p>
		</section>

		{#if k.tindak_lanjut.length > 0}
			<section class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-5 sm:p-6">
				<h2 class="text-base font-semibold text-neutral-900 dark:text-white">Tindak lanjut</h2>
				<ul class="mt-3 space-y-3">
					{#each k.tindak_lanjut as t (t.id)}
						<li>
							<p class="text-sm font-semibold text-neutral-900 dark:text-white">{t.jenis_label}{#if t.status === "selesai"}<span class="ml-2 text-xs font-medium text-green-700 dark:text-green-400">Selesai</span>{/if}</p>
							{#if t.catatan}<p class="text-sm text-neutral-600 dark:text-neutral-400">{t.catatan}</p>{/if}
							{#if t.target_tanggal}<p class="text-xs text-neutral-500">Target {t.target_tanggal}</p>{/if}
						</li>
					{/each}
				</ul>
			</section>
		{/if}

		<section class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-5 sm:p-6">
			<h2 class="text-base font-semibold text-neutral-900 dark:text-white">Tanggapan Anda</h2>
			{#if !editing}
				<p class="mt-2 text-sm leading-6 text-neutral-700 dark:text-neutral-300 whitespace-pre-line">{k.tanggapan_guru}</p>
				<p class="mt-1 text-xs text-neutral-500">Dikirim {k.tanggapan_at}</p>
				{#if isGuru}<button onclick={() => (editing = true)} class="mt-3 text-sm font-semibold text-brand-600 hover:underline dark:text-brand-400">Ubah tanggapan</button>{/if}
			{:else if isGuru}
				<p class="mt-0.5 text-xs text-neutral-500">Tulis tanggapan atau komitmen perbaikan. Koordinator dapat melihatnya.</p>
				<form onsubmit={(e) => { e.preventDefault(); simpan(); }} class="mt-3 space-y-3">
					<textarea bind:value={tanggapan} rows="4" maxlength="2000" required class="w-full px-3.5 py-2.5 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700 text-sm text-neutral-900 dark:text-white outline-none focus:border-brand-400"></textarea>
					<div class="flex justify-end gap-3">
						{#if k.tanggapan_guru}<button type="button" onclick={() => { tanggapan = k.tanggapan_guru; editing = false; }} class="px-4 py-2.5 rounded-xl text-sm font-semibold text-neutral-600 dark:text-neutral-300 hover:bg-neutral-100 dark:hover:bg-neutral-800">Batal</button>{/if}
						<button type="submit" disabled={saving || !tanggapan.trim()} class="px-5 py-2.5 rounded-xl bg-brand-600 hover:bg-brand-700 text-white text-sm font-semibold disabled:opacity-50">{saving ? "Menyimpan..." : "Kirim tanggapan"}</button>
					</div>
				</form>
			{:else}
				<p class="mt-2 text-sm text-neutral-500">Belum ada tanggapan.</p>
			{/if}
		</section>
	</div>
</AppLayout>
