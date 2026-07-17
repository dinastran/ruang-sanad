---
type: source
title: "Observation: fiber-inertia integrated into laju-go"
slug: obs-2026-07-16-fiber-inertia-integrated-into-laju-go
status: observation
created: 2026-07-16
updated: 2026-07-16
relevance: high
observed_at: 2026-07-16T13:00:22.974Z
tags: ["inertia", "go", "fiber", "integration", "laju-go"]
source_context: "Integrating fiber-inertia into laju-go boilerplate"
---
# ⭐ Observation: fiber-inertia integrated into laju-go
Integrated fiber-inertia v0.0.0-20260716125151 into laju-go. Replaced custom app/services/inertia.go with a thin wrapper around fiberinertia.Inertia. Vite asset URLs, CSRF token, and flash messages handled via Config.Render and ShareFunc. Deleted templates/inertia.templ (no longer needed — root HTML rendered by library). Added inertia.Middleware() to Fiber app in main.go. All handlers continue using InertiaService.Render(c, component, props) unchanged. Build and all tests passing.
*Relevance: high*

*Context: Integrating fiber-inertia into laju-go boilerplate*

*Tags: inertia go fiber integration laju-go*
---
*Observed: 2026-07-16T13:00:22.974Z*