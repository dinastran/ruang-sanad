<script lang="ts">
	interface Props {
		status: string;
	}

	let { status }: Props = $props();

	const variants: Record<string, { bg: string; text: string; dot: string }> = {
		lengkap: { bg: "bg-green-500/10", text: "text-green-700 dark:text-green-400", dot: "bg-green-500" },
		aktif: { bg: "bg-green-500/10", text: "text-green-700 dark:text-green-400", dot: "bg-green-500" },
		perlu_dilengkapi: { bg: "bg-yellow-500/10", text: "text-yellow-700 dark:text-yellow-400", dot: "bg-yellow-500" },
		tidak_lanjut: { bg: "bg-red-500/10", text: "text-red-700 dark:text-red-400", dot: "bg-red-500" },
	};

	let v = $derived(variants[status] || { bg: "bg-neutral-500/10", text: "text-neutral-600 dark:text-neutral-400", dot: "bg-neutral-500" });

	let label = $derived(
		status === "perlu_dilengkapi" ? "Perlu Dilengkapi" :
		status === "tidak_lanjut" ? "Tidak Lanjut" :
		status.charAt(0).toUpperCase() + status.slice(1)
	);
</script>

<span class="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium {v.bg} {v.text}">
	<span class="w-1.5 h-1.5 rounded-full {v.dot}"></span>
	{label}
</span>
