<script lang="ts" module>
	export interface RegistrasiPanel {
		registrasi: { id: number; link_judul: string; link_slug: string; link_mode: string; created_at: string }[];
		payments: {
			id: number; metode: string; amount: number; status: string; channel: string; ada_bukti: boolean;
			catatan_sistem: string; perlu_tinjau: boolean; created_at: string; paid_at: string;
		}[];
	}
</script>

<script lang="ts">
	import { router } from "@inertiajs/svelte";
	import { FileImage, TriangleAlert } from "lucide-svelte";

	interface Props { leadID: number; panel: RegistrasiPanel }
	let { leadID, panel }: Props = $props();

	let tolakID = $state(0);
	let alasan = $state("");
	let menyimpan = $state(false);

	const rupiah = (n: number) => "Rp" + (n ?? 0).toLocaleString("id-ID");
	const waktu = (s: string) => (s ? new Date(s).toLocaleString("id-ID", { day: "numeric", month: "short", hour: "2-digit", minute: "2-digit" }) : "");
	const STATUS: Record<string, [string, string]> = {
		menunggu_verifikasi: ["Menunggu verifikasi", "bg-amber-500/10 text-amber-700 dark:text-amber-400"],
		paid: ["Lunas", "bg-green-500/10 text-green-700 dark:text-green-400"],
		ditolak: ["Ditolak", "bg-red-500/10 text-red-700 dark:text-red-400"],
		pending: ["Menunggu bayar", "bg-sky-500/10 text-sky-700 dark:text-sky-400"],
		expired: ["Kedaluwarsa", "bg-neutral-500/10 text-neutral-600"],
		failed: ["Gagal", "bg-red-500/10 text-red-700 dark:text-red-400"],
	};

	function tolak(e: Event) {
		e.preventDefault();
		menyimpan = true;
		router.post(`/app/leads/${leadID}/payments/${tolakID}/tolak`, { alasan }, {
			preserveScroll: true,
			onSuccess: () => { tolakID = 0; alasan = ""; },
			onFinish: () => (menyimpan = false),
		});
	}
</script>

{#if panel.registrasi.length || panel.payments.length}
	<section class="rounded-xl border border-neutral-200/80 bg-white p-4 dark:border-white/[0.06] dark:bg-neutral-925/50">
		<h2 class="text-sm font-semibold text-neutral-900 dark:text-white">Pendaftaran & pembayaran</h2>

		{#if panel.payments.length}
			<ul class="mt-3 space-y-2">
				{#each panel.payments as p (p.id)}
					{@const st = STATUS[p.status] ?? [p.status, "bg-neutral-500/10 text-neutral-600"]}
					<li class="rounded-lg border border-neutral-200/80 p-3 text-sm dark:border-white/[0.06]">
						<div class="flex flex-wrap items-center justify-between gap-2">
							<span class="font-semibold tabular-nums text-neutral-900 dark:text-white">{rupiah(p.amount)}</span>
							<span class="rounded-full px-2 py-0.5 text-xs font-medium {st[1]}">{st[0]}</span>
						</div>
						<p class="mt-1 text-xs text-neutral-500">{p.metode === "manual" ? "Transfer manual" : `Online${p.channel ? " · " + p.channel : ""}`} · {waktu(p.created_at)}{p.paid_at ? ` · lunas ${waktu(p.paid_at)}` : ""}</p>
						{#if p.perlu_tinjau}<p class="mt-1.5 flex items-center gap-1.5 text-xs font-medium text-amber-700 dark:text-amber-400"><TriangleAlert size={13} /> Perlu ditinjau{p.catatan_sistem ? `: ${p.catatan_sistem}` : ""}</p>{:else if p.catatan_sistem}<p class="mt-1 text-xs text-neutral-600 dark:text-neutral-400">Catatan: {p.catatan_sistem}</p>{/if}
						<div class="mt-2 flex flex-wrap gap-3">
							{#if p.ada_bukti}<a href={`/app/leads/${leadID}/payments/${p.id}/bukti`} target="_blank" rel="noopener" class="inline-flex items-center gap-1 text-xs font-semibold text-brand-700 hover:underline dark:text-brand-400"><FileImage size={13} /> Lihat bukti</a>{/if}
							{#if p.status === "menunggu_verifikasi" && tolakID !== p.id}<button type="button" class="text-xs font-semibold text-red-600 hover:underline dark:text-red-400" onclick={() => (tolakID = p.id)}>Tolak bukti</button>{/if}
						</div>
						{#if tolakID === p.id}
							<form class="mt-2 flex gap-2" onsubmit={tolak}>
								<input bind:value={alasan} required maxlength="300" placeholder="Alasan, mis. nominal kurang" aria-label="Alasan penolakan"
									class="h-8 min-w-0 flex-1 rounded-md border border-neutral-200 bg-neutral-50 px-2 text-xs focus:border-brand-500 focus:outline-none focus:ring-2 focus:ring-brand-500/30 dark:border-white/10 dark:bg-neutral-950 dark:text-white" />
								<button type="submit" disabled={menyimpan || !alasan.trim()} class="h-8 rounded-md bg-red-600 px-2.5 text-xs font-semibold text-white disabled:opacity-50">Tolak</button>
								<button type="button" class="h-8 px-1 text-xs text-neutral-500" onclick={() => (tolakID = 0)}>Batal</button>
							</form>
						{/if}
					</li>
				{/each}
			</ul>
			<p class="mt-2 text-xs text-neutral-500">Bukti manual dinyatakan lunas lewat tombol Verifikasi pembayaran.</p>
		{/if}

		{#if panel.registrasi.length}
			<h3 class="mt-4 text-xs font-medium uppercase tracking-wide text-neutral-500">Isi form</h3>
			<ul class="mt-1.5 space-y-1 text-xs text-neutral-600 dark:text-neutral-400">
				{#each panel.registrasi as r (r.id)}
					<li>{waktu(r.created_at)} · {r.link_judul} <span class="font-mono text-neutral-400">/{r.link_slug}</span></li>
				{/each}
			</ul>
		{/if}
	</section>
{/if}
