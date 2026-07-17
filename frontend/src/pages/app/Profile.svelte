<script lang="ts">
    import { router, inertia } from "@inertiajs/svelte";
    import { fly } from "svelte/transition";
    import AppLayout from "@layouts/AppLayout.svelte";
    import { Toast } from "@lib/notifications/toast";
    import { getCSRFToken } from "@lib/utils/csrf";
    import type { User } from "@lib/types";
    import { Upload, Lock, User as UserIcon, Mail } from "lucide-svelte";

    interface Props {
        user?: User;
        success?: string;
        error?: string;
    }

    let { user, success, error }: Props = $props();

    // Intentionally capture initial prop values for form — not reactive
    function getInitialForm() {
        return {
            name: user?.name ?? "",
            email: user?.email ?? "",
            avatar: user?.avatar ?? "",
        };
    }
    let profileForm = $state(getInitialForm());

    let passwordForm = $state({
        current_password: "",
        new_password: "",
        confirm_password: "",
    });

    let isProfileLoading = $state(false);
    let isPasswordLoading = $state(false);
    let showPassword = $state(false);

    let previewUrl = $derived(user?.avatar ?? null);

    function handleAvatarChange(event: Event) {
        const target = event.target as HTMLInputElement;
        const file = target.files?.[0];
        if (file) {
            const formData = new FormData();
            formData.append("file", file);
            isProfileLoading = true;
            fetch("/app/upload", {
                method: "POST",
                headers: {
                    "X-XSRF-TOKEN": getCSRFToken(),
                },
                body: formData,
            })
                .then((response) => response.json())
                .then((data) => {
                    if (data.success && data.url) {
                        // Auto-save avatar URL to database via router
                        router.put("/app/profile", {
                            avatar: data.url,
                        }, {
                            onError: (error) => {
                                isProfileLoading = false;
                                Toast("Failed to save avatar: " + (error as any).message, "error");
                            },
                            onFinish: () => {
                                isProfileLoading = false;
                            },
                        });
                    } else {
                        isProfileLoading = false;
                        Toast(data.error || "Failed to upload avatar", "error");
                    }
                })
                .catch((error) => {
                    isProfileLoading = false;
                    Toast("Failed to upload avatar", "error");
                    console.error("Upload error:", error);
                });
        }
    }

    function handleProfileSubmit(e: Event) {
        e.preventDefault();
        isProfileLoading = true;
        router.put("/app/profile", profileForm, {
            onError: (err) => {
                const msg = (err as any)?.response?.data?.error || (err as any)?.message || "Gagal menyimpan perubahan";
                Toast(msg, "error");
            },
            onFinish: () => {
                isProfileLoading = false;
            },
        });
    }

    function handlePasswordSubmit(e: Event) {
        e.preventDefault();

        if (passwordForm.new_password !== passwordForm.confirm_password) {
            Toast("Passwords don't match", "error");
            return;
        }

        if (!passwordForm.current_password || !passwordForm.new_password || !passwordForm.confirm_password) {
            Toast("Please fill all fields", "error");
            return;
        }

        if (passwordForm.new_password.length < 8) {
            Toast("Password must be at least 8 characters", "error");
            return;
        }

        isPasswordLoading = true;
        router.put("/app/profile/password", passwordForm, {
            onError: (err) => {
                const msg = (err as any)?.response?.data?.error || (err as any)?.message || "Gagal mengubah password";
                Toast(msg, "error");
            },
            onFinish: () => {
                isPasswordLoading = false;
                passwordForm.current_password = "";
                passwordForm.new_password = "";
                passwordForm.confirm_password = "";
            },
        });
    }
</script>

<AppLayout {user} group="profile">

    <!-- Page Header -->
    <div class="pt-8 pb-12 border-b border-neutral-200/80 dark:border-white/[0.04]">
        <div class="max-w-5xl mx-auto px-6">
            <div class="flex items-center gap-2 text-sm text-neutral-500 dark:text-neutral-400 mb-4">
                <a href="/app" use:inertia class="hover:text-brand-600 dark:hover:text-brand-400 transition-colors">Dashboard</a>
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
                </svg>
                <span class="text-neutral-700 dark:text-neutral-300">Settings</span>
            </div>
            <h1 class="text-3xl font-bold text-neutral-900 dark:text-white mb-2">
                Account Settings
            </h1>
            <p class="text-neutral-600 dark:text-neutral-400">
                Manage your profile and security preferences
            </p>
        </div>
    </div>

    <!-- Content Area -->
    <div class="relative max-w-5xl mx-auto px-6 py-12">
        <!-- Flash Messages -->
        {#if success}
            <div
                class="mb-6 bg-green-500/10 border border-green-500/20 backdrop-blur-xl text-green-700 dark:text-green-400 rounded-2xl p-4 flex items-center gap-3 animate-in slide-in-from-top-2 duration-300"
                in:fly={{ y: 20, duration: 300 }}
            >
                <div class="w-8 h-8 rounded-full bg-green-500/20 flex items-center justify-center shrink-0">
                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
                    </svg>
                </div>
                <p class="text-sm font-medium">{success}</p>
            </div>
        {/if}

        {#if error}
            <div
                class="mb-6 bg-red-500/10 border border-red-500/20 backdrop-blur-xl text-red-600 dark:text-red-400 rounded-2xl p-4 flex items-center gap-3 animate-in slide-in-from-top-2 duration-300"
                in:fly={{ y: 20, duration: 300 }}
            >
                <div class="w-8 h-8 rounded-full bg-red-500/20 flex items-center justify-center shrink-0">
                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                    </svg>
                </div>
                <p class="text-sm font-medium">{error}</p>
            </div>
        {/if}

        <!-- Profile Overview Card -->
        <div
            class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-8 mb-8"
            in:fly={{ y: 20, duration: 600 }}
        >
            <div class="flex flex-col sm:flex-row items-center sm:items-start gap-6">
                <!-- Avatar -->
                <div class="relative group">
                    <div class="w-28 h-28 rounded-2xl bg-brand-600 dark:bg-brand-500 p-1 shadow-lg shadow-brand-600/25">
                        <div class="w-full h-full rounded-xl bg-white dark:bg-neutral-950 overflow-hidden">
                            {#if previewUrl}
                                <img src={previewUrl} alt="Profile" class="w-full h-full object-cover" />
                            {:else}
                                <div class="w-full h-full flex items-center justify-center">
                                    <span class="text-4xl font-bold text-brand-600 dark:text-brand-400">{user?.name?.charAt(0)?.toUpperCase() || ""}</span>
                                </div>
                            {/if}
                        </div>
                    </div>
                    <label class="absolute bottom-0 right-0 w-10 h-10 bg-brand-600 hover:bg-brand-700 text-white rounded-xl dark:bg-brand-500 dark:hover:bg-brand-400 flex items-center justify-center cursor-pointer transition-all shadow-lg shadow-brand-400/30 group-hover:scale-110">
                        <Upload class="w-5 h-5" />
                        <input type="file" accept="image/*" onchange={handleAvatarChange} class="hidden" />
                    </label>
                </div>

                <!-- User Info -->
                <div class="flex-1 text-center sm:text-left">
                    <h2 class="text-2xl font-bold text-neutral-900 dark:text-white mb-1">
                        {user?.name || ""}
                    </h2>
                    <p class="text-neutral-600 dark:text-neutral-400 mb-4">
                        {user?.email || ""}
                    </p>
                    <div class="flex flex-wrap justify-center sm:justify-start gap-2">
                        <span class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full text-xs font-medium bg-brand-400/10 text-brand-600 dark:text-brand-400 border border-brand-400/20">
                            <div class="w-1.5 h-1.5 rounded-full bg-brand-400"></div>
                            Active Member
                        </span>
                        <span class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full text-xs font-medium bg-neutral-200/80 dark:bg-neutral-800 text-neutral-600 dark:text-neutral-400 border border-neutral-300 dark:border-white/[0.06]">
                            <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
                            </svg>
                            Verified
                        </span>
                    </div>
                </div>
            </div>
        </div>

        <!-- Settings Grid -->
        <div class="grid md:grid-cols-2 gap-6">
            <!-- Personal Information -->
            <div
                class="bg-white dark:bg-neutral-925/50 rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] p-6"
                in:fly={{ y: 20, duration: 600, delay: 100 }}
            >
                <div class="flex items-center gap-3 mb-6">
                    <div class="w-10 h-10 rounded-xl bg-brand-400/10 flex items-center justify-center">
                        <UserIcon class="w-5 h-5 text-brand-600 dark:text-brand-400" />
                    </div>
                    <div>
                        <h3 class="text-lg font-semibold text-neutral-900 dark:text-white">Personal Information</h3>
                        <p class="text-sm text-neutral-600 dark:text-neutral-500">Update your personal details</p>
                    </div>
                </div>

                <form onsubmit={handleProfileSubmit} class="space-y-5">
                    <div>
                        <label for="name" class="block text-sm font-medium text-neutral-700 dark:text-neutral-400 mb-2">Full Name</label>
                        <div class="relative">
                            <div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
                                <UserIcon class="w-5 h-5 text-neutral-500" />
                            </div>
                            <input
                                bind:value={profileForm.name}
                                type="text"
                                id="name"
                                class="w-full pl-12 pr-4 py-3 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white placeholder-neutral-500 transition-all outline-none"
                                placeholder="Your full name"
                            />
                        </div>
                    </div>

                    <div>
                        <label for="email" class="block text-sm font-medium text-neutral-700 dark:text-neutral-400 mb-2">Email Address</label>
                        <div class="relative">
                            <div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
                                <Mail class="w-5 h-5 text-neutral-500" />
                            </div>
                            <input
                                bind:value={profileForm.email}
                                type="email"
                                id="email"
                                class="w-full pl-12 pr-4 py-3 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white placeholder-neutral-500 transition-all outline-none"
                                placeholder="you@example.com"
                            />
                        </div>
                    </div>

                    <div class="pt-4">
                        <button
                            type="submit"
                            disabled={isProfileLoading}
                            class="w-full px-6 py-3 rounded-xl bg-brand-600 hover:bg-brand-700 text-white font-semibold transition-all dark:bg-brand-500 dark:hover:bg-brand-400 shadow-lg shadow-brand-600/25 hover:shadow-brand-600/40 disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
                        >
                            {#if isProfileLoading}
                                <svg class="animate-spin h-5 w-5" fill="none" viewBox="0 0 24 24">
                                    <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                                    <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                                </svg>
                                Saving...
                            {:else}
                                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
                                </svg>
                                Save Changes
                            {/if}
                        </button>
                    </div>
                </form>
            </div>

            <!-- Change Password -->
            <div
                class="bg-white dark:bg-neutral-925/50 rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] p-6"
                in:fly={{ y: 20, duration: 600, delay: 200 }}
            >
                <div class="flex items-center gap-3 mb-6">
                    <div class="w-10 h-10 rounded-xl bg-red-500/10 flex items-center justify-center">
                        <Lock class="w-5 h-5 text-red-400" />
                    </div>
                    <div>
                        <h3 class="text-lg font-semibold text-neutral-900 dark:text-white">Security</h3>
                        <p class="text-sm text-neutral-600 dark:text-neutral-500">Update your password</p>
                    </div>
                </div>

                <form onsubmit={handlePasswordSubmit} class="space-y-5">
                    <div>
                        <label for="current_password" class="block text-sm font-medium text-neutral-700 dark:text-neutral-400 mb-2">Current Password</label>
                        <div class="relative">
                            <div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
                                <Lock class="w-5 h-5 text-neutral-500" />
                            </div>
                            <input
                                bind:value={passwordForm.current_password}
                                type={showPassword ? "text" : "password"}
                                id="current_password"
                                class="w-full pl-12 pr-12 py-3 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white placeholder-neutral-500 transition-all outline-none"
                                placeholder="••••••••"
                            />
                        </div>
                    </div>

                    <div>
                        <label for="new_password" class="block text-sm font-medium text-neutral-700 dark:text-neutral-400 mb-2">New Password</label>
                        <div class="relative">
                            <div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
                                <Lock class="w-5 h-5 text-neutral-500" />
                            </div>
                            <input
                                bind:value={passwordForm.new_password}
                                type={showPassword ? "text" : "password"}
                                id="new_password"
                                class="w-full pl-12 pr-4 py-3 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white placeholder-neutral-500 transition-all outline-none"
                                placeholder="••••••••"
                                minlength="8"
                            />
                        </div>
                    </div>

                    <div>
                        <label for="confirm_password" class="block text-sm font-medium text-neutral-700 dark:text-neutral-400 mb-2">Confirm New Password</label>
                        <div class="relative">
                            <div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
                                <Lock class="w-5 h-5 text-neutral-500" />
                            </div>
                            <input
                                bind:value={passwordForm.confirm_password}
                                type={showPassword ? "text" : "password"}
                                id="confirm_password"
                                class="w-full pl-12 pr-4 py-3 rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 border border-neutral-300 dark:border-neutral-700/80 focus:ring-2 focus:ring-brand-400/20 focus:border-brand-400 text-neutral-900 dark:text-white placeholder-neutral-500 transition-all outline-none"
                                placeholder="••••••••"
                            />
                        </div>
                    </div>

                    <div class="flex items-center gap-2">
                        <input
                            type="checkbox"
                            id="show_password"
                            bind:checked={showPassword}
                            class="w-4 h-4 rounded border-neutral-300 text-brand-400 focus:ring-brand-400"
                        />
                        <label for="show_password" class="text-sm text-neutral-600 dark:text-neutral-400">
                            Show passwords
                        </label>
                    </div>

                    <div class="pt-4">
                        <button
                            type="submit"
                            disabled={isPasswordLoading}
                            class="w-full px-6 py-3 rounded-xl bg-linear-to-r from-red-600 to-red-500 hover:from-red-700 hover:to-red-600 text-white font-semibold transition-all shadow-lg shadow-red-500/25 hover:shadow-red-500/40 disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
                        >
                            {#if isPasswordLoading}
                                <svg class="animate-spin h-5 w-5" fill="none" viewBox="0 0 24 24">
                                    <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                                    <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                                </svg>
                                Changing...
                            {:else}
                                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
                                </svg>
                                Change Password
                            {/if}
                        </button>
                    </div>
                </form>
            </div>
        </div>
    </div>
</AppLayout>
