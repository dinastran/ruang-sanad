<script lang="ts">
	import { inertia, router } from "@inertiajs/svelte";
	import { fly } from "svelte/transition";
	import AppLayout from "@layouts/AppLayout.svelte";
	import type { GuruRingkas, KelasSimple, Kunjungan, User } from "@lib/types";
	import { Send, Lock, LockOpen, MessageCircle, Trash2, CheckCircle2, RotateCcw, Plus } from "lucide-svelte";

	interface Props { user?: User; kunjungan: Kunjungan; guruList?: GuruRingkas[]; kelasList?: KelasSimple[]; success?: string; error?: string; }
	let { user, kunjungan: k, guruList = [], kelasList = [], success, error }: Props = $props();

	let base = $derived(`/app/koordinator-guru/kunjungan/${k.id}`);
	let terkunci = $derived(k.status_kirim !== "draft");
	const inputClass = "w-full px-3.5 py-2.5 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700 text-sm text-neutral-900 dark:text-white outline-none focus:border-brand-400 disabled:opacity-70";
	const SKALA = [
		{ nilai: 1, label: "Perlu perbaikan" },
		{ nilai: 2, label: "Cukup" },
		{ nilai: 3, label: "Baik" },
		{ nilai: 4, label: "Sangat baik" },
	];
	const KIRIM: Record<string, string> = { draft: "Belum dikirim", terkirim: "Terkirim · belum dibaca", dibaca: "Dibaca guru", ditanggapi: "Ditanggapi guru" };
	const JENIS = [
		{ value: "apresiasi", label: "Apresiasi & reward" },
		{ value: "monitoring", label: "Monitoring kunjungan berikutnya" },
		{ value: "coaching", label: "Coaching" },
		{ value: "pembinaan", label: "Pembinaan" },
		{ value: "koordinasi", label: "Koordinasi dengan pihak terkait" },
	];

	type NilaiKey = "nilai_kedisiplinan" | "nilai_materi" | "nilai_metode" | "nilai_interaksi";
	const nilaiKey = (kode: string) => `nilai_${kode}` as NilaiKey;

	let form = $state({
		guru_id: k.guru_id,
		kelas_id: k.kelas_id ?? 0,
		target_mulai: k.target_mulai,
		target_selesai: k.target_selesai,
		tanggal: k.tanggal,
		jam: k.jam,
		status: k.status,
		catatan: k.catatan,
		nilai_kedisiplinan: k.nilai_kedisiplinan,
		nilai_materi: k.nilai_materi,
		nilai_metode: k.nilai_metode,
		nilai_interaksi: k.nilai_interaksi,
	});
	let saving = $state(false);
	let busy = $state(false);

	let nilaiDiisi = $derived([form.nilai_kedisiplinan, form.nilai_materi, form.nilai_metode, form.nilai_interaksi]);
	let rataRata = $derived(nilaiDiisi.every((n) => n > 0) ? nilaiDiisi.reduce((a, b) => a + b, 0) / nilaiDiisi.length : 0);
	let dirty = $derived(JSON.stringify(form) !== JSON.stringify({
		guru_id: k.guru_id, kelas_id: k.kelas_id ?? 0, target_mulai: k.target_mulai, target_selesai: k.target_selesai,
		tanggal: k.tanggal, jam: k.jam, status: k.status, catatan: k.catatan,
		nilai_kedisiplinan: k.nilai_kedisiplinan, nilai_materi: k.nilai_materi, nilai_metode: k.nilai_metode, nilai_interaksi: k.nilai_interaksi,
	}));
	let bisaKirim = $derived(!terkunci && !dirty && k.status === "terlaksana" && k.nilai_lengkap && !!k.catatan.trim());

	function predikat(n: number): string {
		if (n <= 0) return "";
		if (n >= 3.5) return "Sangat baik";
		if (n >= 2.5) return "Baik";
		if (n >= 1.5) return "Cukup";
		return "Perlu perbaikan";
	}
	const fmt = (n: number) => n.toFixed(2).replace(".", ",");

	function simpan() {
		saving = true;
		router.put(base, form, { preserveScroll: true, onFinish: () => { saving = false; } });
	}
	function kirim() {
		if (!confirm(`Kirim hasil kunjungan ke ${k.guru_nama}? Nilai dan catatan akan dikunci.`)) return;
		busy = true;
		router.post(`${base}/kirim`, {}, { preserveScroll: true, onFinish: () => { busy = false; } });
	}
	function bukaKunci() {
		if (!confirm("Buka kunci untuk mengubah hasil? Setelah diubah, kirim ulang agar guru menerima versi terbaru.")) return;
		busy = true;
		router.post(`${base}/buka-kunci`, {}, { preserveScroll: true, onFinish: () => { busy = false; } });
	}
	function hapus() {
		if (confirm("Hapus kunjungan ini?")) router.delete(base);
	}

	let tl = $state({ jenis: "apresiasi", catatan: "", target_tanggal: "" });
	let savingTL = $state(false);
	function tambahTL() {
		savingTL = true;
		router.post(`${base}/tindak-lanjut`, tl, { preserveScroll: true, onSuccess: () => { tl = { jenis: "apresiasi", catatan: "", target_tanggal: "" }; }, onFinish: () => { savingTL = false; } });
	}
	function setStatusTL(id: number, status: "selesai" | "terbuka") {
		router.put(`${base}/tindak-lanjut/${id}`, { status }, { preserveScroll: true });
	}
	function hapusTL(id: number, monitoring: boolean) {
		const pesan = monitoring ? "Hapus tindak lanjut ini? Jadwal kunjungan monitoring yang belum dilaksanakan ikut dihapus." : "Hapus tindak lanjut ini?";
		if (confirm(pesan)) router.delete(`${base}/tindak-lanjut/${id}`, { preserveScroll: true });
	}
</script>

<AppLayout {user} group="koordinator-kunjungan">
	<div class="pt-8 pb-8 border-b border-neutral-200/80 dark:border-white/[0.04]">
		<div class="max-w-4xl mx-auto px-4 sm:px-6">
			<div class="flex items-center gap-2 text-sm text-neutral-500 mb-3"><a href="/app/koordinator-guru/kunjungan" use:inertia class="hover:text-brand-600">Kunjungan Kelas</a><span>/</span><span class="text-neutral-700 dark:text-neutral-300">{k.guru_nama}</span></div>
			<div class="flex items-end justify-between gap-4 flex-wrap">
				<div>
					<h1 class="text-2xl sm:text-3xl font-bold text-neutral-900 dark:text-white tracking-tight">{k.guru_nama}</h1>
					<p class="mt-1 text-neutral-600 dark:text-neutral-400">{k.kelas_nama || "Kelas belum dipilih"}{k.tanggal ? ` · ${k.tanggal} ${k.jam}` : ""}</p>
					{#if k.status === "terlaksana"}
						<p class="mt-2 inline-flex items-center gap-1.5 text-sm font-medium {terkunci ? 'text-green-700 dark:text-green-400' : 'text-neutral-500'}">{#if terkunci}<Lock class="w-4 h-4" />{/if}{KIRIM[k.status_kirim]}{k.dikirim_at ? ` · dikirim ${k.dikirim_at}` : ""}</p>
					{/if}
				</div>
				<div class="flex items-center gap-2 flex-wrap">
					{#if k.wa_link}<a href={k.wa_link} target="_blank" rel="noopener" class="inline-flex items-center gap-2 px-3.5 py-2.5 rounded-xl border border-green-600/30 text-green-700 dark:text-green-400 text-sm font-semibold hover:bg-green-500/10"><MessageCircle class="w-4 h-4" /> Kirim via WA</a>{/if}
					{#if terkunci}
						<button onclick={bukaKunci} disabled={busy} class="inline-flex items-center gap-2 px-3.5 py-2.5 rounded-xl border border-neutral-300 dark:border-neutral-700 text-sm font-semibold text-neutral-700 dark:text-neutral-300 hover:bg-neutral-100 dark:hover:bg-neutral-800 disabled:opacity-50"><LockOpen class="w-4 h-4" /> Buka kunci</button>
					{:else}
						{#if k.status === "terlaksana"}
							<button onclick={kirim} disabled={!bisaKirim || busy} title={dirty ? "Simpan perubahan dulu" : !k.nilai_lengkap ? "Lengkapi nilai semua aspek" : ""} class="inline-flex items-center gap-2 px-4 py-2.5 rounded-xl bg-brand-600 hover:bg-brand-700 text-white text-sm font-semibold disabled:opacity-50 disabled:cursor-not-allowed"><Send class="w-4 h-4" /> Kirim ke guru</button>
						{/if}
						<button onclick={hapus} aria-label="Hapus kunjungan" class="p-2.5 rounded-xl hover:bg-red-500/10 text-red-500"><Trash2 class="w-4 h-4" /></button>
					{/if}
				</div>
			</div>
		</div>
	</div>

	<div class="max-w-4xl mx-auto px-4 sm:px-6 py-8 space-y-6">
		{#if success}<div class="rounded-xl bg-green-500/10 border border-green-500/20 p-4 text-sm font-medium text-green-700 dark:text-green-400" in:fly={{ y: 10, duration: 200 }}>{success}</div>{/if}
		{#if error}<div class="rounded-xl bg-red-500/10 border border-red-500/20 p-4 text-sm font-medium text-red-700 dark:text-red-400">{error}</div>{/if}
		{#if !terkunci && k.status === "terlaksana" && !k.nilai_lengkap}
			<div class="rounded-xl border border-amber-500/20 bg-amber-500/5 p-4 text-sm text-amber-700 dark:text-amber-400">Lengkapi nilai keempat aspek dan catatan, simpan, lalu klik Kirim ke guru.</div>
		{/if}

		<form onsubmit={(e) => { e.preventDefault(); simpan(); }} class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50">
			<fieldset disabled={terkunci} class="p-5 sm:p-6 space-y-6">
				<legend class="sr-only">Pelaksanaan dan penilaian</legend>
				<div>
					<h2 class="text-base font-semibold text-neutral-900 dark:text-white">Pelaksanaan</h2>
					<div class="mt-4 grid sm:grid-cols-2 gap-4">
						<label class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Guru<select bind:value={form.guru_id} required class={`${inputClass} mt-1.5`}>{#each guruList as g (g.id)}<option value={g.id}>{g.nama}</option>{/each}</select></label>
						<label class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Kelas<select bind:value={form.kelas_id} class={`${inputClass} mt-1.5`}><option value={0}>—</option>{#each kelasList as kl (kl.id)}<option value={kl.id}>{kl.nama_kelas}</option>{/each}</select></label>
						<label class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Target mulai<input type="date" bind:value={form.target_mulai} class={`${inputClass} mt-1.5`} /></label>
						<label class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Target selesai<input type="date" bind:value={form.target_selesai} class={`${inputClass} mt-1.5`} /></label>
						<label class="text-sm font-medium text-neutral-700 dark:text-neutral-300 sm:col-span-2">Status<select bind:value={form.status} class={`${inputClass} mt-1.5`}><option value="dijadwalkan">Dijadwalkan</option><option value="terlaksana">Terlaksana</option><option value="ditunda">Ditunda</option><option value="batal">Batal</option></select></label>
						{#if form.status === "terlaksana"}
							<label class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Tanggal aktual<input type="date" bind:value={form.tanggal} required class={`${inputClass} mt-1.5`} /></label>
							<label class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Jam masuk<input type="time" bind:value={form.jam} required class={`${inputClass} mt-1.5`} /></label>
						{/if}
					</div>
				</div>

				{#if form.status === "terlaksana"}
					<div class="pt-6 border-t border-neutral-200/80 dark:border-white/[0.05]">
						<div class="flex items-end justify-between gap-4 flex-wrap">
							<div>
								<h2 class="text-base font-semibold text-neutral-900 dark:text-white">Penilaian</h2>
								<p class="mt-0.5 text-xs text-neutral-500">Skala 1–4 per aspek. Nilai akhir = rata-rata.</p>
							</div>
							{#if rataRata > 0}<p class="text-sm text-neutral-600 dark:text-neutral-400">Rata-rata <span class="text-lg font-bold tabular-nums text-neutral-900 dark:text-white">{fmt(rataRata)}</span> · {predikat(rataRata)}</p>{/if}
						</div>
						<div class="mt-4 divide-y divide-neutral-200/80 dark:divide-white/[0.05]">
							{#each k.aspek as a (a.kode)}
								<div class="py-3 grid sm:grid-cols-[minmax(0,1fr)_auto] gap-2 sm:items-center">
									<p class="text-sm font-medium text-neutral-800 dark:text-neutral-200">{a.label}</p>
									<div class="grid grid-cols-4 gap-1.5" role="radiogroup" aria-label={a.label}>
										{#each SKALA as s (s.nilai)}
											<label class="cursor-pointer has-[:disabled]:cursor-default">
												<input type="radio" class="peer sr-only" name={a.kode} value={s.nilai} bind:group={form[nilaiKey(a.kode)]} />
												<span title={s.label} class="flex flex-col items-center justify-center w-full sm:w-20 rounded-lg border px-2 py-1.5 text-center border-neutral-300 dark:border-neutral-700 text-neutral-600 dark:text-neutral-400 peer-checked:border-brand-600 peer-checked:bg-brand-600 peer-checked:text-white peer-focus-visible:ring-2 peer-focus-visible:ring-brand-400/40">
													<span class="text-sm font-bold">{s.nilai}</span>
													<span class="text-[11px] leading-tight">{s.label}</span>
												</span>
											</label>
										{/each}
									</div>
								</div>
							{/each}
						</div>
					</div>
				{/if}

				<label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300">Catatan kunjungan{#if form.status === "terlaksana"}<span class="text-error"> *</span>{/if}
					<span class="block text-xs font-normal text-neutral-500">Dikirim ke guru sebagai feedback: hal yang sudah baik dan yang perlu diperbaiki.</span>
					<textarea bind:value={form.catatan} required={form.status === "terlaksana"} rows="5" class={`${inputClass} mt-1.5`}></textarea>
				</label>
			</fieldset>
			{#if !terkunci}
				<div class="px-5 sm:px-6 py-4 border-t border-neutral-200/80 dark:border-white/[0.04] flex items-center justify-end gap-3">
					{#if dirty}<span class="text-xs text-amber-700 dark:text-amber-400">Ada perubahan belum disimpan</span>{/if}
					<button type="submit" disabled={saving || !dirty} class="px-5 py-2.5 rounded-xl bg-neutral-900 hover:bg-neutral-800 dark:bg-white dark:text-neutral-900 dark:hover:bg-neutral-200 text-white text-sm font-semibold disabled:opacity-40">{saving ? "Menyimpan..." : "Simpan"}</button>
				</div>
			{/if}
		</form>

		{#if k.status_kirim !== "draft"}
			<section class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-5 sm:p-6">
				<h2 class="text-base font-semibold text-neutral-900 dark:text-white">Respons guru</h2>
				<dl class="mt-3 grid sm:grid-cols-3 gap-3 text-sm">
					<div><dt class="text-xs text-neutral-500">Dikirim</dt><dd class="text-neutral-800 dark:text-neutral-200">{k.dikirim_at || "-"}</dd></div>
					<div><dt class="text-xs text-neutral-500">Dibaca</dt><dd class="text-neutral-800 dark:text-neutral-200">{k.dibaca_at || "Belum dibaca"}</dd></div>
					<div><dt class="text-xs text-neutral-500">Ditanggapi</dt><dd class="text-neutral-800 dark:text-neutral-200">{k.tanggapan_at || "Belum"}</dd></div>
				</dl>
				{#if k.tanggapan_guru}<blockquote class="mt-4 border-l-2 border-brand-400 pl-4 text-sm text-neutral-700 dark:text-neutral-300 whitespace-pre-line">{k.tanggapan_guru}</blockquote>{/if}
			</section>
		{/if}

		{#if k.status === "terlaksana"}
			<section class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-5 sm:p-6">
				<h2 class="text-base font-semibold text-neutral-900 dark:text-white">Tindak lanjut</h2>
				<p class="mt-0.5 text-xs text-neutral-500">Koordinasi dengan pihak terkait hanya terlihat oleh Koordinator. Tindak lanjut lain tampil di halaman hasil kunjungan guru.</p>

				<ul class="mt-4 divide-y divide-neutral-200/80 dark:divide-white/[0.05]">
					{#each k.tindak_lanjut as t (t.id)}
						<li class="py-3 flex items-start justify-between gap-3 flex-wrap">
							<div class="min-w-0 flex-1">
								<p class="text-sm font-semibold {t.status === 'selesai' ? 'text-neutral-400 line-through' : 'text-neutral-900 dark:text-white'}">{t.jenis_label}
									{#if t.internal}<span class="ml-1 no-underline inline-flex px-1.5 py-0.5 rounded text-[11px] font-medium bg-neutral-500/10 text-neutral-600 dark:text-neutral-400">Internal</span>{/if}
									{#if t.terlambat}<span class="ml-1 inline-flex px-1.5 py-0.5 rounded text-[11px] font-medium bg-red-500/10 text-red-600 dark:text-red-400">Lewat target</span>{/if}
								</p>
								{#if t.catatan}<p class="mt-0.5 text-sm text-neutral-600 dark:text-neutral-400">{t.catatan}</p>{/if}
								<p class="mt-0.5 text-xs text-neutral-500">
									{t.target_tanggal ? `Target ${t.target_tanggal}` : "Tanpa target"}{t.status === "selesai" && t.selesai_at ? ` · selesai ${t.selesai_at}` : ""}
									{#if t.kunjungan_berikutnya_id} · <a href={`/app/koordinator-guru/kunjungan/${t.kunjungan_berikutnya_id}`} use:inertia class="text-brand-600 hover:underline dark:text-brand-400">kunjungan monitoring</a>{/if}
								</p>
							</div>
							<div class="flex items-center gap-1">
								{#if t.status === "terbuka"}
									<button onclick={() => setStatusTL(t.id, "selesai")} class="inline-flex items-center gap-1 px-2.5 py-1.5 rounded-lg text-xs font-semibold text-green-700 hover:bg-green-500/10 dark:text-green-400"><CheckCircle2 class="w-3.5 h-3.5" /> Selesai</button>
								{:else}
									<button onclick={() => setStatusTL(t.id, "terbuka")} class="inline-flex items-center gap-1 px-2.5 py-1.5 rounded-lg text-xs font-semibold text-neutral-600 hover:bg-neutral-100 dark:text-neutral-300 dark:hover:bg-neutral-800"><RotateCcw class="w-3.5 h-3.5" /> Buka lagi</button>
								{/if}
								<button onclick={() => hapusTL(t.id, t.jenis === "monitoring")} aria-label="Hapus tindak lanjut" class="p-1.5 rounded-lg hover:bg-red-500/10 text-red-500"><Trash2 class="w-3.5 h-3.5" /></button>
							</div>
						</li>
					{:else}
						<li class="py-3 text-sm text-neutral-500">Belum ada tindak lanjut.</li>
					{/each}
				</ul>

				<form onsubmit={(e) => { e.preventDefault(); tambahTL(); }} class="mt-4 pt-4 border-t border-neutral-200/80 dark:border-white/[0.05] grid sm:grid-cols-2 gap-3">
					<label class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Jenis<select bind:value={tl.jenis} class={`${inputClass} mt-1.5`}>{#each JENIS as j (j.value)}<option value={j.value}>{j.label}</option>{/each}</select></label>
					<label class="text-sm font-medium text-neutral-700 dark:text-neutral-300">{tl.jenis === "monitoring" ? "Tanggal kunjungan berikutnya" : "Target tanggal"}{#if tl.jenis === "monitoring"}<span class="text-error"> *</span>{/if}<input type="date" bind:value={tl.target_tanggal} required={tl.jenis === "monitoring"} class={`${inputClass} mt-1.5`} /></label>
					<label class="text-sm font-medium text-neutral-700 dark:text-neutral-300 sm:col-span-2">Catatan<input bind:value={tl.catatan} maxlength="1000" placeholder={tl.jenis === "monitoring" ? "Fokus yang dicek pada kunjungan berikutnya" : "Detail tindak lanjut"} class={`${inputClass} mt-1.5`} /></label>
					{#if tl.jenis === "monitoring"}<p class="sm:col-span-2 text-xs text-neutral-500">Kunjungan baru berstatus Dijadwalkan dibuat otomatis untuk guru dan kelas yang sama. Tindak lanjut ini selesai otomatis saat hasil kunjungan tersebut dikirim.</p>{/if}
					{#if tl.jenis === "koordinasi"}<p class="sm:col-span-2 text-xs text-neutral-500">Tidak ditampilkan ke guru.</p>{/if}
					<div class="sm:col-span-2 flex justify-end">
						<button type="submit" disabled={savingTL} class="inline-flex items-center gap-2 px-4 py-2.5 rounded-xl border border-brand-600/40 text-brand-700 dark:text-brand-400 text-sm font-semibold hover:bg-brand-400/10 disabled:opacity-50"><Plus class="w-4 h-4" /> Tambah tindak lanjut</button>
					</div>
				</form>
			</section>
		{/if}
	</div>
</AppLayout>
