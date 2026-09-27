<script lang="ts">
	import { inertia } from "@inertiajs/svelte";
	import AppLayout from "@layouts/AppLayout.svelte";
	import type { Tagihan, TagihanFollowUpLog, User } from "@lib/types";
	import { ArrowLeft, MessageCircle } from "lucide-svelte";

	interface Props {
		user?: User;
		tagihan: Tagihan;
		follow_up_logs?: TagihanFollowUpLog[];
	}

	let props: Props = $props();
	let logs = $derived(props.follow_up_logs ?? []);
	const rupiah = (n: number) =>
		new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 0 }).format(n);
	const waktu = (value: string) => value ? new Date(value).toLocaleString("id-ID") : "-";
</script>

<AppLayout user={props.user}>
	<div class="max-w-3xl mx-auto px-4 sm:px-6 pt-6 text-sm text-neutral-600 dark:text-neutral-400">
		Angkatan Pendaftaran: <strong>{props.tagihan.angkatan || "-"}</strong> · Angkatan Kelas: <strong>{props.tagihan.angkatan_kelas || "-"}</strong>
	</div>

	<main class="max-w-3xl mx-auto px-4 sm:px-6 py-8 space-y-5">
		<a href="/app/keuangan/tagihan" use:inertia class="inline-flex items-center gap-2 text-sm text-neutral-500 hover:text-brand-600"><ArrowLeft size="16" /> Kembali ke daftar</a>

		<section class="rounded-2xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-925 overflow-hidden">
			<header class="p-6 border-b border-neutral-200 dark:border-neutral-800">
				<p class="text-xs font-bold tracking-[0.15em] text-brand-600 uppercase">Tagihan SPP</p>
				<h1 class="mt-2 text-2xl font-bold text-neutral-900 dark:text-white">{props.tagihan.santri_nama}</h1>
				<p class="mt-1 text-sm text-neutral-500">{props.tagihan.id_mahasantri || "Tanpa ID"} · {props.tagihan.kelas_nama || "Tanpa kelas"}</p>
			</header>

			<dl class="grid sm:grid-cols-2 divide-y sm:divide-y-0 sm:divide-x divide-neutral-200 dark:divide-neutral-800">
				<div class="p-5 space-y-3">
					<div>
						<dt class="text-xs text-neutral-500">Nominal</dt>
						<dd class="font-mono font-bold text-lg text-neutral-900 dark:text-white">{rupiah(props.tagihan.nominal)}</dd>
						{#if props.tagihan.nominal_override}<span class="inline-flex mt-1 px-2 py-0.5 rounded-full bg-indigo-500/10 text-indigo-600 dark:text-indigo-400 text-xs font-semibold">Nominal khusus</span>{/if}
					</div>
					<div><dt class="text-xs text-neutral-500">Periode</dt><dd>Bulan ke-{props.tagihan.bulan_ke} · Pertemuan ke-{props.tagihan.pertemuan_ke}</dd></div>
					<div><dt class="text-xs text-neutral-500">Status</dt><dd class="capitalize">{props.tagihan.status.replace("_", " ")}</dd></div>
				</div>
				<div class="p-5 space-y-3">
					<div><dt class="text-xs text-neutral-500">Tanggal tagih / jatuh tempo</dt><dd>{props.tagihan.tanggal_tagih} / {props.tagihan.jatuh_tempo || "-"}</dd></div>
					<div><dt class="text-xs text-neutral-500">Pembayaran</dt><dd>{props.tagihan.tanggal_bayar || "Belum dibayar"}{#if props.tagihan.metode} · {props.tagihan.metode}{/if}</dd></div>
					<div><dt class="text-xs text-neutral-500">Follow-up WhatsApp</dt><dd class="flex items-center gap-1"><MessageCircle size="15" /> {props.tagihan.fu_count} kali{#if props.tagihan.fu_terakhir} · terakhir {waktu(props.tagihan.fu_terakhir)}{/if}</dd></div>
				</div>
			</dl>

			{#if props.tagihan.catatan}
				<div class="border-t border-neutral-200 dark:border-neutral-800 p-5"><p class="text-xs text-neutral-500">Catatan</p><p class="mt-1 text-sm">{props.tagihan.catatan}</p></div>
			{/if}
		</section>

		<section class="rounded-2xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-925 overflow-hidden">
			<header class="px-5 py-4 border-b border-neutral-200 dark:border-neutral-800">
				<h2 class="font-bold text-neutral-900 dark:text-white">Riwayat Follow-up WhatsApp</h2>
				<p class="mt-1 text-xs text-neutral-500">Mencatat template dan pesan yang dibuka melalui tombol follow-up.</p>
			</header>
			<div class="divide-y divide-neutral-100 dark:divide-neutral-800/70">
				{#each logs as log (log.id)}
					<div class="p-5">
						<div class="flex flex-col sm:flex-row sm:items-center justify-between gap-1">
							<p class="text-sm font-semibold text-neutral-900 dark:text-white">{log.template_nama || "Custom"}</p>
							<p class="text-xs text-neutral-500">{waktu(log.created_at)}</p>
						</div>
						<p class="mt-1 text-xs text-neutral-500">Petugas: {log.petugas_nama || "-"}</p>
						<p class="mt-3 text-sm text-neutral-600 dark:text-neutral-300 whitespace-pre-wrap">{log.message_body}</p>
					</div>
				{:else}
					<div class="p-10 text-center text-sm text-neutral-500">Belum ada follow-up yang tercatat.</div>
				{/each}
			</div>
		</section>
	</main>
</AppLayout>
