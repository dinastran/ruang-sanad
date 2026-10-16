<script lang="ts">
	import { untrack } from "svelte";
	import type { Tagihan, WaTemplate } from "@lib/types";
	import { getCSRFToken } from "@lib/utils/csrf";
	import { X } from "lucide-svelte";

	interface Props {
		target: Tagihan;
		templates: WaTemplate[];
		onclose: () => void;
		onsent?: () => void;
	}

	let { target, templates, onclose, onsent }: Props = $props();
	let activeTemplates = $derived(templates.filter((t) => t.is_aktif));

	const rupiah = (n: number) =>
		new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 0 }).format(n);

	function renderTemplate(body: string, t: Tagihan): string {
		const values: Record<string, string> = {
			"{nama}": t.santri_nama || "",
			"{id_mahasantri}": t.id_mahasantri || "",
			"{bulan_ke}": String(t.bulan_ke ?? ""),
			"{pertemuan_ke}": String(t.pertemuan_ke ?? ""),
			"{nominal}": new Intl.NumberFormat("id-ID").format(t.nominal || 0),
			"{tanggal_tagih}": t.tanggal_tagih || "",
			"{jatuh_tempo}": t.jatuh_tempo || "-",
			"{kelas}": t.kelas_nama || "-",
			"{guru}": t.guru_nama || "-",
			"{angkatan}": t.angkatan || "-",
			"{angkatan_kelas}": t.angkatan_kelas || "-",
			"{frekuensi}": t.frekuensi || "-",
		};
		let result = body;
		for (const [key, value] of Object.entries(values)) result = result.split(key).join(value);
		return result;
	}

	// The dialog is mounted per target, so the first active template is the
	// starting point for this follow-up.
	const first = untrack(() => templates.find((t) => t.is_aktif));
	let templateID = $state(first ? String(first.id) : "");
	let message = $state(untrack(() => (first ? renderTemplate(first.body, target) : "")));
	let sending = $state(false);

	function chooseTemplate() {
		const current = templates.find((t) => String(t.id) === templateID);
		message = current ? renderTemplate(current.body, target) : "";
	}

	async function send() {
		if (!message.trim()) return;
		sending = true;
		const popup = window.open("", "_blank");
		try {
			const response = await fetch(`/app/keuangan/tagihan/${target.id}/follow-up`, {
				method: "POST",
				headers: {
					"Content-Type": "application/json",
					"X-XSRF-TOKEN": getCSRFToken(),
				},
				body: JSON.stringify({ template_id: Number(templateID || 0), message }),
			});
			const data = await response.json();
			if (!response.ok) {
				popup?.close();
				alert(data.error ?? data.message ?? "Gagal membuat tautan WhatsApp");
				return;
			}
			if (popup) popup.location.href = data.url;
			else window.location.href = data.url;
			onclose();
			onsent?.();
		} finally {
			sending = false;
		}
	}
</script>

<div class="fixed inset-0 z-50 bg-neutral-950/55 flex items-center justify-center p-4">
	<section class="w-full max-w-xl rounded-2xl bg-white dark:bg-neutral-925 p-6 shadow-xl">
		<div class="flex justify-between gap-4"><div><h2 class="text-xl font-bold text-neutral-900 dark:text-white">Follow-up WhatsApp</h2><p class="text-sm text-neutral-500 mt-1">{target.santri_nama} · {rupiah(target.nominal)}</p></div><button onclick={onclose} class="text-neutral-500"><X /></button></div>
		<div class="mt-5 space-y-4">
			<label class="block text-sm font-medium">Template
				<select bind:value={templateID} onchange={chooseTemplate} class="mt-1 w-full px-3 py-2.5 rounded-xl bg-neutral-100 dark:bg-neutral-800">
					{#each activeTemplates as template}<option value={String(template.id)}>{template.nama}</option>{/each}
				</select>
			</label>
			<label class="block text-sm font-medium">Pesan
				<textarea bind:value={message} rows="9" class="mt-1 w-full px-3 py-2.5 rounded-xl bg-neutral-100 dark:bg-neutral-800"></textarea>
			</label>
			<p class="text-xs text-neutral-500">Pesan boleh diedit untuk pengiriman ini tanpa mengubah template utama.</p>
			<button onclick={send} disabled={sending || !message.trim()} class="w-full py-2.5 rounded-xl bg-brand-600 text-white font-semibold disabled:opacity-50">{sending ? "Menyiapkan WhatsApp..." : "Buka WhatsApp & Catat FU"}</button>
		</div>
	</section>
</div>
