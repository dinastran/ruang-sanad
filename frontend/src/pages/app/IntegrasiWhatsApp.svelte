<script lang="ts">
	import { router } from "@inertiajs/svelte";
	import AppLayout from "@layouts/AppLayout.svelte";
	import type { Flash, User } from "@lib/types";
	import { formatDateTime, relativeTime } from "@lib/leads";
	import { Plus, X } from "lucide-svelte";

	interface Settings {
		paket_ke_kelas_kode: Record<string, string>;
		followup_enabled: boolean;
		followup_send_start: string;
		followup_send_end: string;
		followup_min_gap_seconds: number;
		followup_daily_cap: number;
		transcript_retention_days_lost: number;
		payment_verified_template: string;
		persona_blocked_keywords: string;
		followup_only_label: string;
	}
	interface Page {
		settings: Settings;
		ingest_enabled: boolean;
		outbound_ready: boolean;
		last_bridge_event: string;
		bridge_events_today: number;
		statuses: { name: string; target: string; ok: boolean; message: string }[];
	}
	interface Props {
		user?: User;
		page: Page;
		flash?: Flash;
	}

	let props: Props = $props();
	let page = $derived(props.page);

	// svelte-ignore state_referenced_locally
	let mapping = $state(Object.entries(props.page.settings.paket_ke_kelas_kode).map(([paket, kode]) => ({ paket, kode })));
	// svelte-ignore state_referenced_locally
	let retention = $state(props.page.settings.transcript_retention_days_lost);
	// svelte-ignore state_referenced_locally
	let blockedKeywords = $state(props.page.settings.persona_blocked_keywords);
	// svelte-ignore state_referenced_locally
	let fu = $state({
		followup_enabled: props.page.settings.followup_enabled,
		followup_send_start: props.page.settings.followup_send_start,
		followup_send_end: props.page.settings.followup_send_end,
		followup_min_gap_seconds: props.page.settings.followup_min_gap_seconds,
		followup_daily_cap: props.page.settings.followup_daily_cap,
		followup_only_label: props.page.settings.followup_only_label,
		payment_verified_template: props.page.settings.payment_verified_template,
	});
	let saving = $state(false);

	// Ingest looks stale when nothing arrived for over a day.
	let ingestStale = $derived(!page.last_bridge_event || Date.now() - new Date(page.last_bridge_event).getTime() > 24 * 3600 * 1000);

	function save(e: Event) {
		e.preventDefault();
		saving = true;
		const paket_ke_kelas_kode: Record<string, string> = {};
		for (const row of mapping) if (row.paket.trim()) paket_ke_kelas_kode[row.paket.trim().toLowerCase()] = row.kode.trim();
		if (fu.followup_enabled && !page.settings.followup_enabled && !confirm("Nyalakan pengiriman follow-up lewat nomor CS? Hanya follow-up yang sudah disetujui yang dikirim.")) {
			saving = false;
			return;
		}
		router.put("/app/integrasi-whatsapp/settings", {
			...page.settings,
			...fu,
			followup_min_gap_seconds: Number(fu.followup_min_gap_seconds),
			followup_daily_cap: Number(fu.followup_daily_cap),
			paket_ke_kelas_kode,
			transcript_retention_days_lost: Number(retention),
			persona_blocked_keywords: blockedKeywords,
		}, { preserveScroll: true, onFinish: () => (saving = false) });
	}

	const field = "w-full px-3 py-2 rounded-lg bg-white dark:bg-neutral-900/60 border border-neutral-300 dark:border-neutral-700/80 text-sm text-neutral-900 dark:text-white";
	const card = "rounded-xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-4";
</script>

<AppLayout user={props.user}>
	<div class="max-w-4xl mx-auto px-4 sm:px-6 py-6 space-y-5">
		<div>
			<h1 class="text-2xl sm:text-3xl font-bold text-neutral-900 dark:text-white tracking-tight">Integrasi WhatsApp</h1>
			<p class="mt-1 text-sm text-neutral-600 dark:text-neutral-400">Koneksi GOWA dan bridge CS AI, serta pengaturan CRM leads. Token dan password tetap di file .env server.</p>
		</div>
		{#if props.flash?.success}<div class="rounded-xl border border-success/20 bg-success/10 px-4 py-3 text-sm font-medium text-success">{props.flash.success}</div>{/if}
		{#if props.flash?.error}<div class="rounded-xl border border-error/20 bg-error/10 px-4 py-3 text-sm font-medium text-error">{props.flash.error}</div>{/if}

		<section class={card}>
			<h2 class="text-sm font-semibold text-neutral-900 dark:text-white">Event dari bridge</h2>
			{#if !page.ingest_enabled}
				<p class="mt-2 text-sm text-warning">Ingest mati: CS_BRIDGE_INGEST_TOKEN belum diisi di .env, jadi endpoint /integrations/cs-ai/events membalas 404.</p>
			{:else}
				<dl class="mt-2 grid grid-cols-2 gap-3 text-sm">
					<div><dt class="text-xs text-neutral-500">Event terakhir</dt><dd class="font-medium {ingestStale ? 'text-warning' : 'text-neutral-900 dark:text-white'}">{page.last_bridge_event ? `${relativeTime(page.last_bridge_event)} (${formatDateTime(page.last_bridge_event)})` : "Belum pernah"}</dd></div>
					<div><dt class="text-xs text-neutral-500">Event hari ini</dt><dd class="font-medium text-neutral-900 dark:text-white">{page.bridge_events_today}</dd></div>
				</dl>
				{#if ingestStale}<p class="mt-2 text-xs text-warning">Tidak ada event lebih dari 24 jam. Kalau nomor CS tetap menerima chat, cek bridge dan antrean lokalnya.</p>{/if}
			{/if}
		</section>

		<section class={card}>
			<h2 class="text-sm font-semibold text-neutral-900 dark:text-white">Status koneksi</h2>
			<ul class="mt-2 divide-y divide-neutral-200/80 dark:divide-white/[0.05]">
				{#each page.statuses as s}
					<li class="flex items-start justify-between gap-3 py-2 text-sm">
						<div class="min-w-0"><p class="font-medium text-neutral-800 dark:text-neutral-200">{s.name}</p>{#if s.target}<p class="text-xs text-neutral-500 font-mono break-all">{s.target}</p>{/if}</div>
						<span class="shrink-0 rounded-full px-2 py-0.5 text-xs font-medium {s.ok ? 'bg-success/10 text-success' : 'bg-error/10 text-error'}">{s.ok ? "Terhubung" : s.message || "Gagal"}</span>
					</li>
				{/each}
			</ul>
		</section>

		<form class="{card} space-y-4" onsubmit={save}>
			<div>
				<h2 class="text-sm font-semibold text-neutral-900 dark:text-white">Pemetaan paket ke kode kelas</h2>
				<p class="text-xs text-neutral-500">Paket yang dikirim CS AI (mis. 1x, 2x) diubah menjadi kode kelas Dashboard.</p>
				<div class="mt-3 space-y-2">
					{#each mapping as row, i}
						<div class="flex items-center gap-2">
							<input class="{field} w-24" bind:value={row.paket} placeholder="1x" aria-label="Kode paket" />
							<span class="text-neutral-500">→</span>
							<input class={field} bind:value={row.kode} placeholder="R 1X" aria-label="Kode kelas" />
							<button type="button" class="p-2 rounded-lg hover:bg-neutral-100 dark:hover:bg-neutral-800" aria-label="Hapus baris" onclick={() => (mapping = mapping.filter((_, j) => j !== i))}><X class="w-4 h-4" /></button>
						</div>
					{/each}
					<button type="button" class="inline-flex items-center gap-1 text-sm text-brand-600 dark:text-brand-400" onclick={() => (mapping = [...mapping, { paket: "", kode: "" }])}><Plus class="w-4 h-4" /> Tambah paket</button>
				</div>
			</div>
			<label class="block text-xs font-medium text-neutral-600 dark:text-neutral-400 max-w-xs">
				Hapus transkrip lead hilang / jangan-hubungi setelah (hari)
				<input type="number" min="30" max="3650" class="{field} mt-1" bind:value={retention} />
			</label>
			<fieldset class="space-y-3 border-t border-neutral-200/80 dark:border-white/[0.06] pt-4">
				<legend class="text-sm font-semibold text-neutral-900 dark:text-white">Follow-up lewat nomor CS</legend>
				{#if !page.outbound_ready}<p class="text-xs text-warning">CS_BRIDGE_URL / CS_BRIDGE_OUTBOUND_TOKEN belum diisi di .env, jadi pesan tidak bisa dikirim (akan tercatat gagal).</p>{/if}
				<label class="flex items-start gap-2 text-sm text-neutral-800 dark:text-neutral-200">
					<input type="checkbox" class="mt-0.5" bind:checked={fu.followup_enabled} />
					<span>Kirim follow-up yang sudah disetujui<span class="block text-xs text-neutral-500">Saklar global. Matikan untuk menghentikan semua pengiriman follow-up seketika. Konfirmasi pembayaran tidak terpengaruh.</span></span>
				</label>
				<div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
					<label class="block text-xs font-medium text-neutral-600 dark:text-neutral-400">Jam mulai (WIB)<input type="time" class="{field} mt-1" bind:value={fu.followup_send_start} /></label>
					<label class="block text-xs font-medium text-neutral-600 dark:text-neutral-400">Jam selesai (WIB)<input type="time" class="{field} mt-1" bind:value={fu.followup_send_end} /></label>
					<label class="block text-xs font-medium text-neutral-600 dark:text-neutral-400">Jeda min. (detik)<input type="number" min="10" max="3600" class="{field} mt-1" bind:value={fu.followup_min_gap_seconds} /></label>
					<label class="block text-xs font-medium text-neutral-600 dark:text-neutral-400">Batas per hari<input type="number" min="1" max="500" class="{field} mt-1" bind:value={fu.followup_daily_cap} /></label>
				</div>
				<label class="block text-xs font-medium text-neutral-600 dark:text-neutral-400 max-w-sm">
					Uji bertahap: hanya kirim ke lead berlabel
					<input class="{field} mt-1" bind:value={fu.followup_only_label} placeholder="kosong = semua lead (mis. uji-followup)" />
				</label>
				<label class="block text-xs font-medium text-neutral-600 dark:text-neutral-400">
					Template konfirmasi pembayaran
					<span class="block font-normal text-neutral-500">Dikirim bila CS mencentang "kirim konfirmasi" saat verifikasi. [NAMA] = nama depan, [NOMINAL] = nominal diterima. Kosong = fitur mati.</span>
					<textarea class="{field} mt-1" rows="3" maxlength="1000" bind:value={fu.payment_verified_template}></textarea>
				</label>
			</fieldset>
			<label class="block text-xs font-medium text-neutral-600 dark:text-neutral-400">
				Kata terlarang di catatan persona analisis AI
				<span class="block font-normal text-neutral-500">Satu kata atau frasa per baris. Analisis yang catatan personanya memuat kata ini ditolak (penilaian keagamaan, ekonomi, kesehatan, etnis, politik).</span>
				<textarea class="{field} mt-1 font-mono" rows="6" bind:value={blockedKeywords}></textarea>
			</label>
			<div class="flex justify-end"><button type="submit" disabled={saving} class="px-4 py-2 rounded-lg bg-brand-600 hover:bg-brand-700 text-white text-sm font-semibold dark:bg-brand-500 disabled:opacity-50">{saving ? "Menyimpan..." : "Simpan pengaturan"}</button></div>
		</form>
	</div>
</AppLayout>
