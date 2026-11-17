// Format tanggal & angka untuk dashboard tasbih (locale id-ID).
export const angka = (n: number) => Math.round(n).toLocaleString("id-ID");

export function tglPendek(iso: string): string {
	const [y, m, d] = iso.split("-").map(Number);
	return new Intl.DateTimeFormat("id-ID", { day: "numeric", month: "short" }).format(new Date(y, m - 1, d));
}

export function tglPanjang(iso: string): string {
	const [y, m, d] = iso.split("-").map(Number);
	return new Intl.DateTimeFormat("id-ID", { weekday: "long", day: "numeric", month: "long", year: "numeric" }).format(new Date(y, m - 1, d));
}

export function durasi(detik: number): string {
	if (detik < 60) return `${detik} dtk`;
	const m = Math.floor(detik / 60);
	return m < 60 ? `${m} mnt` : `${Math.floor(m / 60)} j ${m % 60} mnt`;
}
