<script lang="ts">
	import Logo from "@components/Logo.svelte";
	import { firePixelEvents, initPixel, setPixelUser, type PixelEvent } from "@lib/metaPixel";
	import { onMount } from "svelte";
	import { LoaderCircle, MessageCircle } from "lucide-svelte";

	// Mode-WA link with Pixel events: fire them, then open WhatsApp. The wait
	// is short and capped so a slow or blocked Pixel never keeps the visitor
	// from WhatsApp.
	interface Props {
		wa_url: string;
		judul?: string;
		meta?: { pixel_ids: string[]; events: PixelEvent[]; user?: Record<string, string> };
	}
	let { wa_url, judul = "", meta }: Props = $props();

	onMount(() => {
		if (meta && initPixel(meta.pixel_ids)) {
			setPixelUser(meta.user);
			firePixelEvents(meta.events, { content_name: judul });
		}
		const t = setTimeout(() => window.location.replace(wa_url), meta ? 700 : 0);
		return () => clearTimeout(t);
	});
</script>

<svelte:head><title>Membuka WhatsApp · Ruang Sanad</title></svelte:head>

<main class="flex min-h-[100dvh] items-center justify-center bg-neutral-50 px-4 dark:bg-neutral-950">
	<div class="w-full max-w-sm text-center">
		<div class="flex items-center justify-center gap-2.5"><Logo size={32} /><span class="font-semibold text-neutral-900 dark:text-white">Ruang Sanad</span></div>
		<p class="mt-8 inline-flex items-center gap-2 text-sm text-neutral-600 dark:text-neutral-400" role="status">
			<LoaderCircle size={16} class="animate-spin motion-reduce:animate-none" aria-hidden="true" /> Membuka WhatsApp…
		</p>
		<a href={wa_url} class="mt-5 inline-flex min-h-12 w-full items-center justify-center gap-2 rounded-xl bg-brand-700 px-5 text-base font-semibold text-white transition hover:bg-brand-800 dark:bg-brand-600">
			<MessageCircle size={18} aria-hidden="true" /> Buka WhatsApp
		</a>
	</div>
</main>
