<script lang="ts">
	import type { RiayahPenanda } from "@lib/types";

	interface Props {
		penanda: RiayahPenanda;
		showAlasan?: boolean;
	}

	let { penanda, showAlasan = false }: Props = $props();

	const label: Record<string, string> = {
		kehadiran: "Kehadiran",
		kontak: "Belum disapa",
		progres: "Progres macet",
	};

	const warna: Record<string, string> = {
		merah: "bg-red-500/10 text-red-700 dark:text-red-300 border-red-500/20",
		oranye: "bg-orange-500/10 text-orange-700 dark:text-orange-300 border-orange-500/20",
		kuning: "bg-amber-500/10 text-amber-800 dark:text-amber-300 border-amber-500/20",
	};

	const titik: Record<string, string> = {
		merah: "bg-red-500",
		oranye: "bg-orange-500",
		kuning: "bg-amber-500",
	};
</script>

<span class="inline-flex max-w-full items-start gap-1.5 rounded-lg border px-2 py-1 text-xs font-medium {warna[penanda.level]}" title={penanda.alasan}>
	<span class="mt-1 h-1.5 w-1.5 shrink-0 rounded-full {titik[penanda.level]}" aria-hidden="true"></span>
	<span class="min-w-0">
		<span class="font-semibold">{label[penanda.kode]}</span>{#if showAlasan}<span class="font-normal"> · {penanda.alasan}</span>{/if}
	</span>
</span>
