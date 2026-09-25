<script lang="ts">
	import { router } from "@inertiajs/svelte";
	import RiayahDialog from "./RiayahDialog.svelte";
	import { adaFlashError, mediaKontak, todayISO } from "@lib/riayah";

	interface Props {
		santri: { id: number; nama: string };
		/** Halaman tujuan setelah simpan; default profil santri. */
		kembali?: "/app/guru/riayah" | "";
		onclose: () => void;
	}

	let { santri, kembali = "", onclose }: Props = $props();

	const maxTanggal = todayISO();
	let media = $state<string>("wa");
	let tanggal = $state(maxTanggal);
	let catatan = $state("");
	let busy = $state(false);

	function simpan(event: SubmitEvent) {
		event.preventDefault();
		if (busy) return;
		busy = true;
		const query = kembali ? `?kembali=${encodeURIComponent(kembali)}` : "";
		router.post(`/app/guru/santri/${santri.id}/kontak${query}`, { media, tanggal, catatan, jenis: "sapa" }, {
			preserveScroll: true,
			preserveState: true,
			// Server menjawab error dengan redirect + flash; dialog tetap terbuka
			// supaya isian guru tidak hilang.
			onSuccess: (page) => {
				if (!adaFlashError(page)) onclose();
			},
			onFinish: () => (busy = false),
		});
	}
</script>

<RiayahDialog title="Catat sapaan" subtitle={santri.nama} {busy} {onclose}>
	<form onsubmit={simpan} class="space-y-4">
		<fieldset>
			<legend class="mb-2 text-sm font-medium text-neutral-700 dark:text-neutral-300">Cara menghubungi</legend>
			<div class="grid grid-cols-2 gap-2">
				{#each mediaKontak as m (m.value)}
					<label class="flex min-h-11 cursor-pointer items-center gap-2 rounded-xl border px-3 text-sm has-[:checked]:border-brand-500/50 has-[:checked]:bg-brand-400/10 border-neutral-200 dark:border-neutral-800">
						<input type="radio" name="media" value={m.value} bind:group={media} class="accent-brand-600" />
						{m.label}
					</label>
				{/each}
			</div>
		</fieldset>
		<label class="block">
			<span class="mb-1.5 block text-sm font-medium text-neutral-700 dark:text-neutral-300">Tanggal</span>
			<input type="date" bind:value={tanggal} max={maxTanggal} required class="w-full rounded-xl border border-neutral-200 bg-white px-3 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-brand-400/40 dark:border-neutral-800 dark:bg-neutral-925" />
		</label>
		<label class="block">
			<span class="mb-1.5 block text-sm font-medium text-neutral-700 dark:text-neutral-300">Ringkasan (opsional)</span>
			<textarea bind:value={catatan} rows="3" placeholder="Mis. menanyakan kabar, santri sedang sakit, minta izin 2 pekan" class="w-full rounded-xl border border-neutral-200 bg-white px-3 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-brand-400/40 dark:border-neutral-800 dark:bg-neutral-925"></textarea>
		</label>
		<div class="flex justify-end gap-2">
			<button type="button" onclick={onclose} disabled={busy} class="min-h-10 rounded-xl px-4 text-sm font-medium text-neutral-600 hover:bg-neutral-100 disabled:opacity-50 dark:text-neutral-300 dark:hover:bg-neutral-800">Batal</button>
			<button type="submit" disabled={busy} class="min-h-10 whitespace-nowrap rounded-xl bg-brand-600 px-4 text-sm font-semibold text-white hover:bg-brand-700 disabled:opacity-50 focus:outline-none focus-visible:ring-2 focus-visible:ring-brand-400/60">{busy ? "Menyimpan..." : "Simpan"}</button>
		</div>
	</form>
</RiayahDialog>
