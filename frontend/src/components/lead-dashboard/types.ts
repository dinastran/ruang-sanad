export interface Count { key: string; count: number }
export interface FunnelStage { stage: string; reached: number; conversion: number }
export interface Report {
	total_leads: number; won: number; lost: number; open: number; do_not_contact: number;
	funnel: FunnelStage[]; lost_reasons: Count[]; objections: Count[];
	median_hours_to_won: number; followup_sent: number; followup_won: number;
}
export interface Day { day: string; leads: number; closing: number }
export interface Priority { id: number; name: string; stage: string; reason: string; activity_at: string }
export interface Dashboard {
	from: string; to: string; report: Report; closing_period: number;
	previous_leads: number; previous_closing: number; review_count: number; daily: Day[];
	followup_pending: number; followup_approved: number; followup_failed: number;
	followup_eligible: number; followup_replied: number; priorities: Priority[]; insights: string[];
}

/** Shared card surface for every dashboard panel. */
export const panel = "rounded-xl border border-neutral-200/80 bg-white dark:border-white/[0.07] dark:bg-neutral-925/60";

export const fmt = (n: number) => (Number.isFinite(n) ? n : 0).toLocaleString("id-ID");

/** "12,5%" or "—" when there is no denominator. */
export function pct(n: number, total: number): string {
	if (!total) return "—";
	return (n / total * 100).toFixed(1).replace(".", ",") + "%";
}

/** Signed change vs the previous period; null when there is nothing to compare. */
export function delta(current: number, previous: number): number | null {
	if (!previous) return null;
	return (current - previous) / previous * 100;
}

export function formatDelta(v: number): string {
	return (v > 0 ? "+" : "") + v.toFixed(1).replace(".", ",") + "%";
}

/** Median closing time: "18 jam", "2,4 hari", or "—". */
export function formatHours(h: number): string {
	if (!h || !Number.isFinite(h)) return "—";
	if (h < 1) return "< 1 jam";
	if (h < 48) return Math.round(h) + " jam";
	return (h / 24).toFixed(1).replace(".", ",") + " hari";
}

export function shortDate(day: string): string {
	return new Date(day + "T12:00:00+07:00").toLocaleDateString("id-ID", { day: "numeric", month: "short", timeZone: "Asia/Jakarta" });
}

export function longDate(day: string): string {
	return new Date(day + "T12:00:00+07:00").toLocaleDateString("id-ID", { weekday: "short", day: "numeric", month: "short", timeZone: "Asia/Jakarta" });
}
