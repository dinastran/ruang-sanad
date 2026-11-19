<script lang="ts">
	import type { DaftarForm, JadwalOpsi, PublicLink } from "./types";
	import { rupiah } from "./types";

	interface Props {
		link: PublicLink;
		jadwal?: JadwalOpsi[];
		form: DaftarForm;
		errors?: Record<string, string>;
		preview?: boolean;
	}
	let { link, jadwal = [], form = $bindable(), errors = {}, preview = false }: Props = $props();

	const tampil = (f: string) => link.field_config?.[f]?.tampil ?? false;
	const wajib = (f: string) => link.field_config?.[f]?.wajib ?? false;

	let jadwalSesuai = $derived(
		jadwal.filter((j) => !form.jenis_kelamin || !j.gender || j.gender === form.jenis_kelamin),
	);
	const sisa = (j: JadwalOpsi) => (form.jenis_kelamin === "P" ? j.sisa_akhwat : form.jenis_kelamin === "L" ? j.sisa_ikhwan : Math.max(j.sisa_ikhwan, j.sisa_akhwat));

	const inputCls =
		"mt-1.5 w-full rounded-xl border border-neutral-300 bg-white px-4 py-3 text-base text-neutral-900 outline-none transition focus:border-brand-500 focus:ring-2 focus:ring-brand-500/30 aria-[invalid=true]:border-red-500 dark:border-white/10 dark:bg-neutral-900 dark:text-white";
	const labelCls = "text-sm font-medium text-neutral-800 dark:text-neutral-200";
	const opsiCls =
		"flex cursor-pointer items-center justify-between gap-3 rounded-xl border px-4 py-3 text-sm transition has-[:checked]:border-brand-600 has-[:checked]:bg-brand-50 has-[:checked]:ring-1 has-[:checked]:ring-brand-600 has-[:focus-visible]:ring-2 has-[:focus-visible]:ring-brand-500 dark:has-[:checked]:bg-brand-500/10";
</script>

{#snippet galat(field: string)}
	{#if errors[field]}<p id="err-{field}" class="mt-1.5 text-sm text-red-600 dark:text-red-400">{errors[field]}</p>{/if}
{/snippet}

<fieldset class="space-y-4" disabled={preview}>
	<legend class="sr-only">Data diri</legend>
	<label class="block">
		<span class={labelCls}>Nama lengkap</span>
		<input bind:value={form.nama} name="nama" autocomplete="name" required maxlength="80" aria-invalid={!!errors.nama} aria-describedby={errors.nama ? "err-nama" : undefined} class={inputCls} />
		{@render galat("nama")}
	</label>
	<label class="block">
		<span class={labelCls}>Nomor WhatsApp</span>
		<input bind:value={form.no_wa} name="no_wa" type="tel" inputmode="tel" autocomplete="tel" required placeholder="0812xxxxxxxx" aria-invalid={!!errors.no_wa} aria-describedby={errors.no_wa ? "err-no_wa" : undefined} class={inputCls} />
		{@render galat("no_wa")}
	</label>
	<div role="radiogroup" aria-labelledby="lbl-jk">
		<span id="lbl-jk" class={labelCls}>Mendaftar sebagai</span>
		<div class="mt-1.5 grid grid-cols-2 gap-2">
			{#each [["L", "Ikhwan"], ["P", "Akhwat"]] as [val, teks]}
				<label class="{opsiCls} justify-center border-neutral-300 font-medium text-neutral-800 dark:border-white/10 dark:text-neutral-200">
					<input type="radio" name="jenis_kelamin" value={val} bind:group={form.jenis_kelamin} required class="sr-only" onchange={() => (form.jadwal_kode = "")} />
					{teks}
				</label>
			{/each}
		</div>
		{@render galat("jenis_kelamin")}
	</div>
	{#if tampil("usia") || tampil("domisili")}
		<div class="grid grid-cols-[96px_1fr] gap-3">
			{#if tampil("usia")}
				<label class="block">
					<span class={labelCls}>Usia{wajib("usia") ? "" : " (opsional)"}</span>
					<input bind:value={form.usia} name="usia" type="number" inputmode="numeric" min="5" max="90" required={wajib("usia")} aria-invalid={!!errors.usia} class={inputCls} />
				</label>
			{/if}
			{#if tampil("domisili")}
				<label class="block {tampil('usia') ? '' : 'col-span-2'}">
					<span class={labelCls}>Kota domisili{wajib("domisili") ? "" : " (opsional)"}</span>
					<input bind:value={form.domisili} name="domisili" autocomplete="address-level2" required={wajib("domisili")} maxlength="80" aria-invalid={!!errors.domisili} class={inputCls} />
				</label>
			{/if}
		</div>
		{@render galat("usia")}{@render galat("domisili")}
	{/if}
</fieldset>

{#if link.varian.length}
	<fieldset class="mt-7" disabled={preview}>
		<legend class="text-base font-semibold text-neutral-900 dark:text-white">Pilih paket</legend>
		<div class="mt-3 space-y-2">
			{#each link.varian as v (v.id)}
				<label class="{opsiCls} border-neutral-200 dark:border-white/10">
					<span class="flex items-center gap-3">
						<input type="radio" name="varian_id" value={v.id} bind:group={form.varian_id} required class="h-4 w-4 accent-brand-600" />
						<span class="font-medium text-neutral-900 dark:text-white">{v.nama}</span>
					</span>
					<span class="tabular-nums font-semibold text-neutral-900 dark:text-white">{rupiah(v.harga)}</span>
				</label>
			{/each}
		</div>
		{@render galat("varian_id")}
	</fieldset>
{/if}

{#if link.butuh_jadwal}
	<fieldset class="mt-7" disabled={preview}>
		<legend class="text-base font-semibold text-neutral-900 dark:text-white">Pilih jadwal</legend>
		{#if !form.jenis_kelamin}
			<p class="mt-2 text-sm text-neutral-500">Pilih ikhwan atau akhwat dulu untuk melihat jadwal yang tersedia.</p>
		{:else if jadwalSesuai.length === 0}
			<p class="mt-2 rounded-xl bg-amber-500/10 p-4 text-sm text-amber-800 dark:text-amber-300">Belum ada jadwal yang dibuka saat ini. Silakan hubungi CS lewat WhatsApp untuk info jadwal berikutnya.</p>
		{:else}
			<div class="mt-3 space-y-2">
				{#each jadwalSesuai as j (j.kode)}
					{@const habis = sisa(j) <= 0}
					<label class="{opsiCls} border-neutral-200 dark:border-white/10 {habis ? 'cursor-not-allowed opacity-50' : ''}">
						<span class="flex items-center gap-3">
							<input type="radio" name="jadwal_kode" value={j.kode} bind:group={form.jadwal_kode} disabled={habis} required class="h-4 w-4 accent-brand-600" />
							<span>
								<span class="block font-medium text-neutral-900 dark:text-white">{j.label}</span>
								{#if j.catatan}<span class="block text-xs text-neutral-500">{j.catatan}</span>{/if}
							</span>
						</span>
						<span class="shrink-0 text-xs {habis ? 'font-semibold text-red-600 dark:text-red-400' : sisa(j) <= 3 ? 'font-medium text-amber-700 dark:text-amber-400' : 'text-neutral-500'}">
							{habis ? "Penuh" : `Sisa ${sisa(j)}`}
						</span>
					</label>
				{/each}
			</div>
		{/if}
		{@render galat("jadwal_kode")}
	</fieldset>
{/if}

{#if tampil("untuk_siapa") || tampil("riwayat_belajar") || tampil("belajar_dimana") || tampil("waktu_belajar")}
	<fieldset class="mt-7 space-y-4" disabled={preview}>
		<legend class="text-base font-semibold text-neutral-900 dark:text-white">Sedikit tentang Anda</legend>
		{#if tampil("untuk_siapa")}
			<label class="block">
				<span class={labelCls}>Mendaftar untuk{wajib("untuk_siapa") ? "" : " (opsional)"}</span>
				<select bind:value={form.untuk_siapa} name="untuk_siapa" required={wajib("untuk_siapa")} class={inputCls}>
					<option value="">Pilih…</option>
					<option value="diri_sendiri">Diri sendiri</option>
					<option value="anak">Anak</option>
					<option value="keluarga">Anggota keluarga</option>
				</select>
				{@render galat("untuk_siapa")}
			</label>
		{/if}
		{#if tampil("riwayat_belajar")}
			<label class="block">
				<span class={labelCls}>Pengalaman belajar Al-Qur'an{wajib("riwayat_belajar") ? "" : " (opsional)"}</span>
				<textarea bind:value={form.riwayat_belajar} name="riwayat_belajar" rows="3" maxlength="500" required={wajib("riwayat_belajar")} placeholder="Mis. sudah lancar Iqra, pernah tahsin 3 bulan" class={inputCls}></textarea>
				{@render galat("riwayat_belajar")}
			</label>
		{/if}
		{#if tampil("belajar_dimana")}
			<label class="block">
				<span class={labelCls}>Sebelumnya belajar di mana{wajib("belajar_dimana") ? "" : " (opsional)"}</span>
				<input bind:value={form.belajar_dimana} name="belajar_dimana" maxlength="200" required={wajib("belajar_dimana")} class={inputCls} />
				{@render galat("belajar_dimana")}
			</label>
		{/if}
		{#if tampil("waktu_belajar")}
			<label class="block">
				<span class={labelCls}>Waktu belajar yang diinginkan{wajib("waktu_belajar") ? "" : " (opsional)"}</span>
				<input bind:value={form.waktu_belajar} name="waktu_belajar" maxlength="200" required={wajib("waktu_belajar")} placeholder="Mis. malam setelah Isya" class={inputCls} />
				{@render galat("waktu_belajar")}
			</label>
		{/if}
	</fieldset>
{/if}

<!-- Honeypot: tersembunyi dari manusia, diisi bot. -->
<div class="absolute -left-[9999px] h-px w-px overflow-hidden" aria-hidden="true">
	<label>Website <input bind:value={form.website} name="website" tabindex="-1" autocomplete="off" /></label>
</div>
