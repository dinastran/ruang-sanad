<script lang="ts">
	import { inertia, router } from "@inertiajs/svelte";
	import { fly } from "svelte/transition";
	import AppLayout from "@layouts/AppLayout.svelte";
	import type { GuruRingkas, KelasSimple, Kunjungan, User } from "@lib/types";
	import { Plus, X, Pencil, Trash2, BookOpen } from "lucide-svelte";

	interface Props { user?: User; kunjungan?: Kunjungan[]; guruList?: GuruRingkas[]; kelasList?: KelasSimple[]; success?: string; error?: string; }
	let { user, kunjungan = [], guruList = [], kelasList = [], success, error }: Props = $props();

	const inputClass = "w-full px-3.5 py-2.5 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700 text-sm text-neutral-900 dark:text-white outline-none focus:border-brand-400";
	const STATUS: Record<string, string> = { dijadwalkan: "Dijadwalkan", terlaksana: "Terlaksana", ditunda: "Ditunda", batal: "Batal" };
	const STATUS_COLOR: Record<string, string> = { dijadwalkan: "bg-blue-500/10 text-blue-700 dark:text-blue-400", terlaksana: "bg-green-500/10 text-green-700 dark:text-green-400", ditunda: "bg-amber-500/10 text-amber-700 dark:text-amber-400", batal: "bg-red-500/10 text-red-600 dark:text-red-400" };

	let showForm = $state(false);
	let editId = $state<number | null>(null);
	let saving = $state(false);
	const empty = () => ({ guru_id: 0, kelas_id: 0, target_mulai: "", target_selesai: "", tanggal: "", jam: "", status: "dijadwalkan", catatan: "" });
	let form = $state(empty());

	function openCreate() { editId = null; form = empty(); showForm = true; }
	function openEdit(k: Kunjungan) { editId = k.id; form = { guru_id: k.guru_id, kelas_id: k.kelas_id ?? 0, target_mulai: k.target_mulai, target_selesai: k.target_selesai, tanggal: k.tanggal, jam: k.jam, status: k.status, catatan: k.catatan }; showForm = true; }
	function submit() {
		saving = true;
		const opts = { onSuccess: () => { showForm = false; }, onFinish: () => { saving = false; } };
		if (editId) router.put(`/app/koordinator-guru/kunjungan/${editId}`, form, opts);
		else router.post("/app/koordinator-guru/kunjungan", form, opts);
	}
	function del(id: number) { if (confirm("Hapus kunjungan ini?")) router.delete(`/app/koordinator-guru/kunjungan/${id}`); }
</script>

<AppLayout {user} group="koordinator-kunjungan">
	<div class="pt-8 pb-10 border-b border-neutral-200/80 dark:border-white/[0.04]">
		<div class="max-w-5xl mx-auto px-4 sm:px-6 flex items-end justify-between gap-4 flex-wrap">
			<div>
				<div class="flex items-center gap-2 text-sm text-neutral-500 mb-3"><a href="/app/koordinator-guru" use:inertia class="hover:text-brand-600">Koordinator Guru</a><span>/</span><span class="text-neutral-700 dark:text-neutral-300">Kunjungan</span></div>
				<h1 class="text-2xl sm:text-3xl font-bold text-neutral-900 dark:text-white tracking-tight">Kunjungan Kelas</h1>
				<p class="mt-2 text-neutral-600 dark:text-neutral-400">Jadwalkan dan catat hasil kunjungan kelas per guru.</p>
			</div>
			<button onclick={openCreate} class="inline-flex items-center gap-2 px-4 py-2.5 rounded-xl bg-brand-600 hover:bg-brand-700 text-white text-sm font-semibold"><Plus class="w-4 h-4" /> Jadwalkan</button>
		</div>
	</div>

	<div class="max-w-5xl mx-auto px-4 sm:px-6 py-8 space-y-4">
		{#if success}<div class="rounded-xl bg-green-500/10 border border-green-500/20 p-4 text-sm font-medium text-green-700 dark:text-green-400" in:fly={{ y: 10, duration: 200 }}>{success}</div>{/if}
		{#if error}<div class="rounded-xl bg-red-500/10 border border-red-500/20 p-4 text-sm font-medium text-red-700 dark:text-red-400">{error}</div>{/if}

		<div class="space-y-3">
			{#each kunjungan as k (k.id)}
				<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-5">
					<div class="flex items-start justify-between gap-4 flex-wrap">
						<div class="min-w-0 flex-1">
							<div class="flex items-center gap-2 flex-wrap"><h2 class="font-semibold text-neutral-900 dark:text-white">{k.guru_nama}</h2><span class="inline-flex px-2 py-0.5 rounded-full text-xs font-medium {STATUS_COLOR[k.status]}">{STATUS[k.status] ?? k.status}</span></div>
							<p class="mt-1 text-sm text-neutral-600 dark:text-neutral-400">{k.kelas_nama || "Kelas belum dipilih"}</p>
							<p class="mt-1 text-xs text-neutral-500">{#if k.tanggal}Pelaksanaan: {k.tanggal} {k.jam}{:else if k.target_mulai}Target: {k.target_mulai} — {k.target_selesai}{/if}</p>
							{#if k.catatan}<p class="mt-1.5 text-sm text-neutral-600 dark:text-neutral-400 italic">"{k.catatan}"</p>{/if}
						</div>
						<div class="flex items-center gap-2">
							<button onclick={() => openEdit(k)} class="p-2 rounded-lg hover:bg-neutral-100 dark:hover:bg-neutral-800 text-neutral-500"><Pencil class="w-4 h-4" /></button>
							<button onclick={() => del(k.id)} class="p-2 rounded-lg hover:bg-red-500/10 text-red-500"><Trash2 class="w-4 h-4" /></button>
						</div>
					</div>
				</div>
			{:else}
				<div class="rounded-2xl border border-dashed border-neutral-300 dark:border-neutral-700 p-12 text-center text-sm text-neutral-500"><BookOpen class="w-8 h-8 mx-auto mb-2 opacity-40" /> Belum ada kunjungan terjadwal.</div>
			{/each}
		</div>
	</div>

	{#if showForm}
		<div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50" onclick={() => (showForm = false)}>
			<div class="w-full max-w-lg max-h-[90vh] overflow-y-auto rounded-2xl bg-white dark:bg-neutral-925 border border-neutral-200 dark:border-white/[0.06] shadow-xl" onclick={(e) => e.stopPropagation()}>
				<div class="flex items-center justify-between px-5 py-4 border-b border-neutral-200/80 dark:border-white/[0.04]">
					<h2 class="text-lg font-bold text-neutral-900 dark:text-white">{editId ? "Edit" : "Jadwalkan"} Kunjungan</h2>
					<button onclick={() => (showForm = false)} class="p-1.5 rounded-lg hover:bg-neutral-100 dark:hover:bg-neutral-800 text-neutral-500"><X class="w-5 h-5" /></button>
				</div>
				<form onsubmit={(e) => { e.preventDefault(); submit(); }} class="p-5 space-y-4">
					<div class="grid sm:grid-cols-2 gap-4">
						<label class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Guru<select bind:value={form.guru_id} required class={`${inputClass} mt-1.5`}><option value={0} disabled>Pilih guru...</option>{#each guruList as g (g.id)}<option value={g.id}>{g.nama}</option>{/each}</select></label>
						<label class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Kelas<select bind:value={form.kelas_id} class={`${inputClass} mt-1.5`}><option value={0}>—</option>{#each kelasList as k (k.id)}<option value={k.id}>{k.nama_kelas}</option>{/each}</select></label>
						<label class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Target mulai<input type="date" bind:value={form.target_mulai} class={`${inputClass} mt-1.5`} /></label>
						<label class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Target selesai<input type="date" bind:value={form.target_selesai} class={`${inputClass} mt-1.5`} /></label>
						<label class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Tanggal aktual<input type="date" bind:value={form.tanggal} required={form.status === "terlaksana"} class={`${inputClass} mt-1.5`} /></label>
						<label class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Jam masuk<input type="time" bind:value={form.jam} required={form.status === "terlaksana"} class={`${inputClass} mt-1.5`} /></label>
						<label class="text-sm font-medium text-neutral-700 dark:text-neutral-300 sm:col-span-2">Status<select bind:value={form.status} class={`${inputClass} mt-1.5`}><option value="dijadwalkan">Dijadwalkan</option><option value="terlaksana">Terlaksana</option><option value="ditunda">Ditunda</option><option value="batal">Batal</option></select></label>
					</div>
					<label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300">Keterangan kunjungan{#if form.status === "terlaksana"}<span class="text-error"> *</span>{/if}<textarea bind:value={form.catatan} required={form.status === "terlaksana"} rows="2" class={`${inputClass} mt-1.5`}></textarea></label>
					<div class="pt-3 border-t border-neutral-200/80 dark:border-white/[0.04] flex justify-end gap-3">
						<button type="button" onclick={() => (showForm = false)} class="px-4 py-2.5 rounded-xl text-sm font-semibold text-neutral-600 dark:text-neutral-300 hover:bg-neutral-100 dark:hover:bg-neutral-800">Batal</button>
						<button type="submit" disabled={saving || !form.guru_id} class="px-5 py-2.5 rounded-xl bg-brand-600 hover:bg-brand-700 text-white text-sm font-semibold disabled:opacity-50">{saving ? "Menyimpan..." : "Simpan"}</button>
					</div>
				</form>
			</div>
		</div>
	{/if}
</AppLayout>
