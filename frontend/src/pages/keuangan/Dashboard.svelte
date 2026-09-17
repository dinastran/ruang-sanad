<script lang="ts">
	import { inertia } from "@inertiajs/svelte";
	import AppLayout from "@layouts/AppLayout.svelte";
	import type { RingkasanKeuangan, Tagihan, User } from "@lib/types";
	import { ArrowRight, CircleAlert, CircleCheck, WalletCards } from "lucide-svelte";

	interface Props { user?: User; ringkasan?: RingkasanKeuangan; belumBayar?: Tagihan[]; lunas?: Tagihan[]; periode?: string; }
	let props: Props = $props();
	let ringkasan = $derived(props.ringkasan ?? { total_tagihan: 0, nominal_tagihan: 0, total_lunas: 0, nominal_lunas: 0, total_belum_bayar: 0, nominal_belum_bayar: 0, total_terlambat: 0, kolektibilitas: 0 });
	let belumBayar = $derived(props.belumBayar ?? []);
	let lunas = $derived(props.lunas ?? []);
	let tab = $state<"belum" | "lunas">("belum");
	const rupiah = (n: number) => new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 0 }).format(n);
</script>

<AppLayout user={props.user}>
	<main class="max-w-7xl mx-auto px-4 sm:px-6 py-8 space-y-7">
		<section class="flex flex-col sm:flex-row sm:items-end justify-between gap-4">
			<div><p class="text-xs font-bold tracking-[0.16em] uppercase text-brand-600 dark:text-brand-400">Keuangan · {props.periode}</p><h1 class="mt-2 text-3xl font-bold text-neutral-900 dark:text-white">Tagihan yang perlu dijaga</h1><p class="mt-2 text-neutral-600 dark:text-neutral-400">Pantau kolektibilitas dan tindak lanjuti pembayaran SPP.</p></div>
			<a href="/app/keuangan/tagihan" use:inertia class="inline-flex items-center justify-center gap-2 px-4 py-2.5 rounded-xl bg-brand-600 hover:bg-brand-700 text-white font-semibold text-sm"><WalletCards size="17" /> Kelola tagihan <ArrowRight size="16" /></a>
		</section>

		<section class="grid grid-cols-2 lg:grid-cols-4 gap-3">
			<div class="rounded-2xl bg-neutral-900 text-white p-5"><p class="text-xs text-neutral-400">Total tagihan</p><p class="mt-3 text-2xl font-bold">{rupiah(ringkasan.nominal_tagihan)}</p><p class="mt-1 text-sm text-neutral-400">{ringkasan.total_tagihan} tagihan</p></div>
			<div class="rounded-2xl border border-success/20 bg-success/10 p-5"><p class="text-xs text-neutral-600 dark:text-neutral-300">Sudah bayar</p><p class="mt-3 text-2xl font-bold text-success">{rupiah(ringkasan.nominal_lunas)}</p><p class="mt-1 text-sm text-neutral-600 dark:text-neutral-400">{ringkasan.total_lunas} tagihan</p></div>
			<div class="rounded-2xl border border-warning/20 bg-warning/10 p-5"><p class="text-xs text-neutral-600 dark:text-neutral-300">Belum bayar</p><p class="mt-3 text-2xl font-bold text-warning">{rupiah(ringkasan.nominal_belum_bayar)}</p><p class="mt-1 text-sm text-neutral-600 dark:text-neutral-400">{ringkasan.total_belum_bayar} tagihan</p></div>
			<div class="rounded-2xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-925 p-5"><p class="text-xs text-neutral-600 dark:text-neutral-300">Kolektibilitas</p><p class="mt-3 text-2xl font-bold text-brand-600 dark:text-brand-400">{ringkasan.kolektibilitas.toFixed(1)}%</p><p class="mt-1 text-sm {ringkasan.total_terlambat ? 'text-error' : 'text-neutral-600 dark:text-neutral-400'}">{ringkasan.total_terlambat} terlambat</p></div>
		</section>

		<section class="rounded-2xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-925 overflow-hidden">
			<div class="flex items-center justify-between border-b border-neutral-200 dark:border-neutral-800 px-5 py-4"><div class="flex gap-2"><button onclick={() => tab = "belum"} class="px-3 py-1.5 rounded-lg text-sm font-semibold {tab === 'belum' ? 'bg-warning/15 text-warning' : 'text-neutral-500'}">Belum bayar ({belumBayar.length})</button><button onclick={() => tab = "lunas"} class="px-3 py-1.5 rounded-lg text-sm font-semibold {tab === 'lunas' ? 'bg-success/15 text-success' : 'text-neutral-500'}">Sudah bayar ({lunas.length})</button></div><a href="/app/keuangan/tagihan" use:inertia class="text-sm text-brand-600 dark:text-brand-400">Lihat semua</a></div>
			<div class="divide-y divide-neutral-100 dark:divide-neutral-800/70">
				{#each (tab === "belum" ? belumBayar : lunas).slice(0, 8) as tagihan (tagihan.id)}
					<div class="flex items-center gap-3 p-4"><div class="p-2 rounded-xl {tab === 'belum' ? 'bg-warning/10 text-warning' : 'bg-success/10 text-success'}">{#if tab === "belum"}<CircleAlert size="18" />{:else}<CircleCheck size="18" />{/if}</div><div class="min-w-0 flex-1"><p class="font-semibold text-neutral-900 dark:text-white truncate">{tagihan.santri_nama}</p><p class="text-xs text-neutral-500">{tagihan.kelas_nama || "Tanpa kelas"} · Bulan ke-{tagihan.bulan_ke}</p></div><p class="font-mono text-sm text-neutral-700 dark:text-neutral-300">{rupiah(tagihan.nominal)}</p></div>
				{:else}<div class="p-12 text-center text-sm text-neutral-500">Belum ada tagihan pada periode ini.</div>{/each}
			</div>
		</section>
	</main>
</AppLayout>
