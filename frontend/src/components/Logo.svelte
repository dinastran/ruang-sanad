<script lang="ts">
	interface Props {
		/** Pixel size of the logo square. Defaults to 36. */
		size?: number;
	}

	let { size = 36 }: Props = $props();
</script>

<svg
	width={size}
	height={size}
	viewBox="0 0 100 100"
	fill="none"
	xmlns="http://www.w3.org/2000/svg"
	aria-hidden="true"
>
	<defs>
		<linearGradient id="laju-logo-grad" x1="0%" y1="0%" x2="100%" y2="0%">
			<stop offset="0%" style="stop-color:#14b8a6;stop-opacity:1" />
			<stop offset="100%" style="stop-color:#22d3ee;stop-opacity:1" />
		</linearGradient>
	</defs>
	<path d="M30 10 H65 L55 50 H20 Z" fill="url(#laju-logo-grad)" />
	<path d="M20 58 H85 L75 90 H10 Z" fill="url(#laju-logo-grad)" />
	<rect
		x="70"
		y="58"
		width="20"
		height="32"
		transform="skewX(-14)"
		fill="white"
		fill-opacity="0.1"
	/>
</svg>