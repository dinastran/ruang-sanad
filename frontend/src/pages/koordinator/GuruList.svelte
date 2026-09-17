<script lang="ts">
	import { inertia, router } from "@inertiajs/svelte";
	import { fly } from "svelte/transition";
	import AppLayout from "@layouts/AppLayout.svelte";
	import type { GuruDirectory, User } from "@lib/types";
	import { Search, Users, ArrowRight, BookOpen, UserRoundCheck, Plus, X } from "lucide-svelte";

	interface Props {
		user?: User;
		gurus?: GuruDirectory[];
		success?: string;
		error?: string;
	}

	let { user, gurus = [], success, error }: Props = $props();
	let search = $state("");
	let status = $state("semua");
	let filtered = $derived(gurus.filter((guru) => {
		const matchesSearch = `${guru.nama} ${guru.gelar} ${guru.email}`.toLowerCase().includes(search.toLowerCase());
		const matchesStatus = status === "semua" || (status === "aktif" ? guru.is_aktif : !guru.is_aktif);
		return matchesSearch && matchesStatus;
	}));

	let showCreate = $state(false);
	let saving = $state(false);
	const emptyForm = () => ({ nama: "", gelar: "", jenis_kelamin: "L", status: "tetap", no_wa: "", email: "", tanggal_gabung: "", foto: "", is_aktif: true });
	let form = $state(emptyForm());
	const inputClass = "w-full px-3.5 py-2.5 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700 text-sm text-neutral-900 dark:text-white outline-none focus:border-brand-400";

	function openCreate() { form = emptyForm(); showCreate = true; }
	function createGuru() {
		saving = true;
		router.post("/app/koordinator-guru/guru", form, {
			onSuccess: () => { showCreate = false; },
			onFinish: () => { saving = false; },
		});
	}
</script>

<AppLayout {user} group="koordinator-guru">
	<div class="pt-8 pb-10 border-b border-neutral-200/80 dark:border-white/[0.04]">
		<div class="max-w-6xl mx-auto px-4 sm:px-6">
			<div class="flex items-center gap-2 text-sm text-neutral-500 dark:text-neutral-400 mb-4">
				<a href="/app/koordinator-guru" use:inertia class="hover:text-brand-600 dark:hover:text-brand-400 transition-colors">Koordinator Guru</a>
				<span>/</span><span class="text-neutral-700 dark:text-neutral-300">Data Guru</span>
			</div>
			<div class="flex items-end justify-between gap-4 flex-wrap">
				<div>
					<h1 class="text-2xl sm:text-3xl font-bold text-neutral-900 dark:text-white tracking-tight">Data Guru</h1>
					<p class="mt-2 text-neutral-600 dark:text-neutral-400">Profil, status penugasan, dan beban mengajar seluruh guru.</p>
				</div>
				<div class="flex items-center gap-3">
					<div class="inline-flex items-center gap-2 px-3 py-2 rounded-xl bg-brand-400/10 text-brand-700 dark:text-brand-300 text-sm font-semibold">
						<Users class="w-4 h-4" /> {gurus.length} guru
					</div>
					<button onclick={openCreate} class="inline-flex items-center gap-2 px-4 py-2 rounded-xl bg-brand-600 hover:bg-brand-700 text-white text-sm font-semibold">
						<Plus class="w-4 h-4" /> Tambah Guru
					</button>
				</div>
			</div>
		</div>
	</div>

	<div class="max-w-6xl mx-auto px-4 sm:px-6 py-8 space-y-6">
		{#if success}<div class="rounded-xl bg-green-500/10 border border-green-500/20 p-4 text-sm font-medium text-green-700 dark:text-green-400">{success}</div>{/if}
		{#if error}<div class="rounded-xl bg-red-500/10 border border-red-500/20 p-4 text-sm font-medium text-red-700 dark:text-red-400">{error}</div>{/if}

		<div class="flex flex-col sm:flex-row gap-3">
			<label class="relative flex-1">
				<Search class="absolute left-3.5 top-1/2 -translate-y-1/2 w-4 h-4 text-neutral-400" />
				<input bind:value={search} placeholder="Cari nama, gelar, atau email..." class="w-full pl-10 pr-4 py-2.5 rounded-xl bg-white dark:bg-neutral-925 border border-neutral-200 dark:border-neutral-700 text-sm text-neutral-900 dark:text-white outline-none focus:border-brand-400" />
			</label>
			<select bind:value={status} class="px-3.5 py-2.5 rounded-xl bg-white dark:bg-neutral-925 border border-neutral-200 dark:border-neutral-700 text-sm text-neutral-700 dark:text-neutral-300 outline-none focus:border-brand-400">
				<option value="semua">Semua status</option><option value="aktif">Aktif</option><option value="nonaktif">Nonaktif</option>
			</select>
		</div>

		<div class="grid md:grid-cols-2 xl:grid-cols-3 gap-4">
			{#each filtered as guru (guru.id)}
				<a href={`/app/koordinator-guru/guru/${guru.id}`} use:inertia class="group rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-5 hover:border-brand-400/40 hover:shadow-soft transition-all">
					<div class="flex items-start gap-3">
						{#if guru.foto}<img src={guru.foto} alt={guru.nama} class="w-11 h-11 rounded-xl object-cover shrink-0" />{:else}<div class="w-11 h-11 rounded-xl bg-secondary-500/10 text-secondary-600 dark:text-secondary-300 flex items-center justify-center font-bold shrink-0">{guru.nama.charAt(0).toUpperCase()}</div>{/if}
						<div class="min-w-0 flex-1"><h2 class="font-semibold text-neutral-900 dark:text-white truncate">{guru.nama}{guru.gelar ? `, ${guru.gelar}` : ""}</h2><p class="mt-0.5 text-xs text-neutral-500 dark:text-neutral-400">{guru.status === "tetap" ? "Guru Tetap" : "Guru Part Time"}</p></div>
						<span class="w-2.5 h-2.5 rounded-full {guru.is_aktif ? 'bg-green-500' : 'bg-neutral-400'}" title={guru.is_aktif ? "Aktif" : "Nonaktif"}></span>
					</div>
					<div class="mt-5 pt-4 border-t border-neutral-200/80 dark:border-white/[0.04] flex items-center gap-4 text-xs text-neutral-500 dark:text-neutral-400"><span class="inline-flex items-center gap-1"><BookOpen class="w-3.5 h-3.5" /> {guru.total_kelas} kelas</span><span class="inline-flex items-center gap-1"><UserRoundCheck class="w-3.5 h-3.5" /> {guru.total_santri} santri</span><ArrowRight class="ml-auto w-4 h-4 group-hover:text-brand-500 group-hover:translate-x-0.5 transition-all" /></div>
				</a>
			{:else}
				<div class="md:col-span-2 xl:col-span-3 rounded-2xl border border-dashed border-neutral-300 dark:border-neutral-700 p-12 text-center text-sm text-neutral-500">Tidak ada guru yang sesuai.</div>
			{/each}
		</div>
	</div>

	{#if showCreate}
		<div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50" onclick={() => (showCreate = false)} transition:fly={{ duration: 150 }}>
			<div class="w-full max-w-2xl max-h-[90vh] overflow-y-auto rounded-2xl bg-white dark:bg-neutral-925 border border-neutral-200 dark:border-white/[0.06] shadow-xl" onclick={(e) => e.stopPropagation()}>
				<div class="flex items-center justify-between px-5 sm:px-6 py-4 border-b border-neutral-200/80 dark:border-white/[0.04]">
					<h2 class="text-lg font-bold text-neutral-900 dark:text-white">Tambah Guru Baru</h2>
					<button onclick={() => (showCreate = false)} class="p-1.5 rounded-lg hover:bg-neutral-100 dark:hover:bg-neutral-800 text-neutral-500"><X class="w-5 h-5" /></button>
				</div>
				<form onsubmit={(event) => { event.preventDefault(); createGuru(); }} class="p-5 sm:p-6 space-y-5">
					<p class="text-sm text-neutral-600 dark:text-neutral-400">Buat data master guru terlebih dahulu. Setelah tersimpan, hubungkan ke akun login dan Super Admin menetapkan role Guru.</p>
					<div class="grid sm:grid-cols-2 gap-4">
						<label class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Nama<input bind:value={form.nama} required class={`${inputClass} mt-1.5`} /></label>
						<label class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Gelar<input bind:value={form.gelar} placeholder="S.Pd., Lc., dll." class={`${inputClass} mt-1.5`} /></label>
						<label class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Jenis kelamin<select bind:value={form.jenis_kelamin} class={`${inputClass} mt-1.5`}><option value="L">Laki-laki</option><option value="P">Perempuan</option></select></label>
						<label class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Status<select bind:value={form.status} class={`${inputClass} mt-1.5`}><option value="tetap">Guru Tetap</option><option value="part_time">Guru Part Time</option></select></label>
						<label class="text-sm font-medium text-neutral-700 dark:text-neutral-300">No. WhatsApp<input bind:value={form.no_wa} class={`${inputClass} mt-1.5`} /></label>
						<label class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Email<input bind:value={form.email} type="email" class={`${inputClass} mt-1.5`} /></label>
						<label class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Tanggal bergabung<input bind:value={form.tanggal_gabung} type="date" class={`${inputClass} mt-1.5`} /></label>
						<label class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Status akun<select bind:value={form.is_aktif} class={`${inputClass} mt-1.5`}><option value={true}>Aktif</option><option value={false}>Nonaktif</option></select></label>
					</div>
					<label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300">URL foto<input bind:value={form.foto} type="url" placeholder="https://..." class={`${inputClass} mt-1.5`} /></label>
					<div class="pt-4 border-t border-neutral-200/80 dark:border-white/[0.04] flex justify-end gap-3">
						<button type="button" onclick={() => (showCreate = false)} class="px-4 py-2.5 rounded-xl text-sm font-semibold text-neutral-600 dark:text-neutral-300 hover:bg-neutral-100 dark:hover:bg-neutral-800">Batal</button>
						<button type="submit" disabled={saving} class="inline-flex items-center gap-2 px-5 py-2.5 rounded-xl bg-brand-600 hover:bg-brand-700 text-white text-sm font-semibold disabled:opacity-50"><Plus class="w-4 h-4" /> {saving ? "Menyimpan..." : "Simpan"}</button>
					</div>
				</form>
			</div>
		</div>
	{/if}
</AppLayout>
