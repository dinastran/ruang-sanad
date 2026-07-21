<script lang="ts">
	import { inertia, router } from "@inertiajs/svelte";
	import { fly } from "svelte/transition";
	import AppLayout from "@layouts/AppLayout.svelte";
	import type { User } from "@lib/types";
	import { User as UserIcon, Mail, Shield, Calendar, Check, X } from "lucide-svelte";

	interface UserItem {
		id: number;
		email: string;
		name: string;
		avatar: string;
		role: string;
		email_verified: boolean;
		created_at: string;
	}

	const ROLE_OPTIONS = ["cs", "admin_kelas", "keuangan", "super_admin"];

	const ROLE_LABELS: Record<string, string> = {
		cs: "CS",
		admin_kelas: "Admin Kelas",
		keuangan: "Keuangan",
		super_admin: "Super Admin",
	};

	const ROLE_COLORS: Record<string, string> = {
		cs: "bg-blue-500/10 text-blue-700 dark:text-blue-400",
		admin_kelas: "bg-purple-500/10 text-purple-700 dark:text-purple-400",
		keuangan: "bg-green-500/10 text-green-700 dark:text-green-400",
		super_admin: "bg-red-500/10 text-red-700 dark:text-red-400",
	};

	interface Props {
		user?: User;
		users?: UserItem[];
		success?: string;
		error?: string;
	}

	let { user, users = [], success, error }: Props = $props();

	let changingRole = $state<Record<number, boolean>>({});

	function handleRoleChange(userId: number, newRole: string, userIdCurrent: number) {
		if (userId === userIdCurrent) return;
		changingRole[userId] = true;
		router.put(
			`/admin/users/${userId}/role`,
			{ role: newRole },
			{
				onFinish: () => { changingRole[userId] = false; },
				onError: () => { changingRole[userId] = false; },
			}
		);
	}

	function formatDate(d: string): string {
		if (!d) return "-";
		return d.substring(0, 10);
	}
</script>

<AppLayout {user} group="admin">
	<div class="pt-8 pb-10 border-b border-neutral-200/80 dark:border-white/[0.04]">
		<div class="max-w-5xl mx-auto px-6">
			<div class="flex items-center gap-2 text-sm text-neutral-500 dark:text-neutral-400 mb-4">
				<a href="/app" use:inertia class="hover:text-brand-600 dark:hover:text-brand-400 transition-colors">Dashboard</a>
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
				</svg>
				<span class="text-neutral-700 dark:text-neutral-300">Manage Users</span>
			</div>
			<h1 class="text-3xl font-bold text-neutral-900 dark:text-white mb-2 tracking-tight">User Management</h1>
			<p class="text-neutral-600 dark:text-neutral-400">Kelola role dan akses pengguna</p>
		</div>
	</div>

	<div class="relative max-w-5xl mx-auto px-6 py-8 space-y-6">
		{#if success}
			<div class="bg-green-500/10 border border-green-500/20 text-green-700 dark:text-green-400 rounded-2xl p-4 flex items-center gap-3" in:fly={{ y: 20, duration: 300 }}>
				<p class="text-sm font-medium">{success}</p>
			</div>
		{/if}

		{#if error}
			<div class="bg-red-500/10 border border-red-500/20 text-red-600 dark:text-red-400 rounded-2xl p-4 flex items-center gap-3" in:fly={{ y: 20, duration: 300 }}>
				<p class="text-sm font-medium">{error}</p>
			</div>
		{/if}

		<div class="overflow-x-auto rounded-xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50" in:fly={{ y: 20, duration: 500 }}>
			<table class="w-full text-sm">
				<thead>
					<tr class="bg-neutral-50 dark:bg-neutral-900/50 border-b border-neutral-200/80 dark:border-white/[0.04]">
						<th class="px-4 py-3 text-left text-xs font-semibold text-neutral-500 dark:text-neutral-400 uppercase tracking-wider">User</th>
						<th class="px-4 py-3 text-left text-xs font-semibold text-neutral-500 dark:text-neutral-400 uppercase tracking-wider">Email</th>
						<th class="px-4 py-3 text-left text-xs font-semibold text-neutral-500 dark:text-neutral-400 uppercase tracking-wider">Role</th>
						<th class="px-4 py-3 text-left text-xs font-semibold text-neutral-500 dark:text-neutral-400 uppercase tracking-wider">Verified</th>
						<th class="px-4 py-3 text-left text-xs font-semibold text-neutral-500 dark:text-neutral-400 uppercase tracking-wider">Created At</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-neutral-200/80 dark:divide-white/[0.04]">
					{#if users.length === 0}
						<tr>
							<td colspan="5" class="px-4 py-12 text-center text-neutral-500 dark:text-neutral-400">
								Tidak ada user
							</td>
						</tr>
					{:else}
						{#each users as u}
							<tr class="hover:bg-neutral-50/50 dark:hover:bg-white/[0.015] transition-colors">
								<td class="px-4 py-3">
									<div class="flex items-center gap-3">
										<div class="w-9 h-9 rounded-full bg-brand-600 dark:bg-brand-500 flex items-center justify-center text-white font-bold text-sm shrink-0 ring-2 ring-neutral-300 dark:ring-neutral-700">
											{u.name.charAt(0).toUpperCase()}
										</div>
										<span class="font-medium text-neutral-900 dark:text-white">{u.name}</span>
									</div>
								</td>
								<td class="px-4 py-3 text-neutral-600 dark:text-neutral-400 font-mono text-xs">{u.email}</td>
								<td class="px-4 py-3">
									{#if u.id === user?.id}
										<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium {ROLE_COLORS[u.role] || 'bg-neutral-500/10 text-neutral-600 dark:text-neutral-400'}">
											{ROLE_LABELS[u.role] || u.role}
										</span>
									{:else}
										<select
											value={u.role}
											onchange={(e) => handleRoleChange(u.id, (e.target as HTMLSelectElement).value, user?.id ?? 0)}
											disabled={changingRole[u.id]}
											class="px-2.5 py-1 rounded-lg text-xs font-medium border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-900 text-neutral-700 dark:text-neutral-300 focus:outline-none focus:ring-2 focus:ring-brand-400/40 disabled:opacity-50"
										>
											{#each ROLE_OPTIONS as role}
												<option value={role}>{ROLE_LABELS[role] || role}</option>
											{/each}
										</select>
									{/if}
								</td>
								<td class="px-4 py-3">
									{#if u.email_verified}
										<Check class="w-4 h-4 text-green-500" />
									{:else}
										<X class="w-4 h-4 text-red-400" />
									{/if}
								</td>
								<td class="px-4 py-3 text-neutral-500 dark:text-neutral-400 text-xs font-mono">{formatDate(u.created_at)}</td>
							</tr>
						{/each}
					{/if}
				</tbody>
			</table>
		</div>
	</div>
</AppLayout>
