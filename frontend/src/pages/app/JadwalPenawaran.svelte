<script lang="ts">
	import { router } from "@inertiajs/svelte";
	import AppLayout from "@layouts/AppLayout.svelte";
	import type { Flash, User } from "@lib/types";
	import { formatDateTime } from "@lib/leads";
	import { Plus } from "lucide-svelte";

	interface Jadwal {
		id: number;
		kode: string;
		hari: string;
		hari_list: number[];
		frekuensi: number;
		jam_mulai: string;
		jam_selesai: string;
		label: string;
		level: string;
		kuota_per_gender: number;
		is_buka: boolean;
		urutan: number;
		catatan: string;
		periode_mulai: string;
		terisi_ikhwan: number;
		terisi_akhwat: number;
		sisa_ikhwan: number;
		sisa_akhwat: number;
		tanpa_gender: number;
	}
	interface Props {
		user?: User;
		jadwal: Jadwal[];
		flash?: Flash;
	}
	interface FormState {
		kode: string;
		hari: number[];
		jam_mulai: string;
		jam_selesai: string;
		level: string;
		kuota_per_gender: number;
		is_buka: boolean;
		urutan: number;
		catatan: string;
	}

	let props: Props = $props();
	let canEdit = $derived(props.user?.role === "admin_kelas" || props.user?.role === "super_admin");

	const HARI = [
		[1, "Senin"], [2, "Selasa"], [3, "Rabu"], [4, "Kamis"], [5, "Jum'at"], [6, "Sabtu"], [7, "Ahad"],
	] as const;

	const empty = (): FormState => ({ kode: "", hari: [], jam_mulai: "19:30", jam_selesai: "21:00", level: "", kuota_per_gender: 10, is_buka: true, urutan: 50, catatan: "" });
	let editingId = $state<number | null>(null); // 0 = new
	let form = $state<FormState>(empty());
	let busy = $state(false);

	function startNew() {
		form = empty();
		editingId = 0;
	}
	function startEdit(j: Jadwal) {
		form = { kode: j.kode, hari: [...j.hari_list], jam_mulai: j.jam_mulai, jam_selesai: j.jam_selesai, level: j.level, kuota_per_gender: j.kuota_per_gender, is_buka: j.is_buka, urutan: j.urutan, catatan: j.catatan };
		editingId = j.id;
	}
	function toggleHari(d: number) {
		form.hari = form.hari.includes(d) ? form.hari.filter((x) => x !== d) : [...form.hari, d].sort();
	}
	function payload(f: FormState) {
		return { ...f, hari: f.hari.join(","), kuota_per_gender: Number(f.kuota_per_gender), urutan: Number(f.urutan) };
	}
	const opts = { preserveScroll: true, onFinish: () => (busy = false) };
	function save(e: Event) {
		e.preventDefault();
		busy = true;
		const done = { ...opts, onSuccess: () => (editingId = null) };
		if (editingId === 0) router.post("/app/jadwal-penawaran", payload(form), done);
		else router.put(`/app/jadwal-penawaran/${editingId}`, payload(form), done);
	}
	function toggleBuka(j: Jadwal) {
		busy = true;
		const f: FormState = { kode: j.kode, hari: j.hari_list, jam_mulai: j.jam_mulai, jam_selesai: j.jam_selesai, level: j.level, kuota_per_gender: j.kuota_per_gender, is_buka: !j.is_buka, urutan: j.urutan, catatan: j.catatan };
		router.put(`/app/jadwal-penawaran/${j.id}`, payload(f), opts);
	}
	function periodeBaru(j: Jadwal) {
		if (!confirm(`Mulai periode baru untuk ${j.label}? Hitungan kursi terisi kembali ke 0. Lakukan ini saat angkatan baru dibuka.`)) return;
		busy = true;
		router.post(`/app/jadwal-penawaran/${j.id}/periode-baru`, {}, opts);
	}
	function hapus(j: Jadwal) {
		if (!confirm(`Hapus jadwal ${j.label}?`)) return;
		busy = true;
		router.delete(`/app/jadwal-penawaran/${j.id}`, opts);
	}

	const field = "w-full px-3 py-2 rounded-lg bg-white dark:bg-neutral-900/60 border border-neutral-300 dark:border-neutral-700/80 text-sm text-neutral-900 dark:text-white";
	const card = "rounded-xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-4";
	const labelCls = "block text-xs font-medium text-neutral-600 dark:text-neutral-400";
	const btnGhost = "px-3 py-1.5 rounded-lg border border-neutral-300 dark:border-neutral-700 text-xs font-medium text-neutral-700 dark:text-neutral-300 hover:bg-neutral-100 dark:hover:bg-neutral-800 disabled:opacity-50";
	const btnPrimary = "px-4 py-2 rounded-lg bg-brand-600 hover:bg-brand-700 text-white text-sm font-semibold dark:bg-brand-500 disabled:opacity-50";

	function seatCls(sisa: number) {
		return sisa === 0 ? "text-error" : sisa <= 2 ? "text-warning" : "text-neutral-900 dark:text-white";
	}
</script>

{#snippet editor()}
	<form class="{card} space-y-3" onsubmit={save}>
		<h2 class="text-sm font-semibold text-neutral-900 dark:text-white">{editingId === 0 ? "Jadwal baru" : "Ubah jadwal"}</h2>
		{#if editingId === 0}
			<label class={labelCls}>Kode (dipakai CS AI, tidak bisa diubah)
				<input class="{field} mt-1 font-mono" bind:value={form.kode} placeholder="mis. kamis-malam" maxlength="40" required />
			</label>
		{:else}
			<p class="text-xs text-neutral-500">Kode: <span class="font-mono">{form.kode}</span></p>
		{/if}
		<fieldset>
			<legend class={labelCls}>Hari ({form.hari.length}x sepekan)</legend>
			<div class="mt-1 flex flex-wrap gap-2">
				{#each HARI as [d, nama]}
					<button type="button" class="px-3 py-1.5 rounded-full border text-xs font-medium {form.hari.includes(d) ? 'bg-brand-600 border-brand-600 text-white dark:bg-brand-500' : 'border-neutral-300 dark:border-neutral-700 text-neutral-700 dark:text-neutral-300'}" aria-pressed={form.hari.includes(d)} onclick={() => toggleHari(d)}>{nama}</button>
				{/each}
			</div>
		</fieldset>
		<div class="grid grid-cols-2 gap-3">
			<label class={labelCls}>Mulai (WIB)<input type="time" class="{field} mt-1" bind:value={form.jam_mulai} required /></label>
			<label class={labelCls}>Selesai (WIB)<input type="time" class="{field} mt-1" bind:value={form.jam_selesai} required /></label>
			<label class={labelCls}>Level
				<select class="{field} mt-1" bind:value={form.level}>
					<option value="">Semua level</option><option value="1">Level 1</option><option value="2">Level 2</option>
				</select>
			</label>
			<label class={labelCls}>Kuota per gender<input type="number" min="1" max="500" class="{field} mt-1" bind:value={form.kuota_per_gender} /></label>
			<label class={labelCls}>Urutan tampil<input type="number" class="{field} mt-1" bind:value={form.urutan} /></label>
			<label class="flex items-end gap-2 pb-2 text-sm text-neutral-800 dark:text-neutral-200"><input type="checkbox" bind:checked={form.is_buka} /> Dibuka</label>
		</div>
		<label class={labelCls}>Catatan untuk CS AI (opsional)
			<input class="{field} mt-1" bind:value={form.catatan} maxlength="300" placeholder="mis. cocok untuk yang bekerja" />
		</label>
		<div class="flex justify-end gap-2">
			<button type="button" class={btnGhost} onclick={() => (editingId = null)}>Batal</button>
			<button type="submit" class={btnPrimary} disabled={busy || form.hari.length === 0}>{busy ? "Menyimpan..." : "Simpan"}</button>
		</div>
	</form>
{/snippet}

<AppLayout user={props.user}>
	<div class="max-w-3xl mx-auto px-4 sm:px-6 py-6 space-y-5">
		<div class="flex flex-wrap items-end justify-between gap-3">
			<div>
				<h1 class="text-2xl sm:text-3xl font-bold text-neutral-900 dark:text-white tracking-tight">Jadwal Penawaran</h1>
				<p class="mt-1 text-sm text-neutral-600 dark:text-neutral-400">Jadwal yang ditawarkan CS AI ke calon santri. Kursi terhitung terisi saat pembayaran diverifikasi. Jadwal penuh atau ditutup tidak ditawarkan.</p>
			</div>
			{#if canEdit && editingId === null}
				<button type="button" class="inline-flex items-center gap-1 {btnPrimary}" onclick={startNew}><Plus class="w-4 h-4" /> Tambah</button>
			{/if}
		</div>
		{#if props.flash?.success}<div class="rounded-xl border border-success/20 bg-success/10 px-4 py-3 text-sm font-medium text-success">{props.flash.success}</div>{/if}
		{#if props.flash?.error}<div class="rounded-xl border border-error/20 bg-error/10 px-4 py-3 text-sm font-medium text-error">{props.flash.error}</div>{/if}

		{#if editingId === 0}{@render editor()}{/if}

		{#if props.jadwal.length === 0}
			<p class="{card} text-sm text-neutral-600 dark:text-neutral-400">Belum ada jadwal. CS AI akan mengalihkan pertanyaan jadwal ke admin.</p>
		{/if}

		<ul class="space-y-3">
			{#each props.jadwal as j (j.id)}
				<li>
					{#if editingId === j.id}
						{@render editor()}
					{:else}
						<div class="{card} {j.is_buka ? '' : 'opacity-70'}">
							<div class="flex flex-wrap items-start justify-between gap-2">
								<div class="min-w-0">
									<p class="font-semibold text-neutral-900 dark:text-white">{j.label}</p>
									<p class="mt-0.5 text-xs text-neutral-500">{j.frekuensi}x sepekan · {j.level ? `Level ${j.level}` : "Semua level"} · <span class="font-mono">{j.kode}</span></p>
									{#if j.catatan}<p class="mt-1 text-xs text-neutral-600 dark:text-neutral-400">{j.catatan}</p>{/if}
								</div>
								<span class="shrink-0 rounded-full px-2 py-0.5 text-xs font-medium {j.is_buka ? 'bg-success/10 text-success' : 'bg-neutral-200 text-neutral-600 dark:bg-neutral-800 dark:text-neutral-400'}">{j.is_buka ? "Dibuka" : "Ditutup"}</span>
							</div>
							<dl class="mt-3 grid grid-cols-2 gap-3 text-sm">
								<div><dt class="text-xs text-neutral-500">Ikhwan</dt><dd class="font-medium {seatCls(j.sisa_ikhwan)}">{j.terisi_ikhwan}/{j.kuota_per_gender} terisi{j.sisa_ikhwan === 0 ? " · penuh" : ""}</dd></div>
								<div><dt class="text-xs text-neutral-500">Akhwat</dt><dd class="font-medium {seatCls(j.sisa_akhwat)}">{j.terisi_akhwat}/{j.kuota_per_gender} terisi{j.sisa_akhwat === 0 ? " · penuh" : ""}</dd></div>
							</dl>
							{#if j.tanpa_gender > 0}<p class="mt-2 text-xs text-warning">{j.tanpa_gender} lead terverifikasi belum diisi ikhwan/akhwat, jadi belum terhitung di kuota.</p>{/if}
							<p class="mt-2 text-xs text-neutral-500">Periode sejak {formatDateTime(j.periode_mulai)}</p>
							{#if canEdit}
								<div class="mt-3 flex flex-wrap gap-2">
									<button type="button" class={btnGhost} disabled={busy || editingId !== null} onclick={() => startEdit(j)}>Ubah</button>
									<button type="button" class={btnGhost} disabled={busy} onclick={() => toggleBuka(j)}>{j.is_buka ? "Tutup" : "Buka"}</button>
									<button type="button" class={btnGhost} disabled={busy} onclick={() => periodeBaru(j)}>Periode baru</button>
									<button type="button" class="{btnGhost} !text-error" disabled={busy} onclick={() => hapus(j)}>Hapus</button>
								</div>
							{/if}
						</div>
					{/if}
				</li>
			{/each}
		</ul>
	</div>
</AppLayout>
