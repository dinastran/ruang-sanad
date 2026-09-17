<script lang="ts">
	import { inertia, router } from "@inertiajs/svelte";
	import { fly } from "svelte/transition";
	import AppLayout from "@layouts/AppLayout.svelte";
	import type { WaTemplate, User } from "@lib/types";
	import { Plus, X, Pencil, Trash2, MessageCircle } from "lucide-svelte";

	interface Props { user?: User; templates?: WaTemplate[]; success?: string; error?: string; }
	let { user, templates = [], success, error }: Props = $props();

	const inputClass = "w-full px-3.5 py-2.5 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700 text-sm text-neutral-900 dark:text-white outline-none focus:border-brand-400";
	let showForm = $state(false);
	let editId = $state<number | null>(null);
	let saving = $state(false);
	const empty = () => ({ nama: "", target_type: "santri", body: "", is_aktif: true });
	let form = $state(empty());

	function openCreate() { editId = null; form = empty(); showForm = true; }
	function openEdit(t: WaTemplate) { editId = t.id; form = { nama: t.nama, target_type: t.target_type, body: t.body, is_aktif: t.is_aktif }; showForm = true; }
	function submit() {
		saving = true;
		const opts = { onSuccess: () => { showForm = false; }, onFinish: () => { saving = false; } };
		if (editId) router.put(`/app/koordinator-guru/wa-template/${editId}`, form, opts);
		else router.post("/app/koordinator-guru/wa-template", form, opts);
	}
	function del(id: number) { if (confirm("Hapus template ini?")) router.delete(`/app/koordinator-guru/wa-template/${id}`); }
</script>

<AppLayout {user} group="koordinator-wa">
	<div class="pt-8 pb-10 border-b border-neutral-200/80 dark:border-white/[0.04]">
		<div class="max-w-5xl mx-auto px-4 sm:px-6 flex items-end justify-between gap-4 flex-wrap">
			<div>
				<div class="flex items-center gap-2 text-sm text-neutral-500 mb-3"><a href="/app/koordinator-guru" use:inertia class="hover:text-brand-600">Koordinator Guru</a><span>/</span><span class="text-neutral-700 dark:text-neutral-300">Template WA</span></div>
				<h1 class="text-2xl sm:text-3xl font-bold text-neutral-900 dark:text-white tracking-tight">Template Pesan WA</h1>
				<p class="mt-2 text-neutral-600 dark:text-neutral-400">Template pesan seragam. Variabel: <code class="text-xs">{"{nama}"}</code>, <code class="text-xs">{"{nama_kelas}"}</code>, <code class="text-xs">{"{jadwal}"}</code>, dll.</p>
			</div>
			<button onclick={openCreate} class="inline-flex items-center gap-2 px-4 py-2.5 rounded-xl bg-brand-600 hover:bg-brand-700 text-white text-sm font-semibold"><Plus class="w-4 h-4" /> Tambah Template</button>
		</div>
	</div>

	<div class="max-w-5xl mx-auto px-4 sm:px-6 py-8 space-y-4">
		{#if success}<div class="rounded-xl bg-green-500/10 border border-green-500/20 p-4 text-sm font-medium text-green-700 dark:text-green-400" in:fly={{ y: 10, duration: 200 }}>{success}</div>{/if}
		{#if error}<div class="rounded-xl bg-red-500/10 border border-red-500/20 p-4 text-sm font-medium text-red-700 dark:text-red-400">{error}</div>{/if}

		<div class="grid md:grid-cols-2 gap-4">
			{#each templates as t (t.id)}
				<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-5">
					<div class="flex items-start justify-between gap-3">
						<div class="min-w-0"><h2 class="font-semibold text-neutral-900 dark:text-white truncate">{t.nama}</h2><span class="inline-flex mt-1 px-2 py-0.5 rounded-full text-xs font-medium {t.target_type === 'guru' ? 'bg-indigo-500/10 text-indigo-600 dark:text-indigo-400' : 'bg-teal-500/10 text-teal-700 dark:text-teal-400'}">{t.target_type}</span>{#if !t.is_aktif}<span class="ml-1 inline-flex px-2 py-0.5 rounded-full text-xs bg-neutral-500/10 text-neutral-500">nonaktif</span>{/if}</div>
						<div class="flex items-center gap-1 shrink-0">
							<button onclick={() => openEdit(t)} class="p-2 rounded-lg hover:bg-neutral-100 dark:hover:bg-neutral-800 text-neutral-500"><Pencil class="w-4 h-4" /></button>
							<button onclick={() => del(t.id)} class="p-2 rounded-lg hover:bg-red-500/10 text-red-500"><Trash2 class="w-4 h-4" /></button>
						</div>
					</div>
					<p class="mt-3 text-sm text-neutral-600 dark:text-neutral-400 whitespace-pre-wrap line-clamp-4">{t.body}</p>
				</div>
			{:else}
				<div class="md:col-span-2 rounded-2xl border border-dashed border-neutral-300 dark:border-neutral-700 p-12 text-center text-sm text-neutral-500"><MessageCircle class="w-8 h-8 mx-auto mb-2 opacity-40" /> Belum ada template.</div>
			{/each}
		</div>
	</div>

	{#if showForm}
		<div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50" onclick={() => (showForm = false)}>
			<div class="w-full max-w-lg rounded-2xl bg-white dark:bg-neutral-925 border border-neutral-200 dark:border-white/[0.06] shadow-xl" onclick={(e) => e.stopPropagation()}>
				<div class="flex items-center justify-between px-5 py-4 border-b border-neutral-200/80 dark:border-white/[0.04]">
					<h2 class="text-lg font-bold text-neutral-900 dark:text-white">{editId ? "Edit" : "Tambah"} Template</h2>
					<button onclick={() => (showForm = false)} class="p-1.5 rounded-lg hover:bg-neutral-100 dark:hover:bg-neutral-800 text-neutral-500"><X class="w-5 h-5" /></button>
				</div>
				<form onsubmit={(e) => { e.preventDefault(); submit(); }} class="p-5 space-y-4">
					<div class="grid sm:grid-cols-2 gap-4">
						<label class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Nama<input bind:value={form.nama} required class={`${inputClass} mt-1.5`} /></label>
						<label class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Target<select bind:value={form.target_type} class={`${inputClass} mt-1.5`}><option value="santri">Santri</option><option value="guru">Guru</option></select></label>
					</div>
					<label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300">Isi pesan<textarea bind:value={form.body} rows="5" class={`${inputClass} mt-1.5`}></textarea></label>
					<label class="flex items-center gap-2 text-sm font-medium text-neutral-700 dark:text-neutral-300"><input type="checkbox" bind:checked={form.is_aktif} class="rounded" /> Aktif</label>
					<div class="pt-3 border-t border-neutral-200/80 dark:border-white/[0.04] flex justify-end gap-3">
						<button type="button" onclick={() => (showForm = false)} class="px-4 py-2.5 rounded-xl text-sm font-semibold text-neutral-600 dark:text-neutral-300 hover:bg-neutral-100 dark:hover:bg-neutral-800">Batal</button>
						<button type="submit" disabled={saving} class="px-5 py-2.5 rounded-xl bg-brand-600 hover:bg-brand-700 text-white text-sm font-semibold disabled:opacity-50">{saving ? "Menyimpan..." : "Simpan"}</button>
					</div>
				</form>
			</div>
		</div>
	{/if}
</AppLayout>
