<script lang="ts">
	import { router, inertia } from "@inertiajs/svelte";
	import Logo from "@components/Logo.svelte";
	import type { User } from "@lib/types";
	import { CircleAlert, Eye, EyeOff, PencilLine, ShieldCheck, ChevronDown } from "lucide-svelte";

	interface ToolInfo {
		name: string;
		title: string;
		write: boolean;
	}

	interface Props {
		user?: User | null;
		galat?: string;
		role?: string;
		diizinkan?: boolean;
		boleh_tulis?: boolean;
		tools?: ToolInfo[];
		klien?: { nama: string; redirect_host: string };
		permintaan?: { client_id: string; redirect_uri: string; code_challenge: string; state: string; resource: string };
	}

	let { user = null, galat = "", role = "", diizinkan = false, boleh_tulis = false, tools = [], klien, permintaan }: Props = $props();

	const roleLabel: Record<string, string> = {
		super_admin: "Super Admin",
		admin: "Admin",
		keuangan: "Keuangan",
		cs: "CS",
		admin_kelas: "Admin Kelas",
		koordinator_guru: "Koordinator Guru",
		guru: "Guru",
		user: "Pengguna",
	};

	let izinTulis = $state(false);
	let mengirim = $state<"izinkan" | "tolak" | null>(null);
	let lihatFitur = $state(false);

	let fiturBaca = $derived(tools.filter((t) => !t.write));
	let fiturTulis = $derived(tools.filter((t) => t.write));

	function putuskan(izinkan: boolean) {
		if (!permintaan || mengirim) return;
		mengirim = izinkan ? "izinkan" : "tolak";
		router.post(
			"/oauth/authorize",
			{ ...permintaan, izinkan, can_write: izinkan && boleh_tulis && izinTulis },
			{ onFinish: () => (mengirim = null) },
		);
	}
</script>

<svelte:head><title>Hubungkan {klien?.nama ?? "aplikasi AI"} · Ruang Sanad</title></svelte:head>

<main class="min-h-screen bg-neutral-50 dark:bg-neutral-950 px-4 py-10 sm:py-16">
	<div class="mx-auto w-full max-w-md">
		<div class="flex items-center gap-2.5 mb-8">
			<Logo size={32} />
			<span class="font-semibold text-neutral-900 dark:text-white">Ruang Sanad</span>
		</div>

		{#if galat || !klien || !permintaan}
			<section class="rounded-2xl border border-red-500/25 bg-white dark:bg-neutral-925/50 p-6" role="alert">
				<div class="flex gap-3">
					<CircleAlert class="h-5 w-5 shrink-0 text-red-600 dark:text-red-400 mt-0.5" />
					<div>
						<h1 class="font-semibold text-neutral-900 dark:text-white">Koneksi tidak bisa dilanjutkan</h1>
						<p class="mt-1 text-sm text-neutral-600 dark:text-neutral-400">{galat || "Permintaan koneksi tidak lengkap."}</p>
						<a href={user ? "/app/ai-akses" : "/login"} use:inertia class="mt-4 inline-flex min-h-10 items-center rounded-xl border border-neutral-300 dark:border-white/[0.1] px-4 text-sm font-medium text-neutral-700 dark:text-neutral-200 hover:border-brand-400 focus-visible:outline-2 focus-visible:outline-brand-400">
							{user ? "Buka Akses AI" : "Ke halaman masuk"}
						</a>
					</div>
				</div>
			</section>
		{:else}
			<section class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 shadow-sm" aria-labelledby="judul-izin">
				<div class="p-6 border-b border-neutral-200/80 dark:border-white/[0.05]">
					<p class="text-sm text-neutral-500 dark:text-neutral-400">Permintaan akses</p>
					<h1 id="judul-izin" class="mt-1 text-xl font-bold tracking-tight text-neutral-900 dark:text-white">
						{klien.nama} ingin terhubung ke dashboard
					</h1>
					<p class="mt-2 text-sm text-neutral-600 dark:text-neutral-400">
						Masuk sebagai <span class="font-medium text-neutral-900 dark:text-white">{user?.name}</span>
						<span class="text-neutral-400">·</span> {roleLabel[role] ?? role}
					</p>
				</div>

				{#if !diizinkan}
					<div class="p-6">
						<div class="flex gap-3 rounded-xl border border-amber-500/30 bg-amber-500/10 p-4 text-sm text-amber-800 dark:text-amber-300">
							<CircleAlert class="h-5 w-5 shrink-0" />
							<p>Role Anda belum mendapat akses AI agent. Akses tersedia untuk super admin, admin, keuangan, CS, admin kelas, dan koordinator guru.</p>
						</div>
						<button type="button" onclick={() => putuskan(false)} disabled={mengirim !== null}
							class="mt-5 w-full min-h-11 rounded-xl border border-neutral-300 dark:border-white/[0.1] text-sm font-semibold text-neutral-700 dark:text-neutral-200 hover:bg-neutral-100 dark:hover:bg-neutral-800 disabled:opacity-60 focus-visible:outline-2 focus-visible:outline-brand-400">
							{mengirim ? "Mengembalikan…" : `Kembali ke ${klien.nama}`}
						</button>
					</div>
				{:else}
					<div class="p-6 space-y-5">
						<ul class="space-y-4 text-sm">
							<li class="flex gap-3">
								<Eye class="h-5 w-5 shrink-0 text-brand-600 dark:text-brand-400" />
								<div>
									<p class="font-medium text-neutral-900 dark:text-white">Membaca data sesuai role Anda</p>
									<p class="text-neutral-600 dark:text-neutral-400">{fiturBaca.length} fitur baca, sama seperti yang bisa Anda lihat di dashboard.</p>
									<button type="button" onclick={() => (lihatFitur = !lihatFitur)} aria-expanded={lihatFitur}
										class="mt-1 inline-flex items-center gap-1 text-xs font-medium text-brand-700 dark:text-brand-300 hover:underline focus-visible:outline-2 focus-visible:outline-brand-400 rounded">
										{lihatFitur ? "Sembunyikan daftar" : "Lihat daftar fitur"}
										<ChevronDown class="h-3.5 w-3.5 transition-transform {lihatFitur ? 'rotate-180' : ''}" />
									</button>
									{#if lihatFitur}
										<ul class="mt-2 grid gap-1 text-xs text-neutral-600 dark:text-neutral-400">
											{#each fiturBaca as t (t.name)}<li>· {t.title}</li>{/each}
										</ul>
									{/if}
								</div>
							</li>
							<li class="flex gap-3">
								<EyeOff class="h-5 w-5 shrink-0 text-neutral-500" />
								<div>
									<p class="font-medium text-neutral-900 dark:text-white">Tanpa data pribadi santri</p>
									<p class="text-neutral-600 dark:text-neutral-400">Nomor HP, email, domisili, dan voice note tidak pernah dikirim ke AI.</p>
								</div>
							</li>
							<li class="flex gap-3">
								<ShieldCheck class="h-5 w-5 shrink-0 text-neutral-500" />
								<div>
									<p class="font-medium text-neutral-900 dark:text-white">Tercatat dan bisa dicabut</p>
									<p class="text-neutral-600 dark:text-neutral-400">Setiap pemanggilan masuk riwayat. Cabut kapan saja di Profil → Akses AI.</p>
								</div>
							</li>
						</ul>

						{#if boleh_tulis && fiturTulis.length > 0}
							<label class="flex items-start gap-3 rounded-xl border border-amber-500/30 bg-amber-500/5 p-3 text-sm cursor-pointer">
								<input type="checkbox" bind:checked={izinTulis} class="mt-0.5 h-4 w-4 accent-brand-600" />
								<span>
									<span class="flex items-center gap-1.5 font-medium text-neutral-900 dark:text-white"><PencilLine class="h-4 w-4" /> Izinkan juga menulis data CRM</span>
									<span class="block mt-0.5 text-xs text-neutral-600 dark:text-neutral-400">{fiturTulis.length} fitur tulis, misalnya {fiturTulis.slice(0, 3).map((t) => t.title.toLowerCase()).join(", ")}. Biarkan kosong bila hanya untuk bertanya.</span>
								</span>
							</label>
						{/if}

						<p class="text-xs text-neutral-500 dark:text-neutral-400">
							Setelah diizinkan, Anda diarahkan kembali ke <span class="font-mono text-neutral-700 dark:text-neutral-300">{klien.redirect_host}</span>.
							Izinkan hanya bila Anda sendiri yang baru saja menambahkan connector ini.
						</p>

						<div class="grid grid-cols-2 gap-3">
							<button type="button" onclick={() => putuskan(false)} disabled={mengirim !== null}
								class="min-h-11 rounded-xl border border-neutral-300 dark:border-white/[0.1] text-sm font-semibold text-neutral-700 dark:text-neutral-200 hover:bg-neutral-100 dark:hover:bg-neutral-800 disabled:opacity-60 focus-visible:outline-2 focus-visible:outline-brand-400 active:scale-[0.97] transition">
								{mengirim === "tolak" ? "Menolak…" : "Tolak"}
							</button>
							<button type="button" onclick={() => putuskan(true)} disabled={mengirim !== null}
								class="min-h-11 rounded-xl bg-brand-600 hover:bg-brand-700 dark:bg-brand-500 dark:hover:bg-brand-400 text-sm font-semibold text-white disabled:opacity-60 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-400 active:scale-[0.97] transition">
								{mengirim === "izinkan" ? "Menghubungkan…" : "Izinkan"}
							</button>
						</div>
					</div>
				{/if}
			</section>
			<p class="mt-4 text-center text-xs text-neutral-500 dark:text-neutral-400">
				Bukan Anda? <a href="/app/profile" use:inertia class="underline hover:text-brand-600">Buka profil</a> lalu keluar.
			</p>
		{/if}
	</div>
</main>
