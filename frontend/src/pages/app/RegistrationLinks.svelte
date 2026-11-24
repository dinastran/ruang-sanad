<script lang="ts">
	import { inertia, router } from "@inertiajs/svelte";
	import AppLayout from "@layouts/AppLayout.svelte";
	import type { Flash, User } from "@lib/types";
	import LinkForm, { type LinkRow, type MetaPixelInfo, type Option } from "@components/registration/LinkForm.svelte";
	import { ArrowLeft, Check, Copy, ExternalLink, Link2, Pencil, Plus, Settings } from "lucide-svelte";

	interface Rekening { bank: string; no_rekening: string; atas_nama: string }
	interface Settings { cs_wa_number: string; payment_expiry_hours: number; bank_manual_info: string; rekening?: Rekening[] }
	interface Props { user?: User; links: LinkRow[]; programs: Option[]; sources: Option[]; settings: Settings; base_url: string; meta_pixel?: MetaPixelInfo; flash?: Flash }
	let props: Props = $props();
	let links = $derived(props.links ?? []);

	let editing = $state<LinkRow | null | "baru">(null);
	let showSettings = $state(!props.settings?.cs_wa_number);
	// svelte-ignore state_referenced_locally
	let settings = $state({ ...props.settings, rekening: (props.settings?.rekening ?? []).map((r) => ({ ...r })) });
	if (settings.rekening.length === 0) settings.rekening.push({ bank: "", no_rekening: "", atas_nama: "" });
	let menyimpan = $state(false);
	let disalin = $state(0);

	const url = (l: LinkRow) => `${props.base_url}/daftar/${l.slug}`;
	const pct = (a = 0, b = 0) => (b > 0 ? Math.round((a / b) * 100) + "%" : "–");
	const modeLabel = { wa: "WA langsung", short: "Short form", full: "Full form" } as const;

	async function salin(l: LinkRow) {
		try {
			await navigator.clipboard.writeText(url(l));
			disalin = l.id;
			setTimeout(() => (disalin = 0), 1500);
		} catch {
			/* clipboard tidak tersedia */
		}
	}
	function simpanSettings(e: Event) {
		e.preventDefault();
		menyimpan = true;
		router.put("/app/leads/link-pendaftaran/pengaturan", { ...settings, payment_expiry_hours: Number(settings.payment_expiry_hours) }, {
			preserveScroll: true, onSuccess: () => (showSettings = false), onFinish: () => (menyimpan = false),
		});
	}

	const panel = "rounded-xl border border-neutral-200/80 bg-white dark:border-white/[0.07] dark:bg-neutral-925/60";
	const field = "h-9 w-full rounded-md border border-neutral-200 bg-neutral-50 px-2.5 text-sm text-neutral-900 focus:border-brand-500 focus:outline-none focus:ring-2 focus:ring-brand-500/30 dark:border-white/10 dark:bg-neutral-950 dark:text-white";
	const lbl = "block text-[11px] font-medium text-neutral-500 dark:text-neutral-400";
	const btn = "inline-flex h-9 items-center justify-center gap-1.5 rounded-md px-3 text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 disabled:opacity-50";
	const btnPrimary = btn + " bg-brand-600 text-white hover:bg-brand-700 dark:bg-brand-500";
	const btnGhost = btn + " border border-neutral-200 text-neutral-700 hover:bg-neutral-50 dark:border-white/10 dark:text-neutral-300 dark:hover:bg-white/[0.04]";
	const iconBtn = "inline-flex h-8 w-8 items-center justify-center rounded-md text-neutral-500 hover:bg-neutral-100 hover:text-neutral-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 dark:hover:bg-white/[0.06] dark:hover:text-white";
</script>

<svelte:head><title>Link Pendaftaran · Ruang Sanad</title></svelte:head>
<AppLayout user={props.user}>
	<div class="mx-auto max-w-6xl space-y-4 px-4 py-6 sm:px-6 lg:py-8">
		<header class="flex flex-col gap-3 pb-1 sm:flex-row sm:items-end sm:justify-between">
			<div class="min-w-0">
				<a href="/app/leads?view=list" use:inertia class="inline-flex items-center gap-1 text-xs font-medium text-neutral-500 hover:text-neutral-800 dark:text-neutral-400 dark:hover:text-white"><ArrowLeft size={13} /> Leads</a>
				<h1 class="mt-1 text-[28px] font-semibold leading-tight tracking-[-0.02em] text-neutral-950 dark:text-white">Link Pendaftaran</h1>
				<p class="mt-1 max-w-2xl text-sm text-neutral-500 dark:text-neutral-400">Bagikan link ini di bio, iklan, atau grup. Pendaftar otomatis masuk CRM tanpa input manual.</p>
			</div>
			<div class="flex shrink-0 gap-2">
				<button type="button" class={btnGhost} onclick={() => (showSettings = !showSettings)}><Settings size={15} /> Pengaturan</button>
				{#if editing === null}<button type="button" class={btnPrimary} onclick={() => (editing = "baru")}><Plus size={15} /> Link baru</button>{/if}
			</div>
		</header>

		{#if props.flash?.success}<div role="status" class="rounded-lg border border-success/20 bg-success/10 px-4 py-3 text-sm font-medium text-emerald-800 dark:text-emerald-300">{props.flash.success}</div>{/if}
		{#if props.flash?.error}<div role="alert" class="rounded-lg border border-error/20 bg-error/10 px-4 py-3 text-sm font-medium text-red-700 dark:text-red-300">{props.flash.error}</div>{/if}

		{#if showSettings}
			<form class={panel + " grid gap-3 p-4 sm:grid-cols-2 sm:p-5"} onsubmit={simpanSettings}>
				<label class={lbl}>Nomor WhatsApp CS *<input class={field + " mt-1"} type="tel" bind:value={settings.cs_wa_number} required placeholder="0812xxxxxxxx" /></label>
				<label class={lbl}>Masa berlaku tagihan (jam)<input class={field + " mt-1"} type="number" min="1" max="72" bind:value={settings.payment_expiry_hours} /></label>
				<fieldset class="sm:col-span-2">
					<legend class={lbl}>Rekening transfer manual</legend>
					<p class="mt-0.5 text-xs text-neutral-500">Tampil sebagai kartu di halaman bayar; tombol Salin hanya menyalin nomor rekening. Maks. 5 rekening.</p>
					<div class="mt-2 space-y-2">
						{#each settings.rekening as r, i (i)}
							<div class="grid gap-2 rounded-lg border border-neutral-200 p-2.5 sm:grid-cols-[1fr_1fr_1.3fr_auto] dark:border-white/10">
								<input class={field} bind:value={r.bank} maxlength="40" placeholder="Nama bank, mis. Bank Muamalat" aria-label={`Nama bank ${i + 1}`} />
								<input class={field + " font-mono"} bind:value={r.no_rekening} maxlength="40" inputmode="numeric" placeholder="No. rekening" aria-label={`Nomor rekening ${i + 1}`} />
								<input class={field} bind:value={r.atas_nama} maxlength="80" placeholder="Atas nama" aria-label={`Atas nama ${i + 1}`} />
								<button type="button" class={btnGhost + " px-2.5"} aria-label={`Hapus rekening ${i + 1}`} onclick={() => (settings.rekening = settings.rekening.filter((_, j) => j !== i))}>Hapus</button>
							</div>
						{/each}
					</div>
					{#if settings.rekening.length < 5}
						<button type="button" class={btnGhost + " mt-2"} onclick={() => settings.rekening.push({ bank: "", no_rekening: "", atas_nama: "" })}><Plus size={14} /> Tambah rekening</button>
					{/if}
				</fieldset>
				<label class={lbl + " sm:col-span-2"}>Catatan transfer (opsional)<textarea class={field + " mt-1 h-auto py-2"} rows="2" bind:value={settings.bank_manual_info} placeholder="mis. Konfirmasi otomatis setelah bukti diunggah"></textarea>
					<span class="mt-1 block font-normal text-neutral-400">Ditampilkan kecil di bawah kartu rekening. Bila tidak ada rekening terisi, teks ini yang ditampilkan.</span>
				</label>
				<div class="flex justify-end sm:col-span-2"><button type="submit" class={btnPrimary} disabled={menyimpan}>Simpan pengaturan</button></div>
			</form>
		{/if}

		{#if editing !== null}
			<LinkForm programs={props.programs ?? []} sources={props.sources ?? []} initial={editing === "baru" ? null : editing} metaPixel={props.meta_pixel} onclose={() => (editing = null)} />
		{/if}

		{#if links.length === 0 && editing === null}
			<div class={panel + " flex flex-col items-center gap-3 px-6 py-14 text-center"}>
				<Link2 class="h-8 w-8 text-neutral-400" />
				<p class="font-medium text-neutral-900 dark:text-white">Belum ada link pendaftaran</p>
				<p class="max-w-sm text-sm text-neutral-500">Tautkan program ke produk berharga dulu (menu Program), lalu buat link pertama Anda.</p>
				<button type="button" class={btnPrimary} onclick={() => (editing = "baru")}><Plus size={15} /> Link baru</button>
			</div>
		{:else if links.length}
			<div class={panel + " overflow-x-auto"}>
				<table class="w-full min-w-[760px] text-sm">
					<thead>
						<tr class="border-b border-neutral-200/80 text-left text-[11px] font-medium uppercase tracking-wide text-neutral-500 dark:border-white/[0.07]">
							<th class="px-4 py-2.5">Link</th><th class="px-3 py-2.5">Mode</th><th class="px-3 py-2.5">Harga</th>
							<th class="px-3 py-2.5 text-right">Klik</th><th class="px-3 py-2.5 text-right">Daftar</th><th class="px-3 py-2.5 text-right">Bayar</th>
							<th class="px-3 py-2.5 text-right">Konversi</th><th class="px-3 py-2.5"><span class="sr-only">Aksi</span></th>
						</tr>
					</thead>
					<tbody class="divide-y divide-neutral-100 dark:divide-white/[0.05]">
						{#each links as l (l.id)}
							<tr class={l.is_aktif ? "" : "opacity-60"}>
								<td class="px-4 py-3">
									<p class="font-medium text-neutral-900 dark:text-white">{l.judul}{#if !l.is_aktif}<span class="ml-2 rounded bg-neutral-100 px-1.5 py-0.5 text-[11px] font-normal text-neutral-500 dark:bg-neutral-800">nonaktif</span>{/if}</p>
									<p class="mt-0.5 font-mono text-xs text-neutral-500">/daftar/{l.slug}</p>
									<p class="mt-0.5 text-xs text-neutral-500">{l.program_nama} · {l.sumber_nama}</p>
								</td>
								<td class="px-3 py-3 text-neutral-700 dark:text-neutral-300">{modeLabel[l.mode]}{l.mode !== "wa" && l.after_submit === "bayar" ? " → bayar" : ""}</td>
								<td class="px-3 py-3 tabular-nums text-neutral-700 dark:text-neutral-300">{l.harga_label}</td>
								<td class="px-3 py-3 text-right tabular-nums">{l.klik ?? 0}</td>
								<td class="px-3 py-3 text-right tabular-nums">{l.submit ?? 0}</td>
								<td class="px-3 py-3 text-right tabular-nums">{l.bayar ?? 0}</td>
								<td class="px-3 py-3 text-right tabular-nums font-medium text-neutral-900 dark:text-white">{pct(l.submit, l.klik)}</td>
								<td class="px-3 py-3">
									<div class="flex justify-end gap-0.5">
										<button type="button" class={iconBtn} onclick={() => salin(l)} aria-label="Salin link {l.judul}" title="Salin link">{#if disalin === l.id}<Check size={15} />{:else}<Copy size={15} />{/if}</button>
										<a class={iconBtn} href={url(l)} target="_blank" rel="noopener" aria-label="Buka link {l.judul}" title="Buka"><ExternalLink size={15} /></a>
										<button type="button" class={iconBtn} onclick={() => (editing = l)} aria-label="Ubah link {l.judul}" title="Ubah"><Pencil size={15} /></button>
									</div>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
				<p class="border-t border-neutral-100 px-4 py-2 text-xs text-neutral-500 dark:border-white/[0.05]">Angka 30 hari terakhir. Konversi = daftar ÷ klik.</p>
			</div>
		{/if}
	</div>
</AppLayout>
