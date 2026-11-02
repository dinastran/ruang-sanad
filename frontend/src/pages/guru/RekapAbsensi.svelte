<script lang="ts">
	import { inertia, router } from "@inertiajs/svelte";
	import { fly } from "svelte/transition";
	import AppLayout from "@layouts/AppLayout.svelte";
	import { Toast } from "@lib/notifications/toast";
	import { getCSRFToken } from "@lib/utils/csrf";
	import type { User, GuruKelas, SantriGuru, PertemuanGuru } from "@lib/types";
	import { nomorPertemuan } from "@lib/pertemuan";
	import { ArrowLeft, BookOpen, Users, Calendar, Download, FileText, ChevronLeft, ChevronRight, Search, Clock3, BarChart3, CalendarRange, AlertTriangle, CheckCircle2 } from "lucide-svelte";

	interface Props {
		user?: User;
		kelas?: GuruKelas;
		santri: SantriGuru[];
		pertemuan_list: PertemuanGuru[];
		absensi: Record<string, { id: number; status: string; catatan: string; batas_materi: string }>;
	}

	let { user, kelas, santri = [], pertemuan_list = [], absensi = {} }: Props = $props();
	let canEdit = $derived(user?.role === "guru" || user?.role === "admin_kelas" || user?.role === "super_admin");
	// The modal edits catatan and, for materi individual classes, batas materi. It
	// also carries a pending status when switching to hadir/telat needs a batas.
	let catatanModal = $state<{ key: string; santriNama: string; pertemuanKe: number; catatan: string; status: string; batasMateri: string } | null>(null);

	function butuhBatasMateri(status: string): boolean {
		return !!kelas?.materi_individual && (status === "hadir" || status === "telat");
	}
	let isSavingCatatan = $state(false);
	let updatingAbsensi = $state<Record<number, boolean>>({});

	function statusColor(status: string): string {
		const colors: Record<string, string> = {
			hadir: "bg-green-500 text-white",
			izin: "bg-yellow-500 text-white",
			sakit: "bg-blue-500 text-white",
			alpa: "bg-red-500 text-white",
			telat: "bg-orange-500 text-white",
		};
		return colors[status] || "bg-neutral-200 dark:bg-neutral-700 text-neutral-500";
	}

	function statusLabel(status: string): string {
		return status.charAt(0).toUpperCase();
	}

	function attendanceColor(pct: number): string {
		if (pct >= 85) return "text-green-600 dark:text-green-400";
		if (pct >= 70) return "text-amber-600 dark:text-amber-400";
		return "text-red-600 dark:text-red-400";
	}

	let totalPertemuan = $derived(pertemuan_list.length);

	async function updateAbsensi(id: number, status: string, catatan: string, batasMateri: string) {
		const response = await fetch(`/app/guru/kelas/${kelas?.id}/absensi/${id}`, {
			method: "PUT",
			headers: { "Content-Type": "application/json", "X-XSRF-TOKEN": getCSRFToken() },
			body: JSON.stringify({ status, catatan, batas_materi: batasMateri }),
		});
		if (!response.ok) {
			const data = await response.json().catch(() => ({}));
			throw new Error(data.error || "Gagal menyimpan absensi");
		}
	}

	function setUpdatingAbsensi(id: number, value: boolean) {
		updatingAbsensi = { ...updatingAbsensi, [id]: value };
	}

	function reloadAbsensi() {
		return new Promise<void>((resolve) => {
			router.reload({ only: ["absensi"], preserveScroll: true, onFinish: () => resolve() });
		});
	}

	async function changeStatus(key: string, status: string, select: HTMLSelectElement, santriNama: string, pertemuanKe: number) {
		const entry = absensi[key];
		if (!entry || updatingAbsensi[entry.id]) return;
		if (butuhBatasMateri(status) && !entry.batas_materi.trim()) {
			// The server requires batas materi here; ask for it before saving.
			select.value = entry.status;
			catatanModal = { key, santriNama, pertemuanKe, catatan: entry.catatan, status, batasMateri: "" };
			return;
		}
		absensi = { ...absensi, [key]: { ...entry, status } };
		setUpdatingAbsensi(entry.id, true);
		try {
			await updateAbsensi(entry.id, status, entry.catatan, entry.batas_materi);
		} catch (err) {
			Toast(err instanceof Error ? err.message : "Gagal menyimpan absensi", "error");
			await reloadAbsensi();
		} finally {
			setUpdatingAbsensi(entry.id, false);
		}
	}

	function openCatatan(key: string, santriNama: string, pertemuanKe: number) {
		const entry = absensi[key];
		if (!entry) return;
		catatanModal = { key, santriNama, pertemuanKe, catatan: entry.catatan, status: entry.status, batasMateri: entry.batas_materi };
	}

	async function saveCatatan() {
		if (!catatanModal) return;
		const entry = absensi[catatanModal.key];
		if (!entry || updatingAbsensi[entry.id]) return;
		const { key, catatan, status } = catatanModal;
		// Only replace batas materi when the form asks for it; otherwise keep what
		// was recorded (the server clears it for non hadir/telat statuses).
		const batasMateri = butuhBatasMateri(status) ? catatanModal.batasMateri.trim() : entry.batas_materi;
		if (butuhBatasMateri(status) && !batasMateri) {
			Toast("Batas materi wajib diisi untuk santri yang hadir/telat", "error");
			return;
		}
		isSavingCatatan = true;
		setUpdatingAbsensi(entry.id, true);
		try {
		absensi = { ...absensi, [key]: { ...entry, status, catatan, batas_materi: batasMateri } };
			await updateAbsensi(entry.id, status, catatan, batasMateri);
		catatanModal = null;
		Toast("Absensi tersimpan", "success");
	} catch (err) {
		Toast(err instanceof Error ? err.message : "Gagal menyimpan catatan", "error");
		await reloadAbsensi();
	} finally {
		isSavingCatatan = false;
		setUpdatingAbsensi(entry.id, false);
		}
	}

 // Every metric on this page derives from the selected calendar period,
 // not from SantriGuru.persen_hadir (which is lifetime).
 type Mode = "month" | "day" | "range";
 type StatusKey = "hadir" | "telat" | "izin" | "sakit" | "alpa";
 const statusKeys: StatusKey[] = ["hadir","telat","izin","sakit","alpa"];
 const statusNames: Record<StatusKey,string> = {hadir:"Hadir",telat:"Terlambat",izin:"Izin",sakit:"Sakit",alpa:"Alpa"};
 const dateToday = () => new Date().toLocaleDateString("en-CA",{timeZone:"Asia/Jakarta"});
 let monthOptions = $derived([...new Set(pertemuan_list.map(p=>p.tanggal.slice(0,7)).filter(Boolean))].sort().reverse());
 const latestDay = pertemuan_list.reduce((v,p)=>p.tanggal>v?p.tanggal:v,"");
 let mode = $state<Mode>("month");
 let month = $state(latestDay.slice(0,7) || dateToday().slice(0,7));
 let day = $state(latestDay || dateToday());
 let fromDate = $state((latestDay.slice(0,7)||dateToday().slice(0,7))+"-01");
 let toDate = $state(latestDay || dateToday());
 let search = $state("");
 let showOnlyRisk = $state(false);

 let invalidRange = $derived(mode==="range" && (!fromDate||!toDate||fromDate>toDate || (Date.parse(toDate)-Date.parse(fromDate))/86400000>365));
 let filteredMeetings = $derived.by(()=>{
  if(invalidRange) return [];
  const meetings=pertemuan_list.filter(p=>mode==="month"?p.tanggal.slice(0,7)===month:mode==="day"?p.tanggal===day:p.tanggal>=fromDate&&p.tanggal<=toDate);
  return [...meetings].sort((a,b)=>a.tanggal.localeCompare(b.tanggal)||a.pertemuan_ke-b.pertemuan_ke);
 });
 function formatMonth(value:string){if(!value)return "Belum ada periode";return new Date(value+"-01T12:00:00+07:00").toLocaleDateString("id-ID",{month:"long",year:"numeric",timeZone:"Asia/Jakarta"});}
 function shortDay(value:string){return new Date(value+"T12:00:00+07:00").toLocaleDateString("id-ID",{day:"numeric",month:"short",timeZone:"Asia/Jakarta"});}
 let periodTitle=$derived(mode==="month"?formatMonth(month):mode==="day"?shortDay(day):fromDate&&toDate?shortDay(fromDate)+" sampai "+shortDay(toDate):"Pilih periode");
 const rateOf=(a:number,n:number)=>n>0?Math.round(a/n*100):null;
 let rows=$derived(santri.map(s=>{
  const counts:Record<StatusKey,number>={hadir:0,telat:0,izin:0,sakit:0,alpa:0};
  let recorded=0;
  for(const p of filteredMeetings){
   const entry=absensi[`${s.id}:${p.id}`];
   if(entry && statusKeys.includes(entry.status as StatusKey)){
    counts[entry.status as StatusKey]++;recorded++;
   }
  }
  return {...s,counts,recorded,missing:Math.max(0,filteredMeetings.length-recorded),rate:rateOf(counts.hadir,recorded)};
 }));
 let totalRecorded=$derived(rows.reduce((v,s)=>v+s.recorded,0));
 let totalPossible=$derived(rows.length*filteredMeetings.length);
 let totals=$derived(statusKeys.reduce((out,key)=>({...out,[key]:rows.reduce((v,s)=>v+s.counts[key],0)}),{} as Record<StatusKey,number>));
 let attendanceRate=$derived(rateOf(totals.hadir,totalRecorded));
 let completion=$derived(rateOf(totalRecorded,totalPossible));
 let riskCount=$derived(rows.filter(s=>s.recorded>=2 && s.rate!==null && s.rate<75).length);
 let visibleRows=$derived(rows.filter(s=>(!showOnlyRisk||(s.recorded>=2 && s.rate!==null&&s.rate<75)) && (s.nama+" "+s.id_mahasantri).toLowerCase().includes(search.trim().toLowerCase())));
 let meetingStats=$derived(filteredMeetings.map(p=>{
  const counts:Record<StatusKey,number>={hadir:0,telat:0,izin:0,sakit:0,alpa:0};
  let recorded=0;
  for(const s of santri){const a=absensi[`${s.id}:${p.id}`];if(a&&statusKeys.includes(a.status as StatusKey)){counts[a.status as StatusKey]++;recorded++;}}
  return {p,counts,recorded,missing:Math.max(0,santri.length-recorded),pct:rateOf(counts.hadir,recorded)};
 }));
 let exportUrl=$derived.by(()=>{
  const params=new URLSearchParams({mode});
  if(mode==="month")params.set("month",month);
  if(mode==="day")params.set("date",day);
  if(mode==="range"){params.set("from",fromDate);params.set("to",toDate);}
  return `/app/guru/kelas/${kelas?.id}/rekap/export?`+params.toString();
 });
 const surface="rounded-xl border border-neutral-200/80 bg-white dark:border-white/[0.07] dark:bg-neutral-925/60";
 const focus="focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500";
 const input="h-10 w-full rounded-lg border border-neutral-200 bg-white px-3 text-sm text-neutral-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 dark:border-white/10 dark:bg-neutral-900 dark:text-white";
 const rateTone=(r:number|null)=>r===null?"text-neutral-400":r>=85?"text-emerald-700 dark:text-emerald-400":r>=75?"text-amber-700 dark:text-amber-400":"text-rose-700 dark:text-rose-400";
 function switchMonth(delta:number){
  const d=new Date(month+"-01T12:00:00+07:00");
  d.setUTCMonth(d.getUTCMonth()+delta);
  month=d.toLocaleDateString("en-CA",{timeZone:"Asia/Jakarta"}).slice(0,7);
 }

</script>

<AppLayout {user} group="guru-kelas">
 <main class="mx-auto max-w-7xl space-y-5 px-4 py-6 sm:px-6 lg:py-8">
  <header class="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
   <div class="min-w-0">
    <a href={"/app/guru/kelas/"+kelas?.id} use:inertia class="inline-flex min-h-9 items-center gap-1.5 text-sm font-medium text-neutral-500 hover:text-brand-700 dark:text-neutral-400 dark:hover:text-brand-400"><ArrowLeft size={15}/> Kembali ke kelas</a>
    <p class="mt-3 text-[11px] font-semibold uppercase tracking-[0.12em] text-brand-700 dark:text-brand-400">Monitoring pembelajaran / Rekap absensi</p>
    <h1 class="mt-1 text-[28px] font-semibold tracking-[-0.025em] text-neutral-950 dark:text-white">{kelas?.nama_kelas||"Rekap kelas"}</h1>
    <p class="mt-1 text-sm text-neutral-500 dark:text-neutral-400">{kelas?.level||"Kelas"} · {santri.length} santri aktif · {pertemuan_list.length} pertemuan tercatat sepanjang riwayat</p>
   </div>
   <div class="flex flex-wrap items-center gap-2">
    <a href={"/app/guru/kelas/"+kelas?.id+"/riwayat-pertemuan"} use:inertia class={"inline-flex min-h-10 items-center gap-2 rounded-lg border border-neutral-200 px-3 text-sm font-medium text-neutral-700 hover:bg-neutral-50 dark:border-white/10 dark:text-neutral-300 dark:hover:bg-white/[0.05] "+focus}><Calendar size={15}/> Riwayat pertemuan</a>
    {#if !invalidRange}
     <a href={exportUrl} download class={"inline-flex min-h-10 items-center gap-2 rounded-lg bg-brand-700 px-4 text-sm font-semibold text-white hover:bg-brand-800 dark:bg-brand-600 dark:hover:bg-brand-500 "+focus}><Download size={16}/> Export CSV</a>
    {:else}
     <button type="button" disabled class="inline-flex min-h-10 items-center gap-2 rounded-lg bg-neutral-200 px-4 text-sm font-semibold text-neutral-500 dark:bg-neutral-800"><Download size={16}/> Export CSV</button>
    {/if}
   </div>
  </header>

  <section class={surface+" p-4 sm:p-5"} aria-labelledby="period-title">
   <div class="flex flex-wrap items-start justify-between gap-3">
    <div>
     <h2 id="period-title" class="text-base font-semibold text-neutral-900 dark:text-white">Periode rekap</h2>
     <p class="mt-1 text-xs text-neutral-500 dark:text-neutral-400">Semua angka dan hasil export mengikuti periode yang dipilih.</p>
    </div>
    <div role="group" aria-label="Mode filter absensi" class="flex flex-wrap gap-1 rounded-lg bg-neutral-100 p-1 dark:bg-neutral-900">
     {#each [{key:"month",text:"Per bulan"},{key:"day",text:"Tanggal"},{key:"range",text:"Rentang khusus"}] as tab}
      <button type="button" aria-pressed={mode===tab.key} onclick={()=>mode=tab.key as Mode}
       class={"min-h-9 rounded-md px-3 py-1.5 text-xs font-semibold transition-colors "+focus+" "+(mode===tab.key?"bg-white text-neutral-950 shadow-sm dark:bg-neutral-800 dark:text-white":"text-neutral-600 hover:text-neutral-950 dark:text-neutral-400 dark:hover:text-white")}>{tab.text}</button>
     {/each}
    </div>
   </div>
   <div class="mt-4 flex flex-wrap items-end gap-3">
    {#if mode==="month"}
     <div class="flex min-w-[240px] items-end gap-2">
      <button type="button" aria-label="Bulan sebelumnya" onclick={()=>switchMonth(-1)} class={"flex h-10 w-10 shrink-0 items-center justify-center rounded-lg border border-neutral-200 text-neutral-600 hover:bg-neutral-50 dark:border-white/10 dark:text-neutral-300 "+focus}><ChevronLeft size={17}/></button>
      <label class="min-w-0 flex-1 text-xs font-medium text-neutral-600 dark:text-neutral-300">Bulan
       <input type="month" bind:value={month} class={input+" mt-1"} aria-label="Pilih bulan rekap"/>
      </label>
      <button type="button" aria-label="Bulan berikutnya" onclick={()=>switchMonth(1)} class={"flex h-10 w-10 shrink-0 items-center justify-center rounded-lg border border-neutral-200 text-neutral-600 hover:bg-neutral-50 dark:border-white/10 dark:text-neutral-300 "+focus}><ChevronRight size={17}/></button>
     </div>
     {#if monthOptions.length>0}
      <label class="min-w-[175px] text-xs font-medium text-neutral-600 dark:text-neutral-300">Bulan dengan data
       <select bind:value={month} class={input+" mt-1"}><option value={month}>{formatMonth(month)}</option>{#each monthOptions.filter(v=>v!==month) as m}<option value={m}>{formatMonth(m)}</option>{/each}</select>
      </label>
     {/if}
    {:else if mode==="day"}
     <label class="w-full max-w-[240px] text-xs font-medium text-neutral-600 dark:text-neutral-300">Tanggal pembelajaran
      <input type="date" bind:value={day} class={input+" mt-1"}/>
     </label>
    {:else}
     <label class="min-w-[160px] text-xs font-medium text-neutral-600 dark:text-neutral-300">Dari
      <input type="date" bind:value={fromDate} class={input+" mt-1"}/>
     </label>
     <label class="min-w-[160px] text-xs font-medium text-neutral-600 dark:text-neutral-300">Sampai
      <input type="date" bind:value={toDate} class={input+" mt-1"}/>
     </label>
    {/if}
    <p class="pb-2 text-sm font-medium text-neutral-700 dark:text-neutral-200">{periodTitle}</p>
   </div>
   {#if invalidRange}
    <p role="alert" class="mt-3 flex items-center gap-2 text-sm text-rose-700 dark:text-rose-400"><AlertTriangle size={15}/> Tanggal awal harus sebelum tanggal akhir, dengan rentang maksimal 366 hari.</p>
   {/if}
  </section>

  <section aria-label="Indikator periode" class="grid grid-cols-2 gap-3 lg:grid-cols-4">
   <div class={surface+" col-span-2 p-5 lg:col-span-1"}>
    <p class="flex items-center gap-2 text-xs font-medium text-neutral-500 dark:text-neutral-400"><BarChart3 size={15}/> Kehadiran tercatat</p>
    <p class={"mt-2 text-[36px] font-semibold leading-tight tabular-nums tracking-tight "+rateTone(attendanceRate)}>{attendanceRate===null?"—":attendanceRate+"%"}</p>
    <p class="mt-1 text-xs text-neutral-500 dark:text-neutral-400">Hadir dari {totalRecorded} absensi berstatus · tidak termasuk telat</p>
   </div>
   <div class={surface+" p-5"}>
    <p class="flex items-center gap-2 text-xs text-neutral-500 dark:text-neutral-400"><CalendarRange size={15}/> Pertemuan</p>
    <p class="mt-2 text-[30px] font-semibold leading-tight tabular-nums text-neutral-950 dark:text-white">{filteredMeetings.length}</p>
    <p class="mt-1 text-xs text-neutral-500 dark:text-neutral-400">Pada {periodTitle}</p>
   </div>
   <div class={surface+" p-5"}>
    <p class="flex items-center gap-2 text-xs text-neutral-500 dark:text-neutral-400"><CheckCircle2 size={15}/> Kelengkapan</p>
    <p class="mt-2 text-[30px] font-semibold leading-tight tabular-nums text-neutral-950 dark:text-white">{completion===null?"—":completion+"%"}</p>
    <p class="mt-1 text-xs text-neutral-500 dark:text-neutral-400">{totalRecorded} dari {totalPossible} slot absensi</p>
   </div>
   <div class={surface+" p-5"}>
    <p class="flex items-center gap-2 text-xs text-neutral-500 dark:text-neutral-400"><AlertTriangle size={15}/> Perlu perhatian</p>
    <p class="mt-2 text-[30px] font-semibold leading-tight tabular-nums text-neutral-950 dark:text-white">{riskCount}</p>
    <p class="mt-1 text-xs text-neutral-500 dark:text-neutral-400">Santri hadir kurang dari 75%, minimal 2 catatan</p>
   </div>
  </section>

  <div class="grid gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(0,1.1fr)]">
   <section class={surface+" p-5 sm:p-6"} aria-labelledby="composition-title">
    <h2 id="composition-title" class="text-base font-semibold text-neutral-900 dark:text-white">Komposisi status absensi</h2>
    <p class="mt-1 text-xs text-neutral-500 dark:text-neutral-400">Dari catatan yang sudah terisi dalam periode. Tidak termasuk slot kosong.</p>
    <div class="mt-5 flex h-3 w-full overflow-hidden rounded-full bg-neutral-100 dark:bg-white/[0.08]" role="img" aria-label={"Status: "+statusKeys.map(k=>statusNames[k]+" "+totals[k]).join(", ")}>
     <span class="h-full bg-emerald-500" style:width={(totalRecorded?totals.hadir/totalRecorded*100:0)+"%"}></span>
     <span class="h-full bg-amber-500" style:width={(totalRecorded?totals.telat/totalRecorded*100:0)+"%"}></span>
     <span class="h-full bg-sky-500" style:width={(totalRecorded?totals.izin/totalRecorded*100:0)+"%"}></span>
     <span class="h-full bg-indigo-400" style:width={(totalRecorded?totals.sakit/totalRecorded*100:0)+"%"}></span>
     <span class="h-full bg-rose-500" style:width={(totalRecorded?totals.alpa/totalRecorded*100:0)+"%"}></span>
    </div>
    <dl class="mt-5 grid grid-cols-2 gap-3 sm:grid-cols-3">
     {#each [{key:"hadir",color:"bg-emerald-500"},{key:"telat",color:"bg-amber-500"},{key:"izin",color:"bg-sky-500"},{key:"sakit",color:"bg-indigo-400"},{key:"alpa",color:"bg-rose-500"}] as item}
      <div class="flex items-start gap-2">
       <span class={"mt-1.5 h-2.5 w-2.5 rounded-sm "+item.color}></span>
       <div><dt class="text-xs text-neutral-500 dark:text-neutral-400">{statusNames[item.key as StatusKey]}</dt><dd class="text-lg font-semibold tabular-nums text-neutral-900 dark:text-white">{totals[item.key as StatusKey]}</dd></div>
      </div>
     {/each}
     <div class="flex items-start gap-2"><span class="mt-1.5 h-2.5 w-2.5 rounded-sm bg-neutral-300 dark:bg-neutral-600"></span><div><dt class="text-xs text-neutral-500">Belum tercatat</dt><dd class="text-lg font-semibold tabular-nums text-neutral-900 dark:text-white">{totalPossible-totalRecorded}</dd></div></div>
    </dl>
   </section>

   <section class={surface+" p-5 sm:p-6"} aria-labelledby="pertemuan-title">
    <div class="flex flex-wrap items-start justify-between gap-2">
     <div><h2 id="pertemuan-title" class="text-base font-semibold text-neutral-900 dark:text-white">Jejak pertemuan</h2>
      <p class="mt-1 text-xs text-neutral-500 dark:text-neutral-400">Kelengkapan dan jumlah hadir per tanggal.</p>
     </div>
     <span class="text-xs tabular-nums text-neutral-500">{filteredMeetings.length} pertemuan</span>
    </div>
    {#if meetingStats.length===0}
     <p class="py-10 text-center text-sm text-neutral-500 dark:text-neutral-400">Belum ada pertemuan pada periode ini.</p>
    {:else}
     <div class="mt-4 max-h-[280px] space-y-3 overflow-y-auto pr-1">
      {#each meetingStats as m (m.p.id)}
       <div class="grid grid-cols-[minmax(7rem,1fr)_minmax(0,1.2fr)_auto] items-center gap-3">
        <div class="min-w-0"><p class="truncate text-xs font-semibold text-neutral-800 dark:text-neutral-200">Pertemuan #{nomorPertemuan(m.p)}</p><p class="text-[11px] text-neutral-500">{shortDay(m.p.tanggal)}</p></div>
        <div class="h-2 overflow-hidden rounded-full bg-neutral-100 dark:bg-white/[0.07]" role="meter" aria-label={"Kelengkapan pertemuan "+nomorPertemuan(m.p)} aria-valuemin={0} aria-valuemax={santri.length} aria-valuenow={m.recorded}>
         <span class="block h-full rounded-full bg-brand-600 dark:bg-brand-400" style:width={(santri.length?m.recorded/santri.length*100:0)+"%"}></span>
        </div>
        <span class="text-right text-xs tabular-nums text-neutral-600 dark:text-neutral-300">{m.recorded}/{santri.length}</span>
       </div>
      {/each}
     </div>
    {/if}
   </section>
  </div>

  <section class={surface+" overflow-hidden"} aria-labelledby="ledger-title">
   <div class="flex flex-wrap items-start justify-between gap-4 px-5 py-5 sm:px-6">
    <div>
     <p class="text-[11px] font-semibold uppercase tracking-[0.11em] text-brand-700 dark:text-brand-400">Buku absensi bulanan</p>
     <h2 id="ledger-title" class="mt-1 text-lg font-semibold text-neutral-950 dark:text-white">Rekap per santri</h2>
     <p class="mt-1 text-xs text-neutral-500 dark:text-neutral-400">Persentase dihitung dari status yang sudah dicatat pada periode terpilih. Klik status untuk mengubah, atau ikon catatan untuk melihat materi.</p>
    </div>
    <div class="flex flex-wrap items-center gap-2">
     <label class="relative block min-w-[180px]">
      <span class="sr-only">Cari nama santri</span>
      <Search size={15} class="pointer-events-none absolute left-3 top-3 text-neutral-400"/>
      <input type="search" bind:value={search} placeholder="Cari santri atau ID" class={input+" pl-9"} />
     </label>
     <label class="inline-flex min-h-10 cursor-pointer items-center gap-2 rounded-lg border border-neutral-200 px-3 text-xs font-medium text-neutral-700 dark:border-white/10 dark:text-neutral-300"><input type="checkbox" bind:checked={showOnlyRisk} class="accent-brand-600"/> Perlu perhatian</label>
    </div>
   </div>
   {#if !filteredMeetings.length||!santri.length}
    <div class="flex flex-col items-center gap-3 border-t border-neutral-100 px-6 py-12 text-center dark:border-white/[0.06]">
     <Calendar size={24} class="text-neutral-400"/>
     <h3 class="text-sm font-semibold text-neutral-900 dark:text-white">{!santri.length?"Belum ada santri aktif":"Tidak ada pertemuan pada periode ini"}</h3>
     <p class="max-w-md text-xs leading-relaxed text-neutral-500 dark:text-neutral-400">Pilih bulan lain atau gunakan rentang khusus. Data yang ditampilkan mengikuti santri yang masih aktif di kelas saat ini.</p>
    </div>
   {:else if !visibleRows.length}
    <p class="border-t border-neutral-100 px-6 py-10 text-center text-sm text-neutral-500 dark:border-white/[0.06]">Tidak ada santri yang cocok dengan filter.</p>
   {:else}
    <p class="border-t border-neutral-100 px-5 py-2 text-xs text-neutral-500 sm:px-6 dark:border-white/[0.06]">Geser tabel secara horizontal untuk melihat setiap tanggal. {visibleRows.length} dari {santri.length} santri ditampilkan.</p>
    <div class="max-w-full overflow-x-auto">
     <table class="w-full min-w-max border-collapse text-sm">
      <thead class="bg-neutral-50 text-xs text-neutral-500 dark:bg-neutral-900 dark:text-neutral-400">
       <tr class="border-y border-neutral-100 dark:border-white/[0.06]">
        <th scope="col" class="sticky left-0 z-20 min-w-[185px] bg-neutral-50 px-4 py-3 text-left font-semibold dark:bg-neutral-900 sm:px-6">Santri</th>
        <th scope="col" class="min-w-[70px] px-3 py-3 text-right font-semibold">Hadir %</th>
        <th scope="col" class="min-w-[85px] px-3 py-3 text-right font-semibold" title="Hadir/Telat/Izin/Sakit/Alpa">H/T/I/S/A</th>
        {#each filteredMeetings as p (p.id)}
         <th scope="col" class="min-w-[80px] border-l border-neutral-100 px-2 py-2.5 text-center font-semibold dark:border-white/[0.06]" title={"Pertemuan "+nomorPertemuan(p)+" ("+p.tanggal+")"+(p.level_nama?" · "+p.level_nama:"")}>
          <span class="block text-xs text-neutral-700 dark:text-neutral-300">#{nomorPertemuan(p)}</span><span class="mt-0.5 block text-[11px] font-normal text-neutral-500">{shortDay(p.tanggal)}</span>
         </th>
        {/each}
       </tr>
      </thead>
      <tbody class="divide-y divide-neutral-100 dark:divide-white/[0.06]">
       {#each visibleRows as s (s.id)}
        <tr class="group hover:bg-neutral-50/60 dark:hover:bg-white/[0.02]">
         <th scope="row" class="sticky left-0 z-10 max-w-[220px] bg-white px-4 py-3 text-left group-hover:bg-neutral-50 dark:bg-neutral-925 dark:group-hover:bg-neutral-900 sm:px-6">
          <span class="block truncate font-semibold text-neutral-900 dark:text-white">{s.nama}</span><span class="block truncate text-[11px] font-normal text-neutral-500">{s.id_mahasantri}</span>
         </th>
         <td class={"px-3 py-3 text-right font-semibold tabular-nums "+rateTone(s.rate)}>{s.rate===null?"—":s.rate+"%"}</td>
         <td class="px-3 py-3 text-right font-mono text-[11px] tabular-nums text-neutral-600 dark:text-neutral-300">{statusKeys.map(k=>s.counts[k]).join("/")}</td>
         {#each filteredMeetings as p (p.id)}
          {@const key=`${s.id}:${p.id}`}
          {@const entry=absensi[key]}
          <td class="border-l border-neutral-100 px-2 py-3 text-center dark:border-white/[0.05]">
           {#if entry}
            <div class="flex items-center justify-center gap-1">
             {#if canEdit}
              <select value={entry.status} disabled={updatingAbsensi[entry.id]} onchange={(event)=>changeStatus(key,event.currentTarget.value,event.currentTarget,s.nama,nomorPertemuan(p))}
               aria-label={"Absensi "+s.nama+" pertemuan "+nomorPertemuan(p)} class={"h-8 w-9 cursor-pointer rounded-md text-center text-xs font-bold disabled:opacity-50 "+statusColor(entry.status)+" "+focus}>
               <option value="hadir">H</option><option value="telat">T</option><option value="izin">I</option><option value="sakit">S</option><option value="alpa">A</option>
              </select>
             {:else}
              <span class={"inline-flex h-8 w-8 items-center justify-center rounded-md text-xs font-bold "+statusColor(entry.status)}>{statusLabel(entry.status)}</span>
             {/if}
             <button type="button" onclick={()=>openCatatan(key,s.nama,nomorPertemuan(p))} disabled={updatingAbsensi[entry.id]} title={entry.catatan?"Lihat catatan":"Lihat atau tambah catatan"} aria-label={"Catatan "+s.nama+" pertemuan "+nomorPertemuan(p)}
              class={"inline-flex h-8 w-7 items-center justify-center rounded-md hover:bg-neutral-100 disabled:opacity-50 dark:hover:bg-white/[0.07] "+(entry.catatan||entry.batas_materi?"text-brand-700 dark:text-brand-400":"text-neutral-400")+" "+focus}><FileText size={14}/></button>
            </div>
           {:else}
            <span title="Belum ada catatan absensi" class="text-neutral-400">—</span>
           {/if}
          </td>
         {/each}
        </tr>
       {/each}
      </tbody>
     </table>
    </div>
   {/if}
   <div class="flex flex-wrap gap-x-4 gap-y-2 border-t border-neutral-100 px-5 py-4 text-xs text-neutral-600 dark:border-white/[0.06] dark:text-neutral-400 sm:px-6">
    <span><span class="mr-1 font-semibold text-emerald-700 dark:text-emerald-400">H</span>Hadir</span>
    <span><span class="mr-1 font-semibold text-amber-700 dark:text-amber-400">T</span>Telat</span>
    <span><span class="mr-1 font-semibold text-sky-700 dark:text-sky-400">I</span>Izin</span>
    <span><span class="mr-1 font-semibold text-indigo-700 dark:text-indigo-400">S</span>Sakit</span>
    <span><span class="mr-1 font-semibold text-rose-700 dark:text-rose-400">A</span>Alpa</span>
    <span>— Belum dicatat</span>
   </div>
  </section>
  <p class="pb-3 text-xs leading-relaxed text-neutral-500 dark:text-neutral-400">Catatan: data hanya menampilkan santri aktif yang masih berada pada kelas ini, mengikuti cakupan rekap sebelumnya. Persentase hadir hanya menghitung status “hadir” dari absensi yang tercatat; “telat” dipantau terpisah. Export CSV berisi satu baris per santri per pertemuan dan mencakup catatan serta batas materi.</p>
 </main>

	{#if catatanModal}
		<div class="fixed inset-0 z-50 flex items-center justify-center p-4">
			<button class="absolute inset-0 h-full w-full bg-neutral-900/50 backdrop-blur-sm" aria-label="Tutup catatan absensi" onclick={() => (catatanModal = null)}></button>
			<div class="relative w-full max-w-lg rounded-2xl border border-neutral-200/80 bg-white p-6 shadow-xl dark:border-white/[0.06] dark:bg-neutral-925" in:fly={{ y: 16, duration: 200 }} role="dialog" aria-modal="true" aria-labelledby="catatan-absensi-title" tabindex="-1">
				<div class="flex items-start gap-3">
					<div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-brand-400/10 text-brand-600 dark:text-brand-400"><FileText class="h-5 w-5" /></div>
					<div>
						<h2 id="catatan-absensi-title" class="font-semibold text-neutral-900 dark:text-white">Catatan Absensi</h2>
						<p class="mt-1 text-sm text-neutral-500 dark:text-neutral-400">{catatanModal.santriNama} · Pertemuan ke-{catatanModal.pertemuanKe}</p>
					</div>
				</div>

				<div class="mt-5 space-y-4">
					{#if catatanModal.status !== absensi[catatanModal.key]?.status}
						<p class="rounded-xl bg-brand-400/10 px-3 py-2.5 text-sm text-neutral-700 dark:text-neutral-300">Status akan diubah menjadi <span class="font-semibold capitalize">{catatanModal.status}</span>. Isi batas materi untuk menyimpan.</p>
					{/if}
					{#if kelas?.materi_individual}
						{#if canEdit && butuhBatasMateri(catatanModal.status)}
							<label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300">Batas materi
								<input bind:value={catatanModal.batasMateri} placeholder="Batas materi *" class="mt-1.5 w-full rounded-xl border border-neutral-300 bg-neutral-100/80 px-3 py-2.5 text-sm text-neutral-900 outline-none focus:border-brand-400 focus:ring-2 focus:ring-brand-400/20 dark:border-neutral-700/80 dark:bg-neutral-800/50 dark:text-white" />
							</label>
						{:else}
							<div>
								<p class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Batas materi</p>
								<p class="mt-1.5 rounded-xl bg-neutral-100/80 px-3 py-2.5 text-sm text-neutral-700 dark:bg-neutral-800/50 dark:text-neutral-300">{absensi[catatanModal.key]?.batas_materi || "Tidak ada progres baru"}</p>
							</div>
						{/if}
					{/if}
					<label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300">Catatan
						<textarea bind:value={catatanModal.catatan} readonly={!canEdit} placeholder="Catatan absensi mahasantri..." rows="4" class="mt-1.5 w-full resize-none rounded-xl border border-neutral-300 bg-neutral-100/80 px-3 py-2.5 text-sm text-neutral-900 outline-none focus:border-brand-400 focus:ring-2 focus:ring-brand-400/20 dark:border-neutral-700/80 dark:bg-neutral-800/50 dark:text-white"></textarea>
					</label>
				</div>

				<div class="mt-5 flex justify-end gap-2">
					<button onclick={() => (catatanModal = null)} class="rounded-xl border border-neutral-300 px-4 py-2.5 text-sm font-medium text-neutral-700 hover:bg-neutral-100 dark:border-neutral-700/80 dark:text-neutral-300 dark:hover:bg-neutral-800">Tutup</button>
					{#if canEdit}<button onclick={saveCatatan} disabled={isSavingCatatan} class="rounded-xl bg-brand-600 px-4 py-2.5 text-sm font-semibold text-white transition-colors hover:bg-brand-700 disabled:cursor-not-allowed disabled:opacity-50 dark:bg-brand-500 dark:hover:bg-brand-400">{isSavingCatatan ? "Menyimpan..." : "Simpan"}</button>{/if}
				</div>
			</div>
		</div>
	{/if}

</AppLayout>