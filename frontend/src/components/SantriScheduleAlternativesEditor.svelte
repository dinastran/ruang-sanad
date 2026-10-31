<script lang="ts">
	import { router } from "@inertiajs/svelte";
	import { CalendarClock, Save } from "lucide-svelte";

	type Offer = { kode:string; hari:string; jam_mulai:string; jam_selesai:string; gender:string; level:string };
	type Alternative = { id:number; jadwal_kode:string; status:string; consent_channel:string; consent_note:string; consent_confirmed_by:number; consent_confirmed_at:string; updated_at:string; label:string };
	let {santriId, offers=[], choices=[]}: {santriId:number; offers?:Offer[]; choices?:Alternative[]}=$props();
	let selected=$state("");
	let status=$state("proposed");
	let channel=$state("");
	let note=$state("");
	let busy=$state(false);
	const classes="w-full rounded-lg border border-neutral-300 dark:border-neutral-700 bg-white dark:bg-neutral-900 p-2 text-sm text-neutral-900 dark:text-neutral-100";
	const days=["","Senin","Selasa","Rabu","Kamis","Jumat","Sabtu","Ahad"];
	function formatDays(raw:string) {
		return raw.split(",").map(n=>days[Number(n)]??n).join(", ");
	}
	function choose(code:string) {
		selected=code;
		const current=choices.find(x=>x.jadwal_kode===code);
		status=current?.status==="revoked"?"proposed":current?.status??"proposed";
		channel=status==="approved"?(current?.consent_channel??""):"";
		note=current?.consent_note??"";
	}
	function save(e:SubmitEvent) {
		e.preventDefault();
		if(!selected||busy)return;
		busy=true;
		router.put("/app/santri/"+santriId+"/schedule-alternative",{
			jadwal_kode:selected,status,consent_channel:status==="approved"?channel:"",consent_note:note
		},{preserveScroll:true,onFinish:()=>{busy=false;}});
	}
</script>

<section class="rounded-2xl border border-neutral-200 bg-white p-6 dark:border-white/10 dark:bg-neutral-900 space-y-4">
	<div>
		<h2 class="flex items-center gap-2 font-semibold"><CalendarClock class="w-5 h-5"/> Alternatif Jadwal Santri</h2>
		<p class="mt-1 text-xs text-neutral-500">Jadwal alternatif hanya dapat dipakai setelah santri mengonfirmasi. Data ini tidak mengubah pilihan utama atau langsung memindahkan kelas.</p>
	</div>
	{#if choices.length}
		<ul class="space-y-2">
			{#each choices as item (item.id)}
				<li class="rounded-lg border border-neutral-200 dark:border-white/10 p-3 text-sm">
					<div class="flex items-center justify-between gap-2">
						<strong>{item.jadwal_kode} · {item.label}</strong>
						<span class="text-xs">{item.status==="approved"?"Disetujui santri":item.status==="revoked"?"Dicabut":"Diusulkan"}</span>
					</div>
					{#if item.consent_note}<p class="mt-1 text-xs text-neutral-500">Catatan: {item.consent_note}</p>{/if}
					<button type="button" onclick={()=>choose(item.jadwal_kode)} class="mt-2 text-xs font-semibold text-brand-600 dark:text-brand-300">Ubah status</button>
				</li>
			{/each}
		</ul>
	{/if}
	<form onsubmit={save} class="space-y-3">
		<label class="block text-xs font-medium">Jadwal alternatif
			<select class={classes} value={selected} onchange={e=>choose(e.currentTarget.value)} required>
				<option value="">Pilih jadwal penawaran</option>
				{#each offers as item (item.kode)}
					<option value={item.kode}>{item.kode} · {formatDays(item.hari)} {item.jam_mulai}–{item.jam_selesai}</option>
				{/each}
				{#each choices.filter(x=>!offers.some(o=>o.kode===x.jadwal_kode)) as item (item.id)}
					<option value={item.jadwal_kode}>{item.jadwal_kode} · {item.label} (tidak ditawarkan lagi)</option>
				{/each}
			</select>
		</label>
		{#if selected}
			<label class="block text-xs font-medium">Status persetujuan
				<select bind:value={status} class={classes} onchange={()=>{if(status!=="approved")channel="";}}>
					<option value="proposed">Diusulkan, belum disetujui</option>
					<option value="approved">Disetujui santri</option>
					{#if choices.some(x=>x.jadwal_kode===selected)}<option value="revoked">Dicabut</option>{/if}
				</select>
			</label>
			{#if status==="approved"}
				<label class="block text-xs font-medium">Media persetujuan
					<select bind:value={channel} class={classes} required>
						<option value="">Pilih media konfirmasi</option>
						<option value="whatsapp">WhatsApp</option>
						<option value="phone">Telepon</option>
						<option value="other">Lainnya</option>
					</select>
				</label>
			{/if}
			<label class="block text-xs font-medium">{status==="approved"?"Catatan rujukan persetujuan santri":status==="revoked"?"Alasan pencabutan":"Catatan usulan (opsional)"}
				<textarea bind:value={note} maxlength="500" rows="2" required={status!=="proposed"} class={classes} placeholder="Catat tanggal, konteks, dan bukti konfirmasi tanpa menyalin data rahasia."/>
			</label>
			<div class="flex justify-end"><button type="submit" disabled={busy||(status!=="proposed"&&!note.trim())||(status==="approved"&&!channel)} class="inline-flex items-center gap-2 rounded-lg bg-brand-600 px-4 py-2 text-sm font-semibold text-white disabled:opacity-50"><Save class="w-4 h-4"/>{busy?"Menyimpan...":"Simpan alternatif"}</button></div>
		{/if}
	</form>
</section>
