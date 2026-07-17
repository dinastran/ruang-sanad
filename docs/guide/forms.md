# Forms

This guide covers form handling with Inertia.js in Laju Go.

## Overview

Forms use Inertia's `router.post()` / `router.put()` for submission. The backend redirects after processing, and Inertia follows the redirect automatically.

## Basic Form (Login)

### Frontend (Svelte)

```svelte
<script>
  import { router } from '@inertiajs/svelte';

  let email = $state('');
  let password = $state('');
  let processing = $state(false);

  function submit() {
    processing = true;
    router.post('/login', { email, password }, {
      onSuccess: () => { processing = false; },
      onError: (errors) => {
        console.log('Validation errors:', errors);
        processing = false;
      },
    });
  }
</script>

<form onsubmit={(e) => { e.preventDefault(); submit(); }}>
  <input type="email" bind:value={email} required />
  <input type="password" bind:value={password} required />
  <button type="submit" disabled={processing}>
    {processing ? 'Loading...' : 'Login'}
  </button>
</form>
```

### Backend (Go)

```go
func (h *AuthHandler) Login(c *fiber.Ctx) error {
    var req models.LoginRequest
    if err := c.BodyParser(&req); err != nil {
        return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
    }

    user, err := h.authService.Login(req.Email, req.Password)
    if err != nil {
        h.store.Flash(c, "error", "Invalid email or password")
        return c.Redirect("/login")
    }

    sess, _ := h.store.Get(c)
    sess.Set("user_id", user.ID)
    sess.Save()
    return c.Redirect("/app")
}
```

## Form with Flash Errors

Flash messages are auto-injected into Inertia props:

```svelte
<script>
  import { page } from '@inertiajs/svelte';
  $: flash = $page.props.flash;
</script>

{#if flash?.error}
  <div class="error">{flash.error}</div>
{/if}
```

## File Upload Form

```svelte
<script>
  import { router } from '@inertiajs/svelte';

  let avatar = $state(null);

  function upload() {
    const formData = new FormData();
    formData.append('avatar', avatar);
    router.post('/upload', formData, {
      onSuccess: () => alert('Upload successful!'),
    });
  }
</script>

<input type="file" accept="image/*" onchange={e => avatar = e.target.files[0]} />
<button onclick={upload}>Upload</button>
```

## Form Best Practices

1. **Always redirect** on POST — Inertia follows redirects automatically
2. **Use flash messages** for user feedback instead of JSON responses
3. **Validate early** in the handler before calling services
4. **Disable submit button** while processing to prevent double submits

## Next Steps

- [Validation Guide](validation.md) — Input validation
- [Frontend Guide](frontend.md) — Inertia.js integration details
