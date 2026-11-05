<script lang="ts">
  import { inertia, router } from "@inertiajs/svelte";
  import AppLayout from "@layouts/AppLayout.svelte";
  import SapaWADialog from "@components/riayah/SapaWADialog.svelte";
  import CatatKontakDialog from "@components/riayah/CatatKontakDialog.svelte";
  import RiayahDialog from "@components/riayah/RiayahDialog.svelte";
  import { adaFlashError, formatTanggal, hariLalu, mediaKontak } from "@lib/riayah";
  import type { Flash, RiayahSantriItem, RiayahWATemplate, User } from "@lib/types";
  import {
    ArrowLeft, ArrowRight, CheckCheck, CheckCircle2, ChevronRight, Clock3,
    ClipboardList, FilterX, MessageCircle, NotebookPen, PhoneCall,
    RefreshCw, Search, UsersRound, AlertTriangle, CalendarDays,
  } from "lucide-svelte";

  interface Props {
    user?: User;
    perlu?: RiayahSantriItem[];
    sudah?: RiayahSantriItem[];
    wa_templates?: RiayahWATemplate[];
    tanggal_hari_ini?: string;
    flash?: Flash;
  }

  let { user, perlu = [], sudah = [], wa_templates = [], tanggal_hari_ini = "", flash }: Props = $props();

  type Tab = "perlu" | "sudah";
  type Kondisi = "semua" | "3plus" | "7hari" | "tanpa-wa";
  type Urutan = "prioritas" | "terlama" | "terbaru" | "nama";

  let tab = $state<Tab>("perlu");
  let search = $state("");
  let kelasFilter = $state("");
  let guruFilter = $state("");
  let kondisiFilter = $state<Kondisi>("semua");
  let urutan = $state<Urutan>("prioritas");
  let terlihat = $state(20);
  let sedangMemuat = $state(false);

  let waSantri = $state<RiayahSantriItem | null>(null);
  let kontakSantri = $state<RiayahSantriItem | null>(null);
  let tutupSantri = $state<RiayahSantriItem | null>(null);
  let alasanTutup = $state("");
  let sedangMenutup = $state(false);

  const KEMBALI = "/app/kelas/tindak-lanjut-alpa" as const;
  const dasarPanel = "rounded-xl border border-neutral-200/80 bg-white dark:border-white/[0.08] dark:bg-neutral-925/70";
  const kontrol = "min-h-10 w-full rounded-lg border border-neutral-200 bg-white px-3 text-sm text-neutral-800 outline-none focus-visible:border-brand-500 focus-visible:ring-2 focus-visible:ring-brand-500/25 dark:border-white/10 dark:bg-neutral-900 dark:text-neutral-100";
  const tombolRingan = "inline-flex min-h-10 items-center justify-center gap-2 rounded-lg border border-neutral-200 px-3 text-sm font-medium text-neutral-700 transition-colors hover:bg-neutral-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 dark:border-white/10 dark:text-neutral-200 dark:hover:bg-white/[0.04]";
  const tombolUtama = "inline-flex min-h-10 items-center justify-center gap-2 rounded-lg bg-brand-700 px-3 text-sm font-semibold text-white transition-colors hover:bg-brand-800 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 dark:bg-brand-600 dark:hover:bg-brand-500";

  const jejak: Record<string, { warna: string; label: string; singkat: string; teks: string }> = {
    hadir: { warna: "bg-emerald-600", label: "Hadir", singkat: "H", teks: "text-white" },
    telat: { warna: "bg-emerald-300", label: "Terlambat", singkat: "T", teks: "text-emerald-950" },
    izin: { warna: "bg-sky-500", label: "Izin", singkat: "I", teks: "text-white" },
    sakit: { warna: "bg-amber-400", label: "Sakit", singkat: "S", teks: "text-amber-950" },
    alpa: { warna: "bg-rose-500", label: "Alpa", singkat: "A", teks: "text-white" },
  };

  function hariSejak(tanggal: string): number {
    if (!tanggal || !tanggal_hari_ini) return 0;
    const awal = Date.parse(tanggal.slice(0, 10) + "T00:00:00Z");
    const akhir = Date.parse(tanggal_hari_ini.slice(0, 10) + "T00:00:00Z");
    return Number.isFinite(awal) && Number.isFinite(akhir) ? Math.max(0, Math.floor((akhir - awal) / 86400000)) : 0;
  }

  function labelMedia(media: string): string {
    return mediaKontak.find((item) => item.value === media)?.label ?? media;
  }

  function persen(value: number | null): string {
    return value === null ? "Belum tersedia" : Math.round(value) + "%";
  }

  function refresh() {
    if (sedangMemuat) return;
    router.reload({
      only: ["perlu", "sudah", "tanggal_hari_ini"],
      preserveScroll: true,
      onStart: () => { sedangMemuat = true; },
      onFinish: () => { sedangMemuat = false; },
    });
  }

  function gantiTab(value: Tab) {
    tab = value;
    terlihat = 20;
  }

  function resetFilter() {
    search = "";
    kelasFilter = "";
    guruFilter = "";
    kondisiFilter = "semua";
    urutan = "prioritas";
    terlihat = 20;
  }

  function pilihKelas(id: string) {
    tab = "perlu";
    kelasFilter = id;
    guruFilter = "";
    kondisiFilter = "semua";
    search = "";
    terlihat = 20;
  }

  function tutupKasus(event: SubmitEvent) {
    event.preventDefault();
    if (!tutupSantri || sedangMenutup || alasanTutup.trim().length < 10) return;
    sedangMenutup = true;
    router.post(`/app/kelas/tindak-lanjut-alpa/${tutupSantri.id}/selesai`, { alasan: alasanTutup.trim() }, {
      preserveScroll: true,
      preserveState: true,
      onSuccess: (page) => {
        if (!adaFlashError(page)) {
          tutupSantri = null;
          alasanTutup = "";
        }
      },
      onFinish: () => { sedangMenutup = false; },
    });
  }

  let totalTigaPlus = $derived(perlu.filter((s) => (s.tindak_lanjut_alpa?.alpa_beruntun ?? 0) >= 3).length);
  let totalTujuhHari = $derived(perlu.filter((s) => hariSejak(s.tindak_lanjut_alpa?.alpa_terakhir ?? "") >= 7).length);
  let semuaSantri = $derived([...perlu, ...sudah]);

  let kelasTersedia = $derived.by(() => {
    const pilihan = new Map<string, string>();
    for (const item of semuaSantri) pilihan.set(String(item.kelas_id), item.nama_kelas || "Tanpa kelas");
    return [...pilihan.entries()].sort((a, b) => a[1].localeCompare(b[1], "id"));
  });

  let guruTersedia = $derived.by(() => {
    const pilihan = new Set(semuaSantri.map((s) => s.guru_nama).filter(Boolean));
    return [...pilihan].sort((a, b) => a.localeCompare(b, "id"));
  });

  let kelasTerbanyak = $derived.by(() => {
    const byKelas = new Map<string, { id: string; nama: string; guru: string; jumlah: number }>();
    for (const s of perlu) {
      const key = String(s.kelas_id);
      const found = byKelas.get(key);
      if (found) found.jumlah++;
      else byKelas.set(key, { id: key, nama: s.nama_kelas || "Tanpa kelas", guru: s.guru_nama || "Belum ditugaskan", jumlah: 1 });
    }
    return [...byKelas.values()].sort((a, b) => b.jumlah - a.jumlah || a.nama.localeCompare(b.nama, "id")).slice(0, 5);
  });

  let puncakKelas = $derived(Math.max(1, ...kelasTerbanyak.map((item) => item.jumlah)));

  let metodeSelesai = $derived.by(() => {
    const jenis = [
      { id: "wa", nama: "WhatsApp", jumlah: 0 },
      { id: "telepon", nama: "Telepon", jumlah: 0 },
      { id: "tatap_muka", nama: "Tatap muka", jumlah: 0 },
      { id: "alasan", nama: "Ditutup dengan alasan", jumlah: 0 },
      { id: "lainnya", nama: "Media lainnya", jumlah: 0 },
    ];
    for (const s of sudah) {
      const tl = s.tindak_lanjut_alpa;
      if (!tl) continue;
      const id = tl.kontak_jenis === "tutup_alpa" ? "alasan" : tl.kontak_media;
      const entry = jenis.find((x) => x.id === id);
      if (entry) entry.jumlah++;
      else jenis[4].jumlah++;
    }
    return jenis.filter((x) => x.jumlah > 0).sort((a, b) => b.jumlah - a.jumlah);
  });

  let adaFilter = $derived(Boolean(search.trim() || kelasFilter || guruFilter || kondisiFilter !== "semua" || urutan !== "prioritas"));

  let tersaring = $derived.by(() => {
    const q = search.toLocaleLowerCase("id").trim();
    const data = (tab === "perlu" ? perlu : sudah).filter((s) => {
      const tl = s.tindak_lanjut_alpa;
      if (!tl) return false;
      if (kelasFilter && String(s.kelas_id) !== kelasFilter) return false;
      if (guruFilter && s.guru_nama !== guruFilter) return false;
      if (kondisiFilter === "3plus" && tl.alpa_beruntun < 3) return false;
      if (kondisiFilter === "7hari" && hariSejak(tl.alpa_terakhir) < 7) return false;
      if (kondisiFilter === "tanpa-wa" && s.no_wa) return false;
      return !q || [s.nama, s.id_mahasantri, s.nama_kelas, s.guru_nama, s.jadwal].join(" ").toLocaleLowerCase("id").includes(q);
    });
    return [...data].sort((a, b) => {
      const ta = a.tindak_lanjut_alpa!;
      const tb = b.tindak_lanjut_alpa!;
      if (urutan === "nama") return a.nama.localeCompare(b.nama, "id");
      if (urutan === "terbaru") return (tab === "sudah" ? tb.kontak_tanggal.localeCompare(ta.kontak_tanggal) : tb.alpa_terakhir.localeCompare(ta.alpa_terakhir)) || a.nama.localeCompare(b.nama, "id");
      if (urutan === "terlama") return (tab === "sudah" ? ta.kontak_tanggal.localeCompare(tb.kontak_tanggal) : ta.alpa_terakhir.localeCompare(tb.alpa_terakhir)) || a.nama.localeCompare(b.nama, "id");
      if (tab === "sudah") return tb.kontak_tanggal.localeCompare(ta.kontak_tanggal) || a.nama.localeCompare(b.nama, "id");
      return tb.alpa_beruntun - ta.alpa_beruntun || ta.alpa_terakhir.localeCompare(tb.alpa_terakhir) || a.nama.localeCompare(b.nama, "id");
    });
  });
  let daftar = $derived(tersaring.slice(0, terlihat));
</script>

<svelte:head>
  <title>Dashboard Tindak Lanjut Alpa | Ruang Sanad</title>
</svelte:head>

<AppLayout {user} group="tindak-lanjut-alpa">
  <main class="mx-auto max-w-7xl space-y-5 px-4 py-6 sm:px-6 lg:py-8">
    <header class="flex flex-col justify-between gap-4 sm:flex-row sm:items-end">
      <div class="min-w-0">
        <a href="/app/kelas" use:inertia class="inline-flex min-h-9 items-center gap-1.5 text-xs font-medium text-neutral-500 hover:text-brand-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 dark:text-neutral-400 dark:hover:text-brand-400">
          <ArrowLeft size={14} /> Kembali ke manajemen kelas
        </a>
        <p class="mt-3 text-[11px] font-semibold uppercase tracking-[0.12em] text-brand-700 dark:text-brand-400">Riayah mahasantri / Admin Kelas</p>
        <h1 class="mt-1 text-[28px] font-semibold leading-tight tracking-[-0.025em] text-neutral-950 dark:text-white sm:text-[32px]">Tindak lanjut alpa</h1>
        <p class="mt-2 max-w-2xl text-sm leading-relaxed text-neutral-600 dark:text-neutral-400">
          Kasus dimulai setelah dua alpa berturut-turut. Kehadiran berikutnya tidak otomatis menyelesaikannya, sehingga setiap kasus tetap mendapat tindak lanjut yang tercatat.
        </p>
      </div>
      <div class="flex shrink-0 items-center gap-2">
        {#if tanggal_hari_ini}
          <span class="hidden items-center gap-1.5 text-xs text-neutral-500 md:inline-flex dark:text-neutral-400">
            <CalendarDays size={14} /> {formatTanggal(tanggal_hari_ini)} · WIB
          </span>
        {/if}
        <button type="button" class={tombolRingan} onclick={refresh} disabled={sedangMemuat} aria-label="Muat ulang data tindak lanjut">
          <RefreshCw size={15} class={sedangMemuat ? "animate-spin" : ""} />
          {sedangMemuat ? "Memuat" : "Muat ulang"}
        </button>
      </div>
    </header>

    {#if flash?.success}
      <div role="status" class="rounded-lg border border-emerald-500/25 bg-emerald-500/[0.07] px-4 py-3 text-sm text-emerald-800 dark:text-emerald-300">{flash.success}</div>
    {/if}
    {#if flash?.error}
      <div role="alert" class="rounded-lg border border-rose-500/25 bg-rose-500/[0.07] px-4 py-3 text-sm text-rose-800 dark:text-rose-300">{flash.error}</div>
    {/if}

    <section aria-labelledby="ringkasan-heading" class="space-y-2.5">
      <h2 id="ringkasan-heading" class="sr-only">Ringkasan tindak lanjut alpa</h2>
      <div class="grid grid-cols-2 gap-3 lg:grid-cols-[minmax(0,1.75fr)_repeat(3,minmax(0,1fr))]">
        <div class="col-span-2 flex min-h-44 flex-col justify-between rounded-xl bg-neutral-950 p-5 text-white dark:border dark:border-white/10 lg:col-span-1">
          <div class="flex items-start justify-between gap-4">
            <div>
              <p class="text-[11px] font-semibold uppercase tracking-[0.12em] text-neutral-300">Antrean aktif</p>
              <div class="mt-3 flex items-end gap-2">
                <span class="text-[48px] font-semibold leading-none tracking-[-0.055em] tabular-nums">{perlu.length}</span>
                <span class="mb-1 text-sm text-neutral-300">kasus belum selesai</span>
              </div>
            </div>
            <div class="rounded-lg border border-white/15 p-2.5 text-brand-300"><ClipboardList size={21}/></div>
          </div>
          <div class="mt-5 border-t border-white/15 pt-3">
            <p class="text-xs text-neutral-300">{perlu.length ? "Urutkan penanganan berdasarkan jumlah alpa dan kapan terakhir alpa." : "Tidak ada kasus terbuka yang perlu ditangani saat ini."}</p>
          </div>
        </div>

        <div class={dasarPanel + " flex min-h-36 flex-col justify-between p-4 sm:p-5"}>
          <div class="flex items-start justify-between gap-2">
            <p class="text-xs font-medium leading-snug text-neutral-600 dark:text-neutral-400">Alpa 3 kali atau lebih</p>
            <AlertTriangle size={17} class="shrink-0 text-rose-600 dark:text-rose-400"/>
          </div>
          <div>
            <p class="text-[30px] font-semibold leading-none tracking-tight tabular-nums text-neutral-950 dark:text-white">{totalTigaPlus}</p>
            <p class="mt-2 text-xs text-neutral-500 dark:text-neutral-400">Dari kasus terbuka</p>
          </div>
        </div>

        <div class={dasarPanel + " flex min-h-36 flex-col justify-between p-4 sm:p-5"}>
          <div class="flex items-start justify-between gap-2">
            <p class="text-xs font-medium leading-snug text-neutral-600 dark:text-neutral-400">Alpa terakhir ≥7 hari</p>
            <Clock3 size={17} class="shrink-0 text-amber-600 dark:text-amber-400"/>
          </div>
          <div>
            <p class="text-[30px] font-semibold leading-none tracking-tight tabular-nums text-neutral-950 dark:text-white">{totalTujuhHari}</p>
            <p class="mt-2 text-xs text-neutral-500 dark:text-neutral-400">Belum diselesaikan</p>
          </div>
        </div>

        <div class={dasarPanel + " col-span-2 flex min-h-36 flex-col justify-between p-4 sm:p-5 lg:col-span-1"}>
          <div class="flex items-start justify-between gap-2">
            <p class="text-xs font-medium leading-snug text-neutral-600 dark:text-neutral-400">Selesai 14 hari terakhir</p>
            <CheckCheck size={18} class="shrink-0 text-emerald-700 dark:text-emerald-400"/>
          </div>
          <div>
            <p class="text-[30px] font-semibold leading-none tracking-tight tabular-nums text-neutral-950 dark:text-white">{sudah.length}</p>
            <p class="mt-2 text-xs text-neutral-500 dark:text-neutral-400">Kontak atau tutup beralasan</p>
          </div>
        </div>
      </div>
      <p class="px-0.5 text-[11px] text-neutral-500 dark:text-neutral-400">Ringkasan dihitung dari seluruh kasus yang dikirim server dan tidak berubah ketika filter daftar digunakan. Usia kasus mengacu pada tanggal alpa terakhir, bukan SLA kontak.</p>
    </section>

    <div class="grid items-start gap-4 xl:grid-cols-[minmax(0,1fr)_19.5rem]">
      <section id="daftar-kasus" class={dasarPanel + " min-w-0 overflow-hidden"} aria-labelledby="antrean-heading" aria-busy={sedangMemuat}>
        <div class="border-b border-neutral-200/75 px-4 pt-5 pb-4 dark:border-white/[0.07] sm:px-5">
          <div class="flex flex-wrap items-start justify-between gap-3">
            <div>
              <p class="text-[11px] font-semibold uppercase tracking-[0.1em] text-brand-700 dark:text-brand-400">Daftar kerja</p>
              <h2 id="antrean-heading" class="mt-1 text-lg font-semibold tracking-tight text-neutral-950 dark:text-white">Antrean mahasantri</h2>
              <p class="mt-1 text-xs text-neutral-500 dark:text-neutral-400">Pilih kasus, hubungi mahasantri, lalu dokumentasikan hasilnya.</p>
            </div>
            <span class="rounded-md bg-neutral-100 px-2.5 py-1.5 text-xs font-medium tabular-nums text-neutral-700 dark:bg-white/[0.07] dark:text-neutral-300">{tersaring.length} kasus sesuai filter</span>
          </div>

          <div class="mt-5 flex flex-wrap gap-1 border-b border-neutral-200 dark:border-white/[0.07]" role="group" aria-label="Status tindak lanjut">
            <button type="button" aria-pressed={tab === "perlu"} onclick={() => gantiTab("perlu")} class={"-mb-px inline-flex min-h-11 items-center gap-2 border-b-2 px-3 text-sm font-semibold focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 " + (tab === "perlu" ? "border-brand-600 text-brand-800 dark:text-brand-300" : "border-transparent text-neutral-500 hover:text-neutral-900 dark:hover:text-neutral-200")}>
              Perlu ditangani <span class={"rounded-md px-1.5 py-0.5 text-xs tabular-nums " + (tab === "perlu" ? "bg-brand-50 text-brand-800 dark:bg-brand-500/15 dark:text-brand-300" : "bg-neutral-100 text-neutral-500 dark:bg-white/10")}>{perlu.length}</span>
            </button>
            <button type="button" aria-pressed={tab === "sudah"} onclick={() => gantiTab("sudah")} class={"-mb-px inline-flex min-h-11 items-center gap-2 border-b-2 px-3 text-sm font-semibold focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 " + (tab === "sudah" ? "border-brand-600 text-brand-800 dark:text-brand-300" : "border-transparent text-neutral-500 hover:text-neutral-900 dark:hover:text-neutral-200")}>
              Selesai <span class={"rounded-md px-1.5 py-0.5 text-xs tabular-nums " + (tab === "sudah" ? "bg-brand-50 text-brand-800 dark:bg-brand-500/15 dark:text-brand-300" : "bg-neutral-100 text-neutral-500 dark:bg-white/10")}>{sudah.length}</span>
            </button>
          </div>

          <div class="mt-4 space-y-3">
            <label class="block">
              <span class="mb-1 block text-xs font-medium text-neutral-600 dark:text-neutral-400">Cari mahasantri</span>
              <span class="relative block">
                <Search size={16} class="pointer-events-none absolute left-3 top-3 text-neutral-400"/>
                <input type="search" bind:value={search} oninput={() => { terlihat = 20; }} placeholder="Nama, ID, kelas, guru, jadwal..." class={kontrol + " pl-9"}/>
              </span>
            </label>
            <div class="grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-4">
              <label class="block min-w-0">
                <span class="mb-1 block text-xs font-medium text-neutral-600 dark:text-neutral-400">Kelas</span>
                <select bind:value={kelasFilter} onchange={() => { terlihat = 20; }} class={kontrol}>
                  <option value="">Semua kelas</option>
                  {#each kelasTersedia as item (item[0])}<option value={item[0]}>{item[1]}</option>{/each}
                </select>
              </label>
              <label class="block min-w-0">
                <span class="mb-1 block text-xs font-medium text-neutral-600 dark:text-neutral-400">Guru</span>
                <select bind:value={guruFilter} onchange={() => { terlihat = 20; }} class={kontrol}>
                  <option value="">Semua guru</option>
                  {#each guruTersedia as nama (nama)}<option value={nama}>{nama}</option>{/each}
                </select>
              </label>
              <label class="block min-w-0">
                <span class="mb-1 block text-xs font-medium text-neutral-600 dark:text-neutral-400">Kondisi</span>
                <select bind:value={kondisiFilter} onchange={() => { terlihat = 20; }} class={kontrol}>
                  <option value="semua">Semua kondisi</option>
                  <option value="3plus">3+ alpa beruntun</option>
                  <option value="7hari">Alpa terakhir ≥7 hari</option>
                  <option value="tanpa-wa">Tanpa nomor WhatsApp</option>
                </select>
              </label>
              <label class="block min-w-0">
                <span class="mb-1 block text-xs font-medium text-neutral-600 dark:text-neutral-400">Urutkan</span>
                <select bind:value={urutan} onchange={() => { terlihat = 20; }} class={kontrol}>
                  <option value="prioritas">Prioritas penanganan</option>
                  <option value="terlama">Alpa terlama</option>
                  <option value="terbaru">Alpa terbaru</option>
                  <option value="nama">Nama A sampai Z</option>
                </select>
              </label>
            </div>
            {#if adaFilter}
              <div class="flex justify-end">
                <button type="button" onclick={resetFilter} class="inline-flex min-h-9 items-center gap-1.5 text-xs font-medium text-brand-700 hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 dark:text-brand-400">
                  <FilterX size={14}/> Hapus semua filter
                </button>
              </div>
            {/if}
          </div>
        </div>

        {#if tab === "sudah"}
          <div class="border-b border-neutral-200/75 bg-neutral-50 px-4 py-3 text-xs leading-relaxed text-neutral-600 dark:border-white/[0.06] dark:bg-white/[0.02] dark:text-neutral-400 sm:px-5">
            Menampilkan kasus yang ditutup dengan catatan kontak atau alasan selama 14 hari terakhir. Riwayat lengkap tetap tersedia di profil mahasantri.
          </div>
        {/if}

        <div class="divide-y divide-neutral-200/70 dark:divide-white/[0.07]">
          {#each daftar as s (s.id)}
            {@const tl = s.tindak_lanjut_alpa}
            {#if tl}
              <article class="p-4 transition-colors hover:bg-neutral-50/70 dark:hover:bg-white/[0.025] sm:p-5">
                <div class="flex flex-col gap-4">
                  <div class="flex flex-col justify-between gap-3 sm:flex-row">
                    <div class="min-w-0">
                      <div class="flex flex-wrap items-center gap-2">
                        <h3 class="text-[15px] font-semibold text-neutral-950 dark:text-white">{s.nama}</h3>
                        {#if tab === "perlu"}
                          <span class={"rounded-md px-2 py-0.5 text-[11px] font-semibold " + (tl.alpa_beruntun >= 3 ? "bg-rose-500/10 text-rose-700 dark:text-rose-400" : "bg-amber-500/10 text-amber-800 dark:text-amber-400")}>
                            Pernah {tl.alpa_beruntun} kali alpa beruntun
                          </span>
                        {:else}
                          <span class="rounded-md bg-emerald-500/10 px-2 py-0.5 text-[11px] font-semibold text-emerald-800 dark:text-emerald-400">Selesai</span>
                        {/if}
                      </div>
                      <p class="mt-1 break-all text-xs text-neutral-500 dark:text-neutral-400">{s.id_mahasantri}</p>
                      <p class="mt-2 text-sm text-neutral-700 dark:text-neutral-300">{s.nama_kelas || "Kelas belum ditentukan"}</p>
                      <p class="mt-0.5 text-xs leading-relaxed text-neutral-500 dark:text-neutral-400">{[s.guru_nama ? "Guru: " + s.guru_nama : "", s.jadwal].filter(Boolean).join(" · ")}</p>
                    </div>
                    <div class="shrink-0 self-start text-left sm:text-right">
                      <p class="text-[11px] text-neutral-500 dark:text-neutral-400">Alpa terakhir</p>
                      <p class="mt-0.5 text-sm font-semibold tabular-nums text-neutral-800 dark:text-neutral-200">{formatTanggal(tl.alpa_terakhir)}</p>
                      <p class="mt-0.5 text-xs text-neutral-500 dark:text-neutral-400">{hariSejak(tl.alpa_terakhir)} hari sejak alpa terakhir</p>
                    </div>
                  </div>

                  <div class="flex flex-col gap-3 rounded-lg bg-neutral-50 px-3 py-3 dark:bg-white/[0.035] sm:flex-row sm:items-center sm:justify-between">
                    <div class="min-w-0">
                      <p class="text-[11px] font-medium text-neutral-500 dark:text-neutral-400">Lima absensi tercatat terakhir <span class="font-normal">(lama ke baru)</span></p>
                      <div class="mt-2 flex flex-wrap items-center gap-1.5">
                        {#each [...tl.absen_terakhir].reverse() as status, idx (idx)}
                          {@const item = jejak[status]}
                          <span title={item?.label ?? status} aria-label={item?.label ?? status} class={"inline-flex h-7 w-7 items-center justify-center rounded-md text-[11px] font-semibold " + (item?.warna ?? "bg-neutral-300") + " " + (item?.teks ?? "text-neutral-700")}>{item?.singkat ?? "?"}</span>
                        {:else}
                          <span class="text-xs text-neutral-400">Belum ada riwayat</span>
                        {/each}
                      </div>
                    </div>
                    <div class="shrink-0 sm:text-right">
                      <p class="text-[11px] font-medium text-neutral-500 dark:text-neutral-400">Kehadiran 30 hari</p>
                      <p class="mt-1 text-lg font-semibold tabular-nums text-neutral-950 dark:text-white">{persen(s.persen_hadir_30)}</p>
                    </div>
                  </div>

                  {#if tab === "sudah"}
                    <div class="rounded-lg border border-emerald-500/20 bg-emerald-500/[0.05] p-3">
                      <p class="flex items-center gap-1.5 text-xs font-semibold text-emerald-800 dark:text-emerald-300">
                        <CheckCircle2 size={14}/>
                        {tl.kontak_jenis === "tutup_alpa" ? "Ditutup dengan alasan" : "Kontak tercatat"}
                        · {formatTanggal(tl.kontak_tanggal)}
                      </p>
                      <p class="mt-1 text-xs text-neutral-600 dark:text-neutral-300">
                        {tl.kontak_oleh ? "Oleh " + tl.kontak_oleh : "Petugas tidak tercatat"}{tl.kontak_jenis === "tutup_alpa" ? "" : " · " + labelMedia(tl.kontak_media)}
                      </p>
                      {#if tl.kontak_catatan}
                        <p class="mt-2 whitespace-pre-line break-words text-sm leading-relaxed text-neutral-700 dark:text-neutral-300">{tl.kontak_catatan}</p>
                      {/if}
                    </div>
                  {/if}

                  <div class="flex flex-wrap items-center gap-2">
                    {#if tab === "perlu"}
                      {#if s.no_wa}
                        <button type="button" onclick={() => { waSantri = s; }} class={tombolRingan}>
                          <MessageCircle size={15}/> WhatsApp
                        </button>
                      {:else}
                        <span class="inline-flex min-h-10 items-center gap-1.5 rounded-lg border border-dashed border-neutral-300 px-3 text-xs text-neutral-500 dark:border-neutral-700 dark:text-neutral-400">
                          <PhoneCall size={14}/> Nomor WA kosong
                        </span>
                      {/if}
                      <button type="button" onclick={() => { kontakSantri = s; }} class={tombolUtama}>
                        <NotebookPen size={15}/> Catat hasil kontak
                      </button>
                      <button type="button" onclick={() => { tutupSantri = s; alasanTutup = ""; }} class={tombolRingan}>
                        <CheckCircle2 size={15}/> Tutup dengan alasan
                      </button>
                    {/if}
                    <a href={`/app/guru/santri/${s.id}`} use:inertia class="ml-auto inline-flex min-h-10 items-center gap-1 text-xs font-semibold text-brand-700 underline-offset-4 hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 dark:text-brand-400">
                      Profil santri <ChevronRight size={15}/>
                    </a>
                  </div>
                </div>
              </article>
            {/if}
          {:else}
            <div class="flex flex-col items-center px-5 py-14 text-center">
              <div class="rounded-xl bg-neutral-100 p-3 text-neutral-500 dark:bg-white/[0.06] dark:text-neutral-400">
                {#if tab === "perlu"}<ClipboardList size={23}/>{:else}<CheckCheck size={23}/>{/if}
              </div>
              <h3 class="mt-3 text-sm font-semibold text-neutral-900 dark:text-white">
                {#if adaFilter}Tidak ada kasus yang cocok{:else if tab === "perlu"}Belum ada kasus terbuka{:else}Belum ada kasus selesai dalam 14 hari{/if}
              </h3>
              <p class="mt-1 max-w-sm text-xs leading-relaxed text-neutral-500 dark:text-neutral-400">
                {#if adaFilter}Coba ubah pencarian atau kosongkan filter yang dipilih.{:else if tab === "perlu"}Kasus akan muncul saat mahasantri aktif memiliki dua alpa beruntun dan belum ada tindak lanjut yang tercatat.{:else}Penyelesaian yang tercatat akan tampil di sini selama 14 hari.{/if}
              </p>
              {#if adaFilter}
                <button type="button" onclick={resetFilter} class={tombolRingan + " mt-4"}><FilterX size={15}/> Reset filter</button>
              {/if}
            </div>
          {/each}
        </div>

        {#if tersaring.length > terlihat}
          <div class="border-t border-neutral-200/75 px-5 py-4 text-center dark:border-white/[0.07]">
            <button type="button" onclick={() => { terlihat += 20; }} class={tombolRingan}>
              Tampilkan 20 berikutnya <ArrowRight size={15}/>
            </button>
          </div>
        {/if}
        {#if tersaring.length > 0}
          <div class="border-t border-neutral-200/75 px-5 py-3 text-xs tabular-nums text-neutral-500 dark:border-white/[0.07] dark:text-neutral-400">
            Menampilkan {daftar.length} dari {tersaring.length} kasus sesuai filter
          </div>
        {/if}
      </section>

      <aside aria-label="Analisis ringkas" class="space-y-4">
        <section class={dasarPanel + " p-4 sm:p-5"} aria-labelledby="kelas-terbanyak-heading">
          <div class="flex items-start justify-between gap-2">
            <div>
              <p class="text-[11px] font-semibold uppercase tracking-[0.1em] text-brand-700 dark:text-brand-400">Sebaran antrean</p>
              <h2 id="kelas-terbanyak-heading" class="mt-1 text-base font-semibold text-neutral-950 dark:text-white">Kelas dengan kasus terbanyak</h2>
            </div>
            <UsersRound size={18} class="shrink-0 text-neutral-400"/>
          </div>
          <p class="mt-2 text-xs leading-relaxed text-neutral-500 dark:text-neutral-400">Klik nama kelas untuk menyaring antrean yang perlu ditangani.</p>
          {#if kelasTerbanyak.length}
            <div class="mt-5 space-y-4">
              {#each kelasTerbanyak as k (k.id)}
                <button type="button" onclick={() => pilihKelas(k.id)} class="block w-full rounded-lg text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500">
                  <span class="flex items-start justify-between gap-2">
                    <span class="min-w-0 truncate text-xs font-semibold text-neutral-800 dark:text-neutral-200" title={k.nama}>{k.nama}</span>
                    <span class="shrink-0 text-xs font-semibold tabular-nums text-neutral-900 dark:text-white">{k.jumlah}</span>
                  </span>
                  <span class="mt-1 block truncate text-[11px] text-neutral-500 dark:text-neutral-400">{k.guru}</span>
                  <span class="mt-2 block h-1.5 overflow-hidden rounded-full bg-neutral-100 dark:bg-white/10">
                    <span class="block h-full rounded-full bg-brand-600 dark:bg-brand-400" style={`width: ${(k.jumlah / puncakKelas) * 100}%`}></span>
                  </span>
                </button>
              {/each}
            </div>
            {#if perlu.length > 5}
              <p class="mt-4 text-[11px] text-neutral-500 dark:text-neutral-400">Maksimal 5 kelas teratas berdasarkan jumlah kasus terbuka.</p>
            {/if}
          {:else}
            <p class="mt-5 rounded-lg bg-neutral-50 p-3 text-xs text-neutral-500 dark:bg-white/[0.04]">Tidak ada kelas dengan kasus aktif.</p>
          {/if}
        </section>

        <section class={dasarPanel + " p-4 sm:p-5"} aria-labelledby="penyelesaian-heading">
          <div class="flex items-start justify-between gap-2">
            <div>
              <p class="text-[11px] font-semibold uppercase tracking-[0.1em] text-brand-700 dark:text-brand-400">Jejak penyelesaian</p>
              <h2 id="penyelesaian-heading" class="mt-1 text-base font-semibold text-neutral-950 dark:text-white">Cara kasus diselesaikan</h2>
            </div>
            <CheckCheck size={18} class="shrink-0 text-emerald-700 dark:text-emerald-400"/>
          </div>
          <p class="mt-2 text-xs leading-relaxed text-neutral-500 dark:text-neutral-400">Distribusi media kontak dan penutupan beralasan dalam 14 hari terakhir.</p>
          {#if metodeSelesai.length}
            <div class="mt-5 space-y-3">
              {#each metodeSelesai as metode (metode.id)}
                <div>
                  <div class="flex items-center justify-between gap-2 text-xs text-neutral-700 dark:text-neutral-300">
                    <span>{metode.nama}</span>
                    <span class="font-semibold tabular-nums">{metode.jumlah}</span>
                  </div>
                  <div class="mt-1.5 h-1.5 overflow-hidden rounded-full bg-neutral-100 dark:bg-white/10">
                    <div class="h-full rounded-full bg-emerald-600 dark:bg-emerald-400" style={`width: ${(metode.jumlah / Math.max(1, sudah.length)) * 100}%`}></div>
                  </div>
                </div>
              {/each}
            </div>
          {:else}
            <p class="mt-5 rounded-lg bg-neutral-50 p-3 text-xs text-neutral-500 dark:bg-white/[0.04]">Belum ada penyelesaian dalam 14 hari terakhir.</p>
          {/if}
        </section>

        <section class="rounded-xl border border-brand-500/20 bg-brand-500/[0.045] p-4 dark:bg-brand-500/[0.07] sm:p-5" aria-labelledby="panduan-heading">
          <h2 id="panduan-heading" class="flex items-center gap-2 text-sm font-semibold text-neutral-900 dark:text-white"><ClipboardList size={16} class="text-brand-700 dark:text-brand-400"/> Alur penanganan</h2>
          <ol class="mt-3 space-y-2.5 text-xs leading-relaxed text-neutral-600 dark:text-neutral-300">
            <li><span class="font-semibold text-neutral-900 dark:text-white">1.</span> Cek riwayat alpa dan kelas mahasantri.</li>
            <li><span class="font-semibold text-neutral-900 dark:text-white">2.</span> Hubungi mahasantri lewat media yang tersedia.</li>
            <li><span class="font-semibold text-neutral-900 dark:text-white">3.</span> Catat hasil kontak atau tutup kasus dengan alasan yang bisa dipertanggungjawabkan.</li>
          </ol>
          <p class="mt-3 border-t border-brand-500/15 pt-3 text-[11px] leading-relaxed text-neutral-500 dark:text-neutral-400">Rapor bulanan bukan pengganti catatan tindak lanjut alpa. Absensi berikutnya tidak menutup kasus secara otomatis.</p>
        </section>
      </aside>
    </div>
  </main>

  {#if tutupSantri}
    <RiayahDialog title="Tutup kasus tindak lanjut alpa" subtitle={tutupSantri.nama} busy={sedangMenutup} onclose={() => { tutupSantri = null; }}>
      <form onsubmit={tutupKasus} class="space-y-4">
        <p class="text-sm text-neutral-600 dark:text-neutral-400">Jika mahasantri tidak perlu dihubungi lagi, tulis alasan penutupan yang dapat dipertanggungjawabkan.</p>
        <label class="block">
          <span class="mb-1.5 block text-sm font-medium text-neutral-700 dark:text-neutral-300">Alasan penyelesaian (minimal 10 karakter)</span>
          <textarea bind:value={alasanTutup} minlength="10" required rows="3" class={kontrol + " min-h-28 py-3"} placeholder="Contoh: Kondisi sudah ditangani langsung oleh Admin Kelas"></textarea>
        </label>
        <div class="flex justify-end gap-2">
          <button type="button" onclick={() => { tutupSantri = null; }} disabled={sedangMenutup} class={tombolRingan}>Batal</button>
          <button type="submit" disabled={sedangMenutup || alasanTutup.trim().length < 10} class={tombolUtama + " disabled:opacity-50"}>{sedangMenutup ? "Menyimpan..." : "Selesaikan kasus"}</button>
        </div>
      </form>
    </RiayahDialog>
  {/if}

  {#if kontakSantri}
    <CatatKontakDialog santri={kontakSantri} kembali={KEMBALI} onclose={() => { kontakSantri = null; }}/>
  {/if}
  {#if waSantri}
    <SapaWADialog santri={waSantri} templates={wa_templates} pengirim={user?.name ?? ""} kembali={KEMBALI} onclose={() => { waSantri = null; }}/>
  {/if}
</AppLayout>
