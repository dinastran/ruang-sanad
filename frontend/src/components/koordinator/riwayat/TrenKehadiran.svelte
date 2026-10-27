<script lang="ts">
	import { LABEL_STATUS, STATUS_KEHADIRAN, WARNA_STATUS, formatPersen, type BucketTren } from "@lib/riwayatAbsensi";

	interface Props {
		buckets: BucketTren[];
		mingguan: boolean;
	}

	let { buckets, mingguan }: Props = $props();

</script>

<section class="flex h-full min-w-0 flex-col rounded-2xl border border-neutral-200/80 bg-white p-5 dark:border-white/[0.06] dark:bg-neutral-925/50">
	<div class="flex flex-wrap items-start justify-between gap-3">
		<div>
			<h2 class="text-sm font-semibold text-neutral-900 dark:text-white">Tren kehadiran</h2>
			<p class="mt-0.5 text-xs text-neutral-500">Komposisi status absen pembinaan &amp; rapat per {mingguan ? "pekan" : "bulan"}<span class="hidden sm:inline">; angka = persentase hadir</span></p>
		</div>
		<ul class="flex flex-wrap gap-x-3 gap-y-1 text-xs text-neutral-600 dark:text-neutral-400" aria-label="Keterangan warna">
			{#each STATUS_KEHADIRAN as s (s)}
				<li class="inline-flex items-center gap-1.5"><span class="h-2.5 w-2.5 rounded-sm {WARNA_STATUS[s].bar}"></span>{LABEL_STATUS[s]}</li>
			{/each}
		</ul>
	</div>

	{#if buckets.length === 0}
		<div class="flex min-h-48 flex-1 items-center justify-center text-sm text-neutral-500">Belum ada catatan absen pada filter ini.</div>
	{:else}
		<div class="mt-5 flex min-h-56 flex-1 items-stretch gap-1.5 sm:gap-2.5" role="list">
			{#each buckets as b, i (b.key)}
				<div class="flex h-full min-w-0 flex-1 flex-col items-center gap-1.5" role="listitem" aria-label="{b.label}: hadir {b.rekap.hadir}, izin {b.rekap.izin}, sakit {b.rekap.sakit}, alfa {b.rekap.alfa}">
					<div class="flex w-full flex-1 flex-col items-center justify-end gap-1">
						<span class="hidden text-[11px] font-semibold tabular-nums text-neutral-700 sm:block dark:text-neutral-300">{formatPersen(b.rekap.persen === null ? null : Math.round(b.rekap.persen))}</span>
						<div
							class="flex w-full max-w-12 flex-col-reverse overflow-hidden rounded-md"
							style="height: calc(100% - 1.25rem)"
							title="{b.label} · {b.rekap.total} catatan · hadir {b.rekap.hadir}, izin {b.rekap.izin}, sakit {b.rekap.sakit}, alfa {b.rekap.alfa}"
						>
							{#each STATUS_KEHADIRAN as s (s)}
								{#if b.rekap[s] > 0}<div class="{WARNA_STATUS[s].bar} w-full" style="height: {(b.rekap[s] / b.rekap.total) * 100}%"></div>{/if}
							{/each}
						</div>
					</div>
					<!-- Di layar sempit hanya tiap label ketiga (dan terakhir) yang tampil agar tidak terpotong. -->
					<span class="w-full whitespace-nowrap text-center text-[11px] text-neutral-500 {i % 3 === 0 || i === buckets.length - 1 ? '' : 'max-sm:invisible'}">{b.label}</span>
				</div>
			{/each}
		</div>
	{/if}
</section>
