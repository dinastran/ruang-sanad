<script lang="ts">
	import { inertia, router } from "@inertiajs/svelte";
	import { fly } from "svelte/transition";
	import AppLayout from "@layouts/AppLayout.svelte";
	import type { GuruRingkas, KelasSimple, Kunjungan, KunjunganDashboard, User } from "@lib/types";
	import { Plus, X, BookOpen, ChevronRight, TrendingUp, TrendingDown, Minus, CheckCircle2, RotateCcw } from "lucide-svelte";

	type Tab = "kunjungan" | "rekap" | "tindak-lanjut" | "belum-ditanggapi";
	interface Props { user?: User; dashboard: KunjunganDashboard; guruList?: GuruRingkas[]; kelasList?: KelasSimple[]; tab?: string; success?: string; error?: string; }
	let { user, dashboard, guruList = [], kelasList = [], tab: tabAwal = "", success, error }: Props = $props();

	const TABS: { id: Tab; label: string }[] = [
		{ id: "kunjungan", label: "Kunjungan" },
		{ id: "rekap", label: "Rekap Guru" },
		{ id: "tindak-lanjut", label: "Tindak Lanjut" },
		{ id: "belum-ditanggapi", label: "Belum Ditanggapi" },
	];
	let tab = $state<Tab>(TABS.some((t) => t.id === tabAwal) ? (tabAwal as Tab) : "kunjungan");

	function pilihTab(id: Tab) {
		tab = id;
		const url = new URL(window.location.href);
		url.searchParams.set("tab", id);
		history.replaceState(history.state, "", url);
	}

	const inputClass = "w-full px-3.5 py-2.5 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700 text-sm text-neutral-900 dark:text-white outline-none focus:border-brand-400";
	const STATUS: Record<string, string> = { dijadwalkan: "Dijadwalkan", terlaksana: "Terlaksana", ditunda: "Ditunda", batal: "Batal" };
	const STATUS_COLOR: Record<string, string> = { dijadwalkan: "bg-blue-500/10 text-blue-700 dark:text-blue-400", terlaksana: "bg-green-500/10 text-green-700 dark:text-green-400", ditunda: "bg-amber-500/10 text-amber-700 dark:text-amber-400", batal: "bg-red-500/10 text-red-600 dark:text-red-400" };
	const KIRIM: Record<string, string> = { draft: "Belum dikirim", terkirim: "Terkirim · belum dibaca", dibaca: "Dibaca guru", ditanggapi: "Ditanggapi guru" };
	const KIRIM_COLOR: Record<string, string> = { draft: "bg-neutral-500/10 text-neutral-600 dark:text-neutral-400", terkirim: "bg-amber-500/10 text-amber-700 dark:text-amber-400", dibaca: "bg-blue-500/10 text-blue-700 dark:text-blue-400", ditanggapi: "bg-green-500/10 text-green-700 dark:text-green-400" };

	function predikatColor(p: string): string {
		if (p === "Sangat baik") return "text-green-700 dark:text-green-400";
		if (p === "Baik") return "text-brand-600 dark:text-brand-400";
		if (p === "Cukup") return "text-amber-700 dark:text-amber-400";
		if (p === "Perlu perbaikan") return "text-red-600 dark:text-red-400";
		return "text-neutral-500";
	}
	const nilai = (n: number) => n.toFixed(2).replace(".", ",");

	let showForm = $state(false);
	let saving = $state(false);
	const empty = () => ({ guru_id: 0, kelas_id: 0, target_mulai: "", target_selesai: "", tanggal: "", jam: "", status: "dijadwalkan", catatan: "" });
	let form = $state(empty());

	// Kelas mengikuti guru terpilih: hanya kelas aktif yang diampu guru itu.
	let kelasGuru = $derived(kelasList.filter((k) => form.guru_id > 0 && k.guru_id === form.guru_id));

	function pilihGuru() {
		const daftar = kelasList.filter((k) => k.guru_id === form.guru_id);
		form.kelas_id = daftar.length === 1 ? daftar[0].id : 0;
	}

	function openCreate() { form = empty(); showForm = true; }
	function submit() {
		saving = true;
		router.post("/app/koordinator-guru/kunjungan", form, { onSuccess: () => { showForm = false; }, onFinish: () => { saving = false; } });
	}

	function setStatusTL(kunjunganID: number, id: number, status: "selesai" | "terbuka") {
		router.put(`/app/koordinator-guru/kunjungan/${kunjunganID}/tindak-lanjut/${id}`, { status }, { preserveScroll: true });
	}

	const counts = $derived<Record<Tab, number>>({
		kunjungan: dashboard.kunjungan.length,
		rekap: dashboard.rekap.length,
		"tindak-lanjut": dashboard.tindak_lanjut_terbuka.length,
		"belum-ditanggapi": dashboard.belum_ditanggapi.length,
	});
</script>

{#snippet kartu(k: Kunjungan)}
	<a href={`/app/koordinator-guru/kunjungan/${k.id}`} use:inertia class="block rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-5 hover:border-brand-400/60 transition-colors">
		<div class="flex items-start justify-between gap-4">
			<div class="min-w-0 flex-1">
				<div class="flex items-center gap-2 flex-wrap">
					<h2 class="font-semibold text-neutral-900 dark:text-white">{k.guru_nama}</h2>
					<span class="inline-flex px-2 py-0.5 rounded-full text-xs font-medium {STATUS_COLOR[k.status]}">{STATUS[k.status] ?? k.status}</span>
					{#if k.status === "terlaksana"}<span class="inline-flex px-2 py-0.5 rounded-full text-xs font-medium {KIRIM_COLOR[k.status_kirim]}">{KIRIM[k.status_kirim]}</span>{/if}
				</div>
				<p class="mt-1 text-sm text-neutral-600 dark:text-neutral-400">{k.kelas_nama || "Kelas belum dipilih"}</p>
				<p class="mt-1 text-xs text-neutral-500">{#if k.tanggal}Pelaksanaan: {k.tanggal} {k.jam}{:else if k.target_mulai}Target: {k.target_mulai}{k.target_selesai && k.target_selesai !== k.target_mulai ? ` — ${k.target_selesai}` : ""}{/if}</p>
				{#if k.tindak_lanjut.length > 0}
					<div class="mt-2 flex flex-wrap gap-1.5">
						{#each k.tindak_lanjut as tl (tl.id)}
							<span class="inline-flex px-2 py-0.5 rounded-md text-xs border {tl.status === 'selesai' ? 'border-neutral-200 text-neutral-400 line-through dark:border-neutral-800' : 'border-neutral-300 text-neutral-700 dark:border-neutral-700 dark:text-neutral-300'}">{tl.jenis_label}</span>
						{/each}
					</div>
				{/if}
			</div>
			<div class="flex items-center gap-3 shrink-0">
				{#if k.nilai_lengkap}
					<div class="text-right">
						<p class="text-xl font-bold tabular-nums text-neutral-900 dark:text-white">{nilai(k.nilai_rata_rata)}</p>
						<p class="text-xs font-medium {predikatColor(k.predikat)}">{k.predikat}</p>
					</div>
				{/if}
				<ChevronRight class="w-4 h-4 text-neutral-400" />
			</div>
		</div>
	</a>
{/snippet}

<AppLayout {user} group="koordinator-kunjungan">
	<div class="pt-8 pb-6 border-b border-neutral-200/80 dark:border-white/[0.04]">
		<div class="max-w-5xl mx-auto px-4 sm:px-6">
			<div class="flex items-end justify-between gap-4 flex-wrap">
				<div>
					<div class="flex items-center gap-2 text-sm text-neutral-500 mb-3"><a href="/app/koordinator-guru" use:inertia class="hover:text-brand-600">Koordinator Guru</a><span>/</span><span class="text-neutral-700 dark:text-neutral-300">Kunjungan</span></div>
					<h1 class="text-2xl sm:text-3xl font-bold text-neutral-900 dark:text-white tracking-tight">Kunjungan Kelas</h1>
					<p class="mt-2 text-neutral-600 dark:text-neutral-400">Jadwalkan kunjungan, nilai pengajaran, kirim hasilnya ke guru, dan pantau tindak lanjutnya.</p>
				</div>
				<button onclick={openCreate} class="inline-flex items-center gap-2 px-4 py-2.5 rounded-xl bg-brand-600 hover:bg-brand-700 text-white text-sm font-semibold"><Plus class="w-4 h-4" /> Jadwalkan</button>
			</div>
			<div class="mt-6 -mb-6 flex gap-1 overflow-x-auto" role="tablist">
				{#each TABS as t (t.id)}
					<button role="tab" aria-selected={tab === t.id} onclick={() => pilihTab(t.id)} class="shrink-0 px-3.5 py-2.5 text-sm font-medium border-b-2 transition-colors {tab === t.id ? 'border-brand-600 text-brand-700 dark:border-brand-400 dark:text-brand-400' : 'border-transparent text-neutral-500 hover:text-neutral-800 dark:hover:text-neutral-200'}">
						{t.label}{#if t.id !== "kunjungan" && t.id !== "rekap" && counts[t.id] > 0}<span class="ml-1.5 inline-flex min-w-5 justify-center rounded-full bg-amber-500/15 px-1.5 text-xs text-amber-700 dark:text-amber-400">{counts[t.id]}</span>{/if}
					</button>
				{/each}
			</div>
		</div>
	</div>

	<div class="max-w-5xl mx-auto px-4 sm:px-6 py-8 space-y-4">
		{#if success}<div class="rounded-xl bg-green-500/10 border border-green-500/20 p-4 text-sm font-medium text-green-700 dark:text-green-400" in:fly={{ y: 10, duration: 200 }}>{success}</div>{/if}
		{#if error}<div class="rounded-xl bg-red-500/10 border border-red-500/20 p-4 text-sm font-medium text-red-700 dark:text-red-400">{error}</div>{/if}

		{#if tab === "kunjungan"}
			<div class="space-y-3">
				{#each dashboard.kunjungan as k (k.id)}
					{@render kartu(k)}
				{:else}
					<div class="rounded-2xl border border-dashed border-neutral-300 dark:border-neutral-700 p-12 text-center text-sm text-neutral-500"><BookOpen class="w-8 h-8 mx-auto mb-2 opacity-40" /> Belum ada kunjungan terjadwal.</div>
				{/each}
			</div>
		{:else if tab === "rekap"}
			<p class="text-sm text-neutral-600 dark:text-neutral-400">Hanya menghitung hasil yang sudah dikirim ke guru. Urut dari nilai terakhir terendah. Skala 1–4.</p>
			<section class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 overflow-x-auto">
				<table class="w-full min-w-[760px] text-sm">
					<thead class="bg-neutral-50 dark:bg-neutral-900/50 text-xs uppercase tracking-wider text-neutral-500">
						<tr>
							<th class="text-left p-4">Guru</th>
							<th class="text-right p-4">Kunjungan</th>
							<th class="text-right p-4">Nilai terakhir</th>
							<th class="text-left p-4">Tren</th>
							{#each dashboard.rekap[0]?.rata_rata_aspek ?? [] as a (a.kode)}<th class="text-right p-4 normal-case tracking-normal font-medium">{a.label}</th>{/each}
							<th class="text-right p-4">TL terbuka</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-neutral-100 dark:divide-neutral-800">
						{#each dashboard.rekap as r (r.guru_id)}
							<tr>
								<td class="p-4"><p class="font-semibold text-neutral-900 dark:text-white">{r.guru_nama}</p><p class="text-xs text-neutral-500">Terakhir {r.tanggal_terakhir}</p></td>
								<td class="p-4 text-right tabular-nums">{r.jumlah_kunjungan}</td>
								<td class="p-4 text-right"><p class="font-semibold tabular-nums text-neutral-900 dark:text-white">{nilai(r.nilai_terakhir)}</p><p class="text-xs {predikatColor(r.predikat_terakhir)}">{r.predikat_terakhir}</p></td>
								<td class="p-4">
									{#if r.tren === "naik"}<span class="inline-flex items-center gap-1 text-green-700 dark:text-green-400"><TrendingUp class="w-4 h-4" /> dari {nilai(r.nilai_sebelumnya)}</span>
									{:else if r.tren === "turun"}<span class="inline-flex items-center gap-1 text-red-600 dark:text-red-400"><TrendingDown class="w-4 h-4" /> dari {nilai(r.nilai_sebelumnya)}</span>
									{:else if r.tren === "tetap"}<span class="inline-flex items-center gap-1 text-neutral-500"><Minus class="w-4 h-4" /> tetap</span>
									{:else}<span class="text-xs text-neutral-400">Baru 1 kunjungan</span>{/if}
								</td>
								{#each r.rata_rata_aspek as a (a.kode)}<td class="p-4 text-right tabular-nums {a.rata_rata < 2.5 ? 'text-red-600 dark:text-red-400 font-semibold' : 'text-neutral-700 dark:text-neutral-300'}">{nilai(a.rata_rata)}</td>{/each}
								<td class="p-4 text-right tabular-nums">{r.tindak_lanjut_terbuka || "-"}</td>
							</tr>
						{:else}
							<tr><td colspan="9" class="p-12 text-center text-neutral-500">Belum ada hasil kunjungan yang dikirim ke guru.</td></tr>
						{/each}
					</tbody>
				</table>
			</section>
		{:else if tab === "tindak-lanjut"}
			<div class="space-y-3">
				{#each dashboard.tindak_lanjut_terbuka as tl (tl.id)}
					<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-5 flex items-start justify-between gap-4 flex-wrap">
						<div class="min-w-0 flex-1">
							<div class="flex items-center gap-2 flex-wrap">
								<h2 class="font-semibold text-neutral-900 dark:text-white">{tl.jenis_label}</h2>
								{#if tl.internal}<span class="inline-flex px-2 py-0.5 rounded-full text-xs font-medium bg-neutral-500/10 text-neutral-600 dark:text-neutral-400">Internal</span>{/if}
								{#if tl.terlambat}<span class="inline-flex px-2 py-0.5 rounded-full text-xs font-medium bg-red-500/10 text-red-600 dark:text-red-400">Lewat target</span>{/if}
							</div>
							<p class="mt-1 text-sm text-neutral-600 dark:text-neutral-400">{tl.guru_nama} · {tl.kelas_nama || "Tanpa kelas"} · kunjungan {tl.kunjungan_tanggal}</p>
							{#if tl.catatan}<p class="mt-1.5 text-sm text-neutral-700 dark:text-neutral-300">{tl.catatan}</p>{/if}
							<p class="mt-1 text-xs text-neutral-500">{tl.target_tanggal ? `Target ${tl.target_tanggal}` : "Tanpa target tanggal"}</p>
						</div>
						<div class="flex items-center gap-2">
							<a href={`/app/koordinator-guru/kunjungan/${tl.kunjungan_id}`} use:inertia class="px-3 py-2 rounded-lg text-sm font-medium text-neutral-600 dark:text-neutral-300 hover:bg-neutral-100 dark:hover:bg-neutral-800">Lihat kunjungan</a>
							<button onclick={() => setStatusTL(tl.kunjungan_id, tl.id, "selesai")} class="inline-flex items-center gap-1.5 px-3 py-2 rounded-lg text-sm font-semibold text-green-700 hover:bg-green-500/10 dark:text-green-400"><CheckCircle2 class="w-4 h-4" /> Selesai</button>
						</div>
					</div>
				{:else}
					<div class="rounded-2xl border border-dashed border-neutral-300 dark:border-neutral-700 p-12 text-center text-sm text-neutral-500"><CheckCircle2 class="w-8 h-8 mx-auto mb-2 opacity-40" /> Tidak ada tindak lanjut terbuka.</div>
				{/each}
			</div>
		{:else}
			<p class="text-sm text-neutral-600 dark:text-neutral-400">Hasil yang sudah dikirim tetapi belum ditanggapi guru. "Terkirim · belum dibaca" berarti guru belum membuka hasilnya.</p>
			<div class="space-y-3">
				{#each dashboard.belum_ditanggapi as k (k.id)}
					{@render kartu(k)}
				{:else}
					<div class="rounded-2xl border border-dashed border-neutral-300 dark:border-neutral-700 p-12 text-center text-sm text-neutral-500"><RotateCcw class="w-8 h-8 mx-auto mb-2 opacity-40" /> Semua hasil yang dikirim sudah ditanggapi guru.</div>
				{/each}
			</div>
		{/if}
	</div>

	{#if showForm}
		<div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50" role="presentation" onclick={() => (showForm = false)}>
			<div class="w-full max-w-lg max-h-[90vh] overflow-y-auto rounded-2xl bg-white dark:bg-neutral-925 border border-neutral-200 dark:border-white/[0.06] shadow-xl" role="dialog" aria-modal="true" tabindex="-1" onclick={(e) => e.stopPropagation()} onkeydown={(e) => e.key === "Escape" && (showForm = false)}>
				<div class="flex items-center justify-between px-5 py-4 border-b border-neutral-200/80 dark:border-white/[0.04]">
					<h2 class="text-lg font-bold text-neutral-900 dark:text-white">Jadwalkan Kunjungan</h2>
					<button onclick={() => (showForm = false)} aria-label="Tutup" class="p-1.5 rounded-lg hover:bg-neutral-100 dark:hover:bg-neutral-800 text-neutral-500"><X class="w-5 h-5" /></button>
				</div>
				<form onsubmit={(e) => { e.preventDefault(); submit(); }} class="p-5 space-y-4">
					<div class="grid sm:grid-cols-2 gap-4">
						<label class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Guru<select bind:value={form.guru_id} onchange={pilihGuru} required class={`${inputClass} mt-1.5`}><option value={0} disabled>Pilih guru...</option>{#each guruList as g (g.id)}<option value={g.id}>{g.nama}</option>{/each}</select></label>
						<label class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Kelas<select bind:value={form.kelas_id} disabled={!form.guru_id} class={`${inputClass} mt-1.5 disabled:opacity-60`}><option value={0}>{form.guru_id ? "—" : "Pilih guru dulu"}</option>{#each kelasGuru as k (k.id)}<option value={k.id}>{k.nama_kelas}</option>{/each}</select>
							{#if form.guru_id && kelasGuru.length === 0}<span class="mt-1 block text-xs font-normal text-amber-700 dark:text-amber-400">Guru ini belum punya kelas aktif. Kunjungan tetap bisa disimpan tanpa kelas.</span>{/if}
						</label>
						<label class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Target mulai<input type="date" bind:value={form.target_mulai} class={`${inputClass} mt-1.5`} /></label>
						<label class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Target selesai<input type="date" bind:value={form.target_selesai} class={`${inputClass} mt-1.5`} /></label>
						<label class="text-sm font-medium text-neutral-700 dark:text-neutral-300 sm:col-span-2">Status<select bind:value={form.status} class={`${inputClass} mt-1.5`}><option value="dijadwalkan">Dijadwalkan</option><option value="terlaksana">Terlaksana</option></select></label>
						{#if form.status === "terlaksana"}
							<label class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Tanggal aktual<input type="date" bind:value={form.tanggal} required class={`${inputClass} mt-1.5`} /></label>
							<label class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Jam masuk<input type="time" bind:value={form.jam} required class={`${inputClass} mt-1.5`} /></label>
						{/if}
					</div>
					<label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300">Catatan{#if form.status === "terlaksana"}<span class="text-error"> *</span>{/if}<textarea bind:value={form.catatan} required={form.status === "terlaksana"} rows="2" class={`${inputClass} mt-1.5`}></textarea></label>
					{#if form.status === "terlaksana"}<p class="text-xs text-neutral-500">Setelah disimpan, Anda diarahkan ke halaman detail untuk mengisi penilaian per aspek.</p>{/if}
					<div class="pt-3 border-t border-neutral-200/80 dark:border-white/[0.04] flex justify-end gap-3">
						<button type="button" onclick={() => (showForm = false)} class="px-4 py-2.5 rounded-xl text-sm font-semibold text-neutral-600 dark:text-neutral-300 hover:bg-neutral-100 dark:hover:bg-neutral-800">Batal</button>
						<button type="submit" disabled={saving || !form.guru_id} class="px-5 py-2.5 rounded-xl bg-brand-600 hover:bg-brand-700 text-white text-sm font-semibold disabled:opacity-50">{saving ? "Menyimpan..." : "Simpan"}</button>
					</div>
				</form>
			</div>
		</div>
	{/if}
</AppLayout>
