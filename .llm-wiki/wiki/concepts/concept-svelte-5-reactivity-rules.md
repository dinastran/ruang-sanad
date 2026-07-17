---
type: concept
status: stub
---

# Svelte 5 Reactivity Rules

Laju Go enforces strict Svelte 5 rune usage rules to prevent anti-patterns.

## Allowed `$effect` Usage

`$effect` is only for side effects that interact with non-reactive APIs:
- `document.title = user.name`
- `localStorage.setItem('theme', theme)`
- `setInterval`, `addEventListener`

## Prohibited `$effect` Usage

- **Derived state**: Use `$derived()` instead of `$effect(() => filtered = items.filter(...))`
- **Prop initialization**: Use `$state(initial)` or direct assignment instead of `$effect(() => count = initial)`

## Correct Patterns

```svelte
<!-- ✅ Derived state -->
let filtered = $derived(items.filter(...));

<!-- ✅ Prop initialization -->
let count = $state(initial ?? 0);

<!-- ✅ External side effect (ok) -->
$effect(() => { document.title = user.name; });
```

## Source

Captured from [[sources/SRC-2026-07-06-001]].
