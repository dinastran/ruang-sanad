# Validation

This guide covers input validation patterns in Laju Go.

## Overview

Validation happens at two levels:

1. **Handler level** — Basic field checks (presence, format) before calling services
2. **Service level** — Business rule validation (uniqueness, authorization)

## Handler-Level Validation

### Inline Validation

For simple checks, validate directly in the handler:

```go
func (h *AuthHandler) Register(c *fiber.Ctx) error {
    var req models.RegisterRequest
    if err := c.BodyParser(&req); err != nil {
        return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
    }

    // Basic field validation
    if req.Name == "" || req.Email == "" || req.Password == "" {
        h.store.Flash(c, "error", "All fields are required")
        return c.Redirect("/register")
    }

    // Delegate to service
    user, err := h.authService.Register(req.Name, req.Email, req.Password)
    // ...
}
```

### Flash Errors for Inertia Forms

For Inertia-based forms, use flash messages:

```go
// Handler sets flash on validation failure
if req.Email == "" || req.Password == "" {
    h.store.Flash(c, "error", "Email and password are required")
    return c.Redirect("/login")
}

// Handle service errors
if err == services.ErrInvalidCredentials {
    h.store.Flash(c, "error", "Invalid email or password")
    return c.Redirect("/login")
}
```

Flash messages are auto-injected into Inertia props as `props.flash.error`.

### Password Validation

```go
// In handler or service
if len(req.NewPassword) < 8 {
    return h.inertiaService.Render(c, "app/Profile", fiber.Map{
        "error": "Password must be at least 8 characters",
    })
}
if req.NewPassword != req.ConfirmPassword {
    return h.inertiaService.Render(c, "app/Profile", fiber.Map{
        "error": "Passwords do not match",
    })
}
```

## Service-Level Validation

### Business Rules

Services validate business rules and return domain-specific errors:

```go
func (s *AuthService) Register(name, email, password string) (*models.User, error) {
    // Business rule: email must be unique
    _, err := s.querier.GetUserByEmail(context.Background(), email)
    if err == nil {
        return nil, queries.ErrUserAlreadyExists
    }
    // ...
}
```

### Common Validation Patterns

| Pattern | Location | Example |
|---------|----------|---------|
| Required fields | Handler | Check empty strings, nil values |
| Format validation | Handler | Email format, password length |
| Uniqueness | Service | Check DB for duplicates |
| Authorization | Middleware | Role check, ownership check |
| Rate limiting | Middleware | Login attempt throttling |

## Next Steps

- [Handlers Guide](handlers.md) — Handler patterns
- [Forms Guide](forms.md) — Form handling and validation
