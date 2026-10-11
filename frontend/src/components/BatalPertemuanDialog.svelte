<script lang="ts">
	import { router } from "@inertiajs/svelte";
	import { fly } from "svelte/transition";

	// Konfirmasi pembatalan pertemuan yang salah dimulai. Pertemuan dihapus
	// tanpa absensi; alasan wajib dan tercatat di riwayat kelas.
	interface Props { kelasId: number; pertemuanId: number; label: string; onclose: () => void; }
	let { kelasId, pertemuanId, label, onclose }: Props = $props();

	let alasan = $state("");
	let saving = $state(false);

	function batalkan() {
		if (!alasan.trim()) return;
		saving = true;
		router.post(`/app/guru/kelas/${kelasId}/pertemuan/${pertemuanId}/batal`, { alasan: alasan.trim() }, {
			preserveScroll: true,
			onSuccess: onclose,
			onFinish: () => { saving = false; },
		});
	}
</script>

<div class="fixed inset-0 z-50 flex items-center justify-center p-4">
	<button class="absolute inset-0 h-full w-full bg-neutral-900/50 backdrop-blur-sm" aria-label="Tutup" onclick={onclose}></button>
	<div class="relative w-full max-w-md rounded-2xl border border-neutral-200/80 bg-white p-6 shadow-xl dark:border-white/[0.06] dark:bg-neutral-925" in:fly={{ y: 20, duration: 200 }}>
		<h3 class="text-lg font-bold text-neutral-900 dark:text-white">Batalkan pertemuan?</h3>
		<p class="mt-1 text-sm leading-6 text-neutral-600 dark:text-neutral-400">{label}. Pertemuan dihapus tanpa absensi dan tanpa tagihan. Gunakan hanya bila kelas <strong>tidak</strong> berlangsung. Bila kelas benar berlangsung, isi absensi lalu selesaikan.</p>
		<label class="mt-4 block">
			<span class="text-xs font-semibold uppercase tracking-wider text-neutral-500">Alasan <span class="text-red-500">*</span></span>
			<textarea bind:value={alasan} rows="3" maxlength="500" placeholder="Contoh: salah klik Mulai, kelas diliburkan" class="mt-1.5 w-full rounded-xl border border-neutral-300 bg-white px-3 py-2.5 text-sm text-neutral-900 outline-none focus:border-brand-400 dark:border-neutral-700 dark:bg-neutral-800 dark:text-white"></textarea>
		</label>
		<div class="mt-5 flex gap-3">
			<button onclick={onclose} class="flex-1 rounded-xl border border-neutral-300 px-4 py-2.5 text-sm font-medium text-neutral-700 hover:bg-neutral-100 dark:border-neutral-700 dark:text-neutral-300 dark:hover:bg-neutral-800">Kembali</button>
			<button onclick={batalkan} disabled={saving || !alasan.trim()} class="flex-1 rounded-xl bg-red-600 px-4 py-2.5 text-sm font-semibold text-white hover:bg-red-700 disabled:cursor-not-allowed disabled:opacity-50">{saving ? "Membatalkan..." : "Batalkan pertemuan"}</button>
		</div>
	</div>
</div>
