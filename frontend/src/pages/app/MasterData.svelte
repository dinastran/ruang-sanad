<script lang="ts">
	import { router } from "@inertiajs/svelte";
	import { fly } from "svelte/transition";
	import AppLayout from "@layouts/AppLayout.svelte";
	import GenderBadge from "@components/GenderBadge.svelte";
	import type { User } from "@lib/types";
	import { Database, Plus, Trash2, Pencil, Check, X, GraduationCap, CalendarDays, BookOpen, Users, Tag } from "lucide-svelte";

	interface Angkatan { id: number; kode: string; keterangan: string; is_aktif: boolean; }
	interface Level { id: number; kode: string; nama: string; urutan: number; }
	interface Jadwal { id: number; nama: string; }
	interface Guru { id: number; nama: string; jenis_kelamin: string; is_aktif: boolean; }
	interface KodeKelas { id: number; kode: string; tipe: string; frekuensi: string; urutan: number; }

	interface Props {
		user?: User;
		angkatan?: Angkatan[];
		levels?: Level[];
		jadwals?: Jadwal[];
		gurus?: Guru[];
		kode_kelas?: KodeKelas[];
		success?: string;
		error?: string;
	}

	let props: Props = $props();
	let user = $derived(props.user);
	let angkatan = $derived(props.angkatan ?? []);
	let levels = $derived(props.levels ?? []);
	let jadwals = $derived(props.jadwals ?? []);
	let gurus = $derived(props.gurus ?? []);
	let kodeKelas = $derived(props.kode_kelas ?? []);
	let success = $derived(props.success);
	let error = $derived(props.error);

	// --- forms ---
	const HARI = ["Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu", "Ahad"];
	const TIPE_OPTS = ["Reguler", "Private", "Semi Private"];
	const FREKUENSI_OPTS = ["1x/pekan", "2x/pekan", "4x pertemuan", "16x pertemuan"];
	let angkatanForm = $state({ kode: "", keterangan: "" });
	let levelForm = $state({ kode: "", nama: "", urutan: 0 });
	let jadwalForm = $state({ hari: "Senin", jam: "" });
	let kodeKelasForm = $state({ kode: "", tipe: "Reguler", frekuensi: "1x/pekan", urutan: 0 });

	// Combine hari + jam into the stored jadwal name, e.g. "Senin, jam 20.00 WIB".
	function jadwalNama(): string {
		if (!jadwalForm.jam) return jadwalForm.hari;
		return `${jadwalForm.hari}, jam ${jadwalForm.jam.replace(":", ".")} WIB`;
	}

	let loading = $state<string | null>(null);

	function submit(key: string, url: string, body: Record<string, unknown>, reset: () => void) {
		loading = key;
		router.post(url, body, {
			preserveScroll: true,
			onSuccess: () => reset(),
			onFinish: () => { loading = null; },
		});
	}

	function del(url: string, label: string) {
		if (!confirm(`Hapus ${label}?`)) return;
		router.delete(url, { preserveScroll: true });
	}

	// --- inline edit ---
	let editing = $state<{ type: string; id: number } | null>(null);
	let editForm = $state<Record<string, unknown>>({});

	function startEdit(type: string, item: Record<string, unknown>) {
		editing = { type, id: item.id as number };
		editForm = { ...item };
	}

	function isEditing(type: string, id: number): boolean {
		return editing?.type === type && editing?.id === id;
	}

	function saveEdit(type: string) {
		if (!editing) return;
		loading = `edit-${type}`;
		router.put(`/app/master/${type}/${editing.id}`, { ...editForm }, {
			preserveScroll: true,
			onSuccess: () => { editing = null; },
			onFinish: () => { loading = null; },
		});
	}

	const inputCls =
		"w-full px-3.5 py-2.5 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white placeholder-neutral-500 transition-all outline-none text-sm";
</script>

<AppLayout {user}>
	{#snippet editActions(onSave: () => void)}
		<div class="flex items-center gap-2 justify-end sm:shrink-0">
			<button onclick={onSave}
				class="inline-flex items-center gap-1.5 px-3.5 py-2 rounded-lg bg-green-600 hover:bg-green-700 text-white text-xs font-semibold transition-colors">
				<Check class="w-4 h-4" /> Simpan
			</button>
			<button onclick={() => (editing = null)} aria-label="Batal"
				class="inline-flex items-center justify-center p-2 rounded-lg border border-neutral-300 dark:border-neutral-700 text-neutral-500 hover:bg-neutral-100 dark:hover:bg-neutral-800 transition-colors">
				<X class="w-4 h-4" />
			</button>
		</div>
	{/snippet}

	{#snippet rowActions(onEdit: () => void, onDelete: () => void)}
		<div class="ml-auto flex items-center gap-0.5 shrink-0">
			<button onclick={onEdit} aria-label="Edit"
				class="p-2 rounded-lg text-neutral-400 hover:text-brand-500 hover:bg-brand-500/10 transition-colors"><Pencil class="w-4 h-4" /></button>
			<button onclick={onDelete} aria-label="Hapus"
				class="p-2 rounded-lg text-neutral-400 hover:text-red-500 hover:bg-red-500/10 transition-colors"><Trash2 class="w-4 h-4" /></button>
		</div>
	{/snippet}

	<div class="pt-6 sm:pt-8 pb-6 sm:pb-8 border-b border-neutral-200/80 dark:border-white/[0.04]">
		<div class="max-w-6xl mx-auto px-4 sm:px-6">
			<div class="flex items-center gap-3">
				<div class="w-10 h-10 rounded-xl bg-brand-400/10 flex items-center justify-center shrink-0">
					<Database class="w-5 h-5 text-brand-600 dark:text-brand-400" />
				</div>
				<div>
					<h1 class="text-2xl sm:text-3xl font-bold text-neutral-900 dark:text-white tracking-tight">Data Master</h1>
					<p class="text-sm text-neutral-600 dark:text-neutral-400">Kelola angkatan, level, jadwal, dan guru</p>
				</div>
			</div>
		</div>
	</div>

	<div class="max-w-6xl mx-auto px-4 sm:px-6 py-6 space-y-5">
		{#if success}
			<div class="bg-green-500/10 border border-green-500/20 text-green-700 dark:text-green-400 rounded-2xl p-4 text-sm font-medium" in:fly={{ y: 20, duration: 300 }}>{success}</div>
		{/if}
		{#if error}
			<div class="bg-red-500/10 border border-red-500/20 text-red-600 dark:text-red-400 rounded-2xl p-4 text-sm font-medium" in:fly={{ y: 20, duration: 300 }}>{error}</div>
		{/if}

		<div class="grid gap-5 lg:grid-cols-2">
			<!-- Angkatan -->
			<section class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 overflow-hidden">
				<div class="flex items-center gap-2.5 px-5 py-4 border-b border-neutral-200/80 dark:border-white/[0.04]">
					<GraduationCap class="w-5 h-5 text-brand-500 shrink-0" />
					<h2 class="font-semibold text-neutral-900 dark:text-white">Angkatan</h2>
					<span class="ml-auto text-xs font-mono text-neutral-500">{angkatan.length}</span>
				</div>
				<form onsubmit={(e) => { e.preventDefault(); submit("angkatan", "/app/master/angkatan", { ...angkatanForm }, () => angkatanForm = { kode: "", keterangan: "" }); }}
					class="p-4 flex flex-col sm:flex-row gap-2.5 border-b border-neutral-200/80 dark:border-white/[0.04]">
					<input bind:value={angkatanForm.kode} placeholder="Kode (mis. 2024)" class={inputCls} required />
					<input bind:value={angkatanForm.keterangan} placeholder="Keterangan" class={inputCls} />
					<button type="submit" disabled={loading === "angkatan"}
						class="shrink-0 inline-flex items-center justify-center gap-1.5 px-4 py-2.5 rounded-xl bg-brand-600 hover:bg-brand-700 dark:bg-brand-500 dark:hover:bg-brand-400 text-white text-sm font-semibold transition-all disabled:opacity-50">
						<Plus class="w-4 h-4" /> Tambah
					</button>
				</form>
				<ul class="divide-y divide-neutral-200/80 dark:divide-white/[0.04] max-h-72 overflow-y-auto">
					{#each angkatan as a (a.id)}
						<li class="px-4 sm:px-5 py-3 {isEditing('angkatan', a.id) ? 'bg-brand-500/[0.04]' : ''}">
							{#if isEditing("angkatan", a.id)}
								<div class="flex flex-col sm:flex-row sm:items-center gap-2.5">
									<span class="inline-flex items-center px-2 py-1 rounded-md bg-neutral-200/70 dark:bg-neutral-800 font-mono text-xs font-semibold text-neutral-700 dark:text-neutral-300 shrink-0 w-fit">{a.kode}</span>
									<input bind:value={editForm.keterangan} placeholder="Keterangan" class="{inputCls} flex-1" />
									{@render editActions(() => saveEdit("angkatan"))}
								</div>
							{:else}
								<div class="flex items-center gap-3">
									<span class="font-mono text-sm font-medium text-neutral-900 dark:text-white shrink-0">{a.kode}</span>
									<span class="text-sm text-neutral-500 dark:text-neutral-400 truncate">{a.keterangan}</span>
									{@render rowActions(() => startEdit("angkatan", a), () => del(`/app/master/angkatan/${a.id}`, `angkatan ${a.kode}`))}
								</div>
							{/if}
						</li>
					{:else}
						<li class="px-5 py-8 text-center text-sm text-neutral-500">Belum ada angkatan</li>
					{/each}
				</ul>
			</section>

			<!-- Level -->
			<section class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 overflow-hidden">
				<div class="flex items-center gap-2.5 px-5 py-4 border-b border-neutral-200/80 dark:border-white/[0.04]">
					<BookOpen class="w-5 h-5 text-secondary-500 shrink-0" />
					<h2 class="font-semibold text-neutral-900 dark:text-white">Level</h2>
					<span class="ml-auto text-xs font-mono text-neutral-500">{levels.length}</span>
				</div>
				<form onsubmit={(e) => { e.preventDefault(); submit("level", "/app/master/level", { ...levelForm }, () => levelForm = { kode: "", nama: "", urutan: 0 }); }}
					class="p-4 flex flex-col sm:flex-row gap-2.5 border-b border-neutral-200/80 dark:border-white/[0.04]">
					<input bind:value={levelForm.kode} placeholder="Kode" class={inputCls} required />
					<input bind:value={levelForm.nama} placeholder="Nama" class={inputCls} />
					<input type="number" bind:value={levelForm.urutan} placeholder="Urutan" class="{inputCls} sm:w-24" />
					<button type="submit" disabled={loading === "level"}
						class="shrink-0 inline-flex items-center justify-center gap-1.5 px-4 py-2.5 rounded-xl bg-secondary-500 hover:bg-secondary-600 text-white text-sm font-semibold transition-all disabled:opacity-50">
						<Plus class="w-4 h-4" /> Tambah
					</button>
				</form>
				<ul class="divide-y divide-neutral-200/80 dark:divide-white/[0.04] max-h-72 overflow-y-auto">
					{#each levels as l (l.id)}
						<li class="px-4 sm:px-5 py-3 {isEditing('level', l.id) ? 'bg-brand-500/[0.04]' : ''}">
							{#if isEditing("level", l.id)}
								<div class="flex flex-col gap-2.5">
									<div class="flex items-center gap-2.5">
										<span class="inline-flex items-center px-2 py-1 rounded-md bg-neutral-200/70 dark:bg-neutral-800 font-mono text-xs font-semibold text-neutral-700 dark:text-neutral-300 shrink-0 w-fit">{l.kode}</span>
										<input bind:value={editForm.nama} placeholder="Nama" class="{inputCls} flex-1 min-w-0" />
									</div>
									<div class="flex items-center gap-2.5">
										<label class="text-xs text-neutral-500 shrink-0">Urutan</label>
										<input type="number" bind:value={editForm.urutan} class="{inputCls} w-20 shrink-0 text-center px-2" />
										<div class="ml-auto">{@render editActions(() => saveEdit("level"))}</div>
									</div>
								</div>
							{:else}
								<div class="flex items-center gap-3">
									<span class="font-mono text-sm font-medium text-neutral-900 dark:text-white shrink-0">{l.kode}</span>
									<span class="text-sm text-neutral-500 dark:text-neutral-400 truncate">{l.nama}</span>
									<span class="text-xs text-neutral-400 font-mono shrink-0">#{l.urutan}</span>
									{@render rowActions(() => startEdit("level", l), () => del(`/app/master/level/${l.id}`, `level ${l.kode}`))}
								</div>
							{/if}
						</li>
					{:else}
						<li class="px-5 py-8 text-center text-sm text-neutral-500">Belum ada level</li>
					{/each}
				</ul>
			</section>

			<!-- Kode Kelas -->
			<section class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 overflow-hidden">
				<div class="flex items-center gap-2.5 px-5 py-4 border-b border-neutral-200/80 dark:border-white/[0.04]">
					<Tag class="w-5 h-5 text-amber-500 shrink-0" />
					<h2 class="font-semibold text-neutral-900 dark:text-white">Kode Kelas</h2>
					<span class="ml-auto text-xs font-mono text-neutral-500">{kodeKelas.length}</span>
				</div>
				<form onsubmit={(e) => { e.preventDefault(); submit("kode-kelas", "/app/master/kode-kelas", { ...kodeKelasForm }, () => kodeKelasForm = { kode: "", tipe: "Reguler", frekuensi: "1x/pekan", urutan: 0 }); }}
					class="p-4 flex flex-col gap-2.5 border-b border-neutral-200/80 dark:border-white/[0.04]">
					<div class="flex flex-col sm:flex-row gap-2.5">
						<input bind:value={kodeKelasForm.kode} placeholder="Kode (mis. P 2X)" class="{inputCls} sm:w-32" required />
						<select bind:value={kodeKelasForm.tipe} class="{inputCls} flex-1">
							{#each TIPE_OPTS as t}<option value={t}>{t}</option>{/each}
						</select>
						<select bind:value={kodeKelasForm.frekuensi} class="{inputCls} flex-1">
							{#each FREKUENSI_OPTS as f}<option value={f}>{f}</option>{/each}
						</select>
					</div>
					<div class="flex gap-2.5">
						<input type="number" bind:value={kodeKelasForm.urutan} placeholder="Urutan" class="{inputCls} w-24 text-center" />
						<button type="submit" disabled={loading === "kode-kelas"}
							class="ml-auto shrink-0 inline-flex items-center justify-center gap-1.5 px-4 py-2.5 rounded-xl bg-amber-500 hover:bg-amber-600 text-white text-sm font-semibold transition-all disabled:opacity-50">
							<Plus class="w-4 h-4" /> Tambah
						</button>
					</div>
				</form>
				<ul class="divide-y divide-neutral-200/80 dark:divide-white/[0.04] max-h-72 overflow-y-auto">
					{#each kodeKelas as k (k.id)}
						<li class="px-4 sm:px-5 py-3 {isEditing('kode-kelas', k.id) ? 'bg-brand-500/[0.04]' : ''}">
							{#if isEditing("kode-kelas", k.id)}
								<div class="flex flex-col gap-2.5">
									<div class="flex items-center gap-2.5">
										<span class="inline-flex items-center px-2 py-1 rounded-md bg-neutral-200/70 dark:bg-neutral-800 font-mono text-xs font-semibold text-neutral-700 dark:text-neutral-300 shrink-0 w-fit">{k.kode}</span>
										<select bind:value={editForm.tipe} class="{inputCls} flex-1 min-w-0">
											{#each TIPE_OPTS as t}<option value={t}>{t}</option>{/each}
										</select>
									</div>
									<div class="flex items-center gap-2.5">
										<select bind:value={editForm.frekuensi} class="{inputCls} flex-1 min-w-0">
											{#each FREKUENSI_OPTS as f}<option value={f}>{f}</option>{/each}
										</select>
										<input type="number" bind:value={editForm.urutan} class="{inputCls} w-16 shrink-0 text-center px-2" />
										<div class="ml-auto">{@render editActions(() => saveEdit("kode-kelas"))}</div>
									</div>
								</div>
							{:else}
								<div class="flex items-center gap-3">
									<span class="font-mono text-sm font-medium text-neutral-900 dark:text-white shrink-0">{k.kode}</span>
									<span class="text-sm text-neutral-500 dark:text-neutral-400 truncate">{k.tipe} · {k.frekuensi}</span>
									<span class="text-xs text-neutral-400 font-mono shrink-0">#{k.urutan}</span>
									{@render rowActions(() => startEdit("kode-kelas", k), () => del(`/app/master/kode-kelas/${k.id}`, `kode kelas ${k.kode}`))}
								</div>
							{/if}
						</li>
					{:else}
						<li class="px-5 py-8 text-center text-sm text-neutral-500">Belum ada kode kelas</li>
					{/each}
				</ul>
			</section>

			<!-- Jadwal -->
			<section class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 overflow-hidden">
				<div class="flex items-center gap-2.5 px-5 py-4 border-b border-neutral-200/80 dark:border-white/[0.04]">
					<CalendarDays class="w-5 h-5 text-blue-500 shrink-0" />
					<h2 class="font-semibold text-neutral-900 dark:text-white">Jadwal</h2>
					<span class="ml-auto text-xs font-mono text-neutral-500">{jadwals.length}</span>
				</div>
				<form onsubmit={(e) => { e.preventDefault(); submit("jadwal", "/app/master/jadwal", { nama: jadwalNama() }, () => jadwalForm = { hari: "Senin", jam: "" }); }}
					class="p-4 flex flex-col sm:flex-row gap-2.5 border-b border-neutral-200/80 dark:border-white/[0.04]">
					<select bind:value={jadwalForm.hari} class="{inputCls} sm:w-32">
						{#each HARI as h}<option value={h}>{h}</option>{/each}
					</select>
					<input type="time" bind:value={jadwalForm.jam} class="{inputCls} sm:w-32" required />
					<button type="submit" disabled={loading === "jadwal"}
						class="shrink-0 inline-flex items-center justify-center gap-1.5 px-4 py-2.5 rounded-xl bg-blue-600 hover:bg-blue-700 text-white text-sm font-semibold transition-all disabled:opacity-50">
						<Plus class="w-4 h-4" /> Tambah
					</button>
				</form>
				<ul class="divide-y divide-neutral-200/80 dark:divide-white/[0.04] max-h-72 overflow-y-auto">
					{#each jadwals as j (j.id)}
						<li class="px-4 sm:px-5 py-3 {isEditing('jadwal', j.id) ? 'bg-brand-500/[0.04]' : ''}">
							{#if isEditing("jadwal", j.id)}
								<div class="flex flex-col sm:flex-row sm:items-center gap-2.5">
									<input bind:value={editForm.nama} placeholder="mis. Senin, jam 20.00 WIB" class="{inputCls} flex-1" />
									{@render editActions(() => saveEdit("jadwal"))}
								</div>
							{:else}
								<div class="flex items-center gap-3">
									<span class="text-sm font-medium text-neutral-900 dark:text-white truncate">{j.nama}</span>
									{@render rowActions(() => startEdit("jadwal", j), () => del(`/app/master/jadwal/${j.id}`, `jadwal ${j.nama}`))}
								</div>
							{/if}
						</li>
					{:else}
						<li class="px-5 py-8 text-center text-sm text-neutral-500">Belum ada jadwal</li>
					{/each}
				</ul>
			</section>

			<!-- Guru -->
			<section class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 overflow-hidden">
				<div class="flex items-center gap-2.5 px-5 py-4 border-b border-neutral-200/80 dark:border-white/[0.04]">
					<Users class="w-5 h-5 text-green-500 shrink-0" />
					<h2 class="font-semibold text-neutral-900 dark:text-white">Guru</h2>
					<span class="ml-auto text-xs font-mono text-neutral-500">{gurus.length}</span>
				</div>
				<ul class="divide-y divide-neutral-200/80 dark:divide-white/[0.04] max-h-72 overflow-y-auto">
					{#each gurus as g (g.id)}
						<li class="px-4 sm:px-5 py-3">
							<div class="flex items-center gap-3">
								<span class="text-sm font-medium text-neutral-900 dark:text-white truncate">{g.nama}</span>
								<GenderBadge gender={g.jenis_kelamin} />
								<span class="ml-auto text-xs text-neutral-500">Dikelola Koordinator Guru</span>
							</div>
						</li>
					{:else}
						<li class="px-5 py-8 text-center text-sm text-neutral-500">Belum ada guru</li>
					{/each}
				</ul>
			</section>
		</div>
	</div>
</AppLayout>
