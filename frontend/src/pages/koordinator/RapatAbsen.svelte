<script lang="ts">
	import { inertia, router } from "@inertiajs/svelte";
	import AppLayout from "@layouts/AppLayout.svelte";
	import type { AbsenRow, Rapat, User } from "@lib/types";
	import { ArrowLeft, Save, Download, MessageCircle, Copy, X } from "lucide-svelte";

	interface LaporanAbsensi { jenis_kegiatan: string; judul: string; tanggal: string; keterangan: string; total: number; hadir: number; tidak_hadir: number; absen: AbsenRow[]; }
	interface Props { user?: User; rapat: Rapat; absen?: AbsenRow[]; laporan?: LaporanAbsensi; flash?: { success?: string; error?: string }; }
	let { user, rapat, absen = [], laporan, flash }: Props = $props();

	let rows = $state(absen.map((a) => ({ ...a })));
	let saving = $state(false);
	let showExport = $state(false);
	let copied = $state(false);
	function save() {
		const invalid = rows.find((r) => (r.hadir && !r.jam_masuk) || (!r.hadir && !r.alasan.trim()));
		if (invalid) { alert(`Lengkapi absensi ${invalid.nama}: jam masuk untuk hadir atau alasan untuk tidak hadir.`); return; }
		saving = true;
		router.post(`/app/koordinator-guru/rapat/${rapat.id}/absen`, { absen: rows.map((r) => ({ guru_id: r.guru_id, hadir: r.hadir, jam_masuk: r.jam_masuk, keterangan: r.keterangan, alasan: r.alasan })) }, { preserveScroll: true, onFinish: () => (saving = false) });
	}
	function setAttendance(guruID: number, hadir: boolean) { rows = rows.map((r) => r.guru_id === guruID ? { ...r, hadir, jam_masuk: hadir ? r.jam_masuk : "", alasan: hadir ? "" : r.alasan } : r); }
	function toggleAll(hadir: boolean) { rows = rows.map((r) => ({ ...r, hadir, jam_masuk: hadir ? r.jam_masuk : "", alasan: hadir ? "" : r.alasan })); }
	const hadirCount = $derived(rows.filter((r) => r.hadir).length);
	const waText = $derived(laporan ? [
		`*ABSENSI ${laporan.jenis_kegiatan.toUpperCase()}*`, "", `Kegiatan: ${laporan.judul || "-"}`, `Tanggal: ${laporan.tanggal}`,
		laporan.keterangan ? `Keterangan: ${laporan.keterangan}` : "", "", "*REKAP KEHADIRAN*", `Total: ${laporan.total} guru`, `Hadir: ${laporan.hadir} guru`, `Tidak hadir: ${laporan.tidak_hadir} guru`, "",
		"*DETAIL ABSENSI*", ...laporan.absen.map((r, i) => `${i + 1}. ${r.nama} - ${r.hadir ? "Hadir" : "Tidak Hadir"}${r.hadir ? `\n   Jam masuk: ${r.jam_masuk || "-"}` : `\n   Alasan: ${r.alasan || "-"}`}\n   Keterangan: ${r.keterangan || "-"}`)
	].filter(Boolean).join("\n") : "");
	async function copyReport() {
		try { await navigator.clipboard.writeText(waText); copied = true; setTimeout(() => (copied = false), 2000); }
		catch { alert("Teks tidak dapat disalin. Silakan salin dari preview secara manual."); }
	}
	function openWhatsApp() { window.open(`https://wa.me/?text=${encodeURIComponent(waText)}`, "_blank", "noopener,noreferrer"); }
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
		{#if flash?.success}<div class="rounded-xl bg-green-500/10 border border-green-500/20 p-4 text-sm font-medium text-green-700 dark:text-green-400">{flash.success}</div>{/if}
		{#if flash?.error}<div class="rounded-xl bg-red-500/10 border border-red-500/20 p-4 text-sm font-medium text-red-700 dark:text-red-400">{flash.error}</div>{/if}

		<div class="flex items-center justify-between gap-3 flex-wrap">
			<p class="text-sm text-neutral-600 dark:text-neutral-400">Hadir: <span class="font-semibold text-neutral-900 dark:text-white">{hadirCount}</span> / {rows.length}</p>
			<div class="flex gap-2">
				<button onclick={() => toggleAll(true)} class="px-3 py-1.5 rounded-lg bg-green-500/10 text-green-700 dark:text-green-400 text-xs font-semibold">Tandai semua hadir</button>
				<button onclick={() => toggleAll(false)} class="px-3 py-1.5 rounded-lg bg-neutral-500/10 text-neutral-600 dark:text-neutral-400 text-xs font-semibold">Kosongkan</button>
			</div>
		</div>
		{#if laporan}
			<div class="flex justify-end gap-2 flex-wrap">
				<a href={`/app/koordinator-guru/rapat/${rapat.id}/absen/export`} class="inline-flex items-center gap-2 px-3 py-2 rounded-lg border border-neutral-300 dark:border-neutral-700 text-sm font-semibold text-neutral-700 dark:text-neutral-200 hover:border-brand-400"><Download class="w-4 h-4" /> Export Excel</a>
				<button onclick={() => (showExport = true)} class="inline-flex items-center gap-2 px-3 py-2 rounded-lg bg-green-600 hover:bg-green-700 text-white text-sm font-semibold"><MessageCircle class="w-4 h-4" /> Bagikan ke WhatsApp</button>
			</div>
		{/if}

		<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 divide-y divide-neutral-200/80 dark:divide-white/[0.04]">
			{#each rows as r (r.guru_id)}
				<div class="grid sm:grid-cols-[minmax(0,1fr)_auto_9rem] gap-3 p-4">
					<div class="min-w-0"><p class="text-sm font-medium text-neutral-900 dark:text-white truncate">{r.nama}</p><p class="text-xs text-neutral-500">{r.status === "tetap" ? "Guru Tetap" : "Part Time"}</p></div>
					<div class="inline-flex rounded-lg bg-neutral-100 dark:bg-neutral-800 p-1 text-xs font-semibold"><button onclick={() => setAttendance(r.guru_id, true)} class="px-3 py-1.5 rounded-md {r.hadir ? 'bg-green-600 text-white' : 'text-neutral-600 dark:text-neutral-300'}">Hadir</button><button onclick={() => setAttendance(r.guru_id, false)} class="px-3 py-1.5 rounded-md {!r.hadir ? 'bg-red-600 text-white' : 'text-neutral-600 dark:text-neutral-300'}">Tidak hadir</button></div>
					{#if r.hadir}<label class="text-xs text-neutral-500">Jam masuk Zoom<input type="time" bind:value={r.jam_masuk} class="mt-1 w-full px-2 py-1.5 rounded-lg bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700 text-sm outline-none focus:border-brand-400" /></label>{/if}
					<div class="sm:col-span-3 grid sm:grid-cols-2 gap-3"><input bind:value={r.keterangan} placeholder="Keterangan (opsional)" class="px-3 py-2 rounded-lg bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700 text-sm outline-none focus:border-brand-400" />{#if !r.hadir}<input bind:value={r.alasan} placeholder="Alasan tidak hadir (wajib)" class="px-3 py-2 rounded-lg bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700 text-sm outline-none focus:border-brand-400" />{/if}</div>
				</div>
			{/each}
		</div>

		<div class="flex justify-end"><button onclick={save} disabled={saving} class="inline-flex items-center gap-2 px-5 py-2.5 rounded-xl bg-brand-600 hover:bg-brand-700 text-white text-sm font-semibold disabled:opacity-50"><Save class="w-4 h-4" /> {saving ? "Menyimpan..." : "Simpan absensi"}</button></div>
	</div>

	{#if showExport && laporan}
		<div class="fixed inset-0 z-50 flex items-end sm:items-center justify-center p-4 bg-neutral-950/50" role="presentation">
			<div class="w-full max-w-xl rounded-2xl bg-white dark:bg-neutral-900 shadow-xl p-5" role="dialog" aria-modal="true" aria-label="Preview laporan WhatsApp">
				<div class="flex items-center justify-between gap-4"><h2 class="text-lg font-bold text-neutral-900 dark:text-white">Preview WhatsApp</h2><button onclick={() => (showExport = false)} class="p-1 text-neutral-500 hover:text-neutral-900 dark:hover:text-white"><X class="w-5 h-5" /></button></div>
				<pre class="mt-4 max-h-80 overflow-auto whitespace-pre-wrap rounded-xl bg-neutral-100 dark:bg-neutral-800 p-4 text-sm text-neutral-700 dark:text-neutral-200 font-sans">{waText}</pre>
				<div class="mt-4 flex justify-end gap-2"><button onclick={copyReport} class="inline-flex items-center gap-2 px-3 py-2 rounded-lg border border-neutral-300 dark:border-neutral-700 text-sm font-semibold text-neutral-700 dark:text-neutral-200"><Copy class="w-4 h-4" /> {copied ? "Tersalin" : "Salin teks"}</button><button onclick={openWhatsApp} class="inline-flex items-center gap-2 px-3 py-2 rounded-lg bg-green-600 hover:bg-green-700 text-white text-sm font-semibold"><MessageCircle class="w-4 h-4" /> Buka WhatsApp</button></div>
			</div>
		</div>
	{/if}
</AppLayout>
