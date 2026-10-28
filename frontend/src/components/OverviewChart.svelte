<script lang="ts">
  let {
    labels,
    series,
    money = false,
    title,
  }: {
    labels: string[];
    series: { name: string; values: number[] }[];
    money?: boolean;
    title: string;
  } = $props();
  let max = $derived(Math.max(1, ...series.flatMap((s) => s.values)));
  const number = (v: number) =>
    new Intl.NumberFormat(
      "id-ID",
      money ? { notation: "compact", maximumFractionDigits: 1 } : {},
    ).format(v);
  function points(values: number[]) {
    return values
      .map(
        (v, i) =>
          `${46 + i * (508 / Math.max(1, labels.length - 1))},${174 - (v / max) * 146}`,
      )
      .join(" ");
  }
</script>

{#if labels.length}
  <div class="chart-key">
    {#each series as s, i}<span><i class:muted={i > 0}></i>{s.name}</span
      >{/each}
  </div>
  <svg viewBox="0 0 580 215" role="img" aria-label={title}>
    <title>{title}. Angka lengkap tersedia pada tabel di bawah grafik.</title>
    {#each [0, 0.5, 1] as fraction}
      <line
        x1="46"
        y1={174 - fraction * 146}
        x2="554"
        y2={174 - fraction * 146}
        class="gridline"
      />
      <text x="36" y={178 - fraction * 146} text-anchor="end"
        >{number(max * fraction)}</text
      >
    {/each}
    {#each series as s, si}
      <polyline points={points(s.values)} class:secondary={si > 0} />
      {#each s.values as value, i}
        <circle
          cx={46 + i * (508 / Math.max(1, labels.length - 1))}
          cy={174 - (value / max) * 146}
          r="4"
          class:secondary={si > 0}
        >
          <title
            >{labels[i]}: {s.name}
            {money ? "Rp " : ""}{new Intl.NumberFormat("id-ID").format(
              value,
            )}</title
          >
        </circle>
      {/each}
    {/each}
    {#each labels as label, i}<text
        x={46 + i * (508 / Math.max(1, labels.length - 1))}
        y="202"
        text-anchor="middle">{label}</text
      >{/each}
  </svg>
  <details>
    <summary>Lihat angka</summary>
    <div class="scroll">
      <table>
        <thead
          ><tr
            ><th>Periode</th>{#each series as s}<th>{s.name}</th>{/each}</tr
          ></thead
        ><tbody
          >{#each labels as label, i}<tr
              ><td>{label}</td>{#each series as s}<td
                  >{money ? "Rp " : ""}{new Intl.NumberFormat("id-ID").format(
                    s.values[i] ?? 0,
                  )}</td
                >{/each}</tr
            >{/each}</tbody
        >
      </table>
    </div>
  </details>
{:else}
  <p class="empty">
    Tren belum tersedia. Periksa sumber data atau muat ulang dashboard.
  </p>
{/if}

<style>
  svg {
    width: 100%;
    display: block;
    overflow: visible;
    margin-top: 1rem;
  }
  text {
    font-size: 11px;
    fill: var(--color-neutral-500);
    font-family: inherit;
  }
  .gridline {
    stroke: var(--color-neutral-200);
    stroke-dasharray: 3 5;
  }
  :global(.dark) .gridline {
    stroke: var(--color-neutral-800);
  }
  polyline {
    fill: none;
    stroke: var(--color-brand-600);
    stroke-width: 2.5;
    stroke-linejoin: round;
  }
  circle {
    fill: var(--color-brand-600);
    stroke: var(--color-neutral-50);
    stroke-width: 2;
  }
  :global(.dark) circle {
    stroke: var(--color-neutral-925);
  }
  polyline.secondary {
    stroke: var(--color-neutral-400);
    stroke-dasharray: 5 4;
  }
  circle.secondary {
    fill: var(--color-neutral-400);
  }
  .chart-key {
    display: flex;
    gap: 1rem;
    flex-wrap: wrap;
    font-size: 0.75rem;
    color: var(--color-neutral-500);
  }
  .chart-key span {
    display: flex;
    align-items: center;
    gap: 0.4rem;
  }
  i {
    width: 0.65rem;
    height: 0.65rem;
    background: var(--color-brand-600);
    border-radius: 2px;
  }
  i.muted {
    background: var(--color-neutral-400);
  }
  details {
    margin-top: 0.5rem;
    font-size: 0.75rem;
    color: var(--color-neutral-500);
  }
  summary {
    width: fit-content;
    padding: 0.3rem 0;
  }
  .scroll {
    overflow-x: auto;
  }
  table {
    width: 100%;
    text-align: left;
  }
  th,
  td {
    padding: 0.5rem;
    white-space: nowrap;
  }
  .empty {
    padding: 3rem 0;
    text-align: center;
    color: var(--color-neutral-500);
    font-size: 0.875rem;
  }
</style>
