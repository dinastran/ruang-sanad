<script lang="ts">
	import { inertia, router } from "@inertiajs/svelte";
	import AppLayout from "@layouts/AppLayout.svelte";
	import type { Kalam, KalamShareRow, User } from "@lib/types";
	import { ArrowLeft, Check } from "lucide-svelte";

	interface Props { user?: User; kalam: Kalam; share?: KalamShareRow[]; success?: string; error?: string; }
	let { user, kalam, share = [], success, error }: Props = $props();

	function toggle(guruId: number, on: boolean) {
		router.post(`/app/koordinator-guru/kalam/${kalam.id}/share`, { guru_id: guruId, on }, { preserveScroll: true, preserveState: false });
	}
	const shared = $derived(share.filter((s) => s.sudah_share).length);
</script>

<AppLayout {user} group="koordinator-kalam">
	<div class="pt-8 pb-8 border-b border-neutral-200/80 dark:border-white/[0.04]">
		<div class="max-w-3xl mx-auto px-4 sm:px-6">
			<a href="/app/koordinator-guru/kalam" use:inertia class="inline-flex items-center gap-1.5 text-sm text-neutral-500 hover:text-brand-600 mb-4"><ArrowLeft class="w-4 h-4" /> Kalam Bersanad</a>
			<h1 class="text-2xl font-bold text-neutral-900 dark:text-white">Pelacakan Share</h1>
			<p class="mt-1.5 text-neutral-600 dark:text-neutral-400">{kalam.tanggal} — {kalam.topik || "(tanpa topik)"}</p>
		</div>
	</div>

	<div class="max-w-3xl mx-auto px-4 sm:px-6 py-8 space-y-4">
		{#if success}<div class="rounded-xl bg-green-500/10 border border-green-500/20 p-4 text-sm font-medium text-green-700 dark:text-green-400">{success}</div>{/if}
		{#if error}<div class="rounded-xl bg-red-500/10 border border-red-500/20 p-4 text-sm font-medium text-red-700 dark:text-red-400">{error}</div>{/if}

		<p class="text-sm text-neutral-600 dark:text-neutral-400">Sudah share: <span class="font-semibold text-neutral-900 dark:text-white">{shared}</span> / {share.length} guru</p>

		<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 divide-y divide-neutral-200/80 dark:divide-white/[0.04]">
			{#each share as s (s.guru_id)}
				<div class="flex items-center gap-3 p-4">
					<button onclick={() => toggle(s.guru_id, !s.sudah_share)} class="w-6 h-6 rounded-md border-2 flex items-center justify-center shrink-0 {s.sudah_share ? 'bg-green-500 border-green-500 text-white' : 'border-neutral-300 dark:border-neutral-600'}">{#if s.sudah_share}<Check class="w-4 h-4" />{/if}</button>
					<p class="text-sm font-medium text-neutral-900 dark:text-white">{s.nama}</p>
				</div>
			{/each}
		</div>
	</div>
</AppLayout>
