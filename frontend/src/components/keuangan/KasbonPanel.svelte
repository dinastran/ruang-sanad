<script lang="ts">
	import { router } from "@inertiajs/svelte";
	import type { Kasbon, UjrohTarifGuru } from "@lib/types";
	import { ChevronDown, HandCoins, Plus } from "lucide-svelte";

	interface Props {
		kasbon: Kasbon[];
		guru: UjrohTarifGuru[];
		bulan: string;
		bulanTerkunci: boolean;
		canEdit: boolean;
	}

	let { kasbon, guru, bulan, bulanTerkunci, canEdit }: Props = $props();

	const rupiah = (n: number) => new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 0 }).format(n);
	const namaBulan = (b: string) => {
		if (!b) return "";
		const [y, m] = b.split("-").map(Number);
		return new Date(y, m - 1, 1).toLocaleDateString("id-ID", { month: "long", year: "numeric" });
	};
	const tanggalPanjang = (t: string) => (t ? new Date(t + "T00:00:00").toLocaleDateString("id-ID", { day: "numeric", month: "short", year: "numeric" }) : "");
	const inputClass = "w-full rounded-xl border border-neutral-300 bg-neutral-50 px-3 py-2 text-sm text-neutral-900 outline-none focus:border-brand-400 focus:ring-2 focus:ring-brand-400/15 dark:border-neutral-700 dark:bg-neutral-800/60 dark:text-white";
	const hariIni = () => new Date().toLocaleDateString("en-CA", { timeZone: "Asia/Jakarta" });

	function bulanSetelah(b: string): string {
		const [y, m] = b.split("-").map(Number);
		const d = new Date(y, m, 1);
		return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}`;
	}

	let q = $derived(`?bulan=${encodeURIComponent(bulan)}`);
	let open = $state(false);
	let busy = $state(false);
	let tampilSemua = $state(false);
	const opts = { preserveScroll: true, preserveState: true, onStart: () => (busy = true), onFinish: () => (busy = false) };

	let guruAktif = $derived(guru.filter((g) => g.is_aktif));
	let berjalan = $derived(kasbon.filter((k) => k.status === "berjalan"));
	let totalSisa = $derived(berjalan.reduce((a, k) => a + k.sisa, 0));
	let tampil = $derived(tampilSemua ? kasbon : berjalan);

	type Form = { guru_id: number | null; tanggal: string; nominal: number; cara: "sekaligus" | "cicil"; cicilan: number; mulai_bulan: string; keterangan: string };
	let form = $state<Form | null>(null);
	let formValid = $derived(
		!!form && !!form.guru_id && !!form.tanggal && Number(form.nominal) > 0 && !!form.mulai_bulan &&
		(form.cara === "sekaligus" || (Number(form.cicilan) > 0 && Number(form.cicilan) <= Number(form.nominal))),
	);
	let jumlahBulan = $derived(form && form.cara === "cicil" && Number(form.cicilan) > 0 ? Math.ceil(Number(form.nominal) / Number(form.cicilan)) : 1);

	function mulaiTambah() {
		form = {
			guru_id: null, tanggal: hariIni(), nominal: 0, cara: "sekaligus", cicilan: 0,
			// Default ke bulan yang sedang dibuka bila belum dikunci, kalau tidak bulan berikutnya.
			mulai_bulan: bulanTerkunci ? bulanSetelah(bulan) : bulan, keterangan: "",
		};
		open = true;
	}

	function simpan() {
		if (!form || !formValid) return;
		router.post(`/app/keuangan/kasbon${q}`, {
			guru_id: form.guru_id, tanggal: form.tanggal, nominal: Number(form.nominal),
			cicilan: form.cara === "cicil" ? Number(form.cicilan) : 0, mulai_bulan: form.mulai_bulan, keterangan: form.keterangan,
		}, { ...opts, onSuccess: (page) => { if (!(page.props.flash as { error?: string } | undefined)?.error) form = null; } });
	}

	let editID = $state<number | null>(null);
	let edit = $state({ cara: "sekaligus" as "sekaligus" | "cicil", cicilan: 0, keterangan: "" });

	function mulaiEdit(k: Kasbon) {
		editID = k.id;
		edit = { cara: k.cicilan > 0 ? "cicil" : "sekaligus", cicilan: k.cicilan, keterangan: k.keterangan };
	}

	function simpanEdit(k: Kasbon) {
		router.put(`/app/keuangan/kasbon/${k.id}${q}`, { cicilan: edit.cara === "cicil" ? Number(edit.cicilan) : 0, keterangan: edit.keterangan }, {
			...opts,
			onSuccess: () => (editID = null),
		});
	}

	function batalkan(k: Kasbon) {
		if (!confirm(`Batalkan kasbon ${k.guru_nama} ${rupiah(k.nominal)}? Gunakan hanya bila salah input.`)) return;
		router.post(`/app/keuangan/kasbon/${k.id}/batal${q}`, {}, opts);
	}

	const statusClass = (s: string) =>
		s === "lunas" ? "bg-success/10 text-success" : s === "dibatalkan" ? "bg-neutral-100 text-neutral-500 dark:bg-neutral-800" : "bg-warning/10 text-warning";
</script>

<section class="rounded-2xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-925">
	<button onclick={() => (open = !open)} aria-expanded={open} class="flex w-full items-center justify-between gap-3 px-5 py-4 text-left">
		<span class="flex items-center gap-2 font-semibold text-neutral-900 dark:text-white"><HandCoins size="18" /> Kasbon guru</span>
		<span class="flex items-center gap-2 text-sm text-neutral-500">
			{berjalan.length} berjalan · sisa {rupiah(totalSisa)}
			<ChevronDown size="16" class="transition-transform {open ? 'rotate-180' : ''}" />
		</span>
	</button>
	{#if open}
		<div class="space-y-4 border-t border-neutral-200 dark:border-neutral-800 px-5 py-5">
			<div class="flex flex-wrap items-center justify-between gap-2">
				<p class="text-sm text-neutral-600 dark:text-neutral-400">Kasbon dipotong otomatis dari ujroh mulai bulan yang dipilih, tidak melebihi ujroh bulan itu. Potongan dibekukan saat bulan dikunci.</p>
				<div class="flex gap-2">
					<label class="flex items-center gap-1.5 text-xs text-neutral-600 dark:text-neutral-400"><input type="checkbox" bind:checked={tampilSemua} /> Tampilkan lunas/dibatalkan</label>
					{#if canEdit && !form}
						<button onclick={mulaiTambah} class="inline-flex items-center gap-1.5 rounded-xl bg-brand-600 px-3.5 py-2 text-sm font-semibold text-white hover:bg-brand-700"><Plus size="16" /> Catat kasbon</button>
					{/if}
				</div>
			</div>

			{#if form}
				<div class="space-y-3 rounded-xl border border-dashed border-neutral-300 p-4 dark:border-neutral-700">
					<div class="grid gap-3 sm:grid-cols-3">
						<label class="text-xs font-medium text-neutral-600 dark:text-neutral-400">Guru
							<select bind:value={form.guru_id} class="{inputClass} mt-1">
								<option value={null} disabled>Pilih guru</option>
								{#each guruAktif as g (g.guru_id)}<option value={g.guru_id}>{g.nama}</option>{/each}
							</select>
						</label>
						<label class="text-xs font-medium text-neutral-600 dark:text-neutral-400">Tanggal diberikan<input type="date" bind:value={form.tanggal} class="{inputClass} mt-1" /></label>
						<label class="text-xs font-medium text-neutral-600 dark:text-neutral-400">Nominal (Rp)<input type="number" min="0" step="10000" bind:value={form.nominal} class="{inputClass} mt-1" /></label>
					</div>
					<div class="grid gap-3 sm:grid-cols-3">
						<fieldset class="text-xs font-medium text-neutral-600 dark:text-neutral-400">
							<legend>Cara potong</legend>
							<div class="mt-1 flex gap-3 py-2">
								<label class="flex items-center gap-1.5"><input type="radio" value="sekaligus" bind:group={form.cara} /> Sekaligus</label>
								<label class="flex items-center gap-1.5"><input type="radio" value="cicil" bind:group={form.cara} /> Dicicil</label>
							</div>
						</fieldset>
						{#if form.cara === "cicil"}
							<label class="text-xs font-medium text-neutral-600 dark:text-neutral-400">Cicilan per bulan (Rp)<input type="number" min="0" step="10000" bind:value={form.cicilan} class="{inputClass} mt-1" /></label>
						{/if}
						<label class="text-xs font-medium text-neutral-600 dark:text-neutral-400">Mulai dipotong<input type="month" bind:value={form.mulai_bulan} class="{inputClass} mt-1" /></label>
					</div>
					<label class="block text-xs font-medium text-neutral-600 dark:text-neutral-400">Keterangan<input bind:value={form.keterangan} placeholder="mis. keperluan berobat" class="{inputClass} mt-1" /></label>
					<div class="flex flex-wrap items-center justify-between gap-2">
						<p class="text-xs text-neutral-500">
							{#if formValid}
								{form.cara === "cicil" ? `Sekitar ${jumlahBulan} bulan × ${rupiah(Number(form.cicilan))}` : `Dipotong ${rupiah(Number(form.nominal))}`} mulai {namaBulan(form.mulai_bulan)}.
							{:else}
								Lengkapi guru, nominal, dan cicilan (bila dicicil).
							{/if}
						</p>
						<div class="flex gap-2">
							<button onclick={() => (form = null)} class="rounded-lg px-3 py-2 text-sm font-medium text-neutral-600 hover:bg-neutral-100 dark:text-neutral-300 dark:hover:bg-neutral-800">Batal</button>
							<button onclick={simpan} disabled={busy || !formValid} class="rounded-lg bg-brand-600 px-4 py-2 text-sm font-semibold text-white hover:bg-brand-700 disabled:opacity-50">Simpan kasbon</button>
						</div>
					</div>
				</div>
			{/if}

			<div class="overflow-x-auto rounded-xl border border-neutral-200 dark:border-neutral-800">
				<table class="w-full min-w-[720px] text-sm">
					<thead>
						<tr class="border-b border-neutral-200 bg-neutral-50 text-left text-xs font-semibold uppercase tracking-wide text-neutral-500 dark:border-neutral-800 dark:bg-neutral-900/60">
							<th class="px-3 py-2.5">Guru</th>
							<th class="px-3 py-2.5 text-right">Kasbon</th>
							<th class="px-3 py-2.5">Potong</th>
							<th class="px-3 py-2.5 text-right">Terpotong</th>
							<th class="px-3 py-2.5 text-right">Sisa</th>
							<th class="px-3 py-2.5"></th>
						</tr>
					</thead>
					<tbody class="divide-y divide-neutral-100 dark:divide-neutral-800/70">
						{#each tampil as k (k.id)}
							<tr class="align-top {k.status === 'dibatalkan' ? 'opacity-60' : ''}">
								<td class="px-3 py-2.5">
									<p class="font-medium text-neutral-900 dark:text-white">{k.guru_nama}</p>
									<p class="text-xs text-neutral-500">{tanggalPanjang(k.tanggal)}{k.keterangan ? ` · ${k.keterangan}` : ""}</p>
								</td>
								<td class="px-3 py-2.5 text-right font-mono">{rupiah(k.nominal)}</td>
								<td class="px-3 py-2.5 text-xs text-neutral-600 dark:text-neutral-400">
									{#if editID === k.id}
										<div class="space-y-1.5">
											<div class="flex gap-2">
												<label class="flex items-center gap-1"><input type="radio" value="sekaligus" bind:group={edit.cara} /> Sekaligus</label>
												<label class="flex items-center gap-1"><input type="radio" value="cicil" bind:group={edit.cara} /> Cicil</label>
											</div>
											{#if edit.cara === "cicil"}<input type="number" min="0" step="10000" bind:value={edit.cicilan} aria-label="Cicilan per bulan" class="{inputClass} !w-36" />{/if}
											<input bind:value={edit.keterangan} placeholder="Keterangan" aria-label="Keterangan" class="{inputClass} !w-48" />
										</div>
									{:else}
										{k.cicilan > 0 ? `${rupiah(k.cicilan)}/bulan` : "Sekaligus"}<br />mulai {namaBulan(k.mulai_bulan)}
									{/if}
								</td>
								<td class="px-3 py-2.5 text-right font-mono">{rupiah(k.terpotong)}</td>
								<td class="px-3 py-2.5 text-right">
									<p class="font-mono font-semibold text-neutral-900 dark:text-white">{rupiah(Math.max(k.sisa, 0))}</p>
									<span class="mt-0.5 inline-flex rounded-full px-2 py-0.5 text-[11px] font-semibold {statusClass(k.status)}">{k.status}</span>
								</td>
								<td class="px-3 py-2.5 text-right whitespace-nowrap">
									{#if canEdit && k.status === "berjalan"}
										{#if editID === k.id}
											<button onclick={() => (editID = null)} class="rounded-lg px-2.5 py-1.5 text-xs font-medium text-neutral-600 hover:bg-neutral-100 dark:text-neutral-300 dark:hover:bg-neutral-800">Batal</button>
											<button onclick={() => simpanEdit(k)} disabled={busy || (edit.cara === "cicil" && !(Number(edit.cicilan) > 0))} class="rounded-lg bg-brand-600 px-2.5 py-1.5 text-xs font-semibold text-white hover:bg-brand-700 disabled:opacity-50">Simpan</button>
										{:else}
											<button onclick={() => mulaiEdit(k)} class="rounded-lg px-2.5 py-1.5 text-xs font-medium text-neutral-600 hover:bg-neutral-100 dark:text-neutral-300 dark:hover:bg-neutral-800">Ubah cicilan</button>
											{#if k.terpotong === 0}<button onclick={() => batalkan(k)} disabled={busy} class="rounded-lg px-2.5 py-1.5 text-xs font-medium text-error hover:bg-error/10">Batalkan</button>{/if}
										{/if}
									{/if}
								</td>
							</tr>
						{:else}
							<tr><td colspan="6" class="px-3 py-6 text-center text-sm text-neutral-500">{tampilSemua ? "Belum ada kasbon." : "Tidak ada kasbon berjalan."}</td></tr>
						{/each}
					</tbody>
				</table>
			</div>
		</div>
	{/if}
</section>
