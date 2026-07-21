<script lang="ts">
	interface Props {
		gender: string;
	}
	let { gender }: Props = $props();

	let display = $derived(gender === "L" ? "Laki-laki" : gender === "P" ? "Perempuan" : gender);
	let cls = $derived(
		gender === "L"
			? "bg-blue-500/10 text-blue-700 dark:text-blue-400 ring-1 ring-blue-500/20"
			: "bg-pink-500/10 text-pink-700 dark:text-pink-400 ring-1 ring-pink-500/20"
	);
</script>

<span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium {cls}">
	{display}
</span>
