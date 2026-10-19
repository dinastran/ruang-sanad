<script lang="ts">
  import { inertia, router } from "@inertiajs/svelte";
  import AppLayout from "@layouts/AppLayout.svelte";
  import OverviewChart from "@components/OverviewChart.svelte";
  import type { User } from "@lib/types";
  import type { Overview } from "@lib/overview";

  let { user, overview }: { user?: User; overview?: Overview } = $props();
  let loading = $state(false);
  let loadError = $state("");
  let data = $derived(overview);
  let management = $derived(user?.role === "super_admin");
  let financeAllowed = $derived(
    ["super_admin", "keuangan", "admin"].includes(user?.role ?? ""),
  );
  let finance = $derived(financeAllowed ? data?.finance : undefined);
  let summary = $derived(management ? data?.santri : undefined);
  let ops = $derived(management ? data?.operations : undefined);
  let riayah = $derived(management ? data?.riayah : undefined);
  let products = $derived(management ? data?.products : undefined);
  let priorities = $derived(data?.priorities ?? []);
  let programs = $derived(data?.programs ?? []);
  let statusRows = $derived(
    summary
      ? [
          {
            label: "Aktif",
            value: summary.aktif,
            color: "var(--color-brand-600)",
          },
          { label: "Cuti", value: summary.cuti, color: "var(--color-warning)" },
          {
            label: "Nonaktif",
            value: summary.nonaktif,
            color: "var(--color-neutral-400)",
          },
          {
            label: "Tidak lanjut",
            value: summary.tidak_lanjut,
            color: "var(--color-neutral-600)",
          },
          {
            label: "Status lainnya",
            value: Math.max(
              0,
              summary.total -
                summary.aktif -
                summary.cuti -
                summary.nonaktif -
                summary.tidak_lanjut,
            ),
            color: "var(--color-neutral-200)",
          },
        ].filter((s) => s.value > 0)
      : [],
  );
  let registered = $derived(data?.growth?.at(-1)?.total);
  let previousRegistered = $derived(data?.growth?.at(-2)?.total);
  let sourceErrors = $derived(data?.errors ?? []);
  let maxMonth = $derived(data?.updated_at?.slice(0, 7) ?? "");
  const count = (n: number | undefined) =>
    n === undefined
      ? "Belum tersedia"
      : new Intl.NumberFormat("id-ID").format(n);
  const money = (n: number | undefined) =>
    n === undefined
      ? "Belum tersedia"
      : new Intl.NumberFormat("id-ID", {
          style: "currency",
          currency: "IDR",
          maximumFractionDigits: 0,
        }).format(n);
  const percent = (n: number) =>
    new Intl.NumberFormat("id-ID", { maximumFractionDigits: 1 }).format(n) +
    "%";
  const monthLabel = (m: string) =>
    new Intl.DateTimeFormat("id-ID", {
      month: "short",
      year: "2-digit",
      timeZone: "Asia/Jakarta",
    }).format(new Date(`${m}-01T00:00:00+07:00`));
  const refreshed = (iso: string) =>
    new Intl.DateTimeFormat("id-ID", {
      day: "numeric",
      month: "short",
      hour: "2-digit",
      minute: "2-digit",
      timeZone: "Asia/Jakarta",
    }).format(new Date(iso)) + " WIB";
  function visit(month: string) {
    if (loading) return;
    loadError = "";
    router.get("/app", month ? { bulan: month } : {}, {
      preserveScroll: true,
      preserveState: true,
      onStart: () => {
        loading = true;
      },
      onFinish: () => {
        loading = false;
      },
      onError: () => {
        loadError = "Dashboard belum berhasil diperbarui. Coba muat ulang.";
      },
      onNetworkError: () => {
        loadError =
          "Koneksi terputus. Data sebelumnya tetap ditampilkan. Coba perbarui lagi.";
        return false;
      },
      onHttpException: () => {
        loadError =
          "Server belum berhasil memuat dashboard. Coba perbarui lagi.";
        return false;
      },
    });
  }
  function shiftMonth(offset: number) {
    if (!data) return;
    const [year, month] = data.bulan.split("-").map(Number);
    const next = new Date(Date.UTC(year, month - 1 + offset, 1))
      .toISOString()
      .slice(0, 7);
    visit(next);
  }
  function arcOffset(index: number) {
    const total = summary?.total ?? 0;
    return total
      ? (-statusRows.slice(0, index).reduce((a, s) => a + s.value, 0) / total) *
          100
      : 0;
  }
</script>

{#snippet metric(label: string, value: string, detail: string, href: string)}
  <a {href} use:inertia class="metric">
    <span class="metric-label">{label}<span aria-hidden="true">↗</span></span>
    <strong>{value}</strong><span class="caption">{detail}</span>
  </a>
{/snippet}

<svelte:head><title>Ringkasan Ruang Sanad</title></svelte:head>
<AppLayout {user}>
  <div class="overview" aria-busy={loading}>
    <header class="page-header">
      <div>
        <p class="eyebrow">RUANG SANAD / RINGKASAN</p>
        <h1>Kondisi hari ini.<br /><span>Langkah berikutnya.</span></h1>
        <p class="subtitle">
          {management
            ? "Kelas, santri, guru, dan keuangan dalam satu pandangan."
            : "Ringkasan pembayaran, tunggakan, dan arus kas yang tercatat."}
        </p>
      </div>
      <div class="header-controls">
        <label for="overview-month">Periode aktivitas</label>
        <div class="month-control">
          <button
            aria-label="Bulan sebelumnya"
            disabled={loading || data?.bulan === "2000-01"}
            onclick={() => shiftMonth(-1)}>‹</button
          >
          <input
            id="overview-month"
            aria-label="Pilih bulan aktivitas"
            type="month"
            value={data?.bulan ?? ""}
            min="2000-01"
            max={maxMonth}
            disabled={loading}
            onchange={(e) => {
              if (e.currentTarget.validity.valid && e.currentTarget.value)
                visit(e.currentTarget.value);
            }}
          />
          <button
            aria-label="Bulan berikutnya"
            disabled={loading || !data || data.bulan >= maxMonth}
            onclick={() => shiftMonth(1)}>›</button
          >
        </div>
        <button
          class="refresh"
          disabled={loading}
          onclick={() => visit(data?.bulan ?? "")}
          >{loading ? "Memperbarui..." : "Perbarui data"}</button
        >
        {#if data}<p class="updated">
            Diperbarui {refreshed(data.updated_at)}
          </p>{/if}
      </div>
    </header>

    <div class="period-note">
      <span class="live-dot"></span>
      <p>
        Status santri, kelas, riayah, stok, dan piutang adalah kondisi terkini.
        Aktivitas dan grafik mengikuti <strong
          >{data?.bulan_label ?? "periode terpilih"}</strong
        >{data?.current_month ? " (bulan berjalan belum selesai)" : ""}.
      </p>
    </div>
    {#if loadError}<div class="notice" role="alert">{loadError}</div>{/if}
    {#if sourceErrors.length}<div class="notice" role="alert">
        <strong>Sebagian data belum tersedia.</strong>
        {sourceErrors.join(", ")} gagal dimuat. Angka pada bagian tersebut tidak dianggap
        nol.
        <button onclick={() => visit(data?.bulan ?? "")} disabled={loading}
          >Coba lagi</button
        >
      </div>{/if}
    {#if !data}<div class="notice" role="alert">
        Ringkasan belum tersedia. <button onclick={() => visit("")}
          >Muat dashboard</button
        >
      </div>{/if}
    <div class:pending={loading}>
      <section class="metrics" aria-label="Indikator utama">
        {#if management}
          {@render metric(
            "Santri aktif",
            count(summary?.aktif),
            summary
              ? `${count(summary.total)} santri tercatat, seluruh status`
              : "Sumber status santri belum tersedia",
            "/app/analitik-santri",
          )}
          {@render metric(
            "Kelas aktif",
            count(ops?.classes),
            ops
              ? `${count(ops.teachers)} guru aktif`
              : "Sumber operasional belum tersedia",
            "/app/kelas",
          )}
          {@render metric(
            "Perlu perhatian",
            count(riayah?.perlu_perhatian),
            riayah
              ? `Dari ${count(riayah.total_santri)} santri dalam cakupan riayah`
              : "Sumber riayah belum tersedia",
            "/app/guru/riayah",
          )}
        {/if}
        {#if financeAllowed}
          {@render metric(
            "Penerimaan tercatat",
            money(finance?.sekarang.kas_masuk),
            "SPP dibayar + penjualan produk, periode terpilih",
            `/app/keuangan?bulan=${data?.bulan ?? ""}`,
          )}
          {@render metric(
            "Piutang saat ini",
            money(finance?.total_piutang),
            finance
              ? `${count(finance.jumlah_terlambat)} tagihan lewat jatuh tempo`
              : "Sumber keuangan belum tersedia",
            "/app/keuangan/tagihan",
          )}
          {@render metric(
            "Tagihan terbayar",
            finance && finance.sekarang.terbit_nominal > 0
              ? percent(finance.sekarang.kolektibilitas)
              : finance
                ? "Belum ada tagihan"
                : "Belum tersedia",
            "Nominal lunas / nominal terbit pada periode terpilih",
            `/app/keuangan?bulan=${data?.bulan ?? ""}`,
          )}
        {/if}
      </section>

      <section class="priority-layout">
        <div class="panel priority-panel">
          <div class="section-title">
            <div>
              <h2>Yang perlu ditangani</h2>
              <p>
                Masalah operasional lebih dulu, lalu tindak lanjut pendukung.
              </p>
            </div>
            <span class="tag" class:attention={priorities.length > 0}
              >{sourceErrors.length
                ? "Data belum lengkap"
                : `${priorities.length} area perhatian`}</span
            >
          </div>
          {#if priorities.length}
            <div class="priority-list">
              {#each priorities as item, i}<a
                  href={item.href}
                  use:inertia
                  class="priority-item"
                  ><span class="priority-index"
                    >{String(i + 1).padStart(2, "0")}</span
                  >
                  <div>
                    <h3>{item.title}</h3>
                    <p>{item.detail}</p>
                  </div>
                  <span class="priority-count"
                    >{count(item.count)}<span aria-hidden="true">↗</span></span
                  ></a
                >{/each}
            </div>
          {:else}<div class="empty-state">
              <h3>
                {sourceErrors.length
                  ? "Periksa sumber data yang belum tersedia"
                  : "Tidak ada penanda prioritas dari data yang dimuat"}
              </h3>
              <p>
                {sourceErrors.length
                  ? "Daftar perhatian belum dapat dinilai lengkap."
                  : "Tetap tinjau riayah, pencatatan kelas, dan laporan tim secara berkala."}
              </p>
            </div>{/if}
        </div>
        <aside class="panel analysis-panel">
          <div class="section-title">
            <div>
              <h2>Catatan untuk keputusan</h2>
              <p>Otomatis dari data yang tersedia.</p>
            </div>
          </div>
          <div class="analysis-list">
            {#if summary}<article>
                <span>STATUS SANTRI</span>
                <p>
                  {summary.total > 0
                    ? `${percent(summary.active_rate)} dari seluruh santri tercatat berstatus aktif.`
                    : "Belum ada santri tercatat."}
                  {summary.cuti > 0
                    ? `${count(summary.cuti)} santri sedang cuti, perlu dipantau rencana kembalinya.`
                    : ""}
                </p>
                <small>Proporsi status saat ini, bukan angka retention.</small>
              </article>{/if}
            {#if registered !== undefined && previousRegistered !== undefined}<article
              >
                <span>PENDAFTARAN</span>
                <p>
                  {count(registered)} santri terdaftar pada periode terpilih, {registered >=
                  previousRegistered
                    ? "bertambah"
                    : "berkurang"}
                  {count(Math.abs(registered - previousRegistered))} dibanding bulan
                  sebelumnya.
                </p>
                <small
                  >{data?.current_month
                    ? "Bulan berjalan belum selesai. Pembanding adalah satu bulan penuh."
                    : "Berdasarkan tanggal daftar, bukan tanggal pembayaran."}</small
                >
              </article>{/if}
            {#if finance}<article>
                <span>ARUS KAS TERCATAT</span>
                <p>
                  {finance.sekarang.kas_bersih >= 0
                    ? "Selisih positif"
                    : "Selisih negatif"}
                  {money(Math.abs(finance.sekarang.kas_bersih))} setelah penerimaan
                  dikurangi ujroh yang dibayar.
                </p>
                <small
                  >Belum mencakup seluruh biaya perusahaan. Ini bukan laba
                  bersih.</small
                >
              </article>{/if}
            {#if !summary && !finance}<p class="caption">
                Analisis menunggu sumber data berhasil dimuat.
              </p>{/if}
          </div>
        </aside>
      </section>

      {#if management}
        <section class="two-columns">
          <div class="panel">
            <div class="section-title">
              <div>
                <h2>Tren pendaftaran santri</h2>
                <p>Enam bulan terakhir sampai periode terpilih.</p>
              </div>
              <a href="/app/analitik-santri" use:inertia class="text-link"
                >Analitik ↗</a
              >
            </div>
            <OverviewChart
              title="Jumlah pendaftaran santri per bulan"
              labels={(data?.growth ?? []).map((g) => monthLabel(g.month))}
              series={[
                {
                  name: "Santri terdaftar",
                  values: (data?.growth ?? []).map((g) => g.total),
                },
              ]}
            />
            <p class="chart-note">
              Bulan tanpa pendaftaran ditampilkan sebagai nol. Data tanpa
              tanggal daftar tidak masuk tren.
            </p>
          </div>
          <div class="panel">
            <div class="section-title">
              <div>
                <h2>Komposisi status santri</h2>
                <p>Kondisi seluruh santri saat data dimuat.</p>
              </div>
            </div>
            {#if summary && summary.total > 0}<div class="status-composition">
                <div class="donut">
                  <svg
                    viewBox="0 0 160 160"
                    role="img"
                    aria-label={`Proporsi aktif ${percent(summary.active_rate)} dari ${count(summary.total)} santri`}
                    ><circle
                      class="donut-base"
                      cx="80"
                      cy="80"
                      r="62"
                    />{#each statusRows as s, i}<circle
                        cx="80"
                        cy="80"
                        r="62"
                        pathLength="100"
                        fill="none"
                        stroke={s.color}
                        stroke-width="15"
                        stroke-dasharray={`${(s.value / summary.total) * 100} ${100 - (s.value / summary.total) * 100}`}
                        stroke-dashoffset={arcOffset(i)}
                        transform="rotate(-90 80 80)"
                        ><title>{s.label}: {count(s.value)}</title></circle
                      >{/each}</svg
                  >
                  <div>
                    <strong>{percent(summary.active_rate)}</strong><span
                      >berstatus aktif</span
                    >
                  </div>
                </div>
                <dl class="status-legend">
                  {#each statusRows as s}<div>
                      <dt><i style:background={s.color}></i>{s.label}</dt>
                      <dd>{count(s.value)}</dd>
                    </div>{/each}
                </dl>
              </div>{:else}<div class="empty-state">
                {summary
                  ? "Belum ada santri. Tambahkan data melalui menu Santri."
                  : "Status santri belum tersedia. Coba perbarui data."}
              </div>{/if}
          </div>
        </section>

        <section class="panel">
          <div class="section-title">
            <div>
              <h2>Kondisi setiap program kelas</h2>
              <p>
                Dikelompokkan berdasarkan tipe. Status dan kursi adalah kondisi
                terkini.
              </p>
            </div>
            <a class="text-link" href="/app/kelas" use:inertia>Lihat kelas ↗</a>
          </div>
          {#if programs.length}<div class="table-scroll">
              <table class="program-table">
                <thead
                  ><tr
                    ><th>Program / tipe</th><th>Santri aktif</th><th>Cuti</th
                    ><th>Nonaktif + tidak lanjut</th><th>Kelas aktif</th><th
                      >Keterisian kursi</th
                    ></tr
                  ></thead
                ><tbody
                  >{#each programs as p}<tr
                      ><th scope="row">{p.name}</th><td class="strong"
                        >{count(p.active)}</td
                      ><td>{count(p.cuti)}</td><td>{count(p.stopped)}</td><td
                        >{count(p.classes)}</td
                      ><td
                        ><div class="seat-value">
                          <span>{count(p.occupied)} / {count(p.seats)}</span
                          ><strong
                            >{p.seats > 0
                              ? percent((p.occupied / p.seats) * 100)
                              : "Belum ada kapasitas"}</strong
                          >
                        </div>
                        <div class="track">
                          <div
                            style:width={`${Math.min(100, p.seats > 0 ? (p.occupied / p.seats) * 100 : 0)}%`}
                          ></div>
                        </div></td
                      ></tr
                    >{/each}</tbody
                >
              </table>
            </div>
            <p class="chart-note">
              Kursi terisi menghitung santri aktif dan cuti di kelas aktif
              sesuai aturan kelas. Nonaktif dan tidak lanjut adalah status saat
              ini, bukan jumlah yang keluar bulan ini.
            </p>{:else}<div class="empty-state">
              {sourceErrors.includes("Performa program")
                ? "Performa program belum berhasil dimuat."
                : "Belum ada program kelas tercatat."}
            </div>{/if}
        </section>

        <section
          class="three-columns"
          aria-label="Operasional dan dukungan pembelajaran"
        >
          <div class="panel">
            <div class="section-title">
              <div>
                <h2>Pelaksanaan kelas</h2>
                <p>{data?.bulan_label}, pertemuan selesai tercatat.</p>
              </div>
            </div>
            <strong class="section-number">{count(ops?.finished)}</strong>
            <dl class="detail-list">
              <div>
                <dt>Dengan guru pengganti</dt>
                <dd>{count(ops?.badal)}</dd>
              </div>
              <div>
                <dt>Dengan reschedule</dt>
                <dd>{count(ops?.rescheduled)}</dd>
              </div>
              <div>
                <dt>Pertemuan lama belum ditutup</dt>
                <dd>{count(ops?.open_meetings)}</dd>
              </div>
            </dl>
            <a class="text-link" href="/app/koordinator-guru" use:inertia
              >Monitoring guru ↗</a
            >
          </div>
          <div class="panel">
            <div class="section-title">
              <div>
                <h2>Pendampingan santri</h2>
                <p>Penanda riayah saat ini.</p>
              </div>
            </div>
            <dl class="detail-list riayah-details">
              <div>
                <dt>Kehadiran perlu perhatian</dt>
                <dd>{count(riayah?.kehadiran)}</dd>
              </div>
              <div>
                <dt>Kontak perlu ditindaklanjuti</dt>
                <dd>{count(riayah?.kontak)}</dd>
              </div>
              <div>
                <dt>Progres perlu perhatian</dt>
                <dd>{count(riayah?.progres)}</dd>
              </div>
              <div>
                <dt>Rapor {riayah?.rapor_periode ?? ""} terkirim</dt>
                <dd>{count(riayah?.rapor_terkirim)}</dd>
              </div>
            </dl>
            <p class="chart-note">
              Satu santri bisa memiliki beberapa penanda.
            </p>
            <a class="text-link" href="/app/guru/riayah" use:inertia
              >Tindak lanjut riayah ↗</a
            >
          </div>
          <div class="panel">
            <div class="section-title">
              <div>
                <h2>Produk dan bahan belajar</h2>
                <p>Persediaan dan kepemilikan saat ini.</p>
              </div>
            </div>
            <dl class="detail-list riayah-details">
              <div>
                <dt>Produk aktif</dt>
                <dd>{count(products?.active_products)}</dd>
              </div>
              <div>
                <dt>Total stok tercatat</dt>
                <dd>{count(products?.total_stock)}</dd>
              </div>
              <div>
                <dt>Produk stok minimum / habis</dt>
                <dd>{count(products?.low_stock_products)}</dd>
              </div>
              <div>
                <dt>Santri memiliki produk</dt>
                <dd>{count(products?.santri_with_product)}</dd>
              </div>
            </dl>
            <p class="chart-note">
              Kepemilikan produk tidak otomatis berarti penjualan.
            </p>
            <a class="text-link" href="/app/produk-crm" use:inertia
              >Produk dan CRM ↗</a
            >
          </div>
        </section>
      {/if}

      {#if financeAllowed}
        <section class="finance-section">
          <div class="finance-heading">
            <div>
              <h2>Keuangan yang tercatat</h2>
              <p>
                Pisahkan penerimaan, kewajiban, dan piutang sebelum mengambil
                keputusan.
              </p>
            </div>
            <a
              href={`/app/keuangan?bulan=${data?.bulan ?? ""}`}
              use:inertia
              class="text-link">Dashboard keuangan ↗</a
            >
          </div>
          {#if finance}<div class="two-columns">
              <div class="panel">
                <div class="section-title">
                  <div>
                    <h3>Penerimaan dan ujroh dibayar</h3>
                    <p>Enam bulan, berdasarkan tanggal transaksi.</p>
                  </div>
                </div>
                <OverviewChart
                  title="Tren penerimaan dan ujroh dibayar"
                  money
                  labels={finance.tren.map((t) => t.label)}
                  series={[
                    {
                      name: "Penerimaan",
                      values: finance.tren.map((t) => t.kas_masuk),
                    },
                    {
                      name: "Ujroh dibayar",
                      values: finance.tren.map((t) => t.kas_keluar),
                    },
                  ]}
                />
              </div>
              <div class="panel">
                <div class="section-title">
                  <div>
                    <h3>Umur piutang saat ini</h3>
                    <p>
                      Seluruh tagihan belum bayar, termasuk periode terdahulu.
                    </p>
                  </div>
                </div>
                <div class="aging-list">
                  {#each finance.aging as b}<div>
                      <div class="aging-label">
                        <span>{b.label}</span><strong>{money(b.nominal)}</strong
                        >
                      </div>
                      <div class="track">
                        <div
                          style:width={`${finance.total_piutang > 0 ? (b.nominal / finance.total_piutang) * 100 : 0}%`}
                        ></div>
                      </div>
                      <small>{count(b.jumlah)} tagihan</small>
                    </div>{/each}
                </div>
              </div>
            </div>
            <div class="finance-ledger">
              <div>
                <span>SPP diterima</span><strong
                  >{money(finance.sekarang.kas_masuk_spp)}</strong
                >
              </div>
              <div>
                <span>Penjualan produk</span><strong
                  >{money(finance.sekarang.kas_masuk_produk)}</strong
                >
              </div>
              <div>
                <span>Ujroh dibayar</span><strong
                  >{money(finance.sekarang.kas_keluar)}</strong
                >
              </div>
              <div>
                <span>Kewajiban ujroh terkunci, seluruh bulan</span><strong
                  >{money(finance.ujroh.kewajiban_terkunci)}</strong
                >
              </div>
              <div>
                <span>Estimasi ujroh bulan berjalan</span><strong
                  >{money(finance.ujroh.estimasi_berjalan)}</strong
                >
              </div>
            </div>
          {:else}<div class="panel empty-state">
              Keuangan belum tersedia. Coba perbarui data atau buka menu
              Keuangan.
            </div>{/if}
        </section>
      {/if}

      <details class="definitions">
        <summary>Cakupan data dan cara membaca dashboard</summary>
        <div>
          <p>
            Dashboard membaca data tersimpan saat halaman dimuat. Gunakan
            Perbarui data untuk mengambil kondisi terbaru. Pergantian bulan
            hanya mengubah aktivitas dan tren, bukan snapshot saat ini.
          </p>
          {#if management}<p>
              Belum tersedia ringkasan komplain terstruktur dan pengukuran
              retention antarperiode. Daftar perhatian memakai penanda riayah,
              penugasan kelas, tindak lanjut kunjungan, dan tugas koordinator
              yang sudah tercatat. Keberadaan penanda bukan bukti santri pasti
              berhenti.
            </p>{/if}
          <p>
            Penerimaan mencakup SPP berstatus lunas berdasarkan tanggal bayar
            serta order produk aktif sesuai pencatatan menu Keuangan. Tagihan
            batal dan order dibatalkan tidak dihitung. Selisih penerimaan dengan
            ujroh dibayar tidak mencakup seluruh biaya perusahaan.
          </p>
          <p>
            Persentase tagihan terbayar memakai nominal tagihan terbit pada
            bulan terpilih yang kini lunas, bukan hanya pembayaran pada bulan
            itu. Hak akses mengikuti menu sumber di server.
          </p>
        </div>
      </details>
    </div>
    <p class="footer-note">
      Ruang Sanad · Ringkasan untuk memantau dan menindaklanjuti.
    </p>
  </div>
</AppLayout>

<style>
  .overview {
    max-width: 1440px;
    margin: 0 auto;
    padding: 2.5rem clamp(1rem, 3vw, 2.5rem);
    color: var(--color-neutral-800);
  }
  :global(.dark) .overview {
    color: var(--color-neutral-200);
  }
  .page-header {
    display: flex;
    justify-content: space-between;
    gap: 2rem;
    align-items: center;
    margin-bottom: 1.75rem;
  }
  .eyebrow {
    font-size: 0.67rem;
    font-weight: 700;
    letter-spacing: 0.16em;
    color: var(--color-brand-700);
    margin: 0 0 0.8rem;
  }
  :global(.dark) .eyebrow {
    color: var(--color-brand-400);
  }
  h1 {
    font-size: clamp(1.9rem, 3vw, 2.65rem);
    line-height: 1.12;
    letter-spacing: -0.045em;
    font-weight: 650;
    margin: 0;
    color: var(--color-neutral-900);
  }
  :global(.dark) h1 {
    color: var(--color-neutral-50);
  }
  h1 span {
    color: var(--color-neutral-500);
  }
  .subtitle {
    font-size: 0.875rem;
    color: var(--color-neutral-500);
    margin-top: 0.8rem;
  }
  .header-controls {
    display: flex;
    flex-direction: column;
    align-items: flex-end;
    gap: 0.5rem;
    flex-shrink: 0;
  }
  .header-controls label,
  .updated {
    font-size: 0.7rem;
    color: var(--color-neutral-500);
  }
  .month-control {
    display: flex;
    border: 1px solid var(--color-neutral-300);
    border-radius: 0.6rem;
    overflow: hidden;
    align-items: center;
    background: var(--color-neutral-50);
  }
  .month-control button {
    padding: 0.4rem 0.8rem;
    font-size: 1.3rem;
  }
  .month-control input {
    padding: 0.5rem 0.2rem;
    font: inherit;
    font-size: 0.85rem;
    background: transparent;
    min-width: 9rem;
    border: 0;
    color: inherit;
  }
  :global(.dark) .month-control {
    border-color: var(--color-neutral-700);
    background: var(--color-neutral-900);
    color-scheme: dark;
  }
  button:disabled {
    opacity: 0.4;
  }
  .refresh {
    font-size: 0.75rem;
    font-weight: 600;
    color: var(--color-brand-700);
    padding: 0.25rem 0;
  }
  :global(.dark) .refresh {
    color: var(--color-brand-400);
  }
  .period-note {
    display: flex;
    align-items: center;
    gap: 0.65rem;
    margin-bottom: 1.25rem;
    font-size: 0.75rem;
    color: var(--color-neutral-500);
    line-height: 1.6;
  }
  .live-dot {
    width: 0.45rem;
    height: 0.45rem;
    background: var(--color-brand-600);
    border-radius: 50%;
    flex-shrink: 0;
  }
  .notice {
    border: 1px solid var(--color-warning);
    padding: 0.9rem 1rem;
    border-radius: 0.65rem;
    margin-bottom: 1rem;
    font-size: 0.85rem;
  }
  .notice button {
    text-decoration: underline;
    margin-left: 0.5rem;
  }
  .metrics {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 1px;
    background: var(--color-neutral-200);
    border: 1px solid var(--color-neutral-200);
    border-radius: 0.85rem;
    overflow: hidden;
    margin-bottom: 1.5rem;
  }
  .metric {
    background: white;
    padding: 1.25rem 1.4rem;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    transition: background 0.15s;
    min-width: 0;
  }
  .metric:hover {
    background: var(--color-brand-50);
  }
  :global(.dark) .metric {
    background: var(--color-neutral-925);
  }
  :global(.dark) .metric:hover {
    background: var(--color-neutral-900);
  }
  :global(.dark) .metrics {
    background: var(--color-neutral-800);
    border-color: var(--color-neutral-800);
  }
  .metric-label {
    display: flex;
    justify-content: space-between;
    gap: 0.5rem;
    font-size: 0.8rem;
    color: var(--color-neutral-500);
    font-weight: 500;
  }
  .metric-label span {
    color: var(--color-neutral-400);
  }
  .metric strong {
    font-size: clamp(1.25rem, 1.9vw, 1.85rem);
    line-height: 1.2;
    font-weight: 650;
    letter-spacing: -0.04em;
    overflow-wrap: anywhere;
    font-variant-numeric: tabular-nums;
  }
  .caption {
    font-size: 0.7rem;
    color: var(--color-neutral-500);
    line-height: 1.6;
  }
  .priority-layout {
    display: grid;
    grid-template-columns: minmax(0, 1.7fr) minmax(0, 1fr);
    gap: 1.25rem;
    margin-bottom: 1.25rem;
    align-items: start;
  }
  .panel {
    padding: 1.4rem;
    border: 1px solid var(--color-neutral-200);
    border-radius: 0.85rem;
    background: white;
    min-width: 0;
  }
  :global(.dark) .panel {
    background: var(--color-neutral-925);
    border-color: var(--color-neutral-800);
  }
  .section-title {
    display: flex;
    align-items: start;
    justify-content: space-between;
    gap: 1rem;
    margin-bottom: 1.2rem;
  }
  h2,
  h3 {
    font-weight: 600;
    letter-spacing: -0.02em;
  }
  h2 {
    font-size: 1rem;
  }
  h3 {
    font-size: 0.9rem;
  }
  .section-title p,
  .finance-heading p {
    color: var(--color-neutral-500);
    font-size: 0.75rem;
    margin-top: 0.35rem;
    line-height: 1.6;
  }
  .tag {
    font-size: 0.62rem;
    font-weight: 600;
    padding: 0.35rem 0.5rem;
    background: var(--color-neutral-100);
    border-radius: 0.35rem;
    white-space: nowrap;
    color: var(--color-neutral-600);
  }
  .tag.attention {
    background: var(--color-warning-bg);
    color: var(--color-neutral-50);
  }
  .priority-item {
    display: flex;
    align-items: start;
    gap: 0.85rem;
    padding: 1rem 0;
    border-top: 1px solid var(--color-neutral-200);
  }
  :global(.dark) .priority-item {
    border-color: var(--color-neutral-800);
  }
  .priority-item:hover h3 {
    color: var(--color-brand-600);
  }
  .priority-index {
    font-size: 0.7rem;
    font-variant-numeric: tabular-nums;
    color: var(--color-neutral-400);
    padding-top: 0.2rem;
  }
  .priority-item p {
    color: var(--color-neutral-500);
    font-size: 0.75rem;
    line-height: 1.7;
    margin-top: 0.3rem;
  }
  .priority-count {
    margin-left: auto;
    display: flex;
    align-items: center;
    gap: 0.7rem;
    font-weight: 650;
    font-size: 1.05rem;
    white-space: nowrap;
  }
  .priority-count span {
    font-size: 0.85rem;
    color: var(--color-neutral-400);
  }
  .analysis-list article + article {
    margin-top: 1.2rem;
    padding-top: 1.1rem;
    border-top: 1px solid var(--color-neutral-200);
  }
  :global(.dark) .analysis-list article + article {
    border-color: var(--color-neutral-800);
  }
  .analysis-list article > span {
    color: var(--color-brand-700);
    font-size: 0.6rem;
    letter-spacing: 0.09em;
    font-weight: 700;
  }
  :global(.dark) .analysis-list article > span {
    color: var(--color-brand-400);
  }
  .analysis-list p {
    font-size: 0.85rem;
    line-height: 1.75;
    margin: 0.5rem 0;
  }
  .analysis-list small,
  .chart-note {
    font-size: 0.7rem;
    color: var(--color-neutral-500);
    line-height: 1.7;
  }
  .two-columns {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 1.25rem;
    margin-bottom: 1.25rem;
  }
  .three-columns {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 1.25rem;
    margin: 1.25rem 0 2rem;
  }
  .text-link {
    font-size: 0.75rem;
    color: var(--color-brand-700);
    font-weight: 600;
    white-space: nowrap;
  }
  :global(.dark) .text-link {
    color: var(--color-brand-400);
  }
  .status-composition {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 1.5rem;
    padding: 1rem 0;
  }
  .donut {
    position: relative;
    width: 180px;
    flex-shrink: 0;
  }
  .donut svg {
    width: 100%;
  }
  .donut-base {
    fill: none;
    stroke: var(--color-neutral-200);
    stroke-width: 15;
  }
  .donut > div {
    position: absolute;
    inset: 0;
    display: flex;
    flex-direction: column;
    justify-content: center;
    align-items: center;
    gap: 0.25rem;
  }
  .donut strong {
    font-size: 1.65rem;
    font-weight: 650;
    letter-spacing: -0.04em;
  }
  .donut span {
    font-size: 0.65rem;
    color: var(--color-neutral-500);
  }
  .status-legend {
    min-width: 140px;
  }
  .status-legend > div {
    display: flex;
    gap: 1rem;
    justify-content: space-between;
    padding: 0.6rem 0;
    font-size: 0.8rem;
  }
  .status-legend dt {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }
  .status-legend i {
    width: 0.5rem;
    height: 0.5rem;
    border-radius: 2px;
  }
  dd {
    font-weight: 600;
    font-variant-numeric: tabular-nums;
  }
  .table-scroll {
    overflow-x: auto;
  }
  .program-table {
    width: 100%;
    text-align: left;
    border-collapse: collapse;
    font-size: 0.8rem;
  }
  .program-table thead th {
    font-size: 0.65rem;
    font-weight: 500;
    color: var(--color-neutral-500);
    white-space: nowrap;
  }
  .program-table th,
  .program-table td {
    padding: 0.9rem 0.8rem;
    border-bottom: 1px solid var(--color-neutral-200);
  }
  .program-table th:first-child {
    padding-left: 0;
  }
  :global(.dark) .program-table th,
  :global(.dark) .program-table td {
    border-color: var(--color-neutral-800);
  }
  .program-table td:last-child {
    min-width: 190px;
  }
  .strong {
    font-weight: 650;
  }
  .seat-value {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    font-size: 0.65rem;
    margin-bottom: 0.5rem;
    color: var(--color-neutral-500);
  }
  .track {
    height: 5px;
    background: var(--color-neutral-100);
    border-radius: 2px;
    overflow: hidden;
  }
  .track > div {
    height: 100%;
    background: var(--color-brand-600);
    border-radius: 2px;
  }
  :global(.dark) .track {
    background: var(--color-neutral-800);
  }
  .chart-note {
    margin-top: 1rem;
  }
  .empty-state {
    padding: 2.5rem 0.5rem;
    font-size: 0.85rem;
    color: var(--color-neutral-500);
    line-height: 1.7;
  }
  .empty-state p {
    margin-top: 0.5rem;
  }
  .section-number {
    display: block;
    font-size: 2.3rem;
    letter-spacing: -0.05em;
    font-weight: 650;
    line-height: 1;
    margin: 1.5rem 0;
  }
  .detail-list {
    margin-bottom: 1.25rem;
  }
  .detail-list > div {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    padding: 0.65rem 0;
    font-size: 0.75rem;
    border-bottom: 1px solid var(--color-neutral-200);
  }
  .detail-list dt {
    color: var(--color-neutral-500);
  }
  :global(.dark) .detail-list > div {
    border-color: var(--color-neutral-800);
  }
  .finance-heading {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    align-items: center;
    margin-bottom: 1.2rem;
  }
  .finance-heading h2 {
    font-size: 1.3rem;
  }
  .aging-list {
    display: grid;
    gap: 1rem;
  }
  .aging-label {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    font-size: 0.75rem;
    margin-bottom: 0.4rem;
  }
  .aging-list small {
    font-size: 0.65rem;
    color: var(--color-neutral-500);
  }
  .finance-ledger {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 1.5rem;
    padding: 1.4rem 0;
    border-top: 1px solid var(--color-neutral-200);
  }
  :global(.dark) .finance-ledger {
    border-color: var(--color-neutral-800);
  }
  .finance-ledger > div {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }
  .finance-ledger span {
    font-size: 0.7rem;
    color: var(--color-neutral-500);
  }
  .finance-ledger strong {
    font-weight: 600;
    font-size: 1rem;
  }
  .definitions {
    margin-top: 1.5rem;
    padding: 1rem 0;
    border-top: 1px solid var(--color-neutral-200);
    font-size: 0.75rem;
    color: var(--color-neutral-500);
  }
  :global(.dark) .definitions {
    border-color: var(--color-neutral-800);
  }
  .definitions p {
    margin-top: 0.75rem;
    line-height: 1.8;
    max-width: 100ch;
  }
  .footer-note {
    margin-top: 2rem;
    font-size: 0.65rem;
    color: var(--color-neutral-500);
  }
  .pending {
    opacity: 0.5;
    pointer-events: none;
    transition: opacity 0.15s;
  }
  @media (min-width: 1500px) {
    .metrics {
      grid-template-columns: repeat(6, minmax(0, 1fr));
    }
  }
  @media (max-width: 1100px) {
    .status-composition {
      flex-direction: column;
      gap: 0.5rem;
    }
    .status-legend {
      width: 100%;
    }
    .priority-layout {
      grid-template-columns: 1.35fr 1fr;
    }
    .three-columns {
      grid-template-columns: 1fr;
    }
  }
  @media (max-width: 767px) {
    .overview {
      padding: 1.5rem 1rem;
    }
    .page-header {
      flex-direction: column;
      align-items: start;
      gap: 1.5rem;
    }
    .header-controls {
      align-items: start;
      width: 100%;
    }
    .header-controls label {
      font-size: 0.75rem;
    }
    .metrics {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }
    .metric {
      padding: 1rem;
    }
    .priority-layout,
    .two-columns {
      grid-template-columns: 1fr;
    }
    .panel {
      padding: 1.1rem;
    }
    .finance-ledger {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }
    .finance-heading {
      align-items: start;
      flex-direction: column;
    }
    .section-title {
      flex-wrap: wrap;
    }
    .status-composition {
      flex-direction: row;
      flex-wrap: wrap;
    }
    .status-legend {
      width: auto;
      flex: 1;
    }
    .priority-count {
      gap: 0.3rem;
    }
  }
  @media (prefers-reduced-motion: reduce) {
    .metric,
    .pending {
      transition: none;
    }
  }
</style>
