<script lang="ts">
	import AppLayout from "@layouts/AppLayout.svelte";
	import RiwayatAbsensiDashboard from "@components/koordinator/riwayat/RiwayatAbsensiDashboard.svelte";
	import type { GuruRingkas, User } from "@lib/types";
	import type { RiwayatAbsensi } from "@lib/riwayatAbsensi";

	interface Props { user?: User; riwayat?: RiwayatAbsensi[]; guruList?: GuruRingkas[]; guruID?: number; }
	let { user, riwayat = [], guruList = [], guruID }: Props = $props();
</script>

<svelte:head><title>Riwayat Absensi Guru</title></svelte:head>

<AppLayout {user}>
	<main class="mx-auto max-w-6xl space-y-6 px-4 py-8 sm:px-6">
		<header>
			<p class="text-xs font-bold uppercase tracking-[0.16em] text-brand-600 dark:text-brand-400">Koordinator Guru</p>
			<h1 class="mt-2 text-2xl font-bold tracking-tight text-neutral-900 sm:text-3xl dark:text-white">Riwayat absensi guru</h1>
			<p class="mt-1.5 text-neutral-600 dark:text-neutral-400">Kehadiran Pembinaan dan Rapat, serta pelaksanaan Kunjungan Kelas.</p>
		</header>
		<!-- ?guru=<id> dari analisis alfa langsung memfilter ke guru tersebut. -->
		<RiwayatAbsensiDashboard {riwayat} {guruList} guruAwal={guruID ? String(guruID) : ""} />
	</main>
</AppLayout>
