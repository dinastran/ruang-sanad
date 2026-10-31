import type { LeadLabel } from "@lib/leads";

export interface LeadItem {
	id: number;
	nama: string;
	wa_name: string;
	phone: string;
	stage: string;
	status: string;
	paket_kode: string;
	needs_review: boolean;
	ai_paused: boolean;
	assigned_user_id: number;
	assigned_name: string;
	last_inbound_at: string;
	last_activity_at: string;
	labels: LeadLabel[];
}

export interface Filters {
	status: string;
	stage: string;
	label: number;
	mine: boolean;
	review: boolean;
	q: string;
	from: string;
	to: string;
	date_field?: string;
	reached_stage?: string;
	lost_reason?: string;
	/** Lead source id, or "none" for leads without a source. */
	sumber?: string;
}

/** A lead source (landing page / iklan) for selects and filters. */
export interface SourceOption {
	id: number;
	nama: string;
	is_aktif: boolean;
}

export type FilterChange = Partial<Record<keyof Filters, string | number | boolean>>;

/** Columns for closed leads; open leads are grouped by their stage. */
export const CLOSED_COLUMNS = ["won", "lost", "do_not_contact"];
