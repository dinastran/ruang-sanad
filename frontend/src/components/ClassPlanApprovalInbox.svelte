<script lang="ts">
	import { onMount } from "svelte";
	import { getCSRFToken } from "@lib/utils/csrf";
	import { ClipboardCheck, RefreshCcw, CheckCircle2 } from "lucide-svelte";

	type WorkflowPlan = {
		id:number; source:string; status:string; program:string; angkatan:string;
		gender:string; guru_id:number; frequency:number; min_quota:number; capacity:number;
		target_kelas_id:number; version:number; coordinator_confirmed_by:number;
		coordinator_confirmed_version:number; admin_approved_by:number; student_count:number; level_code:string; slots_label:string;
	};
	let { mode }: {mode:"admin"|"coordinator"}=$props();
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
		const labels:Record<string,string>={submit:"Ajukan ke Koordinator Guru",coordinator_confirm:"Konfirmasi penugasan guru",admin_approve:"Setujui untuk proses aktivasi berikutnya"};
		if(!confirm((labels[action]??action)+"? Persetujuan ini belum mengaktifkan kelas."))return;
		busy=plan.id;failure="";message="";
		try {
			const r=await fetch("/app/onboarding/planning/plans/"+plan.id+"/actions",{
				method:"POST",credentials:"same-origin",
				headers:{"Content-Type":"application/json",Accept:"application/json","X-XSRF-TOKEN":getCSRFToken()},
				body:JSON.stringify({action,version:plan.version,reason:labels[action]??action}),
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
					<p class="mt-1 text-xs text-neutral-500">{plan.frequency}x/pekan · {plan.student_count}/{plan.capacity} santri · Guru #{plan.guru_id}</p>
					<p class="mt-1 text-xs text-neutral-500">Level {plan.level_code} · Jadwal: {plan.slots_label || "Belum disusun"} WIB</p>
					<p class="mt-1 text-xs text-neutral-500">{labels[plan.status]??plan.status}{plan.admin_approved_by?" · Disetujui Admin Kelas":""} · Versi {plan.version}</p>
				</div>
				<div>
					{#if mode==="admin" && plan.status==="draft"}
						<button type="button" onclick={()=>submit(plan,"submit")} disabled={busy!==null} class="rounded-lg bg-brand-600 px-3 py-2 text-xs font-semibold text-white disabled:opacity-50">Ajukan</button>
					{:else if mode==="coordinator" && plan.status==="pending_coordinator"}
						<button type="button" onclick={()=>submit(plan,"coordinator_confirm")} disabled={busy!==null} class="rounded-lg bg-brand-600 px-3 py-2 text-xs font-semibold text-white disabled:opacity-50">Konfirmasi guru</button>
					{:else if mode==="admin" && plan.status==="pending_admin" && !plan.admin_approved_by}
						<button type="button" onclick={()=>submit(plan,"admin_approve")} disabled={busy!==null} class="rounded-lg bg-brand-600 px-3 py-2 text-xs font-semibold text-white disabled:opacity-50">Approve</button>
					{:else if plan.status==="pending_exception"}
						<span class="text-xs font-medium text-amber-700 dark:text-amber-300">Menunggu mekanisme exception</span>
					{/if}
				</div>
			</div>
		{/each}
	</div>
</section>
