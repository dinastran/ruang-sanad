# Routing

This guide covers route definitions, middleware setup, and request handling in Laju Go.

## Overview

Routes in Laju Go are defined in `routes/web.go` using the Fiber framework's routing API. Routes map HTTP methods and paths to handler functions.

## Route File Structure

```go
// routes/web.go — routes dipisah ke sub-functions:
// setupStaticRoutes, setupPublicRoutes, setupAuthRoutes, setupAppRoutes

func SetupRoutes(app *fiber.App, handlers Handlers, store *session.Store, userService *services.UserService, mailerService *services.MailerService, csrfMiddleware *middlewares.CSRFMiddleware) {
    setupStaticRoutes(app)
    setupPublicRoutes(app, handlers.Public)
    setupAuthRoutes(app, handlers.Auth, handlers.PasswordReset, store, mailerService)
    setupAppRoutes(app, handlers.App, handlers.Upload, store, userService, csrfMiddleware)
}
```

## HTTP Methods

Fiber supports all standard HTTP methods:

```go
// GET - Retrieve data
app.Get("/users", handlers.GetUsers)

// POST - Create data
app.Post("/users", handlers.CreateUser)

// PUT - Update data (full replacement)
app.Put("/users/:id", handlers.UpdateUser)

// PATCH - Update data (partial)
app.Patch("/users/:id", handlers.PartialUpdateUser)

// DELETE - Delete data
app.Delete("/users/:id", handlers.DeleteUser)
```

## Route Parameters

### Path Parameters

```go
// Define route with parameter
app.Get("/users/:id", handlers.GetUser)

// Access parameter in handler
func GetUser(c *fiber.Ctx) error {
    id := c.Params("id")
    // Use id to fetch user
    return c.JSON(fiber.Map{"id": id})
}
```

### Multiple Parameters

```go
app.Get("/users/:userId/posts/:postId", handlers.GetPost)

func GetPost(c *fiber.Ctx) error {
    userID := c.Params("userId")
    postID := c.Params("postId")
    // ...
}
```

### Optional Parameters

```go
// Optional parameter with default
app.Get("/users/:id?", handlers.GetUser)

func GetUser(c *fiber.Ctx) error {
    id := c.Params("id", "0") // Default to "0" if not provided
    // ...
}
```

### Wildcard Parameters

```go
// Catch-all route
app.Get("/files/*", handlers.GetFile)

func GetFile(c *fiber.Ctx) error {
    path := c.Params("*")
    // ...
}
```

## Query Parameters

```go
// GET /search?q=golang&limit=10
app.Get("/search", handlers.Search)

func Search(c *fiber.Ctx) error {
    query := c.Query("q")
    limit := c.Query("limit", "10") // Default value
    
    // Convert to int
    limitInt, _ := strconv.Atoi(limit)
    
    return c.JSON(fiber.Map{
        "query": query,
        "limit": limitInt,
    })
}
```

## Request Body

### Parse JSON Body

```go
type CreateUserRequest struct {
    Email    string `json:"email"`
    Name     string `json:"name"`
    Password string `json:"password"`
}

app.Post("/users", handlers.CreateUser)

func CreateUser(c *fiber.Ctx) error {
    var req CreateUserRequest
    if err := c.BodyParser(&req); err != nil {
        return c.Status(400).JSON(fiber.Map{
            "error": "Invalid request body",
        })
    }
    
    // Validate and process
    return c.Status(201).JSON(fiber.Map{
        "message": "User created",
        "user": req,
    })
}
```

### Parse Form Data

```go
app.Post("/login", handlers.Login)

func Login(c *fiber.Ctx) error {
    email := c.FormValue("email")
    password := c.FormValue("password")
    
    // ...
}
```

## Route Groups

Group related routes with common prefixes and middleware:

```go
// API routes with /api prefix
api := app.Group("/api")
api.Get("/users", handlers.GetUsers)
api.Post("/users", handlers.CreateUser)

// Admin routes with /admin prefix and auth middleware
admin := app.Group("/admin", middlewares.AuthRequired(store))
admin.Get("/", handlers.AdminDashboard)
admin.Get("/users", handlers.AdminGetUsers)
admin.Delete("/users/:id", handlers.AdminDeleteUser)

// App routes with CSRF protection
protected := app.Group("/", middlewares.AuthRequired(store), middlewares.CSRF(store))
protected.Get("/app", handlers.Dashboard)
protected.Get("/app/profile", handlers.Profile)
```

## Middleware

### Applying Middleware to Routes

```go
// Single middleware
app.Get("/app", middlewares.AuthRequired(store), handlers.Dashboard)

// Multiple middleware
app.Post("/users", 
    middlewares.AuthRequired(store),
    middlewares.CSRF(store),
    middlewares.RateLimit(5, 15*time.Minute),
    handlers.CreateUser,
)
```

### Global Middleware

Apply middleware to all routes:

```go
// Logger middleware (all routes)
app.Use(logger.New())

// Recovery middleware (all routes)
app.Use(recover.New())

// CORS middleware
app.Use(cors.New(cors.Config{
    AllowOrigins: "http://localhost:5173",
    AllowMethods: "GET,POST,PUT,DELETE",
}))
```

### Skip Middleware for Specific Routes

```go
app.Use("/api", func(c *fiber.Ctx) error {
    // Skip for API routes
    return c.Next()
})
```

## Route Matching Order

Routes are matched in the order they are defined:

```go
// ❌ Bad: This route will never be reached
app.Get("/users/new", handlers.NewUser)      // Defined second
app.Get("/users/:id", handlers.GetUser)       // Defined first (matches "new")

// ✅ Good: Specific routes before parameterized routes
app.Get("/users/new", handlers.NewUser)       // Defined first
app.Get("/users/:id", handlers.GetUser)       // Defined second
```

## Named Routes

Fiber doesn't support named routes out of the box, but you can create constants:

```go
// routes/constants.go
const (
    RouteLogin = "/login"
    RouteRegister = "/register"
    RouteDashboard = "/app"
    RouteProfile = "/app/profile"
)

// Usage in handlers
return c.Redirect(routes.RouteDashboard)
```

## Route Handlers

### Anonymous Handlers

```go
app.Get("/health", func(c *fiber.Ctx) error {
    return c.JSON(fiber.Map{
        "status": "ok",
    })
})
```

### Struct-Based Handlers

```go
// app/handlers/auth.go
type AuthHandler struct {
    authService *services.AuthService
}

func NewAuthHandler(authService *services.AuthService) *AuthHandler {
    return &AuthHandler{authService: authService}
}

func (h *AuthHandler) ShowLoginForm(c *fiber.Ctx) error {
    return c.Render("login", fiber.Map{})
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
    // ...
}

// routes/web.go
authHandler := handlers.NewAuthHandler(authService)
app.Get("/login", authHandler.ShowLoginForm)
app.Post("/login", authHandler.Login)
```

## Actual Route Layout

Routes dipisah ke sub-functions:

```go
func setupStaticRoutes(app *fiber.App) {
    app.Static("/dist", "./dist", fiber.Static{CacheDuration: 365 * 24 * time.Hour, MaxAge: 31536000, Compress: true})
    app.Static("/assets", "./dist/assets", fiber.Static{CacheDuration: 365 * 24 * time.Hour, MaxAge: 31536000, Compress: true})
    app.Static("/public", "./public", fiber.Static{CacheDuration: 1 * time.Hour, MaxAge: 3600})
    app.Static("/storage", "./storage", fiber.Static{CacheDuration: 24 * time.Hour, MaxAge: 86400})
}

func setupAuthRoutes(app *fiber.App, authHandler *handlers.AuthHandler, passwordResetHandler *handlers.PasswordResetHandler, store *session.Store, mailerService *services.MailerService) {
    app.Get("/login", middlewares.Guest(store), authHandler.ShowLoginForm)
    app.Post("/login", middlewares.Guest(store), authHandler.Login, middlewares.AuthRateLimit.Limit())
    app.Get("/register", middlewares.Guest(store), authHandler.ShowRegisterForm)
    app.Post("/register", middlewares.Guest(store), authHandler.Register, middlewares.AuthRateLimit.Limit())
    app.Get("/auth/google", authHandler.GoogleLogin)
    app.Get("/auth/google/callback", authHandler.GoogleCallback)
    app.Post("/logout", middlewares.AuthRequired(store), authHandler.Logout)
    app.Get("/forgot-password", passwordResetHandler.ShowForgotPasswordForm)
    app.Post("/forgot-password", passwordResetHandler.SendResetLink, middlewares.PasswordResetRateLimit.Limit())
    app.Get("/reset-password/:token", passwordResetHandler.ShowResetPasswordForm)
    app.Post("/reset-password/:token", passwordResetHandler.ResetPassword)
}

func setupAppRoutes(app *fiber.App, appHandler *handlers.AppHandler, uploadHandler *handlers.UploadHandler, store *session.Store, userService *services.UserService, csrfMiddleware *middlewares.CSRFMiddleware) {
    protected := app.Group("/app", middlewares.AuthRequired(store))
    protected.Use(csrfMiddleware.Protect())
    protected.Get("/", appHandler.Dashboard)
    protected.Get("/profile", appHandler.Profile)
    protected.Put("/profile", appHandler.UpdateProfile)
    protected.Put("/profile/password", appHandler.UpdatePassword)
    protected.Post("/upload", uploadHandler.Upload)

    admin := app.Group("/admin", middlewares.AdminRequired(store, userService))
    admin.Get("/", func(c *fiber.Ctx) error { return c.JSON(fiber.Map{"message": "Admin dashboard"}) })
}
```

## Route Testing

### Test Routes with Fiber Test Client

```go
// routes/web_test.go
func TestRoutes(t *testing.T) {
    app := fiber.New()
    SetupRoutes(app, nil)

    t.Run("GET / returns 200", func(t *testing.T) {
        resp, err := app.Test(httptest.NewRequest("GET", "/", nil))
        if err != nil {
            t.Fatal(err)
        }
        if resp.StatusCode != 200 {
            t.Errorf("Expected status 200, got %d", resp.StatusCode)
        }
    })

    t.Run("GET /login returns 200", func(t *testing.T) {
        resp, err := app.Test(httptest.NewRequest("GET", "/login", nil))
        if err != nil {
            t.Fatal(err)
        }
        if resp.StatusCode != 200 {
            t.Errorf("Expected status 200, got %d", resp.StatusCode)
        }
    })
}
```

## Common Patterns

### Redirect to Login if Not Authenticated

```go
app.Get("/dashboard", func(c *fiber.Ctx) error {
    session, _ := store.Get(c)
    if session.Get("user_id") == nil {
        return c.Redirect("/login")
    }
    return c.Render("dashboard", fiber.Map{})
})
```

### API Versioning

```go
// v1 routes
v1 := app.Group("/api/v1")
v1.Get("/users", handlers.GetUsers)

// v2 routes
v2 := app.Group("/api/v2")
v2.Get("/users", handlers.GetUsersV2)
```

### Subdomain Routing

```go
// API subdomain
api := app.Group("api.")
api.Get("/users", handlers.GetUsers)

// Admin subdomain
admin := app.Group("admin.")
admin.Get("/", handlers.AdminDashboard)
```

## Best Practices

### 1. Keep Routes File Organized

Group related routes together with comments:

```go
// ===== PUBLIC ROUTES =====
// ===== AUTH ROUTES =====
// ===== PROTECTED ROUTES =====
// ===== ADMIN ROUTES =====
```

### 2. Use Middleware Chains

Apply common middleware to route groups:

```go
// ✅ Good: DRY with route groups
protected := app.Group("/", middlewares.AuthRequired(store), middlewares.CSRF(store))
protected.Get("/app", handlers.Dashboard)
protected.Get("/profile", handlers.Profile)

// ❌ Bad: Repetitive middleware
app.Get("/app", middlewares.AuthRequired(store), middlewares.CSRF(store), handlers.Dashboard)
app.Get("/profile", middlewares.AuthRequired(store), middlewares.CSRF(store), handlers.Profile)
```

### 3. Validate Input in Handlers

Don't rely on route-level validation:

```go
// ✅ Good: Validate in handler
app.Post("/users/:id", handlers.UpdateUser)

func UpdateUser(c *fiber.Ctx) error {
    id := c.Params("id")
    if id == "" {
        return c.Status(400).JSON(fiber.Map{"error": "ID is required"})
    }
    // ...
}
```

### 4. Use Consistent Naming

```go
// ✅ Good: Consistent naming
GET    /users           - List users
POST   /users           - Create user
GET    /users/:id       - Get user
PUT    /users/:id       - Update user
DELETE /users/:id       - Delete user

// ❌ Bad: Inconsistent naming
GET    /users           - List users
POST   /create-user     - Create user
GET    /user/:id        - Get user
PUT    /edit-user/:id   - Update user
```

### 5. Document Routes

Keep an API reference or use comments:

```go
// GetUsers godoc
// @Summary      List all users
// @Description  Get a list of all users
// @Tags         users
// @Accept       json
// @Produce      json
// @Success      200  {array}  models.User
// @Router       /users [get]
app.Get("/users", handlers.GetUsers)
```

## Troubleshooting

### Route Not Matching

**Problem**: Route returns 404

**Solutions**:

1. Check route order (specific before parameterized)
2. Verify HTTP method (GET vs POST)
3. Check for typos in path
4. Ensure middleware isn't redirecting

### Middleware Not Running

**Problem**: Middleware skipped

**Solutions**:

1. Verify middleware is applied to route
2. Check if `c.Next()` is called
3. Ensure middleware order is correct

### Parameters Not Accessible

**Problem**: `c.Params("id")` returns empty

**Solutions**:

1. Check parameter name matches route definition
2. Verify route is matched (check order)
3. Use `c.AllParams()` to debug

## Next Steps

- [Handlers Guide](handlers.md) - Building HTTP handlers
- [Middleware Guide](middleware.md) - Creating custom middleware
- [Architecture Guide](architecture.md) - Understanding the layered architecture
