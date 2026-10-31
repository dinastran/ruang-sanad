<script lang="ts">
	import { inertia, router } from "@inertiajs/svelte";
	import AppLayout from "@layouts/AppLayout.svelte";
	import type { Flash, User } from "@lib/types";
	import { STAGE_LABELS, formatDateTime } from "@lib/leads";
	import { fmt } from "@components/lead-dashboard/types";
	import { ArrowLeft, RefreshCw } from "lucide-svelte";

	interface EventRow {
		id: number; lead_id: number; lead_nama: string; event: string; event_time: string; value: number;
		status: string; attempts: number; last_error: string; test_mode: boolean; sent_at: string;
	}
	interface Status {
		enabled: boolean; dataset_id: string; api_version: string; test_mode: boolean; app_secret_proof: boolean;
		counts: Record<string, number>; mapping: { event: string; tahap: string }[]; events: EventRow[];
	}
	interface Props { user?: User; status: Status; flash?: Flash }

	let props: Props = $props();
	let s = $derived(props.status);
	let busy = $state(false);

	const STATUS = {
		pending: { label: "Menunggu", cls: "bg-neutral-500/10 text-neutral-700 dark:text-neutral-300" },
		sent: { label: "Terkirim", cls: "bg-success/10 text-emerald-700 dark:text-emerald-400" },
		failed: { label: "Gagal", cls: "bg-error/10 text-red-700 dark:text-red-400" },
		skipped: { label: "Dilewati", cls: "bg-amber-500/10 text-amber-700 dark:text-amber-400" },
	} as Record<string, { label: string; cls: string }>;
	const tahap = (t: string) => STAGE_LABELS[t] ? `Tahap ${STAGE_LABELS[t]}` : t;
	const rupiah = (n: number) => "Rp" + n.toLocaleString("id-ID");

	function retry() {
		busy = true;
		router.post("/app/leads/meta-capi/retry", {}, { preserveScroll: true, onFinish: () => (busy = false) });
	}

	const panel = "rounded-xl border border-neutral-200/80 bg-white dark:border-white/[0.07] dark:bg-neutral-925/60";
	const kicker = "text-[11px] font-semibold uppercase tracking-[0.08em] text-neutral-500 dark:text-neutral-400";
</script>

<svelte:head><title>Meta CAPI · Ruang Sanad</title></svelte:head>
<AppLayout user={props.user}>
	<div class="mx-auto max-w-5xl space-y-4 px-4 py-6 sm:px-6 lg:py-8">
		<header class="flex flex-col gap-3 pb-1 sm:flex-row sm:items-end sm:justify-between">
			<div class="min-w-0">
				<a href="/app/leads" use:inertia class="inline-flex items-center gap-1 text-xs font-medium text-neutral-500 hover:text-neutral-800 dark:text-neutral-400 dark:hover:text-white"><ArrowLeft size={13} /> Leads</a>
				<h1 class="mt-1 text-[28px] font-semibold leading-tight tracking-[-0.02em] text-neutral-950 dark:text-white">Meta Conversions API</h1>
				<p class="mt-1 max-w-2xl text-sm text-neutral-500 dark:text-neutral-400">Setiap tahap funnel lead dikirim ke Meta sebagai event, sampai Purchase saat jadi santri. Data pribadi hanya dikirim dalam bentuk hash.</p>
			</div>
			{#if s.enabled && (s.counts.failed ?? 0) > 0}
				<button type="button" onclick={retry} disabled={busy} class="inline-flex h-9 shrink-0 items-center gap-1.5 whitespace-nowrap rounded-md border border-neutral-200 px-3 text-sm font-medium text-neutral-700 hover:bg-neutral-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 disabled:opacity-50 dark:border-white/10 dark:text-neutral-300 dark:hover:bg-white/[0.04]"><RefreshCw size={15} /> Kirim ulang yang gagal</button>
			{/if}
		</header>

		{#if props.flash?.success}<div role="status" class="rounded-lg border border-success/20 bg-success/10 px-4 py-3 text-sm font-medium text-emerald-800 dark:text-emerald-300">{props.flash.success}</div>{/if}
		{#if props.flash?.error}<div role="alert" class="rounded-lg border border-error/20 bg-error/10 px-4 py-3 text-sm font-medium text-red-700 dark:text-red-300">{props.flash.error}</div>{/if}

		{#if !s.enabled}
			<div class="rounded-xl border border-amber-500/30 bg-amber-500/[0.06] px-4 py-4 text-sm text-amber-900 sm:px-5 dark:text-amber-200">
				<p class="font-semibold">Belum aktif</p>
				<p class="mt-1">Isi <code class="font-mono text-xs">META_CAPI_DATASET_ID</code> dan <code class="font-mono text-xs">META_CAPI_TOKEN</code> di <code class="font-mono text-xs">.env</code> server lalu restart aplikasi. Langkahnya ada di panduan <code class="font-mono text-xs">docs/panduan/meta-ads-api.md</code> bagian Conversions API.</p>
			</div>
		{/if}

		<section class="grid grid-cols-2 gap-px overflow-hidden rounded-xl border border-neutral-200/80 bg-neutral-100 sm:grid-cols-4 dark:border-white/[0.07] dark:bg-white/[0.05]" aria-label="Ringkasan antrean">
			{#each ["sent", "pending", "failed", "skipped"] as key}
				<div class="bg-white px-4 py-3 sm:px-5 dark:bg-neutral-925">
					<p class="text-xs text-neutral-500 dark:text-neutral-400">{STATUS[key].label}</p>
					<p class={"mt-0.5 text-2xl font-semibold tabular-nums " + (key === "failed" && (s.counts.failed ?? 0) > 0 ? "text-red-700 dark:text-red-400" : "text-neutral-950 dark:text-white")}>{fmt(s.counts[key] ?? 0)}</p>
				</div>
			{/each}
		</section>

		<div class="grid gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
			<section class={panel + " p-5 sm:p-6"} aria-labelledby="capi-config">
				<h2 id="capi-config" class={kicker}>Konfigurasi</h2>
				<dl class="mt-3 space-y-2 text-sm">
					<div class="flex justify-between gap-3"><dt class="text-neutral-500 dark:text-neutral-400">Status</dt><dd class={"font-medium " + (s.enabled ? "text-emerald-700 dark:text-emerald-400" : "text-amber-700 dark:text-amber-400")}>{s.enabled ? "Aktif" : "Belum dikonfigurasi"}</dd></div>
					<div class="flex justify-between gap-3"><dt class="text-neutral-500 dark:text-neutral-400">Dataset / Pixel ID</dt><dd class="font-mono text-neutral-900 dark:text-white">{s.dataset_id || "—"}</dd></div>
					<div class="flex justify-between gap-3"><dt class="text-neutral-500 dark:text-neutral-400">Versi API</dt><dd class="font-mono text-neutral-900 dark:text-white">{s.api_version}</dd></div>
					<div class="flex justify-between gap-3"><dt class="text-neutral-500 dark:text-neutral-400">Mode uji (Test Events)</dt><dd class={"font-medium " + (s.test_mode ? "text-amber-700 dark:text-amber-400" : "text-neutral-900 dark:text-white")}>{s.test_mode ? "Aktif — event hanya tampil di Test Events" : "Nonaktif (produksi)"}</dd></div>
					<div class="flex justify-between gap-3"><dt class="text-neutral-500 dark:text-neutral-400">appsecret_proof</dt><dd class="text-neutral-900 dark:text-white">{s.app_secret_proof ? "Dipakai" : "Tidak dipakai"}</dd></div>
				</dl>
			</section>
			<section class={panel + " p-5 sm:p-6"} aria-labelledby="capi-map">
				<h2 id="capi-map" class={kicker}>Tahap funnel → event Meta</h2>
				<ol class="mt-3 space-y-2 text-sm">
					{#each s.mapping as m, i}
						<li class="flex items-baseline justify-between gap-3">
							<span class="text-neutral-700 dark:text-neutral-300"><span class="mr-2 font-mono text-[11px] text-neutral-400">{String(i + 1).padStart(2, "0")}</span>{tahap(m.tahap)}</span>
							<span class="font-mono text-xs font-medium text-neutral-900 dark:text-white">{m.event}</span>
						</li>
					{/each}
				</ol>
				<p class="mt-3 text-xs leading-relaxed text-neutral-500 dark:text-neutral-400">Event dikirim sekali per lead (event_id unik). Meta hanya menerima event maksimal 7 hari ke belakang.</p>
			</section>
		</div>

		<section class={panel + " overflow-hidden"} aria-labelledby="capi-events">
			<div class="px-5 pt-5 sm:px-6">
				<h2 id="capi-events" class={kicker}>50 event terakhir</h2>
			</div>
			{#if s.events.length === 0}
				<p class="px-5 py-10 text-sm text-neutral-500 sm:px-6 dark:text-neutral-400">Belum ada event. Event muncul otomatis setiap 2 menit setelah CAPI aktif.</p>
			{:else}
				<div class="mt-3 overflow-x-auto">
					<table class="w-full min-w-[720px] text-sm">
						<thead>
							<tr class="border-y border-neutral-100 bg-neutral-50/60 text-left text-[11px] font-medium text-neutral-500 dark:border-white/[0.05] dark:bg-white/[0.015] dark:text-neutral-400">
								<th scope="col" class="py-2 pl-5 pr-3 font-medium sm:pl-6">Lead</th>
								<th scope="col" class="px-3 py-2 font-medium">Event</th>
								<th scope="col" class="px-3 py-2 font-medium">Waktu event</th>
								<th scope="col" class="px-3 py-2 text-right font-medium">Nilai</th>
								<th scope="col" class="py-2 pl-3 pr-5 font-medium sm:pr-6">Status</th>
							</tr>
						</thead>
						<tbody class="divide-y divide-neutral-100 dark:divide-white/[0.05]">
							{#each s.events as e (e.id)}
								<tr class="align-top">
									<td class="py-2.5 pl-5 pr-3 sm:pl-6"><a href={`/app/leads/${e.lead_id}`} use:inertia class="font-medium text-neutral-900 hover:text-brand-700 dark:text-white dark:hover:text-brand-400">{e.lead_nama}</a></td>
									<td class="px-3 py-2.5 font-mono text-xs text-neutral-800 dark:text-neutral-200">{e.event}{#if e.test_mode}<span class="ml-1 rounded bg-amber-500/10 px-1 font-sans text-[10px] text-amber-700 dark:text-amber-400">uji</span>{/if}</td>
									<td class="px-3 py-2.5 tabular-nums text-neutral-600 dark:text-neutral-300">{formatDateTime(e.event_time)}</td>
									<td class="px-3 py-2.5 text-right tabular-nums text-neutral-700 dark:text-neutral-300">{e.event === "Purchase" ? rupiah(e.value) : "—"}</td>
									<td class="py-2.5 pl-3 pr-5 sm:pr-6">
										<span class={"inline-block rounded-full px-2 py-0.5 text-xs font-medium " + (STATUS[e.status]?.cls ?? "")}>{STATUS[e.status]?.label ?? e.status}</span>
										{#if e.attempts > 1}<span class="ml-1 text-[11px] text-neutral-400">{e.attempts}× coba</span>{/if}
										{#if e.last_error}<p class="mt-1 max-w-xs break-words text-[11px] leading-snug text-neutral-500 dark:text-neutral-400">{e.last_error}</p>{/if}
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{/if}
		</section>
	</div>
</AppLayout>
