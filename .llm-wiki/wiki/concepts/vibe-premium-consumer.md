> Related: [[design-principles]]

# Vibe: Premium / High-End Agency

**Source:** [taste-skill/soft-skill](https://github.com/Leonxlnx/taste-skill/tree/main/skills/soft-skill) — high-end-visual-design.

$150k+ agency-tier. Apple, Linear-tier. Haptic depth, cinematic motion, obsessive micro-interactions.

## Absolute Bans

- ❌ Inter, Roboto, Arial, Open Sans, Helvetica — ganti Geist, Clash Display, PP Editorial New
- ❌ Lucide, FontAwesome, Material Icons — pakai Phosphor Light, Remix Line
- ❌ 1px solid gray borders — ganti `border-white/[0.06-0.1]` dark, hairline di light
- ❌ Harsh shadows (`rgba(0,0,0,0.3)`)
- ❌ Symmetrical 3-column grids — variasi bento asimetris
- ❌ Linear ease — pakai `cubic-bezier(0.32, 0.72, 0, 1)`

## Vibe Archetypes (Pilih 1)

1. **Ethereal Glass** (SaaS/AI/Tech) — OLED `#050505`, radial mesh gradients, heavy `backdrop-blur-2xl`
2. **Editorial Luxury** (Lifestyle/Real Estate) — Warm creams `#FDFBF7`, sage, espresso. CSS noise overlay
3. **Soft Structuralism** (Consumer/Health) — Silver-grey, massive Grotesk type, floating airy components

## Layout Archetypes (Pilih 1)

1. **Asymmetrical Bento** — masonry grid, variasi col-span. Mobile: single column
2. **Z-Axis Cascade** — cards overlap dengan `rotate(-2deg)`, depth of field. Mobile: no rotation
3. **Editorial Split** — massive typography kiri, interactive cards kanan. Mobile: stack vertical

## Double-Bezel (Doppelrand) — Card Architecture

```
<div class="p-1.5 rounded-[2rem] bg-black/5 dark:bg-white/5 border border-white/10">
  <div class="rounded-[calc(2rem-0.375rem)] bg-white dark:bg-neutral-900
              shadow-[inset_0_1px_1px_rgba(255,255,255,0.15)]">
    <!-- content -->
  </div>
</div>
```

## Button-in-Button CTA

- Pill button `rounded-full px-6 py-3`
- Arrow icon di dalam wrapper `w-8 h-8 rounded-full bg-black/5` — flush dengan padding kanan

## Motion

- Nav: floating glass pill `mt-6 mx-auto w-max rounded-full`
- Hamburger → X morph dengan rotate
- Menu overlay: `backdrop-blur-3xl bg-black/80`. Links staggered delay
- Entry: `translate-y-16 blur-md opacity-0` → `0` over 800ms
- Button hover: `active:scale-[0.98]`. Inner icon diagonal translate

## Reference

- apple.com, linear.app, sotb.style

**Dials:** VARIANCE: 7-8 | MOTION: 6-7 | DENSITY: 3-4
