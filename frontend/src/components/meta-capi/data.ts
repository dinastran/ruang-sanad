export interface EventRow {
 id:number; lead_id:number; lead_nama:string; event:string; event_id:string; fbtrace_id:string;
 event_time:string; value:number; status:string; attempts:number; last_error:string;
 test_mode:boolean; sent_at:string;
}
export interface CAPIStatus {
 enabled:boolean; dataset_id:string; api_version:string; test_mode:boolean; app_secret_proof:boolean;
 counts:Record<string,number>; resendable_test:number; event_source_url:string;
 sources_without_url:string[]; mapping:{event:string;tahap:string}[]; events:EventRow[];
}
export interface CapiDay {
 day:string; sent:number; test_sent:number; failed:number; pending:number; skipped:number;
}
export interface CapiEventSummary {
 event:string; total:number; sent:number; test_sent:number; failed:number; pending:number; skipped:number; purchase_value:number;
}
export interface CapiAnalytics {
 from:string;to:string;total:number;sent:number;test_sent:number;pending:number;failed:number;skipped:number;
 purchase_value:number;days:CapiDay[];events:CapiEventSummary[];
}
export const statLabels:Record<string,string>={
 sent:"Terkirim",pending:"Menunggu",failed:"Gagal",skipped:"Dilewati"
};
export const eventLabels:Record<string,string>={
 Lead:"Lead masuk",CompleteRegistration:"Data terisi",
 InitiateCheckout:"Info pembayaran",AddPaymentInfo:"Bukti masuk",Purchase:"Jadi santri"
};
export const currency=(value:number)=>"Rp"+(value||0).toLocaleString("id-ID");
export const prettyDay=(day:string)=>new Date(day+"T12:00:00+07:00").toLocaleDateString("id-ID",{day:"numeric",month:"short",timeZone:"Asia/Jakarta"});
