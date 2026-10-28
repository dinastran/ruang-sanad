<script lang="ts">
	import { inertia } from "@inertiajs/svelte";
	import type { AttendanceMonth } from "@lib/overview";
	import { attendanceRate, attendanceTotal, count, focusRing, kicker, monthLong, monthShort, panel, percent } from "./ui";

	let { attendance, available }: { attendance: AttendanceMonth[]; available: boolean } = $props();

	const parts = [
		{ key: "hadir", label: "Hadir", swatch: "bg-brand-600 dark:bg-brand-500" },
		{ key: "izin", label: "Izin", swatch: "bg-brand-200 dark:bg-brand-800" },
		{ key: "sakit", label: "Sakit", swatch: "bg-neutral-300 dark:bg-neutral-600" },
		{ key: "alpa", label: "Alpa", swatch: "bg-amber-500" },
	] as const;

	let latest = $derived(attendance.at(-1));
	let latestRate = $derived(attendanceRate(latest));
	let hasRecords = $derived(attendance.some((a) => attendanceTotal(a) > 0));
	let previousRate = $derived(attendanceRate(attendance.at(-2)));
	let diff = $derived(latestRate !== null && previousRate !== null ? latestRate - previousRate : null);
</script>

<section class={panel + " flex min-w-0 flex-col"} aria-labelledby="kehadiran-title">
	<div class="flex items-start justify-between gap-3 px-5 pt-5 sm:px-6">
		<div>
			<h2 id="kehadiran-title" class={kicker}>Kehadiran santri</h2>
			<p class="mt-1 text-sm text-neutral-600 dark:text-neutral-300">Absensi pada pertemuan yang sudah selesai.</p>
		</div>
		<a href="/app/koordinator-guru" use:inertia class={"shrink-0 rounded text-xs font-medium text-brand-700 hover:underline dark:text-brand-400 " + focusRing}>Monitoring</a>
	</div>

	{#if !available}
		<p class="px-5 py-12 text-sm text-neutral-500 sm:px-6 dark:text-neutral-400">Data kehadiran belum berhasil dimuat. Coba perbarui data.</p>
	{:else if !hasRecords}
		<p class="px-5 py-12 text-sm text-neutral-500 sm:px-6 dark:text-neutral-400">Belum ada absensi tercatat pada enam bulan terakhir.</p>
	{:else}
		<div class="flex flex-wrap items-end gap-x-3 gap-y-1 px-5 pt-4 sm:px-6">
			<span class="text-[32px] font-semibold leading-none tracking-[-0.02em] tabular-nums text-neutral-950 dark:text-white">{percent(latestRate)}</span>
			<span class="pb-0.5 text-xs text-neutral-500 dark:text-neutral-400">
				hadir pada {latest ? monthLong(latest.month) : ""}
				{#if diff !== null}· <span class={"font-medium tabular-nums " + (diff >= 0 ? "text-emerald-700 dark:text-emerald-400" : "text-red-700 dark:text-red-400")}>{diff >= 0 ? "+" : "−"}{percent(Math.abs(diff)).replace("%", "")} poin</span> dari bulan lalu{/if}
			</span>
		</div>

		<div class="grid grid-cols-6 items-end gap-2 px-5 pt-5 sm:gap-3 sm:px-6" role="img" aria-label="Komposisi kehadiran per bulan, enam bulan terakhir">
			{#each attendance as a (a.month)}
				{@const total = attendanceTotal(a)}
				{@const rate = attendanceRate(a)}
				<div class="flex min-w-0 flex-col items-stretch gap-1.5">
					<span class="text-center text-[11px] font-medium tabular-nums text-neutral-600 dark:text-neutral-300">{rate === null ? "—" : Math.round(rate) + "%"}</span>
					<div class="flex h-32 flex-col-reverse overflow-hidden rounded-md bg-neutral-100 dark:bg-white/[0.04]" title={`${monthLong(a.month)}: ${count(a.meetings)} pertemuan · hadir ${count(a.hadir)}, izin ${count(a.izin)}, sakit ${count(a.sakit)}, alpa ${count(a.alpa)}`}>
						{#if total}
							{#each parts as p (p.key)}
								{#if a[p.key] > 0}<div class={p.swatch} style:height={`${(a[p.key] / total) * 100}%`}></div>{/if}
							{/each}
						{/if}
					</div>
					<span class="text-center text-[11px] text-neutral-500 dark:text-neutral-400">{monthShort(a.month)}</span>
				</div>
			{/each}
		</div>

		<ul class="flex flex-wrap gap-x-4 gap-y-1 px-5 pt-4 text-xs text-neutral-600 sm:px-6 dark:text-neutral-300">
			{#each parts as p (p.key)}
				<li class="flex items-center gap-1.5"><i class={"h-2.5 w-2.5 rounded-sm " + p.swatch}></i>{p.label}{latest ? ` ${count(latest[p.key])}` : ""}</li>
			{/each}
		</ul>

		<details class="mt-auto px-5 pb-4 pt-3 text-xs text-neutral-500 sm:px-6 dark:text-neutral-400">
			<summary class="w-fit py-1">Lihat angka</summary>
			<div class="overflow-x-auto">
				<table class="mt-1 w-full text-left tabular-nums">
					<thead><tr><th class="py-1 pr-3 font-medium">Bulan</th><th class="py-1 pr-3 text-right font-medium">Pertemuan</th>{#each parts as p (p.key)}<th class="py-1 pr-3 text-right font-medium">{p.label}</th>{/each}</tr></thead>
					<tbody>
						{#each attendance as a (a.month)}
							<tr class="border-t border-neutral-100 dark:border-white/[0.05]"><td class="py-1 pr-3">{monthShort(a.month)}</td><td class="py-1 pr-3 text-right">{count(a.meetings)}</td>{#each parts as p (p.key)}<td class="py-1 pr-3 text-right">{count(a[p.key])}</td>{/each}</tr>
						{/each}
					</tbody>
				</table>
			</div>
		</details>
	{/if}
</section>
