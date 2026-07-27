import { createInertiaApp, router } from "@inertiajs/svelte";

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
	"app/ImportCSV": "Import CSV",
	"app/Profile": "Profil",
	"app/UploadTest": "Upload",
	"admin/Users": "Kelola User",
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
	"app/ImportCSV": "Impor data santri massal dari berkas CSV.",
	"app/Profile": "Kelola profil dan kata sandi akun Anda.",
	"admin/Users": "Kelola pengguna dan peran akses aplikasi.",
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
