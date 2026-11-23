<script lang="ts">
	import type { DaftarForm, JadwalOpsi, PublicLink } from "./types";
	import { rupiah } from "./types";
	import { Check, ChevronDown } from "lucide-svelte";

	interface Props {
		link: PublicLink;
		jadwal?: JadwalOpsi[];
		form: DaftarForm;
		errors?: Record<string, string>;
		preview?: boolean;
		/** Which part to render: everything (admin preview), step 1 or step 2. */
		bagian?: "semua" | "diri" | "pilihan";
		onblurfield?: (field: string) => void;
	}
	let { link, jadwal = [], form = $bindable(), errors = {}, preview = false, bagian = "semua", onblurfield }: Props = $props();

	const tampil = (f: string) => link.field_config?.[f]?.tampil ?? false;
	const wajib = (f: string) => link.field_config?.[f]?.wajib ?? false;
	const diri = $derived(bagian !== "pilihan");
	const pilihan = $derived(bagian !== "diri");

	let jadwalSesuai = $derived(
		jadwal.filter((j) => !form.jenis_kelamin || !j.gender || j.gender === form.jenis_kelamin),
	);
	const sisa = (j: JadwalOpsi) => (form.jenis_kelamin === "P" ? j.sisa_akhwat : form.jenis_kelamin === "L" ? j.sisa_ikhwan : Math.max(j.sisa_ikhwan, j.sisa_akhwat));

	let referalTerbuka = $state(false);
	let adaTentang = $derived(tampil("untuk_siapa") || tampil("riwayat_belajar") || tampil("belajar_dimana") || tampil("waktu_belajar"));

	const inputCls =
		"mt-1.5 block h-12 w-full rounded-xl border border-neutral-300 bg-white px-4 text-base text-neutral-900 placeholder:text-neutral-400 shadow-sm outline-none transition-[border-color,box-shadow] duration-150 focus:border-brand-600 focus:ring-4 focus:ring-brand-500/15 aria-[invalid=true]:border-red-500 aria-[invalid=true]:ring-red-500/15 disabled:opacity-60 dark:border-white/10 dark:bg-neutral-900 dark:text-white dark:placeholder:text-neutral-500";
	const areaCls = inputCls.replace("h-12", "min-h-24 py-3");
	const labelCls = "text-sm font-medium text-neutral-800 dark:text-neutral-200";
	const opsiCls =
		"group relative flex cursor-pointer items-center gap-3 rounded-xl border border-neutral-200 bg-white px-4 py-3.5 text-sm transition-[border-color,background-color,box-shadow] duration-150 hover:border-neutral-300 has-[:checked]:border-brand-600 has-[:checked]:bg-brand-50/70 has-[:checked]:shadow-[0_0_0_1px_var(--color-brand-600)] has-[:focus-visible]:ring-4 has-[:focus-visible]:ring-brand-500/25 dark:border-white/10 dark:bg-neutral-900 dark:hover:border-white/20 dark:has-[:checked]:bg-brand-500/10";
	const legendCls = "text-[15px] font-semibold text-neutral-900 dark:text-white";
	const opsional = (f: string) => (wajib(f) ? "" : " (opsional)");
</script>

{#snippet galat(field: string)}
	{#if errors[field]}<p id="err-{field}" class="mt-1.5 text-sm font-medium text-red-600 dark:text-red-400">{errors[field]}</p>{/if}
{/snippet}

{#snippet tanda()}
	<span class="ml-auto flex h-5 w-5 shrink-0 items-center justify-center rounded-full border border-neutral-300 text-transparent transition group-has-[:checked]:border-brand-600 group-has-[:checked]:bg-brand-600 group-has-[:checked]:text-white dark:border-white/20" aria-hidden="true">
		<Check size={13} strokeWidth={3} />
	</span>
{/snippet}

{#if diri}
	<fieldset class="space-y-5" disabled={preview}>
		<legend class="sr-only">Data diri</legend>
		<div>
			<label for="f-nama" class={labelCls}>Nama lengkap</label>
			<input id="f-nama" bind:value={form.nama} name="nama" autocomplete="name" autocapitalize="words" enterkeyhint="next" required maxlength="80"
				placeholder="Sesuai yang ingin dipanggil"
				aria-invalid={!!errors.nama} aria-describedby={errors.nama ? "err-nama" : undefined} class={inputCls}
				onblur={() => onblurfield?.("nama")} />
			{@render galat("nama")}
		</div>
		<div>
			<label for="f-wa" class={labelCls}>Nomor WhatsApp</label>
			<input id="f-wa" bind:value={form.no_wa} name="no_wa" type="tel" inputmode="tel" autocomplete="tel" enterkeyhint="next" required placeholder="0812 3456 7890"
				aria-invalid={!!errors.no_wa} aria-describedby={errors.no_wa ? "err-no_wa" : "hint-wa"} class={inputCls}
				onblur={() => onblurfield?.("no_wa")} />
			{#if errors.no_wa}{@render galat("no_wa")}{:else}<p id="hint-wa" class="mt-1.5 text-xs text-neutral-500 dark:text-neutral-400">CS kami akan menghubungi Anda lewat nomor ini.</p>{/if}
		</div>
		<div role="radiogroup" aria-labelledby="lbl-jk" aria-describedby={errors.jenis_kelamin ? "err-jenis_kelamin" : undefined}>
			<span id="lbl-jk" class={labelCls}>Mendaftar sebagai</span>
			<div class="mt-1.5 grid grid-cols-2 gap-2.5">
				{#each [["L", "Ikhwan", "Laki-laki"], ["P", "Akhwat", "Perempuan"]] as [val, teks, sub]}
					<label class="{opsiCls} min-h-14">
						<input type="radio" name="jenis_kelamin" value={val} bind:group={form.jenis_kelamin} required class="sr-only" onchange={() => (form.jadwal_kode = "")} />
						<span>
							<span class="block text-base font-semibold text-neutral-900 dark:text-white">{teks}</span>
							<span class="block text-xs text-neutral-500 dark:text-neutral-400">{sub}</span>
						</span>
						{@render tanda()}
					</label>
				{/each}
			</div>
			{@render galat("jenis_kelamin")}
		</div>
	</fieldset>
{/if}

{#if pilihan}
	{#if tampil("usia") || tampil("domisili")}
		<fieldset class="{diri ? 'mt-5' : ''} grid grid-cols-[minmax(0,104px)_minmax(0,1fr)] gap-3" disabled={preview}>
			<legend class="sr-only">Usia dan domisili</legend>
			{#if tampil("usia")}
				<div class={tampil("domisili") ? "" : "col-span-2"}>
					<label for="f-usia" class={labelCls}>Usia<span class="font-normal text-neutral-500">{opsional("usia")}</span></label>
					<input id="f-usia" bind:value={form.usia} name="usia" type="number" inputmode="numeric" min="5" max="90" required={wajib("usia")} placeholder="25"
						aria-invalid={!!errors.usia} aria-describedby={errors.usia ? "err-usia" : undefined} class={inputCls} />
				</div>
			{/if}
			{#if tampil("domisili")}
				<div class={tampil("usia") ? "" : "col-span-2"}>
					<label for="f-domisili" class={labelCls}>Kota domisili<span class="font-normal text-neutral-500">{opsional("domisili")}</span></label>
					<input id="f-domisili" bind:value={form.domisili} name="domisili" autocomplete="address-level2" required={wajib("domisili")} maxlength="80" placeholder="mis. Bekasi"
						aria-invalid={!!errors.domisili} aria-describedby={errors.domisili ? "err-domisili" : undefined} class={inputCls} />
				</div>
			{/if}
			<div class="col-span-2 -mt-1 empty:hidden">{@render galat("usia")}{@render galat("domisili")}</div>
		</fieldset>
	{/if}

	{#if link.varian.length}
		<fieldset class="mt-7" disabled={preview} aria-describedby={errors.varian_id ? "err-varian_id" : undefined}>
			<legend class={legendCls}>Pilih paket</legend>
			<div class="mt-3 space-y-2.5">
				{#each link.varian as v (v.id)}
					<label class={opsiCls}>
						<input type="radio" name="varian_id" value={v.id} bind:group={form.varian_id} required class="sr-only" />
						<span class="min-w-0">
							<span class="block font-semibold text-neutral-900 dark:text-white">{v.nama}</span>
							<span class="mt-0.5 block tabular-nums text-neutral-600 dark:text-neutral-300">{rupiah(v.harga)}</span>
						</span>
						{@render tanda()}
					</label>
				{/each}
			</div>
			{@render galat("varian_id")}
		</fieldset>
	{/if}

	{#if link.butuh_jadwal}
		<fieldset class="mt-7" disabled={preview} aria-describedby={errors.jadwal_kode ? "err-jadwal_kode" : undefined}>
			<legend class={legendCls}>Pilih jadwal kelas</legend>
			{#if !form.jenis_kelamin}
				<p class="mt-2 rounded-xl bg-neutral-100 p-4 text-sm text-neutral-600 dark:bg-white/5 dark:text-neutral-400">Pilih ikhwan atau akhwat dulu untuk melihat jadwal yang tersedia.</p>
			{:else if jadwalSesuai.length === 0}
				<p class="mt-2 rounded-xl bg-amber-500/10 p-4 text-sm text-amber-800 dark:text-amber-300">Belum ada jadwal yang dibuka saat ini. Silakan hubungi CS lewat WhatsApp untuk info jadwal berikutnya.</p>
			{:else}
				<div class="mt-3 space-y-2.5">
					{#each jadwalSesuai as j (j.kode)}
						{@const s = sisa(j)}
						{@const habis = s <= 0}
						<label class="{opsiCls} {habis ? 'pointer-events-none opacity-55' : ''}">
							<input type="radio" name="jadwal_kode" value={j.kode} bind:group={form.jadwal_kode} disabled={habis} required class="sr-only" />
							<span class="min-w-0 flex-1">
								<span class="block font-semibold text-neutral-900 dark:text-white">{j.label}</span>
								{#if j.catatan}<span class="mt-0.5 block text-xs text-neutral-500 dark:text-neutral-400">{j.catatan}</span>{/if}
							</span>
							<span class="shrink-0 rounded-full px-2 py-0.5 text-xs font-semibold tabular-nums {habis ? 'bg-red-500/10 text-red-700 dark:text-red-400' : s <= 3 ? 'bg-amber-500/15 text-amber-800 dark:text-amber-300' : 'bg-neutral-100 text-neutral-600 dark:bg-white/5 dark:text-neutral-400'}">
								{habis ? "Penuh" : s <= 3 ? `Sisa ${s} kursi` : `${s} kursi`}
							</span>
							{@render tanda()}
						</label>
					{/each}
				</div>
			{/if}
			{@render galat("jadwal_kode")}
		</fieldset>
	{/if}

	{#if adaTentang}
		<fieldset class="mt-7 space-y-5" disabled={preview}>
			<legend class={legendCls}>Sedikit tentang Anda</legend>
			<p class="-mt-1 text-sm text-neutral-500 dark:text-neutral-400">Membantu ustadz/ustadzah menyiapkan kelas yang pas.</p>
			{#if tampil("untuk_siapa")}
				<div>
					<label for="f-untuk" class={labelCls}>Mendaftar untuk<span class="font-normal text-neutral-500">{opsional("untuk_siapa")}</span></label>
					<div class="relative">
						<select id="f-untuk" bind:value={form.untuk_siapa} name="untuk_siapa" required={wajib("untuk_siapa")} class="{inputCls} appearance-none pr-10"
							aria-invalid={!!errors.untuk_siapa} aria-describedby={errors.untuk_siapa ? "err-untuk_siapa" : undefined}>
							<option value="">Pilih…</option>
							<option value="diri_sendiri">Diri sendiri</option>
							<option value="anak">Anak</option>
							<option value="keluarga">Anggota keluarga</option>
						</select>
						<ChevronDown class="pointer-events-none absolute right-3.5 top-1/2 mt-[3px] h-4 w-4 -translate-y-1/2 text-neutral-500" aria-hidden="true" />
					</div>
					{@render galat("untuk_siapa")}
				</div>
			{/if}
			{#if tampil("riwayat_belajar")}
				<div>
					<label for="f-riwayat" class={labelCls}>Pengalaman belajar Al-Qur'an<span class="font-normal text-neutral-500">{opsional("riwayat_belajar")}</span></label>
					<textarea id="f-riwayat" bind:value={form.riwayat_belajar} name="riwayat_belajar" rows="3" maxlength="500" required={wajib("riwayat_belajar")} placeholder="mis. sudah lancar Iqra, pernah tahsin 3 bulan" class={areaCls}
						aria-invalid={!!errors.riwayat_belajar} aria-describedby={errors.riwayat_belajar ? "err-riwayat_belajar" : undefined}></textarea>
					{@render galat("riwayat_belajar")}
				</div>
			{/if}
			{#if tampil("belajar_dimana")}
				<div>
					<label for="f-dimana" class={labelCls}>Sebelumnya belajar di mana<span class="font-normal text-neutral-500">{opsional("belajar_dimana")}</span></label>
					<input id="f-dimana" bind:value={form.belajar_dimana} name="belajar_dimana" maxlength="200" required={wajib("belajar_dimana")} class={inputCls}
						aria-invalid={!!errors.belajar_dimana} aria-describedby={errors.belajar_dimana ? "err-belajar_dimana" : undefined} />
					{@render galat("belajar_dimana")}
				</div>
			{/if}
			{#if tampil("waktu_belajar")}
				<div>
					<label for="f-waktu" class={labelCls}>Waktu belajar yang diinginkan<span class="font-normal text-neutral-500">{opsional("waktu_belajar")}</span></label>
					<input id="f-waktu" bind:value={form.waktu_belajar} name="waktu_belajar" maxlength="200" required={wajib("waktu_belajar")} placeholder="mis. malam setelah Isya" class={inputCls}
						aria-invalid={!!errors.waktu_belajar} aria-describedby={errors.waktu_belajar ? "err-waktu_belajar" : undefined} />
					{@render galat("waktu_belajar")}
				</div>
			{/if}
		</fieldset>
	{/if}

	{#if link.kode_referal}
		<div class="mt-6">
			{#if referalTerbuka || form.kode_referal || errors.kode_referal || preview}
				<label for="f-referal" class={labelCls}>Kode referal<span class="font-normal text-neutral-500"> (opsional)</span></label>
				<input id="f-referal" bind:value={form.kode_referal} name="kode_referal" maxlength="20" autocapitalize="characters" placeholder="Dari ustadz/teman yang mengajak" disabled={preview}
					aria-invalid={!!errors.kode_referal} aria-describedby={errors.kode_referal ? "err-kode_referal" : undefined} class="{inputCls} uppercase placeholder:normal-case" />
				{@render galat("kode_referal")}
			{:else}
				<button type="button" class="cursor-pointer rounded text-sm font-medium text-brand-700 underline-offset-4 hover:underline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-600 dark:text-brand-400" onclick={() => (referalTerbuka = true)}>
					Punya kode referal?
				</button>
			{/if}
		</div>
	{/if}

	<!-- Honeypot: tersembunyi dari manusia, diisi bot. -->
	<div class="absolute -left-[9999px] h-px w-px overflow-hidden" aria-hidden="true">
		<label>Website <input bind:value={form.website} name="website" tabindex="-1" autocomplete="off" /></label>
	</div>
{/if}
