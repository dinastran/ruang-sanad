<script lang="ts">
	import { inertia, router } from "@inertiajs/svelte";
	import AppLayout from "@layouts/AppLayout.svelte";
	import type { Flash, User } from "@lib/types";
	import { fmt } from "@components/lead-dashboard/types";
	import { ArrowLeft, Info, Plus, RefreshCw, X } from "lucide-svelte";

	interface Pattern { id: number; teks: string }
	interface Source { id: number; nama: string; keterangan: string; is_aktif: boolean; patterns: Pattern[]; total_leads: number }
	interface Props { user?: User; sources: Source[]; flash?: Flash }

	let props: Props = $props();
	let canEdit = $derived(props.user?.role === "cs" || props.user?.role === "super_admin");
	let sources = $derived(props.sources ?? []);

	let busy = $state(false);
	let newSource = $state({ nama: "", keterangan: "" });
	let editingId = $state<number | null>(null);
	let editForm = $state({ nama: "", keterangan: "", is_aktif: true });
	// One draft trigger text per source card.
	let drafts = $state<Record<number, string>>({});

	const opts = (onSuccess?: () => void) => ({ preserveScroll: true, onFinish: () => (busy = false), onSuccess });
	function addSource(e: Event) {
		e.preventDefault();
		busy = true;
		router.post("/app/leads/sumber", { ...newSource, is_aktif: true }, opts(() => (newSource = { nama: "", keterangan: "" })));
	}
	function startEdit(s: Source) {
		editForm = { nama: s.nama, keterangan: s.keterangan, is_aktif: s.is_aktif };
		editingId = s.id;
	}
	function saveEdit(e: Event, s: Source) {
		e.preventDefault();
		busy = true;
		router.put(`/app/leads/sumber/${s.id}`, editForm, opts(() => (editingId = null)));
	}
	function toggleAktif(s: Source) {
		busy = true;
		router.put(`/app/leads/sumber/${s.id}`, { nama: s.nama, keterangan: s.keterangan, is_aktif: !s.is_aktif }, opts());
	}
	function hapus(s: Source) {
		if (!confirm(`Hapus sumber "${s.nama}" beserta teks pemicunya?`)) return;
		busy = true;
		router.delete(`/app/leads/sumber/${s.id}`, opts());
	}
	function addPattern(e: Event, s: Source) {
		e.preventDefault();
		busy = true;
		router.post(`/app/leads/sumber/${s.id}/pola`, { teks: drafts[s.id] ?? "" }, opts(() => (drafts[s.id] = "")));
	}
	function removePattern(p: Pattern) {
		if (!confirm(`Hapus teks pemicu "${p.teks}"? Lead yang sudah tercatat tidak berubah.`)) return;
		busy = true;
		router.delete(`/app/leads/sumber/pola/${p.id}`, opts());
	}
	function rematch() {
		if (!confirm("Cocokkan ulang semua lead yang belum punya sumber memakai pesan pertamanya? Lead yang sumbernya diisi manual oleh CS tidak diubah.")) return;
		busy = true;
		router.post("/app/leads/sumber/cocokkan", {}, opts());
	}

	const panel = "rounded-xl border border-neutral-200/80 bg-white dark:border-white/[0.07] dark:bg-neutral-925/60";
	const field = "h-9 w-full rounded-md border border-neutral-200 bg-neutral-50 px-2.5 text-sm text-neutral-900 placeholder:text-neutral-400 focus:border-brand-500 focus:outline-none focus:ring-2 focus:ring-brand-500/30 dark:border-white/10 dark:bg-neutral-950 dark:text-white";
	const btn = "inline-flex h-9 items-center justify-center gap-1.5 rounded-md px-3 text-sm font-medium transition-colors duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 disabled:opacity-50";
	const btnPrimary = btn + " bg-brand-600 text-white hover:bg-brand-700 dark:bg-brand-500";
	const btnGhost = btn + " border border-neutral-200 text-neutral-700 hover:bg-neutral-50 dark:border-white/10 dark:text-neutral-300 dark:hover:bg-white/[0.04]";
	const lbl = "block text-[11px] font-medium text-neutral-500 dark:text-neutral-400";
</script>

<svelte:head><title>Sumber Lead · Ruang Sanad</title></svelte:head>
<AppLayout user={props.user}>
	<div class="mx-auto max-w-4xl space-y-4 px-4 py-6 sm:px-6 lg:py-8">
		<header class="flex flex-col gap-3 pb-1 sm:flex-row sm:items-end sm:justify-between">
			<div class="min-w-0">
				<a href="/app/leads" use:inertia class="inline-flex items-center gap-1 text-xs font-medium text-neutral-500 hover:text-neutral-800 dark:text-neutral-400 dark:hover:text-white"><ArrowLeft size={13} /> Leads</a>
				<h1 class="mt-1 text-[28px] font-semibold leading-tight tracking-[-0.02em] text-neutral-950 dark:text-white">Sumber lead</h1>
				<p class="mt-1 max-w-2xl text-sm text-neutral-500 dark:text-neutral-400">LP atau iklan asal calon santri, dikenali dari teks pesan WhatsApp pertama yang sudah terisi dari tombol LP/iklan.</p>
			</div>
			{#if canEdit}
				<button type="button" class={btnGhost + " shrink-0 whitespace-nowrap"} disabled={busy} onclick={rematch}><RefreshCw size={15} /> Cocokkan lead tanpa sumber</button>
			{/if}
		</header>

		{#if props.flash?.success}<div role="status" class="rounded-lg border border-success/20 bg-success/10 px-4 py-3 text-sm font-medium text-emerald-800 dark:text-emerald-300">{props.flash.success}</div>{/if}
		{#if props.flash?.error}<div role="alert" class="rounded-lg border border-error/20 bg-error/10 px-4 py-3 text-sm font-medium text-red-700 dark:text-red-300">{props.flash.error}</div>{/if}

		<details class={panel + " px-4 py-3 text-sm text-neutral-600 sm:px-5 dark:text-neutral-300"}>
			<summary class="flex cursor-pointer items-center gap-2 font-medium text-neutral-800 dark:text-neutral-100"><Info size={15} class="text-brand-600 dark:text-brand-400" /> Cara kerja dan tips</summary>
			<ul class="mt-2 list-disc space-y-1.5 pl-5 text-xs leading-relaxed text-neutral-500 dark:text-neutral-400">
				<li>Hanya <strong>pesan pertama</strong> lead yang dicocokkan. Huruf besar/kecil, tanda baca, dan salam di depan diabaikan; cukup isi bagian khas dari teks LP, mis. <em>kelas reguler tahsin ruang sanad</em>.</li>
				<li>Bila beberapa teks cocok, teks yang paling panjang (paling spesifik) yang dipakai.</li>
				<li>Sumber tersimpan permanen di lead. Mengubah atau menghapus teks pemicu tidak mengubah lead lama; gunakan <strong>Cocokkan lead tanpa sumber</strong> untuk mengisi lead yang belum punya sumber.</li>
				<li>Teks bawaan iklan Meta (“Bisakah saya mendapatkan info selengkapnya tentang ini?”) sama untuk semua iklan. Agar tiap iklan/LP bisa dibedakan, isi teks pembuka yang unik di setiap iklan.</li>
			</ul>
		</details>

		{#if canEdit}
			<form class={panel + " grid gap-3 p-4 sm:grid-cols-[1fr_1.4fr_auto] sm:items-end sm:p-5"} onsubmit={addSource}>
				<label class={lbl}>Nama sumber baru<input class={field + " mt-1"} bind:value={newSource.nama} maxlength="80" placeholder="mis. LP Tahsin Anak" required /></label>
				<label class={lbl}>Keterangan (opsional)<input class={field + " mt-1"} bind:value={newSource.keterangan} maxlength="300" placeholder="mis. iklan Meta kampanye Oktober" /></label>
				<button type="submit" class={btnPrimary} disabled={busy || newSource.nama.trim().length < 2}><Plus size={15} /> Tambah</button>
			</form>
		{/if}

		{#if sources.length === 0}
			<p class="rounded-xl border border-dashed border-neutral-300 px-6 py-12 text-center text-sm text-neutral-500 dark:border-white/10 dark:text-neutral-400">Belum ada sumber lead. Tambahkan nama LP/iklan, lalu isi teks pemicunya.</p>
		{/if}

		<ul class="space-y-3">
			{#each sources as s (s.id)}
				<li class={panel + " p-4 sm:p-5 " + (s.is_aktif ? "" : "opacity-75")}>
					{#if editingId === s.id}
						<form class="grid gap-3 sm:grid-cols-[1fr_1.4fr]" onsubmit={(e) => saveEdit(e, s)}>
							<label class={lbl}>Nama<input class={field + " mt-1"} bind:value={editForm.nama} maxlength="80" required /></label>
							<label class={lbl}>Keterangan<input class={field + " mt-1"} bind:value={editForm.keterangan} maxlength="300" /></label>
							<div class="flex justify-end gap-2 sm:col-span-2">
								<button type="button" class={btnGhost} onclick={() => (editingId = null)}>Batal</button>
								<button type="submit" class={btnPrimary} disabled={busy}>Simpan</button>
							</div>
						</form>
					{:else}
						<div class="flex flex-wrap items-start justify-between gap-3">
							<div class="min-w-0">
								<div class="flex flex-wrap items-center gap-2">
									<h2 class="text-base font-semibold text-neutral-950 dark:text-white">{s.nama}</h2>
									{#if !s.is_aktif}<span class="rounded-full bg-neutral-200 px-2 py-0.5 text-[11px] font-medium text-neutral-600 dark:bg-neutral-800 dark:text-neutral-400">Nonaktif</span>{/if}
								</div>
								{#if s.keterangan}<p class="mt-0.5 text-xs text-neutral-500 dark:text-neutral-400">{s.keterangan}</p>{/if}
							</div>
							<a href={`/app/leads?status=all&sumber=${s.id}`} use:inertia class="shrink-0 rounded text-right focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500">
								<span class="block text-xl font-semibold tabular-nums text-neutral-950 dark:text-white">{fmt(s.total_leads)}</span>
								<span class="block text-[11px] text-brand-700 hover:underline dark:text-brand-400">lead tercatat</span>
							</a>
						</div>

						<div class="mt-3">
							<p class={lbl}>Teks pemicu</p>
							{#if s.patterns.length === 0}
								<p class="mt-1 text-xs text-amber-700 dark:text-amber-400">Belum ada teks pemicu; sumber ini tidak akan dikenali otomatis.</p>
							{:else}
								<ul class="mt-1.5 flex flex-wrap gap-1.5">
									{#each s.patterns as p (p.id)}
										<li class="inline-flex max-w-full items-center gap-1 rounded-md border border-neutral-200 bg-neutral-50 py-1 pl-2 pr-1 text-xs text-neutral-700 dark:border-white/10 dark:bg-white/[0.03] dark:text-neutral-300">
											<span class="break-words">“{p.teks}”</span>
											{#if canEdit}<button type="button" class="rounded p-0.5 text-neutral-400 hover:text-red-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500" aria-label={`Hapus teks pemicu ${p.teks}`} disabled={busy} onclick={() => removePattern(p)}><X size={12} /></button>{/if}
										</li>
									{/each}
								</ul>
							{/if}
							{#if canEdit}
								<form class="mt-2 flex gap-2" onsubmit={(e) => addPattern(e, s)}>
									<input class={field} bind:value={drafts[s.id]} maxlength="300" placeholder="Tempel teks pesan dari LP/iklan ini" aria-label={`Teks pemicu baru untuk ${s.nama}`} />
									<button type="submit" class={btnGhost + " shrink-0"} disabled={busy || !(drafts[s.id] ?? "").trim()}><Plus size={14} /> Teks</button>
								</form>
							{/if}
						</div>

						{#if canEdit}
							<div class="mt-3 flex flex-wrap gap-2 border-t border-neutral-100 pt-3 dark:border-white/[0.05]">
								<button type="button" class={btnGhost + " h-8 text-xs"} disabled={busy || editingId !== null} onclick={() => startEdit(s)}>Ubah</button>
								<button type="button" class={btnGhost + " h-8 text-xs"} disabled={busy} onclick={() => toggleAktif(s)}>{s.is_aktif ? "Nonaktifkan" : "Aktifkan"}</button>
								{#if s.total_leads === 0}<button type="button" class={btnGhost + " h-8 text-xs !text-red-600"} disabled={busy} onclick={() => hapus(s)}>Hapus</button>{/if}
							</div>
						{/if}
					{/if}
				</li>
			{/each}
		</ul>
	</div>
</AppLayout>
