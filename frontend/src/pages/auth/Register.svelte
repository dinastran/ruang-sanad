<script lang="ts">
    import { router, inertia } from "@inertiajs/svelte";
    import { Lock, Mail, User, ArrowRight, Eye, EyeOff, Sparkles } from "lucide-svelte";
    import AuthLayout from "@layouts/AuthLayout.svelte";
    import type { Flash } from "@lib/types";

    let form = $state({
        name: "",
        email: "",
        password: "",
        password_confirmation: "",
    });

    let isLoading = $state(false);
    let showPassword = $state(false);
    let passwordError = $state("");

    interface Props {
        flash?: Flash;
    }

    let { flash }: Props = $props();

    function generatePassword() {
        const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ123456789!@#_";
        let password = "";
        for (let i = 0; i < 12; i++) {
            password += chars.charAt(Math.floor(Math.random() * chars.length));
        }
        form.password = password;
        form.password_confirmation = password;
        showPassword = true;
    }

    function submitForm(e: Event) {
        e.preventDefault();

        if (form.password !== form.password_confirmation) {
            passwordError = "Passwords do not match";
            return;
        }
        passwordError = "";
        isLoading = true;

        router.post("/register", form, {
            onFinish: () => {
                setTimeout(() => {
                    isLoading = false;
                }, 500);
            },
        });
    }
</script>

<AuthLayout
    title="Create account"
    subtitle="Get started with your free account"
    headline="Start building today"
    subheadline="Join developers who build blazing-fast applications with the high-performance Go + Svelte framework."
    {flash}
>
    <a
        href="/auth/google"
        class="w-full flex items-center justify-center gap-3 px-4 py-3 rounded-xl border border-neutral-300 dark:border-white/[0.08] bg-white dark:bg-neutral-800/50 text-neutral-700 dark:text-white font-medium hover:bg-neutral-50 dark:hover:bg-neutral-800 transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-brand-400/50 focus:ring-offset-2 focus:ring-offset-neutral-100 dark:focus:ring-offset-neutral-900"
    >
        <svg class="h-5 w-5" viewBox="0 0 24 24">
            <path d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z" fill="#4285F4"/>
            <path d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" fill="#34A853"/>
            <path d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z" fill="#FBBC05"/>
            <path d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" fill="#EA4335"/>
        </svg>
        Sign up with Google
    </a>

    <div class="relative my-6">
        <div class="absolute inset-0 flex items-center">
            <div class="w-full border-t border-neutral-200/80 dark:border-white/[0.04]"></div>
        </div>
        <div class="relative flex justify-center">
            <span class="px-4 text-sm text-neutral-500 bg-white dark:bg-neutral-925">or sign up with email</span>
        </div>
    </div>

    <form class="space-y-4" onsubmit={submitForm}>
        <div class="space-y-2">
            <label for="name" class="block text-sm font-medium text-neutral-700 dark:text-neutral-300">Full Name</label>
            <div class="relative">
                <div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
                    <User class="w-5 h-5 text-neutral-500" />
                </div>
                <input
                    bind:value={form.name}
                    required
                    type="text"
                    name="name"
                    id="name"
                    class="w-full pl-12 pr-4 py-3 rounded-xl bg-white dark:bg-neutral-900 border border-neutral-300 dark:border-neutral-700/80 text-neutral-900 dark:text-white placeholder-neutral-400 dark:placeholder-neutral-500 focus:outline-none focus:border-brand-400 focus:ring-2 focus:ring-brand-400/20 transition-colors duration-200"
                    placeholder="John Doe"
                />
            </div>
        </div>

        <div class="space-y-2">
            <label for="email" class="block text-sm font-medium text-neutral-700 dark:text-neutral-300">Email</label>
            <div class="relative">
                <div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
                    <Mail class="w-5 h-5 text-neutral-500" />
                </div>
                <input
                    bind:value={form.email}
                    required
                    type="email"
                    name="email"
                    id="email"
                    class="w-full pl-12 pr-4 py-3 rounded-xl bg-white dark:bg-neutral-900 border border-neutral-300 dark:border-neutral-700/80 text-neutral-900 dark:text-white placeholder-neutral-400 dark:placeholder-neutral-500 focus:outline-none focus:border-brand-400 focus:ring-2 focus:ring-brand-400/20 transition-colors duration-200"
                    placeholder="you@example.com"
                />
            </div>
        </div>

        <div class="space-y-2">
            <label for="password" class="block text-sm font-medium text-neutral-700 dark:text-neutral-300">Password</label>
            <div class="relative">
                <div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
                    <Lock class="w-5 h-5 text-neutral-500" />
                </div>
                <input
                    bind:value={form.password}
                    required
                    type={showPassword ? 'text' : 'password'}
                    name="password"
                    id="password"
                    placeholder="••••••••"
                    class="w-full pl-12 pr-12 py-3 rounded-xl bg-white dark:bg-neutral-900 border border-neutral-300 dark:border-neutral-700/80 text-neutral-900 dark:text-white placeholder-neutral-400 dark:placeholder-neutral-500 focus:outline-none focus:border-brand-400 focus:ring-2 focus:ring-brand-400/20 transition-colors duration-200"
                />
                <button
                    type="button"
                    onclick={() => showPassword = !showPassword}
                    class="absolute inset-y-0 right-0 pr-4 flex items-center text-neutral-400 hover:text-neutral-700 dark:hover:text-neutral-300 transition-colors"
                >
                    {#if showPassword}
                        <EyeOff class="w-5 h-5" />
                    {:else}
                        <Eye class="w-5 h-5" />
                    {/if}
                </button>
            </div>
            <button
                type="button"
                onclick={generatePassword}
                class="text-xs text-brand-600 hover:text-brand-700 dark:text-brand-400 dark:hover:text-brand-300 transition-colors flex items-center gap-1"
            >
                <Sparkles class="w-3 h-3" />
                Generate secure password
            </button>
        </div>

        <div class="space-y-2">
            <label for="confirm-password" class="block text-sm font-medium text-neutral-700 dark:text-neutral-300">Confirm Password</label>
            <div class="relative">
                <div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
                    <Lock class="w-5 h-5 text-neutral-500" />
                </div>
                <input
                    bind:value={form.password_confirmation}
                    required
                    type={showPassword ? 'text' : 'password'}
                    name="confirm-password"
                    id="confirm-password"
                    placeholder="••••••••"
                    class="w-full pl-12 pr-4 py-3 rounded-xl bg-white dark:bg-neutral-900 border border-neutral-300 dark:border-neutral-700/80 text-neutral-900 dark:text-white placeholder-neutral-400 dark:placeholder-neutral-500 focus:outline-none focus:border-brand-400 focus:ring-2 focus:ring-brand-400/20 transition-colors duration-200"
                />
            </div>
            {#if passwordError}
                <p class="text-xs text-red-400">{passwordError}</p>
            {/if}
        </div>

        <button
            type="submit"
            disabled={isLoading}
            class="w-full py-3 px-4 rounded-xl bg-brand-600 hover:bg-brand-700 text-white font-semibold dark:bg-brand-500 dark:hover:bg-brand-400 focus:outline-none focus:ring-2 focus:ring-brand-400/50 focus:ring-offset-2 focus:ring-offset-neutral-100 dark:focus:ring-offset-neutral-900 transition-all duration-200 disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2 mt-6"
        >
            {#if isLoading}
                <svg class="animate-spin h-5 w-5" viewBox="0 0 24 24">
                    <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none"></circle>
                    <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                </svg>
                Creating account...
            {:else}
                Create account
                <ArrowRight class="w-5 h-5" />
            {/if}
        </button>
    </form>

    <p class="mt-6 text-center text-sm text-neutral-600 dark:text-neutral-400">
        Already have an account?
        <a href="/login" use:inertia class="text-brand-600 hover:text-brand-700 dark:text-brand-400 dark:hover:text-brand-300 font-medium transition-colors">Sign in</a>
    </p>
</AuthLayout>