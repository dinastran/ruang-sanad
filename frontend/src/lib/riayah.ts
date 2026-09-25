// Utilitas halaman riayah santri.

/** Tanggal hari ini (zona waktu perangkat) dalam format YYYY-MM-DD. */
export function todayISO(): string {
	const d = new Date();
	const pad = (n: number) => String(n).padStart(2, "0");
	return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}

export function formatTanggal(t: string): string {
	if (!t) return "–";
	const d = new Date(t.slice(0, 10) + "T00:00:00");
	return Number.isNaN(d.getTime()) ? t : d.toLocaleDateString("id-ID", { day: "numeric", month: "short", year: "numeric" });
}

/** "hari ini", "kemarin", "5 hari lalu", atau "" jika kosong. */
export function hariLalu(t: string): string {
	if (!t) return "";
	const d = new Date(t.slice(0, 10) + "T00:00:00");
	const today = new Date(todayISO() + "T00:00:00");
	const selisih = Math.round((today.getTime() - d.getTime()) / 86_400_000);
	if (Number.isNaN(selisih)) return t;
	if (selisih <= 0) return "hari ini";
	if (selisih === 1) return "kemarin";
	return `${selisih} hari lalu`;
}

export const mediaKontak = [
	{ value: "wa", label: "WhatsApp" },
	{ value: "telepon", label: "Telepon" },
	{ value: "tatap_muka", label: "Tatap muka" },
	{ value: "lainnya", label: "Lainnya" },
] as const;

/** Normalisasi nomor WA Indonesia ke format wa.me (tanpa "+"). */
export function waPhone(noWa: string): string {
	let digits = (noWa || "").replace(/[^\d]/g, "");
	if (!digits) return "";
	if (digits.startsWith("0")) digits = "62" + digits.slice(1);
	else if (!digits.startsWith("62")) digits = "62" + digits;
	return digits;
}

export function waLink(noWa: string, text: string): string {
	const phone = waPhone(noWa);
	return phone ? `https://wa.me/${phone}?text=${encodeURIComponent(text)}` : "";
}
