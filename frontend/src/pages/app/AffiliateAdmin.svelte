<script lang="ts">
	import { inertia, router } from "@inertiajs/svelte";
	import AppLayout from "@layouts/AppLayout.svelte";
	import type { Flash, User } from "@lib/types";
	import { ArrowLeft, Check, Copy, MessageCircle, RefreshCw, Trash2 } from "lucide-svelte";

	interface Row { id: number; kode: string; tipe: string; nama: string; no_wa: string; email: string; status: string; bank_nama: string; bank_rekening: string; bank_atas_nama: string; punya_akun: boolean; klik: number; daftar: number; santri: number; komisi_siap: number; komisi_total: number }
	interface Rule { id: number; scope: string; target_id: number; jenis: string; nilai: number; pola: string; maks_bulan: { Int64: number; Valid: boolean }; is_aktif: number }
	interface Ledger { id: number; affiliate_nama: string; affiliate_kode: string; affiliate_tipe: string; rekening: string; lead_id: number; lead_nama: string; sumber: string; bulan_ke: number; basis: number; jumlah: number; status: string; metode_bayar: string; ujroh_bulan: string; catatan: string; alasan_batal: string; created_at: string }
	interface Opt { id: number; nama: string; aktif: boolean }
	interface Props {
		user?: User; affiliates: Row[]; rules: Rule[]; ledger: Ledger[]; status_filter: string;
		settings: { masa_berlaku_hari: number; masa_tahan_hari: number; link_tujuan_default_id: number; template_bagikan: string };
		programs: Opt[]; links: { slug: string; judul: string }[]; base_url: string; tab: string; can_edit: boolean;
		invite?: { link: string; wa_url: string; email: string; emailed: boolean }; flash?: Flash;
	}
	let props: Props = $props();
	let tab = $state(props.tab || "affiliate");
	let rows = $derived(props.affiliates ?? []);
	let menunggu = $derived(rows.filter((r) => r.status === "menunggu"));
	let aktif = $derived(rows.filter((r) => r.status !== "menunggu"));
	let leaderboard = $derived([...aktif].sort((a, b) => b.santri - a.santri || b.daftar - a.daftar).slice(0, 5));
	let affName = $derived(new Map(rows.map((r) => [r.id, `${r.nama} (${r.kode})`])));
	let progName = $derived(new Map((props.programs ?? []).map((p) => [p.id, p.nama])));

	let ruleForm = $state({ scope: "global", target_id: 0, jenis: "persen", nilai_text: "10", pola: "sekali", maks: "6" });
	let settings = $state({ ...props.settings });
	let pilih = $state<number[]>([]);
	let catatanTransfer = $state("");
	let disalin = $state(false);

	const rupiah = (n: number) => "Rp" + (n ?? 0).toLocaleString("id-ID");
	const ruleLabel = (r: Rule) => (r.jenis === "persen" ? `${r.nilai / 100}%` : rupiah(r.nilai)) + (r.pola === "bulanan" ? ` · bulanan ${r.maks_bulan?.Valid ? r.maks_bulan.Int64 + " bln" : "selama aktif"}` : " · sekali");
	const ruleTarget = (r: Rule) => (r.scope === "global" ? "Semua (default)" : r.scope === "program" ? `Program: ${progName.get(r.target_id) ?? r.target_id}` : `Affiliate: ${affName.get(r.target_id) ?? r.target_id}`);
	const opts = { preserveScroll: true };

	function gantiTab(t: string) {
		tab = t;
		history.replaceState(null, "", `?tab=${t}`);
	}
	function simpanRule(e: Event) {
		e.preventDefault();
		const nilai = ruleForm.jenis === "persen" ? Math.round(parseFloat(ruleForm.nilai_text.replace(",", ".")) * 100) : parseInt(ruleForm.nilai_text.replace(/\D/g, ""), 10);
		const maks = ruleForm.pola !== "bulanan" ? null : ruleForm.maks === "aktif" ? -1 : Number(ruleForm.maks);
		router.post("/app/leads/affiliate/aturan", { scope: ruleForm.scope, target_id: Number(ruleForm.target_id), jenis: ruleForm.jenis, nilai, pola: ruleForm.pola, maks_bulan: maks }, opts);
	}
	function simpanSettings(e: Event) {
		e.preventDefault();
		router.put("/app/leads/affiliate/pengaturan", { ...settings, masa_berlaku_hari: Number(settings.masa_berlaku_hari), masa_tahan_hari: Number(settings.masa_tahan_hari), link_tujuan_default_id: Number(settings.link_tujuan_default_id) }, opts);
	}
	function transfer() {
		if (!pilih.length || !confirm(`Tandai ${pilih.length} komisi sudah ditransfer?`)) return;
		router.post("/app/leads/affiliate/komisi/transfer", { ids: pilih, catatan: catatanTransfer }, { ...opts, onSuccess: () => { pilih = []; catatanTransfer = ""; } });
	}
	function batalkan(id: number) {
		const alasan = prompt("Alasan membatalkan komisi:");
		if (alasan) router.post(`/app/leads/affiliate/komisi/${id}/batal`, { alasan }, opts);
	}
	function ubahKode(r: Row) {
		const kode = prompt(`Kode baru untuk ${r.nama}:`, r.kode);
		if (kode && kode !== r.kode) router.put(`/app/leads/affiliate/${r.id}/kode`, { kode }, opts);
	}
	async function salinLink(l: string) {
		try { await navigator.clipboard.writeText(l); disalin = true; setTimeout(() => (disalin = false), 1500); } catch { /* */ }
	}

	const panel = "rounded-xl border border-neutral-200/80 bg-white dark:border-white/[0.07] dark:bg-neutral-925/60";
	const field = "h-9 w-full rounded-md border border-neutral-200 bg-neutral-50 px-2.5 text-sm text-neutral-900 focus:border-brand-500 focus:outline-none focus:ring-2 focus:ring-brand-500/30 dark:border-white/10 dark:bg-neutral-950 dark:text-white";
	const lbl = "block text-[11px] font-medium text-neutral-500 dark:text-neutral-400";
	const btn = "inline-flex h-9 items-center justify-center gap-1.5 rounded-md px-3 text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 disabled:opacity-50";
	const btnPrimary = btn + " bg-brand-600 text-white hover:bg-brand-700 dark:bg-brand-500";
	const btnGhost = btn + " border border-neutral-200 text-neutral-700 hover:bg-neutral-50 dark:border-white/10 dark:text-neutral-300 dark:hover:bg-white/[0.04]";
	const statusCls: Record<string, string> = {
		pending: "bg-amber-500/10 text-amber-700 dark:text-amber-400", siap: "bg-sky-500/10 text-sky-700 dark:text-sky-400",
		dibayar: "bg-green-500/10 text-green-700 dark:text-green-400", batal: "bg-neutral-500/10 text-neutral-500",
		aktif: "bg-green-500/10 text-green-700 dark:text-green-400", nonaktif: "bg-neutral-500/10 text-neutral-500", ditolak: "bg-red-500/10 text-red-600",
	};
</script>

<svelte:head><title>Affiliate · Ruang Sanad</title></svelte:head>
<AppLayout user={props.user}>
	<div class="mx-auto max-w-6xl space-y-4 px-4 py-6 sm:px-6 lg:py-8">
		<header class="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
			<div>
				<a href="/app/leads?view=list" use:inertia class="inline-flex items-center gap-1 text-xs font-medium text-neutral-500 hover:text-neutral-800 dark:text-neutral-400 dark:hover:text-white"><ArrowLeft size={13} /> Leads</a>
				<h1 class="mt-1 text-[28px] font-semibold leading-tight tracking-[-0.02em] text-neutral-950 dark:text-white">Affiliate</h1>
				<p class="mt-1 text-sm text-neutral-500 dark:text-neutral-400">Form pendaftaran eksternal: <a class="font-mono text-brand-700 hover:underline dark:text-brand-400" href="/affiliate/daftar" target="_blank" rel="noopener">{props.base_url}/affiliate/daftar</a></p>
			</div>
			<button type="button" class={btnGhost} onclick={() => router.post("/app/leads/affiliate/hitung-ulang", {}, opts)}><RefreshCw size={15} /> Hitung ulang komisi</button>
		</header>

		{#if props.flash?.success}<div role="status" class="rounded-lg border border-success/20 bg-success/10 px-4 py-3 text-sm font-medium text-emerald-800 dark:text-emerald-300">{props.flash.success}</div>{/if}
		{#if props.flash?.error}<div role="alert" class="rounded-lg border border-error/20 bg-error/10 px-4 py-3 text-sm font-medium text-red-700 dark:text-red-300">{props.flash.error}</div>{/if}
		{#if props.invite}
			<div class={panel + " space-y-2 p-4"} role="status">
				<p class="text-sm font-medium text-neutral-900 dark:text-white">Link aktivasi untuk {props.invite.email} (berlaku 3 hari){props.invite.emailed ? " — sudah dikirim ke email." : "."}</p>
				<div class="flex flex-wrap gap-2">
					<code class="min-w-0 flex-1 truncate rounded bg-neutral-100 px-2 py-1.5 text-xs dark:bg-neutral-900">{props.invite.link}</code>
					<button type="button" class={btnGhost} onclick={() => salinLink(props.invite!.link)}>{#if disalin}<Check size={15} />{:else}<Copy size={15} />{/if} Salin</button>
					{#if props.invite.wa_url}<a class={btnPrimary} href={props.invite.wa_url} target="_blank" rel="noopener"><MessageCircle size={15} /> Kirim via WhatsApp</a>{/if}
				</div>
			</div>
		{/if}

		<div class="flex gap-1 overflow-x-auto border-b border-neutral-200 dark:border-white/10" role="tablist">
			{#each [["affiliate", `Affiliate${menunggu.length ? ` (${menunggu.length} menunggu)` : ""}`], ["aturan", "Aturan komisi"], ["komisi", "Buku komisi"], ["pengaturan", "Pengaturan"]] as [k, t]}
				<button type="button" role="tab" aria-selected={tab === k} onclick={() => gantiTab(k)}
					class="-mb-px whitespace-nowrap border-b-2 px-3 py-2 text-sm font-medium {tab === k ? 'border-brand-600 text-brand-700 dark:text-brand-400' : 'border-transparent text-neutral-500 hover:text-neutral-800 dark:hover:text-white'}">{t}</button>
			{/each}
		</div>

		{#if tab === "affiliate"}
			{#if menunggu.length}
				<section class={panel}>
					<h2 class="border-b border-neutral-100 px-4 py-2.5 text-sm font-semibold dark:border-white/[0.06]">Menunggu persetujuan</h2>
					<ul class="divide-y divide-neutral-100 dark:divide-white/[0.05]">
						{#each menunggu as r (r.id)}
							<li class="flex flex-wrap items-center justify-between gap-3 px-4 py-3 text-sm">
								<div><p class="font-medium text-neutral-900 dark:text-white">{r.nama}</p><p class="text-xs text-neutral-500">{r.no_wa} · {r.email}{r.bank_rekening ? ` · ${r.bank_nama} ${r.bank_rekening}` : ""}</p></div>
								{#if props.can_edit}
									<div class="flex gap-2">
										<button type="button" class={btnGhost} onclick={() => router.put(`/app/leads/affiliate/${r.id}/status`, { status: "ditolak" }, opts)}>Tolak</button>
										<button type="button" class={btnPrimary} onclick={() => router.post(`/app/leads/affiliate/${r.id}/setujui`, {}, opts)}>Setujui</button>
									</div>
								{/if}
							</li>
						{/each}
					</ul>
				</section>
			{/if}
			{#if leaderboard.some((r) => r.daftar > 0)}
				<section class={panel + " p-4"}>
					<h2 class="text-sm font-semibold text-neutral-900 dark:text-white">Teratas</h2>
					<ol class="mt-2 grid gap-2 sm:grid-cols-5">
						{#each leaderboard as r, i (r.id)}
							<li class="rounded-lg bg-neutral-50 p-3 text-sm dark:bg-neutral-900/60"><span class="text-xs text-neutral-400">#{i + 1}</span><p class="truncate font-medium text-neutral-900 dark:text-white">{r.nama}</p><p class="text-xs text-neutral-500">{r.santri} bayar · {r.daftar} daftar</p></li>
						{/each}
					</ol>
				</section>
			{/if}
			<div class={panel + " overflow-x-auto"}>
				<table class="w-full min-w-[760px] text-sm">
					<thead><tr class="border-b border-neutral-200/80 text-left text-[11px] font-medium uppercase tracking-wide text-neutral-500 dark:border-white/[0.07]">
						<th class="px-4 py-2.5">Affiliate</th><th class="px-3 py-2.5">Status</th><th class="px-3 py-2.5 text-right">Klik 30h</th><th class="px-3 py-2.5 text-right">Daftar</th><th class="px-3 py-2.5 text-right">Bayar</th><th class="px-3 py-2.5 text-right">Siap cair</th><th class="px-3 py-2.5 text-right">Total komisi</th><th class="px-3 py-2.5"><span class="sr-only">Aksi</span></th>
					</tr></thead>
					<tbody class="divide-y divide-neutral-100 dark:divide-white/[0.05]">
						{#each aktif as r (r.id)}
							<tr>
								<td class="px-4 py-3"><p class="font-medium text-neutral-900 dark:text-white">{r.nama}</p><p class="font-mono text-xs text-neutral-500">{r.kode} · {r.tipe}</p></td>
								<td class="px-3 py-3"><span class="rounded-full px-2 py-0.5 text-xs font-medium {statusCls[r.status] ?? ''}">{r.status}</span></td>
								<td class="px-3 py-3 text-right tabular-nums">{r.klik}</td>
								<td class="px-3 py-3 text-right tabular-nums">{r.daftar}</td>
								<td class="px-3 py-3 text-right tabular-nums">{r.santri}</td>
								<td class="px-3 py-3 text-right tabular-nums">{rupiah(r.komisi_siap)}</td>
								<td class="px-3 py-3 text-right tabular-nums font-medium">{rupiah(r.komisi_total)}</td>
								<td class="px-3 py-3">
									{#if props.can_edit}
										<div class="flex justify-end gap-1 text-xs">
											<button type="button" class="rounded px-2 py-1 text-neutral-600 hover:bg-neutral-100 dark:text-neutral-300 dark:hover:bg-white/[0.06]" onclick={() => ubahKode(r)}>Kode</button>
											{#if r.tipe === "eksternal" && r.status === "aktif"}<button type="button" class="rounded px-2 py-1 text-neutral-600 hover:bg-neutral-100 dark:text-neutral-300 dark:hover:bg-white/[0.06]" onclick={() => router.post(`/app/leads/affiliate/${r.id}/undangan`, {}, opts)}>Link aktivasi</button>{/if}
											<button type="button" class="rounded px-2 py-1 text-neutral-600 hover:bg-neutral-100 dark:text-neutral-300 dark:hover:bg-white/[0.06]" onclick={() => router.put(`/app/leads/affiliate/${r.id}/status`, { status: r.status === "aktif" ? "nonaktif" : "aktif" }, opts)}>{r.status === "aktif" ? "Nonaktifkan" : "Aktifkan"}</button>
										</div>
									{/if}
								</td>
							</tr>
						{:else}
							<tr><td colspan="8" class="px-4 py-10 text-center text-sm text-neutral-500">Belum ada affiliate. Guru dengan akun login otomatis terdaftar; eksternal mendaftar lewat form di atas.</td></tr>
						{/each}
					</tbody>
				</table>
			</div>
		{:else if tab === "aturan"}
			<div class="grid gap-4 lg:grid-cols-[minmax(0,1fr)_340px]">
				<div class={panel}>
					<p class="border-b border-neutral-100 px-4 py-2.5 text-xs text-neutral-500 dark:border-white/[0.06]">Urutan berlaku: aturan affiliate → aturan program → default. Tanpa aturan = tanpa komisi. Perubahan tidak berlaku mundur.</p>
					<ul class="divide-y divide-neutral-100 dark:divide-white/[0.05]">
						{#each props.rules ?? [] as r (r.id)}
							<li class="flex items-center justify-between gap-3 px-4 py-3 text-sm {r.is_aktif ? '' : 'opacity-50'}">
								<div><p class="font-medium text-neutral-900 dark:text-white">{ruleTarget(r)}</p><p class="text-xs text-neutral-500">{ruleLabel(r)}</p></div>
								{#if props.can_edit && r.scope !== "global"}<button type="button" aria-label="Hapus aturan" class="text-neutral-400 hover:text-red-600" onclick={() => confirm("Hapus aturan ini?") && router.delete(`/app/leads/affiliate/aturan/${r.id}`, opts)}><Trash2 size={15} /></button>{/if}
							</li>
						{:else}
							<li class="px-4 py-10 text-center text-sm text-amber-700 dark:text-amber-400">Belum ada aturan. Buat aturan default agar affiliate mendapat komisi.</li>
						{/each}
					</ul>
				</div>
				{#if props.can_edit}
					<form class={panel + " space-y-3 p-4"} onsubmit={simpanRule}>
						<h2 class="text-sm font-semibold text-neutral-900 dark:text-white">Tambah / ubah aturan</h2>
						<label class={lbl}>Berlaku untuk
							<select class={field + " mt-1"} bind:value={ruleForm.scope} onchange={() => (ruleForm.target_id = 0)}>
								<option value="global">Semua (default)</option><option value="program">Program tertentu</option><option value="affiliate">Affiliate tertentu</option>
							</select>
						</label>
						{#if ruleForm.scope === "program"}
							<select class={field} bind:value={ruleForm.target_id} aria-label="Program" required>{#each props.programs as p (p.id)}<option value={p.id}>{p.nama}</option>{/each}</select>
						{:else if ruleForm.scope === "affiliate"}
							<select class={field} bind:value={ruleForm.target_id} aria-label="Affiliate" required>{#each aktif as r (r.id)}<option value={r.id}>{r.nama} ({r.kode})</option>{/each}</select>
						{/if}
						<div class="grid grid-cols-2 gap-2">
							<label class={lbl}>Jenis<select class={field + " mt-1"} bind:value={ruleForm.jenis}><option value="persen">Persen</option><option value="flat">Nominal tetap</option></select></label>
							<label class={lbl}>{ruleForm.jenis === "persen" ? "Persen (%)" : "Nominal (Rp)"}<input class={field + " mt-1"} bind:value={ruleForm.nilai_text} inputmode="decimal" required /></label>
						</div>
						<div class="grid grid-cols-2 gap-2">
							<label class={lbl}>Diberikan<select class={field + " mt-1"} bind:value={ruleForm.pola}><option value="sekali">Sekali (pendaftaran)</option><option value="bulanan">Tiap bulan (SPP)</option></select></label>
							{#if ruleForm.pola === "bulanan"}
								<label class={lbl}>Selama<select class={field + " mt-1"} bind:value={ruleForm.maks}>{#each ["3", "6", "12"] as m}<option value={m}>{m} bulan</option>{/each}<option value="aktif">Selama aktif</option></select></label>
							{/if}
						</div>
						<button type="submit" class={btnPrimary + " w-full"}>Simpan aturan</button>
					</form>
				{/if}
			</div>
		{:else if tab === "komisi"}
			<div class="flex flex-wrap items-center gap-2">
				{#each [["", "Semua"], ["pending", "Tertahan"], ["siap", "Siap cair"], ["dibayar", "Dibayar"], ["batal", "Batal"]] as [s, t]}
					<a href={`/app/leads/affiliate?tab=komisi${s ? `&status=${s}` : ""}`} use:inertia class="rounded-full px-3 py-1 text-xs font-medium {props.status_filter === s ? 'bg-neutral-900 text-white dark:bg-white dark:text-neutral-900' : 'bg-neutral-100 text-neutral-600 dark:bg-neutral-800 dark:text-neutral-300'}">{t}</a>
				{/each}
			</div>
			{#if pilih.length}
				<div class={panel + " flex flex-wrap items-center gap-2 p-3"}>
					<span class="text-sm">{pilih.length} dipilih</span>
					<input class={field + " max-w-xs"} bind:value={catatanTransfer} placeholder="Catatan / no. referensi transfer" aria-label="Catatan transfer" />
					<button type="button" class={btnPrimary} onclick={transfer}>Tandai sudah ditransfer</button>
				</div>
			{/if}
			<div class={panel + " overflow-x-auto"}>
				<table class="w-full min-w-[820px] text-sm">
					<thead><tr class="border-b border-neutral-200/80 text-left text-[11px] font-medium uppercase tracking-wide text-neutral-500 dark:border-white/[0.07]">
						<th class="w-8 px-3 py-2.5"><span class="sr-only">Pilih</span></th><th class="px-3 py-2.5">Affiliate</th><th class="px-3 py-2.5">Rujukan</th><th class="px-3 py-2.5">Sumber</th><th class="px-3 py-2.5 text-right">Jumlah</th><th class="px-3 py-2.5">Status</th><th class="px-3 py-2.5"><span class="sr-only">Aksi</span></th>
					</tr></thead>
					<tbody class="divide-y divide-neutral-100 dark:divide-white/[0.05]">
						{#each props.ledger ?? [] as c (c.id)}
							<tr>
								<td class="px-3 py-3">{#if c.status === "siap" && c.affiliate_tipe === "eksternal"}<input type="checkbox" aria-label="Pilih komisi {c.id}" bind:group={pilih} value={c.id} />{/if}</td>
								<td class="px-3 py-3"><p class="font-medium text-neutral-900 dark:text-white">{c.affiliate_nama}</p><p class="text-xs text-neutral-500">{c.affiliate_tipe === "guru" ? "Guru · via ujroh" : c.rekening || "Rekening belum diisi"}</p></td>
								<td class="px-3 py-3"><a href={`/app/leads/${c.lead_id}`} use:inertia class="hover:underline">{c.lead_nama || `L-${c.lead_id}`}</a></td>
								<td class="px-3 py-3 text-neutral-600 dark:text-neutral-400">{c.sumber === "spp" ? `SPP bulan ${c.bulan_ke}` : "Pendaftaran"} · {rupiah(c.basis)}</td>
								<td class="px-3 py-3 text-right font-medium tabular-nums">{rupiah(c.jumlah)}</td>
								<td class="px-3 py-3"><span class="rounded-full px-2 py-0.5 text-xs font-medium {statusCls[c.status] ?? ''}">{c.status === "pending" ? "tertahan" : c.status}</span>
									{#if c.ujroh_bulan}<span class="ml-1 text-xs text-neutral-500">ujroh {c.ujroh_bulan}</span>{/if}
									{#if c.alasan_batal}<p class="text-xs text-neutral-500">{c.alasan_batal}</p>{/if}</td>
								<td class="px-3 py-3 text-right">{#if props.can_edit && (c.status === "pending" || c.status === "siap")}<button type="button" class="text-xs text-red-600 hover:underline" onclick={() => batalkan(c.id)}>Batalkan</button>{/if}</td>
							</tr>
						{:else}
							<tr><td colspan="7" class="px-4 py-10 text-center text-sm text-neutral-500">Belum ada komisi. Komisi muncul setelah pembayaran rujukan terverifikasi (dihitung tiap 10 menit).</td></tr>
						{/each}
					</tbody>
				</table>
			</div>
		{:else}
			<form class={panel + " grid max-w-2xl gap-4 p-5"} onsubmit={simpanSettings}>
				<label class={lbl}>Masa berlaku link (cookie)
					<select class={field + " mt-1"} bind:value={settings.masa_berlaku_hari} disabled={!props.can_edit}>{#each [30, 60, 90] as d}<option value={d}>{d} hari</option>{/each}</select>
				</label>
				<label class={lbl}>Masa tahan komisi (hari)<input class={field + " mt-1"} type="number" min="0" max="90" bind:value={settings.masa_tahan_hari} disabled={!props.can_edit} /></label>
				<label class={lbl}>Tujuan default link affiliate
					<select class={field + " mt-1"} bind:value={settings.link_tujuan_default_id} disabled={!props.can_edit}><option value={0}>— Pilih link pendaftaran —</option>{#each props.links as l (l.slug)}<option value={Number(l.slug)}>{l.judul}</option>{/each}</select>
				</label>
				<label class={lbl}>Template pesan "Bagikan ke WhatsApp"
					<textarea class={field + " mt-1 h-auto py-2"} rows="3" bind:value={settings.template_bagikan} maxlength="500" disabled={!props.can_edit} placeholder={"Assalamu'alaikum, yuk belajar Al-Qur'an bersama Ruang Sanad. Daftar lewat link ini: {link}"}></textarea>
					<span class="mt-1 block font-normal text-neutral-400">Gunakan {"{link}"} untuk link affiliate.</span>
				</label>
				{#if props.can_edit}<div class="flex justify-end"><button type="submit" class={btnPrimary}>Simpan pengaturan</button></div>{/if}
			</form>
		{/if}
	</div>
</AppLayout>
