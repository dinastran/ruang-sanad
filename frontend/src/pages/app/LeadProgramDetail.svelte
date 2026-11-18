<script lang="ts">
	import { inertia, router } from "@inertiajs/svelte";
	import AppLayout from "@layouts/AppLayout.svelte";
	import type { Flash, User } from "@lib/types";
	import { Toast } from "@lib/notifications/toast";
	import { getCSRFToken } from "@lib/utils/csrf";
	import { fmt } from "@components/lead-dashboard/types";
	import { ArrowLeft, Check, Copy, ExternalLink, Pencil, Plus, Trash2, UserRoundPlus } from "lucide-svelte";

	interface Program {
		id: number;
		nama: string;
		jenis: "reguler" | "program";
		produk_id: number;
		produk_nama: string;
		produk_harga: number;
		produk_batch_id: number;
		batch_nama: string;
		keterangan: string;
		is_aktif: boolean;
		sumber_ids: number[];
	}
	interface Sesi { id: number; ke: number; judul: string; tanggal: string; jam_mulai: string; token: string; hadir: number }
	interface Peserta { order_id: number; order_no: string; nama: string; no_wa: string; tanggal: string; lead_id: number; lead_status: string; hadir: number[] }
	interface Produk { id: number; nama: string; harga: number; is_aktif: boolean; batches: { id: number; nama: string }[] }
	interface Props {
		user?: User;
		program: Program;
		sesi: Sesi[];
		peserta: Peserta[];
		sources: { id: number; nama: string; is_aktif: boolean }[];
		produk: Produk[];
		can_edit: boolean;
		flash?: Flash;
	}

	let props: Props = $props();
	let program = $derived(props.program);
	let sesi = $derived(props.sesi ?? []);
	let isPendek = $derived(program.jenis === "program");
	let busy = $state(false);
	const opts = (onSuccess?: () => void) => ({ preserveScroll: true, onFinish: () => (busy = false), onSuccess });

	// --- Edit program ---
	let editing = $state(false);
	let form = $state({ nama: "", produk_id: 0, produk_batch_id: 0, keterangan: "", is_aktif: true });
	let produkDipilih = $derived((props.produk ?? []).find((p) => p.id === Number(form.produk_id)));
	function startEdit() {
		form = { nama: program.nama, produk_id: program.produk_id, produk_batch_id: program.produk_batch_id, keterangan: program.keterangan, is_aktif: program.is_aktif };
		editing = true;
	}
	function saveProgram(e: Event) {
		e.preventDefault();
		busy = true;
		router.put(`/app/leads/program/${program.id}`, { ...form, jenis: program.jenis, produk_id: Number(form.produk_id), produk_batch_id: Number(form.produk_batch_id) }, opts(() => (editing = false)));
	}

	// --- Sources ---
	// svelte-ignore state_referenced_locally
	let sumber = $state<number[]>([...props.program.sumber_ids]);
	let sumberBerubah = $derived(JSON.stringify([...sumber].sort()) !== JSON.stringify([...program.sumber_ids].sort()));
	function saveSumber() {
		busy = true;
		router.put(`/app/leads/program/${program.id}/sumber`, { sumber_ids: sumber }, opts());
	}

	// --- Sessions ---
	let sesiForm = $state<{ id: number; ke: number; tanggal: string; jam_mulai: string; judul: string } | null>(null);
	function newSesi() {
		const ke = sesi.reduce((m, s) => Math.max(m, s.ke), 0) + 1;
		sesiForm = { id: 0, ke, tanggal: "", jam_mulai: sesi[sesi.length - 1]?.jam_mulai ?? "", judul: "" };
	}
	function saveSesi(e: Event) {
		e.preventDefault();
		if (!sesiForm) return;
		busy = true;
		const data = { ke: Number(sesiForm.ke), tanggal: sesiForm.tanggal, jam_mulai: sesiForm.jam_mulai, judul: sesiForm.judul };
		if (sesiForm.id) router.put(`/app/leads/program/sesi/${sesiForm.id}?program=${program.id}`, data, opts(() => (sesiForm = null)));
		else router.post(`/app/leads/program/${program.id}/sesi`, data, opts(() => (sesiForm = null)));
	}
	function hapusSesi(s: Sesi) {
		if (!confirm(`Hapus sesi ke-${s.ke}? Link absensinya tidak berlaku lagi.`)) return;
		busy = true;
		router.delete(`/app/leads/program/sesi/${s.id}?program=${program.id}`, opts());
	}
	const linkSesi = (s: Sesi) => `${window.location.origin}/hadir/${s.token}`;
	let tersalin = $state<number | null>(null);
	async function salin(s: Sesi) {
		try {
			await navigator.clipboard.writeText(linkSesi(s));
			tersalin = s.id;
			setTimeout(() => (tersalin === s.id ? (tersalin = null) : null), 2000);
		} catch {
			Toast("Gagal menyalin; salin manual dari kotak link.", "error");
		}
	}
	const fmtTanggal = (t: string) => {
		const d = new Date(t + "T00:00:00");
		return Number.isNaN(d.getTime()) ? t : d.toLocaleDateString("id-ID", { weekday: "short", day: "numeric", month: "short" });
	};

	// --- Participants & attendance (toggled in place with fetch) ---
	// svelte-ignore state_referenced_locally
	let hadir = $state<Record<string, boolean>>(Object.fromEntries((props.peserta ?? []).flatMap((p) => p.hadir.map((sid) => [`${p.order_id}:${sid}`, true]))));
	let peserta = $derived(props.peserta ?? []);
	const totalHadir = (p: Peserta) => sesi.filter((s) => hadir[`${p.order_id}:${s.id}`]).length;
	const hadirSesi = (s: Sesi) => peserta.filter((p) => hadir[`${p.order_id}:${s.id}`]).length;
	async function toggleHadir(p: Peserta, s: Sesi) {
		const key = `${p.order_id}:${s.id}`;
		const next = !hadir[key];
		hadir[key] = next;
		try {
			const res = await fetch(`/app/leads/program/sesi/${s.id}/hadir/${p.order_id}`, {
				method: "PUT",
				headers: { "Content-Type": "application/json", Accept: "application/json", "X-XSRF-TOKEN": getCSRFToken() },
				body: JSON.stringify({ hadir: next }),
			});
			if (!res.ok) throw new Error((await res.json().catch(() => ({}))).error || "Gagal menyimpan");
		} catch (e) {
			hadir[key] = !next;
			Toast(e instanceof Error ? e.message : "Gagal menyimpan", "error");
		}
	}
	function tawarkan(p: Peserta) {
		if (!confirm(`Buka lagi lead ${p.nama} di Tahsin Reguler untuk ditawari kelas reguler?`)) return;
		busy = true;
		router.post(`/app/leads/${p.lead_id}/tawarkan-reguler?program=${program.id}`, {}, opts());
	}

	const panel = "rounded-xl border border-neutral-200/80 bg-white dark:border-white/[0.07] dark:bg-neutral-925/60";
	const field = "h-9 w-full rounded-md border border-neutral-200 bg-neutral-50 px-2.5 text-sm text-neutral-900 placeholder:text-neutral-400 focus:border-brand-500 focus:outline-none focus:ring-2 focus:ring-brand-500/30 dark:border-white/10 dark:bg-neutral-950 dark:text-white";
	const btn = "inline-flex h-9 items-center justify-center gap-1.5 rounded-md px-3 text-sm font-medium transition-colors duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 disabled:opacity-50";
	const btnPrimary = btn + " bg-brand-600 text-white hover:bg-brand-700 dark:bg-brand-500";
	const btnGhost = btn + " border border-neutral-200 text-neutral-700 hover:bg-neutral-50 dark:border-white/10 dark:text-neutral-300 dark:hover:bg-white/[0.04]";
	const lbl = "block text-[11px] font-medium text-neutral-500 dark:text-neutral-400";
	const h2 = "text-base font-semibold text-neutral-950 dark:text-white";
</script>

<svelte:head><title>{program.nama} · Program CRM</title></svelte:head>
<AppLayout user={props.user}>
	<div class="mx-auto max-w-5xl space-y-4 px-4 py-6 sm:px-6 lg:py-8">
		<header class="flex flex-col gap-3 pb-1 sm:flex-row sm:items-end sm:justify-between">
			<div class="min-w-0">
				<a href="/app/leads/program" use:inertia class="inline-flex items-center gap-1 text-xs font-medium text-neutral-500 hover:text-neutral-800 dark:text-neutral-400 dark:hover:text-white"><ArrowLeft size={13} /> Program</a>
				<div class="mt-1 flex flex-wrap items-center gap-2">
					<h1 class="text-[28px] font-semibold leading-tight tracking-[-0.02em] text-neutral-950 dark:text-white">{program.nama}</h1>
					{#if !program.is_aktif}<span class="rounded-full bg-neutral-200 px-2 py-0.5 text-[11px] font-medium text-neutral-600 dark:bg-neutral-800 dark:text-neutral-400">Nonaktif</span>{/if}
				</div>
				<p class="mt-1 text-sm text-neutral-500 dark:text-neutral-400">
					{#if isPendek}Program pendek · closing jadi peserta ({program.produk_nama || "produk belum dipilih"}{program.batch_nama ? ` · ${program.batch_nama}` : ""}{program.produk_harga ? ` · Rp${fmt(program.produk_harga)}` : ""}){:else}Program bawaan · closing jadi santri kelas reguler{/if}
				</p>
			</div>
			<div class="flex shrink-0 flex-wrap gap-2">
				<a href={`/app/leads?view=list&status=all&program=${program.id}`} use:inertia class={btnGhost}>Lihat lead program</a>
				{#if props.can_edit && !editing}<button type="button" class={btnGhost} onclick={startEdit}><Pencil size={14} /> Ubah</button>{/if}
			</div>
		</header>

		{#if props.flash?.success}<div role="status" class="rounded-lg border border-success/20 bg-success/10 px-4 py-3 text-sm font-medium text-emerald-800 dark:text-emerald-300">{props.flash.success}</div>{/if}
		{#if props.flash?.error}<div role="alert" class="rounded-lg border border-error/20 bg-error/10 px-4 py-3 text-sm font-medium text-red-700 dark:text-red-300">{props.flash.error}</div>{/if}

		{#if editing}
			<form class={panel + " grid gap-3 p-4 sm:grid-cols-2 sm:p-5"} onsubmit={saveProgram}>
				<label class={lbl + " sm:col-span-2"}>Nama program<input class={field + " mt-1"} bind:value={form.nama} maxlength="80" required /></label>
				{#if isPendek}
					<label class={lbl}>Produk program
						<select class={field + " mt-1"} bind:value={form.produk_id} onchange={() => (form.produk_batch_id = 0)}>
							{#each props.produk ?? [] as p (p.id)}<option value={p.id}>{p.nama}{p.harga ? ` · Rp${fmt(p.harga)}` : ""}</option>{/each}
						</select>
					</label>
					<label class={lbl}>Batch
						<select class={field + " mt-1"} bind:value={form.produk_batch_id} disabled={!produkDipilih || produkDipilih.batches.length === 0}>
							<option value={0}>Semua batch</option>
							{#each produkDipilih?.batches ?? [] as b (b.id)}<option value={b.id}>{b.nama}</option>{/each}
						</select>
					</label>
				{/if}
				<label class={lbl + " sm:col-span-2"}>Keterangan<input class={field + " mt-1"} bind:value={form.keterangan} maxlength="300" /></label>
				{#if isPendek}<label class="flex items-center gap-2 text-sm text-neutral-700 dark:text-neutral-300"><input type="checkbox" bind:checked={form.is_aktif} /> Aktif (nonaktif = link absensi & pilihan program ditutup)</label>{/if}
				<div class="flex justify-end gap-2 sm:col-span-2">
					<button type="button" class={btnGhost} onclick={() => (editing = false)}>Batal</button>
					<button type="submit" class={btnPrimary} disabled={busy}>Simpan</button>
				</div>
			</form>
		{/if}

		{#if isPendek}
			<section class={panel + " p-4 sm:p-5"} aria-labelledby="sumber-title">
				<h2 id="sumber-title" class={h2}>Sumber lead program ini</h2>
				<p class="mt-0.5 text-xs text-neutral-500 dark:text-neutral-400">Lead dari sumber yang dicentang otomatis masuk program ini (kecuali programnya dipilih manual di halaman lead).</p>
				<div class="mt-3 flex flex-wrap gap-2">
					{#each props.sources ?? [] as s (s.id)}
						<label class="inline-flex items-center gap-2 rounded-md border px-2.5 py-1.5 text-xs {sumber.includes(s.id) ? 'border-brand-500 bg-brand-500/10 text-brand-800 dark:text-brand-200' : 'border-neutral-200 text-neutral-700 dark:border-white/10 dark:text-neutral-300'}">
							<input type="checkbox" value={s.id} bind:group={sumber} disabled={!props.can_edit} />{s.nama}{s.is_aktif ? "" : " (nonaktif)"}
						</label>
					{/each}
				</div>
				{#if props.can_edit && sumberBerubah}
					<div class="mt-3 flex justify-end"><button type="button" class={btnPrimary} disabled={busy} onclick={saveSumber}>Simpan sumber</button></div>
				{/if}
			</section>

			<section class={panel + " p-4 sm:p-5"} aria-labelledby="sesi-title">
				<div class="flex flex-wrap items-center justify-between gap-2">
					<div>
						<h2 id="sesi-title" class={h2}>Sesi & link absensi</h2>
						<p class="mt-0.5 text-xs text-neutral-500 dark:text-neutral-400">Bagikan link di chat Zoom. Link aktif 30 menit sebelum jam mulai sampai 3 jam sesudahnya; peserta mengisi nomor WhatsApp pendaftaran.</p>
					</div>
					{#if props.can_edit && !sesiForm}<button type="button" class={btnGhost} onclick={newSesi}><Plus size={14} /> Sesi</button>{/if}
				</div>
				{#if sesiForm}
					<form class="mt-3 grid gap-3 rounded-lg bg-neutral-50 p-3 sm:grid-cols-[5rem_1fr_7rem_2fr_auto] sm:items-end dark:bg-white/[0.03]" onsubmit={saveSesi}>
						<label class={lbl}>Sesi ke<input type="number" min="1" class={field + " mt-1"} bind:value={sesiForm.ke} required /></label>
						<label class={lbl}>Tanggal<input type="date" class={field + " mt-1"} bind:value={sesiForm.tanggal} required /></label>
						<label class={lbl}>Jam mulai (WIB)<input type="time" class={field + " mt-1"} bind:value={sesiForm.jam_mulai} /></label>
						<label class={lbl}>Judul (opsional)<input class={field + " mt-1"} bind:value={sesiForm.judul} maxlength="120" placeholder="mis. Makharijul huruf Al-Fatihah" /></label>
						<div class="flex gap-2"><button type="button" class={btnGhost} onclick={() => (sesiForm = null)}>Batal</button><button type="submit" class={btnPrimary} disabled={busy || !sesiForm.tanggal}>Simpan</button></div>
					</form>
				{/if}
				{#if sesi.length === 0}
					<p class="mt-3 rounded-lg border border-dashed border-neutral-300 px-4 py-6 text-center text-sm text-neutral-500 dark:border-white/10">Belum ada sesi. Tambahkan sesi untuk tiap tanggal Zoom.</p>
				{:else}
					<ul class="mt-3 divide-y divide-neutral-100 dark:divide-white/[0.05]">
						{#each sesi as s (s.id)}
							<li class="flex flex-col gap-2 py-3 sm:flex-row sm:items-center">
								<div class="min-w-0 flex-1">
									<p class="text-sm font-medium text-neutral-900 dark:text-white">Sesi {s.ke} · {fmtTanggal(s.tanggal)}{s.jam_mulai ? `, ${s.jam_mulai} WIB` : ""}</p>
									<p class="text-xs text-neutral-500">{s.judul || "Tanpa judul"} · hadir <span class="font-semibold tabular-nums text-neutral-700 dark:text-neutral-300">{hadirSesi(s)}</span> dari {peserta.length}</p>
								</div>
								<div class="flex items-center gap-2">
									<code class="hidden max-w-[16rem] truncate rounded bg-neutral-100 px-2 py-1 text-xs text-neutral-700 md:block dark:bg-neutral-900 dark:text-neutral-300">{linkSesi(s)}</code>
									<button type="button" class={btnGhost + " h-8 text-xs"} onclick={() => salin(s)}>{#if tersalin === s.id}<Check size={13} /> Tersalin{:else}<Copy size={13} /> Salin link{/if}</button>
									<a href={linkSesi(s)} target="_blank" rel="noopener noreferrer" class={btnGhost + " h-8 px-2"} aria-label={`Buka link absensi sesi ${s.ke}`}><ExternalLink size={13} /></a>
									{#if props.can_edit}
										<button type="button" class={btnGhost + " h-8 px-2"} aria-label={`Ubah sesi ${s.ke}`} onclick={() => (sesiForm = { id: s.id, ke: s.ke, tanggal: s.tanggal, jam_mulai: s.jam_mulai, judul: s.judul })}><Pencil size={13} /></button>
										{#if s.hadir === 0}<button type="button" class={btnGhost + " h-8 px-2 !text-red-600"} aria-label={`Hapus sesi ${s.ke}`} disabled={busy} onclick={() => hapusSesi(s)}><Trash2 size={13} /></button>{/if}
									{/if}
								</div>
							</li>
						{/each}
					</ul>
				{/if}
			</section>

			<section class={panel + " overflow-hidden"} aria-labelledby="peserta-title">
				<div class="p-4 sm:p-5">
					<h2 id="peserta-title" class={h2}>Peserta & kehadiran <span class="text-sm font-normal text-neutral-500">({peserta.length})</span></h2>
					<p class="mt-0.5 text-xs text-neutral-500 dark:text-neutral-400">Peserta = pesanan aktif produk {program.produk_nama || "program"}{program.batch_nama ? ` (${program.batch_nama})` : ""}: dari tombol Daftarkan peserta di lead, atau dicatat manual di <a href="/app/produk-crm/orders" use:inertia class="underline">Pesanan Produk</a>. Centang untuk koreksi kehadiran.</p>
				</div>
				{#if peserta.length === 0}
					<p class="border-t border-neutral-100 px-4 py-8 text-center text-sm text-neutral-500 dark:border-white/[0.05]">Belum ada peserta.</p>
				{:else}
					<div class="overflow-x-auto border-t border-neutral-100 dark:border-white/[0.05]">
						<table class="w-full text-sm">
							<thead class="bg-neutral-50 text-left text-xs text-neutral-500 dark:bg-neutral-900 dark:text-neutral-400">
								<tr>
									<th class="px-4 py-2.5 font-medium">Peserta</th>
									{#each sesi as s (s.id)}<th class="px-2 py-2.5 text-center font-medium">S{s.ke}<span class="block font-normal">{fmtTanggal(s.tanggal)}</span></th>{/each}
									<th class="px-3 py-2.5 text-center font-medium">Hadir</th>
									<th class="px-4 py-2.5"><span class="sr-only">Aksi</span></th>
								</tr>
							</thead>
							<tbody class="divide-y divide-neutral-100 dark:divide-white/[0.05]">
								{#each peserta as p (p.order_id)}
									<tr>
										<td class="px-4 py-2.5">
											{#if p.lead_id}<a href={`/app/leads/${p.lead_id}`} use:inertia class="font-medium text-neutral-900 hover:text-brand-700 hover:underline dark:text-white">{p.nama}</a>{:else}<span class="font-medium text-neutral-900 dark:text-white">{p.nama}</span>{/if}
											<span class="block text-xs text-neutral-500">{p.no_wa || "tanpa nomor"} · {p.order_no}</span>
										</td>
										{#each sesi as s (s.id)}
											<td class="px-2 py-2.5 text-center">
												<input type="checkbox" class="h-4 w-4 accent-brand-600" checked={!!hadir[`${p.order_id}:${s.id}`]} disabled={!props.can_edit} aria-label={`${p.nama} hadir sesi ${s.ke}`} onchange={() => toggleHadir(p, s)} />
											</td>
										{/each}
										<td class="px-3 py-2.5 text-center font-semibold tabular-nums text-neutral-900 dark:text-white">{totalHadir(p)}/{sesi.length}</td>
										<td class="px-4 py-2.5 text-right">
											{#if p.lead_id && p.lead_status === "won"}
												<button type="button" class={btnGhost + " h-8 whitespace-nowrap text-xs"} disabled={busy} onclick={() => tawarkan(p)}><UserRoundPlus size={13} /> Tawarkan reguler</button>
											{:else if p.lead_id && p.lead_status === "open"}
												<span class="text-xs text-brand-700 dark:text-brand-400">Ditawari reguler</span>
											{/if}
										</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				{/if}
			</section>
		{:else}
			<section class={panel + " p-4 text-sm text-neutral-600 sm:p-5 dark:text-neutral-300"}>
				<p>Semua lead yang sumbernya tidak dipetakan ke program pendek masuk ke <strong>{program.nama}</strong>. Closing lewat tombol <strong>Jadikan santri</strong> di halaman lead (kelas, SPP, dan onboarding seperti biasa).</p>
			</section>
		{/if}
	</div>
</AppLayout>
