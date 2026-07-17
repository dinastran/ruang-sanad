# Frontend Development

This guide covers frontend development with Inertia.js in Laju Go.

## Overview

Laju Go ships with **Svelte 5** as the default frontend framework, but the frontend layer is **swappable** — thanks to [Inertia.js](https://inertiajs.com/), you can use React, Vue, or Svelte interchangeably without changing any backend code.

### How Inertia.js Works

Inertia.js bridges your backend and frontend without building a separate API:

```
Initial page load:
  Browser ──GET──► Server
                      │
                      ▼
               InertiaService.Render("Component", props)
                      │
                      ├──► JSON {component, props, url}
                      │         embedded in HTML shell
                      │
                      ◄── Full HTML page with Svelte/React/Vue app

Subsequent navigation:
  Browser ──XHR (X-Inertia: true)──► Server
                                        │
                                        ▼
                                 InertiaService.Render("Component", props)
                                        │
                                        ◄── JSON {component, props, url}
  Browser swaps components without full page reload
```

**Key insight**: Backend routes return `inertiaService.Render(componentName, props)` — they don't render HTML directly. The frontend framework handles rendering. This is what makes framework migration possible without touching Go code.

### Framework Support

| Framework | Inertia Adapter | Package | Status |
|-----------|----------------|---------|--------|
| **Svelte 5** | `@inertiajs/svelte` | Included | ✅ Default |
| **React** | `@inertiajs/react` | Swap required | ✅ Supported |
| **Vue 3** | `@inertiajs/vue3` | Swap required | ✅ Supported |

## Project Structure

```
frontend/
├── src/
│   ├── components/        # Reusable UI components
│   ├── pages/             # Page components (matched by Inertia component name)
│   ├── layouts/           # Layout components
│   ├── lib/               # Utilities (api, i18n, types, utils)
│   ├── main.ts            # Inertia app entry point (framework-specific)
│   └── app.css            # Global styles (Tailwind)
├── package.json
├── vite.config.js
└── tsconfig.json
```

The only framework-specific files are:

| File | Purpose | Changes when migrating |
|------|---------|-----------------------|
| `main.ts` | Inertia app initialization | Swap adapter + imports |
| `components/` | UI components | Rewrite in target framework |
| `pages/` | Page components | Rewrite in target framework |
| `layouts/` | Layout components | Rewrite in target framework |
| `package.json` | Dependencies | Swap framework packages |
| `vite.config.js` | Vite plugin | Swap Svelte → React plugin |

## Inertia.js Integration

### Page Component (Framework-Agnostic)

The backend renders pages by name. The frontend maps these names to components:

```go
// Go handler — never changes regardless of frontend framework
func (h *AppHandler) Dashboard(c *fiber.Ctx) error {
    return h.inertiaService.Render(c, "app/Dashboard", fiber.Map{
        "user": user,
        "stats": fiber.Map{
            "totalUsers":  100,
            "activeUsers": 50,
        },
    })
}
```

The component name `"app/Dashboard"` maps to `frontend/src/pages/app/Dashboard.svelte` (or `.tsx` for React).

### Server Props

Props from the server are available via the Inertia page object:

```go
// Backend — always the same pattern
inertiaService.Render(c, "PageName", fiber.Map{
    "key": value,
})
```

## Navigation

Inertia provides client-side navigation without page reloads:

### Inertia Links

```svelte
<!-- Svelte -->
<script>
  import { Link } from '@inertiajs/svelte';
</script>
<Link href="/app">Dashboard</Link>
```

```tsx
// React
import { Link } from '@inertiajs/react';
<Link href="/app">Dashboard</Link>
```

### Programmatic Navigation

```svelte
<!-- Svelte -->
<script>
  import { router } from '@inertiajs/svelte';
  router.visit('/app/profile');
</script>
```

```tsx
// React
import { router } from '@inertiajs/react';
router.visit('/app/profile');
```

## Form Handling

Forms use Inertia's `router.post()` / `router.put()` — no manual fetch/AJAX:

```svelte
<!-- Svelte -->
<script>
  import { router } from '@inertiajs/svelte';

  let email = $state('');
  let password = $state('');

  function submit() {
    router.post('/login', { email, password }, {
      onSuccess: () => { /* redirect happens automatically */ },
      onError: (errors) => { /* handle validation errors */ },
    });
  }
</script>
```

```tsx
// React
import { router } from '@inertiajs/react';

function submit(e: React.FormEvent) {
  e.preventDefault();
  router.post('/login', { email, password }, {
    onSuccess: () => {},
    onError: (errors) => {},
  });
}
```

---

## Svelte 5 (Default)

This section documents the current default frontend stack.

### Entry Point (`frontend/src/main.ts`)

```typescript
import { createInertiaApp } from '@inertiajs/svelte';
import { mount } from 'svelte';

createInertiaApp({
  resolve: (name) => {
    const pages = import.meta.glob('./pages/**/*.svelte', { eager: true });
    return pages[`./pages/${name}.svelte`];
  },
  setup({ el, App, props }) {
    mount(App, { target: el, props });
  },
});
```

### Component Basics

```svelte
<script>
  let count = $state(0);
  let user = $state({ name: '', email: '' });
  $: doubled = count * 2;
</script>

<button onclick={() => count++}>
  Count: {count}
</button>
```

### Props from Server

```svelte
<script>
  import { page } from '@inertiajs/svelte';
  const props = $page.props;
  const user = props.user;
</script>

<h1>Welcome, {user.name}!</h1>
```

### Icons

Landing page menggunakan inline SVG icons via Go helper `featureIcon()` di `templates/index.templ`. Halaman Inertia (protected pages) menggunakan Svelte components sesuai implementasi masing-masing.

> **Catatan**: Tidak ada `lucide-svelte` atau package icon terpisah. Semua icon landing page adalah inline SVGs.

---

## Migration Guide: Svelte → React

This guide walks through migrating from Svelte 5 to React using an **AI Agent** (Claude, Cursor, Copilot, etc.). The backend Go code requires **zero changes** — only the `frontend/` directory needs modification.

### What Changes

| Aspect | Before (Svelte) | After (React) |
|--------|-----------------|---------------|
| `package.json` deps | `@inertiajs/svelte` + `svelte` | `@inertiajs/react` + `react` + `react-dom` |
| `vite.config.js` | `@sveltejs/vite-plugin-svelte` | `@vitejs/plugin-react` |
| `main.ts` | `createInertiaApp` from `@inertiajs/svelte` | `createInertiaApp` from `@inertiajs/react` |
| Components | `.svelte` files | `.tsx` files |
| Reactivity | `$state`, `$derived` | `useState`, `useMemo` |
| Inertia adapter | `@inertiajs/svelte` | `@inertiajs/react` |

### What Does NOT Change

- All Go backend code (`handlers/`, `services/`, `routes/`, `main.go`)
- `inertiaService.Render()` calls in handlers
- Tailwind CSS classes
- `app.css`
- `templates/inertia.templ` (HTML shell)

### Step-by-Step Migration

#### Step 1: Update Dependencies

```bash
# Remove Svelte packages
npm uninstall svelte @sveltejs/vite-plugin-svelte @inertiajs/svelte @tsconfig/svelte

# Install React packages
npm install react react-dom @types/react @types/react-dom
npm install -D @vitejs/plugin-react
npm install @inertiajs/react
```

#### Step 2: Update Vite Config

```diff
// vite.config.js
- import { svelte } from '@sveltejs/vite-plugin-svelte';
+ import react from '@vitejs/plugin-react';

export default defineConfig({
-  plugins: [svelte()],
+  plugins: [react()],
   build: { outDir: 'dist', manifest: true },
});
```

#### Step 3: Update Entry Point

```diff
// frontend/src/main.ts
- import { createInertiaApp } from '@inertiajs/svelte';
- import { mount } from 'svelte';
+ import { createInertiaApp } from '@inertiajs/react';
+ import { createRoot } from 'react-dom/client';

createInertiaApp({
   resolve: (name) => {
-    const pages = import.meta.glob('./pages/**/*.svelte', { eager: true });
-    return pages[`./pages/${name}.svelte`];
+    const pages = import.meta.glob('./pages/**/*.tsx', { eager: true });
+    return pages[`./pages/${name}.tsx`];
   },
-  setup({ el, App, props }) {
-    mount(App, { target: el, props });
-  },
+  setup({ el, App, props }) {
+    createRoot(el).render(<App {...props} />);
+  },
});
```

#### Step 4: Convert Components (.svelte → .tsx)

This is the bulk of the work. Use the AI Agent prompt below.

---

### AI Agent Migration Prompt

Copy the following prompt and paste it to an AI Agent (Claude, Cursor, Copilot, etc.) along with your project files:

```
You are migrating a Svelte 5 + Inertia.js project to React + Inertia.js.
The Inertia.js backend (Go/Fiber) stays exactly the same — only the frontend/ directory changes.

## Migration Rules

1. Convert every `.svelte` file in `frontend/src/` to `.tsx`
2. Keep the same file names and directory structure (e.g., `pages/auth/Login.svelte` → `pages/auth/Login.tsx`)
3. Use TypeScript for all `.tsx` files
4. Replace Svelte syntax with React equivalents:

   | Svelte | React |
   |--------|-------|
   | `{#if condition}` | `{condition && (...)}` or ternary |
   | `{#each items as item}` | `{items.map(item => (...))}` |
   | `{#snippet name()}` | Helper function component |
   | `{@render name()}` | `<Name />` |
   | `$state()` | `useState()` |
   | `$derived()` | `useMemo()` |
   | `$effect()` | `useEffect()` |
   | `onclick={handler}` | `onClick={handler}` |
   | `bind:value={var}` | `value={var} onChange={e => setVar(e.target.value)}` |
   | `class:active={cond}` | `className={cond ? 'active' : ''}` |
   | `$page.props` from `@inertiajs/svelte` | `usePage().props` from `@inertiajs/react` |
   | `import { Link, router } from '@inertiajs/svelte'` | `import { Link, router } from '@inertiajs/react'` |
   | Svelte `export let prop` | React `interface Props { prop: type }` |
   | `<slot />` | `{children}` prop |
   | `import Component from './Component.svelte'` | `import Component from './Component'` |

5. Keep all Tailwind CSS classes exactly as they are
6. Keep `main.ts` as the entry point (update its content for React)
7. Do NOT change any Go backend code

## Files to Convert

- frontend/src/main.ts → (update entry point for React)
- frontend/src/components/*.svelte → *.tsx
- frontend/src/pages/**/*.svelte → *.tsx
- frontend/src/layouts/*.svelte → *.tsx

## Files to Keep (no changes needed)

- frontend/src/app.css
- frontend/src/lib/** (utility files may need minor import changes)
```

### Example Conversion

**Before** (`pages/auth/Login.svelte`):

```svelte
<script>
  import { router } from '@inertiajs/svelte';
  let email = $state('');
  let password = $state('');

  function submit() {
    router.post('/login', { email, password });
  }
</script>

<form onsubmit={(e) => { e.preventDefault(); submit(); }}>
  <input type="email" bind:value={email} />
  <input type="password" bind:value={password} />
  <button type="submit">Login</button>
</form>
```

**After** (`pages/auth/Login.tsx`):

```tsx
import { router } from '@inertiajs/react';
import { useState } from 'react';

export default function Login() {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');

  function submit(e: React.FormEvent) {
    e.preventDefault();
    router.post('/login', { email, password });
  }

  return (
    <form onSubmit={submit}>
      <input type="email" value={email} onChange={e => setEmail(e.target.value)} />
      <input type="password" value={password} onChange={e => setPassword(e.target.value)} />
      <button type="submit">Login</button>
    </form>
  );
}
```

### Post-Migration Checklist

- [ ] `npm run dev` starts without errors
- [ ] All pages render correctly (login, register, dashboard, profile)
- [ ] Form submissions work with Inertia POST/PUT
- [ ] Navigation via `Link` and `router.visit()` works
- [ ] Flash messages (error/success) display correctly
- [ ] File uploads function properly
- [ ] Dark mode toggle works (may need React conversion)
- [ ] `npm run build` produces valid `dist/` output
- [ ] `go build` succeeds with the new frontend build

### Testing

```bash
npm run dev:all      # Test in development
npm run build:all    # Test production build
```

### Rolling Back

```bash
git checkout -- frontend/
git checkout -- package.json package-lock.json
npm install
```

Go backend is untouched, so the app remains functional.

## Best Practices

### 1. Framework-Agnostic Go Handlers

```go
// ✅ Works with any frontend framework
func (h *AppHandler) Dashboard(c *fiber.Ctx) error {
    return h.inertiaService.Render(c, "app/Dashboard", fiber.Map{
        "user": user,
    })
}
```

### 2. Use Flash Messages

Flash messages set in Go handlers are auto-injected into Inertia props:

```go
// Go handler
h.store.Flash(c, "error", "Invalid email or password")
return c.Redirect("/login")
```

```svelte
<!-- Frontend: flash is available in props.flash -->
<script>
  import { page } from '@inertiajs/svelte';
  $: flash = $page.props.flash;
</script>
```

### 3. Use TypeScript

```svelte
<script lang="ts">
  interface User { id: number; name: string; email: string; }
  let { user }: { user: User } = $props();
</script>
```

## Next Steps

- [Architecture Guide](architecture.md) — How frontend fits into the architecture
- [Styling Guide](styling.md) — Tailwind CSS styling
- [Inertia.js Guide](inertia.md) — Deep dive into Inertia.js
