<script lang="ts">
	import { inertia, router } from "@inertiajs/svelte";
	import AppLayout from "@layouts/AppLayout.svelte";
	import type { Flash, User } from "@lib/types";
	import { fmt } from "@components/lead-dashboard/types";
	import { ArrowLeft, CalendarRange, ChevronRight, Info, Plus } from "lucide-svelte";

	interface Program {
		id: number;
		nama: string;
		jenis: "reguler" | "program";
		produk_id: number;
		produk_nama: string;
		produk_harga: number;
		batch_nama: string;
		keterangan: string;
		is_aktif: boolean;
		sumber_ids: number[];
		lead_open: number;
		lead_won: number;
		jumlah_sesi: number;
		jumlah_peserta: number;
	}
	interface Produk { id: number; nama: string; harga: number; is_aktif: boolean; batches: { id: number; nama: string; is_aktif?: boolean }[] }
	interface Props { user?: User; programs: Program[]; sources: { id: number; nama: string }[]; produk: Produk[]; can_edit: boolean; flash?: Flash }

	let props: Props = $props();
	let programs = $derived(props.programs ?? []);
	let sourceName = $derived(new Map((props.sources ?? []).map((s) => [s.id, s.nama])));

	let showForm = $state(false);
	let busy = $state(false);
	let form = $state({ nama: "", produk_id: 0, produk_batch_id: 0, keterangan: "", kapasitas_kelas: 30 });
	let produkDipilih = $derived((props.produk ?? []).find((p) => p.id === Number(form.produk_id)));

	function submit(e: Event) {
		e.preventDefault();
		busy = true;
		router.post("/app/leads/program", { ...form, jenis: "program", is_aktif: true, produk_id: Number(form.produk_id), produk_batch_id: Number(form.produk_batch_id), kapasitas_kelas: Number(form.kapasitas_kelas) || 30 }, {
			preserveScroll: true,
			onFinish: () => (busy = false),
		});
	}

	const panel = "rounded-xl border border-neutral-200/80 bg-white dark:border-white/[0.07] dark:bg-neutral-925/60";
	const field = "h-9 w-full rounded-md border border-neutral-200 bg-neutral-50 px-2.5 text-sm text-neutral-900 placeholder:text-neutral-400 focus:border-brand-500 focus:outline-none focus:ring-2 focus:ring-brand-500/30 dark:border-white/10 dark:bg-neutral-950 dark:text-white";
	const btn = "inline-flex h-9 items-center justify-center gap-1.5 rounded-md px-3 text-sm font-medium transition-colors duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 disabled:opacity-50";
	const btnPrimary = btn + " bg-brand-600 text-white hover:bg-brand-700 dark:bg-brand-500";
	const btnGhost = btn + " border border-neutral-200 text-neutral-700 hover:bg-neutral-50 dark:border-white/10 dark:text-neutral-300 dark:hover:bg-white/[0.04]";
	const lbl = "block text-[11px] font-medium text-neutral-500 dark:text-neutral-400";
</script>

<svelte:head><title>Program CRM · Ruang Sanad</title></svelte:head>
<AppLayout user={props.user}>
	<div class="mx-auto max-w-4xl space-y-4 px-4 py-6 sm:px-6 lg:py-8">
		<header class="flex flex-col gap-3 pb-1 sm:flex-row sm:items-end sm:justify-between">
			<div class="min-w-0">
				<a href="/app/leads?view=list" use:inertia class="inline-flex items-center gap-1 text-xs font-medium text-neutral-500 hover:text-neutral-800 dark:text-neutral-400 dark:hover:text-white"><ArrowLeft size={13} /> Leads</a>
				<h1 class="mt-1 text-[28px] font-semibold leading-tight tracking-[-0.02em] text-neutral-950 dark:text-white">Program</h1>
				<p class="mt-1 max-w-2xl text-sm text-neutral-500 dark:text-neutral-400">Program yang dipasarkan lewat CRM. Lead Tahsin Reguler closing menjadi santri; lead program pendek closing menjadi peserta tanpa kelas atau SPP.</p>
			</div>
			{#if props.can_edit && !showForm}
				<button type="button" class={btnPrimary + " shrink-0"} onclick={() => (showForm = true)}><Plus size={15} /> Program pendek</button>
			{/if}
		</header>

		{#if props.flash?.success}<div role="status" class="rounded-lg border border-success/20 bg-success/10 px-4 py-3 text-sm font-medium text-emerald-800 dark:text-emerald-300">{props.flash.success}</div>{/if}
		{#if props.flash?.error}<div role="alert" class="rounded-lg border border-error/20 bg-error/10 px-4 py-3 text-sm font-medium text-red-700 dark:text-red-300">{props.flash.error}</div>{/if}

		<details class={panel + " px-4 py-3 text-sm sm:px-5"}>
			<summary class="flex cursor-pointer items-center gap-2 font-medium text-neutral-800 dark:text-neutral-100"><Info size={15} class="text-brand-600 dark:text-brand-400" /> Cara kerja</summary>
			<ol class="mt-2 list-decimal space-y-1.5 pl-5 text-xs leading-relaxed text-neutral-500 dark:text-neutral-400">
				<li>Buat produk berkategori <strong>program</strong> di Produk CRM (mis. "Sepekan Bersama Al-Fatihah", harga sesuai LP), lalu buat program di sini dan pilih produk itu.</li>
				<li>Di halaman program, pilih <strong>sumber lead</strong> (LP/iklan) program itu. Lead dari sumber tersebut otomatis masuk program ini; CS juga bisa memindahkan program di halaman lead.</li>
				<li>Setelah pembayaran diverifikasi, tombol <strong>Daftarkan peserta</strong> menggantikan "Jadikan santri": lead closing dan tercatat sebagai pesanan program di Produk CRM.</li>
				<li>Tambahkan <strong>sesi</strong> (tanggal & jam Zoom). Setiap sesi punya link absensi untuk dibagikan di chat Zoom; peserta cukup mengisi nomor WhatsApp pendaftaran.</li>
				<li>Setelah program selesai, gunakan <strong>Tawarkan kelas reguler</strong> untuk membuka lagi lead peserta di Tahsin Reguler.</li>
			</ol>
		</details>

		{#if showForm}
			<form class={panel + " grid gap-3 p-4 sm:grid-cols-2 sm:p-5"} onsubmit={submit}>
				<label class={lbl + " sm:col-span-2"}>Nama program *<input class={field + " mt-1"} bind:value={form.nama} maxlength="80" placeholder="mis. Sepekan Bersama Al-Fatihah" required /></label>
				<label class={lbl}>Produk program (Produk CRM) *
					<select class={field + " mt-1"} bind:value={form.produk_id} onchange={() => (form.produk_batch_id = 0)} required>
						<option value={0}>Pilih produk</option>
						{#each props.produk ?? [] as p (p.id)}<option value={p.id}>{p.nama}{p.harga ? ` · Rp${fmt(p.harga)}` : ""}{p.is_aktif ? "" : " (nonaktif)"}</option>{/each}
					</select>
				</label>
				<label class={lbl}>Batch (opsional)
					<select class={field + " mt-1"} bind:value={form.produk_batch_id} disabled={!produkDipilih || produkDipilih.batches.length === 0}>
						<option value={0}>Semua batch</option>
						{#each produkDipilih?.batches ?? [] as b (b.id)}<option value={b.id}>{b.nama}</option>{/each}
					</select>
				</label>
				{#if (props.produk ?? []).length === 0}
					<p class="text-xs text-amber-700 sm:col-span-2 dark:text-amber-400">Belum ada produk berkategori program. Buat dulu di <a href="/app/produk-crm" use:inertia class="font-semibold underline">Produk CRM</a>.</p>
				{/if}
				<label class={lbl}>Keterangan<input class={field + " mt-1"} bind:value={form.keterangan} maxlength="300" placeholder="mis. 3 sesi Zoom: 22, 25 & 28 Nov" /></label>
				<label class={lbl}>Maks. peserta per kelas<input type="number" min="1" max="500" class={field + " mt-1"} bind:value={form.kapasitas_kelas} />
					<span class="mt-1 block font-normal text-neutral-400">Kelas otomatis dipisah ikhwan/akhwat; kelas baru dibuka saat penuh.</span>
				</label>
				<div class="flex justify-end gap-2 sm:col-span-2">
					<button type="button" class={btnGhost} onclick={() => (showForm = false)}>Batal</button>
					<button type="submit" class={btnPrimary} disabled={busy || form.nama.trim().length < 2 || !Number(form.produk_id)}>Buat program</button>
				</div>
			</form>
		{/if}

		<ul class="space-y-3">
			{#each programs as p (p.id)}
				<li>
					<a href={`/app/leads/program/${p.id}`} use:inertia class={panel + " group flex items-start gap-4 p-4 transition-colors hover:border-brand-400/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 sm:p-5 " + (p.is_aktif ? "" : "opacity-70")}>
						<span class="mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-lg {p.jenis === 'reguler' ? 'bg-brand-400/10 text-brand-700 dark:text-brand-300' : 'bg-sky-500/10 text-sky-700 dark:text-sky-300'}"><CalendarRange size={17} /></span>
						<div class="min-w-0 flex-1">
							<div class="flex flex-wrap items-center gap-2">
								<h2 class="text-base font-semibold text-neutral-950 dark:text-white">{p.nama}</h2>
								<span class="rounded-full px-2 py-0.5 text-[11px] font-medium {p.jenis === 'reguler' ? 'bg-brand-400/10 text-brand-700 dark:text-brand-300' : 'bg-sky-500/10 text-sky-700 dark:text-sky-300'}">{p.jenis === "reguler" ? "Closing jadi santri" : "Program pendek · closing jadi peserta"}</span>
								{#if !p.is_aktif}<span class="rounded-full bg-neutral-200 px-2 py-0.5 text-[11px] font-medium text-neutral-600 dark:bg-neutral-800 dark:text-neutral-400">Nonaktif</span>{/if}
							</div>
							{#if p.keterangan}<p class="mt-0.5 text-xs text-neutral-500 dark:text-neutral-400">{p.keterangan}</p>{/if}
							<p class="mt-1.5 text-xs text-neutral-500 dark:text-neutral-400">
								Sumber: {p.jenis === "reguler" && p.sumber_ids.length === 0 ? "semua sumber lain" : p.sumber_ids.map((id) => sourceName.get(id) ?? `#${id}`).join(", ") || "belum dipetakan"}
							</p>
						</div>
						<dl class="grid shrink-0 grid-cols-2 gap-x-5 gap-y-1 text-right text-xs {p.jenis === 'program' ? 'sm:grid-cols-4' : ''}">
							<div><dt class="text-neutral-500">Lead aktif</dt><dd class="text-base font-semibold tabular-nums text-neutral-950 dark:text-white">{fmt(p.lead_open)}</dd></div>
							<div><dt class="text-neutral-500">Closing</dt><dd class="text-base font-semibold tabular-nums text-neutral-950 dark:text-white">{fmt(p.lead_won)}</dd></div>
							{#if p.jenis === "program"}
								<div><dt class="text-neutral-500">Peserta</dt><dd class="text-base font-semibold tabular-nums text-neutral-950 dark:text-white">{fmt(p.jumlah_peserta)}</dd></div>
								<div><dt class="text-neutral-500">Sesi</dt><dd class="text-base font-semibold tabular-nums text-neutral-950 dark:text-white">{fmt(p.jumlah_sesi)}</dd></div>
							{/if}
						</dl>
						<ChevronRight size={16} class="mt-2 shrink-0 text-neutral-400 group-hover:text-brand-600" />
					</a>
				</li>
			{/each}
		</ul>
	</div>
</AppLayout>
