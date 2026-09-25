<script lang="ts">
	import { untrack } from "svelte";
	import { router } from "@inertiajs/svelte";
	import RiayahDialog from "./RiayahDialog.svelte";
	import type { RiayahSantriItem, RiayahWATemplate } from "@lib/types";
	import { isiTemplate, salinTeks, waLink } from "@lib/riayah";
	import { MessageCircle, Copy, CheckCircle } from "lucide-svelte";

	interface Props {
		santri: RiayahSantriItem;
		templates: RiayahWATemplate[];
		pengirim?: string;
		kembali?: "/app/guru/riayah" | "";
		onclose: () => void;
	}

	let { santri, templates, pengirim = "", kembali = "", onclose }: Props = $props();

	const vars = $derived({ nama: santri.nama, nama_kelas: santri.nama_kelas, jadwal: santri.jadwal, level: santri.level, guru: pengirim });

	let templateIndex = $state(0);
	// Draf diisi sekali saat dialog dibuka, setelah itu bebas diedit guru.
	let pesan = $state(untrack(() => isiTemplate(templates[0]?.body ?? "", vars)));
	let langkah = $state<"tulis" | "konfirmasi">("tulis");
	let busy = $state(false);
	let disalin = $state(false);

	let link = $derived(waLink(santri.no_wa, pesan.trim()));

	function pilihTemplate(index: number) {
		templateIndex = index;
		pesan = isiTemplate(templates[index]?.body ?? "", vars);
	}

	function bukaWA() {
		if (!link || !pesan.trim()) return;
		window.open(link, "_blank", "noopener");
		langkah = "konfirmasi";
	}

	async function salin() {
		disalin = await salinTeks(pesan.trim());
		if (disalin) setTimeout(() => (disalin = false), 2000);
	}

	function catat() {
		if (busy) return;
		busy = true;
		const query = kembali ? `?kembali=${encodeURIComponent(kembali)}` : "";
		router.post(`/app/guru/santri/${santri.id}/kontak${query}`, { media: "wa", jenis: "sapa", catatan: pesan.trim() }, {
			preserveScroll: true,
			onSuccess: () => onclose(),
			onFinish: () => (busy = false),
		});
	}
</script>

<RiayahDialog title="Sapa via WhatsApp" subtitle={santri.nama} {busy} {onclose}>
	{#if langkah === "tulis"}
		<div class="space-y-4">
			{#if templates.length > 0}
				<label class="block">
					<span class="mb-1.5 block text-sm font-medium text-neutral-700 dark:text-neutral-300">Mulai dari template</span>
					<select value={templateIndex} onchange={(e) => pilihTemplate(Number(e.currentTarget.value))} class="w-full rounded-xl border border-neutral-200 bg-white px-3 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-brand-400/40 dark:border-neutral-800 dark:bg-neutral-925">
						{#each templates as tpl, i (i)}<option value={i}>{tpl.nama}</option>{/each}
					</select>
				</label>
			{/if}
			<label class="block">
				<span class="mb-1.5 block text-sm font-medium text-neutral-700 dark:text-neutral-300">Pesan</span>
				<textarea bind:value={pesan} rows="7" class="w-full rounded-xl border border-neutral-200 bg-white px-3 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-brand-400/40 dark:border-neutral-800 dark:bg-neutral-925"></textarea>
				<span class="mt-1 block text-xs text-neutral-500">Sesuaikan dengan kondisi santri. Pesan yang personal lebih terasa daripada template.</span>
			</label>
			{#if !santri.no_wa}
				<p class="rounded-xl bg-amber-500/10 px-3 py-2 text-sm text-amber-800 dark:text-amber-300">Nomor WA santri belum tercatat. Salin pesan lalu kirim manual.</p>
			{/if}
			<div class="flex flex-wrap justify-end gap-2">
				<button type="button" onclick={salin} disabled={!pesan.trim()} class="inline-flex min-h-10 items-center gap-1.5 whitespace-nowrap rounded-xl border border-neutral-200 px-4 text-sm font-medium text-neutral-700 hover:border-brand-400/40 disabled:opacity-50 dark:border-neutral-700 dark:text-neutral-300">
					{#if disalin}<CheckCircle class="h-4 w-4 text-green-600" /> Tersalin{:else}<Copy class="h-4 w-4" /> Salin{/if}
				</button>
				<button type="button" onclick={bukaWA} disabled={!link || !pesan.trim()} class="inline-flex min-h-10 items-center gap-1.5 whitespace-nowrap rounded-xl bg-brand-600 px-4 text-sm font-semibold text-white hover:bg-brand-700 disabled:opacity-50 focus:outline-none focus-visible:ring-2 focus-visible:ring-brand-400/60">
					<MessageCircle class="h-4 w-4" /> Buka WhatsApp
				</button>
			</div>
			{#if !santri.no_wa}
				<div class="border-t border-neutral-200 pt-3 text-right dark:border-neutral-800">
					<button type="button" onclick={() => (langkah = "konfirmasi")} class="text-sm font-medium text-brand-700 underline dark:text-brand-300">Sudah saya kirim manual</button>
				</div>
			{/if}
		</div>
	{:else}
		<div class="space-y-4">
			<p class="text-sm text-neutral-700 dark:text-neutral-300">Apakah pesan sudah benar-benar terkirim ke {santri.nama}?</p>
			<p class="text-xs text-neutral-500">Sapaan hanya dicatat setelah Anda konfirmasi, supaya catatan kontak sesuai kenyataan.</p>
			<div class="flex flex-wrap justify-end gap-2">
				<button type="button" onclick={() => (langkah = "tulis")} disabled={busy} class="min-h-10 whitespace-nowrap rounded-xl px-4 text-sm font-medium text-neutral-600 hover:bg-neutral-100 disabled:opacity-50 dark:text-neutral-300 dark:hover:bg-neutral-800">Belum, kembali</button>
				<button type="button" onclick={catat} disabled={busy} class="min-h-10 whitespace-nowrap rounded-xl bg-brand-600 px-4 text-sm font-semibold text-white hover:bg-brand-700 disabled:opacity-50">{busy ? "Mencatat..." : "Sudah, catat sapaan"}</button>
			</div>
		</div>
	{/if}
</RiayahDialog>
