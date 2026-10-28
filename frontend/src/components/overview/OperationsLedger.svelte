<script lang="ts">
	import { inertia } from "@inertiajs/svelte";
	import type { Overview } from "@lib/overview";
	import { count, focusRing, kicker, panel } from "./ui";

	let { ops, riayah, products, monthLabel }: {
		ops: Overview["operations"];
		riayah: Overview["riayah"];
		products: Overview["products"];
		monthLabel: string;
	} = $props();

	let groups = $derived([
		{
			title: "Pelaksanaan kelas",
			note: monthLabel,
			href: "/app/koordinator-guru",
			link: "Monitoring guru",
			rows: [
				{ label: "Pertemuan selesai", value: ops?.finished },
				{ label: "Dengan guru pengganti", value: ops?.badal },
				{ label: "Dengan reschedule", value: ops?.rescheduled },
			],
		},
		{
			title: "Pendampingan santri",
			note: "Penanda riayah saat ini; satu santri bisa punya beberapa",
			href: "/app/guru/riayah",
			link: "Tindak lanjut riayah",
			rows: [
				{ label: "Kehadiran perlu perhatian", value: riayah?.kehadiran, warn: true },
				{ label: "Kontak perlu ditindaklanjuti", value: riayah?.kontak, warn: true },
				{ label: "Progres perlu perhatian", value: riayah?.progres, warn: true },
				{ label: `Rapor ${riayah?.rapor_periode ?? ""} terkirim`, value: riayah?.rapor_terkirim },
			],
		},
		{
			title: "Produk dan bahan belajar",
			note: "Persediaan saat ini; kepemilikan bukan berarti penjualan",
			href: "/app/produk-crm",
			link: "Produk dan CRM",
			rows: [
				{ label: "Produk aktif", value: products?.active_products },
				{ label: "Total stok tercatat", value: products?.total_stock },
				{ label: "Stok minimum / habis", value: products?.low_stock_products, warn: true },
				{ label: "Santri memiliki produk", value: products?.santri_with_product },
			],
		},
	]);
</script>

<section class={panel + " grid min-w-0 grid-cols-1 divide-y divide-neutral-100 md:grid-cols-3 md:divide-x md:divide-y-0 dark:divide-white/[0.05]"} aria-label="Operasional dan dukungan pembelajaran">
	{#each groups as g (g.title)}
		<div class="flex min-w-0 flex-col px-5 py-5 sm:px-6">
			<h2 class={kicker}>{g.title}</h2>
			<p class="mt-1 text-xs text-neutral-500 dark:text-neutral-400">{g.note}</p>
			<dl class="mt-3 flex-1 space-y-2">
				{#each g.rows as r (r.label)}
					<div class="flex items-baseline justify-between gap-3 text-sm">
						<dt class="text-neutral-600 dark:text-neutral-300">{r.label}</dt>
						<dd class={"font-semibold tabular-nums " + ("warn" in r && r.warn && (r.value ?? 0) > 0 ? "text-amber-700 dark:text-amber-400" : "text-neutral-900 dark:text-white")}>{count(r.value)}</dd>
					</div>
				{/each}
			</dl>
			<a href={g.href} use:inertia class={"mt-4 w-fit rounded text-xs font-medium text-brand-700 hover:underline dark:text-brand-400 " + focusRing}>{g.link}</a>
		</div>
	{/each}
</section>
