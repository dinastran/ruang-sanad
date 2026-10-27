<script lang="ts">
	import { todayWIB } from "@lib/leads";
	import { CalendarDays } from "lucide-svelte";

	interface Props { from: string; to: string; loading?: boolean; onapply: (from: string, to: string) => void }
	let { from, to, loading = false, onapply }: Props = $props();

	let customOpen = $state(false);
	let customFrom = $state("");
	let customTo = $state("");
	let validation = $state("");

	function dateOffset(offset: number) {
		const v = new Date(todayWIB() + "T12:00:00+07:00");
		v.setUTCDate(v.getUTCDate() + offset);
		return v.toLocaleDateString("en-CA", { timeZone: "Asia/Jakarta" });
	}
	const presets = [
		{ key: 1, label: "Hari ini" },
		{ key: -1, label: "Kemarin" },
		{ key: 7, label: "7 hari" },
		{ key: 14, label: "14 hari" },
		{ key: 30, label: "30 hari" },
	];
	let active = $derived.by(() => {
		const today = todayWIB();
		if (from === today && to === today) return 1;
		if (from === dateOffset(-1) && to === dateOffset(-1)) return -1;
		for (const n of [7, 14, 30]) if (from === dateOffset(1 - n) && to === today) return n;
		return 0;
	});

	function apply(f: string, t: string) {
		if (!f || !t || f > t) { validation = "Tanggal mulai harus sebelum atau sama dengan tanggal akhir."; return; }
		const days = (new Date(t + "T00:00:00+07:00").getTime() - new Date(f + "T00:00:00+07:00").getTime()) / 86400000;
		if (days >= 366) { validation = "Maksimal 366 hari."; return; }
		validation = "";
		onapply(f, t);
	}
	function toggleCustom() {
		if (!customOpen) { customFrom = from; customTo = to; }
		customOpen = !customOpen;
	}
	function preset(key: number) {
		customOpen = false;
		const end = key === -1 ? dateOffset(-1) : todayWIB();
		apply(key === -1 ? end : dateOffset(1 - key), end);
	}
	const seg = "relative rounded-md px-3 py-1.5 text-sm font-medium transition-colors duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500";
	const segIdle = " text-neutral-600 hover:text-neutral-900 dark:text-neutral-400 dark:hover:text-white";
	const segOn = " bg-white text-neutral-900 shadow-[0_0_0_1px_rgba(15,23,42,.08),0_1px_2px_rgba(15,23,42,.08)] dark:bg-neutral-800 dark:text-white dark:shadow-[0_0_0_1px_rgba(255,255,255,.08)]";
</script>

<div class="flex flex-col items-stretch gap-2 sm:items-end">
	<div role="group" aria-label="Periode" class="flex flex-wrap gap-0.5 rounded-lg bg-neutral-100 p-1 dark:bg-neutral-900" aria-busy={loading}>
		{#each presets as p}
			<button type="button" onclick={() => preset(p.key)} aria-pressed={active === p.key} class={seg + (active === p.key && !customOpen ? segOn : segIdle)}>{p.label}</button>
		{/each}
		<button type="button" onclick={toggleCustom} aria-expanded={customOpen} aria-pressed={active === 0}
			class={seg + " inline-flex items-center gap-1.5" + (active === 0 || customOpen ? segOn : segIdle)}>
			<CalendarDays size={14} /> Custom
		</button>
	</div>
	{#if customOpen}
		<form class="flex flex-wrap items-end gap-2" onsubmit={(e) => { e.preventDefault(); apply(customFrom, customTo); }}>
			<label class="text-xs font-medium text-neutral-500 dark:text-neutral-400">Dari
				<input type="date" bind:value={customFrom} class="mt-1 block rounded-md border border-neutral-300 bg-neutral-50 px-2 py-1.5 text-sm text-neutral-900 dark:border-white/10 dark:bg-neutral-950 dark:text-white" />
			</label>
			<label class="text-xs font-medium text-neutral-500 dark:text-neutral-400">Sampai
				<input type="date" bind:value={customTo} class="mt-1 block rounded-md border border-neutral-300 bg-neutral-50 px-2 py-1.5 text-sm text-neutral-900 dark:border-white/10 dark:bg-neutral-950 dark:text-white" />
			</label>
			<button type="submit" class="rounded-md bg-brand-600 px-3 py-2 text-sm font-semibold text-white transition-colors hover:bg-brand-700 active:scale-[0.97] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 focus-visible:ring-offset-2 dark:focus-visible:ring-offset-neutral-950">Terapkan</button>
		</form>
	{/if}
	{#if validation}<p role="alert" class="text-sm text-error">{validation}</p>{/if}
</div>
