<script lang="ts" module>
	import type { FieldRule } from "./types";
	export interface Option { id: number; nama: string; info?: string; aktif: boolean }
	export interface LinkRow {
		id: number;
		slug: string;
		judul: string;
		deskripsi: string;
		program_id: number;
		lead_source_id: number;
		mode: "wa" | "short" | "full";
		after_submit: "wa" | "bayar";
		nominal_override: number | null;
		field_config: Record<string, FieldRule>;
		wa_template: string;
		izinkan_manual: boolean;
		is_aktif: boolean;
		program_nama?: string;
		sumber_nama?: string;
		harga_label?: string;
		klik?: number;
		submit?: number;
		bayar?: number;
	}
</script>

<script lang="ts">
	import { router } from "@inertiajs/svelte";
	import DaftarFields from "./DaftarFields.svelte";
	import { formKosong, type PublicLink } from "./types";

	interface Props { programs: Option[]; sources: Option[]; initial?: LinkRow | null; onclose: () => void }
	let { programs, sources, initial = null, onclose }: Props = $props();

	const OPSIONAL: [string, string][] = [
		["usia", "Usia"], ["domisili", "Kota domisili"], ["untuk_siapa", "Mendaftar untuk"],
		["riwayat_belajar", "Pengalaman belajar"], ["belajar_dimana", "Belajar di mana"], ["waktu_belajar", "Waktu belajar"],
	];
	const DEFAULT_SHORT = new Set(["usia", "domisili"]);

	function awal() {
		const base = initial ?? {
			slug: "", judul: "", deskripsi: "", program_id: programs.find((p) => p.aktif && p.info !== "belum terhubung ke produk")?.id ?? 0,
			lead_source_id: 0, mode: "short", after_submit: "wa", nominal_override: null, field_config: {}, wa_template: "", izinkan_manual: true, is_aktif: true,
		};
		const cfg: Record<string, FieldRule> = {};
		for (const [k] of OPSIONAL) {
			cfg[k] = base.field_config?.[k] ?? { tampil: base.mode === "full" || DEFAULT_SHORT.has(k), wajib: false };
		}
		return { ...base, field_config: cfg, nominal_text: base.nominal_override == null ? "" : String(base.nominal_override) };
	}
	let form = $state(awal());
	let menyimpan = $state(false);
	let previewForm = $state(formKosong());

	let programDipilih = $derived(programs.find((p) => p.id === Number(form.program_id)));
	let preview = $derived<PublicLink>({
		id: 0, slug: form.slug, judul: form.judul || "Judul halaman", deskripsi: form.deskripsi, mode: form.mode, after_submit: form.after_submit,
		program_nama: programDipilih?.nama ?? "Program", butuh_jadwal: Number(form.program_id) === 1, harga: Number(form.nominal_text) || 0,
		varian: [], field_config: form.field_config, izinkan_manual: form.izinkan_manual,
	});

	function slugDariJudul() {
		if (form.slug || !form.judul) return;
		form.slug = form.judul.toLowerCase().normalize("NFKD").replace(/[^a-z0-9]+/g, "-").replace(/^-+|-+$/g, "").slice(0, 60);
	}

	function simpan(e: Event) {
		e.preventDefault();
		menyimpan = true;
		const { nominal_text, ...rest } = form;
		const body = {
			...rest, program_id: Number(form.program_id), lead_source_id: Number(form.lead_source_id),
			nominal_override: nominal_text.trim() === "" ? null : Number(nominal_text),
		};
		const opts = { preserveScroll: true, onSuccess: onclose, onFinish: () => (menyimpan = false) };
		if (initial) router.put(`/app/leads/link-pendaftaran/${initial.id}`, body, opts);
		else router.post("/app/leads/link-pendaftaran", body, opts);
	}

	const panel = "rounded-xl border border-neutral-200/80 bg-white dark:border-white/[0.07] dark:bg-neutral-925/60";
	const field = "h-9 w-full rounded-md border border-neutral-200 bg-neutral-50 px-2.5 text-sm text-neutral-900 placeholder:text-neutral-400 focus:border-brand-500 focus:outline-none focus:ring-2 focus:ring-brand-500/30 dark:border-white/10 dark:bg-neutral-950 dark:text-white";
	const lbl = "block text-[11px] font-medium text-neutral-500 dark:text-neutral-400";
	const btn = "inline-flex h-9 items-center justify-center gap-1.5 rounded-md px-3 text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 disabled:opacity-50";
	const seg = "flex-1 cursor-pointer rounded-md px-2 py-1.5 text-center text-xs font-medium text-neutral-600 transition has-[:checked]:bg-white has-[:checked]:text-neutral-900 has-[:checked]:shadow-sm has-[:focus-visible]:ring-2 has-[:focus-visible]:ring-brand-500 dark:text-neutral-400 dark:has-[:checked]:bg-neutral-800 dark:has-[:checked]:text-white";
</script>

<div class="grid gap-4 lg:grid-cols-[minmax(0,1fr)_380px]">
	<form class={panel + " space-y-4 p-4 sm:p-5"} onsubmit={simpan}>
		<h2 class="text-base font-semibold text-neutral-900 dark:text-white">{initial ? "Ubah link" : "Link pendaftaran baru"}</h2>
		<div class="grid gap-3 sm:grid-cols-2">
			<label class={lbl + " sm:col-span-2"}>Judul halaman *<input class={field + " mt-1"} bind:value={form.judul} onblur={slugDariJudul} maxlength="120" required placeholder="mis. Daftar Kelas Tahsin Reguler" /></label>
			<label class={lbl}>Slug (alamat link) *<input class={field + " mt-1 font-mono"} bind:value={form.slug} required pattern="[a-z0-9]+(-[a-z0-9]+)*" minlength="3" maxlength="60" placeholder="tahsin-reguler" /></label>
			<label class={lbl}>Program *
				<select class={field + " mt-1"} bind:value={form.program_id} required>
					{#each programs as p (p.id)}<option value={p.id} disabled={p.info === "belum terhubung ke produk"}>{p.nama}{p.info ? ` · ${p.info}` : ""}{p.aktif ? "" : " (nonaktif)"}</option>{/each}
				</select>
			</label>
			<label class={lbl}>Sumber lead *
				<select class={field + " mt-1"} bind:value={form.lead_source_id} required>
					<option value={0} disabled>Pilih sumber</option>
					{#each sources as s (s.id)}<option value={s.id}>{s.nama}{s.aktif ? "" : " (nonaktif)"}</option>{/each}
				</select>
			</label>
			<label class={lbl}>Harga khusus (opsional)<input class={field + " mt-1"} type="number" min="0" step="1000" bind:value={form.nominal_text} placeholder="Kosong = harga produk/varian" /></label>
			<label class={lbl + " sm:col-span-2"}>Deskripsi<textarea class={field + " mt-1 h-auto py-2"} rows="2" bind:value={form.deskripsi} maxlength="1000"></textarea></label>
		</div>

		<div class="grid gap-3 sm:grid-cols-2">
			<div>
				<span class={lbl}>Mode</span>
				<div class="mt-1 flex gap-1 rounded-lg bg-neutral-100 p-1 dark:bg-neutral-900" role="radiogroup" aria-label="Mode">
					{#each [["wa", "WA langsung"], ["short", "Short form"], ["full", "Full form"]] as [v, t]}
						<label class={seg}><input type="radio" class="sr-only" bind:group={form.mode} value={v} />{t}</label>
					{/each}
				</div>
			</div>
			<div>
				<span class={lbl}>Setelah submit</span>
				<div class="mt-1 flex gap-1 rounded-lg bg-neutral-100 p-1 dark:bg-neutral-900 {form.mode === 'wa' ? 'opacity-50' : ''}" role="radiogroup" aria-label="Setelah submit">
					{#each [["wa", "Lanjut ke WA"], ["bayar", "Halaman bayar"]] as [v, t]}
						<label class={seg}><input type="radio" class="sr-only" bind:group={form.after_submit} value={v} disabled={form.mode === "wa"} />{t}</label>
					{/each}
				</div>
			</div>
		</div>

		{#if form.mode !== "wa"}
			<div>
				<p class={lbl}>Pertanyaan tambahan</p>
				<p class="mt-0.5 text-xs text-neutral-500">Nama, nomor WA, ikhwan/akhwat, paket, dan jadwal (Reguler) selalu ditanyakan.</p>
				<ul class="mt-2 divide-y divide-neutral-100 rounded-lg border border-neutral-200 dark:divide-white/[0.06] dark:border-white/10">
					{#each OPSIONAL as [k, t]}
						<li class="flex items-center justify-between gap-3 px-3 py-2 text-sm">
							<span class="text-neutral-800 dark:text-neutral-200">{t}</span>
							<span class="flex items-center gap-4 text-xs text-neutral-600 dark:text-neutral-400">
								<label class="flex items-center gap-1.5"><input type="checkbox" bind:checked={form.field_config[k].tampil} onchange={() => { if (!form.field_config[k].tampil) form.field_config[k].wajib = false; }} /> Tampil</label>
								<label class="flex items-center gap-1.5"><input type="checkbox" bind:checked={form.field_config[k].wajib} disabled={!form.field_config[k].tampil} /> Wajib</label>
							</span>
						</li>
					{/each}
				</ul>
			</div>
		{/if}

		<label class={lbl}>Template pesan WhatsApp (opsional)
			<textarea class={field + " mt-1 h-auto py-2"} rows="2" bind:value={form.wa_template} maxlength="500" placeholder={"Assalamu'alaikum, saya {nama} sudah isi form pendaftaran {program} ({ref})."}></textarea>
			<span class="mt-1 block font-normal text-neutral-400">Bisa memakai {"{nama}"}, {"{program}"}, {"{jadwal}"}, {"{ref}"}.</span>
		</label>

		<div class="flex flex-wrap items-center gap-x-5 gap-y-2 text-sm text-neutral-700 dark:text-neutral-300">
			{#if form.after_submit === "bayar" && form.mode !== "wa"}<label class="flex items-center gap-2"><input type="checkbox" bind:checked={form.izinkan_manual} /> Izinkan transfer manual + unggah bukti</label>{/if}
			<label class="flex items-center gap-2"><input type="checkbox" bind:checked={form.is_aktif} /> Link aktif</label>
		</div>

		<div class="flex justify-end gap-2 border-t border-neutral-100 pt-4 dark:border-white/[0.06]">
			<button type="button" class={btn + " border border-neutral-200 text-neutral-700 hover:bg-neutral-50 dark:border-white/10 dark:text-neutral-300"} onclick={onclose}>Batal</button>
			<button type="submit" class={btn + " bg-brand-600 text-white hover:bg-brand-700 dark:bg-brand-500"} disabled={menyimpan}>{initial ? "Simpan" : "Buat link"}</button>
		</div>
	</form>

	<aside class="hidden lg:block" aria-label="Pratinjau">
		<p class="mb-2 text-[11px] font-medium uppercase tracking-wide text-neutral-500">Pratinjau di HP</p>
		<div class="max-h-[720px] overflow-y-auto rounded-[28px] border-8 border-neutral-900 bg-neutral-50 p-4 dark:border-neutral-700 dark:bg-neutral-950">
			{#if form.mode === "wa"}
				<p class="py-20 text-center text-sm text-neutral-500">Pengunjung langsung diarahkan ke WhatsApp CS.</p>
			{:else}
				<div class="rounded-2xl border border-neutral-200/80 bg-white dark:border-white/[0.06] dark:bg-neutral-925/50">
					<div class="border-b border-neutral-200/80 p-4 dark:border-white/[0.05]">
						<p class="text-xs font-medium text-brand-700 dark:text-brand-400">{preview.program_nama}</p>
						<p class="mt-0.5 font-bold text-neutral-900 dark:text-white">{preview.judul}</p>
						{#if preview.deskripsi}<p class="mt-1 text-xs text-neutral-600 dark:text-neutral-400">{preview.deskripsi}</p>{/if}
					</div>
					<div class="relative p-4 text-sm"><DaftarFields link={preview} bind:form={previewForm} preview /></div>
				</div>
			{/if}
		</div>
	</aside>
</div>
