<script lang="ts">
	import { inertia } from "@inertiajs/svelte";
	import { ArrowDown, ArrowUp, Search } from "lucide-svelte";
	import type { Overview } from "@lib/overview";
	import { count, focusRing, kicker, panel, percent } from "./ui";

	type Program = Overview["programs"][number];
	type Key = "name" | "active" | "cuti" | "stopped" | "classes" | "fill";

	let { programs, failed }: { programs: Program[]; failed: boolean } = $props();

	let query = $state("");
	let sortKey = $state<Key>("active");
	let ascending = $state(false);

	const fill = (p: Program) => (p.seats > 0 ? (p.occupied / p.seats) * 100 : -1);
	const columns: { key: Key; label: string; numeric: boolean }[] = [
		{ key: "name", label: "Program / tipe", numeric: false },
		{ key: "active", label: "Santri aktif", numeric: true },
		{ key: "cuti", label: "Cuti", numeric: true },
		{ key: "stopped", label: "Berhenti", numeric: true },
		{ key: "classes", label: "Kelas aktif", numeric: true },
		{ key: "fill", label: "Keterisian kursi", numeric: true },
	];

	let rows = $derived.by(() => {
		const q = query.trim().toLowerCase();
		const list = q ? programs.filter((p) => p.name.toLowerCase().includes(q)) : [...programs];
		const value = (p: Program) => (sortKey === "fill" ? fill(p) : p[sortKey]);
		list.sort((a, b) => {
			const va = value(a), vb = value(b);
			const order = typeof va === "string" ? va.localeCompare(vb as string, "id") : (va as number) - (vb as number);
			return ascending ? order : -order;
		});
		return list;
	});
	let totals = $derived(
		programs.reduce(
			(t, p) => ({ active: t.active + p.active, seats: t.seats + p.seats, occupied: t.occupied + p.occupied }),
			{ active: 0, seats: 0, occupied: 0 },
		),
	);

	function sortBy(key: Key) {
		if (sortKey === key) ascending = !ascending;
		else {
			sortKey = key;
			ascending = key === "name";
		}
	}
</script>

<section class={panel + " min-w-0"} aria-labelledby="program-title">
	<div class="flex flex-wrap items-end justify-between gap-3 px-5 pt-5 sm:px-6">
		<div>
			<h2 id="program-title" class={kicker}>Kondisi program kelas</h2>
			<p class="mt-1 text-sm text-neutral-600 dark:text-neutral-300">
				{#if programs.length}
					<span class="font-semibold tabular-nums text-neutral-900 dark:text-white">{count(totals.active)}</span> santri aktif ·
					kursi terisi <span class="font-semibold tabular-nums text-neutral-900 dark:text-white">{totals.seats ? percent((totals.occupied / totals.seats) * 100) : "—"}</span>
				{:else}
					Dikelompokkan berdasarkan tipe kelas.
				{/if}
			</p>
		</div>
		<div class="flex items-center gap-3">
			{#if programs.length > 4}
				<label class="relative block">
					<span class="sr-only">Cari program</span>
					<Search size={14} class="pointer-events-none absolute left-2.5 top-1/2 -translate-y-1/2 text-neutral-400" />
					<input type="search" bind:value={query} placeholder="Cari program" class="h-9 w-44 rounded-lg border border-neutral-200 bg-neutral-50 pl-8 pr-2 text-sm text-neutral-900 placeholder:text-neutral-400 focus:border-brand-500 focus:outline-none focus:ring-2 focus:ring-brand-500/30 dark:border-white/[0.08] dark:bg-neutral-950 dark:text-white" />
				</label>
			{/if}
			<a href="/app/kelas" use:inertia class={"rounded text-xs font-medium text-brand-700 hover:underline dark:text-brand-400 " + focusRing}>Lihat kelas</a>
		</div>
	</div>

	{#if !programs.length}
		<p class="px-5 py-10 text-sm text-neutral-500 sm:px-6 dark:text-neutral-400">{failed ? "Performa program belum berhasil dimuat." : "Belum ada program kelas tercatat."}</p>
	{:else}
		<div class="mt-4 overflow-x-auto">
			<table class="w-full min-w-[640px] text-sm">
				<thead>
					<tr class="border-y border-neutral-100 bg-neutral-50/60 dark:border-white/[0.05] dark:bg-white/[0.015]">
						{#each columns as c (c.key)}
							<th scope="col" class={"px-3 py-0 font-medium first:pl-5 last:pr-5 sm:first:pl-6 sm:last:pr-6 " + (c.numeric ? "text-right" : "text-left")} aria-sort={sortKey === c.key ? (ascending ? "ascending" : "descending") : "none"}>
								<button type="button" onclick={() => sortBy(c.key)} class={"inline-flex h-10 items-center gap-1 rounded text-xs text-neutral-500 hover:text-neutral-900 dark:text-neutral-400 dark:hover:text-white " + focusRing + (sortKey === c.key ? " text-neutral-900 dark:text-white" : "")}>
									{c.label}
									{#if sortKey === c.key}{#if ascending}<ArrowUp size={12} />{:else}<ArrowDown size={12} />{/if}{/if}
								</button>
							</th>
						{/each}
					</tr>
				</thead>
				<tbody class="divide-y divide-neutral-100 dark:divide-white/[0.05]">
					{#each rows as p (p.name)}
						{@const f = fill(p)}
						<tr>
							<th scope="row" class="py-3 pl-5 pr-3 text-left font-medium text-neutral-900 sm:pl-6 dark:text-white">{p.name}</th>
							<td class="px-3 py-3 text-right font-semibold tabular-nums text-neutral-900 dark:text-white">{count(p.active)}</td>
							<td class="px-3 py-3 text-right tabular-nums text-neutral-600 dark:text-neutral-300">{count(p.cuti)}</td>
							<td class="px-3 py-3 text-right tabular-nums text-neutral-600 dark:text-neutral-300">{count(p.stopped)}</td>
							<td class="px-3 py-3 text-right tabular-nums text-neutral-600 dark:text-neutral-300">{count(p.classes)}</td>
							<td class="py-3 pl-3 pr-5 sm:pr-6">
								<div class="ml-auto flex w-40 items-center gap-2">
									<div class="h-1.5 flex-1 overflow-hidden rounded-full bg-neutral-100 dark:bg-white/[0.06]">
										<div class={"h-full rounded-full " + (f >= 100 ? "bg-amber-500" : "bg-brand-600 dark:bg-brand-500")} style:width={`${Math.min(100, Math.max(0, f))}%`}></div>
									</div>
									<span class="w-20 text-right text-xs tabular-nums text-neutral-600 dark:text-neutral-300" title={`${count(p.occupied)} dari ${count(p.seats)} kursi`}>{f < 0 ? "Tanpa kursi" : `${count(p.occupied)}/${count(p.seats)}`}</span>
								</div>
							</td>
						</tr>
					{:else}
						<tr><td colspan="6" class="px-5 py-8 text-sm text-neutral-500 sm:px-6 dark:text-neutral-400">Tidak ada program dengan nama "{query}". <button type="button" class="font-medium text-brand-700 underline dark:text-brand-400" onclick={() => (query = "")}>Hapus pencarian</button></td></tr>
					{/each}
				</tbody>
			</table>
		</div>
		<p class="px-5 py-3 text-[11px] leading-relaxed text-neutral-500 sm:px-6 dark:text-neutral-400">Kursi terisi menghitung santri aktif dan cuti di kelas aktif. Berhenti = nonaktif + tidak lanjut saat ini, bukan jumlah yang keluar bulan ini.</p>
	{/if}
</section>
