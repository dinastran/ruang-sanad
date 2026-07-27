<script lang="ts">
	interface Props {
		/** Pixel size of the logo square. Defaults to 36. */
		size?: number;
	}

	let { size = 36 }: Props = $props();
</script>

<img
	src="/public/android-chrome-512x512.png"
	alt="Ruang Sanad"
	width={size}
	height={size}
	style="width:{size}px;height:{size}px"
	class="object-contain shrink-0"
	loading="eager"
	decoding="async"
/>
