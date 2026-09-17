<script lang="ts">
	import { inertia, router } from "@inertiajs/svelte";
	import { fly } from "svelte/transition";
	import AppLayout from "@layouts/AppLayout.svelte";
	import type { Todo, User } from "@lib/types";
	import { Plus, X, Pencil, Trash2, ClipboardList, ExternalLink, Repeat } from "lucide-svelte";

	interface Props { user?: User; todos?: Todo[]; success?: string; error?: string; }
	let { user, todos = [], success, error }: Props = $props();

	const inputClass = "w-full px-3.5 py-2.5 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700 text-sm text-neutral-900 dark:text-white outline-none focus:border-brand-400";
	const STATUS = ["belum", "proses", "selesai", "batal"];
	const STATUS_LABEL: Record<string, string> = { belum: "Belum", proses: "Proses", selesai: "Selesai", batal: "Batal" };
	const STATUS_COLOR: Record<string, string> = { belum: "bg-neutral-500/10 text-neutral-600 dark:text-neutral-400", proses: "bg-amber-500/10 text-amber-700 dark:text-amber-400", selesai: "bg-green-500/10 text-green-700 dark:text-green-400", batal: "bg-red-500/10 text-red-600 dark:text-red-400" };
	const RECUR_LABEL: Record<string, string> = { none: "", harian: "Harian", mingguan: "Mingguan", bulanan: "Bulanan", "4bulanan": "4 Bulanan" };

	let showForm = $state(false);
	let editId = $state<number | null>(null);
	let saving = $state(false);
	const empty = () => ({ judul: "", teknis: "", kebutuhan: "", deadline: "", pic: "", status: "belum", recurring: "none", link_pendukung: "", catatan: "" });
	let form = $state(empty());

	function openCreate() { editId = null; form = empty(); showForm = true; }
	function openEdit(t: Todo) { editId = t.id; form = { judul: t.judul, teknis: t.teknis, kebutuhan: t.kebutuhan, deadline: t.deadline, pic: t.pic, status: t.status, recurring: t.recurring, link_pendukung: t.link_pendukung, catatan: t.catatan }; showForm = true; }
	function submit() {
		saving = true;
		const opts = { onSuccess: () => { showForm = false; }, onFinish: () => { saving = false; } };
		if (editId) router.put(`/app/koordinator-guru/todo/${editId}`, form, opts);
		else router.post("/app/koordinator-guru/todo", form, opts);
	}
	function setStatus(t: Todo, status: string) { router.put(`/app/koordinator-guru/todo/${t.id}/status`, { status }, { preserveScroll: true }); }
	function del(id: number) { if (confirm("Hapus todo ini?")) router.delete(`/app/koordinator-guru/todo/${id}`); }
</script>

<AppLayout {user} group="koordinator-todo">
	<div class="pt-8 pb-10 border-b border-neutral-200/80 dark:border-white/[0.04]">
		<div class="max-w-5xl mx-auto px-4 sm:px-6 flex items-end justify-between gap-4 flex-wrap">
			<div>
				<div class="flex items-center gap-2 text-sm text-neutral-500 mb-3"><a href="/app/koordinator-guru" use:inertia class="hover:text-brand-600">Koordinator Guru</a><span>/</span><span class="text-neutral-700 dark:text-neutral-300">Todo</span></div>
				<h1 class="text-2xl sm:text-3xl font-bold text-neutral-900 dark:text-white tracking-tight">Todo Koordinator</h1>
				<p class="mt-2 text-neutral-600 dark:text-neutral-400">Rencana & eksekusi kegiatan koordinator, dengan status dan penanda berulang.</p>
			</div>
			<button onclick={openCreate} class="inline-flex items-center gap-2 px-4 py-2.5 rounded-xl bg-brand-600 hover:bg-brand-700 text-white text-sm font-semibold"><Plus class="w-4 h-4" /> Tambah Todo</button>
		</div>
	</div>

	<div class="max-w-5xl mx-auto px-4 sm:px-6 py-8 space-y-3">
		{#if success}<div class="rounded-xl bg-green-500/10 border border-green-500/20 p-4 text-sm font-medium text-green-700 dark:text-green-400" in:fly={{ y: 10, duration: 200 }}>{success}</div>{/if}
		{#if error}<div class="rounded-xl bg-red-500/10 border border-red-500/20 p-4 text-sm font-medium text-red-700 dark:text-red-400">{error}</div>{/if}

		{#each todos as t (t.id)}
			<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-5">
				<div class="flex items-start justify-between gap-4 flex-wrap">
					<div class="min-w-0 flex-1">
						<div class="flex items-center gap-2 flex-wrap">
							<h2 class="font-semibold text-neutral-900 dark:text-white {t.status === 'selesai' ? 'line-through text-neutral-500' : ''}">{t.judul}</h2>
							{#if t.recurring !== "none"}<span class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs bg-brand-400/10 text-brand-700 dark:text-brand-300"><Repeat class="w-3 h-3" /> {RECUR_LABEL[t.recurring]}</span>{/if}
							{#if t.deadline}<span class="text-xs font-mono text-neutral-500">⏰ {t.deadline}</span>{/if}
							{#if t.pic}<span class="text-xs text-neutral-500">PIC: {t.pic}</span>{/if}
						</div>
						{#if t.teknis}<p class="mt-1 text-sm text-neutral-600 dark:text-neutral-400">{t.teknis}</p>{/if}
						{#if t.kebutuhan}<p class="mt-0.5 text-xs text-neutral-500">Kebutuhan: {t.kebutuhan}</p>{/if}
						{#if t.catatan}<p class="mt-1 text-sm text-neutral-600 dark:text-neutral-400 italic">"{t.catatan}"</p>{/if}
						{#if t.link_pendukung}<a href={t.link_pendukung} target="_blank" rel="noopener" class="mt-1 inline-flex items-center gap-1 text-xs text-brand-600 hover:underline"><ExternalLink class="w-3 h-3" /> Link pendukung</a>{/if}
					</div>
					<div class="flex items-center gap-2 shrink-0">
						<select value={t.status} onchange={(e) => setStatus(t, (e.target as HTMLSelectElement).value)} class="px-2.5 py-1.5 rounded-lg text-xs font-medium border border-neutral-200/80 dark:border-white/[0.06] {STATUS_COLOR[t.status]} outline-none">
							{#each STATUS as s}<option value={s}>{STATUS_LABEL[s]}</option>{/each}
						</select>
						<button onclick={() => openEdit(t)} class="p-2 rounded-lg hover:bg-neutral-100 dark:hover:bg-neutral-800 text-neutral-500"><Pencil class="w-4 h-4" /></button>
						<button onclick={() => del(t.id)} class="p-2 rounded-lg hover:bg-red-500/10 text-red-500"><Trash2 class="w-4 h-4" /></button>
					</div>
				</div>
			</div>
		{:else}
			<div class="rounded-2xl border border-dashed border-neutral-300 dark:border-neutral-700 p-12 text-center text-sm text-neutral-500"><ClipboardList class="w-8 h-8 mx-auto mb-2 opacity-40" /> Belum ada todo.</div>
		{/each}
	</div>

	{#if showForm}
		<div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50" onclick={() => (showForm = false)}>
			<div class="w-full max-w-lg max-h-[90vh] overflow-y-auto rounded-2xl bg-white dark:bg-neutral-925 border border-neutral-200 dark:border-white/[0.06] shadow-xl" onclick={(e) => e.stopPropagation()}>
				<div class="flex items-center justify-between px-5 py-4 border-b border-neutral-200/80 dark:border-white/[0.04]">
					<h2 class="text-lg font-bold text-neutral-900 dark:text-white">{editId ? "Edit" : "Tambah"} Todo</h2>
					<button onclick={() => (showForm = false)} class="p-1.5 rounded-lg hover:bg-neutral-100 dark:hover:bg-neutral-800 text-neutral-500"><X class="w-5 h-5" /></button>
				</div>
				<form onsubmit={(e) => { e.preventDefault(); submit(); }} class="p-5 space-y-4">
					<label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300">Judul / kegiatan<input bind:value={form.judul} required class={`${inputClass} mt-1.5`} /></label>
					<label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300">Teknis pelaksanaan<textarea bind:value={form.teknis} rows="2" class={`${inputClass} mt-1.5`}></textarea></label>
					<div class="grid sm:grid-cols-2 gap-4">
						<label class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Kebutuhan<input bind:value={form.kebutuhan} class={`${inputClass} mt-1.5`} /></label>
						<label class="text-sm font-medium text-neutral-700 dark:text-neutral-300">PIC<input bind:value={form.pic} class={`${inputClass} mt-1.5`} /></label>
						<label class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Deadline<input type="date" bind:value={form.deadline} class={`${inputClass} mt-1.5`} /></label>
						<label class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Status<select bind:value={form.status} class={`${inputClass} mt-1.5`}>{#each STATUS as s}<option value={s}>{STATUS_LABEL[s]}</option>{/each}</select></label>
						<label class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Berulang<select bind:value={form.recurring} class={`${inputClass} mt-1.5`}><option value="none">Tidak</option><option value="harian">Harian</option><option value="mingguan">Mingguan</option><option value="bulanan">Bulanan</option><option value="4bulanan">4 Bulanan</option></select></label>
						<label class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Link pendukung<input bind:value={form.link_pendukung} type="url" placeholder="https://..." class={`${inputClass} mt-1.5`} /></label>
					</div>
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
