# Styling

This guide covers styling with Tailwind CSS in Laju Go.

## Overview

Laju Go uses **Tailwind CSS 4** for styling. The framework applies to both:

- **Svelte components** — Scoped styles in `.svelte` files
- **Landing page** — Inline styles in `templates/index.templ` (full CSS included at build time)

## Global Styles

Global styles and Tailwind directives are in `frontend/src/app.css`:

```css
@import "tailwindcss";
```

## Dark Mode

Laju Go includes dark mode support via a `DarkModeToggle` component. It toggles the `.dark` class on `<html>`:

```css
:root {
  --bg-primary: #ffffff;
  --text-primary: #1a1a1a;
}

.dark {
  --bg-primary: #1a1a1a;
  --text-primary: #ffffff;
}
```

## Component Styling

### In Svelte Components

Use Tailwind classes directly or scoped `<style>` blocks:

```svelte
<button class="btn-primary bg-purple-600 text-white px-4 py-2 rounded-lg">
  Click me
</button>

<style>
  .btn-primary:hover {
    opacity: 0.9;
  }
</style>
```

### In Templ Templates

The landing page uses inline `<style>` blocks with CSS variables:

```templ
<style>
  :root {
    --accent-purple: #8b5cf6;
    --text-secondary: #9ca3af;
  }
  .gradient-text {
    background: linear-gradient(135deg, var(--accent-purple), #a78bfa);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
  }
</style>
```

## Icons

The project uses [Lucide](https://lucide.dev/) icons via `lucide-svelte`:

```svelte
<script>
  import { Sun, Moon, User, Settings } from 'lucide-svelte';
</script>

<Sun size={20} />
<Moon size={20} />
<User />
```

## Responsive Design

Tailwind's responsive prefixes work as expected:

```svelte
<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
  <!-- Responsive grid -->
</div>
```

## Next Steps

- [Frontend Guide](frontend.md) — Component structure and Inertia.js
