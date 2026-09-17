import { createInertiaApp, router } from "@inertiajs/svelte";
import GuruDashboard from "./pages/guru/Dashboard.svelte";
import GuruKelasList from "./pages/guru/KelasList.svelte";
import GuruKelasDetail from "./pages/guru/KelasDetail.svelte";
import GuruPertemuanMulai from "./pages/guru/PertemuanMulai.svelte";
import GuruPertemuanSelesai from "./pages/guru/PertemuanSelesai.svelte";
import GuruRekapAbsensi from "./pages/guru/RekapAbsensi.svelte";
import GuruTSI from "./pages/guru/TSISaya.svelte";

const APP_NAME = "Ruang Sanad";
const DEFAULT_DESC =
	"Sistem pengelolaan mahasantri Ruang Sanad — pendaftaran, pembagian kelas, jadwal, dan laporan keuangan.";

// Per-page browser tab title (component name -> label).
const PAGE_TITLES: Record<string, string> = {
	"auth/Login": "Masuk",
	"auth/Register": "Daftar",
	"auth/ForgotPassword": "Lupa Password",
	"auth/ResetPassword": "Reset Password",
	"app/Dashboard": "Dashboard",
	"app/SantriList": "Data Santri",
	"app/SantriForm": "Form Santri",
	"app/PerluDilengkapi": "Perlu Dilengkapi",
	"app/KelasList": "Daftar Kelas",
	"app/KelasDetail": "Detail Kelas",
	"app/MasterData": "Data Master",
	"app/Keuangan": "Keuangan",
	"app/LaporanKeuangan": "Laporan Keuangan",
	"keuangan/Dashboard": "Dashboard Tagihan",
	"keuangan/Tagihan": "Daftar Tagihan",
	"keuangan/DetailTagihan": "Detail Tagihan",
	"app/ImportCSV": "Import CSV",
	"app/Profile": "Profil",
	"app/UploadTest": "Upload",
	"admin/Users": "Kelola User",
	"guru/Dashboard": "Dashboard Guru",
	"guru/KelasList": "Kelas Saya",
	"guru/KelasDetail": "Detail Kelas",
	"guru/PertemuanMulai": "Mulai Pertemuan",
	"guru/PertemuanSelesai": "Selesai Pertemuan",
	"guru/RekapAbsensi": "Rekap Absensi",
	"guru/TSISaya": "Nilai TSI",
	"koordinator/Dashboard": "Dashboard Koordinator",
	"koordinator/GuruList": "Data Guru",
	"koordinator/GuruDetail": "Detail Guru",
	"koordinator/Pembinaan": "Pembinaan",
	"koordinator/PembinaanAbsen": "Absensi Pembinaan",
	"koordinator/Rapat": "Rapat Guru",
	"koordinator/RapatAbsen": "Absensi Rapat",
	"koordinator/RiwayatAbsensi": "Riwayat Absensi Guru",
	"koordinator/Kunjungan": "Kunjungan Kelas",
	"koordinator/Kalam": "Kalam Bersanad",
	"koordinator/KalamShare": "Share Kalam",
	"koordinator/WaTemplate": "Template WA",
	"koordinator/TSIRekap": "Rekap TSI",
	"koordinator/TSIPenilaian": "Penilaian TSI",
	"koordinator/Todo": "Todo Koordinator",
};

// Per-page meta description.
const PAGE_DESC: Record<string, string> = {
	"auth/Login": "Masuk ke dashboard pengelolaan mahasantri Ruang Sanad.",
	"auth/Register": "Buat akun untuk mengakses dashboard Ruang Sanad.",
	"auth/ForgotPassword": "Atur ulang kata sandi akun Ruang Sanad Anda.",
	"auth/ResetPassword": "Setel kata sandi baru untuk akun Ruang Sanad.",
	"app/Dashboard": "Ringkasan santri, kelas, dan keuangan Ruang Sanad.",
	"app/SantriList": "Kelola dan cari data mahasantri Ruang Sanad.",
	"app/SantriForm": "Tambah atau perbarui data mahasantri.",
	"app/PerluDilengkapi": "Santri yang data kelasnya perlu dilengkapi.",
	"app/KelasList": "Daftar kelas beserta kapasitas dan jadwalnya.",
	"app/KelasDetail": "Detail kelas, guru, dan daftar santri.",
	"app/MasterData": "Kelola angkatan, level, kode kelas, jadwal, dan guru.",
	"app/Keuangan": "Catat infaq bulanan dan status pembayaran santri.",
	"app/LaporanKeuangan": "Rekap pembayaran, infaq, dan santri tidak lanjut.",
	"keuangan/Dashboard": "Ringkasan kolektibilitas dan tagihan SPP Ruang Sanad.",
	"keuangan/Tagihan": "Daftar, pelunasan, dan tindak lanjut tagihan SPP.",
	"keuangan/DetailTagihan": "Rincian tagihan dan riwayat tindak lanjut SPP.",
	"app/ImportCSV": "Impor data santri massal dari berkas CSV.",
	"app/Profile": "Kelola profil dan kata sandi akun Anda.",
	"admin/Users": "Kelola pengguna dan peran akses aplikasi.",
	"guru/Dashboard": "Ringkasan kelas dan aktivitas mengajar.",
	"guru/KelasList": "Daftar kelas yang diajar.",
	"guru/KelasDetail": "Detail kelas, santri, dan absensi.",
	"guru/PertemuanMulai": "Mulai pertemuan baru.",
	"guru/PertemuanSelesai": "Catat absensi dan selesaikan pertemuan.",
	"guru/RekapAbsensi": "Rekap absensi santri per kelas.",
	"guru/TSISaya": "Nilai TSI santri.",
	"koordinator/Dashboard": "Pengelolaan guru, penilaian TSI, dan monitoring aktivitas.",
	"koordinator/RiwayatAbsensi": "Rekap kehadiran guru pada pembinaan, rapat, dan kunjungan kelas.",
};

function applyMeta(component?: string) {
	const label = component ? PAGE_TITLES[component] : undefined;
	document.title = label ? `${label} — ${APP_NAME}` : `${APP_NAME} — Pengelolaan Mahasantri`;

	const desc = (component && PAGE_DESC[component]) || DEFAULT_DESC;
	let meta = document.querySelector('meta[name="description"]');
	if (!meta) {
		meta = document.createElement("meta");
		meta.setAttribute("name", "description");
		document.head.appendChild(meta);
	}
	meta.setAttribute("content", desc);
}

createInertiaApp();

// Initial title from the server-rendered page payload.
try {
	const el = document.querySelector("script[data-page]");
	if (el?.textContent) applyMeta(JSON.parse(el.textContent).component);
} catch {
	/* ignore malformed payload */
}

// Update title/description on every Inertia navigation.
router.on("navigate", (event: unknown) => {
	const detail = (event as CustomEvent<{ page?: { component?: string } }>)?.detail;
	applyMeta(detail?.page?.component);
});
