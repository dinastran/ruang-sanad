<script lang="ts">
	import { inertia, router } from "@inertiajs/svelte";
	import { fly } from "svelte/transition";
	import AppLayout from "@layouts/AppLayout.svelte";
	import { Toast } from "@lib/notifications/toast";
	import type { Flash, User } from "@lib/types";
	import { Upload, Download, FileSpreadsheet, CheckCircle, XCircle, AlertTriangle } from "lucide-svelte";

	interface ImportResult {
		total: number;
		berhasil: number;
		gagal: number;
		catatan: string;
	}

	interface Props {
		user?: User;
		result: ImportResult | null;
		success?: string;
		error?: string;
		flash?: Flash;
	}

	let { user, result = null, success, error, flash }: Props = $props();
	let successMessage = $derived(flash?.success ?? success);

	let isDragOver = $state(false);
	let isUploading = $state(false);
	let selectedFile = $state<File | null>(null);
	let fileInput = $state<HTMLInputElement>();

	let parsedErrors = $derived(
		result?.catatan ? result.catatan.split("\n").filter((line) => line.trim().length > 0) : []
	);

	function handleDragOver(e: DragEvent) {
		e.preventDefault();
		isDragOver = true;
	}

	function handleDragLeave() {
		isDragOver = false;
	}

	function handleDrop(e: DragEvent) {
		e.preventDefault();
		isDragOver = false;
		const file = e.dataTransfer?.files?.[0];
		if (file) {
			if (file.name.endsWith(".csv")) {
				selectedFile = file;
			} else {
				Toast("Hanya file CSV yang diperbolehkan", "error");
			}
		}
	}

	function handleFileSelect(e: Event) {
		const target = e.target as HTMLInputElement;
		const file = target.files?.[0];
		if (file) {
			selectedFile = file;
		}
		target.value = "";
	}

	function handleSubmit() {
		if (!selectedFile) {
			Toast("Pilih file CSV terlebih dahulu", "error");
			return;
		}

		isUploading = true;
		const formData = new FormData();
		formData.append("file", selectedFile);

		router.post("/admin/import", formData, {
			forceFormData: true,
			onHttpException: () => {
				Toast("Gagal mengimpor file", "error");
				return false;
			},
			onNetworkError: () => {
				Toast("Gagal menghubungi server", "error");
				return false;
			},
			onFinish: () => {
				isUploading = false;
			},
		});
	}

	function resetForm() {
		selectedFile = null;
	}
</script>

<AppLayout {user} group="dashboard">
	<div class="pt-8 pb-10 border-b border-neutral-200/80 dark:border-white/[0.04]">
		<div class="max-w-6xl mx-auto px-4 sm:px-6">
			<div class="flex items-center gap-2 text-sm text-neutral-500 dark:text-neutral-400 mb-4">
				<a href="/app" use:inertia class="hover:text-brand-600 dark:hover:text-brand-400 transition-colors">Dashboard</a>
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
				</svg>
				<span class="text-neutral-700 dark:text-neutral-300">Import CSV</span>
			</div>
			<div class="flex items-start justify-between gap-4 flex-wrap">
				<div>
					<h1 class="text-2xl sm:text-3xl font-bold text-neutral-900 dark:text-white mb-2 tracking-tight">
						Import CSV Santri
					</h1>
					<p class="text-neutral-600 dark:text-neutral-400">
						Upload data santri secara massal melalui file CSV
					</p>
				</div>
			</div>
		</div>
	</div>

	<div class="relative max-w-6xl mx-auto px-4 sm:px-6 py-8 space-y-6">
		{#if successMessage}
			<div class="bg-green-500/10 border border-green-500/20 text-green-700 dark:text-green-400 rounded-2xl p-4 flex items-center gap-3" in:fly={{ y: 20, duration: 300 }}>
				<p class="text-sm font-medium">{successMessage}</p>
			</div>
		{/if}

		{#if error || flash?.error}
			<div class="bg-red-500/10 border border-red-500/20 text-red-600 dark:text-red-400 rounded-2xl p-4 flex items-center gap-3" in:fly={{ y: 20, duration: 300 }}>
				<p class="text-sm font-medium">{error || flash?.error}</p>
			</div>
		{/if}

		<div class="grid lg:grid-cols-5 gap-6">
			<div class="lg:col-span-3 space-y-6">
				<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-6" in:fly={{ y: 20, duration: 600 }}>
					<div class="flex items-center gap-3 mb-6">
						<div class="w-10 h-10 rounded-xl bg-brand-400/10 flex items-center justify-center">
							<FileSpreadsheet class="w-5 h-5 text-brand-600 dark:text-brand-400" />
						</div>
						<div>
							<h3 class="text-lg font-semibold text-neutral-900 dark:text-white">Upload File CSV</h3>
							<p class="text-sm text-neutral-500 dark:text-neutral-400">Pilih file CSV untuk diimport</p>
						</div>
					</div>

					<div
						class="relative rounded-2xl border-2 border-dashed transition-all duration-300 overflow-hidden {isDragOver ? 'border-brand-400 bg-brand-400/5 scale-[1.01] shadow-xl shadow-brand-400/10' : 'border-neutral-300 dark:border-neutral-700 hover:border-brand-400/50 bg-neutral-50/50 dark:bg-neutral-900/30'}"
						role="button"
						tabindex="0"
						aria-label="Upload CSV drop zone"
						ondragover={handleDragOver}
						ondragleave={handleDragLeave}
						ondrop={handleDrop}
					>
						<input type="file" accept=".csv" bind:this={fileInput} onchange={handleFileSelect} class="hidden" />
						<button onclick={() => fileInput?.click()} class="w-full py-12 px-8 flex flex-col items-center justify-center gap-4 cursor-pointer group">
							<div class="w-16 h-16 rounded-2xl {selectedFile ? 'bg-green-500/10' : 'bg-brand-400/10'} flex items-center justify-center group-hover:scale-110 transition-transform">
								{#if selectedFile}
									<CheckCircle class="w-8 h-8 text-green-600 dark:text-green-400" />
								{:else}
									<Upload class="w-8 h-8 text-brand-600 dark:text-brand-400" />
								{/if}
							</div>
							<div class="text-center">
								{#if selectedFile}
									<p class="text-lg font-semibold text-green-700 dark:text-green-400 mb-1">{selectedFile.name}</p>
									<p class="text-sm text-neutral-500">{(selectedFile.size / 1024).toFixed(1)} KB — klik atau drop ulang untuk ganti file</p>
								{:else}
									<p class="text-lg font-semibold text-neutral-900 dark:text-white mb-1">
										{isDragOver ? "Lepaskan file di sini" : "Drop file CSV atau klik untuk upload"}
									</p>
									<p class="text-sm text-neutral-500 dark:text-neutral-400">Hanya file .csv yang diperbolehkan</p>
								{/if}
							</div>
						</button>
					</div>

					<div class="mt-6 flex items-center gap-3">
						<button
							onclick={handleSubmit}
							disabled={!selectedFile || isUploading}
							class="flex-1 px-6 py-3 rounded-xl bg-brand-600 hover:bg-brand-700 text-white font-semibold transition-all dark:bg-brand-500 dark:hover:bg-brand-400 shadow-lg shadow-brand-600/25 disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
						>
							{#if isUploading}
								<svg class="animate-spin h-5 w-5" fill="none" viewBox="0 0 24 24">
									<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
									<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
								</svg>
								Mengupload...
							{:else}
								<Upload class="w-5 h-5" />
								Import CSV
							{/if}
						</button>
						{#if selectedFile}
							<button
								onclick={resetForm}
								class="px-4 py-3 rounded-xl border border-neutral-300 dark:border-neutral-700/80 text-sm font-medium text-neutral-700 dark:text-neutral-300 hover:bg-neutral-100 dark:hover:bg-neutral-800 transition-colors"
							>
								Batal
							</button>
						{/if}
					</div>
				</div>

				{#if result}
					<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 overflow-hidden" in:fly={{ y: 20, duration: 500 }}>
						<div class="flex items-center gap-2.5 px-6 py-4 border-b border-neutral-200/80 dark:border-white/[0.04]">
							<FileSpreadsheet class="w-5 h-5 text-neutral-500" />
							<h3 class="text-base font-semibold text-neutral-900 dark:text-white">Hasil Import</h3>
						</div>
						<div class="p-6">
							<div class="grid grid-cols-3 gap-4 mb-4">
								<div class="rounded-xl bg-neutral-100/80 dark:bg-neutral-800/50 p-4 text-center">
									<p class="text-xs font-medium text-neutral-500 dark:text-neutral-400 mb-1">Total</p>
									<p class="text-2xl font-bold text-neutral-900 dark:text-white font-mono">{result.total}</p>
								</div>
								<div class="rounded-xl bg-green-500/10 p-4 text-center">
									<div class="flex items-center justify-center gap-1.5 mb-1">
										<CheckCircle class="w-4 h-4 text-green-600 dark:text-green-400" />
										<span class="text-xs font-medium text-green-700 dark:text-green-400">Berhasil</span>
									</div>
									<p class="text-2xl font-bold text-green-700 dark:text-green-400 font-mono">{result.berhasil}</p>
								</div>
								<div class="rounded-xl bg-red-500/10 p-4 text-center">
									<div class="flex items-center justify-center gap-1.5 mb-1">
										<XCircle class="w-4 h-4 text-red-600 dark:text-red-400" />
										<span class="text-xs font-medium text-red-700 dark:text-red-400">Gagal</span>
									</div>
									<p class="text-2xl font-bold text-red-700 dark:text-red-400 font-mono">{result.gagal}</p>
								</div>
							</div>

							{#if result.catatan}
								<div class="mt-4 pt-4 border-t border-neutral-200/80 dark:border-white/[0.04]">
									<div class="flex items-center gap-2 mb-3">
										<AlertTriangle class="w-4 h-4 text-amber-500" />
										<span class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Catatan</span>
									</div>
									<div class="max-h-48 overflow-y-auto space-y-1">
										{#each result.catatan.split("\n").filter((l) => l.trim()) as line}
											<p class="text-xs text-neutral-600 dark:text-neutral-400 font-mono bg-neutral-100/80 dark:bg-neutral-800/50 px-3 py-1.5 rounded-lg">{line}</p>
										{/each}
									</div>
								</div>
							{/if}
						</div>
					</div>
				{/if}
			</div>

			<div class="lg:col-span-2 space-y-6" in:fly={{ y: 20, duration: 600, delay: 100 }}>
				<div class="rounded-2xl border border-neutral-200/80 dark:border-white/[0.06] bg-white dark:bg-neutral-925/50 p-6">
					<div class="flex items-center gap-3 mb-4">
						<div class="w-10 h-10 rounded-xl bg-amber-500/10 flex items-center justify-center">
							<AlertTriangle class="w-5 h-5 text-amber-600 dark:text-amber-400" />
						</div>
						<div>
							<h3 class="text-sm font-semibold text-neutral-900 dark:text-white">Petunjuk Format CSV</h3>
							<p class="text-xs text-neutral-500 dark:text-neutral-400">Pastikan format kolom sesuai</p>
						</div>
					</div>

					<div class="bg-neutral-100/80 dark:bg-neutral-800/50 rounded-xl p-4 mb-4">
						<p class="text-xs text-neutral-600 dark:text-neutral-400 leading-relaxed">
							Upload file CSV dengan format kolom:
						</p>
						<code class="block text-xs font-mono text-brand-600 dark:text-brand-400 mt-2 bg-neutral-200/50 dark:bg-neutral-800 px-3 py-2 rounded-lg">
							kelas_kode, nama, no_whatsapp, email, jenis_kelamin, nominal, tanggal_daftar, angkatan, usia, domisili
						</code>
						<p class="mt-3 text-xs font-medium text-warning">Nama, jenis_kelamin, tanggal_daftar, dan kode master Angkatan Pendaftaran wajib valid. Angkatan Kelas ditetapkan kemudian oleh Admin Kelas.</p>
					</div>

					<div class="space-y-2 text-xs text-neutral-600 dark:text-neutral-400">
						<div class="flex items-start gap-2">
							<span class="font-mono font-bold text-brand-600 dark:text-brand-400 shrink-0 w-24">kelas_kode</span>
							<span>Kode kelas tujuan (string)</span>
						</div>
						<div class="flex items-start gap-2">
							<span class="font-mono font-bold text-brand-600 dark:text-brand-400 shrink-0 w-24">nama</span>
							<span>Nama lengkap santri</span>
						</div>
						<div class="flex items-start gap-2">
							<span class="font-mono font-bold text-brand-600 dark:text-brand-400 shrink-0 w-24">no_whatsapp</span>
							<span>Nomor WhatsApp aktif (opsional)</span>
						</div>
						<div class="flex items-start gap-2">
							<span class="font-mono font-bold text-brand-600 dark:text-brand-400 shrink-0 w-24">email</span>
							<span>Email santri (opsional)</span>
						</div>
						<div class="flex items-start gap-2">
							<span class="font-mono font-bold text-brand-600 dark:text-brand-400 shrink-0 w-24">jenis_kelamin</span>
							<span>L untuk Laki-laki, P untuk Perempuan</span>
						</div>
						<div class="flex items-start gap-2">
							<span class="font-mono font-bold text-brand-600 dark:text-brand-400 shrink-0 w-24">nominal</span>
							<span>Nominal infaq (angka)</span>
						</div>
						<div class="flex items-start gap-2">
							<span class="font-mono font-bold text-brand-600 dark:text-brand-400 shrink-0 w-24">tanggal_daftar</span>
							<span>Wajib, format YYYY-MM-DD</span>
						</div>
						<div class="flex items-start gap-2">
							<span class="font-mono font-bold text-brand-600 dark:text-brand-400 shrink-0 w-24">angkatan</span>
							<span>Angkatan Pendaftaran, wajib memakai kode persis dari master (contoh: AKA38)</span>
						</div>
						<div class="flex items-start gap-2">
							<span class="font-mono font-bold text-brand-600 dark:text-brand-400 shrink-0 w-24">usia</span>
							<span>Usia santri (angka)</span>
						</div>
						<div class="flex items-start gap-2">
							<span class="font-mono font-bold text-brand-600 dark:text-brand-400 shrink-0 w-24">domisili</span>
							<span>Kota/domisili santri</span>
						</div>
					</div>

					<div class="mt-6 pt-4 border-t border-neutral-200/80 dark:border-white/[0.04]">
						<a
							href="/admin/import/template"
							class="w-full flex items-center justify-center gap-2 px-4 py-2.5 rounded-xl border border-brand-400/30 text-brand-600 dark:text-brand-400 hover:bg-brand-400/10 font-semibold transition-all text-sm"
						>
							<Download class="w-4 h-4" />
							Download Template CSV
						</a>
					</div>
				</div>
			</div>
		</div>
	</div>
</AppLayout>
