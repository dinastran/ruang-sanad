<script lang="ts">
	import { Cloud, CloudOff, CloudUpload, RefreshCw } from "lucide-svelte";
	import { syncState } from "@lib/tasbih/sync.svelte";

	const info = $derived(
		syncState.status === "gagal"
			? { label: syncState.pending > 0 ? `Penyimpanan perangkat gagal · ${syncState.pending} belum terkirim` : "Penyimpanan perangkat gagal · jangan tutup tab sebelum tersinkron", icon: CloudOff, cls: "text-red-600 dark:text-red-400", spin: false }
			: syncState.status === "menyinkron"
			? { label: "Menyinkron…", icon: RefreshCw, cls: "text-brand-600 dark:text-brand-400", spin: true }
			: syncState.status === "offline"
				? { label: `Offline · ${syncState.pending} menunggu`, icon: CloudOff, cls: "text-warning", spin: false }
				: syncState.pending > 0
					? { label: `Menunggu sinkron (${syncState.pending})`, icon: CloudUpload, cls: "text-warning", spin: false }
					: { label: "Tersimpan", icon: Cloud, cls: "text-neutral-500 dark:text-neutral-400", spin: false },
	);
</script>

<span class="inline-flex items-center gap-1.5 text-xs font-medium {info.cls}" role="status" aria-live="polite" data-testid="sync-status">
	<info.icon size={14} class={info.spin ? "animate-spin motion-reduce:animate-none" : ""} aria-hidden="true" />
	<span>{info.label}</span>
</span>
