<script lang="ts">
	import { router } from "@inertiajs/svelte";
	import AppLayout from "@layouts/AppLayout.svelte";
	import type { User, UjrohRekap, UjrohTarif, UjrohTarifGuru, UjrohGuru, UjrohPertemuan } from "@lib/types";
	import { ChevronDown, CircleAlert, CircleCheck, Download, Lock, LockOpen, Settings2, TriangleAlert } from "lucide-svelte";

	interface Props {
		user?: User;
		rekap?: UjrohRekap;
		tarif?: UjrohTarif;
		tarif_guru?: UjrohTarifGuru[];
		flash?: { success?: string; error?: string };
	}

	let props: Props = $props();
	const kosong: UjrohRekap = {
		bulan: "", terkunci: false, dikunci_at: "", dikunci_oleh: "", dibuka_at: "", dibuka_oleh: "",
		guru: [], ringkasan: { total_ujroh: 0, total_dibayar: 0, total_belum: 0, jumlah_guru: 0, jumlah_pertemuan: 0 },
		tanpa_guru: [], berlangsung: 0, guru_baru_selisih: [],
	};
	let rekap = $derived(props.rekap ?? kosong);
	let tarif = $derived(props.tarif ?? { tetap: 0, part_time: 0 });
	let tarifGuru = $derived(props.tarif_guru ?? []);
	let role = $derived(props.user?.role ?? "");
	let canEdit = $derived(role === "keuangan" || role === "super_admin");
	let isSuperAdmin = $derived(role === "super_admin");
	let bulanIni = $derived(new Date().toLocaleDateString("en-CA", { timeZone: "Asia/Jakarta" }).slice(0, 7));
	let hariIni = $derived(new Date().toLocaleDateString("en-CA", { timeZone: "Asia/Jakarta" }));
	let adaSelisih = $derived(rekap.guru.some((g) => g.selisih) || rekap.guru_baru_selisih.length > 0);

	let expanded = $state<number | null>(null);
	let payingGuru = $state<number | null>(null);
	let payTanggal = $state("");
	let payCatatan = $state("");
	let showTarif = $state(false);
	let tarifForm = $state({ tetap: 0, part_time: 0 });
	let overrideDraft = $state<Record<number, string | number | null>>({});
	let busy = $state(false);

	const rupiah = (n: number) => new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 0 }).format(n);
	const statusLabel = (s: string) => (s === "tetap" ? "Tetap" : s === "part_time" ? "Part Time" : s || "-");
	const namaBulan = (b: string) => {
		if (!b) return "";
		const [y, m] = b.split("-").map(Number);
		return new Date(y, m - 1, 1).toLocaleDateString("id-ID", { month: "long", year: "numeric" });
	};
	const tanggalPendek = (t: string) => (t ? new Date(t + "T00:00:00").toLocaleDateString("id-ID", { day: "numeric", month: "short" }) : "");

	function perKelas(items: UjrohPertemuan[]) {
		const groups = new Map<number, { nama: string; items: UjrohPertemuan[] }>();
		for (const p of items) {
			const g = groups.get(p.kelas_id) ?? { nama: p.kelas_nama, items: [] };
			g.items.push(p);
			groups.set(p.kelas_id, g);
		}
		return Array.from(groups.entries()).map(([id, g]) => ({ id, ...g }));
	}

	const opts = { preserveScroll: true, preserveState: true, onStart: () => (busy = true), onFinish: () => (busy = false) };
	const q = (bulan: string) => `?bulan=${encodeURIComponent(bulan)}`;

	function gantiBulan(bulan: string) {
		if (!bulan) return;
		expanded = null;
		payingGuru = null;
		router.get("/app/keuangan/ujroh", { bulan }, { preserveState: false });
	}

	function kunci() {
		const peringatan: string[] = [];
		if (rekap.bulan === bulanIni) peringatan.push("Bulan ini belum selesai; pertemuan setelah dikunci tidak akan ikut terhitung.");
		if (rekap.berlangsung > 0) peringatan.push(`${rekap.berlangsung} pertemuan masih berlangsung dan tidak ikut terhitung.`);
		if (rekap.tanpa_guru.length > 0) peringatan.push(`${rekap.tanpa_guru.length} pertemuan tanpa guru tidak akan dibayar.`);
		const pesan = [`Kunci ujroh ${namaBulan(rekap.bulan)}? Angka akan dibekukan.`, ...peringatan].join("\n\n");
		if (!confirm(pesan)) return;
		router.post(`/app/keuangan/ujroh/${rekap.bulan}/kunci`, {}, opts);
	}

	function buka() {
		if (!confirm(`Buka kunci ujroh ${namaBulan(rekap.bulan)}? Angka akan dihitung ulang dari data terbaru.`)) return;
		router.post(`/app/keuangan/ujroh/${rekap.bulan}/buka`, {}, opts);
	}

	function mulaiBayar(g: UjrohGuru) {
		payingGuru = g.guru_id;
		payTanggal = hariIni;
		payCatatan = "";
	}

	function simpanBayar(g: UjrohGuru) {
		router.put(`/app/keuangan/ujroh/${rekap.bulan}/guru/${g.guru_id}/bayar`, { tanggal: payTanggal, catatan: payCatatan }, {
			...opts,
			onSuccess: () => (payingGuru = null),
		});
	}

	function batalBayar(g: UjrohGuru) {
		if (!confirm(`Batalkan tanda dibayar untuk ${g.guru_nama}?`)) return;
		router.delete(`/app/keuangan/ujroh/${rekap.bulan}/guru/${g.guru_id}/bayar`, opts);
	}

	function bukaTarif() {
		tarifForm = { tetap: tarif.tetap, part_time: tarif.part_time };
		overrideDraft = Object.fromEntries(tarifGuru.map((g) => [g.guru_id, g.tarif_khusus === null ? "" : String(g.tarif_khusus)]));
		showTarif = !showTarif;
	}

	function simpanTarif() {
		router.put(`/app/keuangan/ujroh/tarif${q(rekap.bulan)}`, { tetap: Number(tarifForm.tetap), part_time: Number(tarifForm.part_time) }, opts);
	}

	function simpanTarifGuru(g: UjrohTarifGuru) {
		const v = overrideDraft[g.guru_id];
		const raw = v === null || v === undefined ? "" : String(v).trim();
		const nominal = raw === "" ? null : Number(raw);
		if (nominal !== null && (!Number.isFinite(nominal) || nominal < 0)) return;
		router.put(`/app/keuangan/ujroh/tarif-guru/${g.guru_id}${q(rekap.bulan)}`, { nominal }, opts);
	}

	const inputClass = "w-full rounded-xl border border-neutral-300 bg-neutral-50 px-3 py-2 text-sm text-neutral-900 outline-none focus:border-brand-400 focus:ring-2 focus:ring-brand-400/15 dark:border-neutral-700 dark:bg-neutral-800/60 dark:text-white";
</script>

<AppLayout user={props.user}>
	<main class="max-w-7xl mx-auto px-4 sm:px-6 py-8 space-y-6">
		<section class="flex flex-col sm:flex-row sm:items-end justify-between gap-4">
			<div>
				<p class="text-xs font-bold tracking-[0.16em] uppercase text-brand-600 dark:text-brand-400">Keuangan</p>
				<h1 class="mt-2 text-3xl font-bold text-neutral-900 dark:text-white">Ujroh Guru</h1>
				<p class="mt-2 text-neutral-600 dark:text-neutral-400">Ujroh per guru dari pertemuan selesai yang diajar, termasuk badal.</p>
			</div>
			<div class="flex flex-wrap items-end gap-2">
				<label class="text-xs font-medium text-neutral-600 dark:text-neutral-400">
					Bulan
					<input type="month" value={rekap.bulan} max={bulanIni} onchange={(e) => gantiBulan(e.currentTarget.value)} class="{inputClass} mt-1 w-44" />
				</label>
				<a href={`/app/keuangan/ujroh/export${q(rekap.bulan)}`} class="inline-flex items-center gap-2 rounded-xl border border-neutral-300 px-3.5 py-2 text-sm font-semibold text-neutral-700 hover:bg-neutral-100 dark:border-neutral-700 dark:text-neutral-200 dark:hover:bg-neutral-800">
					<Download size="16" /> Ekspor Excel
				</a>
			</div>
		</section>

		{#if props.flash?.success}
			<div role="status" class="rounded-2xl border border-success/20 bg-success/10 p-4 text-sm font-medium text-success">{props.flash.success}</div>
		{/if}
		{#if props.flash?.error}
			<div role="alert" class="rounded-2xl border border-error/20 bg-error/10 p-4 text-sm font-medium text-error">{props.flash.error}</div>
		{/if}

		<section class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 rounded-2xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-925 p-4">
			<div class="flex items-start gap-3">
				<div class="p-2 rounded-xl {rekap.terkunci ? 'bg-success/10 text-success' : 'bg-warning/10 text-warning'}">
					{#if rekap.terkunci}<Lock size="18" />{:else}<LockOpen size="18" />{/if}
				</div>
				<div>
					<p class="font-semibold text-neutral-900 dark:text-white">{namaBulan(rekap.bulan)} · {rekap.terkunci ? "Dikunci" : "Belum dikunci"}</p>
					<p class="text-sm text-neutral-600 dark:text-neutral-400">
						{#if rekap.terkunci}
							Dikunci {rekap.dikunci_at}{rekap.dikunci_oleh ? ` oleh ${rekap.dikunci_oleh}` : ""}. Angka tidak berubah walau tarif atau data pertemuan berubah.
						{:else}
							Angka dihitung langsung dari data pertemuan dan tarif saat ini. Kunci bulan sebelum menandai pembayaran.
						{/if}
					</p>
					{#if rekap.dibuka_at}<p class="mt-1 text-xs text-neutral-500">Terakhir dibuka {rekap.dibuka_at}{rekap.dibuka_oleh ? ` oleh ${rekap.dibuka_oleh}` : ""}.</p>{/if}
				</div>
			</div>
			{#if canEdit && !rekap.terkunci}
				<button onclick={kunci} disabled={busy} class="inline-flex items-center justify-center gap-2 rounded-xl bg-brand-600 hover:bg-brand-700 px-4 py-2.5 text-sm font-semibold text-white disabled:opacity-50 whitespace-nowrap">
					<Lock size="16" /> Kunci bulan
				</button>
			{:else if isSuperAdmin && rekap.terkunci}
				<button onclick={buka} disabled={busy} class="inline-flex items-center justify-center gap-2 rounded-xl border border-neutral-300 px-4 py-2.5 text-sm font-semibold text-neutral-700 hover:bg-neutral-100 disabled:opacity-50 dark:border-neutral-700 dark:text-neutral-200 dark:hover:bg-neutral-800 whitespace-nowrap">
					<LockOpen size="16" /> Buka kunci
				</button>
			{/if}
		</section>

		{#if rekap.berlangsung > 0 || rekap.tanpa_guru.length > 0 || adaSelisih}
			<section class="space-y-2" aria-label="Peringatan">
				{#if rekap.berlangsung > 0}
					<div class="flex items-start gap-3 rounded-2xl border border-warning/20 bg-warning/10 p-4 text-sm text-neutral-700 dark:text-neutral-300">
						<TriangleAlert size="18" class="text-warning shrink-0 mt-0.5" />
						<p><strong>{rekap.berlangsung} pertemuan masih berlangsung</strong> di bulan ini dan belum dihitung. Minta guru menutup pertemuannya agar ujroh-nya masuk.</p>
					</div>
				{/if}
				{#if rekap.tanpa_guru.length > 0}
					<div class="flex items-start gap-3 rounded-2xl border border-warning/20 bg-warning/10 p-4 text-sm text-neutral-700 dark:text-neutral-300">
						<TriangleAlert size="18" class="text-warning shrink-0 mt-0.5" />
						<div>
							<p><strong>{rekap.tanpa_guru.length} pertemuan tidak punya guru</strong> sehingga tidak dibayar ke siapa pun. Tetapkan guru kelasnya di menu Kelas.</p>
							<ul class="mt-1 text-xs text-neutral-600 dark:text-neutral-400">
								{#each rekap.tanpa_guru as p (p.pertemuan_id)}<li>{p.tanggal} · {p.kelas_nama} · pertemuan ke-{p.pertemuan_ke}</li>{/each}
							</ul>
						</div>
					</div>
				{/if}
				{#if adaSelisih}
					<div class="flex items-start gap-3 rounded-2xl border border-error/20 bg-error/10 p-4 text-sm text-neutral-700 dark:text-neutral-300">
						<CircleAlert size="18" class="text-error shrink-0 mt-0.5" />
						<div>
							<p><strong>Data pertemuan berubah setelah bulan dikunci.</strong> Angka terkunci tidak ikut berubah. Periksa baris bertanda "Selisih"; bila belum ada yang dibayar, super admin dapat membuka kunci lalu mengunci ulang.</p>
							{#if rekap.guru_baru_selisih.length > 0}
								<ul class="mt-1 text-xs text-neutral-600 dark:text-neutral-400">
									{#each rekap.guru_baru_selisih as g (g.guru_id)}<li>{g.guru_nama}: {g.live_jumlah} pertemuan ({rupiah(g.live_total)}) belum masuk rekap terkunci</li>{/each}
								</ul>
							{/if}
						</div>
					</div>
				{/if}
			</section>
		{/if}

		<section class="grid grid-cols-2 lg:grid-cols-4 gap-3">
			<div class="rounded-2xl bg-neutral-900 text-white p-5"><p class="text-xs text-neutral-400">Total ujroh</p><p class="mt-3 text-2xl font-bold">{rupiah(rekap.ringkasan.total_ujroh)}</p><p class="mt-1 text-sm text-neutral-400">{rekap.ringkasan.jumlah_guru} guru</p></div>
			<div class="rounded-2xl border border-success/20 bg-success/10 p-5"><p class="text-xs text-neutral-600 dark:text-neutral-300">Sudah dibayar</p><p class="mt-3 text-2xl font-bold text-success">{rupiah(rekap.ringkasan.total_dibayar)}</p></div>
			<div class="rounded-2xl border border-warning/20 bg-warning/10 p-5"><p class="text-xs text-neutral-600 dark:text-neutral-300">Belum dibayar</p><p class="mt-3 text-2xl font-bold text-warning">{rupiah(rekap.ringkasan.total_belum)}</p></div>
			<div class="rounded-2xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-925 p-5"><p class="text-xs text-neutral-600 dark:text-neutral-300">Pertemuan selesai</p><p class="mt-3 text-2xl font-bold text-neutral-900 dark:text-white">{rekap.ringkasan.jumlah_pertemuan}</p></div>
		</section>

		<section class="rounded-2xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-925 overflow-hidden">
			{#if rekap.guru.length === 0}
				<div class="p-12 text-center">
					<p class="font-medium text-neutral-700 dark:text-neutral-300">Belum ada pertemuan selesai di {namaBulan(rekap.bulan)}</p>
					<p class="mt-1 text-sm text-neutral-500">Ujroh muncul otomatis setelah guru menyelesaikan pertemuan. Pilih bulan lain untuk melihat rekap sebelumnya.</p>
				</div>
			{:else}
				<div class="overflow-x-auto">
					<table class="w-full min-w-[820px] text-sm">
						<thead>
							<tr class="border-b border-neutral-200 dark:border-neutral-800 bg-neutral-50 dark:bg-neutral-900/60 text-left text-xs font-semibold uppercase tracking-wide text-neutral-500">
								<th class="px-4 py-3">Guru</th>
								<th class="px-4 py-3 text-right">Pertemuan</th>
								<th class="px-4 py-3 text-right">Tarif</th>
								<th class="px-4 py-3 text-right">Total</th>
								<th class="px-4 py-3">Pembayaran</th>
							</tr>
						</thead>
						<tbody class="divide-y divide-neutral-100 dark:divide-neutral-800/70">
							{#each rekap.guru as g (g.guru_id)}
								<tr class="align-top">
									<td class="px-4 py-3">
										<button onclick={() => (expanded = expanded === g.guru_id ? null : g.guru_id)} aria-expanded={expanded === g.guru_id} class="flex items-start gap-2 text-left">
											<ChevronDown size="16" class="mt-0.5 shrink-0 text-neutral-400 transition-transform {expanded === g.guru_id ? 'rotate-180' : ''}" />
											<span>
												<span class="font-semibold text-neutral-900 dark:text-white">{g.guru_nama}</span>
												<span class="mt-0.5 flex flex-wrap gap-1.5 text-xs text-neutral-500">
													<span>{statusLabel(g.guru_status)}</span>
													{#if g.tarif_khusus}<span class="rounded-md bg-brand-400/10 px-1.5 text-brand-700 dark:text-brand-300">tarif khusus</span>{/if}
													{#if g.selisih}<span class="rounded-md bg-error/10 px-1.5 text-error">Selisih: data kini {g.live_jumlah} pertemuan / {rupiah(g.live_total)}</span>{/if}
												</span>
											</span>
										</button>
									</td>
									<td class="px-4 py-3 text-right font-mono">{g.jumlah_pertemuan}{#if g.jumlah_badal > 0}<span class="block text-xs text-neutral-500">{g.jumlah_badal} badal</span>{/if}</td>
									<td class="px-4 py-3 text-right font-mono">{rupiah(g.tarif)}</td>
									<td class="px-4 py-3 text-right font-mono font-semibold text-neutral-900 dark:text-white">{rupiah(g.total)}</td>
									<td class="px-4 py-3">
										{#if g.dibayar}
											<div class="flex items-start gap-2">
												<CircleCheck size="16" class="mt-0.5 shrink-0 text-success" />
												<div class="text-xs">
													<p class="font-semibold text-success">Dibayar {g.dibayar_at}</p>
													{#if g.dibayar_oleh}<p class="text-neutral-500">oleh {g.dibayar_oleh}</p>{/if}
													{#if g.catatan_bayar}<p class="text-neutral-500">{g.catatan_bayar}</p>{/if}
													{#if canEdit}<button onclick={() => batalBayar(g)} disabled={busy} class="mt-1 text-neutral-500 underline hover:text-error">Batalkan</button>{/if}
												</div>
											</div>
										{:else if !rekap.terkunci}
											<span class="text-xs text-neutral-500">Kunci bulan dulu</span>
										{:else if canEdit && payingGuru === g.guru_id}
											<div class="space-y-2">
												<label class="block text-xs text-neutral-600 dark:text-neutral-400">Tanggal bayar<input type="date" bind:value={payTanggal} class="{inputClass} mt-1" /></label>
												<label class="block text-xs text-neutral-600 dark:text-neutral-400">Catatan<input bind:value={payCatatan} placeholder="mis. transfer BSI" class="{inputClass} mt-1" /></label>
												<div class="flex gap-2">
													<button onclick={() => simpanBayar(g)} disabled={busy || !payTanggal} class="rounded-lg bg-brand-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-brand-700 disabled:opacity-50">Simpan</button>
													<button onclick={() => (payingGuru = null)} class="rounded-lg px-3 py-1.5 text-xs font-medium text-neutral-600 hover:bg-neutral-100 dark:text-neutral-300 dark:hover:bg-neutral-800">Batal</button>
												</div>
											</div>
										{:else if canEdit}
											<button onclick={() => mulaiBayar(g)} class="rounded-lg border border-neutral-300 px-3 py-1.5 text-xs font-semibold text-neutral-700 hover:bg-neutral-100 dark:border-neutral-700 dark:text-neutral-200 dark:hover:bg-neutral-800 whitespace-nowrap">Tandai dibayar</button>
										{:else}
											<span class="text-xs text-warning">Belum dibayar</span>
										{/if}
									</td>
								</tr>
								{#if expanded === g.guru_id}
									<tr class="bg-neutral-50/70 dark:bg-neutral-900/40">
										<td colspan="5" class="px-4 py-4">
											<div class="grid gap-3 md:grid-cols-2">
												{#each perKelas(g.pertemuan) as k (k.id)}
													<div class="rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-925 p-3">
														<p class="text-sm font-semibold text-neutral-900 dark:text-white">{k.nama}</p>
														<p class="text-xs text-neutral-500">{k.items.length} pertemuan · {rupiah(k.items.reduce((a, p) => a + p.tarif, 0))}</p>
														<ul class="mt-2 flex flex-wrap gap-1.5">
															{#each k.items as p (p.pertemuan_id)}
																<li class="rounded-md px-2 py-0.5 text-xs {p.is_badal ? 'bg-brand-400/10 text-brand-700 dark:text-brand-300' : 'bg-neutral-100 text-neutral-700 dark:bg-neutral-800 dark:text-neutral-300'}" title="Pertemuan ke-{p.pertemuan_ke}">
																	{tanggalPendek(p.tanggal)} · ke-{p.pertemuan_ke}{p.is_badal ? " · badal" : ""}
																</li>
															{/each}
														</ul>
													</div>
												{/each}
											</div>
										</td>
									</tr>
								{/if}
							{/each}
						</tbody>
					</table>
				</div>
			{/if}
		</section>

		<section class="rounded-2xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-925">
			<button onclick={bukaTarif} aria-expanded={showTarif} class="flex w-full items-center justify-between gap-3 px-5 py-4 text-left">
				<span class="flex items-center gap-2 font-semibold text-neutral-900 dark:text-white"><Settings2 size="18" /> Tarif ujroh</span>
				<span class="text-sm text-neutral-500">Tetap {rupiah(tarif.tetap)} · Part Time {rupiah(tarif.part_time)} per pertemuan</span>
			</button>
			{#if showTarif}
				<div class="space-y-5 border-t border-neutral-200 dark:border-neutral-800 px-5 py-5">
					<p class="text-sm text-neutral-600 dark:text-neutral-400">Perubahan tarif hanya memengaruhi bulan yang belum dikunci.</p>
					<div class="grid gap-3 sm:grid-cols-3 sm:items-end">
						<label class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Guru Tetap (Rp)<input type="number" min="0" step="1000" bind:value={tarifForm.tetap} disabled={!canEdit} class="{inputClass} mt-1.5" /></label>
						<label class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Guru Part Time (Rp)<input type="number" min="0" step="1000" bind:value={tarifForm.part_time} disabled={!canEdit} class="{inputClass} mt-1.5" /></label>
						{#if canEdit}<button onclick={simpanTarif} disabled={busy} class="rounded-xl bg-brand-600 px-4 py-2.5 text-sm font-semibold text-white hover:bg-brand-700 disabled:opacity-50">Simpan tarif</button>{/if}
					</div>

					<div>
						<p class="text-sm font-semibold text-neutral-900 dark:text-white">Tarif khusus per guru</p>
						<p class="text-xs text-neutral-500">Kosongkan untuk memakai tarif sesuai status.</p>
						<div class="mt-3 divide-y divide-neutral-100 dark:divide-neutral-800/70 rounded-xl border border-neutral-200 dark:border-neutral-800">
							{#each tarifGuru as g (g.guru_id)}
								<div class="flex flex-col sm:flex-row sm:items-center gap-2 px-3 py-2.5">
									<div class="min-w-0 flex-1">
										<p class="text-sm font-medium text-neutral-900 dark:text-white">{g.nama}{#if !g.is_aktif}<span class="ml-1 text-xs text-neutral-500">(nonaktif)</span>{/if}</p>
										<p class="text-xs text-neutral-500">{statusLabel(g.status)} · default {rupiah(g.status === "tetap" ? tarif.tetap : tarif.part_time)}</p>
									</div>
									<div class="flex items-center gap-2">
										<label class="sr-only" for="tarif-guru-{g.guru_id}">Tarif khusus {g.nama}</label>
										<input id="tarif-guru-{g.guru_id}" type="number" min="0" step="1000" placeholder="Default" bind:value={overrideDraft[g.guru_id]} disabled={!canEdit} class="{inputClass} w-36" />
										{#if canEdit}<button onclick={() => simpanTarifGuru(g)} disabled={busy} class="rounded-lg border border-neutral-300 px-3 py-2 text-xs font-semibold text-neutral-700 hover:bg-neutral-100 disabled:opacity-50 dark:border-neutral-700 dark:text-neutral-200 dark:hover:bg-neutral-800">Simpan</button>{/if}
									</div>
								</div>
							{:else}
								<p class="px-3 py-6 text-center text-sm text-neutral-500">Belum ada guru aktif.</p>
							{/each}
						</div>
					</div>
				</div>
			{/if}
		</section>
	</main>
</AppLayout>
