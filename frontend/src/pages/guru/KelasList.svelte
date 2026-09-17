<script lang="ts">
	import { inertia, router } from "@inertiajs/svelte";
	import { fly } from "svelte/transition";
	import AppLayout from "@layouts/AppLayout.svelte";
	import GenderBadge from "@components/GenderBadge.svelte";
	import type { User, GuruKelas } from "@lib/types";
	import { BookOpen, Users, Calendar, Clock, ArrowRight } from "lucide-svelte";

	interface Props {
		user?: User;
		kelas?: GuruKelas[];
		success?: string;
		error?: string;
	}

	let { user, kelas = [], success, error }: Props = $props();
	let isAllGuruView = $derived(user?.role === "super_admin" || user?.role === "admin_kelas");
	let selectedGuru = $state("");
	let guruOptions = $derived(
		Array.from(new Map(kelas.filter((k) => k.guru_id).map((k) => [String(k.guru_id), k.guru_nama])).entries())
			.map(([id, nama]) => ({ id, nama }))
			.sort((a, b) => a.nama.localeCompare(b.nama)),
	);
	let filteredKelas = $derived(selectedGuru ? kelas.filter((k) => String(k.guru_id ?? "") === selectedGuru) : kelas);

	function groupKelas(list: GuruKelas[]) {
		const groups = new Map<string, { guruNama: string; kelas: GuruKelas[] }>();
		for (const item of list) {
			const key = String(item.guru_id ?? "belum-ditugaskan");
			const group = groups.get(key) ?? { guruNama: item.guru_nama || "Belum ditugaskan", kelas: [] };
			group.kelas.push(item);
			groups.set(key, group);
		}
		return Array.from(groups.values()).sort((a, b) => a.guruNama.localeCompare(b.guruNama));
	}

	let kelasPerGuru = $derived(groupKelas(filteredKelas));
</script>

<AppLayout {user} group="guru-kelas">
	<div class="pt-8 pb-10 border-b border-neutral-200/80 dark:border-white/[0.04]">
		<div class="max-w-6xl mx-auto px-4 sm:px-6">
			<div class="flex items-center gap-2 text-sm text-neutral-500 dark:text-neutral-400 mb-4">
				<a href="/app" use:inertia class="hover:text-brand-600 dark:hover:text-brand-400 transition-colors">Dashboard</a>
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
				</svg>
				<span class="text-neutral-700 dark:text-neutral-300">{isAllGuruView ? "Kelas Guru" : "Kelas Saya"}</span>
			</div>
			<h1 class="text-2xl sm:text-3xl font-bold text-neutral-900 dark:text-white mb-2 tracking-tight">
				{isAllGuruView ? "Kelas Guru" : "Kelas Saya"}
			</h1>
			<p class="text-neutral-600 dark:text-neutral-400">{isAllGuruView ? "Daftar seluruh kelas, dikelompokkan berdasarkan guru" : "Daftar kelas yang Anda ajar"}</p>
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

		{#if kelas.length > 0}
			{#if isAllGuruView}
				<div class="flex items-center gap-3 rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-4" in:fly={{ y: 20, duration: 500 }}>
					<label for="guru-filter" class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Guru</label>
					<select id="guru-filter" bind:value={selectedGuru} class="min-w-52 rounded-xl border border-neutral-300 dark:border-neutral-700 bg-neutral-100/80 dark:bg-neutral-800/50 px-3 py-2 text-sm text-neutral-900 dark:text-white outline-none focus:border-brand-400">
						<option value="">Semua guru</option>
						{#each guruOptions as guru}
							<option value={guru.id}>{guru.nama}</option>
						{/each}
					</select>
				</div>
			{/if}

			{#if kelasPerGuru.length > 0}
				<div class="space-y-7" in:fly={{ y: 20, duration: 600 }}>
					{#each kelasPerGuru as group}
						<section>
							{#if isAllGuruView}
								<div class="flex items-center gap-2 mb-3">
									<Users class="w-4 h-4 text-brand-600 dark:text-brand-400" />
									<h2 class="text-base font-semibold text-neutral-900 dark:text-white">{group.guruNama}</h2>
									<span class="text-xs text-neutral-500 dark:text-neutral-400">{group.kelas.length} kelas</span>
								</div>
							{/if}
							<div class="grid md:grid-cols-2 xl:grid-cols-3 gap-4">
								{#each group.kelas as k}
					<a href={"/app/guru/kelas/" + k.id} use:inertia
						class="group relative rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-5 transition-all hover:border-brand-400/30 hover:shadow-lg hover:shadow-brand-400/5"
					>
						<div class="flex items-start justify-between gap-2 mb-3">
							<div class="flex items-start gap-2.5 min-w-0">
								<div class="w-10 h-10 rounded-xl bg-brand-400/10 flex items-center justify-center shrink-0">
									<BookOpen class="w-5 h-5 text-brand-600 dark:text-brand-400" />
								</div>
								<div class="min-w-0">
									<h3 class="font-semibold text-neutral-900 dark:text-white leading-snug break-words">{k.nama_kelas}</h3>
									<p class="text-xs text-neutral-500 dark:text-neutral-400 mt-0.5">{k.tipe}</p>
								</div>
							</div>
							<GenderBadge gender={k.jenis_kelamin} />
						</div>

						<div class="space-y-1.5 mb-3">
							<div class="flex items-center gap-2 text-xs text-neutral-600 dark:text-neutral-400">
								<Clock class="w-3.5 h-3.5 shrink-0" />
								<span class="font-medium">{k.level}</span>
								<span class="text-neutral-500">·</span>
								<span>{k.frekuensi}</span>
							</div>
							<div class="flex items-start gap-2 text-xs text-neutral-600 dark:text-neutral-400">
								<Calendar class="w-3.5 h-3.5 shrink-0 mt-0.5" />
								<span class="min-w-0 break-words">{k.jadwal}</span>
							</div>
						</div>

						<div class="flex items-center justify-between pt-3 border-t border-neutral-200/70 dark:border-white/[0.04]">
							<div class="flex items-center gap-1.5 text-xs font-medium">
								<Users class="w-3.5 h-3.5 text-neutral-500" />
								<span class="text-neutral-700 dark:text-neutral-300">{k.jumlah_santri}/{k.kapasitas}</span>
							</div>
							<div class="flex items-center gap-2">
								{#if k.tanggal_terakhir}
									<span class="text-[10px] text-neutral-500 dark:text-neutral-400 bg-neutral-100 dark:bg-neutral-800 px-2 py-0.5 rounded-md">
										{k.tanggal_terakhir}
									</span>
								{/if}
								<ArrowRight class="w-4 h-4 text-neutral-400 group-hover:text-brand-500 transition-colors shrink-0" />
							</div>
						</div>
					</a>
								{/each}
							</div>
						</section>
					{/each}
				</div>
			{:else}
				<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-10 text-center text-sm text-neutral-500 dark:text-neutral-400">Tidak ada kelas untuk guru yang dipilih.</div>
			{/if}
		{:else}
			<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-12 text-center" in:fly={{ y: 20, duration: 500, delay: 100 }}>
				<div class="w-16 h-16 rounded-2xl bg-neutral-100 dark:bg-neutral-800 flex items-center justify-center mx-auto mb-4">
					<BookOpen class="w-8 h-8 text-neutral-500" />
				</div>
				<h3 class="text-lg font-semibold text-neutral-900 dark:text-white mb-2">Belum ada kelas</h3>
				<p class="text-neutral-500 dark:text-neutral-400 max-w-md mx-auto text-sm">
					Anda belum memiliki kelas yang diajar. Hubungi admin untuk mendapatkan penugasan kelas.
				</p>
			</div>
		{/if}
	</div>
</AppLayout>
