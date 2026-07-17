---
type: concept
status: stub
---

# Design Standards

Laju Go enforces high-end visual design standards on every page — both SSR `.templ` and Inertia Svelte.

## Principles

### 1. Visual Hierarchy
Focal point must be clear. Use size, weight, and contrast to guide the eye. Every page needs one primary action/ element.

### 2. Whitespace
Don't crowd. Minimum 16-24px padding. 32-48px spacing between sections.

### 3. Typography
Headings: bold/heavy weight. Body: clean sans-serif. Loose letter-spacing (tracking) for headlines.

### 4. Color — 60-30-10 Rule
- 60% neutral dominant
- 30% secondary
- 10% accent (minimal)
Avoid high-saturation colors unless explicitly requested.

### 5. Micro-interactions
Every interactive element must have hover states, smooth transitions, subtle shadows. No dead elements.

### 6. Mobile-First
All pages responsive. Test at 375px, 768px, 1440px viewports.

## Quality Check

If unsure about visual output, request a screenshot via `agent_browser` — review and iterate.

## Source

Migrated from [[sources/SRC-2026-07-06-001]] (AGENTS.md).
