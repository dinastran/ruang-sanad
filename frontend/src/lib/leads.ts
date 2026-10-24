/**
 * Shared labels and helpers for the CRM Leads pages.
 * Must stay in sync with Go `app/services/lead*.go` and `app/models/lead.go`.
 */

export const STAGE_LABELS: Record<string, string> = {
	baru: "Baru",
	discovery: "Discovery",
	rekomendasi: "Rekomendasi",
	form_dikirim: "Form dikirim",
	data_terisi: "Data terisi",
	info_bayar_dikirim: "Info bayar dikirim",
	bukti_masuk: "Bukti transfer masuk",
	terverifikasi: "Terverifikasi",
	won: "Jadi santri",
};

export const STATUS_LABELS: Record<string, string> = {
	open: "Berjalan",
	won: "Jadi santri",
	lost: "Hilang",
	do_not_contact: "Jangan hubungi",
};

export const LOST_REASON_LABELS: Record<string, string> = {
	harga: "Harga",
	jadwal: "Jadwal",
	belum_prioritas: "Belum prioritas",
	keluarga: "Keluarga",
	tidak_cocok: "Tidak cocok",
	tidak_merespons: "Tidak merespons",
	lainnya: "Lainnya",
};

export const SENDER_LABELS: Record<string, string> = {
	lead: "Calon santri",
	ai: "CS AI",
	admin: "Admin (HP)",
	followup: "Follow-up",
};

export const UNTUK_SIAPA_LABELS: Record<string, string> = {
	"": "Belum diketahui",
	diri_sendiri: "Diri sendiri",
	anak: "Anak",
	keluarga: "Keluarga",
};

export const FIELD_LABELS: Record<string, string> = {
	nama: "Nama",
	usia: "Usia",
	domisili: "Domisili",
	jenis_kelamin: "Ikhwan/Akhwat",
	paket_kode: "Paket",
	kelas_kode: "Kode kelas",
	waktu_belajar: "Waktu belajar",
	riwayat_belajar: "Pernah belajar",
	belajar_dimana: "Belajar di mana",
	untuk_siapa: "Untuk siapa",
};

/** Tailwind classes per stage, using theme tokens only. */
export function stageTone(stage: string, status = "open"): string {
	if (status === "won") return "bg-success/10 text-success";
	if (status === "lost") return "bg-neutral-500/10 text-neutral-600 dark:text-neutral-400";
	if (status === "do_not_contact") return "bg-error/10 text-error";
	switch (stage) {
		case "bukti_masuk":
			return "bg-warning/15 text-warning";
		case "terverifikasi":
			return "bg-success/10 text-success";
		case "info_bayar_dikirim":
		case "data_terisi":
			return "bg-brand-500/10 text-brand-700 dark:text-brand-300";
		default:
			return "bg-neutral-500/10 text-neutral-700 dark:text-neutral-300";
	}
}

export function leadName(l: { nama?: string; wa_name?: string; phone?: string; id: number }): string {
	return l.nama || l.wa_name || l.phone || `Lead L-${l.id}`;
}

export function formatRupiah(n: number): string {
	return "Rp" + (n || 0).toLocaleString("id-ID");
}

/** "5 menit lalu", "3 jam lalu", "2 hari lalu", or a date. */
export function relativeTime(iso: string): string {
	if (!iso) return "";
	const t = new Date(iso).getTime();
	if (Number.isNaN(t)) return "";
	const diff = Math.max(0, Date.now() - t);
	const min = Math.floor(diff / 60000);
	if (min < 1) return "baru saja";
	if (min < 60) return `${min} menit lalu`;
	const hours = Math.floor(min / 60);
	if (hours < 24) return `${hours} jam lalu`;
	const days = Math.floor(hours / 24);
	if (days < 7) return `${days} hari lalu`;
	return formatDateTime(iso, false);
}

export function formatDateTime(iso: string, withTime = true): string {
	if (!iso) return "";
	const d = new Date(iso);
	if (Number.isNaN(d.getTime())) return "";
	return d.toLocaleString("id-ID", {
		timeZone: "Asia/Jakarta",
		day: "numeric",
		month: "short",
		year: "numeric",
		...(withTime ? { hour: "2-digit", minute: "2-digit" } : {}),
	});
}

/** Today's date in WIB as YYYY-MM-DD. */
export function todayWIB(): string {
	return new Date().toLocaleDateString("en-CA", { timeZone: "Asia/Jakarta" });
}

export function waLink(phone: string): string {
	return `https://wa.me/${phone.replace(/\D/g, "")}`;
}

export interface LeadLabel {
	id: number;
	nama: string;
	warna: string;
}

export function labelTone(warna: string): string {
	switch (warna) {
		case "brand":
			return "border-brand-500/30 text-brand-700 dark:text-brand-300";
		case "secondary":
			return "border-secondary-500/30 text-secondary-700 dark:text-secondary-300";
		default:
			return "border-neutral-300 text-neutral-700 dark:border-neutral-700 dark:text-neutral-300";
	}
}

export const FOLLOWUP_STATUS_LABELS: Record<string, string> = {
	pending_approval: "Menunggu persetujuan",
	approved: "Disetujui",
	sent: "Terkirim",
	cancelled: "Dibatalkan",
	skipped: "Dilewati",
	failed: "Gagal",
};

export function followupStatusTone(status: string): string {
	switch (status) {
		case "pending_approval":
			return "bg-warning/10 text-warning";
		case "approved":
			return "bg-brand-500/10 text-brand-700 dark:text-brand-300";
		case "sent":
			return "bg-success/10 text-success";
		default:
			return "bg-neutral-200/70 text-neutral-600 dark:bg-neutral-800 dark:text-neutral-400";
	}
}

/** ISO time → "YYYY-MM-DDTHH:MM" in WIB, for <input type="datetime-local"> (server reads it as WIB). */
export function toWIBInput(iso: string): string {
	const d = iso ? new Date(iso) : new Date();
	if (Number.isNaN(d.getTime())) return "";
	const parts = new Intl.DateTimeFormat("en-CA", {
		timeZone: "Asia/Jakarta", year: "numeric", month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit", hourCycle: "h23",
	}).formatToParts(d);
	const get = (t: string) => parts.find((p) => p.type === t)?.value ?? "";
	return `${get("year")}-${get("month")}-${get("day")}T${get("hour")}:${get("minute")}`;
}

/** Tomorrow 09:00 WIB as a datetime-local value. */
export function tomorrowMorningWIB(): string {
	const d = new Date(`${todayWIB()}T09:00:00+07:00`);
	d.setUTCDate(d.getUTCDate() + 1);
	return toWIBInput(d.toISOString());
}
