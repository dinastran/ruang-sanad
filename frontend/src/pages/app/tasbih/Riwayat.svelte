<script lang="ts">
	import { onMount } from "svelte";
	import { router } from "@inertiajs/svelte";
	import AppLayout from "@layouts/AppLayout.svelte";
	import RiwayatList from "@components/tasbih/RiwayatList.svelte";
	import SyncIndicator from "@components/tasbih/SyncIndicator.svelte";
	import { antreanSemua, setPemilik } from "@lib/tasbih/db";
	import { syncSekarang } from "@lib/tasbih/sync.svelte";
	import type { Dzikir, RiwayatItem, SesiTertutup } from "@lib/tasbih/types";
	import type { User } from "@lib/types";

	interface Props {
		user?: User;
		riwayat: { items: RiwayatItem[]; ada_lagi: boolean };
		dzikir: Dzikir[];
	}
	let { user, riwayat, dzikir }: Props = $props();

	if (user?.id) setPemilik(user.id);

	let lokal = $state<SesiTertutup[]>([]);

	async function bacaLokal() {
		try {
			lokal = await antreanSemua();
		} catch {
			lokal = [];
		}
	}

	onMount(() => {
		(async () => {
			await bacaLokal();
			const h = await syncSekarang(true);
			await bacaLokal();
			if ((h?.diterima.length ?? 0) > 0) router.reload({ only: ["riwayat"] });
		})();
	});
</script>

<AppLayout {user}>
	<div class="mx-auto w-full min-w-0 max-w-xl pb-8">
		<header class="mb-4 flex items-end justify-between gap-3">
			<h1 class="text-[22px] font-semibold tracking-tight text-neutral-900 dark:text-white">Riwayat</h1>
			<SyncIndicator />
		</header>
		{#key riwayat}
			<RiwayatList awal={riwayat.items} adaLagiAwal={riwayat.ada_lagi} {dzikir} {lokal} />
		{/key}
	</div>
</AppLayout>
