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

/**
 * Normalisasi nomor WA ke format wa.me (tanpa "+").
 * "+60..." dan "0060..." dianggap nomor internasional dan dipertahankan;
 * "08..." dan "8..." dianggap nomor Indonesia.
 */
export function waPhone(noWa: string): string {
	const raw = (noWa || "").trim();
	let digits = raw.replace(/[^\d]/g, "");
	if (!digits) return "";
	if (raw.startsWith("+")) return digits;
	if (digits.startsWith("00")) return digits.slice(2);
	if (digits.startsWith("62")) return digits;
	if (digits.startsWith("0")) return "62" + digits.slice(1);
	return "62" + digits;
}

/** true bila respons Inertia membawa flash error (server menolak aksi). */
export function adaFlashError(page: { props: Record<string, unknown> }): boolean {
	const flash = page.props.flash as { error?: string } | undefined;
	return Boolean(flash?.error || page.props.error);
}

export function waLink(noWa: string, text: string): string {
	const phone = waPhone(noWa);
	return phone ? `https://wa.me/${phone}?text=${encodeURIComponent(text)}` : "";
}

/** Ganti variabel {nama}, {nama_kelas}, {jadwal}, {level}, {guru} pada template. */
export function isiTemplate(body: string, vars: Record<string, string>): string {
	return body.replace(/\{(\w+)\}/g, (match, key: string) => (key in vars && vars[key] ? vars[key] : match));
}

/** "2026-08" -> "Agustus 2026" */
export function labelPeriode(periode: string): string {
	const d = new Date(periode + "-01T00:00:00");
	return Number.isNaN(d.getTime()) ? periode : d.toLocaleDateString("id-ID", { month: "long", year: "numeric" });
}

export async function salinTeks(text: string): Promise<boolean> {
	try {
		await navigator.clipboard.writeText(text);
		return true;
	} catch {
		return false;
	}
}
