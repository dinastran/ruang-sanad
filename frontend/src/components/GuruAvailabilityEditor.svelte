<script lang="ts">
	import { router } from "@inertiajs/svelte";
	import { CalendarClock, Plus, Pencil, Trash2, Save, X } from "lucide-svelte";

	type Slot = {
		id: number; guru_id: number; hari: number;
		jam_mulai: string; jam_selesai: string;
		berlaku_mulai: string; berlaku_sampai: string;
		confirmed_by: number; confirmed_at: string;
	};
	type Input = {
		hari: number; jam_mulai: string; jam_selesai: string;
		berlaku_mulai: string; berlaku_sampai: string;
	};
	let { guruId, items = [] }: { guruId: number; items?: Slot[] } = $props();
	const hariNames = ["", "Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu", "Ahad"];
	const todayWIB = new Intl.DateTimeFormat("sv-SE", { timeZone: "Asia/Jakarta", year: "numeric", month: "2-digit", day: "2-digit" }).format(new Date());
	const empty = (): Input => ({ hari: 1, jam_mulai: "19:30", jam_selesai: "21:00", berlaku_mulai: todayWIB, berlaku_sampai: "" });
	let editingId = $state<number | null>(null);
	let form = $state<Input>(empty());
	let busy = $state(false);
	const field = "w-full rounded-lg border border-neutral-300 dark:border-neutral-700 bg-white dark:bg-neutral-900 px-3 py-2 text-sm text-neutral-900 dark:text-white";

	function addSlot() { form = empty(); editingId = 0; }
	function editSlot(item: Slot) {
		editingId = item.id;
		form = {
			hari: item.hari, jam_mulai: item.jam_mulai, jam_selesai: item.jam_selesai,
			berlaku_mulai: item.berlaku_mulai, berlaku_sampai: item.berlaku_sampai || "",
		};
	}
	function save(event: SubmitEvent) {
		event.preventDefault();
		if (busy || editingId === null) return;
		busy = true;
		const base = "/app/koordinator-guru/guru/" + guruId + "/availability";
		const options = {
			preserveScroll: true,
			onSuccess: (page: { props: Record<string, unknown> }) => {
				const flash = page.props.flash as { error?: string } | undefined;
				if (!flash?.error) editingId = null;
			},
			onFinish: () => { busy = false; },
		};
		if (editingId === 0) router.post(base, { ...form }, options);
		else router.put(base + "/" + editingId, { ...form }, options);
	}
	function remove(item: Slot) {
		if (busy || !confirm("Hapus slot available " + hariNames[item.hari] + " " + item.jam_mulai + "–" + item.jam_selesai + "? Jadwal kelas yang sudah berjalan tidak berubah.")) return;
		busy = true;
		router.delete("/app/koordinator-guru/guru/" + guruId + "/availability/" + item.id, { preserveScroll: true, onFinish: () => { busy = false; } });
	}
	function dateLabel(value: string) {
		if (!value) return "seterusnya";
		const parts = value.slice(0, 10).split("-");
		return parts.length === 3 ? parts[2] + "/" + parts[1] + "/" + parts[0] : value;
	}
</script>

<section class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-5 sm:p-6 space-y-4">
	<div class="flex flex-wrap items-center justify-between gap-3">
		<div>
			<h2 class="flex items-center gap-2 font-semibold text-neutral-900 dark:text-white"><CalendarClock class="w-5 h-5 text-brand-600 dark:text-brand-300" /> Jadwal Available Guru</h2>
			<p class="mt-1 text-sm text-neutral-600 dark:text-neutral-400">Kelola waktu yang guru konfirmasi tersedia untuk menerima kelas baru.</p>
		</div>
		{#if editingId === null}
			<button type="button" onclick={addSlot} class="inline-flex items-center gap-2 rounded-xl bg-brand-600 px-4 py-2 text-sm font-semibold text-white hover:bg-brand-700"><Plus class="h-4 w-4"/> Tambah slot</button>
		{/if}
	</div>
	<p class="text-xs text-neutral-500">Slot di sini adalah komitmen availability, bukan jadwal kelas aktif. Rekomendasi kelas tetap harus memeriksa benturan dengan jadwal mengajar aktual. Durasi availability minimal 90 menit.</p>
	<div class="space-y-2">
		{#each items as item (item.id)}
			<div class="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-neutral-200 dark:border-white/10 p-3">
				<div>
					<p class="text-sm font-semibold text-neutral-900 dark:text-white">{hariNames[item.hari]} · {item.jam_mulai}–{item.jam_selesai} WIB</p>
					<p class="text-xs text-neutral-500">Berlaku {dateLabel(item.berlaku_mulai)} sampai {dateLabel(item.berlaku_sampai)}</p>
					<p class="text-xs text-neutral-500">{item.berlaku_sampai && item.berlaku_sampai < todayWIB ? "Periode berakhir" : "Terkonfirmasi Koordinator Guru"}</p>
				</div>
				<div class="flex items-center gap-2">
					<button type="button" onclick={() => editSlot(item)} disabled={busy} class="inline-flex items-center gap-1 rounded-lg border border-neutral-300 dark:border-neutral-700 px-3 py-2 text-xs font-medium hover:bg-neutral-100 dark:hover:bg-neutral-800 disabled:opacity-50"><Pencil class="h-3.5 w-3.5"/> Edit</button>
					<button type="button" onclick={() => remove(item)} disabled={busy} class="inline-flex items-center gap-1 rounded-lg border border-red-500/30 px-3 py-2 text-xs font-medium text-red-600 dark:text-red-400 hover:bg-red-500/10 disabled:opacity-50"><Trash2 class="h-3.5 w-3.5"/> Hapus</button>
				</div>
			</div>
		{:else}
			<p class="rounded-xl bg-neutral-100/70 p-4 text-sm text-neutral-500 dark:bg-neutral-800/40">Belum ada jadwal available yang dikonfirmasi.</p>
		{/each}
	</div>
	{#if editingId !== null}
		<form onsubmit={save} class="space-y-3 rounded-xl border border-brand-500/20 bg-brand-500/5 p-4">
			<h3 class="text-sm font-semibold text-neutral-900 dark:text-white">{editingId === 0 ? "Tambah jadwal available" : "Edit jadwal available"}</h3>
			<div class="grid gap-3 sm:grid-cols-3">
				<label class="text-xs font-medium text-neutral-700 dark:text-neutral-300">Hari
					<select bind:value={form.hari} class={field}>
						{#each hariNames.slice(1) as name, index}<option value={index + 1}>{name}</option>{/each}
					</select>
				</label>
				<label class="text-xs font-medium text-neutral-700 dark:text-neutral-300">Jam mulai
					<input type="time" bind:value={form.jam_mulai} required class={field}/>
				</label>
				<label class="text-xs font-medium text-neutral-700 dark:text-neutral-300">Jam selesai
					<input type="time" bind:value={form.jam_selesai} required class={field}/>
				</label>
				<label class="text-xs font-medium text-neutral-700 dark:text-neutral-300">Berlaku mulai
					<input type="date" bind:value={form.berlaku_mulai} required class={field}/>
				</label>
				<label class="text-xs font-medium text-neutral-700 dark:text-neutral-300">Berlaku sampai (opsional)
					<input type="date" min={form.berlaku_mulai} bind:value={form.berlaku_sampai} class={field}/>
				</label>
			</div>
			<div class="flex flex-wrap justify-end gap-2">
				<button type="button" onclick={() => (editingId = null)} disabled={busy} class="inline-flex items-center gap-1 rounded-lg border border-neutral-300 dark:border-neutral-700 px-4 py-2 text-sm"><X class="h-4 w-4"/> Batal</button>
				<button type="submit" disabled={busy} class="inline-flex items-center gap-2 rounded-lg bg-brand-600 px-4 py-2 text-sm font-semibold text-white disabled:opacity-50"><Save class="h-4 w-4"/> {busy ? "Menyimpan..." : "Simpan dan konfirmasi"}</button>
			</div>
		</form>
	{/if}
</section>
