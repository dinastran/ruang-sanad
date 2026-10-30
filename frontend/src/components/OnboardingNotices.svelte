<script lang="ts">
 import { inertia } from "@inertiajs/svelte";
 import { Bell, Check } from "lucide-svelte";

 interface Notice {id:number;type:string;title:string;message:string;action_url:string;read:boolean;created_at:string}
 interface Props {items:Notice[]}
 let {items}:Props=$props();
 let marked=$state<number[]>([]);
 let busy=$state<number|null>(null);
 let errorMessage=$state("");
 const isRead=(n:Notice)=>n.read||marked.includes(n.id);
 const date=(s:string)=>new Date(s).toLocaleString("id-ID",{timeZone:"Asia/Jakarta",day:"numeric",month:"short",hour:"2-digit",minute:"2-digit"});
 function csrf():string {
  const c=document.cookie.split("; ").find(c=>c.startsWith("XSRF-TOKEN="));
  return c?decodeURIComponent(c.split("=").slice(1).join("=")):"";
 }
 async function markRead(id:number) {
  if(busy!==null)return;
  busy=id;errorMessage="";
  try {
   const res=await fetch(`/app/onboarding/notifications/${id}/read`,{
    method:"PUT",credentials:"same-origin",
    headers:{"Accept":"application/json","Content-Type":"application/json","X-XSRF-TOKEN":csrf()},
    body:"{}"
   });
   const body:{error?:string;ok?:boolean}=await res.json().catch(()=>({}));
   if(!res.ok||!body.ok)throw new Error(body.error??"Gagal menandai notifikasi");
   marked=[...marked,id];
  }catch(e){errorMessage=e instanceof Error?e.message:"Gagal menandai notifikasi"}
  finally{busy=null}
 }
</script>

<section class="rounded-xl border border-neutral-200 bg-white p-4 dark:border-white/10 dark:bg-neutral-900" aria-label="Notifikasi onboarding saya">
 <div class="flex items-center justify-between gap-2">
  <h2 class="flex items-center gap-2 text-base font-semibold"><Bell size={16}/> Notifikasi Onboarding Saya</h2>
  <span class="text-xs text-neutral-500">{items.filter(x=>!isRead(x)).length} belum dibaca</span>
 </div>
 {#if errorMessage}<p role="alert" class="mt-2 text-xs text-red-600">{errorMessage}</p>{/if}
 {#if items.length===0}
  <p class="mt-3 text-sm text-neutral-500">Belum ada notifikasi untuk akun ini. Pemeriksaan eskalasi berjalan otomatis pada server.</p>
 {:else}
  <ul class="mt-3 max-h-80 space-y-2 overflow-y-auto">
   {#each items as notice (notice.id)}
    <li class={"rounded-lg border p-3 text-sm "+(isRead(notice)?"border-neutral-100 bg-neutral-50/50 dark:border-white/5 dark:bg-white/5":"border-amber-200 bg-amber-50/50 dark:border-amber-500/20 dark:bg-amber-500/5")}>
     <div class="flex items-start justify-between gap-2">
      <div class="min-w-0 flex-1">
       <a href={notice.action_url||"/app/onboarding/classes"} use:inertia class="font-semibold hover:underline">{notice.title}</a>
       <p class="mt-1 text-xs text-neutral-600 dark:text-neutral-300">{notice.message}</p>
       <p class="mt-1 text-xs text-neutral-500">{date(notice.created_at)} WIB</p>
      </div>
      {#if !isRead(notice)}
       <button type="button" disabled={busy!==null} onclick={()=>markRead(notice.id)} class="shrink-0 rounded-md border border-neutral-300 p-1.5 hover:bg-white disabled:opacity-50 dark:border-white/20" title="Tandai dibaca" aria-label={"Tandai dibaca: "+notice.title}><Check size={14}/></button>
      {/if}
     </div>
    </li>
   {/each}
  </ul>
 {/if}
</section>
