import type { AttendanceMonth } from "@lib/overview";

/** Shared card surface for every overview panel (same as the leads dashboard). */
export const panel = "rounded-xl border border-neutral-200/80 bg-white dark:border-white/[0.07] dark:bg-neutral-925/60";

/** Small uppercase heading used once per panel. */
export const kicker = "text-[11px] font-semibold uppercase tracking-[0.08em] text-neutral-500 dark:text-neutral-400";

export const focusRing = "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500";

const nf = new Intl.NumberFormat("id-ID");

/** "1.204" or "—" when the source is unavailable. */
export const count = (n: number | undefined | null) => (n === undefined || n === null || !Number.isFinite(n) ? "—" : nf.format(n));

export const money = (n: number | undefined | null) =>
	n === undefined || n === null || !Number.isFinite(n)
		? "—"
		: new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 0 }).format(n);

/** "Rp48,2 jt" for tight spots. */
export const moneyShort = (n: number | undefined | null) =>
	n === undefined || n === null || !Number.isFinite(n)
		? "—"
		: "Rp" + new Intl.NumberFormat("id-ID", { notation: "compact", maximumFractionDigits: 1 }).format(n);

/** "87,5%" from a 0–100 value. */
export const percent = (n: number | undefined | null) =>
	n === undefined || n === null || !Number.isFinite(n) ? "—" : new Intl.NumberFormat("id-ID", { maximumFractionDigits: 1 }).format(n) + "%";

/** Signed relative change in percent, or null when there is no base to compare. */
export function change(current: number | undefined, previous: number | undefined): number | null {
	if (current === undefined || previous === undefined || !previous) return null;
	return ((current - previous) / previous) * 100;
}

export const signed = (v: number, unit = "%") =>
	(v > 0 ? "+" : v < 0 ? "−" : "") + new Intl.NumberFormat("id-ID", { maximumFractionDigits: 1 }).format(Math.abs(v)) + unit;

export const monthShort = (m: string) =>
	new Intl.DateTimeFormat("id-ID", { month: "short", timeZone: "Asia/Jakarta" }).format(new Date(`${m}-01T00:00:00+07:00`));

export const monthLong = (m: string) =>
	new Intl.DateTimeFormat("id-ID", { month: "long", year: "numeric", timeZone: "Asia/Jakarta" }).format(new Date(`${m}-01T00:00:00+07:00`));

/** Records of a month with any attendance status. */
export const attendanceTotal = (a: AttendanceMonth) => a.hadir + a.izin + a.sakit + a.alpa;

/** Share of "hadir" among recorded attendance, 0–100, or null without records. */
export function attendanceRate(a: AttendanceMonth | undefined): number | null {
	if (!a) return null;
	const total = attendanceTotal(a);
	return total ? (a.hadir / total) * 100 : null;
}
