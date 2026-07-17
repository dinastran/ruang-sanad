# Handlers

This guide covers building HTTP handlers in Laju Go, including request parsing, validation, and response handling.

## Overview

Handlers (also called controllers) are functions that handle HTTP requests and return responses. In Laju Go, handlers are organized by feature in the `app/handlers/` directory.

## Handler Structure

Laju Go uses **struct-based handlers** for better organization and dependency injection:

```go
// app/handlers/auth.go
package handlers

import (
    "github.com/gofiber/fiber/v2"
    "laju-go/app/services"
    "laju-go/app/session"
)

type AuthHandler struct {
    authService    *services.AuthService
    store          *session.Store
    inertiaService *services.InertiaService
}

func NewAuthHandler(
    authService *services.AuthService,
    store *session.Store,
    inertiaService *services.InertiaService,
) *AuthHandler {
    return &AuthHandler{
        authService:    authService,
        store:          store,
        inertiaService: inertiaService,
    }
}

func (h *AuthHandler) ShowLoginForm(c *fiber.Ctx) error {
    return c.Render("auth/login", fiber.Map{
        "title": "Login",
    })
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
    // Handler logic
}
```

## Handler Files

| File | Struct | Purpose |
|------|--------|---------|
| `auth.go` | `AuthHandler` | Login, register, OAuth, logout, me, avatar |
| `app.go` | `AppHandler` | Dashboard, profile, update profile/password |
| `public.go` | `PublicHandler` | Landing page, about |
| `upload.go` | `UploadHandler` | File upload with validation |
| `password-reset.go` | `PasswordResetHandler` | Forgot/reset password flow |

## Request Handling

### Parsing Request Body

```go
type LoginRequest struct {
    Email    string `json:"email"`
    Password string `json:"password"`
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
    var req LoginRequest
    
    // Parse JSON or form data
    if err := c.BodyParser(&req); err != nil {
        return c.Status(400).JSON(fiber.Map{
            "error": "Invalid request body",
        })
    }
    
    // Process request
    user, err := h.authService.LoginByEmail(req.Email, req.Password)
    if err != nil {
        return c.Status(401).JSON(fiber.Map{
            "error": err.Error(),
        })
    }
    
    // Store session
    session, _ := h.store.Get(c)
    session.Set("user_id", user.ID)
    session.Save()
    
    return c.Redirect("/app")
}
```

### Parsing URL Parameters

```go
func (h *UserHandler) GetUser(c *fiber.Ctx) error {
    id := c.Params("id")
    
    user, err := h.userRepo.GetByID(id)
    if err != nil {
        return c.Status(404).JSON(fiber.Map{
            "error": "User not found",
        })
    }
    
    return c.JSON(user)
}
```

### Parsing Query Parameters

```go
func (h *UserHandler) GetUsers(c *fiber.Ctx) error {
    page := c.QueryInt("page", 1)           // Default: 1
    limit := c.QueryInt("limit", 10)        // Default: 10
    search := c.Query("search", "")         // Default: ""
    
    users, err := h.userRepo.GetUsers(page, limit, search)
    if err != nil {
        return c.Status(500).JSON(fiber.Map{
            "error": "Failed to fetch users",
        })
    }
    
    return c.JSON(users)
}
```

### Parsing Form Data

```go
func (h *AuthHandler) Register(c *fiber.Ctx) error {
    name := c.FormValue("name")
    email := c.FormValue("email")
    password := c.FormValue("password")
    
    // Process registration
    user, err := h.authService.Register(name, email, password)
    if err != nil {
        return c.Status(400).JSON(fiber.Map{
            "error": err.Error(),
        })
    }
    
    return c.JSON(user)
}
```

## Response Handling

### JSON Response

```go
func (h *UserHandler) GetUser(c *fiber.Ctx) error {
    return c.JSON(fiber.Map{
        "id": user.ID,
        "email": user.Email,
        "name": user.Name,
    })
}

// With status code
func (h *UserHandler) CreateUser(c *fiber.Ctx) error {
    // ... create user ...
    
    return c.Status(201).JSON(fiber.Map{
        "message": "User created successfully",
        "user": user,
    })
}
```

### Redirect Response

```go
func (h *AuthHandler) Login(c *fiber.Ctx) error {
    // ... login logic ...
    
    return c.Redirect("/app")
}

// Redirect with message
return c.Redirect("/login?error=invalid_credentials")
```

### Direct Template Rendering (for landing page)

```go
func (h *PublicHandler) Index(c *fiber.Ctx) error {
    assetData := h.assetService.GetAssetData()
    isDev := assetData["IsDev"].(bool)
    viteURL, _ := assetData["ViteURL"].(string)
    mainCSS, _ := assetData["MainCSS"].(string)

    c.Set("Content-Type", "text/html; charset=utf-8")
    return templates.LandingPage("Laju Go", isDev, viteURL, mainCSS).
        Render(c.Context(), c.Response().BodyWriter())
}
```

### Inertia Response (default for protected routes)

```go
func (h *AppHandler) Dashboard(c *fiber.Ctx) error {
    return h.inertiaService.Render(c, "app/Dashboard", fiber.Map{
        "user": user,
    })
}
```

> InertiaService auto-detects whether to return full HTML (initial load) or JSON (XHR navigation). Component name maps to `frontend/src/pages/{name}.svelte`.

### File Download

```go
func (h *FileHandler) Download(c *fiber.Ctx) error {
    return c.Download("/path/to/file.pdf", "report.pdf")
}
```

### File Send

```go
func (h *FileHandler) GetFile(c *fiber.Ctx) error {
    return c.SendFile("/path/to/file.pdf")
}
```

## Input Validation

### Manual Validation

```go
func (h *AuthHandler) Register(c *fiber.Ctx) error {
    var req dto.RegisterRequest
    if err := c.BodyParser(&req); err != nil {
        return c.Status(400).JSON(fiber.Map{
            "error": "Invalid request body",
        })
    }
    
    // Validate email
    if req.Email == "" {
        return c.Status(400).JSON(fiber.Map{
            "error": "Email is required",
        })
    }
    if !isValidEmail(req.Email) {
        return c.Status(400).JSON(fiber.Map{
            "error": "Invalid email format",
        })
    }
    
    // Validate password
    if len(req.Password) < 8 {
        return c.Status(400).JSON(fiber.Map{
            "error": "Password must be at least 8 characters",
        })
    }
    
    // Validate name
    if req.Name == "" {
        return c.Status(400).JSON(fiber.Map{
            "error": "Name is required",
        })
    }
    
    // Process registration
    user, err := h.authService.Register(req)
    if err != nil {
        return c.Status(400).JSON(fiber.Map{
            "error": err.Error(),
        })
    }
    
    return c.Status(201).JSON(user)
}

func isValidEmail(email string) bool {
    // Simple email validation
    return strings.Contains(email, "@") && strings.Contains(email, ".")
}
```

### Using Validator Library

```go
import "github.com/go-playground/validator/v10"

type RegisterRequest struct {
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required,min=8"`
    Name     string `json:"name" validate:"required,min=2,max=100"`
}

var validate = validator.New()

func (h *AuthHandler) Register(c *fiber.Ctx) error {
    var req RegisterRequest
    if err := c.BodyParser(&req); err != nil {
        return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
    }
    
    if err := validate.Struct(&req); err != nil {
        return c.Status(400).JSON(fiber.Map{
            "error": "Validation failed",
            "details": err.Error(),
        })
    }
    
    // Process registration
}
```

## Error Handling

### Basic Error Handling

```go
func (h *UserHandler) GetUser(c *fiber.Ctx) error {
    user, err := h.userService.GetByID(c.Params("id"))
    if err != nil {
        if err == services.ErrUserNotFound {
            return c.Status(404).JSON(fiber.Map{
                "error": "User not found",
            })
        }
        return c.Status(500).JSON(fiber.Map{
            "error": "An unexpected error occurred",
        })
    }
    return c.JSON(user)
}
```

### Custom Error Types

```go
// app/errors/errors.go
type AppError struct {
    Code    int
    Message string
    Err     error
}

func (e *AppError) Error() string {
    return e.Message
}

var (
    ErrUserNotFound = &AppError{
        Code:    404,
        Message: "User not found",
    }
    ErrUnauthorized = &AppError{
        Code:    401,
        Message: "Unauthorized",
    }
)

// Usage in handler
func (h *UserHandler) GetUser(c *fiber.Ctx) error {
    user, err := h.userService.GetByID(c.Params("id"))
    if err != nil {
        if appErr, ok := err.(*errors.AppError); ok {
            return c.Status(appErr.Code).JSON(fiber.Map{
                "error": appErr.Message,
            })
        }
        return c.Status(500).JSON(fiber.Map{
            "error": "Internal server error",
        })
    }
    return c.JSON(user)
}
```

## Session Handling

### Storing Session Data

```go
func (h *AuthHandler) Login(c *fiber.Ctx) error {
    // ... authenticate user ...
    
    session, _ := h.store.Get(c)
    session.Set("user_id", user.ID)
    session.Set("email", user.Email)
    session.Set("role", user.Role)
    
    if err := session.Save(); err != nil {
        return c.Status(500).JSON(fiber.Map{
            "error": "Failed to create session",
        })
    }
    
    return c.Redirect("/app")
}
```

### Destroying Session (Logout)

```go
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
    session, _ := h.store.Get(c)
    
    // Clear session data
    session.Destroy()
    
    return c.Redirect("/login")
}
```

## File Upload Handling

```go
// app/handlers/upload.go
func (h *UploadHandler) Upload(c *fiber.Ctx) error {
    // Parse multipart form (max 5MB)
    form, err := c.MultipartForm()
    if err != nil {
        return c.Status(400).JSON(fiber.Map{
            "error": "Failed to parse form",
        })
    }
    
    // Get uploaded file
    files := form.File["avatar"]
    if len(files) == 0 {
        return c.Status(400).JSON(fiber.Map{
            "error": "No file uploaded",
        })
    }
    
    file := files[0]
    
    // Validate file type
    allowedTypes := []string{"image/jpeg", "image/png", "image/gif"}
    if !contains(allowedTypes, file.Header.Get("Content-Type")) {
        return c.Status(400).JSON(fiber.Map{
            "error": "Invalid file type",
        })
    }
    
    // Validate file size (5MB)
    if file.Size > 5*1024*1024 {
        return c.Status(400).JSON(fiber.Map{
            "error": "File too large",
        })
    }
    
    // Generate unique filename
    filename := fmt.Sprintf("%d_%s", time.Now().Unix(), file.Filename)
    filepath := fmt.Sprintf("storage/avatars/%s", filename)
    
    // Save file
    if err := c.SaveFile(file, filepath); err != nil {
        return c.Status(500).JSON(fiber.Map{
            "error": "Failed to save file",
        })
    }
    
    return c.JSON(fiber.Map{
        "message": "File uploaded successfully",
        "path": filepath,
    })
}
```

## CSRF Protection

### Getting CSRF Token

```go
// For Inertia.js requests
func (h *AppHandler) Dashboard(c *fiber.Ctx) error {
    csrf := c.Locals("csrf").(string)
    
    return h.inertiaService.Render(c, "Dashboard", fiber.Map{
        "csrf": csrf,
        "user": c.Locals("user"),
    })
}
```

### Validating CSRF Token

CSRF validation is handled by middleware:

```go
// app/middlewares/csrf.go
func CSRF(store *session.Store) fiber.Handler {
    return func(c *fiber.Ctx) error {
        // Skip for GET, HEAD, OPTIONS
        if c.Method() == "GET" || c.Method() == "HEAD" || c.Method() == "OPTIONS" {
            return c.Next()
        }
        
        // Get token from header or form
        token := c.Get("X-CSRF-Token")
        if token == "" {
            token = c.FormValue("csrf_token")
        }
        
        // Validate token
        session, _ := store.Get(c)
        storedToken := session.Get("csrf_token")
        
        if token != storedToken {
            return c.Status(403).JSON(fiber.Map{
                "error": "Invalid CSRF token",
            })
        }
        
        return c.Next()
    }
}
```

## Rate Limiting

Rate limiting is handled by middleware:

```go
// Apply rate limit to login route
app.Post("/login", 
    middlewares.RateLimit(5, 15*time.Minute),
    handlers.AuthHandler.Login,
)
```

## Handler Testing

### Unit Testing Handlers

```go
// handlers/auth_test.go
func TestAuthHandler_Login(t *testing.T) {
    // Setup
    authService := &MockAuthService{}
    store := session.NewStore()
    handler := NewAuthHandler(authService, nil, store)
    
    app := fiber.New()
    app.Post("/login", handler.Login)
    
    // Test valid login
    body := `{"email":"test@example.com","password":"password123"}`
    req := httptest.NewRequest("POST", "/login", strings.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := app.Test(req)
    if err != nil {
        t.Fatal(err)
    }
    
    if resp.StatusCode != 302 {
        t.Errorf("Expected redirect, got %d", resp.StatusCode)
    }
}
```

## Best Practices

### 1. Keep Handlers Thin

Delegate business logic to services:

```go
// ❌ Bad: Business logic in handler
func (h *Handler) Login(c *fiber.Ctx) error {
    // Don't do this in handler!
    user, err := db.Query("SELECT * FROM users WHERE email = ?", email)
    if err != nil {
        return err
    }
    
    err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
    // ...
}

// ✅ Good: Delegate to service
func (h *Handler) Login(c *fiber.Ctx) error {
    user, err := h.authService.Login(email, password)
    if err != nil {
        return c.Status(401).JSON(fiber.Map{"error": err.Error()})
    }
    // ...
}
```

### 2. Use DTOs for Requests

```go
// ✅ Good: Dedicated request type
type CreateUserRequest struct {
    Email    string `json:"email"`
    Name     string `json:"name"`
    Password string `json:"password"`
}

func (h *Handler) CreateUser(c *fiber.Ctx) error {
    var req CreateUserRequest
    if err := c.BodyParser(&req); err != nil {
        return err
    }
    // ...
}
```

### 3. Return Consistent Error Format

```go
// ✅ Good: Consistent error format
return c.Status(400).JSON(fiber.Map{
    "error": "Error message here",
})

// For validation errors
return c.Status(400).JSON(fiber.Map{
    "error": "Validation failed",
    "details": fiber.Map{
        "email": "Email is required",
        "password": "Password must be 8+ characters",
    },
})
```

### 4. Use Appropriate Status Codes

| Status | When to Use |
|--------|-------------|
| 200 | Successful GET, PUT, PATCH |
| 201 | Successful resource creation (POST) |
| 302 | Redirect after successful action |
| 400 | Invalid request, validation errors |
| 401 | Not authenticated |
| 403 | Insufficient permissions |
| 404 | Resource not found |
| 422 | Validation error (Inertia) |
| 500 | Server error |

### 5. Sanitize Input

```go
func (h *Handler) CreateUser(c *fiber.Ctx) error {
    var req CreateUserRequest
    if err := c.BodyParser(&req); err != nil {
        return err
    }
    
    // Sanitize input
    req.Email = strings.TrimSpace(strings.ToLower(req.Email))
    req.Name = strings.TrimSpace(req.Name)
    
    // ...
}
```

## Next Steps

- [Database Guide](database.md) - Database operations with repositories
- [Services Guide](../reference/services.md) - Business logic layer
- [Authentication Guide](authentication.md) - Complete auth flow
