<script lang="ts">
	import { router } from "@inertiajs/svelte";

	interface FilterItem {
		key: string;
		label: string;
		options: { value: string; label: string }[];
	}

	interface Props {
		filters: FilterItem[];
		baseUrl: string;
	}

	let { filters, baseUrl }: Props = $props();

	let params = $state<Record<string, string>>({});

	function onFilterChange(key: string, value: string) {
		params = { ...params, [key]: value };
	}

	function applyFilters() {
		const q = new URLSearchParams();
		for (const [k, v] of Object.entries(params)) {
			if (v) q.set(k, v);
		}
		router.get(`${baseUrl}?${q.toString()}`);
	}

	function resetFilters() {
		params = {};
		router.get(baseUrl);
	}
</script>

<div class="bg-white dark:bg-neutral-925/50 border border-neutral-200/80 dark:border-white/[0.06] rounded-xl p-4 mb-4">
	<div class="flex flex-wrap gap-3 items-end">
		{#each filters as f}
			<div class="flex flex-col gap-1">
				<label class="text-xs font-medium text-neutral-500 dark:text-neutral-400">{f.label}</label>
				<select
					onchange={(e) => onFilterChange(f.key, e.currentTarget.value)}
					class="px-3 py-1.5 rounded-lg border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-900 text-sm text-neutral-700 dark:text-neutral-300 focus:outline-none focus:ring-2 focus:ring-brand-400/40"
				>
					<option value="">Semua</option>
					{#each f.options as opt}
						<option value={opt.value}>{opt.label}</option>
					{/each}
				</select>
			</div>
		{/each}
		<div class="flex gap-2">
			<button onclick={applyFilters} class="px-4 py-1.5 rounded-lg bg-brand-600 hover:bg-brand-700 text-white text-sm font-medium transition-colors">
				Filter
			</button>
			<button onclick={resetFilters} class="px-4 py-1.5 rounded-lg bg-neutral-200/80 dark:bg-neutral-800 text-neutral-700 dark:text-neutral-300 text-sm font-medium transition-colors">
				Reset
			</button>
		</div>
	</div>
</div>
