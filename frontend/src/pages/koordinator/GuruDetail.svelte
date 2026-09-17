<script lang="ts">
	import { inertia, router } from "@inertiajs/svelte";
	import AppLayout from "@layouts/AppLayout.svelte";
	import type { GuruDirectory, Kompetensi, LinkableUser, Riayah, User } from "@lib/types";
	import { ArrowLeft, BookOpen, Save, Users, Link2, Unlink, Info, Award, NotebookPen } from "lucide-svelte";

	interface Props { user?: User; guru: GuruDirectory; linkableUsers?: LinkableUser[]; canLink?: boolean; kompetensi?: Kompetensi; riayah?: Riayah[]; success?: string; error?: string; }
	let { user, guru, linkableUsers = [], canLink = false, kompetensi, riayah = [], success, error }: Props = $props();

	const KOMP_FIELDS: { key: keyof Kompetensi; label: string }[] = [
		{ key: "hafalan_quran", label: "Hafalan Qur'an" },
		{ key: "hafalan_tuhfah", label: "Hafalan Tuhfah" },
		{ key: "hafalan_jazariy", label: "Hafalan Jazariy" },
		{ key: "hafalan_khaqaniy", label: "Hafalan Khaqaniy" },
		{ key: "hafalan_syakhawiy", label: "Hafalan Syakhawiy" },
		{ key: "sanad_qiroah", label: "Sanad Qiro'ah" },
		{ key: "bahasa_arab_pasif", label: "Bahasa Arab Pasif" },
		{ key: "bahasa_arab_aktif", label: "Bahasa Arab Aktif" },
	];
	const LEVELS = ["Belum", "Dasar", "Mahir", "Bersanad"];
	let komp = $state<Kompetensi>(kompetensi ?? { hafalan_quran: 0, hafalan_tuhfah: 0, hafalan_jazariy: 0, hafalan_khaqaniy: 0, hafalan_syakhawiy: 0, sanad_qiroah: 0, bahasa_arab_pasif: 0, bahasa_arab_aktif: 0 });
	let savingKomp = $state(false);
	function saveKomp() { savingKomp = true; router.put(`/app/koordinator-guru/guru/${guru.id}/kompetensi`, komp, { preserveScroll: true, onFinish: () => (savingKomp = false) }); }

	let catatan = $state("");
	let savingRiayah = $state(false);
	function addRiayah() {
		if (!catatan.trim()) return;
		savingRiayah = true;
		router.post(`/app/koordinator-guru/guru/${guru.id}/riayah`, { catatan }, { preserveScroll: true, onSuccess: () => { catatan = ""; }, onFinish: () => (savingRiayah = false) });
	}
	function formatRiayahDate(value: string): string {
		if (!value) return "Tanggal tidak tersedia";
		const date = new Date(value.replace(" ", "T"));
		if (Number.isNaN(date.getTime())) return value;
		return new Intl.DateTimeFormat("id-ID", { dateStyle: "long", timeStyle: "short" }).format(date);
	}
	let form = $state({ nama: guru.nama, gelar: guru.gelar, jenis_kelamin: guru.jenis_kelamin, status: guru.status, no_wa: guru.no_wa, email: guru.email, tanggal_gabung: guru.tanggal_gabung, foto: guru.foto, is_aktif: guru.is_aktif });
	let saving = $state(false);
	const inputClass = "w-full px-3.5 py-2.5 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700 text-sm text-neutral-900 dark:text-white outline-none focus:border-brand-400";
	function save() { saving = true; router.put(`/app/koordinator-guru/guru/${guru.id}`, form, { preserveScroll: true, onFinish: () => saving = false }); }

	let selectedUser = $state<number | "">("");
	let linking = $state(false);
	function linkAccount() {
		if (!selectedUser) return;
		linking = true;
		router.put(`/app/koordinator-guru/guru/${guru.id}/link`, { user_id: Number(selectedUser) }, { preserveScroll: true, onFinish: () => (linking = false) });
	}
	function unlinkAccount() {
		if (!confirm("Lepas akun login dari data guru ini? Guru tidak akan bisa login sampai dihubungkan kembali.")) return;
		linking = true;
		router.delete(`/app/koordinator-guru/guru/${guru.id}/link`, { preserveScroll: true, onFinish: () => (linking = false) });
	}
</script>

<AppLayout {user} group="koordinator-guru">
	<div class="pt-8 pb-10 border-b border-neutral-200/80 dark:border-white/[0.04]"><div class="max-w-4xl mx-auto px-4 sm:px-6"><a href="/app/koordinator-guru/guru" use:inertia class="inline-flex items-center gap-1.5 text-sm text-neutral-500 hover:text-brand-600 dark:hover:text-brand-400 mb-5"><ArrowLeft class="w-4 h-4" /> Data Guru</a><h1 class="text-2xl sm:text-3xl font-bold text-neutral-900 dark:text-white">{guru.nama}</h1><p class="mt-2 text-neutral-600 dark:text-neutral-400">Kelola informasi profil dan status penugasan guru.</p></div></div>
	<div class="max-w-4xl mx-auto px-4 sm:px-6 py-8 space-y-6">
		{#if success}<div class="rounded-xl bg-green-500/10 border border-green-500/20 p-4 text-sm font-medium text-green-700 dark:text-green-400">{success}</div>{/if}
		{#if error}<div class="rounded-xl bg-red-500/10 border border-red-500/20 p-4 text-sm font-medium text-red-700 dark:text-red-400">{error}</div>{/if}
		<div class="grid sm:grid-cols-3 gap-4"><div class="rounded-2xl bg-brand-400/10 border border-brand-400/20 p-5"><BookOpen class="w-5 h-5 text-brand-600 dark:text-brand-300" /><p class="mt-3 text-2xl font-bold text-neutral-900 dark:text-white">{guru.total_kelas}</p><p class="text-sm text-neutral-600 dark:text-neutral-400">Kelas aktif</p></div><div class="rounded-2xl bg-secondary-500/10 border border-secondary-500/20 p-5"><Users class="w-5 h-5 text-secondary-600 dark:text-secondary-300" /><p class="mt-3 text-2xl font-bold text-neutral-900 dark:text-white">{guru.total_santri}</p><p class="text-sm text-neutral-600 dark:text-neutral-400">Santri aktif</p></div><div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-5"><p class="text-sm text-neutral-500">Akun login</p><p class="mt-3 text-sm font-semibold {guru.user_id ? 'text-green-600 dark:text-green-400' : 'text-amber-600 dark:text-amber-400'}">{guru.user_id ? `Terhubung (#${guru.user_id})` : "Belum terhubung"}</p></div></div>

		<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-5 sm:p-6 space-y-4">
			<div class="flex items-center gap-2"><Link2 class="w-5 h-5 text-brand-600 dark:text-brand-300" /><h2 class="font-semibold text-neutral-900 dark:text-white">Akun Login</h2></div>
			{#if guru.user_id}
				<div class="flex items-center justify-between gap-4 flex-wrap rounded-xl bg-green-500/5 border border-green-500/20 p-4">
					<div><p class="text-sm font-semibold text-neutral-900 dark:text-white">Terhubung dengan akun #{guru.user_id}</p><p class="mt-0.5 text-xs text-neutral-500 dark:text-neutral-400">Guru dapat login dan melihat kelas yang ditugaskan.</p></div>
					{#if canLink}
						<button onclick={unlinkAccount} disabled={linking} class="inline-flex items-center gap-2 px-4 py-2 rounded-xl bg-red-500/10 hover:bg-red-500/20 text-red-600 dark:text-red-400 text-sm font-semibold disabled:opacity-50"><Unlink class="w-4 h-4" /> Lepas akun</button>
					{/if}
				</div>
			{:else if canLink}
				<p class="text-sm text-neutral-600 dark:text-neutral-400">Hubungkan data guru ini ke sebuah akun login. Hanya akun yang belum terhubung ke guru lain yang ditampilkan.</p>
				<div class="flex flex-col sm:flex-row gap-3">
					<select bind:value={selectedUser} class={inputClass}>
						<option value="">Pilih akun...</option>
						{#each linkableUsers as u (u.id)}
							<option value={u.id}>{u.name} — {u.email} ({u.role})</option>
						{/each}
					</select>
					<button onclick={linkAccount} disabled={linking || !selectedUser} class="inline-flex items-center justify-center gap-2 px-5 py-2.5 rounded-xl bg-brand-600 hover:bg-brand-700 text-white text-sm font-semibold whitespace-nowrap disabled:opacity-50"><Link2 class="w-4 h-4" /> {linking ? "Menghubungkan..." : "Hubungkan"}</button>
				</div>
				<div class="flex items-start gap-2 rounded-xl bg-brand-400/5 border border-brand-400/20 p-3 text-xs text-neutral-600 dark:text-neutral-400"><Info class="w-4 h-4 shrink-0 text-brand-500 mt-0.5" /><span>Setelah dihubungkan, buka <a href="/admin/users" use:inertia class="font-semibold text-brand-600 dark:text-brand-400 hover:underline">Kelola User</a> dan ubah role akun tersebut menjadi <strong>Guru</strong> agar bisa mengakses menu guru.</span></div>
			{:else}
				<div class="flex items-start gap-2 rounded-xl bg-amber-500/5 border border-amber-500/20 p-3 text-xs text-neutral-600 dark:text-neutral-400"><Info class="w-4 h-4 shrink-0 text-amber-500 mt-0.5" /><span>Belum terhubung ke akun login. Hanya Super Admin yang dapat menghubungkan akun ke data guru.</span></div>
			{/if}
		</div>

		<form onsubmit={(event) => { event.preventDefault(); save(); }} class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-5 sm:p-6 space-y-5"><div class="grid sm:grid-cols-2 gap-4"><label class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Nama<input bind:value={form.nama} required class={`${inputClass} mt-1.5`} /></label><label class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Gelar<input bind:value={form.gelar} placeholder="S.Pd., Lc., dll." class={`${inputClass} mt-1.5`} /></label><label class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Jenis kelamin<select bind:value={form.jenis_kelamin} class={`${inputClass} mt-1.5`}><option value="L">Laki-laki</option><option value="P">Perempuan</option></select></label><label class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Status<select bind:value={form.status} class={`${inputClass} mt-1.5`}><option value="tetap">Guru Tetap</option><option value="part_time">Guru Part Time</option></select></label><label class="text-sm font-medium text-neutral-700 dark:text-neutral-300">No. WhatsApp<input bind:value={form.no_wa} class={`${inputClass} mt-1.5`} /></label><label class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Email<input bind:value={form.email} type="email" class={`${inputClass} mt-1.5`} /></label><label class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Tanggal bergabung<input bind:value={form.tanggal_gabung} type="date" class={`${inputClass} mt-1.5`} /></label><label class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Status akun<select bind:value={form.is_aktif} class={`${inputClass} mt-1.5`}><option value={true}>Aktif</option><option value={false}>Nonaktif</option></select></label></div><label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300">URL foto<input bind:value={form.foto} type="url" placeholder="https://..." class={`${inputClass} mt-1.5`} /></label><div class="pt-4 border-t border-neutral-200/80 dark:border-white/[0.04] flex justify-end"><button type="submit" disabled={saving} class="inline-flex items-center gap-2 px-5 py-2.5 rounded-xl bg-brand-600 hover:bg-brand-700 text-white text-sm font-semibold disabled:opacity-50"><Save class="w-4 h-4" /> {saving ? "Menyimpan..." : "Simpan perubahan"}</button></div></form>

		<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-5 sm:p-6 space-y-4">
			<div class="flex items-center gap-2"><Award class="w-5 h-5 text-brand-600 dark:text-brand-300" /><h2 class="font-semibold text-neutral-900 dark:text-white">Matriks Kompetensi</h2></div>
			<p class="text-sm text-neutral-600 dark:text-neutral-400">Skala 0–3 per bidang, dipakai untuk rekomendasi penempatan guru ke kelas.</p>
			<div class="grid sm:grid-cols-2 gap-4">
				{#each KOMP_FIELDS as f (f.key)}
					<label class="text-sm font-medium text-neutral-700 dark:text-neutral-300 flex items-center justify-between gap-3">
						{f.label}
						<select bind:value={komp[f.key]} class="px-3 py-2 rounded-lg bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700 text-sm outline-none focus:border-brand-400">
							{#each LEVELS as lv, i}<option value={i}>{lv}</option>{/each}
						</select>
					</label>
				{/each}
			</div>
			<div class="pt-4 border-t border-neutral-200/80 dark:border-white/[0.04] flex justify-end"><button onclick={saveKomp} disabled={savingKomp} class="inline-flex items-center gap-2 px-5 py-2.5 rounded-xl bg-brand-600 hover:bg-brand-700 text-white text-sm font-semibold disabled:opacity-50"><Save class="w-4 h-4" /> {savingKomp ? "Menyimpan..." : "Simpan kompetensi"}</button></div>
		</div>

		<section class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 overflow-hidden">
			<div class="p-5 sm:p-6 border-b border-neutral-200/80 dark:border-white/[0.04]">
				<div class="flex items-center justify-between gap-3"><div class="flex items-center gap-2"><NotebookPen class="w-5 h-5 text-brand-600 dark:text-brand-300" /><h2 class="font-semibold text-neutral-900 dark:text-white">Catatan Riayah Guru</h2></div><span class="text-xs font-medium text-neutral-500">{riayah.length} catatan</span></div>
				<p class="mt-2 text-sm text-neutral-600 dark:text-neutral-400">Catat pembinaan, komunikasi, atau tindak lanjut guru. Catatan tersimpan beserta waktu pencatatannya.</p>
				<div class="mt-4 space-y-3"><textarea bind:value={catatan} rows="3" placeholder="Tulis catatan pembinaan / komunikasi dengan guru..." class={`${inputClass} resize-y`}></textarea><div class="flex justify-end"><button onclick={addRiayah} disabled={savingRiayah || !catatan.trim()} class="inline-flex items-center gap-2 px-5 py-2.5 rounded-xl bg-brand-600 hover:bg-brand-700 text-white text-sm font-semibold disabled:opacity-50"><Save class="w-4 h-4" /> {savingRiayah ? "Menyimpan..." : "Simpan catatan"}</button></div></div>
			</div>
			<div class="p-5 sm:p-6"><h3 class="text-sm font-semibold text-neutral-900 dark:text-white mb-3">Riwayat Catatan</h3><div class="space-y-3">{#each riayah as r (r.id)}<article class="relative border-l-2 border-brand-400/40 pl-4 py-1"><p class="text-sm text-neutral-800 dark:text-neutral-200 whitespace-pre-wrap">{r.catatan}</p><p class="mt-2 text-xs text-neutral-500">Dicatat {formatRiayahDate(r.created_at)}{#if r.penulis_nama} · oleh {r.penulis_nama}{/if}</p></article>{:else}<p class="rounded-xl bg-neutral-100/60 dark:bg-neutral-800/40 p-4 text-sm text-neutral-500">Belum ada catatan riayah untuk guru ini.</p>{/each}</div></div>
		</section>
	</div>
</AppLayout>
