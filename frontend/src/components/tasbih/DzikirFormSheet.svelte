<script lang="ts">
	// Form tambah/ubah dzikir custom. Kirim lewat router.post/put (Inertia).
	import { router } from "@inertiajs/svelte";
	import Sheet from "./Sheet.svelte";
	import type { Dzikir } from "@lib/tasbih/types";

	interface Props {
		open: boolean;
		edit: Dzikir | null;
		onclose: () => void;
	}
	let { open, edit, onclose }: Props = $props();

	let latin = $state("");
	let arab = $state("");
	let arti = $state("");
	let target = $state(33);
	let menyimpan = $state(false);
	let galat = $state("");

	// Isi ulang form setiap kali sheet dibuka (side effect terhadap state form).
	$effect(() => {
		if (open) {
			latin = edit?.latin ?? "";
			arab = edit?.arab ?? "";
			arti = edit?.arti ?? "";
			target = edit?.target_default ?? 33;
			galat = "";
		}
	});

	function simpan(e: SubmitEvent) {
		e.preventDefault();
		if (!latin.trim()) {
			galat = "Teks latin wajib diisi.";
			return;
		}
		if (!Number.isInteger(target) || target < 1 || target > 100000) {
			galat = "Target harus 1 sampai 100.000.";
			return;
		}
		menyimpan = true;
		const data = { latin, arab, arti, target };
		const opts = {
			preserveScroll: true,
			onSuccess: () => onclose(),
			onFinish: () => (menyimpan = false),
		};
		if (edit) router.put(`/app/tasbih/dzikir/${edit.id}`, data, opts);
		else router.post("/app/tasbih/dzikir", data, opts);
	}

	const inputCls =
		"mt-1.5 w-full rounded-xl border border-neutral-300 bg-white px-3.5 py-2.5 text-base text-neutral-900 focus:border-brand-500 focus:outline-none focus:ring-2 focus:ring-brand-500/30 dark:border-neutral-700 dark:bg-neutral-950 dark:text-white";
</script>

<Sheet {open} title={edit ? "Ubah dzikir" : "Tambah dzikir"} {onclose}>
	<form class="space-y-4" onsubmit={simpan} novalidate>
		<div>
			<label for="dz-latin" class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Teks latin <span class="text-red-500">*</span></label>
			<input id="dz-latin" class={inputCls} maxlength="300" bind:value={latin} placeholder="mis. Yaa Hayyu yaa Qayyuum" />
		</div>
		<div>
			<label for="dz-arab" class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Teks Arab (opsional)</label>
			<input id="dz-arab" dir="rtl" lang="ar" class="{inputCls} font-arab text-xl" maxlength="500" bind:value={arab} />
		</div>
		<div>
			<label for="dz-arti" class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Arti (opsional)</label>
			<input id="dz-arti" class={inputCls} maxlength="500" bind:value={arti} />
		</div>
		<div>
			<label for="dz-target" class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Target per sesi</label>
			<input id="dz-target" type="number" inputmode="numeric" min="1" max="100000" class={inputCls} bind:value={target} />
		</div>
		{#if galat}<p class="text-sm text-red-600 dark:text-red-400" role="alert">{galat}</p>{/if}
		<button type="submit" disabled={menyimpan} class="h-12 w-full rounded-xl bg-brand-600 font-semibold text-white transition active:scale-[0.98] disabled:opacity-60 dark:bg-brand-500 dark:text-neutral-950">
			{menyimpan ? "Menyimpan…" : "Simpan"}
		</button>
	</form>
</Sheet>
