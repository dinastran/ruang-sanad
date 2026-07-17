---
type: entity
status: stub
---

# Templ

**templ** is a type-safe HTML templating language for Go used in Laju Go for server-rendered pages and the Inertia HTML shell.

## Workflow

- Edit `.templ` files → run `templ generate` → commit both `.templ` and `*_templ.go`
- Air does NOT watch `.templ` files by default — regenerate manually or add `"templ"` to `.air.toml` `include_ext`
- Rendering: `templates.ComponentName(args...).Render(ctx, writer)` instead of `c.Render("name", data)`

## SVG Icons Gotcha

String params in templ are HTML-escaped. Do not pass SVG strings directly as parameters. Use a Go helper function that returns the SVG string, then use `@templ.Raw(helper(key))` in markup.

Example: `@templ.Raw(featureIcon("auth"))`

## Source

Captured from [[sources/SRC-2026-07-06-001]].
