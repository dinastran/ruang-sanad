/**
 * Data layer of the Performa Iklan dashboard. The server sends raw daily rows
 * (per ad, and CRM leads per source) for the period and the one before it;
 * everything shown is derived here from (rows + filters), so a filter moves
 * every number, chart and table at once.
 */

export interface SyncInfo { trigger: string; status: string; rows: number; error: string; date_from: string; date_to: string; started_at: string; finished_at: string }
export interface AdRow { d: string; ad: string; as: string; c: string; sp: number; im: number; cl: number; lk: number; ms: number }
export interface AdInfo { id: string; name: string; adset_id: string; adset_name: string; campaign_id: string }
export interface CampaignInfo { id: string; name: string; last_date: string; source_id: number }
export interface LeadDay { d: string; s: number; l: number; w: number }
export interface SourceOption { id: number; nama: string; is_aktif: boolean }
export interface Report {
	from: string; to: string; prev_from: string; prev_to: string;
	enabled: boolean; accounts: string[]; last_run: SyncInfo | null; last_success: SyncInfo | null;
	rows: AdRow[]; ads: AdInfo[]; campaigns: CampaignInfo[]; lead_days: LeadDay[]; sources: SourceOption[];
}

export interface Filters {
	/** Selected campaign IDs; empty = all. */
	campaigns: string[];
	/** Lead source id; 0 = all. */
	source: number;
}

export interface Metrics { spend: number; impressions: number; clicks: number; link: number; chats: number; leads: number; won: number }
export const emptyMetrics = (): Metrics => ({ spend: 0, impressions: 0, clicks: 0, link: 0, chats: 0, leads: 0, won: 0 });

/** The data a view works with after filters. */
export interface Scope {
	campaignIds: Set<string>;
	/** Lead sources counted on the CRM side (mapped to a campaign in scope). */
	sourceIds: Set<number>;
	/** True when at least one campaign in scope is mapped to a lead source. */
	hasMapping: boolean;
}

export function scopeOf(r: Report, f: Filters): Scope {
	let campaigns = r.campaigns;
	if (f.campaigns.length) campaigns = campaigns.filter((c) => f.campaigns.includes(c.id));
	if (f.source) campaigns = campaigns.filter((c) => c.source_id === f.source);
	const sourceIds = new Set(campaigns.filter((c) => c.source_id > 0).map((c) => c.source_id));
	return { campaignIds: new Set(campaigns.map((c) => c.id)), sourceIds, hasMapping: sourceIds.size > 0 };
}

const inRange = (d: string, from: string, to: string) => d >= from && d <= to;

export function totals(r: Report, s: Scope, from: string, to: string): Metrics {
	const m = emptyMetrics();
	for (const row of r.rows) {
		if (!inRange(row.d, from, to) || !s.campaignIds.has(row.c)) continue;
		m.spend += row.sp; m.impressions += row.im; m.clicks += row.cl; m.link += row.lk; m.chats += row.ms;
	}
	for (const l of r.lead_days) {
		if (!inRange(l.d, from, to) || !s.sourceIds.has(l.s)) continue;
		m.leads += l.l; m.won += l.w;
	}
	return m;
}

/** Days from..to inclusive as YYYY-MM-DD. */
export function daysBetween(from: string, to: string): string[] {
	const out: string[] = [];
	const d = new Date(from + "T00:00:00Z");
	const end = new Date(to + "T00:00:00Z");
	while (d <= end) {
		out.push(d.toISOString().slice(0, 10));
		d.setUTCDate(d.getUTCDate() + 1);
	}
	return out;
}

export function daily(r: Report, s: Scope, from: string, to: string): Metrics[] {
	const days = daysBetween(from, to);
	const idx = new Map(days.map((d, i) => [d, i]));
	const out = days.map(() => emptyMetrics());
	for (const row of r.rows) {
		const i = idx.get(row.d);
		if (i === undefined || !s.campaignIds.has(row.c)) continue;
		const m = out[i];
		m.spend += row.sp; m.impressions += row.im; m.clicks += row.cl; m.link += row.lk; m.chats += row.ms;
	}
	for (const l of r.lead_days) {
		const i = idx.get(l.d);
		if (i === undefined || !s.sourceIds.has(l.s)) continue;
		out[i].leads += l.l; out[i].won += l.w;
	}
	return out;
}

export type Level = "campaign" | "adset" | "ad";

export interface Entity extends Metrics {
	id: string;
	name: string;
	parent: string;
	campaignId: string;
	/** Lead source of the campaign (0 = not mapped). */
	sourceId: number;
	/** Campaigns in scope sharing that source; CRM numbers are per source. */
	sharedWith: number;
}

/** Ad metrics grouped by level for the period; CRM leads only at campaign level. */
export function entities(r: Report, s: Scope, level: Level, from: string, to: string): Entity[] {
	const adInfo = new Map(r.ads.map((a) => [a.id, a]));
	const campInfo = new Map(r.campaigns.map((c) => [c.id, c]));
	const groups = new Map<string, Entity>();
	for (const row of r.rows) {
		if (!inRange(row.d, from, to) || !s.campaignIds.has(row.c)) continue;
		const key = level === "campaign" ? row.c : level === "adset" ? row.as : row.ad;
		let e = groups.get(key);
		if (!e) {
			const camp = campInfo.get(row.c);
			const ad = adInfo.get(row.ad);
			const name = level === "campaign" ? camp?.name : level === "adset" ? ad?.adset_name : ad?.name;
			const parent = level === "campaign" ? "" : level === "adset" ? (camp?.name ?? "") : `${camp?.name ?? ""} › ${ad?.adset_name ?? ""}`;
			e = { ...emptyMetrics(), id: key, name: name || key, parent, campaignId: row.c, sourceId: camp?.source_id ?? 0, sharedWith: 0 };
			groups.set(key, e);
		}
		e.spend += row.sp; e.impressions += row.im; e.clicks += row.cl; e.link += row.lk; e.chats += row.ms;
	}
	const list = [...groups.values()];
	if (level === "campaign") {
		const perSource = new Map<number, Metrics>();
		for (const l of r.lead_days) {
			if (!inRange(l.d, from, to) || !s.sourceIds.has(l.s)) continue;
			const m = perSource.get(l.s) ?? emptyMetrics();
			m.leads += l.l; m.won += l.w;
			perSource.set(l.s, m);
		}
		const shared = new Map<number, number>();
		for (const e of list) if (e.sourceId) shared.set(e.sourceId, (shared.get(e.sourceId) ?? 0) + 1);
		for (const e of list) {
			if (!e.sourceId) continue;
			const m = perSource.get(e.sourceId);
			e.leads = m?.leads ?? 0; e.won = m?.won ?? 0;
			e.sharedWith = shared.get(e.sourceId) ?? 1;
		}
	}
	return list;
}

export interface SourceStat extends Metrics { id: number; nama: string; campaigns: number }

/** Mapped lead sources in scope: spend of their campaigns vs their leads. */
export function sourceStats(r: Report, s: Scope, from: string, to: string): SourceStat[] {
	const names = new Map(r.sources.map((x) => [x.id, x.nama]));
	const campSource = new Map(r.campaigns.map((c) => [c.id, c.source_id]));
	const stats = new Map<number, SourceStat>();
	const get = (id: number) => {
		let st = stats.get(id);
		if (!st) { st = { ...emptyMetrics(), id, nama: names.get(id) ?? `Sumber ${id}`, campaigns: 0 }; stats.set(id, st); }
		return st;
	};
	for (const c of r.campaigns) if (s.campaignIds.has(c.id) && c.source_id > 0) get(c.source_id).campaigns++;
	for (const row of r.rows) {
		if (!inRange(row.d, from, to) || !s.campaignIds.has(row.c)) continue;
		const src = campSource.get(row.c) ?? 0;
		if (!src) continue;
		const st = get(src);
		st.spend += row.sp; st.impressions += row.im; st.clicks += row.cl; st.link += row.lk; st.chats += row.ms;
	}
	for (const l of r.lead_days) {
		if (!inRange(l.d, from, to) || !s.sourceIds.has(l.s)) continue;
		const st = get(l.s);
		st.leads += l.l; st.won += l.w;
	}
	return [...stats.values()].sort((a, b) => b.spend - a.spend || a.nama.localeCompare(b.nama));
}

// ---- formatting ----

const nf = new Intl.NumberFormat("id-ID");
export const num = (n: number) => (Number.isFinite(n) ? nf.format(Math.round(n)) : "—");
export const rp = (n: number) => (Number.isFinite(n) ? "Rp" + nf.format(Math.round(n)) : "—");
/** "Rp10,3 jt", "Rp645 rb" for tight spots. */
export const rpShort = (n: number) => {
	if (!Number.isFinite(n)) return "—";
	const abs = Math.abs(n);
	const f = (v: number, u: string) => "Rp" + v.toLocaleString("id-ID", { maximumFractionDigits: v < 10 ? 1 : 0 }) + " " + u;
	if (abs >= 1e9) return f(n / 1e9, "M");
	if (abs >= 1e6) return f(n / 1e6, "jt");
	if (abs >= 1e3) return f(n / 1e3, "rb");
	return rp(n);
};
export const compact = (n: number) => (Number.isFinite(n) ? new Intl.NumberFormat("id-ID", { notation: "compact", maximumFractionDigits: 1 }).format(n) : "—");
/** a / b or NaN when there is nothing to divide by. */
export const ratio = (a: number, b: number) => (b > 0 ? a / b : NaN);
export const pct = (v: number, digits = 1) => (Number.isFinite(v) ? (v * 100).toLocaleString("id-ID", { maximumFractionDigits: digits }) + "%" : "—");
/** Relative change; NaN without a previous value. */
export const change = (cur: number, prev: number) => (Number.isFinite(cur) && Number.isFinite(prev) && prev > 0 ? (cur - prev) / prev : NaN);
export const signedPct = (v: number) => (v > 0 ? "+" : v < 0 ? "−" : "") + Math.abs(v * 100).toLocaleString("id-ID", { maximumFractionDigits: 1 }) + "%";

export function shortDay(d: string) {
	return new Date(d + "T12:00:00+07:00").toLocaleDateString("id-ID", { day: "numeric", month: "short", timeZone: "Asia/Jakarta" });
}
