<script lang="ts">
	import { inertia, router } from "@inertiajs/svelte";
	import { fly } from "svelte/transition";
	import AppLayout from "@layouts/AppLayout.svelte";
	import type { Flash, User } from "@lib/types";
	import { Toast } from "@lib/notifications/toast";
	import { getCSRFToken } from "@lib/utils/csrf";
	import {
		STAGE_LABELS,
		STATUS_LABELS,
		LOST_REASON_LABELS,
		SENDER_LABELS,
		UNTUK_SIAPA_LABELS,
		FIELD_LABELS,
		stageTone,
		leadName,
		formatRupiah,
		relativeTime,
		formatDateTime,
		todayWIB,
		waLink,
		labelTone,
		type LeadLabel,
	} from "@lib/leads";
	import { ArrowLeft, Copy, ExternalLink, AlertTriangle, PauseCircle, Sparkles, X } from "lucide-svelte";

	interface Message {
		id: number;
		direction: string;
		sender: string;
		body: string;
		media_type: string;
		sent_at: string;
	}
	interface LeadEvent {
		id: number;
		type: string;
		actor: string;
		actor_name: string;
		payload: Record<string, unknown>;
		created_at: string;
	}
	interface Analysis {
		summary: string;
		persona: Record<string, unknown>;
		interest_score: number;
		main_objection: string;
		suggested_stage: string;
		suggested_next_action: string;
		confidence: string;
		created_at: string;
	}
	interface Followup {
		id: number;
		step: string;
		scheduled_at: string;
		draft_text: string;
		status: string;
		sent_at: string;
		error: string;
	}
	interface Lead {
		id: number;
		phone: string;
		wa_name: string;
		nama: string;
		usia: number;
		domisili: string;
		jenis_kelamin: string;
		paket_kode: string;
		kelas_kode: string;
		waktu_belajar: string;
		riwayat_belajar: string;
		belajar_dimana: string;
		untuk_siapa: string;
		stage: string;
		status: string;
		lost_reason: string;
		followup_permission: string;
		assigned_user_id: number;
		assigned_name: string;
		payment_amount: number;
		payment_info_sent_at: string;
		proof_received_at: string;
		verified_at: string;
		verified_amount: number;
		verified_date: string;
		santri_id: number;
		needs_review: boolean;
		review_fields: string[];
		ai_paused: boolean;
		ai_paused_until: string;
		contacted_again: boolean;
		last_inbound_at: string;
		created_at: string;
		labels: LeadLabel[];
		messages: Message[];
		has_older_messages: boolean;
		events: LeadEvent[];
		analysis: Analysis | null;
		followups: Followup[];
	}
	interface Duplicate {
		id: number;
		nama: string;
		id_mahasantri: string;
		no_wa: string;
	}
	interface ConvertOptions {
		angkatan: { kode: string; keterangan: string }[];
		kode_kelas: string[];
		duplicates: Duplicate[];
	}
	interface Props {
		user?: User;
		lead: Lead;
		labels?: LeadLabel[];
		users?: { id: number; name: string; role: string }[];
		stages?: string[];
		lost_reasons?: string[];
		convert?: ConvertOptions;
		flash?: Flash;
	}

	let props: Props = $props();
	let user = $derived(props.user);
	let lead = $derived(props.lead);
	let allLabels = $derived(props.labels ?? []);
	let stages = $derived(props.stages ?? []);
	let success = $derived(props.flash?.success);
	let error = $derived(props.flash?.error);

	let isSuperAdmin = $derived(user?.role === "super_admin");
	// Keuangan may convert leads but cannot open Data Santri pages.
	let canViewSantri = $derived(user?.role === "cs" || isSuperAdmin);
	let isOpen = $derived(lead.status === "open");
	let isWon = $derived(lead.status === "won");
	let canVerify = $derived(isOpen && (lead.stage === "info_bayar_dikirim" || lead.stage === "bukti_masuk"));
	let canConvert = $derived(isOpen && lead.stage === "terverifikasi" && !lead.santri_id && !!props.convert);
	let unusedLabels = $derived(allLabels.filter((l) => !lead.labels.some((x) => x.id === l.id)));

	// --- Transcript: older pages are fetched and kept locally. ---
	let older = $state<Message[]>([]);
	let olderMore = $state<boolean | null>(null);
	let loadingOlder = $state(false);
	let messages = $derived([...older, ...lead.messages]);
	let hasOlder = $derived(olderMore ?? lead.has_older_messages);

	async function loadOlder() {
		const first = messages[0];
		if (!first || loadingOlder) return;
		loadingOlder = true;
		try {
			const res = await fetch(`/app/leads/${lead.id}/messages?before=${first.id}`, {
				headers: { Accept: "application/json", "X-XSRF-TOKEN": getCSRFToken() },
			});
			if (!res.ok) throw new Error();
			const data = (await res.json()) as { messages: Message[]; has_more: boolean };
			older = [...data.messages, ...older];
			olderMore = data.has_more;
		} catch {
			Toast("Gagal memuat pesan lama", "error");
		} finally {
			loadingOlder = false;
		}
	}

	function mediaLabel(type: string): string {
		const map: Record<string, string> = { gambar: "📷 gambar", dokumen: "📄 dokumen", audio: "🎤 audio", video: "🎬 video", stiker: "stiker" };
		return map[type] ?? `📎 ${type}`;
	}

	// --- Tabs on small screens; both columns on desktop. ---
	let tab = $state<"chat" | "data" | "timeline">("chat");

	// --- Generic action helper: Inertia request, then toast flash errors. ---
	let busy = $state(false);
	function send(method: "put" | "post" | "delete", url: string, data: Record<string, unknown> = {}, onDone?: () => void) {
		busy = true;
		// preserveState keeps dialog inputs when the server flashes an error.
		const opts = {
			preserveScroll: true,
			preserveState: true,
			onSuccess: (p: { props: Record<string, unknown> }) => {
				const err = (p.props.flash as Flash | undefined)?.error;
				if (err) {
					if (dialog === "convert" && err.includes("sudah terdaftar")) serverDuplicate = true;
					Toast(err, "error");
					return;
				}
				onDone?.();
			},
			onFinish: () => {
				busy = false;
			},
		};
		if (method === "delete") router.delete(url, opts);
		else router[method](url, data, opts);
	}

	function copyPhone() {
		navigator.clipboard?.writeText(lead.phone).then(
			() => Toast("Nomor disalin", "success"),
			() => Toast("Gagal menyalin", "error"),
		);
	}

	// --- Data form ---
	// svelte-ignore state_referenced_locally
	let form = $state({
		nama: lead.nama,
		usia: lead.usia || 0,
		domisili: lead.domisili,
		jenis_kelamin: lead.jenis_kelamin,
		paket_kode: lead.paket_kode,
		kelas_kode: lead.kelas_kode,
		waktu_belajar: lead.waktu_belajar,
		riwayat_belajar: lead.riwayat_belajar,
		belajar_dimana: lead.belajar_dimana,
		untuk_siapa: lead.untuk_siapa,
		mark_reviewed: false,
	});
	function saveData(e: Event) {
		e.preventDefault();
		send("put", `/app/leads/${lead.id}`, { ...form, usia: Number(form.usia) || 0 }, () => (form.mark_reviewed = false));
	}
	let review = $derived(new Set(lead.review_fields));

	// --- Dialogs ---
	type Dialog = "stage" | "lost" | "verify" | "convert" | "delete" | null;
	let dialog = $state<Dialog>(null);

	// svelte-ignore state_referenced_locally
	let stageTarget = $state(lead.stage);
	let lostReason = $state("");

	let verify = $state({ amount: 0, date: "", note: "", no_proof_confirmed: false });
	function openVerify() {
		verify = { amount: lead.payment_amount || 0, date: todayWIB(), note: "", no_proof_confirmed: false };
		dialog = "verify";
	}
	let amountDiffers = $derived(lead.payment_amount > 0 && Number(verify.amount) !== lead.payment_amount);
	let verifyValid = $derived(
		Number(verify.amount) > 0 && !!verify.date && (!amountDiffers || verify.note.trim() !== "") && (lead.stage !== "info_bayar_dikirim" || verify.no_proof_confirmed),
	);

	function convertDefaults() {
		return {
			nama: lead.nama,
			jenis_kelamin: lead.jenis_kelamin,
			usia: lead.usia || 0,
			domisili: lead.domisili,
			no_wa: lead.phone,
			kelas_kode: lead.kelas_kode,
			nominal: lead.verified_amount || lead.payment_amount || 0,
			tanggal_daftar: lead.verified_date || todayWIB(),
			angkatan: props.convert?.angkatan[0]?.kode ?? "",
			force_new: false,
		};
	}
	// svelte-ignore state_referenced_locally
	let convert = $state(convertDefaults());
	function openConvert() {
		convert = convertDefaults();
		serverDuplicate = false;
		dialog = "convert";
	}
	let duplicates = $derived(props.convert?.duplicates ?? []);
	// Set when the server finds a duplicate for an edited WhatsApp number.
	let serverDuplicate = $state(false);
	let convertValid = $derived(
		convert.nama.trim() !== "" && (convert.jenis_kelamin === "L" || convert.jenis_kelamin === "P") && convert.angkatan !== "" && !!convert.tanggal_daftar && (duplicates.length === 0 && !serverDuplicate || convert.force_new),
	);
	function submitConvert() {
		send("post", `/app/leads/${lead.id}/convert`, { ...convert, usia: Number(convert.usia) || 0, nominal: Number(convert.nominal) || 0 });
	}
	function linkTo(santriId: number) {
		send("post", `/app/leads/${lead.id}/convert`, { link_santri_id: santriId });
	}

	// --- Timeline ---
	const EVENT_LABELS: Record<string, string> = {
		"message.in": "Pesan masuk",
		"message.out": "Pesan keluar",
		"form.sent": "Form dikirim",
		"form.parsed": "Form terbaca",
		"form.unreadable": "Form tidak terbaca, perlu dicek",
		"payment_info.sent": "Info pembayaran dikirim",
		"proof.received": "Bukti transfer masuk",
		handoff: "Diteruskan ke admin",
		"ai.paused": "AI dijeda",
		"ai.resumed": "AI aktif lagi",
		"optout.detected": "Lead minta berhenti dihubungi",
		"stage.changed": "Tahap diubah",
		"stage.ai_suggestion_applied": "Saran tahap AI diterapkan",
		"payment.verified": "Pembayaran diverifikasi",
		"lead.converted": "Dijadikan santri",
		"lead.lost": "Ditandai hilang",
		"lead.do_not_contact": "Ditandai jangan hubungi",
		"lead.reopened": "Dibuka kembali",
		"lead.assigned": "Penanggung jawab diubah",
		"lead.updated": "Data diperbarui",
		"label.added": "Label ditambah",
		"label.removed": "Label dihapus",
		"analysis.saved": "Analisis AI disimpan",
		"followup.proposed": "Follow-up diusulkan",
	};
	// Message events are already visible in the transcript.
	let timeline = $derived(lead.events.filter((e) => e.type !== "message.in" && e.type !== "message.out"));

	function eventDetail(e: LeadEvent): string {
		const p = e.payload ?? {};
		const parts: string[] = [];
		if (p.stage_from && p.stage_to) parts.push(`${STAGE_LABELS[p.stage_from as string] ?? p.stage_from} → ${STAGE_LABELS[p.stage_to as string] ?? p.stage_to}`);
		if (p.reason && e.type === "lead.lost") parts.push(LOST_REASON_LABELS[p.reason as string] ?? String(p.reason));
		else if (p.reason) parts.push(String(p.reason));
		if (p.label) parts.push(String(p.label));
		if (e.type === "analysis.saved" && p.interest_score !== undefined) parts.push(`skor minat ${p.interest_score}`);
		if (e.type === "followup.proposed" && p.step) parts.push(`${String(p.step).toUpperCase()} · ${p.scheduled_at}`);
		if (e.type === "payment.verified") parts.push(`${formatRupiah(Number(p.amount))} · ${p.date}${p.note ? ` · ${p.note}` : ""}`);
		if (e.type === "payment_info.sent" && p.nominal) parts.push(`Paket ${p.paket_kode} · ${formatRupiah(Number(p.nominal))}`);
		if (Array.isArray(p.fields) && p.fields.length) parts.push((p.fields as string[]).map((f) => FIELD_LABELS[f] ?? f).join(", "));
		if (e.type === "message.in" && p.contacted_again) parts.push("menghubungi lagi");
		return parts.join(" · ");
	}
	// Persona values are snake_case codes or lists of them.
	function personaText(v: unknown): string {
		if (Array.isArray(v)) return v.length ? v.map((x) => String(x).replaceAll("_", " ")).join(", ") : "-";
		if (v === undefined || v === null || v === "") return "-";
		return String(v).replaceAll("_", " ");
	}
	function actorLabel(e: LeadEvent): string {
		if (e.actor === "bridge") return "CS AI";
		if (e.actor === "system") return "Sistem";
		if (e.actor.startsWith("ai:")) return "Agen AI";
		return e.actor_name || "User";
	}

	const field = "w-full px-3 py-2 rounded-lg bg-white dark:bg-neutral-900/60 border border-neutral-300 dark:border-neutral-700/80 text-sm text-neutral-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-brand-400/30 disabled:opacity-60";
	const labelCls = "block text-xs font-medium text-neutral-600 dark:text-neutral-400";
	const btn = "inline-flex items-center justify-center gap-1.5 px-3 py-2 rounded-lg text-sm font-medium transition-colors disabled:opacity-50 whitespace-nowrap";
	const btnGhost = `${btn} border border-neutral-300 dark:border-neutral-700/80 text-neutral-700 dark:text-neutral-300 hover:bg-neutral-100 dark:hover:bg-neutral-800`;
	const btnPrimary = `${btn} bg-brand-600 hover:bg-brand-700 text-white dark:bg-brand-500 dark:hover:bg-brand-400`;
</script>

<AppLayout {user}>
	<div class="border-b border-neutral-200/80 dark:border-white/[0.04]">
		<div class="max-w-7xl mx-auto px-4 sm:px-6 pt-6 pb-5">
			<a href="/app/leads" use:inertia class="inline-flex items-center gap-1.5 text-sm text-neutral-500 hover:text-brand-600 dark:hover:text-brand-400">
				<ArrowLeft class="w-4 h-4" /> Leads
			</a>
			<div class="mt-3 flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
				<div class="min-w-0">
					<h1 class="text-2xl font-bold text-neutral-900 dark:text-white tracking-tight break-words">{leadName(lead)}</h1>
					<div class="mt-1.5 flex flex-wrap items-center gap-x-3 gap-y-1.5 text-sm text-neutral-600 dark:text-neutral-400">
						{#if lead.nama && lead.wa_name}<span>WA: {lead.wa_name}</span>{/if}
						{#if lead.phone}
							<span class="inline-flex items-center gap-1 font-mono">
								{lead.phone}
								<button type="button" onclick={copyPhone} class="p-1 rounded hover:bg-neutral-100 dark:hover:bg-neutral-800" aria-label="Salin nomor"><Copy class="w-3.5 h-3.5" /></button>
								<a href={waLink(lead.phone)} target="_blank" rel="noopener noreferrer" class="p-1 rounded hover:bg-neutral-100 dark:hover:bg-neutral-800" aria-label="Buka di WhatsApp"><ExternalLink class="w-3.5 h-3.5" /></a>
							</span>
						{:else}
							<span class="text-xs">Nomor belum diketahui (akun LID)</span>
						{/if}
						<span class="text-xs">Masuk {formatDateTime(lead.created_at, false)}</span>
					</div>
					<div class="mt-2.5 flex flex-wrap items-center gap-2">
						<span class="rounded-full px-2.5 py-0.5 text-xs font-medium {stageTone(lead.stage, lead.status)}">{isOpen ? STAGE_LABELS[lead.stage] : STATUS_LABELS[lead.status]}</span>
						{#if !isOpen && !isWon}<span class="text-xs text-neutral-500">tahap terakhir: {STAGE_LABELS[lead.stage]}{lead.lost_reason ? ` · alasan: ${LOST_REASON_LABELS[lead.lost_reason]}` : ""}</span>{/if}
						{#each lead.labels as l}
							<span class="inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-xs {labelTone(l.warna)}">
								{l.nama}
								<button type="button" disabled={busy} onclick={() => send("delete", `/app/leads/${lead.id}/labels/${l.id}`)} aria-label="Hapus label {l.nama}" class="hover:text-error"><X class="w-3 h-3" /></button>
							</span>
						{/each}
						{#if unusedLabels.length > 0}
							<select class="rounded-full border border-dashed border-neutral-300 dark:border-neutral-700 bg-transparent px-2 py-0.5 text-xs text-neutral-600 dark:text-neutral-400" aria-label="Tambah label" value="" onchange={(e) => { const v = Number((e.currentTarget as HTMLSelectElement).value); if (v) send("post", `/app/leads/${lead.id}/labels`, { label_id: v }); }}>
								<option value="">+ Label</option>
								{#each unusedLabels as l}<option value={l.id}>{l.nama}</option>{/each}
							</select>
						{/if}
					</div>
					<p class="mt-2 text-xs text-neutral-500">
						Penanggung jawab: <span class="font-medium text-neutral-700 dark:text-neutral-300">{lead.assigned_name || "belum ada"}</span>
						{#if isSuperAdmin && props.users}
							<select class="ml-1 rounded border border-neutral-300 dark:border-neutral-700 bg-transparent px-1 py-0.5 text-xs" aria-label="Tugaskan ke" value={String(lead.assigned_user_id || "")} onchange={(e) => send("put", `/app/leads/${lead.id}/assign`, { user_id: Number((e.currentTarget as HTMLSelectElement).value) || 0 })}>
								<option value="">— tidak ada —</option>
								{#each props.users as u}<option value={String(u.id)}>{u.name}</option>{/each}
							</select>
						{/if}
					</p>
				</div>

				<div class="flex flex-wrap gap-2 lg:justify-end lg:max-w-md">
					{#if lead.santri_id && canViewSantri}
						<a href="/app/santri/{lead.santri_id}" use:inertia class={btnPrimary}>Lihat santri</a>
					{/if}
					{#if canConvert}
						<button type="button" class={btnPrimary} onclick={openConvert}>Jadikan Santri</button>
					{/if}
					{#if canVerify}
						<button type="button" class={btnPrimary} onclick={openVerify}>Verifikasi pembayaran</button>
					{/if}
					{#if !isWon && lead.assigned_user_id !== user?.id}
						<button type="button" class={btnGhost} disabled={busy} onclick={() => send("put", `/app/leads/${lead.id}/assign`, { user_id: user?.id })}>Ambil</button>
					{/if}
					{#if isOpen}
						<button type="button" class={btnGhost} onclick={() => { stageTarget = lead.stage; dialog = "stage"; }}>Ubah tahap</button>
						<button type="button" class={btnGhost} onclick={() => (dialog = "lost")}>Tandai hilang</button>
					{/if}
					{#if isOpen || lead.status === "lost"}
						<button type="button" class="{btnGhost} text-error" disabled={busy} onclick={() => { if (confirm("Tandai lead ini 'jangan hubungi'? Semua follow-up dibatalkan.")) send("put", `/app/leads/${lead.id}/status`, { status: "do_not_contact" }); }}>Jangan hubungi</button>
					{/if}
					{#if lead.status === "lost" || lead.status === "do_not_contact"}
						<button type="button" class={btnGhost} disabled={busy} onclick={() => send("put", `/app/leads/${lead.id}/status`, { status: "open" })}>Buka kembali</button>
					{/if}
					{#if isSuperAdmin}
						<button type="button" class="{btnGhost} text-error" onclick={() => (dialog = "delete")}>Hapus data</button>
					{/if}
				</div>
			</div>
		</div>
	</div>

	<div class="max-w-7xl mx-auto px-4 sm:px-6 py-5 space-y-4">
		{#if success}<div class="rounded-xl border border-success/20 bg-success/10 px-4 py-3 text-sm font-medium text-success">{success}</div>{/if}
		{#if error}<div class="rounded-xl border border-error/20 bg-error/10 px-4 py-3 text-sm font-medium text-error">{error}</div>{/if}

		{#if lead.needs_review}
			<div class="flex items-start gap-2 rounded-xl border border-warning/30 bg-warning/10 px-4 py-3 text-sm text-neutral-800 dark:text-neutral-200">
				<AlertTriangle class="w-4 h-4 mt-0.5 shrink-0 text-warning" />
				<p>Data dari form <strong>perlu dicek</strong>. Ada isian yang tidak bisa dibaca otomatis, jadi dibiarkan kosong. Cocokkan dengan chat lalu simpan dengan "Sudah dicek".</p>
			</div>
		{/if}
		{#if lead.ai_paused}
			<div class="flex items-start gap-2 rounded-xl border border-info/30 bg-info/10 px-4 py-3 text-sm text-neutral-800 dark:text-neutral-200">
				<PauseCircle class="w-4 h-4 mt-0.5 shrink-0 text-info" />
				<p>CS AI sedang <strong>dijeda</strong> untuk chat ini{lead.ai_paused_until ? ` sampai ${formatDateTime(lead.ai_paused_until)}` : ""}. Admin sedang menangani dari HP.</p>
			</div>
		{/if}
		{#if lead.contacted_again}
			<div class="rounded-xl border border-error/30 bg-error/10 px-4 py-3 text-sm text-neutral-800 dark:text-neutral-200">Lead ini berstatus <strong>jangan hubungi</strong> tetapi <strong>menghubungi lagi</strong>. Baca chat terbaru dan buka kembali bila memang ingin lanjut.</div>
		{/if}
		{#if lead.followup_permission === "no" && isOpen}
			<p class="text-xs text-neutral-500">Lead pernah meminta berhenti dihubungi: follow-up otomatis tidak akan dikirim.</p>
		{/if}

		<div class="lg:hidden flex rounded-lg border border-neutral-200/80 dark:border-white/[0.06] p-0.5" role="tablist">
			{#each [["chat", "Percakapan"], ["data", "Data"], ["timeline", "Riwayat"]] as [key, label]}
				<button type="button" role="tab" aria-selected={tab === key} onclick={() => (tab = key as typeof tab)} class="flex-1 px-3 py-2 rounded-md text-sm font-medium {tab === key ? 'bg-brand-500/10 text-brand-700 dark:text-brand-300' : 'text-neutral-600 dark:text-neutral-400'}">{label}</button>
			{/each}
		</div>

		<div class="grid gap-5 lg:grid-cols-[minmax(0,1fr)_minmax(0,1.1fr)]">
			<div class="space-y-5 {tab === 'chat' ? 'hidden lg:block' : ''}">
				<div class={tab === "timeline" ? "hidden lg:block space-y-5" : "space-y-5"}>
					{#if lead.analysis}
						{@const a = lead.analysis}
						<section class="rounded-xl border border-brand-500/20 bg-brand-500/[0.04] p-4">
							<div class="flex items-start justify-between gap-3">
								<h2 class="inline-flex items-center gap-1.5 text-sm font-semibold text-neutral-900 dark:text-white"><Sparkles class="w-4 h-4 text-brand-600 dark:text-brand-400" /> Analisis AI</h2>
								<span class="text-xs text-neutral-500">dianalisis {relativeTime(a.created_at)}</span>
							</div>
							<p class="mt-1 text-xs font-medium text-warning">Dugaan AI — periksa sebelum dipakai</p>
							<p class="mt-2 text-sm text-neutral-800 dark:text-neutral-200 whitespace-pre-wrap break-words">{a.summary}</p>
							<dl class="mt-3 grid grid-cols-2 gap-x-4 gap-y-2 text-xs">
								<div><dt class="text-neutral-500">Skor minat</dt><dd class="font-semibold text-neutral-900 dark:text-white">{a.interest_score}/100</dd></div>
								<div><dt class="text-neutral-500">Keyakinan</dt><dd class="font-semibold text-neutral-900 dark:text-white capitalize">{a.confidence}</dd></div>
								<div><dt class="text-neutral-500">Keberatan utama</dt><dd class="text-neutral-800 dark:text-neutral-200">{personaText(a.main_objection)}</dd></div>
								<div><dt class="text-neutral-500">Untuk</dt><dd class="text-neutral-800 dark:text-neutral-200">{personaText(a.persona?.untuk_siapa)}</dd></div>
								<div><dt class="text-neutral-500">Level</dt><dd class="text-neutral-800 dark:text-neutral-200">{personaText(a.persona?.level)}</dd></div>
								<div><dt class="text-neutral-500">Waktu</dt><dd class="text-neutral-800 dark:text-neutral-200">{personaText(a.persona?.preferensi_waktu)}</dd></div>
								<div><dt class="text-neutral-500">Paket diminati</dt><dd class="text-neutral-800 dark:text-neutral-200">{personaText(a.persona?.paket_diminati)}</dd></div>
								<div><dt class="text-neutral-500">Gaya komunikasi</dt><dd class="text-neutral-800 dark:text-neutral-200">{personaText(a.persona?.gaya_komunikasi)}</dd></div>
								<div class="col-span-2"><dt class="text-neutral-500">Tujuan</dt><dd class="text-neutral-800 dark:text-neutral-200">{personaText(a.persona?.tujuan)}</dd></div>
								<div class="col-span-2"><dt class="text-neutral-500">Hambatan</dt><dd class="text-neutral-800 dark:text-neutral-200">{personaText(a.persona?.hambatan)}</dd></div>
							</dl>
							{#if a.persona?.catatan}<p class="mt-2 text-xs text-neutral-600 dark:text-neutral-400 whitespace-pre-wrap break-words">{String(a.persona.catatan)}</p>{/if}
							{#if a.suggested_next_action}<p class="mt-3 text-sm text-neutral-800 dark:text-neutral-200"><span class="text-neutral-500">Saran tindakan:</span> {a.suggested_next_action}</p>{/if}
							{#if a.suggested_stage && isOpen && a.suggested_stage !== lead.stage && a.suggested_stage !== "terverifikasi" && stages.includes(a.suggested_stage)}
								<button type="button" class="{btnGhost} mt-3" disabled={busy} onclick={() => send("put", `/app/leads/${lead.id}/stage?from=ai`, { stage: a.suggested_stage })}>Terapkan saran tahap: {STAGE_LABELS[a.suggested_stage]}</button>
							{/if}
						</section>
					{/if}

					<section class="rounded-xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-4">
						<div class="flex items-center justify-between gap-2">
							<h2 class="text-sm font-semibold text-neutral-900 dark:text-white">Data calon santri</h2>
							{#if lead.payment_amount}<span class="text-xs text-neutral-500">Tagihan {formatRupiah(lead.payment_amount)}</span>{/if}
						</div>
						<form class="mt-3 grid grid-cols-1 sm:grid-cols-2 gap-3" onsubmit={saveData}>
							{#snippet reviewMark(name: string)}
								{#if review.has(name)}<span class="ml-1 rounded bg-warning/15 px-1 text-[10px] font-semibold text-warning">Perlu dicek</span>{/if}
							{/snippet}
							<label class="{labelCls} sm:col-span-2">Nama {@render reviewMark("nama")}<input class="{field} mt-1" bind:value={form.nama} disabled={isWon} maxlength="200" /></label>
							<label class={labelCls}>Ikhwan/Akhwat {@render reviewMark("jenis_kelamin")}
								<select class="{field} mt-1" bind:value={form.jenis_kelamin} disabled={isWon}>
									<option value="">Belum diketahui</option><option value="L">Ikhwan (L)</option><option value="P">Akhwat (P)</option>
								</select>
							</label>
							<label class={labelCls}>Usia {@render reviewMark("usia")}<input type="number" min="0" max="99" class="{field} mt-1" bind:value={form.usia} disabled={isWon} /></label>
							<label class="{labelCls} sm:col-span-2">Domisili<input class="{field} mt-1" bind:value={form.domisili} disabled={isWon} maxlength="200" /></label>
							<label class={labelCls}>Paket {@render reviewMark("paket_kode")}
								<select class="{field} mt-1" bind:value={form.paket_kode} disabled={isWon}>
									<option value="">Belum dipilih</option><option value="1x">1x sepekan</option><option value="2x">2x sepekan</option>
								</select>
							</label>
							<label class={labelCls}>Kode kelas<input class="{field} mt-1" bind:value={form.kelas_kode} disabled={isWon} placeholder="mis. R 1X" /></label>
							<label class={labelCls}>Waktu belajar<input class="{field} mt-1" bind:value={form.waktu_belajar} disabled={isWon} /></label>
							<label class={labelCls}>Untuk siapa
								<select class="{field} mt-1" bind:value={form.untuk_siapa} disabled={isWon}>
									{#each Object.entries(UNTUK_SIAPA_LABELS) as [v, l]}<option value={v}>{l}</option>{/each}
								</select>
							</label>
							<label class={labelCls}>Pernah belajar tahsin<input class="{field} mt-1" bind:value={form.riwayat_belajar} disabled={isWon} /></label>
							<label class={labelCls}>Belajar di mana<input class="{field} mt-1" bind:value={form.belajar_dimana} disabled={isWon} /></label>
							{#if !isWon}
								<div class="sm:col-span-2 flex flex-wrap items-center justify-between gap-3 pt-1">
									{#if lead.needs_review}
										<label class="inline-flex items-center gap-2 text-sm text-neutral-700 dark:text-neutral-300"><input type="checkbox" bind:checked={form.mark_reviewed} /> Sudah dicek</label>
									{:else}<span></span>{/if}
									<button type="submit" class={btnPrimary} disabled={busy}>Simpan data</button>
								</div>
							{/if}
						</form>
					</section>

					{#if lead.followups.length > 0}
						<section class="rounded-xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-4">
							<h2 class="text-sm font-semibold text-neutral-900 dark:text-white">Follow-up</h2>
							<ul class="mt-2 divide-y divide-neutral-200/80 dark:divide-white/[0.05]">
								{#each lead.followups as f}
									<li class="py-2 text-sm">
										<div class="flex items-center justify-between gap-2"><span class="font-medium uppercase text-neutral-800 dark:text-neutral-200">{f.step}</span><span class="text-xs text-neutral-500">{f.status.replaceAll("_", " ")} · {formatDateTime(f.sent_at || f.scheduled_at)}</span></div>
										<p class="mt-1 text-neutral-600 dark:text-neutral-400 whitespace-pre-wrap break-words">{f.draft_text}</p>
									</li>
								{/each}
							</ul>
						</section>
					{/if}
				</div>

				<section class="rounded-xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-4 {tab === 'data' ? 'hidden lg:block' : ''}">
					<h2 class="text-sm font-semibold text-neutral-900 dark:text-white">Riwayat</h2>
					{#if timeline.length === 0}
						<p class="mt-2 text-sm text-neutral-500">Belum ada kejadian.</p>
					{:else}
						<ol class="mt-3 space-y-3 border-l border-neutral-200 dark:border-neutral-800 pl-4">
							{#each timeline as e (e.id)}
								<li class="relative">
									<span class="absolute -left-[21px] top-1.5 h-2 w-2 rounded-full bg-neutral-400"></span>
									<p class="text-sm text-neutral-800 dark:text-neutral-200"><span class="font-medium">{EVENT_LABELS[e.type] ?? e.type}</span>{#if eventDetail(e)}<span class="text-neutral-600 dark:text-neutral-400"> · {eventDetail(e)}</span>{/if}</p>
									<p class="text-xs text-neutral-500">{actorLabel(e)} · {formatDateTime(e.created_at)}</p>
								</li>
							{/each}
						</ol>
					{/if}
				</section>
			</div>

			<section class="rounded-xl border border-neutral-200/80 dark:border-white/[0.06] bg-neutral-50 dark:bg-neutral-900/40 flex flex-col lg:sticky lg:top-4 lg:max-h-[calc(100vh-2rem)] {tab === 'chat' ? '' : 'hidden lg:flex'}">
				<header class="px-4 py-3 border-b border-neutral-200/80 dark:border-white/[0.06]">
					<h2 class="text-sm font-semibold text-neutral-900 dark:text-white">Percakapan</h2>
					<p class="text-xs text-neutral-500">Transkrip chat nomor CS. File media tidak disimpan.</p>
				</header>
				<div class="flex-1 overflow-y-auto px-3 py-3 space-y-2 min-h-64 max-h-[70vh] lg:max-h-none">
					{#if hasOlder}
						<div class="text-center"><button type="button" onclick={loadOlder} disabled={loadingOlder} class="text-xs font-medium text-brand-600 dark:text-brand-400 hover:underline disabled:opacity-50">{loadingOlder ? "Memuat..." : "Muat pesan sebelumnya"}</button></div>
					{/if}
					{#if messages.length === 0}
						<p class="py-10 text-center text-sm text-neutral-500">Belum ada pesan.</p>
					{/if}
					{#each messages as m (m.id)}
						{@const inbound = m.direction === "in"}
						<div class="flex {inbound ? 'justify-start' : 'justify-end'}">
							<div class="max-w-[85%] rounded-2xl px-3 py-2 text-sm {inbound ? 'bg-white dark:bg-neutral-800 text-neutral-900 dark:text-neutral-100 rounded-bl-sm' : m.sender === 'followup' ? 'bg-secondary-500/15 text-neutral-900 dark:text-neutral-100 rounded-br-sm' : 'bg-brand-500/15 text-neutral-900 dark:text-neutral-100 rounded-br-sm'}">
								{#if !inbound}<p class="mb-0.5 text-[10px] font-semibold uppercase tracking-wide text-neutral-500">{SENDER_LABELS[m.sender] ?? m.sender}</p>{/if}
								{#if m.media_type}<p class="text-xs text-neutral-600 dark:text-neutral-400">{mediaLabel(m.media_type)}</p>{/if}
								{#if m.body}<p class="whitespace-pre-wrap break-words">{m.body}</p>{/if}
								<p class="mt-1 text-right text-[10px] text-neutral-500">{formatDateTime(m.sent_at)}</p>
							</div>
						</div>
					{/each}
				</div>
			</section>
		</div>
	</div>
</AppLayout>

{#if dialog}
	<div class="fixed inset-0 z-50 flex items-end sm:items-center justify-center sm:p-4">
		<button class="absolute inset-0 w-full h-full bg-neutral-900/50 backdrop-blur-sm" aria-label="Tutup" onclick={() => (dialog = null)}></button>
		<div class="relative w-full sm:max-w-lg max-h-[92vh] overflow-y-auto bg-white dark:bg-neutral-925 rounded-t-2xl sm:rounded-2xl shadow-xl border border-neutral-200/80 dark:border-white/[0.06] p-5" in:fly={{ y: 20, duration: 200 }} role="dialog" aria-modal="true">
			{#if dialog === "stage"}
				<h3 class="text-lg font-bold text-neutral-900 dark:text-white">Ubah tahap</h3>
				<p class="mt-1 text-sm text-neutral-600 dark:text-neutral-400">Boleh maju atau mundur. Terverifikasi hanya lewat Verifikasi pembayaran.</p>
				<div class="mt-4 grid grid-cols-2 gap-2">
					{#each stages.filter((s) => s !== "terverifikasi") as s}
						<label class="flex cursor-pointer items-center justify-center rounded-lg border px-3 py-2 text-sm {stageTarget === s ? 'border-brand-500 bg-brand-500/10 text-brand-700 dark:text-brand-300' : 'border-neutral-300 dark:border-neutral-700 text-neutral-700 dark:text-neutral-300'}">
							<input type="radio" class="sr-only" bind:group={stageTarget} value={s} />{STAGE_LABELS[s]}
						</label>
					{/each}
				</div>
				<div class="mt-5 flex gap-2"><button class="{btnGhost} flex-1" onclick={() => (dialog = null)}>Batal</button><button class="{btnPrimary} flex-1" disabled={busy || stageTarget === lead.stage} onclick={() => send("put", `/app/leads/${lead.id}/stage`, { stage: stageTarget }, () => (dialog = null))}>Simpan</button></div>
			{:else if dialog === "lost"}
				<h3 class="text-lg font-bold text-neutral-900 dark:text-white">Tandai hilang</h3>
				<p class="mt-1 text-sm text-neutral-600 dark:text-neutral-400">Follow-up yang menunggu akan dibatalkan. Kalau lead chat lagi, lead otomatis dibuka kembali.</p>
				<div class="mt-4 grid grid-cols-2 gap-2">
					{#each props.lost_reasons ?? [] as r}
						<label class="flex cursor-pointer items-center justify-center rounded-lg border px-3 py-2 text-sm {lostReason === r ? 'border-brand-500 bg-brand-500/10 text-brand-700 dark:text-brand-300' : 'border-neutral-300 dark:border-neutral-700 text-neutral-700 dark:text-neutral-300'}">
							<input type="radio" class="sr-only" bind:group={lostReason} value={r} />{LOST_REASON_LABELS[r] ?? r}
						</label>
					{/each}
				</div>
				<div class="mt-5 flex gap-2"><button class="{btnGhost} flex-1" onclick={() => (dialog = null)}>Batal</button><button class="{btnPrimary} flex-1" disabled={busy || !lostReason} onclick={() => send("put", `/app/leads/${lead.id}/status`, { status: "lost", lost_reason: lostReason }, () => (dialog = null))}>Simpan</button></div>
			{:else if dialog === "verify"}
				<h3 class="text-lg font-bold text-neutral-900 dark:text-white">Verifikasi pembayaran</h3>
				<p class="mt-1 text-sm text-neutral-600 dark:text-neutral-400">Cek bukti transfer di WhatsApp CS dan mutasi rekening sebelum memverifikasi.</p>
				<div class="mt-4 space-y-3">
					<label class={labelCls}>Nominal diterima (Rp)<input type="number" min="0" class="{field} mt-1" bind:value={verify.amount} /></label>
					{#if amountDiffers}<p class="rounded-lg bg-warning/10 px-3 py-2 text-xs text-neutral-800 dark:text-neutral-200">Berbeda dari tagihan {formatRupiah(lead.payment_amount)}. Catatan wajib diisi.</p>{/if}
					<label class={labelCls}>Tanggal diterima<input type="date" max={todayWIB()} class="{field} mt-1" bind:value={verify.date} /></label>
					<label class={labelCls}>Catatan {amountDiffers ? "(wajib)" : "(opsional)"}<textarea rows="2" maxlength="500" class="{field} mt-1" bind:value={verify.note}></textarea></label>
					{#if lead.stage === "info_bayar_dikirim"}
						<label class="flex items-start gap-2 rounded-lg bg-warning/10 px-3 py-2 text-sm text-neutral-800 dark:text-neutral-200"><input type="checkbox" class="mt-0.5" bind:checked={verify.no_proof_confirmed} /> Bukti belum tercatat dari WA. Tetap verifikasi?</label>
					{/if}
				</div>
				<div class="mt-5 flex gap-2"><button class="{btnGhost} flex-1" onclick={() => (dialog = null)}>Batal</button><button class="{btnPrimary} flex-1" disabled={busy || !verifyValid} onclick={() => send("post", `/app/leads/${lead.id}/verify`, { ...verify, amount: Number(verify.amount) }, () => (dialog = null))}>Verifikasi</button></div>
			{:else if dialog === "convert" && props.convert}
				<h3 class="text-lg font-bold text-neutral-900 dark:text-white">Jadikan santri</h3>
				<p class="mt-1 text-sm text-neutral-600 dark:text-neutral-400">Data terisi dari form dan bisa diubah. Setelah dibuat, Admin Kelas mendapat notifikasi untuk penempatan kelas.</p>
				{#if duplicates.length > 0}
					<div class="mt-4 rounded-lg border border-warning/30 bg-warning/10 p-3 text-sm text-neutral-800 dark:text-neutral-200">
						<p class="font-medium">Nomor ini sudah terdaftar sebagai santri:</p>
						<ul class="mt-2 space-y-2">
							{#each duplicates as d}
								<li class="flex items-center justify-between gap-2">
									<span>{#if canViewSantri}<a href="/app/santri/{d.id}" use:inertia class="underline">{d.nama}</a>{:else}{d.nama}{/if} <span class="text-xs text-neutral-500">{d.id_mahasantri}</span></span>
									<button type="button" class={btnGhost} disabled={busy} onclick={() => linkTo(d.id)}>Tautkan</button>
								</li>
							{/each}
						</ul>
						<label class="mt-3 flex items-center gap-2"><input type="checkbox" bind:checked={convert.force_new} /> Tetap buat santri baru</label>
					</div>
				{:else if serverDuplicate}
					<div class="mt-4 rounded-lg border border-warning/30 bg-warning/10 p-3 text-sm text-neutral-800 dark:text-neutral-200">
						<p>Nomor WhatsApp yang diisi sudah dipakai santri lain. Ganti nomornya, atau:</p>
						<label class="mt-2 flex items-center gap-2"><input type="checkbox" bind:checked={convert.force_new} /> Tetap buat santri baru</label>
					</div>
				{/if}
				<div class="mt-4 grid grid-cols-1 sm:grid-cols-2 gap-3">
					<label class="{labelCls} sm:col-span-2">Nama *<input class="{field} mt-1" bind:value={convert.nama} /></label>
					<label class={labelCls}>Ikhwan/Akhwat *
						<select class="{field} mt-1" bind:value={convert.jenis_kelamin}><option value="">Pilih</option><option value="L">Ikhwan (L)</option><option value="P">Akhwat (P)</option></select>
					</label>
					<label class={labelCls}>Usia<input type="number" min="0" max="99" class="{field} mt-1" bind:value={convert.usia} /></label>
					<label class="{labelCls} sm:col-span-2">Domisili<input class="{field} mt-1" bind:value={convert.domisili} /></label>
					<label class={labelCls}>No. WhatsApp<input class="{field} mt-1" bind:value={convert.no_wa} /></label>
					<label class={labelCls}>Kode kelas
						<select class="{field} mt-1" bind:value={convert.kelas_kode}><option value="">Belum ditentukan</option>{#each props.convert.kode_kelas as k}<option value={k}>{k}</option>{/each}</select>
					</label>
					<label class={labelCls}>Nominal (Rp)<input type="number" min="0" class="{field} mt-1" bind:value={convert.nominal} /></label>
					<label class={labelCls}>Tanggal daftar *<input type="date" class="{field} mt-1" bind:value={convert.tanggal_daftar} /></label>
					<label class="{labelCls} sm:col-span-2">Angkatan *
						<select class="{field} mt-1" bind:value={convert.angkatan}>
							{#if props.convert.angkatan.length === 0}<option value="">Belum ada angkatan aktif</option>{/if}
							{#each props.convert.angkatan as a}<option value={a.kode}>{a.kode}{a.keterangan ? ` — ${a.keterangan}` : ""}</option>{/each}
						</select>
					</label>
				</div>
				<div class="mt-5 flex gap-2"><button class="{btnGhost} flex-1" onclick={() => (dialog = null)}>Batal</button><button class="{btnPrimary} flex-1" disabled={busy || !convertValid} onclick={submitConvert}>Buat santri</button></div>
			{:else if dialog === "delete"}
				<h3 class="text-lg font-bold text-neutral-900 dark:text-white">Hapus data lead?</h3>
				<p class="mt-1 text-sm text-neutral-600 dark:text-neutral-400">Lead, transkrip, analisis, follow-up, dan riwayatnya dihapus permanen (hak hapus data). Santri yang sudah dibuat tidak ikut terhapus.</p>
				<div class="mt-5 flex gap-2"><button class="{btnGhost} flex-1" onclick={() => (dialog = null)}>Batal</button><button class="{btn} flex-1 bg-error text-white hover:opacity-90" disabled={busy} onclick={() => send("delete", `/app/leads/${lead.id}`)}>Hapus permanen</button></div>
			{/if}
		</div>
	</div>
{/if}
