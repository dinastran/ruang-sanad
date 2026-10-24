<script lang="ts">
	import { inertia, router } from "@inertiajs/svelte";
	import AppLayout from "@layouts/AppLayout.svelte";
	import type { Flash, User } from "@lib/types";
	import { Toast } from "@lib/notifications/toast";
	import { SENDER_LABELS, STAGE_LABELS, formatDateTime, relativeTime, toWIBInput } from "@lib/leads";
	import { AlertTriangle, CheckCheck, Clock, RotateCcw, Send } from "lucide-svelte";

	interface Message {
		id: number;
		direction: string;
		sender: string;
		body: string;
		media_type: string;
		sent_at: string;
	}
	interface Item {
		id: number;
		lead_id: number;
		lead_name: string;
		lead_stage: string;
		step: string;
		scheduled_at: string;
		draft_text: string;
		reason: string;
		status: string;
		created_by_kind: string;
		error: string;
		outbox_status: string;
		warnings: string[];
		blocked: boolean;
		recent_messages: Message[];
	}
	interface Failed {
		outbox_id: number;
		lead_id: number;
		lead_name: string;
		purpose: string;
		step: string;
		body: string;
		error: string;
		attempts: number;
		updated_at: string;
	}
	interface Page {
		pending: Item[];
		approved: Item[];
		failed: Failed[];
		followup_enabled: boolean;
		send_start: string;
		send_end: string;
		daily_cap: number;
		sent_today: number;
		only_label: string;
		can_approve: boolean;
	}
	interface Props {
		user?: User;
		page: Page;
		flash?: Flash;
	}

	let props: Props = $props();
	let page = $derived(props.page);

	// Local edits per follow-up; untouched items fall back to the server value.
	let drafts = $state<Record<number, string>>({});
	let times = $state<Record<number, string>>({});
	let selected = $state<Record<number, boolean>>({});
	let busy = $state(false);

	const draftOf = (it: Item) => drafts[it.id] ?? it.draft_text;
	const timeOf = (it: Item) => times[it.id] ?? toWIBInput(it.scheduled_at);
	const edited = (it: Item) => draftOf(it).trim() !== it.draft_text || timeOf(it) !== toWIBInput(it.scheduled_at);
	const bulkable = (it: Item) => it.warnings.length === 0 && !it.blocked;

	let selectedIds = $derived(page.pending.filter((it) => selected[it.id] && bulkable(it)).map((it) => it.id));

	function act(method: "put" | "post", url: string, data: Record<string, string | number | boolean | number[]> = {}, onDone?: () => void) {
		busy = true;
		router[method](url, data, {
			preserveScroll: true,
			onSuccess: (p) => {
				const flash = p.props.flash as Flash | undefined;
				if (flash?.error) Toast(flash.error, "error");
				else if (flash?.success) Toast(flash.success, "success");
				onDone?.();
			},
			onFinish: () => (busy = false),
		});
	}
	function reset(id: number) {
		delete drafts[id];
		delete times[id];
		delete selected[id];
	}
	function approve(it: Item) {
		const data: Record<string, string> = edited(it) ? { draft_text: draftOf(it), scheduled_at: timeOf(it) } : {};
		act("put", `/app/leads/followups/${it.id}/approve`, data, () => reset(it.id));
	}
	function reschedule(it: Item) {
		act("put", `/app/leads/followups/${it.id}/schedule`, { scheduled_at: timeOf(it) }, () => reset(it.id));
	}
	function cancel(it: Item) {
		if (!confirm(`Batalkan follow-up ${it.step.toUpperCase()} untuk ${it.lead_name}?`)) return;
		act("post", `/app/leads/followups/${it.id}/cancel`, {}, () => reset(it.id));
	}
	function bulkApprove() {
		if (selectedIds.length === 0) return;
		act("post", "/app/leads/followups/bulk-approve", { ids: selectedIds }, () => (selected = {}));
	}
	function retry(f: Failed) {
		act("post", `/app/leads/outbox/${f.outbox_id}/retry`);
	}

	const card = "rounded-xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-4";
	const field = "w-full px-3 py-2 rounded-lg bg-white dark:bg-neutral-900/60 border border-neutral-300 dark:border-neutral-700/80 text-sm text-neutral-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-brand-400/30";
	const btn = "inline-flex items-center justify-center gap-1.5 px-3 py-2 rounded-lg text-sm font-medium transition-colors disabled:opacity-50 whitespace-nowrap";
	const btnGhost = `${btn} border border-neutral-300 dark:border-neutral-700/80 text-neutral-700 dark:text-neutral-300 hover:bg-neutral-100 dark:hover:bg-neutral-800`;
	const btnPrimary = `${btn} bg-brand-600 hover:bg-brand-700 text-white dark:bg-brand-500 dark:hover:bg-brand-400`;
</script>

<AppLayout user={props.user}>
	<div class="max-w-4xl mx-auto px-4 sm:px-6 py-6 space-y-5">
		<div>
			<h1 class="text-2xl sm:text-3xl font-bold text-neutral-900 dark:text-white tracking-tight">Follow-up</h1>
			<p class="mt-1 text-sm text-neutral-600 dark:text-neutral-400">Usulan follow-up dari AI dan CS. Tidak ada yang terkirim sebelum disetujui; pengiriman lewat nomor CS dengan jeda dan batas harian.</p>
		</div>

		<section class="{card} flex flex-wrap items-center gap-x-6 gap-y-2 text-sm">
			<span class="inline-flex items-center gap-1.5 font-medium {page.followup_enabled ? 'text-success' : 'text-warning'}">
				<Send class="w-4 h-4" />{page.followup_enabled ? "Pengiriman aktif" : "Pengiriman dimatikan"}
			</span>
			<span class="inline-flex items-center gap-1.5 text-neutral-600 dark:text-neutral-400"><Clock class="w-4 h-4" />{page.send_start}–{page.send_end} WIB</span>
			<span class="text-neutral-600 dark:text-neutral-400">Terkirim hari ini {page.sent_today}/{page.daily_cap}</span>
			{#if page.only_label}<span class="text-neutral-600 dark:text-neutral-400">Uji bertahap: hanya label <b>{page.only_label}</b></span>{/if}
			{#if !page.followup_enabled}<p class="w-full text-xs text-neutral-500">Follow-up yang disetujui akan menunggu sampai super admin menyalakan pengiriman di Integrasi WhatsApp.</p>{/if}
		</section>

		{#if page.failed.length > 0}
			<section class="{card} border-error/30">
				<h2 class="inline-flex items-center gap-1.5 text-sm font-semibold text-error"><AlertTriangle class="w-4 h-4" /> Gagal kirim ({page.failed.length})</h2>
				<ul class="mt-2 divide-y divide-neutral-200/80 dark:divide-white/[0.05]">
					{#each page.failed as f (f.outbox_id)}
						<li class="py-3 text-sm">
							<div class="flex flex-wrap items-start justify-between gap-2">
								<div class="min-w-0">
									<a href={`/app/leads/${f.lead_id}`} use:inertia class="font-medium text-neutral-900 dark:text-white hover:text-brand-600">{f.lead_name}</a>
									<span class="ml-1 text-xs text-neutral-500">{f.purpose === "followup" ? f.step.toUpperCase() : "Konfirmasi pembayaran"} · {f.attempts}× percobaan · {relativeTime(f.updated_at)}</span>
								</div>
								{#if page.can_approve}<button type="button" class="{btnGhost} !py-1.5 text-xs" disabled={busy} onclick={() => retry(f)}><RotateCcw class="w-3.5 h-3.5" /> Coba lagi</button>{/if}
							</div>
							<p class="mt-1 text-xs text-error">{f.error}</p>
							<p class="mt-1 text-neutral-600 dark:text-neutral-400 whitespace-pre-wrap break-words line-clamp-3">{f.body}</p>
						</li>
					{/each}
				</ul>
			</section>
		{/if}

		<section class="space-y-3">
			<div class="flex flex-wrap items-center justify-between gap-2">
				<h2 class="text-sm font-semibold text-neutral-900 dark:text-white">Menunggu persetujuan ({page.pending.length})</h2>
				{#if page.can_approve && page.pending.some(bulkable)}
					<button type="button" class={btnPrimary} disabled={busy || selectedIds.length === 0} onclick={bulkApprove}><CheckCheck class="w-4 h-4" /> Setujui terpilih ({selectedIds.length})</button>
				{/if}
			</div>
			{#if page.pending.length === 0}
				<div class="{card} text-sm text-neutral-500">Tidak ada usulan yang menunggu.</div>
			{/if}
			{#each page.pending as it (it.id)}
				<article class="{card} space-y-3 {it.blocked ? 'opacity-80' : ''}">
					<div class="flex flex-wrap items-start justify-between gap-2">
						<label class="flex min-w-0 items-start gap-2">
							{#if page.can_approve && bulkable(it)}<input type="checkbox" class="mt-1" bind:checked={selected[it.id]} aria-label="Pilih untuk disetujui" />{/if}
							<span class="min-w-0">
								<a href={`/app/leads/${it.lead_id}`} use:inertia class="font-semibold text-neutral-900 dark:text-white hover:text-brand-600 break-words">{it.lead_name}</a>
								<span class="block text-xs text-neutral-500">{STAGE_LABELS[it.lead_stage] ?? it.lead_stage} · {it.created_by_kind === "ai" ? "usulan AI" : "dibuat CS"}</span>
							</span>
						</label>
						<span class="rounded-full bg-neutral-100 dark:bg-neutral-800 px-2 py-0.5 text-xs font-semibold uppercase text-neutral-700 dark:text-neutral-300">{it.step}</span>
					</div>

					{#if it.warnings.length > 0}
						<ul class="space-y-1">
							{#each it.warnings as w}<li class="flex items-start gap-1.5 text-xs font-medium text-error"><AlertTriangle class="w-3.5 h-3.5 shrink-0 mt-px" />{w}</li>{/each}
						</ul>
					{/if}
					{#if it.reason}<p class="text-xs text-neutral-600 dark:text-neutral-400"><span class="font-medium">Alasan:</span> {it.reason}</p>{/if}

					{#if it.recent_messages.length > 0}
						<div class="rounded-lg bg-neutral-50 dark:bg-neutral-900/60 p-2 space-y-1">
							{#each it.recent_messages as m (m.id)}
								<p class="text-xs text-neutral-700 dark:text-neutral-300 break-words"><span class="font-medium">{m.direction === "in" ? "Lead" : (SENDER_LABELS[m.sender] ?? m.sender)}:</span> {m.body || m.media_type} <span class="text-neutral-400">· {relativeTime(m.sent_at)}</span></p>
							{/each}
						</div>
					{/if}

					<label class="block text-xs font-medium text-neutral-600 dark:text-neutral-400">Pesan <span class="font-normal">({draftOf(it).length}/1000)</span>
						<textarea rows="4" maxlength="1000" class="{field} mt-1" value={draftOf(it)} disabled={!page.can_approve || it.blocked} oninput={(e) => (drafts[it.id] = (e.currentTarget as HTMLTextAreaElement).value)}></textarea>
					</label>
					<label class="block text-xs font-medium text-neutral-600 dark:text-neutral-400 max-w-xs">Jadwal kirim (WIB)
						<input type="datetime-local" class="{field} mt-1" value={timeOf(it)} disabled={!page.can_approve || it.blocked} oninput={(e) => (times[it.id] = (e.currentTarget as HTMLInputElement).value)} />
					</label>

					{#if page.can_approve}
						<div class="flex flex-wrap gap-2">
							<button type="button" class={btnPrimary} disabled={busy || it.blocked || draftOf(it).trim() === ""} onclick={() => approve(it)}>{edited(it) ? "Edit & setujui" : "Setujui"}</button>
							{#if timeOf(it) !== toWIBInput(it.scheduled_at) && draftOf(it).trim() === it.draft_text}
								<button type="button" class={btnGhost} disabled={busy || it.blocked} onclick={() => reschedule(it)}>Ubah jadwal saja</button>
							{/if}
							<button type="button" class={btnGhost} disabled={busy} onclick={() => cancel(it)}>Batalkan</button>
						</div>
					{/if}
				</article>
			{/each}
		</section>

		<section class={card}>
			<h2 class="text-sm font-semibold text-neutral-900 dark:text-white">Terjadwal ({page.approved.length})</h2>
			{#if page.approved.length === 0}
				<p class="mt-2 text-sm text-neutral-500">Belum ada follow-up yang disetujui.</p>
			{:else}
				<ul class="mt-2 divide-y divide-neutral-200/80 dark:divide-white/[0.05]">
					{#each page.approved as it (it.id)}
						<li class="py-3 text-sm space-y-2">
							<div class="flex flex-wrap items-center justify-between gap-2">
								<a href={`/app/leads/${it.lead_id}`} use:inertia class="font-medium text-neutral-900 dark:text-white hover:text-brand-600">{it.lead_name}</a>
								<span class="text-xs text-neutral-500"><span class="font-semibold uppercase">{it.step}</span> · {formatDateTime(it.scheduled_at)} WIB{it.outbox_status === "queued" ? " · di antrean" : it.outbox_status === "sending" ? " · sedang dikirim" : ""}</span>
							</div>
							<p class="text-neutral-600 dark:text-neutral-400 whitespace-pre-wrap break-words line-clamp-3">{it.draft_text}</p>
							{#if it.error}<p class="text-xs text-warning">{it.error}</p>{/if}
							{#if page.can_approve}
								<div class="flex flex-wrap items-end gap-2">
									<input type="datetime-local" class="{field} max-w-[14rem]" aria-label="Jadwal baru (WIB)" value={timeOf(it)} oninput={(e) => (times[it.id] = (e.currentTarget as HTMLInputElement).value)} />
									<button type="button" class={btnGhost} disabled={busy || timeOf(it) === toWIBInput(it.scheduled_at)} onclick={() => reschedule(it)}>Ubah jadwal</button>
									<button type="button" class={btnGhost} disabled={busy} onclick={() => cancel(it)}>Batalkan</button>
								</div>
							{/if}
						</li>
					{/each}
				</ul>
			{/if}
		</section>
	</div>
</AppLayout>
