<script lang="ts">
	import { inertia, router } from "@inertiajs/svelte";
	import { fly } from "svelte/transition";
	import AppLayout from "@layouts/AppLayout.svelte";
	import type { Rapat, User } from "@lib/types";
	import { Plus, X, Pencil, Trash2, ClipboardCheck, Users } from "lucide-svelte";

	interface Props { user?: User; rapat?: Rapat[]; success?: string; error?: string; }
	let { user, rapat = [], success, error }: Props = $props();

	const inputClass = "w-full px-3.5 py-2.5 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700 text-sm text-neutral-900 dark:text-white outline-none focus:border-brand-400";
	const STATUS: Record<string, string> = { dijadwalkan: "Dijadwalkan", terlaksana: "Terlaksana", batal: "Batal" };
	const STATUS_COLOR: Record<string, string> = { dijadwalkan: "bg-blue-500/10 text-blue-700 dark:text-blue-400", terlaksana: "bg-green-500/10 text-green-700 dark:text-green-400", batal: "bg-red-500/10 text-red-600 dark:text-red-400" };

	let showForm = $state(false);
	let editId = $state<number | null>(null);
	let saving = $state(false);
	const empty = () => ({ tanggal: "", judul: "", catatan: "", status: "dijadwalkan" });
	let form = $state(empty());

	function openCreate() { editId = null; form = empty(); showForm = true; }
	function openEdit(r: Rapat) { editId = r.id; form = { tanggal: r.tanggal, judul: r.judul, catatan: r.catatan, status: r.status }; showForm = true; }
	function submit() {
		saving = true;
		const opts = { onSuccess: () => { showForm = false; }, onFinish: () => { saving = false; } };
		if (editId) router.put(`/app/koordinator-guru/rapat/${editId}`, form, opts);
		else router.post("/app/koordinator-guru/rapat", form, opts);
	}
	function del(id: number) { if (confirm("Hapus rapat ini?")) router.delete(`/app/koordinator-guru/rapat/${id}`); }
</script>

<AppLayout {user} group="koordinator-rapat">
	<div class="pt-8 pb-10 border-b border-neutral-200/80 dark:border-white/[0.04]">
		<div class="max-w-5xl mx-auto px-4 sm:px-6 flex items-end justify-between gap-4 flex-wrap">
			<div>
				<div class="flex items-center gap-2 text-sm text-neutral-500 mb-3"><a href="/app/koordinator-guru" use:inertia class="hover:text-brand-600">Koordinator Guru</a><span>/</span><span class="text-neutral-700 dark:text-neutral-300">Rapat</span></div>
				<h1 class="text-2xl sm:text-3xl font-bold text-neutral-900 dark:text-white tracking-tight">Rapat Guru</h1>
				<p class="mt-2 text-neutral-600 dark:text-neutral-400">Jadwal rapat bulanan dan absensi kehadiran.</p>
			</div>
			<button onclick={openCreate} class="inline-flex items-center gap-2 px-4 py-2.5 rounded-xl bg-brand-600 hover:bg-brand-700 text-white text-sm font-semibold"><Plus class="w-4 h-4" /> Tambah Rapat</button>
		</div>
	</div>

	<div class="max-w-5xl mx-auto px-4 sm:px-6 py-8 space-y-4">
		{#if success}<div class="rounded-xl bg-green-500/10 border border-green-500/20 p-4 text-sm font-medium text-green-700 dark:text-green-400" in:fly={{ y: 10, duration: 200 }}>{success}</div>{/if}
		{#if error}<div class="rounded-xl bg-red-500/10 border border-red-500/20 p-4 text-sm font-medium text-red-700 dark:text-red-400">{error}</div>{/if}

		<div class="space-y-3">
			{#each rapat as r (r.id)}
				<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-5">
					<div class="flex items-start justify-between gap-4 flex-wrap">
						<div class="min-w-0 flex-1">
							<div class="flex items-center gap-2 flex-wrap"><span class="text-xs font-mono text-neutral-500">{r.tanggal}</span><span class="inline-flex px-2 py-0.5 rounded-full text-xs font-medium {STATUS_COLOR[r.status]}">{STATUS[r.status] ?? r.status}</span></div>
							<h2 class="mt-1.5 font-semibold text-neutral-900 dark:text-white">{r.judul || "(tanpa judul)"}</h2>
							{#if r.catatan}<p class="mt-1 text-sm text-neutral-600 dark:text-neutral-400">{r.catatan}</p>{/if}
						</div>
						<div class="flex items-center gap-2">
							<a href={`/app/koordinator-guru/rapat/${r.id}/absen`} use:inertia class="inline-flex items-center gap-1.5 px-3 py-2 rounded-lg bg-brand-400/10 text-brand-700 dark:text-brand-300 text-xs font-semibold"><ClipboardCheck class="w-4 h-4" /> Absensi</a>
							<button onclick={() => openEdit(r)} class="p-2 rounded-lg hover:bg-neutral-100 dark:hover:bg-neutral-800 text-neutral-500"><Pencil class="w-4 h-4" /></button>
							<button onclick={() => del(r.id)} class="p-2 rounded-lg hover:bg-red-500/10 text-red-500"><Trash2 class="w-4 h-4" /></button>
						</div>
					</div>
				</div>
			{:else}
				<div class="rounded-2xl border border-dashed border-neutral-300 dark:border-neutral-700 p-12 text-center text-sm text-neutral-500"><Users class="w-8 h-8 mx-auto mb-2 opacity-40" /> Belum ada rapat.</div>
			{/each}
		</div>
	</div>

	{#if showForm}
		<div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50" onclick={() => (showForm = false)}>
			<div class="w-full max-w-lg rounded-2xl bg-white dark:bg-neutral-925 border border-neutral-200 dark:border-white/[0.06] shadow-xl" onclick={(e) => e.stopPropagation()}>
				<div class="flex items-center justify-between px-5 py-4 border-b border-neutral-200/80 dark:border-white/[0.04]">
					<h2 class="text-lg font-bold text-neutral-900 dark:text-white">{editId ? "Edit" : "Tambah"} Rapat</h2>
					<button onclick={() => (showForm = false)} class="p-1.5 rounded-lg hover:bg-neutral-100 dark:hover:bg-neutral-800 text-neutral-500"><X class="w-5 h-5" /></button>
				</div>
				<form onsubmit={(e) => { e.preventDefault(); submit(); }} class="p-5 space-y-4">
					<div class="grid sm:grid-cols-2 gap-4">
						<label class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Tanggal<input type="date" bind:value={form.tanggal} required class={`${inputClass} mt-1.5`} /></label>
						<label class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Status<select bind:value={form.status} class={`${inputClass} mt-1.5`}><option value="dijadwalkan">Dijadwalkan</option><option value="terlaksana">Terlaksana</option><option value="batal">Batal</option></select></label>
					</div>
					<label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300">Judul<input bind:value={form.judul} class={`${inputClass} mt-1.5`} /></label>
					<label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300">Catatan<textarea bind:value={form.catatan} rows="2" class={`${inputClass} mt-1.5`}></textarea></label>
					<div class="pt-3 border-t border-neutral-200/80 dark:border-white/[0.04] flex justify-end gap-3">
						<button type="button" onclick={() => (showForm = false)} class="px-4 py-2.5 rounded-xl text-sm font-semibold text-neutral-600 dark:text-neutral-300 hover:bg-neutral-100 dark:hover:bg-neutral-800">Batal</button>
						<button type="submit" disabled={saving} class="px-5 py-2.5 rounded-xl bg-brand-600 hover:bg-brand-700 text-white text-sm font-semibold disabled:opacity-50">{saving ? "Menyimpan..." : "Simpan"}</button>
					</div>
				</form>
			</div>
		</div>
	{/if}
</AppLayout>
