<script lang="ts">
	import { onMount } from "svelte";
	import { getCSRFToken } from "@lib/utils/csrf";
	import { ClipboardCheck, RefreshCcw, CheckCircle2 } from "lucide-svelte";

	type WorkflowPlan = {
		id:number; source:string; status:string; program:string; angkatan:string;
		gender:string; guru_id:number; frequency:number; min_quota:number; capacity:number;
		target_kelas_id:number; version:number; coordinator_confirmed_by:number;
		coordinator_confirmed_version:number; admin_approved_by:number; student_count:number; reserved_count:number; level_code:string; slots_label:string; exception_reason:string; exception_status:string;
	};
	let { mode }: {mode:"admin"|"coordinator"|"super_admin"}=$props();
	let items=$state<WorkflowPlan[]>([]);
	let loading=$state(false);
	let busy=$state<number|null>(null);
	let message=$state("");
	let failure=$state("");
	const labels:Record<string,string>={
		draft:"Draft",pending_coordinator:"Menunggu Koordinator Guru",
		pending_admin:"Menunggu approval Admin Kelas",
		pending_exception:"Perlu exception kuota",blocked:"Diblokir",
	};
	async function load() {
		if(loading)return;
		loading=true;failure="";
		try{
			const r=await fetch("/app/onboarding/planning/plans",{credentials:"same-origin",headers:{Accept:"application/json"}});
			if(!r.ok)throw new Error("Tidak bisa memuat daftar rencana kelas");
			const data=await r.json() as {items:WorkflowPlan[]};
			items=data.items??[];
		}catch(e){failure=e instanceof Error?e.message:"Gagal memuat perencanaan"}
		finally{loading=false}
	}
	onMount(()=>{void load()});
	async function submit(plan:WorkflowPlan,action:string) {
		if(busy!==null)return;
		const labels:Record<string,string>={submit:"Ajukan rencana",coordinator_confirm:"Konfirmasi penugasan guru",admin_approve:"Setujui untuk proses aktivasi berikutnya",exception_approve:"Setujui exception kuota",exception_reject:"Tolak exception kuota",reserve:"Reservasi kandidat yang sudah disetujui",cancel:"Batalkan rencana dan lepas seluruh reservasi"};
		let reason=labels[action]??action;
		const needsDecisionNote=action.startsWith("exception_")||action==="cancel"||(action==="submit"&&plan.student_count<plan.min_quota);
		if(needsDecisionNote){
			const entered=prompt(action==="submit"?"Jelaskan mengapa kelas di bawah 8 santri perlu pengecualian (minimal 10 karakter):":"Masukkan alasan keputusan exception (minimal 10 karakter):","");
			if(entered===null)return;
			reason=entered.trim();
			if(reason.length<10){failure="Alasan exception minimal 10 karakter.";return;}
		}
		if(!confirm((labels[action]??action)+"? Persetujuan ini belum mengaktifkan kelas."))return;
		busy=plan.id;failure="";message="";
		try {
			const r=await fetch("/app/onboarding/planning/plans/"+plan.id+"/actions",{
				method:"POST",credentials:"same-origin",
				headers:{"Content-Type":"application/json",Accept:"application/json","X-XSRF-TOKEN":getCSRFToken()},
				body:JSON.stringify({action,version:plan.version,reason}),
			});
			if(!r.ok){
				const data=await r.json().catch(()=>({error:"Aksi tidak berhasil"})) as {error?:string};
				throw new Error(data.error??"Gagal menyimpan approval");
			}
			message="Status rencana #"+plan.id+" berhasil diperbarui.";
			await load();
		}catch(e){failure=e instanceof Error?e.message:"Gagal menyimpan approval"}
		finally{busy=null}
	}
</script>

<section class="rounded-xl border border-neutral-200 bg-white p-4 dark:border-white/10 dark:bg-neutral-900" aria-label="Workflow persetujuan perencanaan kelas">
	<div class="flex flex-wrap items-center justify-between gap-2">
		<div>
			<h2 class="flex items-center gap-2 font-semibold"><ClipboardCheck class="h-4 w-4"/> Persetujuan Rencana Kelas</h2>
			<p class="mt-1 text-xs text-neutral-500">Draft, konfirmasi guru, dan approval. Tidak mengaktifkan kelas atau memindahkan santri.</p>
		</div>
		<button type="button" onclick={load} disabled={loading||busy!==null} class="inline-flex items-center gap-1 rounded-lg border border-neutral-300 px-3 py-2 text-xs font-medium dark:border-neutral-700 disabled:opacity-50"><RefreshCcw class="h-3.5 w-3.5"/>{loading?"Memuat...":"Muat ulang"}</button>
	</div>
	{#if failure}<p role="alert" class="mt-3 text-sm text-red-600">{failure}</p>{/if}
	{#if message}<p role="status" class="mt-3 flex items-center gap-1 text-sm text-emerald-700 dark:text-emerald-300"><CheckCircle2 class="h-4 w-4"/>{message}</p>{/if}
	{#if !loading && items.length===0}<p class="mt-3 text-sm text-neutral-500">Belum ada rencana kelas dalam proses approval.</p>{/if}
	<div class="mt-3 space-y-2">
		{#each items as plan (plan.id)}
			<div class="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-neutral-200 p-3 dark:border-white/10">
				<div>
					<p class="text-sm font-semibold">Rencana #{plan.id} · {plan.gender==="P"?"Akhwat":"Ikhwan"} · {plan.angkatan}</p>
					<p class="mt-1 text-xs text-neutral-500">{plan.frequency}x/pekan · {plan.student_count}/{plan.capacity} santri · {plan.reserved_count} direservasi · Guru #{plan.guru_id}</p>
					<p class="mt-1 text-xs text-neutral-500">Level {plan.level_code} · Jadwal: {plan.slots_label || "Belum disusun"} WIB</p>
					<p class="mt-1 text-xs text-neutral-500">{labels[plan.status]??plan.status}{plan.admin_approved_by?" · Disetujui Admin Kelas":""} · Versi {plan.version}</p>
					{#if plan.exception_reason}<p class="mt-1 text-xs text-amber-700 dark:text-amber-300">Alasan exception: {plan.exception_reason} · {plan.exception_status}</p>{/if}
				</div>
				<div>
					{#if mode!=="coordinator" && plan.status==="draft"}
						<button type="button" onclick={()=>submit(plan,"submit")} disabled={busy!==null} class="rounded-lg bg-brand-600 px-3 py-2 text-xs font-semibold text-white disabled:opacity-50">Ajukan</button>
					{:else if mode==="coordinator" && plan.status==="pending_coordinator"}
						<button type="button" onclick={()=>submit(plan,"coordinator_confirm")} disabled={busy!==null} class="rounded-lg bg-brand-600 px-3 py-2 text-xs font-semibold text-white disabled:opacity-50">Konfirmasi guru</button>
					{:else if mode!=="coordinator" && plan.status==="pending_admin" && !plan.admin_approved_by}
						<button type="button" onclick={()=>submit(plan,"admin_approve")} disabled={busy!==null} class="rounded-lg bg-brand-600 px-3 py-2 text-xs font-semibold text-white disabled:opacity-50">Approve</button>
					{:else if mode!=="coordinator" && plan.status==="pending_admin" && plan.admin_approved_by && plan.reserved_count===0}
						<button type="button" onclick={()=>submit(plan,"reserve")} disabled={busy!==null} class="rounded-lg bg-brand-600 px-3 py-2 text-xs font-semibold text-white disabled:opacity-50">Reservasi kandidat</button>
					{:else if mode!=="coordinator" && plan.status==="pending_admin" && plan.admin_approved_by && plan.reserved_count>0}
						<button type="button" onclick={()=>submit(plan,"cancel")} disabled={busy!==null} class="rounded-lg border border-red-300 px-3 py-2 text-xs font-semibold text-red-700 disabled:opacity-50">Batalkan dan lepas reservasi</button>
					{:else if plan.status==="pending_exception" && mode==="super_admin"}
						<div class="flex flex-wrap gap-2">
							<button type="button" onclick={()=>submit(plan,"exception_approve")} disabled={busy!==null} class="rounded-lg bg-brand-600 px-3 py-2 text-xs font-semibold text-white disabled:opacity-50">Setujui exception</button>
							<button type="button" onclick={()=>submit(plan,"exception_reject")} disabled={busy!==null} class="rounded-lg border border-red-300 px-3 py-2 text-xs font-semibold text-red-700 disabled:opacity-50">Tolak</button>
						</div>
					{:else if plan.status==="pending_exception"}
						<span class="text-xs font-medium text-amber-700 dark:text-amber-300">Menunggu keputusan Super Admin</span>
					{/if}
				</div>
			</div>
		{/each}
	</div>
</section>
