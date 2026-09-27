<script lang="ts">
	import { inertia, router } from "@inertiajs/svelte";
	import AppLayout from "@layouts/AppLayout.svelte";
	import type { Tagihan, User, WaTemplate } from "@lib/types";
	import { getCSRFToken } from "@lib/utils/csrf";
	import { Check, Eye, MessageCircle, RefreshCw, Search, X, Pencil, RotateCcw, Settings2, Plus, Trash2 } from "lucide-svelte";

	interface Props {
		user?: User;
		tagihan?: Tagihan[];
		templates?: WaTemplate[];
		filter?: {
			status?: string;
			search?: string;
			tanggal_dari?: string;
			tanggal_sampai?: string;
			kelas_id?: string;
			angkatan_kelas?: string;
			guru_id?: string;
			frekuensi?: string;
			level?: string;
			gender?: string;
			bulan_ke?: string;
		};
	}

	let props: Props = $props();
	let tagihan = $derived(props.tagihan ?? []);
	let templates = $derived(props.templates ?? []);
	let activeTemplates = $derived(templates.filter((t) => t.is_aktif));
	let canManage = $derived(props.user?.role === "keuangan" || props.user?.role === "super_admin");

	let filter = $state(props.filter ?? {});
	let search = $state(filter.search ?? "");
	let status = $state(filter.status ?? "");
	let tanggalDari = $state(filter.tanggal_dari ?? "");
	let tanggalSampai = $state(filter.tanggal_sampai ?? "");
	let kelasID = $state(filter.kelas_id ?? "");
	let angkatan = $state(filter.angkatan_kelas ?? "");
	let guruID = $state(filter.guru_id ?? "");
	let frekuensi = $state(filter.frekuensi ?? "");
	let level = $state(filter.level ?? "");
	let gender = $state(filter.gender ?? "");
	let bulanKe = $state(filter.bulan_ke ?? "");

	let selected = $state<Tagihan | null>(null);
	let metode = $state("");
	let catatan = $state("");
	let tanggalBayar = $state(new Date().toISOString().slice(0, 10));

	let nominalTarget = $state<Tagihan | null>(null);
	let nominalValue = $state(0);

	let followUpTarget = $state<Tagihan | null>(null);
	let followUpTemplateID = $state("");
	let followUpMessage = $state("");
	let sendingFollowUp = $state(false);

	let showTemplateManager = $state(false);
	let templateEditID = $state<number | null>(null);
	let templateForm = $state({ nama: "", body: "", is_aktif: true });

	const rupiah = (n: number) =>
		new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 0 }).format(n);

	function applyFilter() {
		const p = new URLSearchParams();
		const values = {
			search,
			status,
			tanggal_dari: tanggalDari,
			tanggal_sampai: tanggalSampai,
			kelas_id: kelasID,
			angkatan_kelas: angkatan,
			guru_id: guruID,
			frekuensi,
			level,
			gender,
			bulan_ke: bulanKe,
		};
		for (const [key, value] of Object.entries(values)) if (value) p.set(key, value);
		router.get(`/app/keuangan/tagihan?${p}`);
	}

	function resetFilter() {
		search = "";
		status = "";
		tanggalDari = "";
		tanggalSampai = "";
		kelasID = "";
		angkatan = "";
		guruID = "";
		frekuensi = "";
		level = "";
		gender = "";
		bulanKe = "";
		router.get("/app/keuangan/tagihan");
	}

	function lunasi() {
		if (!selected) return;
		router.put(
			`/app/keuangan/tagihan/${selected.id}/lunas`,
			{ tanggal_bayar: tanggalBayar, metode, catatan },
			{ onSuccess: () => { selected = null; } },
		);
	}

	function batal(t: Tagihan) {
		if (!canManage) return;
		if (confirm(`Batalkan tagihan ${t.santri_nama}?`)) {
			router.put(`/app/keuangan/tagihan/${t.id}/batal`, { catatan: "Dibatalkan oleh petugas" });
		}
	}

	function sync() {
		if (canManage) router.post("/app/keuangan/tagihan/sync");
	}

	function openNominal(t: Tagihan) {
		nominalTarget = t;
		nominalValue = t.nominal;
	}

	function saveNominal() {
		if (!nominalTarget || nominalValue < 0) return;
		router.put(
			`/app/keuangan/tagihan/${nominalTarget.id}/nominal`,
			{ nominal: nominalValue },
			{ onSuccess: () => { nominalTarget = null; } },
		);
	}

	function resetNominal(t: Tagihan) {
		if (!canManage || !t.nominal_override) return;
		if (!confirm(`Kembalikan nominal tagihan ${t.santri_nama} agar mengikuti Data Santri?`)) return;
		router.delete(`/app/keuangan/tagihan/${t.id}/nominal`);
	}

	function renderTemplate(body: string, t: Tagihan): string {
		const values: Record<string, string> = {
			"{nama}": t.santri_nama || "",
			"{id_mahasantri}": t.id_mahasantri || "",
			"{bulan_ke}": String(t.bulan_ke ?? ""),
			"{pertemuan_ke}": String(t.pertemuan_ke ?? ""),
			"{nominal}": new Intl.NumberFormat("id-ID").format(t.nominal || 0),
			"{tanggal_tagih}": t.tanggal_tagih || "",
			"{jatuh_tempo}": t.jatuh_tempo || "-",
			"{kelas}": t.kelas_nama || "-",
			"{guru}": t.guru_nama || "-",
			"{angkatan}": t.angkatan || "-",
			"{angkatan_kelas}": t.angkatan_kelas || "-",
			"{frekuensi}": t.frekuensi || "-",
		};
		let result = body;
		for (const [key, value] of Object.entries(values)) result = result.split(key).join(value);
		return result;
	}

	function openFollowUp(t: Tagihan) {
		followUpTarget = t;
		const first = activeTemplates[0];
		followUpTemplateID = first ? String(first.id) : "";
		followUpMessage = first ? renderTemplate(first.body, t) : "";
	}

	function chooseFollowUpTemplate() {
		if (!followUpTarget) return;
		const current = templates.find((t) => String(t.id) === followUpTemplateID);
		followUpMessage = current ? renderTemplate(current.body, followUpTarget) : "";
	}

	async function sendFollowUp() {
		if (!followUpTarget || !followUpMessage.trim()) return;
		sendingFollowUp = true;
		const popup = window.open("", "_blank");
		try {
			const response = await fetch(`/app/keuangan/tagihan/${followUpTarget.id}/follow-up`, {
				method: "POST",
				headers: {
					"Content-Type": "application/json",
					"X-XSRF-TOKEN": getCSRFToken(),
				},
				body: JSON.stringify({
					template_id: Number(followUpTemplateID || 0),
					message: followUpMessage,
				}),
			});
			const data = await response.json();
			if (!response.ok) {
				popup?.close();
				alert(data.error ?? data.message ?? "Gagal membuat tautan WhatsApp");
				return;
			}
			if (popup) popup.location.href = data.url;
			else window.location.href = data.url;
			followUpTarget = null;
			router.reload({ only: ["tagihan"] });
		} finally {
			sendingFollowUp = false;
		}
	}

	function newTemplate() {
		templateEditID = null;
		templateForm = { nama: "", body: "", is_aktif: true };
	}

	function editTemplate(t: WaTemplate) {
		templateEditID = t.id;
		templateForm = { nama: t.nama, body: t.body, is_aktif: t.is_aktif };
	}

	function saveTemplate() {
		if (!templateForm.nama.trim() || !templateForm.body.trim()) return;
		const payload = { ...templateForm, target_type: "tagihan" };
		const opts = { onSuccess: () => { newTemplate(); } };
		if (templateEditID) router.put(`/app/keuangan/tagihan/template/${templateEditID}`, payload, opts);
		else router.post("/app/keuangan/tagihan/template", payload, opts);
	}

	function deleteTemplate(t: WaTemplate) {
		if (!confirm(`Hapus template "${t.nama}"?`)) return;
		router.delete(`/app/keuangan/tagihan/template/${t.id}`);
		if (templateEditID === t.id) newTemplate();
	}

	function statusClass(value: string) {
		return value === "lunas"
			? "bg-success/10 text-success"
			: value === "batal"
				? "bg-neutral-200 text-neutral-600 dark:bg-neutral-800 dark:text-neutral-300"
				: value === "terlambat"
					? "bg-error/10 text-error"
					: "bg-warning/10 text-warning";
	}
</script>

<AppLayout user={props.user}>
	<main class="max-w-7xl mx-auto px-4 sm:px-6 py-8 space-y-6">
		<header class="flex flex-col sm:flex-row sm:items-end justify-between gap-4">
			<div>
				<p class="text-xs font-bold tracking-[0.16em] uppercase text-brand-600 dark:text-brand-400">Keuangan</p>
				<h1 class="mt-2 text-3xl font-bold text-neutral-900 dark:text-white">Daftar tagihan</h1>
				<p class="mt-2 text-sm text-neutral-600 dark:text-neutral-400">{tagihan.length} tagihan sesuai filter. Filter angkatan menggunakan Angkatan Kelas.</p>
			</div>
			{#if canManage}
				<div class="flex flex-wrap gap-2">
					<button onclick={() => { showTemplateManager = true; newTemplate(); }} class="inline-flex items-center justify-center gap-2 px-4 py-2.5 rounded-xl border border-neutral-300 dark:border-neutral-700 text-neutral-700 dark:text-neutral-200 font-semibold text-sm">
						<Settings2 size="16" /> Template FU
					</button>
					<button onclick={sync} class="inline-flex items-center justify-center gap-2 px-4 py-2.5 rounded-xl border border-brand-500/30 text-brand-700 dark:text-brand-300 font-semibold text-sm">
						<RefreshCw size="16" /> Sinkronkan tagihan
					</button>
				</div>
			{/if}
		</header>

		<form onsubmit={(e) => { e.preventDefault(); applyFilter(); }} class="grid sm:grid-cols-2 lg:grid-cols-4 gap-3 p-4 rounded-2xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-925">
			<div class="lg:col-span-2 relative"><Search size="16" class="absolute top-3 left-3 text-neutral-400" /><input bind:value={search} placeholder="Nama atau ID mahasantri" class="w-full pl-9 pr-3 py-2.5 rounded-xl bg-neutral-100 dark:bg-neutral-800 text-sm outline-none" /></div>
			<select bind:value={status} class="px-3 py-2.5 rounded-xl bg-neutral-100 dark:bg-neutral-800 text-sm"><option value="">Semua status</option><option value="belum_bayar">Belum bayar</option><option value="terlambat">Terlambat</option><option value="lunas">Lunas</option><option value="batal">Batal</option></select>
			<input bind:value={bulanKe} type="number" min="1" placeholder="Bulan ke-" class="px-3 py-2.5 rounded-xl bg-neutral-100 dark:bg-neutral-800 text-sm" />
			<input type="date" bind:value={tanggalDari} title="Tanggal tagih dari" class="px-3 py-2.5 rounded-xl bg-neutral-100 dark:bg-neutral-800 text-sm" />
			<input type="date" bind:value={tanggalSampai} title="Tanggal tagih sampai" class="px-3 py-2.5 rounded-xl bg-neutral-100 dark:bg-neutral-800 text-sm" />
			<input bind:value={angkatan} placeholder="Angkatan" class="px-3 py-2.5 rounded-xl bg-neutral-100 dark:bg-neutral-800 text-sm" />
			<input bind:value={level} placeholder="Level" class="px-3 py-2.5 rounded-xl bg-neutral-100 dark:bg-neutral-800 text-sm" />
			<select bind:value={frekuensi} class="px-3 py-2.5 rounded-xl bg-neutral-100 dark:bg-neutral-800 text-sm"><option value="">Semua frekuensi</option><option value="1x/pekan">1x/pekan</option><option value="2x/pekan">2x/pekan</option></select>
			<select bind:value={gender} class="px-3 py-2.5 rounded-xl bg-neutral-100 dark:bg-neutral-800 text-sm"><option value="">Semua gender</option><option value="L">Laki-laki</option><option value="P">Perempuan</option></select>
			<input bind:value={kelasID} type="number" min="1" placeholder="ID kelas" class="px-3 py-2.5 rounded-xl bg-neutral-100 dark:bg-neutral-800 text-sm" />
			<input bind:value={guruID} type="number" min="1" placeholder="ID guru" class="px-3 py-2.5 rounded-xl bg-neutral-100 dark:bg-neutral-800 text-sm" />
			<div class="flex gap-2 sm:col-span-2 lg:col-span-2">
				<button type="submit" class="flex-1 px-4 py-2.5 rounded-xl bg-brand-600 text-white text-sm font-semibold">Terapkan</button>
				<button type="button" onclick={resetFilter} class="inline-flex items-center justify-center gap-2 px-4 py-2.5 rounded-xl border border-neutral-300 dark:border-neutral-700 text-neutral-700 dark:text-neutral-200 text-sm font-semibold hover:bg-neutral-50 dark:hover:bg-neutral-800">
					<RotateCcw size="16" /> Reset Filter
				</button>
			</div>
		</form>

		<div class="rounded-2xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-925 overflow-x-auto">
			<table class="w-full min-w-[1000px] text-sm">
				<thead class="text-left text-xs uppercase tracking-wider text-neutral-500 border-b border-neutral-200 dark:border-neutral-800">
					<tr><th class="p-4">Santri</th><th class="p-4">Kelas</th><th class="p-4">Periode</th><th class="p-4">Nominal</th><th class="p-4">Tagih / Tempo</th><th class="p-4">Status</th><th class="p-4 text-right">Aksi</th></tr>
				</thead>
				<tbody class="divide-y divide-neutral-100 dark:divide-neutral-800/70">
					{#each tagihan as t (t.id)}
						<tr>
							<td class="p-4"><p class="font-semibold text-neutral-900 dark:text-white">{t.santri_nama}</p><p class="text-xs text-neutral-500">{t.id_mahasantri || "-"} · {t.frekuensi}</p></td>
							<td class="p-4 text-neutral-600 dark:text-neutral-300">{t.kelas_nama || "-"}<br /><span class="text-xs text-neutral-500">{t.guru_nama}</span></td>
							<td class="p-4">Bulan {t.bulan_ke}<br /><span class="text-xs text-neutral-500">P-{t.pertemuan_ke}</span></td>
							<td class="p-4">
								<p class="font-mono">{rupiah(t.nominal)}</p>
								{#if t.nominal_override}<span class="mt-1 inline-flex px-2 py-0.5 rounded-full bg-indigo-500/10 text-indigo-600 dark:text-indigo-400 text-[11px] font-semibold">Nominal khusus</span>{/if}
							</td>
							<td class="p-4 text-xs">{t.tanggal_tagih}<br /><span class="text-neutral-500">{t.jatuh_tempo || "-"}</span></td>
							<td class="p-4"><span class="inline-flex px-2 py-1 rounded-full text-xs font-bold {statusClass(t.status)}">{t.status.replace("_", " ")}</span></td>
							<td class="p-4">
								<div class="flex justify-end gap-1">
									<a href={`/app/keuangan/tagihan/${t.id}`} use:inertia title="Detail" class="p-2 text-neutral-500 hover:bg-neutral-100 dark:hover:bg-neutral-800 rounded-lg"><Eye size="17" /></a>
									{#if canManage && (t.status === "belum_bayar" || t.status === "terlambat")}
										<button onclick={() => { selected = t; metode = ""; catatan = ""; }} title="Tandai lunas" class="p-2 text-success hover:bg-success/10 rounded-lg"><Check size="17" /></button>
										<button onclick={() => openNominal(t)} title="Edit nominal tagihan" class="p-2 text-indigo-600 hover:bg-indigo-500/10 rounded-lg"><Pencil size="17" /></button>
										{#if t.nominal_override}<button onclick={() => resetNominal(t)} title="Kembali ikuti nominal Data Santri" class="p-2 text-neutral-500 hover:bg-neutral-100 dark:hover:bg-neutral-800 rounded-lg"><RotateCcw size="17" /></button>{/if}
									{/if}
									<button onclick={() => openFollowUp(t)} disabled={!t.no_wa || activeTemplates.length === 0} title={t.no_wa ? "Follow-up WhatsApp" : "Nomor WA belum diisi"} class="p-2 text-brand-600 hover:bg-brand-500/10 rounded-lg disabled:opacity-30"><MessageCircle size="17" /></button>
									{#if canManage && t.status !== "lunas" && t.status !== "batal"}<button onclick={() => batal(t)} title="Batalkan" class="p-2 text-error hover:bg-error/10 rounded-lg"><X size="17" /></button>{/if}
								</div>
							</td>
						</tr>
					{:else}
						<tr><td colspan="7" class="p-14 text-center text-neutral-500">Tidak ada tagihan yang ditemukan.</td></tr>
					{/each}
				</tbody>
			</table>
		</div>
	</main>

	{#if selected}
		<div class="fixed inset-0 z-50 bg-neutral-950/55 flex items-center justify-center p-4">
			<section class="w-full max-w-md rounded-2xl bg-white dark:bg-neutral-925 p-6 shadow-xl">
				<div class="flex justify-between gap-4"><div><h2 class="text-xl font-bold text-neutral-900 dark:text-white">Tandai lunas</h2><p class="text-sm text-neutral-500 mt-1">{selected.santri_nama} · {rupiah(selected.nominal)}</p></div><button onclick={() => selected = null} class="text-neutral-500"><X /></button></div>
				<div class="mt-5 space-y-3">
					<label class="block text-sm font-medium">Tanggal bayar<input type="date" bind:value={tanggalBayar} class="mt-1 w-full px-3 py-2 rounded-lg bg-neutral-100 dark:bg-neutral-800" /></label>
					<label class="block text-sm font-medium">Metode<input bind:value={metode} placeholder="Contoh: transfer BCA" class="mt-1 w-full px-3 py-2 rounded-lg bg-neutral-100 dark:bg-neutral-800" /></label>
					<label class="block text-sm font-medium">Catatan<textarea bind:value={catatan} class="mt-1 w-full px-3 py-2 rounded-lg bg-neutral-100 dark:bg-neutral-800"></textarea></label>
					<button onclick={lunasi} class="w-full py-2.5 rounded-xl bg-success text-white font-semibold">Simpan pelunasan</button>
				</div>
			</section>
		</div>
	{/if}

	{#if nominalTarget}
		<div class="fixed inset-0 z-50 bg-neutral-950/55 flex items-center justify-center p-4">
			<section class="w-full max-w-md rounded-2xl bg-white dark:bg-neutral-925 p-6 shadow-xl">
				<div class="flex justify-between gap-4"><div><h2 class="text-xl font-bold text-neutral-900 dark:text-white">Edit nominal tagihan</h2><p class="text-sm text-neutral-500 mt-1">{nominalTarget.santri_nama} · Bulan {nominalTarget.bulan_ke}</p></div><button onclick={() => nominalTarget = null} class="text-neutral-500"><X /></button></div>
				<p class="mt-4 text-sm text-neutral-600 dark:text-neutral-400">Nominal ini hanya berlaku untuk tagihan ini. Perubahan berikutnya di Data Santri tidak akan menimpanya sampai nominal direset.</p>
				<label class="block mt-4 text-sm font-medium">Nominal<input type="number" min="0" step="1000" bind:value={nominalValue} class="mt-1 w-full px-3 py-2.5 rounded-lg bg-neutral-100 dark:bg-neutral-800" /></label>
				<div class="mt-5 flex justify-end gap-2"><button onclick={() => nominalTarget = null} class="px-4 py-2.5 rounded-xl text-sm font-semibold">Batal</button><button onclick={saveNominal} class="px-4 py-2.5 rounded-xl bg-brand-600 text-white text-sm font-semibold">Simpan nominal khusus</button></div>
			</section>
		</div>
	{/if}

	{#if followUpTarget}
		<div class="fixed inset-0 z-50 bg-neutral-950/55 flex items-center justify-center p-4">
			<section class="w-full max-w-xl rounded-2xl bg-white dark:bg-neutral-925 p-6 shadow-xl">
				<div class="flex justify-between gap-4"><div><h2 class="text-xl font-bold text-neutral-900 dark:text-white">Follow-up WhatsApp</h2><p class="text-sm text-neutral-500 mt-1">{followUpTarget.santri_nama} · {rupiah(followUpTarget.nominal)}</p></div><button onclick={() => followUpTarget = null} class="text-neutral-500"><X /></button></div>
				<div class="mt-5 space-y-4">
					<label class="block text-sm font-medium">Template
						<select bind:value={followUpTemplateID} onchange={chooseFollowUpTemplate} class="mt-1 w-full px-3 py-2.5 rounded-xl bg-neutral-100 dark:bg-neutral-800">
							{#each activeTemplates as template}<option value={String(template.id)}>{template.nama}</option>{/each}
						</select>
					</label>
					<label class="block text-sm font-medium">Pesan
						<textarea bind:value={followUpMessage} rows="9" class="mt-1 w-full px-3 py-2.5 rounded-xl bg-neutral-100 dark:bg-neutral-800"></textarea>
					</label>
					<p class="text-xs text-neutral-500">Pesan boleh diedit untuk pengiriman ini tanpa mengubah template utama.</p>
					<button onclick={sendFollowUp} disabled={sendingFollowUp || !followUpMessage.trim()} class="w-full py-2.5 rounded-xl bg-brand-600 text-white font-semibold disabled:opacity-50">{sendingFollowUp ? "Menyiapkan WhatsApp..." : "Buka WhatsApp & Catat FU"}</button>
				</div>
			</section>
		</div>
	{/if}

	{#if showTemplateManager}
		<div class="fixed inset-0 z-50 bg-neutral-950/55 flex items-center justify-center p-4">
			<section class="w-full max-w-4xl max-h-[90vh] overflow-y-auto rounded-2xl bg-white dark:bg-neutral-925 shadow-xl">
				<header class="sticky top-0 bg-white dark:bg-neutral-925 flex items-center justify-between px-6 py-4 border-b border-neutral-200 dark:border-neutral-800">
					<div><h2 class="text-xl font-bold text-neutral-900 dark:text-white">Template Follow-up Tagihan</h2><p class="text-sm text-neutral-500">Template global untuk seluruh tim Keuangan.</p></div>
					<button onclick={() => showTemplateManager = false} class="text-neutral-500"><X /></button>
				</header>
				<div class="grid lg:grid-cols-2 gap-6 p-6">
					<div class="space-y-3">
						<div class="flex items-center justify-between"><h3 class="font-semibold">Daftar template</h3><button onclick={newTemplate} class="inline-flex items-center gap-1 text-sm font-semibold text-brand-600"><Plus size="16" /> Baru</button></div>
						{#each templates as t (t.id)}
							<div class="rounded-xl border border-neutral-200 dark:border-neutral-800 p-4">
								<div class="flex items-start justify-between gap-3">
									<div><p class="font-semibold text-neutral-900 dark:text-white">{t.nama}</p><p class="text-xs mt-1 {t.is_aktif ? 'text-success' : 'text-neutral-500'}">{t.is_aktif ? "Aktif" : "Nonaktif"}</p></div>
									<div class="flex gap-1"><button onclick={() => editTemplate(t)} class="p-2 text-neutral-500 hover:bg-neutral-100 dark:hover:bg-neutral-800 rounded-lg"><Pencil size="16" /></button><button onclick={() => deleteTemplate(t)} class="p-2 text-error hover:bg-error/10 rounded-lg"><Trash2 size="16" /></button></div>
								</div>
								<p class="mt-3 text-xs text-neutral-500 line-clamp-3 whitespace-pre-wrap">{t.body}</p>
							</div>
						{:else}<p class="text-sm text-neutral-500">Belum ada template tagihan.</p>{/each}
					</div>
					<form onsubmit={(e) => { e.preventDefault(); saveTemplate(); }} class="space-y-4 rounded-xl border border-neutral-200 dark:border-neutral-800 p-5 h-fit">
						<h3 class="font-semibold">{templateEditID ? "Edit template" : "Tambah template"}</h3>
						<label class="block text-sm font-medium">Nama<input bind:value={templateForm.nama} required class="mt-1 w-full px-3 py-2.5 rounded-xl bg-neutral-100 dark:bg-neutral-800" placeholder="Contoh: FU 4 - Konfirmasi Akhir" /></label>
						<label class="block text-sm font-medium">Isi pesan<textarea bind:value={templateForm.body} required rows="9" class="mt-1 w-full px-3 py-2.5 rounded-xl bg-neutral-100 dark:bg-neutral-800"></textarea></label>
						<p class="text-xs text-neutral-500">Variabel: <code>{"{nama}"}</code>, <code>{"{id_mahasantri}"}</code>, <code>{"{nominal}"}</code>, <code>{"{bulan_ke}"}</code>, <code>{"{pertemuan_ke}"}</code>, <code>{"{tanggal_tagih}"}</code>, <code>{"{jatuh_tempo}"}</code>, <code>{"{kelas}"}</code>, <code>{"{guru}"}</code>, <code>{"{angkatan}"}</code>, <code>{"{angkatan_kelas}"}</code>, <code>{"{frekuensi}"}</code>.</p>
						<label class="flex items-center gap-2 text-sm"><input type="checkbox" bind:checked={templateForm.is_aktif} /> Aktif untuk dipilih saat follow-up</label>
						<div class="flex justify-end gap-2"><button type="button" onclick={newTemplate} class="px-4 py-2.5 rounded-xl text-sm font-semibold">Reset</button><button type="submit" class="px-4 py-2.5 rounded-xl bg-brand-600 text-white text-sm font-semibold">Simpan template</button></div>
					</form>
				</div>
			</section>
		</div>
	{/if}
</AppLayout>
