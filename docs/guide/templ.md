# Templ Templates

This guide covers [templ](https://templ.guide/) — the type-safe HTML template engine used in Laju Go.

## Overview

[templ](https://templ.guide/) compiles Go templates into type-safe Go code at build time. Instead of runtime parsing and reflection (like `html/template`), templ generates `.go` files that are compiled as part of your application.

**Benefits**:

- **Type safety** — Template parameters are checked at compile time
- **IDE support** — Autocomplete, refactoring, and error detection
- **Performance** — No runtime template parsing overhead
- **Go syntax** — Use Go control flow directly in templates

## Template Files

Laju Go uses two templ templates:

```
templates/
├── inertia.templ           # Inertia.js base HTML shell
├── inertia_templ.go        # Generated Go code (DO NOT EDIT)
├── index.templ             # Public landing page
└── index_templ.go          # Generated Go code (DO NOT EDIT)
```

### InertiaPage (`inertia.templ`)

The HTML shell for Inertia.js-powered pages. Renders the Svelte app mount point with embedded JSON page data:

```templ
templ InertiaPage(title string, pageJSON string, isDev bool, viteURL string, csrfToken string, mainJS string, mainCSS string, styles []string) {
    <!doctype html>
    <html lang="en">
        <head>
            <meta charset="UTF-8"/>
            <meta name="viewport" content="width=device-width, initial-scale=1.0"/>
            <title>{ title } - Laju</title>
            <meta name="csrf-token" content={ csrfToken }/>
            for _, style := range styles {
                <link rel="stylesheet" href={ style }/>
            }
        </head>
        <body class="bg-gray-50 text-gray-900">
            <div id="app"></div>
            @templ.Raw(`<script data-page="app" type="application/json">` + pageJSON + `</script>`)
            if isDev {
                <script type="module" src={ viteURL + "/@vite/client" }></script>
                <link rel="stylesheet" href={ viteURL + "/src/app.css" }/>
                <script type="module" src={ viteURL + "/src/main.ts" }></script>
            } else {
                <link rel="stylesheet" href={ mainCSS }/>
                <script type="module" src={ mainJS }></script>
            }
        </body>
    </html>
}
```

**Parameters**:

| Parameter | Type | Purpose |
|-----------|------|---------|
| `title` | `string` | Page title |
| `pageJSON` | `string` | JSON-encoded Inertia page data |
| `isDev` | `bool` | Whether running in development mode |
| `viteURL` | `string` | Vite dev server URL (empty in production) |
| `csrfToken` | `string` | CSRF token embedded as meta tag |
| `mainJS` | `string` | Hashed JS file path (production only) |
| `mainCSS` | `string` | Hashed CSS file path (production only) |
| `styles` | `[]string` | Additional stylesheet URLs |

### LandingPage (`index.templ`)

The public landing page — a full standalone HTML page with hero section, features, tech stack, and CTA. Used at the `/` route.

```templ
templ LandingPage(title string, isDev bool, viteURL string, mainCSS string) {
    <!doctype html>
    <html lang="en" class="dark">
        <!-- Full landing page with CSS animations, gradients, scroll-reveal, testimonials, FAQ, footer -->
    </html>
}
```

## Usage in Handlers

### Inertia Page Rendering

The `InertiaService` handles rendering automatically:

```go
// Inertia handles both initial HTML load and XHR navigation
return h.inertiaService.Render(c, "app/Dashboard", fiber.Map{
    "user": user,
})
```

This internally calls `templates.InertiaPage(...)` for initial page loads.

### Direct Template Rendering

For non-Inertia pages (like the landing page), templates are rendered directly:

```go
// app/handlers/public.go
func (h *PublicHandler) Index(c *fiber.Ctx) error {
    assetData := h.assetService.GetAssetData()
    isDev := assetData["IsDev"].(bool)
    viteURL, _ := assetData["ViteURL"].(string)
    mainCSS, _ := assetData["MainCSS"].(string)

    c.Set("Content-Type", "text/html; charset=utf-8")
    return templates.LandingPage("Welcome to Laju", isDev, viteURL, mainCSS).
        Render(c.Context(), c.Response().BodyWriter())
}
```

## templ Syntax Reference

### Variables

```templ
<div>{ myVariable }</div>
```

### Conditionals

```templ
if condition {
    <div>Condition is true</div>
} else {
    <div>Condition is false</div>
}
```

### Loops

```templ
for _, item := range items {
    <li>{ item.Name }</li>
}
```

### Raw HTML (Unescaped)

```templ
@templ.Raw(`<script>...</script>`)
```

> Use `templ.Raw()` for injecting raw HTML/JSON. Be careful with user content — templ auto-escapes `{ }` variables by default.

### Component Parameters

```templ
templ MyComponent(name string, items []string) {
    <div>
        <h2>Hello, { name }</h2>
        for _, item := range items {
            <span>{ item }</span>
        }
    </div>
}
```

## Workflow

### 1. Edit `.templ` files

Make changes in `templates/*.templ`.

### 2. Regenerate Go code

```bash
templ generate
```

This creates/updates `*_templ.go` files.

### 3. Commit both files

Both the `.templ` source and generated `_templ.go` files must be committed:

```bash
git add templates/
git commit -m "update templates"
```

> Air does **not** watch `.templ` files by default. After editing templates, run `templ generate` manually (or configure Air to include `.templ` in `include_ext`).

## Best Practices

### 1. Keep Templates Focused

Each template should have a single responsibility. The `InertiaPage` handles the base layout; page-specific content comes from Svelte components via Inertia.

### 2. Use `@templ.Raw()` Sparingly

Prefer `{ }` interpolation for automatic escaping. Only use `templ.Raw()` for safe content like pre-encoded JSON.

### 3. Use Asset Service for URLs

Always use `AssetService` for asset URLs instead of hardcoding:

```go
// ✅ Good: AssetService handles dev vs production
assetData := h.assetService.GetAssetData()

// ❌ Bad: Hardcoded paths break in production
"/dist/main.js"
```

### 4. Handle Dev vs Production

The `InertiaPage` template checks `viteServerURL` to switch between Vite dev server and production hashed assets:

```templ
if viteServerURL != "" {
    // Dev: Vite HMR
} else {
    // Prod: static files from manifest
}
```

## Next Steps

- [Architecture Guide](architecture.md) — How templates fit into the architecture
- [Frontend Guide](frontend.md) — Svelte 5 components and Inertia.js
