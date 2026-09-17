<script lang="ts">
	import { inertia, router } from "@inertiajs/svelte";
	import AppLayout from "@layouts/AppLayout.svelte";
	import type { AbsenRow, Rapat, User } from "@lib/types";
	import { ArrowLeft, Save, Check } from "lucide-svelte";

	interface Props { user?: User; rapat: Rapat; absen?: AbsenRow[]; success?: string; error?: string; }
	let { user, rapat, absen = [], success, error }: Props = $props();

	let rows = $state(absen.map((a) => ({ ...a })));
	let saving = $state(false);
	function save() {
		const invalid = rows.find((r) => !r.keterangan.trim() || (r.hadir && !r.jam_masuk) || (!r.hadir && !r.alasan.trim()));
		if (invalid) { alert(`Lengkapi absensi ${invalid.nama}: keterangan wajib, jam masuk untuk hadir, dan alasan untuk tidak hadir.`); return; }
		saving = true;
		router.post(`/app/koordinator-guru/rapat/${rapat.id}/absen`, { absen: rows.map((r) => ({ guru_id: r.guru_id, hadir: r.hadir, jam_masuk: r.jam_masuk, keterangan: r.keterangan, alasan: r.alasan })) }, { preserveScroll: true, onFinish: () => (saving = false) });
	}
	function toggleAll(v: boolean) { rows = rows.map((r) => ({ ...r, hadir: v })); }
	const hadirCount = $derived(rows.filter((r) => r.hadir).length);
</script>

<AppLayout {user} group="koordinator-rapat">
	<div class="pt-8 pb-8 border-b border-neutral-200/80 dark:border-white/[0.04]">
		<div class="max-w-4xl mx-auto px-4 sm:px-6">
			<a href="/app/koordinator-guru/rapat" use:inertia class="inline-flex items-center gap-1.5 text-sm text-neutral-500 hover:text-brand-600 mb-4"><ArrowLeft class="w-4 h-4" /> Rapat</a>
			<h1 class="text-2xl font-bold text-neutral-900 dark:text-white">Absensi Rapat</h1>
			<p class="mt-1.5 text-neutral-600 dark:text-neutral-400">{rapat.tanggal} — {rapat.judul || "(tanpa judul)"}</p>
		</div>
	</div>

	<div class="max-w-4xl mx-auto px-4 sm:px-6 py-8 space-y-4">
		{#if success}<div class="rounded-xl bg-green-500/10 border border-green-500/20 p-4 text-sm font-medium text-green-700 dark:text-green-400">{success}</div>{/if}
		{#if error}<div class="rounded-xl bg-red-500/10 border border-red-500/20 p-4 text-sm font-medium text-red-700 dark:text-red-400">{error}</div>{/if}

		<div class="flex items-center justify-between gap-3 flex-wrap">
			<p class="text-sm text-neutral-600 dark:text-neutral-400">Hadir: <span class="font-semibold text-neutral-900 dark:text-white">{hadirCount}</span> / {rows.length}</p>
			<div class="flex gap-2">
				<button onclick={() => toggleAll(true)} class="px-3 py-1.5 rounded-lg bg-green-500/10 text-green-700 dark:text-green-400 text-xs font-semibold">Tandai semua hadir</button>
				<button onclick={() => toggleAll(false)} class="px-3 py-1.5 rounded-lg bg-neutral-500/10 text-neutral-600 dark:text-neutral-400 text-xs font-semibold">Kosongkan</button>
			</div>
		</div>

		<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 divide-y divide-neutral-200/80 dark:divide-white/[0.04]">
			{#each rows as r (r.guru_id)}
				<div class="grid sm:grid-cols-[auto_minmax(0,1fr)_9rem] gap-3 p-4">
					<button onclick={() => (r.hadir = !r.hadir)} class="w-6 h-6 rounded-md border-2 flex items-center justify-center shrink-0 {r.hadir ? 'bg-green-500 border-green-500 text-white' : 'border-neutral-300 dark:border-neutral-600'}">{#if r.hadir}<Check class="w-4 h-4" />{/if}</button>
					<div class="min-w-0"><p class="text-sm font-medium text-neutral-900 dark:text-white truncate">{r.nama}</p><p class="text-xs text-neutral-500">{r.status === "tetap" ? "Guru Tetap" : "Part Time"}</p></div>
					{#if r.hadir}<label class="text-xs text-neutral-500">Jam masuk Zoom<input type="time" bind:value={r.jam_masuk} class="mt-1 w-full px-2 py-1.5 rounded-lg bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700 text-sm outline-none focus:border-brand-400" /></label>{/if}
					<div class="sm:col-span-3 grid sm:grid-cols-2 gap-3"><input bind:value={r.keterangan} placeholder="Keterangan kehadiran (wajib)" class="px-3 py-2 rounded-lg bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700 text-sm outline-none focus:border-brand-400" />{#if !r.hadir}<input bind:value={r.alasan} placeholder="Alasan tidak hadir (wajib)" class="px-3 py-2 rounded-lg bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700 text-sm outline-none focus:border-brand-400" />{/if}</div>
				</div>
			{/each}
		</div>

		<div class="flex justify-end"><button onclick={save} disabled={saving} class="inline-flex items-center gap-2 px-5 py-2.5 rounded-xl bg-brand-600 hover:bg-brand-700 text-white text-sm font-semibold disabled:opacity-50"><Save class="w-4 h-4" /> {saving ? "Menyimpan..." : "Simpan absensi"}</button></div>
	</div>
</AppLayout>
