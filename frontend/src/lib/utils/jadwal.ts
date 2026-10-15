/**
 * Format jadwal untuk label singkat, mis. "Sen, 13 Okt 08.00".
 * Nilai yang tidak bisa diparse dikembalikan apa adanya.
 */
export function formatTanggalJam(tanggal: string, jam = ""): string {
	const date = new Date(`${tanggal}T00:00:00`);
	if (!tanggal || Number.isNaN(date.getTime())) return [tanggal, jam].filter(Boolean).join(" ");
	const tgl = date.toLocaleDateString("id-ID", { weekday: "short", day: "numeric", month: "short" });
	return jam ? `${tgl} ${jam.slice(0, 5).replace(":", ".")}` : tgl;
}

/** Format kolom jadwal_semula ("YYYY-MM-DD HH:MM") menjadi label singkat. */
export function formatJadwalSemula(value: string): string {
	const v = (value || "").trim();
	if (/^\d{4}-\d{2}-\d{2}/.test(v)) return formatTanggalJam(v.slice(0, 10), v.slice(10).trim());
	return v;
}
