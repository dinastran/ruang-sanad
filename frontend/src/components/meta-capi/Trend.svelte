<script lang="ts">
 import type { CapiDay } from "./data";
 import { prettyDay } from "./data";
 import { panel, fmt } from "@components/lead-dashboard/types";
 interface Props { days:CapiDay[] }
 let { days }:Props=$props();
 type Key="sent"|"test_sent"|"failed"|"pending";
 const metrics:{key:Key;name:string;stroke:string;circle:string}[]=[
  {key:"sent",name:"Produksi",stroke:"stroke-brand-600 dark:stroke-brand-400",circle:"fill-brand-600 dark:fill-brand-400"},
  {key:"test_sent",name:"Mode uji",stroke:"stroke-slate-500",circle:"fill-slate-500"},
  {key:"failed",name:"Gagal",stroke:"stroke-rose-500",circle:"fill-rose-500"},
  {key:"pending",name:"Menunggu",stroke:"stroke-amber-500",circle:"fill-amber-500"}
 ];
 let metric=$state<Key>("sent");
 let selected=$derived(metrics.find(x=>x.key===metric)!);
 let width=$state(620);
 const height=206,left=32,right=12,top=12,bottom=28;
 let max=$derived(Math.max(1,...days.map(d=>d[metric]??0)));
 const x=(i:number,n:number)=>left+(n<2?(width-left-right)/2:i*(width-left-right)/(n-1));
 const y=(n:number)=>top+(height-top-bottom)*(1-n/max);
 let plot=$derived(days.map((day,i)=>(i===0?"M":"L")+x(i,days.length).toFixed(1)+","+y(day[metric]??0).toFixed(1)).join(" "));
 let hover=$state<number|null>(null);
 let step=$derived(Math.max(1,Math.ceil(days.length/Math.max(2,Math.floor(width/72)))));
 const total=$derived(days.reduce((sum,d)=>sum+d[metric],0));
 const tab="rounded-md px-2.5 py-1.5 text-xs font-medium focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500";
 function move(e:PointerEvent){
  if(!days.length) return;
  const svg=e.currentTarget as SVGSVGElement;
  const rect=svg.getBoundingClientRect();
  const px=(e.clientX-rect.left)/rect.width*width;
  const i=days.length<2?0:Math.round((px-left)/(width-left-right)*(days.length-1));
  hover=Math.max(0,Math.min(days.length-1,i));
 }
</script>
<section class={panel+" min-w-0 p-5 sm:p-6"} aria-labelledby="capi-trend">
 <div class="flex flex-wrap items-start justify-between gap-3">
  <div>
   <h2 id="capi-trend" class="text-base font-semibold text-neutral-950 dark:text-white">Aktivitas pengiriman</h2>
   <p class="mt-1 text-xs text-neutral-500 dark:text-neutral-400">Dihitung dari waktu event, WIB. Pilih status untuk melihat tren.</p>
  </div>
  <div role="group" aria-label="Metrik grafik" class="flex flex-wrap gap-1 rounded-lg bg-neutral-100 p-1 dark:bg-neutral-900">
   {#each metrics as m}
    <button type="button" aria-pressed={metric===m.key} onclick={()=>{metric=m.key;hover=null;}} class={tab+(metric===m.key?" bg-white text-neutral-900 shadow-sm dark:bg-neutral-800 dark:text-white":" text-neutral-600 hover:text-neutral-950 dark:text-neutral-400")}>{m.name}</button>
   {/each}
  </div>
 </div>
 <div class="mt-5 flex items-end justify-between">
  <p class="text-sm tabular-nums text-neutral-500 dark:text-neutral-400"><strong class="text-2xl font-semibold text-neutral-950 dark:text-white">{fmt(total)}</strong> event {selected.name.toLowerCase()}</p>
  <p class="text-xs text-neutral-500">{hover!==null&&days[hover]?prettyDay(days[hover].day)+": "+fmt(days[hover][metric]):"Periode terpilih"}</p>
 </div>
 <div class="mt-3 min-w-0" bind:clientWidth={width}>
  <svg width={width} height={height} viewBox={`0 0 ${width} ${height}`} class="block w-full overflow-visible" role="img" aria-label={`Tren event ${selected.name.toLowerCase()} per hari`} onpointermove={move} onpointerleave={()=>hover=null}>
   {#each [0,0.5,1] as part}
    <line x1={left} x2={width-right} y1={y(max*part)} y2={y(max*part)} class="stroke-neutral-200 dark:stroke-neutral-800" stroke-dasharray="3 5"/>
    <text x={left-6} y={y(max*part)+4} text-anchor="end" class="fill-neutral-500 text-[10px]">{fmt(Math.round(max*part))}</text>
   {/each}
   {#if days.length>1}
    <path d={plot} fill="none" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" class={selected.stroke}/>
   {/if}
   {#each days as day,i}
    {#if i%step===0}
     <text x={x(i,days.length)} y={height-7} text-anchor="middle" class="fill-neutral-500 text-[10px]">{prettyDay(day.day)}</text>
    {/if}
   {/each}
   {#if hover!==null&&days[hover]}
    <line x1={x(hover,days.length)} x2={x(hover,days.length)} y1={top} y2={height-bottom} class="stroke-neutral-300 dark:stroke-neutral-600" stroke-dasharray="3 4"/>
    <circle cx={x(hover,days.length)} cy={y(days[hover][metric])} r="4" class={selected.circle} stroke="white" stroke-width="2"/>
   {/if}
  </svg>
 </div>
 <details class="mt-3 text-xs text-neutral-500 dark:text-neutral-400">
  <summary class="w-fit cursor-pointer rounded focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500">Lihat data harian</summary>
  <div class="mt-2 max-h-56 overflow-auto rounded-lg border border-neutral-200 dark:border-white/10">
   <table class="w-full text-left tabular-nums">
    <thead><tr class="border-b border-neutral-200 dark:border-white/10"><th class="p-2 font-medium">Tanggal</th><th class="p-2 text-right font-medium">{selected.name}</th></tr></thead>
    <tbody>{#each days as d}<tr class="border-b border-neutral-100 last:border-0 dark:border-white/[0.05]"><td class="p-2">{prettyDay(d.day)}</td><td class="p-2 text-right">{fmt(d[metric])}</td></tr>{/each}</tbody>
   </table>
  </div>
 </details>
</section>