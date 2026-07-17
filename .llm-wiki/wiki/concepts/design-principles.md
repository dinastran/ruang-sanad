# Design Principles

> **Boilerplate frontend principles.** Baca brief → infer design direction → apply anti-slop rules.
> Framework agnostic. Contextual — tidak semua aturan aktif otomatis.

---

## 0. BRIEF INFERENCE

Sebelum nulis kode, **baca ruangannya.** Jangan lompat ke default aesthetic.

### 0.A Signals

1. **Page kind** — landing (SaaS/consumer/agency), portfolio, editorial, dashboard/admin, auth page
2. **Vibe words** — "minimalist", "premium", "playful", "brutalist", "editorial", "dark tech", "Apple-y", "B2B serious"
3. **Reference** — URL, screenshot, produk, brand kompetitor
4. **Audience** — B2B procurement vs consumer vs developer. Audience pilih aesthetic, bukan selera kamu
5. **Quiet constraints** — aksesibilitas, sektor publik, regulated industry. Ini **override** aesthetic preference

### 0.B Design Read

Sebelum generate, output satu baris:

> *"Reading this as: \<page kind> for \<audience>, with a \<vibe> language."*

### 0.C Anti-Default Discipline

Jangan default ke: purple gradient, centered hero di atas dark mesh, 3 equal feature cards, glassmorphism di semua card, Inter + slate-900. Ini LLM defaults. **Reach past them deliberately.**

---

## 1. THE THREE DIALS

Setelah design read, set tiga dial. Semua keputusan layout/motion/density di bawah digate oleh ini.

Lihat juga: [[vibe-minimalist]], [[vibe-premium-consumer]], [[vibe-playful-experimental]], [[vibe-dark-tech]], [[vibe-brutalist]]

> 📖 **Vibe library bersifat opsional.** Ini cuma referensi — setiap project bisa punya vibe sendiri.
> Mau nambah vibe? Buat page `concepts/vibe-<nama>` dan link dari sini.
> Mau skip vibe? Langsung ke pre-flight checklist aja.

| Dial | 1 | 10 | Default |
|------|---|----|---------|
| **DESIGN_VARIANCE** | Perfect Symmetry | Artsy Chaos | **8** |
| **MOTION_INTENSITY** | Static | Cinematic / Physics | **6** |
| **VISUAL_DENSITY** | Airy gallery | Cockpit packed | **4** |

### Dial Inference

| Signal / Brief | VARIANCE | MOTION | DENSITY | Related vibe |
|---------------|----------|--------|---------|-------------|
| Minimalist / clean / editorial / Linear-style | 5-6 | 3-4 | 2-3 | [[vibe-minimalist]] |
| Premium consumer / Apple-y / luxury / brand | 7-8 | 5-7 | 3-4 | [[vibe-premium-consumer]] |
| Playful / wild / Awwwards / experimental / agency | 9-10 | 8-10 | 3-4 | [[vibe-playful-experimental]] |
| Dark tech / devtool / hacker | 4-5 | 2-3 | 5-7 | [[vibe-dark-tech]] |
| Brutalist / raw / anti-design | 8-10 | 2-3 | 4-6 | [[vibe-brutalist]] |
| Trust-first / public-sector / regulated / aksesibilitas | 3-4 | 2-3 | 4-5 | — |
| Landing page / portfolio / marketing site (default) | 7-9 | 6-8 | 3-5 | — |
| Redesign - preserve | match existing | +1 | match existing | — |
| Redesign - overhaul | +2 | +2 | match existing | — |

### Use-Case Presets

| Use case | VARIANCE | MOTION | DENSITY |
|----------|----------|--------|---------|
| Landing (SaaS, mainstream) | 7 | 6 | 4 |
| Landing (Agency / creative) | 9 | 8 | 3 |
| Landing (Premium consumer) | 7 | 6 | 3 |
| Portfolio (Designer / studio) | 8 | 7 | 3 |
| Portfolio (Developer) | 6 | 5 | 4 |
| Editorial / Blog | 6 | 4 | 3 |
| Public-sector service | 3 | 2 | 5 |
| Redesign - preserve | match | match+1 | match |
| Redesign - overhaul | +2 | +2 | match |

---

## 2. ANTI AI-SLOP RULES

### 🎨 Warna

| ❌ AI Slop | ✅ Ganti |
|-----------|---------|
| Purple/blue glow + dark mesh hero | Gradient solid yang intentional, atau radial gradient lembut |
| Warm beige/cream bg (`#f5f1ea`, `#efeae0`) | Cool neutrals (slate/zinc/stone scale) |
| Brass/ochre/oxblood accent (`#b08947`, `#9a2436`) | High-contrast singular accent (emerald, electric blue, deep rose) |
| Random gradient di tiap section | Satu gradient di hero aja |
| Pure black `#000` dark mode | Navy-tinted near-black (`#0f172a`, `#070b16`) |
| Opacity stacking 3+ layer | Max 2 layer overlay |
| **Premium-consumer palette ban** — beige+brass+espresso (`#f5f1ea` / `#b08947` / `#1a1714`) | Cold luxury, forest, black+tan, cobalt+cream, terracotta+slate |
| Max 1 accent per page | Satu accent, konsisten dari hero sampe footer |

### 🔤 Tipografi

| ❌ AI Slop | ✅ Ganti |
|-----------|---------|
| Inter sebagai default font | OK (boleh), tapi jangan import ulang via Google Fonts. Geist > Inter untuk modern feel |
| Fraunces / Instrument Serif | **Banned.** Dua font favorit LLM |
| Serif buat "creative"/"premium" | **Default sans-serif.** Serif cuma kalo brand explicit nyebut |
| Mixed-family emphasis (sans headline + serif word) | Italic/bold dari font yang SAMA |
| Headline > 8 words | Max 8 words display; >8 pake 2 lines |
| Em-dash (`—`) tanpa spasi | `—` with spaces (`word — word`) |

### 📐 Layout

| ❌ AI Slop | ✅ Ganti |
|-----------|---------|
| Centered hero + CTA | Split screen, left-aligned + asset, asymmetric |
| 3 equal feature cards | Variasi ukuran (1 besar + 2 kecil), grid asimetris |
| Cards-inside-cards-inside-cards | Flat hierarchy, border/divider cukup |
| Left text + right image tiap section | Variasi: full-width, grid, overlap, background-image |
| Zigzag alternation > 2 sections | Break dengan full-width section, bento, marquee |
| Glassmorphism di semua card | Border tipis + bg solid. Glass cuma untuk overlay/navbar |
| Eyebrow di **setiap** section | Max 1 eyebrow per 3 sections |
| Split-header (left headline + right explainer) | Stack vertical. Split cuma kalo ada alasan komposisi |
| Button text wrap di desktop | Perpendek label. Max 3 words untuk primary CTA |
| Duplicate CTA intent | Satu label per intent. "Contact" + "Get in touch" = pilih satu |

### 🧩 Komponen & State

| ❌ AI Slop | ✅ Ganti |
|-----------|---------|
| Loading = spinner saja | Skeletal loader sesuai shape final layout |
| Empty state = kosong | Beautiful empty state + instruksi |
| Hanya implementasi "success state" | Selalu implement: loading, empty, error, success |
| Keyboard/Screen reader diabaikan | Focus visible, label, role, contrast WCAG AA |
| Placeholder-as-label | Label di atas input. Placeholder cuma contoh |

### 📸 Gambar

Landing dan portfolio adalah **visual product.** Text-only pages dengan fake-screenshot divs adalah slop.

Prioritas:

1. **Generate gambar** — kalo ada image-gen tool di environment, pakai
2. **Real photos** — picsum.photos, Unsplash, atau brand assets
3. **Last resort** — placeholder `<!-- TODO: hero image -->`, jangan fake divs

**Logo wall** harus real SVG. Jangan plain text wordmarks. Source: Simple Icons, atau inline SVG monogram.

### 🎬 Motion

| Prinsip | Aturan |
|---------|--------|
| Entry animasi | Ada, tapi subtle. `fade` + `translateY` cukup |
| Scroll-triggered | Hanya untuk hero dan section divider |
| Hover cards | `translateY(-2px)` + shadow, atau border highlight |
| Button press | `scale(0.97)` — tactile feedback |
| **Banned** | Infinite auto-scroll carousel, parallax everywhere, confetti |

---

## 3. DESIGN SYSTEM MAP

Pilih foundation sesuai brief. Jangan invent CSS untuk yang punya official package.

| Brief | Official Design System |
|-------|----------------------|
| Microsoft/enterprise/dashboard | Fluent UI |
| Google-ish UI | Material Web (Material 3) |
| IBM-style B2B | Carbon |
| GitHub-style devtool | Primer |
| Public-sector UK | GOV.UK Frontend |
| US public-sector | USWDS |
| Modern accessible React | Radix Themes |
| Tailwind SaaS (default) | shadcn/ui (with customization) |

**Satu sistem per project.** Jangan mix Fluent + Carbon dalam satu tree.

Untuk aesthetic murni (bukan sistem resmi):

| Aesthetic | Implementasi |
|-----------|-------------|
| Glassmorphism | `backdrop-filter` + border highlight + solid fallback |
| Bento grid | CSS Grid. No library owns this |
| Brutalism | Native CSS, monospace, raw borders |
| Editorial | Serif type, asymmetric grid, whitespace |
| Dark tech | Mono + accent neon, terminal motifs |

---

## 4. LAYOUT DISCIPLINE (Hard Rules)

Aturan ini **wajib**. Melanggar = shipping broken work.

### Hero

- **MUST fit in viewport** — headline max 2 lines, subtext max 20 words, CTA visible tanpa scroll
- Font scale: `text-4xl md:text-5xl lg:text-6xl` default. Jangan `text-7xl` untuk headline >6 words
- Top padding max `pt-24` — lebih dari itu floating
- Max **4 text elements**: eyebrow (opsional) + headline + subtext + CTAs
- **Banned in hero:** tagline below CTA, trust strip, pricing teaser, feature list, avatar row, logo wall
- Logo wall = dedicated section **di bawah hero**

### Navigation

- Satu line di desktop. Kalo tidak muat di `lg`: kondens label, drop secondary items, atau hamburger
- Height max 80px, default 64-72px

### Bento Grids

- Jumlah cell = jumlah konten. 3 items → 3 cells. **Jangan ada empty cell**
- Rhythm: variasi komposisi, jangan 6 left-image-right-text berturut-turut
- Butuh variasi visual: minimal 2-3 cell punya gambar/gradient/pattern, bukan text-only

### Section Repetition

- Satu layout family bisa muncul **max 1 kali** per page
- Landing 8 sections harus pakai minimal **4 layout families** berbeda
- Zigzag (image+text alternating) max **2 sections berturut-turut**. Ke-3 = fail

### Mobile

- Setiap multi-column layout harus declare `< 768px` fallback
- Navigation single-line di mobile → hamburger/drawer

---

## 5. PRE-FLIGHT CHECKLIST

Sebelum declare selesai, cek ini:

- [ ] Hero fits viewport? Headline ≤2 lines, subtext ≤20 words?
- [ ] Navigation satu line di desktop?
- [ ] Eyebrow count ≤ ceil(sectionCount / 3)?
- [ ] Zigzag alternation ≤ 2 consecutive?
- [ ] Loading + empty + error states ada?
- [ ] Button text tidak wrap?
- [ ] CTA intent tidak duplikat?
- [ ] Contrast WCAG AA (4.5:1 body, 3:1 large)?
- [ ] Satu accent warna per page?
- [ ] no `#000` for dark mode bg?
- [ ] Gambar real (generated/photo)? Bukan fake divs?
- [ ] Focus visible + keyboard navigable?
- [ ] `min-h-[100dvh]` bukan `h-screen` untuk hero?
