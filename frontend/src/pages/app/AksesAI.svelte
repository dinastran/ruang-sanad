<script lang="ts">
	import { router, inertia } from "@inertiajs/svelte";
	import AppLayout from "@layouts/AppLayout.svelte";
	import { Toast } from "@lib/notifications/toast";
	import { getCSRFToken } from "@lib/utils/csrf";
	import type { Flash, User } from "@lib/types";
	import { Bot, Check, Copy, KeyRound, ShieldCheck, Trash2, History, CircleAlert } from "lucide-svelte";

	interface AIToken {
		id: number;
		nama: string;
		prefix: string;
		expires_at: string;
		last_used_at: string | null;
		revoked_at: string | null;
		created_at: string;
		can_write: boolean;
	}

	interface AuditEntry {
		id: number;
		user_name: string;
		tool: string;
		status: string;
		pesan: string;
		durasi_ms: number;
		created_at: string;
	}

	interface ToolInfo {
		name: string;
		title: string;
		description: string;
		write: boolean;
	}

	interface Props {
		user?: User;
		diizinkan: boolean;
		mcp_url: string;
		tokens: AIToken[];
		audit: AuditEntry[];
		audit_semua: boolean;
		tools: ToolInfo[];
		boleh_tulis: boolean;
	}

	let { user, diizinkan, mcp_url, tokens, audit, audit_semua, tools, boleh_tulis }: Props = $props();

	let nama = $state("");
	let masaBerlaku = $state(90);
	let bolehMenulis = $state(false);
	let membuat = $state(false);
	let tokenBaru = $state<string | null>(null);
	let copied = $state<string | null>(null);
	let klien = $state<"claude-desktop" | "claude-code" | "hermes">("claude-desktop");

	const tokenTampil = $derived(tokenBaru ?? "<TOKEN_ANDA>");

	const snippet = $derived.by(() => {
		if (klien === "claude-code") {
			return `claude mcp add --transport http ruang-sanad ${mcp_url} \\\n  --header "Authorization: Bearer ${tokenTampil}"`;
		}
		if (klien === "hermes") {
			return `# ~/.hermes/config.yaml\nmcp_servers:\n  ruang_sanad:\n    url: "${mcp_url}"\n    headers:\n      Authorization: "Bearer ${tokenTampil}"`;
		}
		return JSON.stringify(
			{
				mcpServers: {
					"ruang-sanad": {
						command: "npx",
						args: ["-y", "mcp-remote", mcp_url, "--header", "Authorization:${RS_AUTH}"],
						env: { RS_AUTH: `Bearer ${tokenTampil}` },
					},
				},
			},
			null,
			2,
		);
	});

	const fmt = new Intl.DateTimeFormat("id-ID", { day: "numeric", month: "short", year: "numeric" });
	const fmtWaktu = new Intl.DateTimeFormat("id-ID", { day: "numeric", month: "short", hour: "2-digit", minute: "2-digit" });

	function tanggal(v: string | null, withTime = false): string {
		if (!v) return "—";
		const d = new Date(v);
		if (Number.isNaN(d.getTime())) return v;
		return (withTime ? fmtWaktu : fmt).format(d);
	}

	function statusToken(t: AIToken): { label: string; cls: string } {
		if (t.revoked_at) return { label: "Dicabut", cls: "bg-neutral-200/80 text-neutral-600 dark:bg-neutral-800 dark:text-neutral-400" };
		if (new Date(t.expires_at).getTime() <= Date.now()) return { label: "Kedaluwarsa", cls: "bg-amber-500/10 text-amber-700 dark:text-amber-400" };
		return { label: "Aktif", cls: "bg-brand-400/10 text-brand-700 dark:text-brand-400" };
	}

	async function salin(teks: string, kunci: string) {
		try {
			await navigator.clipboard.writeText(teks);
			copied = kunci;
			setTimeout(() => (copied === kunci ? (copied = null) : null), 2000);
		} catch {
			Toast("Gagal menyalin. Salin manual dari kotak teks.", "error");
		}
	}

	async function buatToken(event: SubmitEvent) {
		event.preventDefault();
		if (!nama.trim()) {
			Toast("Isi nama token terlebih dahulu", "error");
			return;
		}
		membuat = true;
		try {
			const res = await fetch("/app/ai-akses/token", {
				method: "POST",
				headers: { "Content-Type": "application/json", Accept: "application/json", "X-XSRF-TOKEN": getCSRFToken() },
				body: JSON.stringify({ nama: nama.trim(), masa_berlaku_hari: Number(masaBerlaku), can_write: boleh_tulis && bolehMenulis }),
			});
			const data = await res.json().catch(() => ({}));
			if (!res.ok) {
				Toast(data.error || "Gagal membuat token", "error");
				return;
			}
			tokenBaru = data.plaintext;
			nama = "";
			bolehMenulis = false;
			router.reload({ only: ["tokens"] });
		} catch {
			Toast("Koneksi gagal. Coba lagi.", "error");
		} finally {
			membuat = false;
		}
	}

	function cabut(t: AIToken) {
		if (!confirm(`Cabut token "${t.nama}"? AI agent yang memakainya langsung kehilangan akses.`)) return;
		router.delete(`/app/ai-akses/token/${t.id}`, {
			preserveScroll: true,
			onSuccess: (page) => {
				const flash = page.props.flash as Flash | undefined;
				if (flash?.error) Toast(flash.error, "error");
				else if (flash?.success) Toast(flash.success, "success");
			},
		});
	}
</script>

<svelte:head><title>Akses AI · Ruang Sanad</title></svelte:head>

<AppLayout {user}>
	<div class="pt-8 pb-8 border-b border-neutral-200/80 dark:border-white/[0.04]">
		<div class="max-w-5xl mx-auto px-4 sm:px-6">
			<div class="flex items-center gap-2 text-sm text-neutral-500 dark:text-neutral-400 mb-4">
				<a href="/app/profile" use:inertia class="hover:text-brand-600 dark:hover:text-brand-400 transition-colors">Profil</a>
				<span aria-hidden="true">/</span>
				<span class="text-neutral-700 dark:text-neutral-300">Akses AI</span>
			</div>
			<h1 class="text-2xl sm:text-3xl font-bold text-neutral-900 dark:text-white mb-2">Akses AI</h1>
			<p class="text-neutral-600 dark:text-neutral-400 max-w-2xl">
				Hubungkan Claude, Hermes, atau AI agent lain ke data dashboard. Aksesnya membaca sesuai role Anda (token tulis CRM
				khusus dibuat super admin), dan tidak pernah mengirim nomor HP, email, domisili, atau voice note.
			</p>
		</div>
	</div>

	<div class="max-w-5xl mx-auto px-4 sm:px-6 py-10 space-y-10">
		{#if !diizinkan}
			<div class="rounded-2xl border border-amber-500/30 bg-amber-500/10 p-5 flex gap-3 text-amber-800 dark:text-amber-300">
				<CircleAlert class="w-5 h-5 shrink-0 mt-0.5" />
				<div>
					<p class="font-semibold">Role Anda belum mendapat akses AI agent.</p>
					<p class="text-sm mt-1">Tahap pertama tersedia untuk super admin, admin, keuangan, CS, admin kelas, dan koordinator guru. Hubungi super admin bila perlu.</p>
				</div>
			</div>
		{:else}
			<!-- Server address -->
			<section aria-labelledby="alamat-mcp">
				<h2 id="alamat-mcp" class="text-lg font-semibold text-neutral-900 dark:text-white mb-1">Alamat server</h2>
				<p class="text-sm text-neutral-600 dark:text-neutral-400 mb-3">Masukkan alamat ini di pengaturan MCP AI agent Anda.</p>
				<div class="flex items-stretch gap-2">
					<code class="flex-1 min-w-0 truncate rounded-xl border border-neutral-200 dark:border-white/[0.08] bg-neutral-50 dark:bg-neutral-900 px-4 py-3 text-sm text-neutral-800 dark:text-neutral-200">{mcp_url}</code>
					<button type="button" onclick={() => salin(mcp_url, "url")}
						class="inline-flex items-center gap-2 rounded-xl border border-neutral-200 dark:border-white/[0.08] px-4 text-sm font-medium text-neutral-700 dark:text-neutral-300 hover:border-brand-400 focus-visible:outline-2 focus-visible:outline-brand-400 active:scale-[0.97] transition">
						{#if copied === "url"}<Check class="w-4 h-4" />Tersalin{:else}<Copy class="w-4 h-4" />Salin{/if}
					</button>
				</div>
			</section>

			<!-- Create token -->
			<section aria-labelledby="buat-token" class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-6">
				<h2 id="buat-token" class="text-lg font-semibold text-neutral-900 dark:text-white mb-1 flex items-center gap-2"><KeyRound class="w-5 h-5 text-brand-600 dark:text-brand-400" />Buat token</h2>
				<p class="text-sm text-neutral-600 dark:text-neutral-400 mb-5">Satu token per aplikasi AI. Token mewakili akun Anda, jadi jangan dibagikan.</p>
				<form onsubmit={buatToken} class="grid gap-4 sm:grid-cols-[1fr_180px_auto] sm:items-end">
					<div>
						<label for="nama-token" class="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-1.5">Nama token</label>
						<input id="nama-token" bind:value={nama} maxlength="60" placeholder="contoh: Claude Desktop laptop kantor"
							class="w-full rounded-xl border border-neutral-300 dark:border-white/[0.1] bg-white dark:bg-neutral-900 px-4 py-2.5 text-sm text-neutral-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-brand-400" />
					</div>
					<div>
						<label for="masa-berlaku" class="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-1.5">Berlaku</label>
						<select id="masa-berlaku" bind:value={masaBerlaku}
							class="w-full rounded-xl border border-neutral-300 dark:border-white/[0.1] bg-white dark:bg-neutral-900 px-3 py-2.5 text-sm text-neutral-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-brand-400">
							<option value={30}>30 hari</option>
							<option value={90}>90 hari</option>
							<option value={180}>180 hari</option>
							<option value={365}>1 tahun</option>
						</select>
					</div>
					<button type="submit" disabled={membuat}
						class="rounded-xl bg-brand-600 hover:bg-brand-700 dark:bg-brand-500 dark:hover:bg-brand-400 text-white px-5 py-2.5 text-sm font-semibold disabled:opacity-60 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-400 active:scale-[0.97] transition">
						{membuat ? "Membuat…" : "Buat token"}
					</button>
					{#if boleh_tulis}
						<label class="sm:col-span-3 flex items-start gap-3 rounded-xl border border-amber-500/30 bg-amber-500/5 p-3 text-sm text-neutral-700 dark:text-neutral-300">
							<input type="checkbox" bind:checked={bolehMenulis} class="mt-0.5" />
							<span>
								<span class="font-medium text-neutral-900 dark:text-white">Boleh menulis analisis CRM</span>
								<span class="block text-xs text-neutral-500 dark:text-neutral-400 mt-0.5">Untuk job harian Hermes. AI bisa menyimpan analisis lead, mengusulkan follow-up (menunggu persetujuan CS), dan memasang label yang sudah ada. AI tetap tidak bisa mengubah tahap, memverifikasi, atau mengirim pesan.</span>
							</span>
						</label>
					{/if}
				</form>

				{#if tokenBaru}
					<div class="mt-6 rounded-xl border border-brand-400/40 bg-brand-400/10 p-4" role="status">
						<p class="text-sm font-semibold text-neutral-900 dark:text-white">Simpan token ini sekarang. Token tidak akan ditampilkan lagi.</p>
						<div class="mt-3 flex items-stretch gap-2">
							<code class="flex-1 min-w-0 break-all rounded-lg bg-white dark:bg-neutral-950 border border-neutral-200 dark:border-white/[0.08] px-3 py-2 text-xs text-neutral-800 dark:text-neutral-200">{tokenBaru}</code>
							<button type="button" onclick={() => salin(tokenBaru ?? "", "token")}
								class="inline-flex items-center gap-2 rounded-lg bg-brand-600 dark:bg-brand-500 text-white px-3 text-sm font-medium focus-visible:outline-2 focus-visible:outline-brand-400 active:scale-[0.97] transition">
								{#if copied === "token"}<Check class="w-4 h-4" />Tersalin{:else}<Copy class="w-4 h-4" />Salin{/if}
							</button>
						</div>
					</div>
				{/if}
			</section>

			<!-- Setup snippets -->
			<section aria-labelledby="cara-pasang">
				<h2 id="cara-pasang" class="text-lg font-semibold text-neutral-900 dark:text-white mb-1">Cara memasang</h2>
				<p class="text-sm text-neutral-600 dark:text-neutral-400 mb-3">
					{tokenBaru ? "Token baru sudah dimasukkan ke contoh di bawah." : "Ganti <TOKEN_ANDA> dengan token yang Anda buat."}
				</p>
				<div class="flex flex-wrap gap-2 mb-3" role="tablist" aria-label="Pilih aplikasi AI">
					{#each [["claude-desktop", "Claude Desktop"], ["claude-code", "Claude Code"], ["hermes", "Hermes Agent"]] as [key, label] (key)}
						<button type="button" role="tab" aria-selected={klien === key} onclick={() => (klien = key as typeof klien)}
							class="rounded-lg px-3 py-1.5 text-sm font-medium transition focus-visible:outline-2 focus-visible:outline-brand-400 {klien === key ? 'bg-brand-400/10 text-brand-700 dark:text-brand-400' : 'text-neutral-600 dark:text-neutral-400 hover:bg-neutral-100 dark:hover:bg-neutral-800'}">
							{label}
						</button>
					{/each}
				</div>
				<div class="relative">
					<pre class="overflow-x-auto rounded-xl bg-neutral-900 text-neutral-100 p-4 text-xs leading-relaxed"><code>{snippet}</code></pre>
					<button type="button" onclick={() => salin(snippet, "snippet")}
						class="absolute top-2 right-2 inline-flex items-center gap-1.5 rounded-lg bg-white/10 hover:bg-white/20 text-white px-2.5 py-1 text-xs focus-visible:outline-2 focus-visible:outline-brand-400">
						{#if copied === "snippet"}<Check class="w-3.5 h-3.5" />Tersalin{:else}<Copy class="w-3.5 h-3.5" />Salin{/if}
					</button>
				</div>
				<p class="text-xs text-neutral-500 dark:text-neutral-400 mt-2">
					{#if klien === "claude-desktop"}
						Tempel di Settings → Developer → Edit Config (claude_desktop_config.json), lalu restart Claude Desktop. Butuh Node.js terpasang.
					{:else if klien === "claude-code"}
						Jalankan di terminal. Cek dengan perintah <code>claude mcp list</code>.
					{:else}
						Tambahkan ke konfigurasi Hermes, lalu restart Hermes. Format bisa berbeda antar versi; cek dokumentasi Hermes Anda.
					{/if}
					Connect langsung dari claude.ai (web) dan ChatGPT menyusul di tahap 2 (login OAuth).
				</p>
			</section>

			<!-- Tokens -->
			<section aria-labelledby="token-anda">
				<h2 id="token-anda" class="text-lg font-semibold text-neutral-900 dark:text-white mb-3">Token Anda</h2>
				{#if tokens.length === 0}
					<div class="rounded-2xl border border-dashed border-neutral-300 dark:border-white/[0.1] p-8 text-center">
						<KeyRound class="w-8 h-8 mx-auto text-neutral-400 mb-2" />
						<p class="text-sm text-neutral-600 dark:text-neutral-400">Belum ada token. Buat satu di atas untuk menghubungkan AI agent.</p>
					</div>
				{:else}
					<div class="overflow-x-auto rounded-2xl border border-neutral-200/80 dark:border-white/[0.06]">
						<table class="w-full text-sm">
							<thead class="bg-neutral-50 dark:bg-neutral-900 text-left text-neutral-500 dark:text-neutral-400">
								<tr>
									<th class="px-4 py-3 font-medium">Nama</th>
									<th class="px-4 py-3 font-medium">Status</th>
									<th class="px-4 py-3 font-medium">Terakhir dipakai</th>
									<th class="px-4 py-3 font-medium">Berlaku sampai</th>
									<th class="px-4 py-3"><span class="sr-only">Aksi</span></th>
								</tr>
							</thead>
							<tbody class="divide-y divide-neutral-200/80 dark:divide-white/[0.06]">
								{#each tokens as t (t.id)}
									{@const st = statusToken(t)}
									<tr>
										<td class="px-4 py-3">
											<p class="font-medium text-neutral-900 dark:text-white">{t.nama}</p>
											<p class="text-xs text-neutral-500 font-mono">{t.prefix}… · dibuat {tanggal(t.created_at)}</p>
											{#if t.can_write}<span class="mt-1 inline-block rounded-full bg-amber-500/10 px-2 py-0.5 text-[11px] font-medium text-amber-700 dark:text-amber-400">Tulis CRM</span>{/if}
										</td>
										<td class="px-4 py-3"><span class="rounded-full px-2.5 py-1 text-xs font-medium {st.cls}">{st.label}</span></td>
										<td class="px-4 py-3 text-neutral-600 dark:text-neutral-400">{tanggal(t.last_used_at, true)}</td>
										<td class="px-4 py-3 text-neutral-600 dark:text-neutral-400">{tanggal(t.expires_at)}</td>
										<td class="px-4 py-3 text-right">
											{#if st.label === "Aktif"}
												<button type="button" onclick={() => cabut(t)}
													class="inline-flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs font-medium text-red-600 dark:text-red-400 hover:bg-red-500/10 focus-visible:outline-2 focus-visible:outline-red-400">
													<Trash2 class="w-3.5 h-3.5" />Cabut
												</button>
											{/if}
										</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				{/if}
			</section>

			<!-- Tools -->
			<section aria-labelledby="data-ai">
				<h2 id="data-ai" class="text-lg font-semibold text-neutral-900 dark:text-white mb-1 flex items-center gap-2"><ShieldCheck class="w-5 h-5 text-brand-600 dark:text-brand-400" />Data yang bisa dibaca AI dengan role Anda</h2>
				<p class="text-sm text-neutral-600 dark:text-neutral-400 mb-4">AI memanggil fitur berikut sesuai pertanyaan. Fitur bertanda "tulis" hanya aktif untuk token dengan izin menulis analisis CRM; tidak ada fitur untuk menghapus data.</p>
				<ul class="divide-y divide-neutral-200/80 dark:divide-white/[0.06] border-y border-neutral-200/80 dark:border-white/[0.06]">
					{#each tools as tool (tool.name)}
						<li class="py-3">
							<p class="text-sm font-medium text-neutral-900 dark:text-white">{tool.title} <span class="ml-1 font-mono text-xs text-neutral-500">{tool.name}</span>{#if tool.write}<span class="ml-2 rounded-full bg-amber-500/10 px-2 py-0.5 text-[11px] font-medium text-amber-700 dark:text-amber-400">tulis</span>{/if}</p>
							<p class="text-sm text-neutral-600 dark:text-neutral-400 mt-0.5">{tool.description}</p>
						</li>
					{/each}
				</ul>
			</section>
		{/if}

		<!-- Audit -->
		<section aria-labelledby="riwayat-ai">
			<h2 id="riwayat-ai" class="text-lg font-semibold text-neutral-900 dark:text-white mb-1 flex items-center gap-2"><History class="w-5 h-5 text-brand-600 dark:text-brand-400" />Riwayat akses AI</h2>
			<p class="text-sm text-neutral-600 dark:text-neutral-400 mb-3">{audit_semua ? "50 pemanggilan terakhir dari semua pengguna." : "50 pemanggilan terakhir dari token Anda."}</p>
			{#if audit.length === 0}
				<div class="rounded-2xl border border-dashed border-neutral-300 dark:border-white/[0.1] p-8 text-center">
					<Bot class="w-8 h-8 mx-auto text-neutral-400 mb-2" />
					<p class="text-sm text-neutral-600 dark:text-neutral-400">Belum ada AI agent yang membaca data.</p>
				</div>
			{:else}
				<div class="overflow-x-auto rounded-2xl border border-neutral-200/80 dark:border-white/[0.06]">
					<table class="w-full text-sm">
						<thead class="bg-neutral-50 dark:bg-neutral-900 text-left text-neutral-500 dark:text-neutral-400">
							<tr>
								<th class="px-4 py-3 font-medium">Waktu</th>
								{#if audit_semua}<th class="px-4 py-3 font-medium">Pengguna</th>{/if}
								<th class="px-4 py-3 font-medium">Fitur</th>
								<th class="px-4 py-3 font-medium">Hasil</th>
							</tr>
						</thead>
						<tbody class="divide-y divide-neutral-200/80 dark:divide-white/[0.06]">
							{#each audit as a (a.id)}
								<tr>
									<td class="px-4 py-2.5 text-neutral-600 dark:text-neutral-400 whitespace-nowrap">{tanggal(a.created_at, true)}</td>
									{#if audit_semua}<td class="px-4 py-2.5 text-neutral-800 dark:text-neutral-200">{a.user_name || "—"}</td>{/if}
									<td class="px-4 py-2.5 font-mono text-xs text-neutral-800 dark:text-neutral-200">{a.tool}</td>
									<td class="px-4 py-2.5">
										{#if a.status === "ok"}
											<span class="text-brand-700 dark:text-brand-400">Berhasil</span> <span class="text-xs text-neutral-500">· {a.durasi_ms} ms</span>
										{:else}
											<span class="text-red-600 dark:text-red-400" title={a.pesan}>Gagal</span> <span class="text-xs text-neutral-500">· {a.pesan}</span>
										{/if}
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{/if}
		</section>
	</div>
</AppLayout>
