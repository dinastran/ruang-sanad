<script lang="ts">
	import { inertia, router } from "@inertiajs/svelte";
	import { fly } from "svelte/transition";
	import AppLayout from "@layouts/AppLayout.svelte";
	import { Toast } from "@lib/notifications/toast";
	import { getCSRFToken } from "@lib/utils/csrf";
	import type { User } from "@lib/types";
	import { Check, ChevronDown, ChevronUp, Save, BookOpen, Calendar, Users, Hash, ClipboardList, AlertCircle, ArrowRight, Plus, Trash2, Search, X, Upload } from "lucide-svelte";

	interface SantriItem {
		id: number;
		id_mahasantri: string;
		nama: string;
		jenis_kelamin: string;
		angkatan: string;
		kelas_kode: string;
		frekuensi: string;
		fu: string;
		tanggal_vn: string;
		hasil_vn: string;
		voice_note_url: string;
		keterangan_vn: string;
		masuk_grup: string;
		mulai_belajar: string;
		jumlah: number;
		level: string;
		jadwal: string;
		guru: string;
		tipe: string;
		is_lengkap: boolean;
		status: string;
	}

	interface MasterItem {
		id: number;
		kode?: string;
		nama?: string;
	}

	interface Props {
		user?: User;
		santri?: SantriItem[];
		levels?: MasterItem[];
		jadwals?: MasterItem[];
		gurus?: MasterItem[];
		success?: string;
		error?: string;
	}

	let { user, santri = [], levels = [], jadwals = [], gurus = [], success, error }: Props = $props();

	let expandedId = $state<number | null>(null);
	let savingId = $state<number | null>(null);
	let search = $state("");

	// All santri that still need completing — used for the counters (unaffected
	// by the search box so the header always shows the true totals).
	let allIncomplete = $derived(santri.filter((s) => !s.is_lengkap));
	let completedCount = $derived(santri.length - allIncomplete.length);

	// The list actually rendered, narrowed by the search query (nama / id / angkatan).
	let incompleteSantri = $derived(
		allIncomplete.filter((s) => {
			const q = search.trim().toLowerCase();
			if (!q) return true;
			return (
				(s.nama || "").toLowerCase().includes(q) ||
				(s.id_mahasantri || "").toLowerCase().includes(q) ||
				(s.angkatan || "").toLowerCase().includes(q)
			);
		}),
	);

	let forms = $state<Record<number, { fu: string; tanggal_vn: string; hasil_vn: string; masuk_grup: string; mulai_belajar: string; jumlah: number; level: string; jadwal: string; guru: string }>>({});

	// Multi-jadwal per santri: a class can meet more than once (2x/pekan, private
	// 4x/16x). Sessions are stored joined by " & " in the single jadwal field.
	let jadwalRowsMap = $state<Record<number, string[]>>({});
	let voiceNoteDescriptions = $state<Record<number, string>>({});
	let voiceNoteFiles = $state<Record<number, File | null>>({});
	let voiceNoteSavingId = $state<number | null>(null);

	// Initialise a santri's form lazily — but ONLY from an event handler, never
	// during render. Mutating $state inside the template ({@const ...}) throws
	// Svelte's state_unsafe_mutation and breaks the accordion.
	function ensureForm(s: SantriItem) {
		if (!forms[s.id]) {
			forms[s.id] = {
				fu: s.fu || "",
				tanggal_vn: s.tanggal_vn || "",
				hasil_vn: s.hasil_vn || "",
				masuk_grup: s.masuk_grup || "",
				mulai_belajar: s.mulai_belajar || "",
				jumlah: s.jumlah || 0,
				level: s.level || "",
				jadwal: s.jadwal || "",
				guru: s.guru || "",
			};
			jadwalRowsMap[s.id] = s.jadwal ? s.jadwal.split(" & ").map((x) => x.trim()) : [""];
			voiceNoteDescriptions[s.id] = s.keterangan_vn || "";
		}
	}

	function addJadwalRow(id: number) {
		jadwalRowsMap[id] = [...(jadwalRowsMap[id] || [""]), ""];
	}
	function removeJadwalRow(id: number, i: number) {
		const rows = jadwalRowsMap[id] || [""];
		jadwalRowsMap[id] = rows.length > 1 ? rows.filter((_, idx) => idx !== i) : [""];
	}

	function handleSubmit(santriId: number) {
		savingId = santriId;
		if (forms[santriId]) {
			forms[santriId].jadwal = (jadwalRowsMap[santriId] || []).map((x) => x.trim()).filter(Boolean).join(" & ");
		}
		router.put(`/app/santri/${santriId}/kelas-data`, forms[santriId] || {}, {
			onFinish: () => { savingId = null; },
			onError: () => { savingId = null; },
		});
	}

	function toggleExpand(s: SantriItem) {
		if (expandedId === s.id) {
			expandedId = null;
			return;
		}
		ensureForm(s);
		expandedId = s.id;
	}

	function handleVoiceNoteFile(santriID: number, event: Event) {
		const input = event.currentTarget as HTMLInputElement;
		voiceNoteFiles[santriID] = input.files?.[0] ?? null;
	}

	async function saveVoiceNote(s: SantriItem) {
		const formData = new FormData();
		const file = voiceNoteFiles[s.id];
		if (file) formData.append("file", file);
		formData.append("keterangan_vn", voiceNoteDescriptions[s.id] ?? s.keterangan_vn ?? "");
		voiceNoteSavingId = s.id;

		try {
			const response = await fetch(`/app/santri/${s.id}/voice-note`, {
				method: "POST",
				headers: { "X-XSRF-TOKEN": getCSRFToken() },
				body: formData,
			});
			const data = await response.json();
			if (!response.ok || !data.success) throw new Error(data.error || "Gagal menyimpan VN");
			Toast("Voice note berhasil disimpan", "success");
			voiceNoteFiles[s.id] = null;
			router.reload({ only: ["santri"] });
		} catch (error) {
			Toast(error instanceof Error ? error.message : "Gagal menyimpan VN", "error");
		} finally {
			voiceNoteSavingId = null;
		}
	}

	const pipelineSteps = [
		{ key: "fu", label: "FU", icon: ClipboardList },
		{ key: "tanggal_vn", label: "Tanggal VN", icon: Calendar },
		{ key: "hasil_vn", label: "Hasil VN", icon: ClipboardList },
		{ key: "masuk_grup", label: "Masuk Grup", icon: Users },
		{ key: "mulai_belajar", label: "Mulai Belajar", icon: BookOpen },
	];

	function isStepCompleted(s: SantriItem, stepKey: string): boolean {
		if (stepKey === "jumlah") return s.jumlah > 0;
		if (stepKey === "mulai_belajar") return !!s.mulai_belajar;
		const val = (s as Record<string, unknown>)[stepKey];
		return !!val;
	}
</script>

<AppLayout {user} group="santri">
	<div class="pt-8 pb-10 border-b border-neutral-200/80 dark:border-white/[0.04]">
		<div class="max-w-6xl mx-auto px-4 sm:px-6">
			<div class="flex items-center gap-2 text-sm text-neutral-500 dark:text-neutral-400 mb-4">
				<a href="/app" use:inertia class="hover:text-brand-600 dark:hover:text-brand-400 transition-colors">Dashboard</a>
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
				</svg>
				<span class="text-neutral-700 dark:text-neutral-300">Perlu Dilengkapi</span>
			</div>
			<div class="flex items-start justify-between gap-4 flex-wrap">
				<div>
					<h1 class="text-2xl sm:text-3xl font-bold text-neutral-900 dark:text-white mb-2 tracking-tight">Perlu Dilengkapi</h1>
					<p class="text-neutral-600 dark:text-neutral-400">
						{allIncomplete.length} santri perlu dilengkapi data kelasnya
					</p>
				</div>
				<div class="flex items-center gap-2 px-4 py-2 rounded-xl bg-amber-500/10 border border-amber-500/20 text-amber-700 dark:text-amber-400 text-sm font-medium">
					<AlertCircle class="w-4 h-4" />
					{completedCount}/{santri.length} lengkap
				</div>
			</div>
		</div>
	</div>

	<div class="relative max-w-6xl mx-auto px-4 sm:px-6 py-8 space-y-6">
		{#if success}
			<div class="bg-green-500/10 border border-green-500/20 text-green-700 dark:text-green-400 rounded-2xl p-4 flex items-center gap-3" in:fly={{ y: 20, duration: 300 }}>
				<p class="text-sm font-medium">{success}</p>
			</div>
		{/if}

		{#if error}
			<div class="bg-red-500/10 border border-red-500/20 text-red-600 dark:text-red-400 rounded-2xl p-4 flex items-center gap-3" in:fly={{ y: 20, duration: 300 }}>
				<p class="text-sm font-medium">{error}</p>
			</div>
		{/if}

		{#if allIncomplete.length > 0}
			<div class="relative" in:fly={{ y: 20, duration: 400 }}>
				<div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
					<Search class="w-4 h-4 text-neutral-500" />
				</div>
				<input
					type="text"
					bind:value={search}
					placeholder="Cari nama, ID mahasantri, atau angkatan..."
					class="w-full pl-12 pr-12 py-3 rounded-xl bg-white dark:bg-neutral-925/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white placeholder-neutral-500 transition-all outline-none"
				/>
				{#if search}
					<button type="button" onclick={() => (search = "")} aria-label="Hapus pencarian"
						class="absolute inset-y-0 right-0 pr-4 flex items-center text-neutral-400 hover:text-neutral-600 dark:hover:text-neutral-300">
						<X class="w-4 h-4" />
					</button>
				{/if}
			</div>
		{/if}

		{#if allIncomplete.length === 0}
			<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-12 text-center" in:fly={{ y: 20, duration: 500 }}>
				<div class="w-16 h-16 rounded-2xl bg-green-500/10 flex items-center justify-center mx-auto mb-4">
					<Check class="w-8 h-8 text-green-500" />
				</div>
				<h3 class="text-lg font-semibold text-neutral-900 dark:text-white mb-2">Semua Data Lengkap</h3>
				<p class="text-neutral-500 dark:text-neutral-400">Tidak ada santri yang perlu dilengkapi data kelasnya.</p>
			</div>
		{:else if incompleteSantri.length === 0}
			<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-12 text-center" in:fly={{ y: 20, duration: 500 }}>
				<div class="w-16 h-16 rounded-2xl bg-neutral-200/60 dark:bg-neutral-800 flex items-center justify-center mx-auto mb-4">
					<Search class="w-8 h-8 text-neutral-400" />
				</div>
				<h3 class="text-lg font-semibold text-neutral-900 dark:text-white mb-2">Tidak Ada Hasil</h3>
				<p class="text-neutral-500 dark:text-neutral-400">Tidak ada santri yang cocok dengan "{search}".</p>
			</div>
		{/if}

		{#each incompleteSantri as s, i}
			<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 overflow-hidden transition-all" in:fly={{ y: 20, duration: 500, delay: i * 80 }}>
				<button
					onclick={() => toggleExpand(s)}
					class="w-full flex items-center gap-4 px-6 py-4 hover:bg-neutral-50/50 dark:hover:bg-white/[0.015] transition-colors text-left"
				>
					<div class="w-10 h-10 rounded-xl bg-amber-500/10 flex items-center justify-center shrink-0">
						<AlertCircle class="w-5 h-5 text-amber-500" />
					</div>
					<div class="flex-1 min-w-0">
						<p class="text-sm font-semibold text-neutral-900 dark:text-white truncate">{s.nama}</p>
						<p class="text-xs text-neutral-500 dark:text-neutral-400">
							{s.id_mahasantri || '-'} · {s.angkatan}
						</p>
					</div>
					<div class="flex items-center gap-3">
						{#if expandedId === s.id}
							<ChevronUp class="w-4 h-4 text-neutral-500" />
						{:else}
							<ChevronDown class="w-4 h-4 text-neutral-500" />
						{/if}
					</div>
				</button>

				{#if expandedId === s.id}
					<div class="border-t border-neutral-200/80 dark:border-white/[0.04] px-6 py-6 space-y-6" in:fly={{ y: 10, duration: 300 }}>
						<div class="flex items-center gap-2 flex-wrap">
							{#each pipelineSteps as step, si}
								{@const StepIcon = step.icon}
								{@const done = isStepCompleted(s, step.key)}
								<div class="flex items-center gap-2">
									<div class="flex items-center gap-1.5 px-3 py-1.5 rounded-full text-xs font-medium {done ? 'bg-green-500/10 text-green-700 dark:text-green-400' : 'bg-neutral-200/80 dark:bg-neutral-800 text-neutral-500 dark:text-neutral-400'}">
										{#if done}
											<Check class="w-3 h-3" />
										{:else}
											<StepIcon class="w-3 h-3" />
										{/if}
										{step.label}
									</div>
									{#if si < pipelineSteps.length - 1}
										<ArrowRight class="w-3 h-3 text-neutral-400" />
									{/if}
								</div>
							{/each}
						</div>

						{#if forms[s.id]}
							{@const form = forms[s.id]}
							<form onsubmit={(e) => { e.preventDefault(); handleSubmit(s.id); }} class="space-y-5">
								<div class="grid md:grid-cols-2 gap-5">
									<div>
										<label for="fu-{s.id}" class="block text-sm font-medium text-neutral-700 dark:text-neutral-400 mb-2">FU</label>
										<input
											id="fu-{s.id}"
											type="text"
											bind:value={form.fu}
										class="w-full px-4 py-2.5 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white placeholder-neutral-500 transition-all outline-none"
										placeholder="FU"
									/>
								</div>
								<div>
									<label for="tanggal_vn-{s.id}" class="block text-sm font-medium text-neutral-700 dark:text-neutral-400 mb-2">Tanggal VN</label>
									<input
										id="tanggal_vn-{s.id}"
										type="date"
										bind:value={form.tanggal_vn}
										class="w-full px-4 py-2.5 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white transition-all outline-none"
									/>
								</div>
							</div>

							<div class="grid md:grid-cols-2 gap-5">
								<div>
									<label for="hasil_vn-{s.id}" class="block text-sm font-medium text-neutral-700 dark:text-neutral-400 mb-2">Hasil VN</label>
									<input
										id="hasil_vn-{s.id}"
										type="text"
										bind:value={form.hasil_vn}
										class="w-full px-4 py-2.5 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white placeholder-neutral-500 transition-all outline-none"
										placeholder="Hasil VN"
									/>
								</div>
								<div>
									<label for="masuk_grup-{s.id}" class="block text-sm font-medium text-neutral-700 dark:text-neutral-400 mb-2">Masuk Grup</label>
									<input
										id="masuk_grup-{s.id}"
										type="text"
										bind:value={form.masuk_grup}
										class="w-full px-4 py-2.5 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white placeholder-neutral-500 transition-all outline-none"
										placeholder="Nama grup"
									/>
								</div>
							</div>

							<div class="rounded-xl border border-brand-500/20 bg-brand-500/5 p-4 space-y-4">
								<div>
									<h4 class="text-sm font-semibold text-neutral-900 dark:text-white">Voice Note</h4>
									<p class="text-xs text-neutral-600 dark:text-neutral-400 mt-1">Unggah rekaman VN dan tambahkan keterangannya.</p>
								</div>
								{#if s.voice_note_url}
									<audio controls src={s.voice_note_url} class="w-full h-10">Browser tidak mendukung pemutar audio.</audio>
								{/if}
								<div>
									<label for="keterangan_vn-{s.id}" class="block text-sm font-medium text-neutral-700 dark:text-neutral-400 mb-2">Keterangan VN</label>
									<textarea
										id="keterangan_vn-{s.id}"
										bind:value={voiceNoteDescriptions[s.id]}
										rows="3"
										placeholder="Tulis keterangan voice note"
										class="w-full px-4 py-2.5 rounded-xl bg-white dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white placeholder-neutral-500 transition-all outline-none"
									></textarea>
								</div>
								<div class="flex flex-col sm:flex-row sm:items-center gap-3">
									<label class="inline-flex items-center justify-center gap-2 px-4 py-2.5 rounded-xl border border-neutral-300 dark:border-neutral-700/80 text-sm font-medium text-neutral-700 dark:text-neutral-300 cursor-pointer hover:bg-neutral-100 dark:hover:bg-neutral-800 transition-colors">
										<Upload class="w-4 h-4" />
										<span>{voiceNoteFiles[s.id]?.name || "Pilih file audio"}</span>
										<input type="file" accept="audio/*" class="sr-only" onchange={(event) => handleVoiceNoteFile(s.id, event)} />
									</label>
									<button
										type="button"
										onclick={() => saveVoiceNote(s)}
										disabled={voiceNoteSavingId === s.id}
										class="inline-flex items-center justify-center gap-2 px-4 py-2.5 rounded-xl bg-brand-600 hover:bg-brand-700 text-white text-sm font-semibold disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
									>
										<Save class="w-4 h-4" />
										{voiceNoteSavingId === s.id ? "Menyimpan VN..." : "Simpan VN"}
									</button>
								</div>
							</div>

							<div class="grid md:grid-cols-2 gap-5">
								<div>
									<label for="mulai_belajar-{s.id}" class="block text-sm font-medium text-neutral-700 dark:text-neutral-400 mb-2">Mulai Belajar</label>
									<input
										id="mulai_belajar-{s.id}"
										type="date"
										bind:value={form.mulai_belajar}
										class="w-full px-4 py-2.5 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white transition-all outline-none"
									/>
								</div>
								<div>
									<label for="jumlah-{s.id}" class="block text-sm font-medium text-neutral-700 dark:text-neutral-400 mb-2">Jumlah</label>
									<input
										id="jumlah-{s.id}"
										type="number"
										bind:value={form.jumlah}
										class="w-full px-4 py-2.5 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white placeholder-neutral-500 transition-all outline-none"
										placeholder="0"
									/>
								</div>
							</div>

							<div class="grid md:grid-cols-2 gap-5">
								<div>
									<label for="level-{s.id}" class="block text-sm font-medium text-neutral-700 dark:text-neutral-400 mb-2">Level</label>
									<select
										id="level-{s.id}"
										bind:value={form.level}
										class="w-full px-4 py-2.5 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white transition-all outline-none"
									>
										<option value="">Pilih Level</option>
										{#each levels as l}
											<option value={l.kode || String(l.id)}>{l.nama || l.kode || String(l.id)}</option>
										{/each}
									</select>
								</div>
								<div>
									<div class="flex items-center justify-between mb-2">
										<label class="block text-sm font-medium text-neutral-700 dark:text-neutral-400">
											Jadwal{#if s.frekuensi}<span class="ml-1 text-xs font-normal text-neutral-400">· {s.frekuensi}</span>{/if}
										</label>
										<button type="button" onclick={() => addJadwalRow(s.id)}
											class="inline-flex items-center gap-1 text-xs font-medium text-brand-600 dark:text-brand-400 hover:underline">
											<Plus class="w-3.5 h-3.5" /> Tambah sesi
										</button>
									</div>
									<div class="space-y-2">
										{#each (jadwalRowsMap[s.id] || [""]) as _row, ji}
											<div class="flex items-center gap-2">
												<select
													bind:value={jadwalRowsMap[s.id][ji]}
													class="flex-1 px-4 py-2.5 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white transition-all outline-none"
												>
													<option value="">Pilih Jadwal</option>
													{#each jadwals as j}
														<option value={j.nama || String(j.id)}>{j.nama || String(j.id)}</option>
													{/each}
												</select>
												{#if (jadwalRowsMap[s.id] || []).length > 1}
													<button type="button" onclick={() => removeJadwalRow(s.id, ji)} aria-label="Hapus sesi"
														class="shrink-0 p-2.5 rounded-xl text-neutral-400 hover:text-red-500 hover:bg-red-500/10 transition-colors">
														<Trash2 class="w-4 h-4" />
													</button>
												{/if}
											</div>
										{/each}
									</div>
								</div>
							</div>

							<div>
								<label for="guru-{s.id}" class="block text-sm font-medium text-neutral-700 dark:text-neutral-400 mb-2">Guru</label>
								<select
									id="guru-{s.id}"
									bind:value={form.guru}
									class="w-full px-4 py-2.5 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white transition-all outline-none"
								>
									<option value="">Pilih Guru</option>
									{#each gurus as g}
										<option value={g.nama || String(g.id)}>{g.nama || String(g.id)}</option>
									{/each}
								</select>
							</div>

							<div class="pt-2">
								<button
									type="submit"
									disabled={savingId === s.id}
									class="inline-flex items-center gap-2 px-6 py-2.5 rounded-xl bg-brand-600 hover:bg-brand-700 text-white font-semibold transition-all dark:bg-brand-500 dark:hover:bg-brand-400 shadow-lg shadow-brand-600/25 disabled:opacity-50 disabled:cursor-not-allowed"
								>
									<Save class="w-4 h-4" />
									{savingId === s.id ? 'Menyimpan...' : 'Simpan'}
								</button>
							</div>
						</form>
						{/if}
					</div>
				{/if}
			</div>
		{/each}
	</div>
</AppLayout>
