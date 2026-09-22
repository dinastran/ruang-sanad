/**
 * Shared frontend types.
 *
 * Must stay in sync with Go backend `app/models/dto.go` UserResponse.
 */

export interface User {
	id: number;
	email: string;
	name: string;
	avatar: string;
	role: string;
	email_verified: boolean;
}

/**
 * Common flash message shape from Inertia.
 * Backend sets via `s.store.Flash(c, "error", ...)` and `inertiaService.Render`
 * merges into `props.flash` automatically.
 */
export interface GuruKelas {
	id: number;
	guru_id?: number;
	guru_nama: string;
	nama_kelas: string;
	level: string;
	tipe: string;
	frekuensi: string;
	jadwal: string;
	jenis_kelamin: string;
	kapasitas: number;
	jumlah_santri: number;
	pertemuan_terakhir?: string;
	tanggal_terakhir?: string;
	materi_terakhir?: string;
	materi_individual?: boolean;
}

export interface SantriGuru {
	id: number;
	id_mahasantri: string;
	nama: string;
	usia?: number;
	domisili: string;
	no_wa: string;
	tanggal_mulai: string;
	persen_hadir: number;
	total_hadir: number;
	total_izin: number;
	total_sakit: number;
	total_alpa: number;
	total_telat: number;
	tanggal_hadir_terakhir: string;
	batas_materi_terakhir: string;
}

export interface PertemuanGuru {
	id: number;
	kelas_id: number;
	pertemuan_ke: number;
	pertemuan_level_ke?: number;
	level_nama?: string;
	tanggal: string;
	jam_mulai: string;
	jam_selesai: string;
	materi: string;
	catatan: string;
	batas_materi: string;
	is_reschedule: boolean;
	jadwal_semula: string;
	alasan_reschedule: string;
	is_badal: boolean;
	guru_pengganti_id?: number;
	alasan_badal: string;
	status: string;
}

export interface AbsensiGuru {
	id: number;
	pertemuan_id: number;
	santri_id: number;
	santri_nama: string;
	status: string;
	catatan: string;
}

export interface JadwalPertemuanGuru {
	id: number;
	kelas_id: number;
	kelas_nama: string;
	guru_utama_id?: number;
	guru_utama_nama: string;
	tanggal: string;
	jam_mulai: string;
	catatan: string;
	is_reschedule: boolean;
	jadwal_semula: string;
	alasan_reschedule: string;
	guru_pengganti_id?: number;
	guru_pengganti_nama: string;
	alasan_badal: string;
	jadwal_kelas_berubah?: boolean;
	status: string;
	pertemuan_id?: number;
	can_manage: boolean;
	can_start: boolean;
}

export interface GuruDashboard {
	total_kelas: number;
	total_santri: number;
	santri_aktif: number;
	total_pertemuan: number;
	jadwal_hari_ini: GuruKelas[];
	kelas_belum_absen: GuruKelas[];
}

export interface GuruDirectory {
	id: number;
	nama: string;
	gelar: string;
	jenis_kelamin: string;
	status: string;
	no_wa: string;
	email: string;
	tanggal_gabung: string;
	foto: string;
	is_aktif: boolean;
	user_id?: number;
	total_kelas: number;
	total_santri: number;
}

export interface LinkableUser {
	id: number;
	name: string;
	email: string;
	role: string;
}

export interface GuruRingkas {
	id: number;
	nama: string;
	no_wa: string;
}

export interface KoordinatorDashboard {
	total_guru: number;
	guru_tetap: number;
	guru_part_time: number;
	total_kelas: number;
	total_santri: number;
	guru_belum_absen: GuruRingkas[];
}

export interface Kompetensi {
	hafalan_quran: number;
	hafalan_tuhfah: number;
	hafalan_jazariy: number;
	hafalan_khaqaniy: number;
	hafalan_syakhawiy: number;
	sanad_qiroah: number;
	bahasa_arab_pasif: number;
	bahasa_arab_aktif: number;
}

export interface Riayah {
	id: number;
	target_type: string;
	target_id: number;
	target_nama?: string;
	catatan: string;
	penulis_nama: string;
	created_at: string;
}

export interface AbsenRow {
	guru_id: number;
	nama: string;
	status: string;
	hadir: boolean;
	jam_masuk: string;
	keterangan: string;
	alasan: string;
}

export interface Pembinaan {
	id: number;
	tanggal: string;
	bulan: string;
	pekan_ke: number;
	topik: string;
	keterangan: string;
	status: string;
}

export interface Rapat {
	id: number;
	tanggal: string;
	judul: string;
	catatan: string;
	status: string;
}

export interface Kunjungan {
	id: number;
	guru_id: number;
	guru_nama: string;
	kelas_id?: number;
	kelas_nama: string;
	target_mulai: string;
	target_selesai: string;
	tanggal: string;
	jam: string;
	status: string;
	catatan: string;
}

export interface Kalam {
	id: number;
	tanggal: string;
	topik: string;
	kitab: string;
	keterangan: string;
	total_share: number;
}

export interface KalamShareRow {
	guru_id: number;
	nama: string;
	sudah_share: boolean;
}

export interface WaTemplate {
	id: number;
	nama: string;
	target_type: string;
	body: string;
	is_aktif: boolean;
}

export interface TilawahStatus {
	sudah_hari_ini: boolean;
	bulan_ini: number;
	tanggal: string[];
}

export interface KelasSimple {
	id: number;
	nama_kelas: string;
}

export interface MonitoringKelasSummary {
	total: number;
	belum_mulai: number;
	berlangsung: number;
	selesai: number;
	dibatalkan: number;
	perlu_tindakan: number;
}

export interface MonitoringKelasAttendance {
	santri_id: number;
	santri_nama: string;
	id_mahasantri: string;
	attendance_status: string;
	attendance_note: string;
	batas_materi: string;
}

export interface MonitoringKelasNote {
	id: number;
	note: string;
	author_name: string;
	created_at: string;
}

export interface MonitoringKelasActivity {
	id: number;
	action: string;
	details: string;
	actor_name: string;
	created_at: string;
}

export interface MonitoringKelasItem {
	schedule_id: number;
	tanpa_jadwal: boolean;
	kelas_id: number;
	nama_kelas: string;
	angkatan: string;
	level: string;
	frekuensi: string;
	jadwal_kelas: string;
	materi_individual: boolean;
	tanggal: string;
	jam_mulai: string;
	schedule_note: string;
	status: "belum_mulai" | "berlangsung" | "selesai" | "dibatalkan" | "terlambat";
	schedule_status: string;
	is_reschedule: boolean;
	jadwal_semula: string;
	alasan_reschedule: string;
	guru_utama_id?: number;
	guru_utama_nama: string;
	guru_pengganti_id?: number;
	guru_pengganti_nama: string;
	alasan_badal: string;
	assigned_user_id?: number;
	pertemuan_id?: number;
	actual_start_time: string;
	actual_end_time: string;
	teacher_checked_in_at: string;
	teacher_attendance: "belum_hadir" | "hadir" | "terlambat" | "tidak_berlaku";
	student_attendance: "belum_tersedia" | "belum_lengkap" | "lengkap" | "tidak_berlaku";
	active_student_count: number;
	attendance_count: number;
	materi: string;
	meeting_note: string;
	needs_action: boolean;
	attendance: MonitoringKelasAttendance[];
	notes: MonitoringKelasNote[];
	activities: MonitoringKelasActivity[];
}

export interface MonitoringKelasData {
	summary: MonitoringKelasSummary;
	items: MonitoringKelasItem[];
}

export interface MonitoringKelasFilters {
	StartDate: string;
	EndDate: string;
	GuruID: number;
	KelasID: number;
	Status: string;
}

export interface AppNotification {
	id: number;
	type: string;
	title: string;
	message: string;
	action_url: string;
	read: boolean;
	created_at: string;
}

export interface TsiKriteriaNilai {
	kriteria_id: number;
	kategori: string;
	bobot: number;
	urutan: number;
	sumber: string;
	kode: string;
	nama: string;
	nilai: number | null;
	auto_nilai: number | null;
	raw_display: string;
	is_override: boolean;
	alasan_override: string;
	catatan: string;
}

export interface TsiKategoriSkor {
	kategori: string;
	bobot: number;
	rata_rata: number | null;
	skor: number;
	terisi: number;
	total: number;
}

export interface TsiPenilaian {
	periode_id: number;
	guru_id: number;
	guru_nama: string;
	bulan: string;
	status: string;
	kriteria: TsiKriteriaNilai[];
	kategori: TsiKategoriSkor[];
	total: number;
	predikat: string;
}

export interface Todo {
	id: number;
	judul: string;
	teknis: string;
	kebutuhan: string;
	deadline: string;
	pic: string;
	status: string;
	recurring: string;
	link_pendukung: string;
	catatan: string;
}

export interface TsiRekapRow {
	guru_id: number;
	guru_nama: string;
	guru_status: string;
	bulan: string;
	status: string;
	kompetensi: number;
	kepuasan: number;
	kedisiplinan: number;
	kontribusi: number;
	total: number;
	predikat: string;
}

export interface Riayah {
	id: number;
	target_type: string;
	target_id: number;
	target_nama: string;
	catatan: string;
	penulis_nama: string;
	created_at: string;
}

export interface Flash {
	error?: string;
	success?: string;
	warning?: string;
	info?: string;
}

export interface Tagihan {
	id: number;
	santri_id: number;
	santri_nama: string;
	id_mahasantri: string;
	no_wa: string;
	kelas_id?: number;
	kelas_nama: string;
	guru_nama: string;
	angkatan: string;
	angkatan_kelas: string;
	frekuensi: string;
	bulan_ke: number;
	pertemuan_ke: number;
	nominal: number;
	tanggal_tagih: string;
	jatuh_tempo: string;
	status: string;
	tanggal_bayar: string;
	metode: string;
	catatan: string;
	fu_terakhir: string;
	fu_count: number;
}

export interface RingkasanKeuangan {
	total_tagihan: number;
	nominal_tagihan: number;
	total_lunas: number;
	nominal_lunas: number;
	total_belum_bayar: number;
	nominal_belum_bayar: number;
	total_terlambat: number;
	kolektibilitas: number;
}
