<script lang="ts">
	// Ajakan pasang aplikasi: Android/Chrome (sekali ketuk) atau panduan iPhone.
	import { fade } from "svelte/transition";
	import { Download, Share, SquarePlus, X } from "lucide-svelte";
	import { bannerDitutupBaruBaru, pasangAplikasi, pwa, tutupBanner } from "@lib/pwa.svelte";

	let ditutup = $state(bannerDitutupBaruBaru());
	let panduanIOS = $state(false);
	const tampil = $derived(!ditutup && !pwa.standalone && !pwa.terpasang && (pwa.bisaPasang || pwa.ios));

	function tutup() {
		tutupBanner();
		ditutup = true;
	}

	async function pasang() {
		if (pwa.bisaPasang) {
			const ok = await pasangAplikasi();
			if (!ok) tutup();
		} else {
			panduanIOS = !panduanIOS;
		}
	}
</script>

{#if tampil}
	<section class="mb-4 rounded-2xl border border-brand-200 bg-brand-50 p-3 dark:border-brand-900 dark:bg-brand-950/40" transition:fade={{ duration: 150 }} aria-label="Pasang aplikasi" data-testid="install-banner">
		<div class="flex items-center gap-3">
			<img src="/public/icons/tasbih-192.png" alt="" class="h-11 w-11 shrink-0 rounded-xl" width="44" height="44" />
			<div class="min-w-0 flex-1">
				<p class="text-sm font-semibold text-neutral-900 dark:text-white">Pasang aplikasi Tasbih</p>
				<p class="text-xs text-neutral-600 dark:text-neutral-400">Buka dari layar utama, tetap jalan tanpa internet.</p>
			</div>
			<button type="button" onclick={pasang} class="inline-flex min-h-10 shrink-0 items-center gap-1.5 rounded-xl bg-brand-600 px-3 text-sm font-semibold text-white dark:bg-brand-500 dark:text-neutral-950">
				<Download size={15} aria-hidden="true" /> Pasang
			</button>
			<button type="button" onclick={tutup} class="grid h-10 w-10 shrink-0 place-items-center rounded-lg text-neutral-500 hover:bg-white/60 dark:hover:bg-neutral-800" aria-label="Tutup ajakan pasang">
				<X size={16} />
			</button>
		</div>
		{#if panduanIOS}
			<ol class="mt-3 space-y-2.5 border-t border-brand-200 pt-3 text-sm text-neutral-700 dark:border-brand-900 dark:text-neutral-300" data-testid="panduan-ios">
				{#each [1, 2, 3] as n}
					<li class="flex items-start gap-2.5">
						<span class="grid h-6 w-6 shrink-0 place-items-center rounded-full bg-white text-xs font-semibold dark:bg-neutral-900">{n}</span>
						<span class="pt-0.5 leading-snug">
							{#if n === 1}
								Ketuk <Share size={15} class="inline -mt-0.5 text-brand-600 dark:text-brand-400" aria-label="ikon Bagikan" /> <b>Bagikan</b> di bawah layar Safari
							{:else if n === 2}
								Pilih <SquarePlus size={15} class="inline -mt-0.5 text-brand-600 dark:text-brand-400" aria-hidden="true" /> <b>Tambah ke Layar Utama</b>
							{:else}
								Ketuk <b>Tambah</b>, lalu buka <b>Tasbih</b> dari layar utama
							{/if}
						</span>
					</li>
				{/each}
			</ol>
		{/if}
	</section>
{/if}
