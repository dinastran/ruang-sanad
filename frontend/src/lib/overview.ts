export interface Overview {
  bulan: string;
  bulan_label: string;
  updated_at: string;
  current_month: boolean;
  errors: string[];
  santri?: {
    total: number;
    aktif: number;
    cuti: number;
    nonaktif: number;
    tidak_lanjut: number;
    active_rate: number;
  };
  growth: { month: string; total: number }[];
  programs: {
    name: string;
    active: number;
    cuti: number;
    stopped: number;
    classes: number;
    seats: number;
    occupied: number;
  }[];
  operations?: {
    incomplete: number;
    teachers: number;
    classes: number;
    without_teacher: number;
    without_class: number;
    open_meetings: number;
    follow_ups: number;
    todos: number;
    finished: number;
    badal: number;
    rescheduled: number;
  };
  riayah?: {
    total_santri: number;
    perlu_perhatian: number;
    kehadiran: number;
    kontak: number;
    progres: number;
    rapor_periode: string;
    rapor_terkirim: number;
  };
  priorities: { title: string; detail: string; href: string; count: number }[];
  products?: {
    active_products: number;
    total_stock: number;
    low_stock_products: number;
    coverage_rate: number;
    santri_with_product: number;
    santri_total: number;
  };
  finance?: {
    sekarang: FinanceMonth;
    sebelumnya: FinanceMonth;
    tren: FinanceMonth[];
    total_piutang: number;
    jumlah_piutang: number;
    jumlah_terlambat: number;
    aging: { key: string; label: string; jumlah: number; nominal: number }[];
    ujroh: { kewajiban_terkunci: number; estimasi_berjalan: number };
    produk: { jumlah_order: number; jumlah_qty: number; omzet: number };
  };
}
interface FinanceMonth {
  bulan: string;
  label: string;
  kas_masuk: number;
  kas_masuk_spp: number;
  kas_masuk_produk: number;
  kas_keluar: number;
  kas_bersih: number;
  terbit_nominal: number;
  terbit_lunas_nominal: number;
  kolektibilitas: number;
}
