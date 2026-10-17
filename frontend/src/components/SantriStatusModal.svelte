<script lang="ts">
	import { router } from "@inertiajs/svelte";
	import { fly } from "svelte/transition";
	import StatusBadge from "@components/StatusBadge.svelte";
	import type { Flash } from "@lib/types";
	import { Toast } from "@lib/notifications/toast";

	interface SantriStatusTarget {
		id: number;
		nama: string;
		status: string;
		status_alasan: string;
		cuti_mulai: string;
		cuti_selesai: string;
	}

	interface Props {
		santri: SantriStatusTarget;
		// PUT endpoint for this santri (including any return_to query).
		action: string;
		onclose: () => void;
		onsaved?: () => void;
	}

	let { santri, action, onclose, onsaved }: Props = $props();

	const labelStatus: Record<string, string> = { aktif: "Aktif", cuti: "Cuti", nonaktif: "Nonaktif" };

	function formatTanggal(iso: string): string {
		if (!iso) return "";
		const [y, m, d] = iso.split("-").map(Number);
		return new Date(y, m - 1, d).toLocaleDateString("id-ID", { day: "numeric", month: "short", year: "numeric" });
	}

	function hariIni(): string {
		const d = new Date();
		return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
	}

	// The modal is mounted per santri, so initial values are taken once.
	// svelte-ignore state_referenced_locally
	let statusBaru = $state<"aktif" | "cuti" | "nonaktif">(santri.status === "aktif" || santri.status === "cuti" ? "cuti" : "aktif");
	// svelte-ignore state_referenced_locally
	let statusAlasan = $state(santri.status === "cuti" || santri.cuti_mulai ? santri.status_alasan : "");
	// An aktif santri with cuti dates has a scheduled cuti; prefill it for editing.
	// svelte-ignore state_referenced_locally
	let cutiMulai = $state(santri.cuti_mulai ? santri.cuti_mulai : hariIni());
	// svelte-ignore state_referenced_locally
	let cutiSelesai = $state(santri.cuti_selesai);
	let isLoading = $state(false);

	let formValid = $derived.by(() => {
		// aktif -> aktif is allowed only to cancel a scheduled cuti.
		if (statusBaru === santri.status && statusBaru !== "cuti" && !(statusBaru === "aktif" && santri.cuti_mulai)) return false;
		if (statusBaru === "aktif") return true;
		if (!statusAlasan.trim()) return false;
		if (statusBaru === "cuti") return !!cutiMulai && !!cutiSelesai && cutiSelesai >= cutiMulai;
		return true;
	});

	function submit() {
		if (!formValid) return;
		isLoading = true;
		router.put(action, {
			status: statusBaru,
			alasan: statusBaru === "aktif" ? "" : statusAlasan.trim(),
			cuti_mulai: statusBaru === "cuti" ? cutiMulai : "",
			cuti_selesai: statusBaru === "cuti" ? cutiSelesai : "",
		}, {
			preserveScroll: true,
			preserveState: true,
			onSuccess: (p) => {
				// Status handlers report failures via the session flash and still redirect.
				const err = (p.props.flash as Flash | undefined)?.error;
				if (err) { Toast(err, "error"); return; }
				onsaved?.();
				onclose();
			},
			onFinish: () => { isLoading = false; },
		});
	}
</script>

<div class="fixed inset-0 z-50 flex items-center justify-center p-4">
	<button class="absolute inset-0 w-full h-full bg-neutral-900/50 backdrop-blur-sm" aria-label="Tutup modal" onclick={onclose}></button>
	<div class="relative w-full max-w-md max-h-[90vh] overflow-y-auto bg-white dark:bg-neutral-925 rounded-2xl shadow-xl border border-neutral-200/80 dark:border-white/[0.06] p-6" in:fly={{ y: 20, duration: 200 }}>
		<h3 class="text-lg font-bold text-neutral-900 dark:text-white mb-1">Ubah Status Santri</h3>
		<p class="text-sm text-neutral-600 dark:text-neutral-400 mb-4">
			<strong>{santri.nama}</strong> · saat ini <StatusBadge status={santri.status} />
		</p>
		{#if santri.status === "tidak_lanjut"}
			<p class="-mt-2 mb-4 rounded-xl bg-warning/10 px-3 py-2 text-xs text-neutral-700 dark:text-neutral-300">
				Keterangan Tidak Lanjut akan dikosongkan dan diganti status baru.
			</p>
		{/if}

		<fieldset class="mb-4">
			<legend class="text-xs font-semibold uppercase tracking-wider text-neutral-500 mb-2">Status baru</legend>
			<div class="grid grid-cols-3 gap-2">
				{#each ["aktif", "cuti", "nonaktif"] as opsi}
					{@const disabled = opsi === santri.status && opsi !== "cuti" && !(opsi === "aktif" && santri.cuti_mulai)}
					<label class="flex cursor-pointer items-center justify-center rounded-xl border px-3 py-2.5 text-sm font-medium transition-colors {statusBaru === opsi ? 'border-brand-500 bg-brand-400/10 text-brand-700 dark:text-brand-300' : 'border-neutral-300 dark:border-neutral-700/80 text-neutral-700 dark:text-neutral-300 hover:bg-neutral-100 dark:hover:bg-neutral-800'} {disabled ? 'opacity-40 cursor-not-allowed' : ''}">
						<input type="radio" class="sr-only" name="status_baru" value={opsi} bind:group={statusBaru} {disabled} />
						{labelStatus[opsi]}
					</label>
				{/each}
			</div>
			<p class="mt-2 text-xs text-neutral-500 dark:text-neutral-400">
				{#if statusBaru === "cuti" && cutiMulai > hariIni()}
					Cuti terjadwal: santri tetap Aktif (diabsen dan ditagih) sampai tanggal mulai, lalu otomatis menjadi Cuti dan kembali Aktif setelah tanggal selesai.
				{:else if statusBaru === "cuti"}
					Cuti tetap memegang kursi kelas, tidak diabsen, dan tidak ditagih infaq. Status kembali Aktif otomatis setelah tanggal selesai.
				{:else if statusBaru === "aktif" && santri.status === "aktif" && santri.cuti_mulai}
					Membatalkan cuti terjadwal {formatTanggal(santri.cuti_mulai)} – {formatTanggal(santri.cuti_selesai)}.
				{:else if statusBaru === "nonaktif"}
					Nonaktif melepas kursi kelas, tidak diabsen, dan tidak ditagih infaq.
				{:else}
					Santri kembali diabsen dan ditagih seperti biasa.
				{/if}
			</p>
		</fieldset>

		{#if statusBaru === "cuti"}
			<div class="grid grid-cols-2 gap-3 mb-4">
				<label class="block">
					<span class="text-xs font-semibold uppercase tracking-wider text-neutral-500">Mulai cuti</span>
					<input type="date" bind:value={cutiMulai} class="mt-1 w-full px-3 py-2.5 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white outline-none text-sm" />
				</label>
				<label class="block">
					<span class="text-xs font-semibold uppercase tracking-wider text-neutral-500">Sampai (hari terakhir)</span>
					<input type="date" bind:value={cutiSelesai} min={cutiMulai > hariIni() ? cutiMulai : hariIni()} class="mt-1 w-full px-3 py-2.5 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white outline-none text-sm" />
				</label>
			</div>
		{/if}

		{#if statusBaru !== "aktif"}
			<label class="block mb-4">
				<span class="text-xs font-semibold uppercase tracking-wider text-neutral-500">Alasan <span class="text-red-500">*</span></span>
				<textarea bind:value={statusAlasan} rows="3" placeholder={statusBaru === "cuti" ? "Contoh: sakit, safar, melahirkan" : "Contoh: mengundurkan diri, pindah domisili"} class="mt-1 w-full px-3 py-2.5 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white outline-none text-sm"></textarea>
			</label>
		{/if}

		<div class="flex items-center gap-3">
			<button onclick={onclose} class="flex-1 px-4 py-2.5 rounded-xl border border-neutral-300 dark:border-neutral-700/80 text-sm font-medium text-neutral-700 dark:text-neutral-300 hover:bg-neutral-100 dark:hover:bg-neutral-800 transition-colors">
				Batal
			</button>
			<button
				onclick={submit}
				disabled={!formValid || isLoading}
				class="flex-1 px-4 py-2.5 rounded-xl bg-brand-600 hover:bg-brand-700 text-white font-semibold transition-all dark:bg-brand-500 dark:hover:bg-brand-400 disabled:opacity-50 disabled:cursor-not-allowed text-sm"
			>
				{isLoading ? "Menyimpan..." : "Simpan Status"}
			</button>
		</div>
	</div>
</div>
