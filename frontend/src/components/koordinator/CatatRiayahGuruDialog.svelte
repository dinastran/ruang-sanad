<script lang="ts">
	import { router } from "@inertiajs/svelte";
	import RiayahDialog from "@components/riayah/RiayahDialog.svelte";
	import { adaFlashError } from "@lib/riayah";

	interface Props {
		guru: { guru_id: number; nama: string };
		/** Isi awal catatan, mis. ringkasan alfa berturut-turut. */
		awal?: string;
		onclose: () => void;
	}

	let { guru, awal = "", onclose }: Props = $props();

	const catatanAwal = () => awal;
	let catatan = $state(catatanAwal());
	let busy = $state(false);

	function simpan(event: SubmitEvent) {
		event.preventDefault();
		if (busy || !catatan.trim()) return;
		busy = true;
		router.post(`/app/koordinator-guru/guru/${guru.guru_id}/riayah`, { catatan }, {
			preserveScroll: true,
			preserveState: true,
			onSuccess: (page) => {
				if (!adaFlashError(page)) onclose();
			},
			onFinish: () => (busy = false),
		});
	}
</script>

<RiayahDialog title="Catat riayah guru" subtitle={guru.nama} {busy} {onclose}>
	<form onsubmit={simpan} class="space-y-4">
		<label class="block">
			<span class="mb-1.5 block text-sm font-medium text-neutral-700 dark:text-neutral-300">Catatan</span>
			<textarea bind:value={catatan} rows="4" required placeholder="Mis. sudah ditanyakan kabarnya, sedang ada kendala jadwal kerja" class="w-full rounded-xl border border-neutral-200 bg-white px-3 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-brand-400/40 dark:border-neutral-800 dark:bg-neutral-925"></textarea>
		</label>
		<p class="text-xs text-neutral-500">Catatan tersimpan di riwayat riayah pada profil guru.</p>
		<div class="flex justify-end gap-2">
			<button type="button" onclick={onclose} disabled={busy} class="min-h-10 rounded-xl px-4 text-sm font-medium text-neutral-600 hover:bg-neutral-100 disabled:opacity-50 dark:text-neutral-300 dark:hover:bg-neutral-800">Batal</button>
			<button type="submit" disabled={busy || !catatan.trim()} class="min-h-10 whitespace-nowrap rounded-xl bg-brand-600 px-4 text-sm font-semibold text-white hover:bg-brand-700 disabled:opacity-50 focus:outline-none focus-visible:ring-2 focus-visible:ring-brand-400/60">{busy ? "Menyimpan..." : "Simpan"}</button>
		</div>
	</form>
</RiayahDialog>
