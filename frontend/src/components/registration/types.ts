export interface FieldRule {
	tampil: boolean;
	wajib: boolean;
}
export interface PublicVarian {
	id: number;
	nama: string;
	harga: number;
}
export interface PublicLink {
	id: number;
	slug: string;
	judul: string;
	deskripsi: string;
	mode: "wa" | "short" | "full";
	after_submit: "wa" | "bayar";
	program_nama: string;
	butuh_jadwal: boolean;
	harga: number;
	varian: PublicVarian[];
	field_config: Record<string, FieldRule>;
	izinkan_manual: boolean;
	kode_referal?: boolean;
	meta?: { pixel_ids: string[]; view_event_id: string };
}
export interface JadwalOpsi {
	kode: string;
	label: string;
	gender: string;
	sisa_ikhwan: number;
	sisa_akhwat: number;
	catatan: string;
}
export interface DaftarForm {
	nama: string;
	no_wa: string;
	jenis_kelamin: "" | "L" | "P";
	usia: number | null;
	domisili: string;
	varian_id: number;
	jadwal_kode: string;
	riwayat_belajar: string;
	belajar_dimana: string;
	untuk_siapa: string;
	waktu_belajar: string;
	kode_referal: string;
	website: string;
}
export const formKosong = (): DaftarForm => ({
	nama: "", no_wa: "", jenis_kelamin: "", usia: null, domisili: "", varian_id: 0, jadwal_kode: "",
	riwayat_belajar: "", belajar_dimana: "", untuk_siapa: "", waktu_belajar: "", kode_referal: "", website: "",
});
export const rupiah = (n: number) => "Rp" + (n ?? 0).toLocaleString("id-ID");
