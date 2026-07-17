> Related: [[design-principles]]

# Vibe: Industrial Brutalism

**Source:** [taste-skill/brutalist-skill](https://github.com/Leonxlnx/taste-skill/tree/main/skills/brutalist-skill) — industrial-brutalist-ui.

Swiss typographic print + military terminal aesthetics. Rigid grids, extreme type scale, utilitarian color, analog degradation. Untuk data dashboard, portfolio, atau editorial yang mau feel "declassified blueprint."

## Visual Archetypes (Pilih 1, jangan campur)

### 1. Swiss Industrial Print (Light)

- Newsprint/off-white substrate. Monolithic heavy sans-serif. Rigid grids.
- Oversized numerals bleeding viewport. Primary red accent.

### 2. Tactical Telemetry / CRT Terminal (Dark)

- Dark mode exclusive. High-density tabular data. Monospace dominant.
- ASCII brackets, crosshairs, phosphor glow, scanlines.

## Typography

- **Macro (headers):** Neue Haas Grotesk Black, Inter Extra Bold, Archivo Black. Scale: `clamp(4rem, 10vw, 15rem)`. Tracking: -0.03em. Leading: 0.85. Exclusively UPPERCASE.
- **Micro (data):** JetBrains Mono, IBM Plex Mono, Space Mono. Fixed 10-14px. Tracking 0.05-0.1em. UPPERCASE.
- **Textural contrast:** Playfair Display, EB Garamond — tapi harus degraded dengan halftone/dithering.

## Color

### Swiss Print (Light)

| Role | Value |
|------|-------|
| Background | `#F4F4F0` / `#EAE8E3` (matte paper) |
| Foreground | `#050505` - `#111111` (carbon ink) |
| Accent | `#E61919` (hazard red) — satu-satunya accent |

### Tactical Telemetry (Dark)

| Role | Value |
|------|-------|
| Background | `#0A0A0A` / `#121212` (jangan pure black) |
| Foreground | `#EAEAEA` (white phosphor) |
| Accent | `#E61919` (red) |
| Terminal green | `#4AF626` — opsional, 1 elemen aja |

## Layout

- **CSS Grid deterministik.** No floating. Grid tracks + intersections.
- **Visible compartmentalization:** solid `1-2px` borders memisahkan zona. `<hr>` full-width.
- **Bimodal density:** data tight-packed + macro-typography luas bergantian.
- **`border-radius: 0`**. No rounded corners. Mekanik rigidity.

## Effects

- Halftone / 1-bit dithering (CSS `mix-blend-mode: multiply` + SVG dot pattern)
- CRT scanlines (`repeating-linear-gradient`)
- Mechanical noise overlay (SVG static filter, global)
- ASCII framing: `[ DELIVERY SYSTEMS ]`, `>>>`, `\\\\`, crosshairs `+`

## Reference

- brutalistwebsites.com, craigslist.org

**Dials:** VARIANCE: 8-10 | MOTION: 2 | DENSITY: 4-6
