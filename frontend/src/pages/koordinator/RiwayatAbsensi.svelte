<script lang="ts">
	import { inertia } from "@inertiajs/svelte";
	import AppLayout from "@layouts/AppLayout.svelte";
	import type { GuruRingkas, User } from "@lib/types";
	import { ClipboardList, Search } from "lucide-svelte";

	interface Riwayat { guru_id: number; guru_nama: string; kegiatan: string; kegiatan_id: number; tanggal: string; judul: string; hadir: boolean; jam_masuk: string; keterangan: string; alasan: string; }
	interface Props { user?: User; riwayat?: Riwayat[]; guruList?: GuruRingkas[]; }
	let props: Props = $props();
	let guruID = $state("");
	let search = $state("");
	let rows = $derived((props.riwayat ?? []).filter((r) => (!guruID || r.guru_id === Number(guruID)) && `${r.guru_nama} ${r.judul} ${r.kegiatan}`.toLowerCase().includes(search.toLowerCase())));
	const labelKegiatan = (value: string) => value === "pembinaan" ? "Pembinaan" : value === "rapat" ? "Rapat" : "Kunjungan";
</script>

<AppLayout user={props.user}>
	<main class="max-w-6xl mx-auto px-4 sm:px-6 py-8 space-y-6"><header><p class="text-xs font-bold tracking-[0.16em] uppercase text-brand-600 dark:text-brand-400">Koordinator Guru</p><h1 class="mt-2 text-3xl font-bold text-neutral-900 dark:text-white">Riwayat absensi guru</h1><p class="mt-2 text-neutral-600 dark:text-neutral-400">Rekap Pembinaan, Rapat, dan Kunjungan Kelas.</p></header><div class="grid sm:grid-cols-2 gap-3"><label class="relative"><Search size="16" class="absolute left-3 top-3 text-neutral-400" /><input bind:value={search} placeholder="Cari guru atau kegiatan" class="w-full pl-9 pr-3 py-2.5 rounded-xl bg-white dark:bg-neutral-925 border border-neutral-200 dark:border-neutral-800 text-sm" /></label><select bind:value={guruID} class="px-3 py-2.5 rounded-xl bg-white dark:bg-neutral-925 border border-neutral-200 dark:border-neutral-800 text-sm"><option value="">Semua guru</option>{#each props.guruList ?? [] as guru (guru.id)}<option value={guru.id}>{guru.nama}</option>{/each}</select></div><section class="rounded-2xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-925 overflow-x-auto"><table class="w-full min-w-[800px] text-sm"><thead class="bg-neutral-50 dark:bg-neutral-900/50 text-xs uppercase text-neutral-500"><tr><th class="text-left p-4">Tanggal</th><th class="text-left p-4">Guru</th><th class="text-left p-4">Kegiatan</th><th class="text-left p-4">Kehadiran</th><th class="text-left p-4">Jam masuk</th><th class="text-left p-4">Keterangan</th></tr></thead><tbody class="divide-y divide-neutral-100 dark:divide-neutral-800">{#each rows as row (`${row.kegiatan}-${row.kegiatan_id}-${row.guru_id}`)}<tr><td class="p-4 text-neutral-500">{row.tanggal}</td><td class="p-4 font-semibold text-neutral-900 dark:text-white">{row.guru_nama}</td><td class="p-4"><p>{labelKegiatan(row.kegiatan)}</p><p class="text-xs text-neutral-500">{row.judul || "-"}</p></td><td class="p-4"><span class="px-2 py-1 rounded-full text-xs font-semibold {row.hadir ? 'bg-success/10 text-success' : 'bg-error/10 text-error'}">{row.hadir ? "Hadir" : "Tidak hadir"}</span></td><td class="p-4 font-mono">{row.jam_masuk || "-"}</td><td class="p-4"><p>{row.keterangan || "-"}</p>{#if row.alasan}<p class="mt-1 text-xs text-error">Alasan: {row.alasan}</p>{/if}</td></tr>{:else}<tr><td colspan="6" class="p-12 text-center text-neutral-500"><ClipboardList class="w-7 h-7 mx-auto mb-2 opacity-50" />Belum ada riwayat sesuai filter.</td></tr>{/each}</tbody></table></section></main>
</AppLayout>
