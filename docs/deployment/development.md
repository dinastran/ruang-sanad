# Development Workflow

This guide covers the development workflow, hot reload setup, and available scripts for Laju Go.

## Overview

Laju Go provides a smooth development experience with:

- **Vite HMR** - Instant frontend updates
- **Air Hot Reload** - Automatic Go server restart
- **Concurrent Scripts** - Run both servers with one command

## Development Servers

### Option 1: Run Everything Together (Recommended)

Start both Vite and Go servers with hot reload:

```bash
npm run dev:all
```

This runs:
- Vite dev server (frontend HMR)
- Air (Go hot reload)

Both servers run concurrently in a single terminal.

### Option 2: Run Servers Separately

**Terminal 1** - Vite dev server:

```bash
npm run dev
```

**Terminal 2** - Go server with Air:

```bash
air
# Or via npm
npm run dev:go
```

### Option 3: Manual Run

**Terminal 1** - Vite:

```bash
npm run dev
```

**Terminal 2** - Go (manual restart required):

```bash
go run ./cmd/laju-go
```

## Available Scripts

```json
{
  "scripts": {
    "dev": "vite",
    "dev:go": "air",
    "dev:all": "concurrently \"npm run dev\" \"npm run dev:go\"",
    "build": "npm run build:frontend && npm run build:go",
    "build:frontend": "vite build",
    "build:go": "go build -o laju-go ./cmd/laju-go",
    "serve": "./laju-go",
  }
}
```

### Script Descriptions

| Script | Description |
|--------|-------------|
| `npm run dev` | Start Vite dev server (frontend HMR) |
| `npm run dev:go` | Start Go server with Air hot reload |
| `npm run dev:all` | Run both Vite and Air concurrently |
| `npm run build` | Build frontend and Go binary |
| `npm run build:frontend` | Build frontend assets only |
| `npm run build:go` | Build Go binary only |
| `npm run serve` | Run production binary |

## Hot Module Replacement (HMR)

### Frontend HMR (Vite)

Vite provides instant updates for frontend changes:

| File Type | Update Behavior |
|-----------|-----------------|
| `.svelte` | Component updates instantly |
| `.css` | Styles update instantly |
| `.ts/.js` | Module reloads instantly |
| `.json` | Data updates instantly |

**Example**:

1. Edit `frontend/src/pages/app/Dashboard.svelte`
2. Add `<h1>Test</h1>` to template
3. Save file
4. Browser updates instantly (no refresh)

### Backend Hot Reload (Air)

Air automatically rebuilds and restarts the Go server:

| File Type | Update Behavior |
|-----------|-----------------|
| `.go` | Server rebuilds and restarts (~1-2 sec) |
| `.env` | Server restarts to load new env |
| `migrations/` | Server restarts (migrations re-run) |

**Example**:

1. Edit `app/handlers/auth.go`
2. Add `fmt.Println("Login called")`
3. Save file
4. Terminal shows Air rebuilding
5. Server restarts automatically

## Air Configuration

Air is configured via `.air.toml`:

```toml
# .air.toml
root = "."
testdata_dir = "testdata"
tmp_dir = "tmp"

[build]
  bin = "./tmp/main"
  cmd = "go build -o ./tmp/main ./cmd/laju-go"
  delay = 1000
  exclude_dir = ["assets", "tmp", "vendor", "node_modules"]
  exclude_file = []
  exclude_regex = ["_test.go"]
  exclude_unchanged = false
  follow_symlink = false
  full_bin = ""
  include_dir = []
  include_ext = ["go", "tpl", "tmpl", "html"]
  log = "air.log"
  poll = false
  poll_interval = 0
  rerun = false
  rerun_delay = 500
  send_interrupt = false
  stop_on_error = true

[color]
  app = ""
  build = "yellow"
  main = "magenta"
  runner = "green"
  watcher = "cyan"

[log]
  time = false

[misc]
  clean_on_exit = false

[proxy]
  app_port = 8080
  enabled = false
  proxy_port = 0

[screen]
  clear_on_rebuild = false
```

### Custom Air Configuration

To customize Air behavior:

```toml
# Include additional file types
include_ext = ["go", "html", "tmpl"]

# Exclude specific directories
exclude_dir = ["storage", "data", "tmp"]

# Stop on first error
stop_on_error = true

# Clear screen on rebuild
clear_on_rebuild = true
```

## Vite Configuration

Vite is configured in `vite.config.js`:

```javascript
import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'
import { ViteInertiaPlugin } from 'vite-plugin-inertia'

export default defineConfig({
  plugins: [
    svelte(),
    ViteInertiaPlugin(),
  ],
  server: {
    port: 5173,
  },
  build: {
    outDir: 'dist',
    manifest: true,
  },
})
```

### Vite Port Detection

Laju Go uses a custom plugin to detect Vite's port:

1. Vite writes port to `.vite-port` on startup
2. Go server reads `.vite-port` to proxy requests
3. Cleanup on Vite exit

This allows multiple instances without port conflicts.

## Development Workflow

### Typical Development Session

```bash
# 1. Start development servers
npm run dev:all

# 2. Make changes to frontend
# Edit frontend/src/pages/app/Dashboard.svelte
# Changes appear instantly in browser

# 3. Make changes to backend
# Edit app/handlers/auth.go
# Server rebuilds automatically

# 4. Test changes in browser
# Visit http://localhost:8080

# 5. Run tests (in another terminal)
```

### Debugging

#### Frontend Debugging

Use browser DevTools:

1. Open DevTools (F12)
2. Check Console for errors
3. Use Sources tab to debug Svelte components
4. Network tab for API requests

#### Backend Debugging

**Option 1: Logging**

```go
import "log"

func (h *AuthHandler) Login(c *fiber.Ctx) error {
    log.Printf("Login attempt for email: %s", email)
    // ...
}
```

**Option 2: Delve Debugger**

```bash
# Install Delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug with Delve
dlv debug .
```

**Option 3: VS Code Debugger**

Create `.vscode/launch.json`:

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Debug Laju Go",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}",
      "env": {},
      "args": []
    }
  ]
}
```

## Environment Variables

Use `.env` for development configuration:

```bash
# .env
APP_ENV=development
APP_PORT=8080
DB_PATH=data/app.db
SESSION_SECRET=dev-secret-key-not-for-production

# Enable verbose logging (if implemented)
# LOG_LEVEL=debug
```

## Database in Development

### Auto Migrations

Migrations run automatically on server start. No manual intervention needed.

### Reset Database

```bash
# Delete database file
rm data/app.db

# Restart server (migrations run automatically)
go run ./cmd/laju-go
```

### Seed Data

Create a seed script:

```go
// cmd/seed/main.go
package main

func main() {
    db := initDatabase()
    
    // Create test users
    db.Exec("INSERT INTO users (email, name, password, role) VALUES (?, ?, ?, ?)",
        "test@example.com", "Test User", hashedPassword, "user")
    
    db.Exec("INSERT INTO users (email, name, password, role) VALUES (?, ?, ?, ?)",
        "admin@example.com", "Admin User", hashedPassword, "admin")
    
    fmt.Println("Database seeded!")
}
```

Run seeder:

```bash
go run ./cmd/laju-go
```

## Frontend Development

### Component Development

Create components in `frontend/src/components/`:

```svelte
<!-- frontend/src/components/Card.svelte -->
<script>
  export let title = '';
  export let subtitle = '';
</script>

<div class="card">
  <h2>{title}</h2>
  {#if subtitle}
    <p>{subtitle}</p>
  {/if}
  <slot />
</div>

<style>
  .card {
    border: 1px solid #ddd;
    border-radius: 8px;
    padding: 16px;
  }
</style>
```

### Page Development

Create pages in `frontend/src/pages/`:

```svelte
<!-- frontend/src/pages/app/Settings.svelte -->
<script>
  import { page } from '@inertiajs/svelte';
  
  const user = $page.props.user;
</script>

<h1>Settings</h1>
<p>Welcome, {user.name}</p>
```

### Using Inertia

```svelte
<script>
  import { router } from '@inertiajs/svelte';
  
  function handleSubmit() {
    router.post('/settings', {
      name: formData.name,
      email: formData.email,
    }, {
      onSuccess: () => {
        console.log('Settings updated!');
      },
      onError: (errors) => {
        console.log('Validation errors:', errors);
      },
    });
  }
</script>
```

## Testing

### Frontend Tests

```bash
# Run tests (headless)

# Run tests with UI

# Run specific test file
```

### Backend Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test ./app/handlers/...

# Run with verbose output
go test -v ./...
```

## Common Issues

### Port Already in Use

**Error**: `listen tcp :8080: bind: address already in use`

**Solution**:

```bash
# Kill process on port 8080
lsof -ti:8080 | xargs kill -9

# Or change port in .env
APP_PORT=8081
```

### Vite Port Detection Fails

**Error**: Go server can't connect to Vite

**Solution**:

```bash
# Remove port cache
rm .vite-port

# Restart Vite
npm run dev
```

### Air Not Rebuilding

**Problem**: Changes to `.go` files don't trigger rebuild

**Solution**:

1. Check `.air.toml` includes correct extensions
2. Ensure file is not in excluded directories
3. Restart Air

### HMR Not Working

**Problem**: Frontend changes don't appear

**Solution**:

1. Check Vite dev server is running
2. Clear browser cache
3. Check browser console for errors
4. Restart Vite server

## Best Practices

### 1. Use `.env` for Configuration

```bash
# ✅ Good: Environment-specific config
cp .env.example .env
nano .env

# ❌ Bad: Hardcoded values
const port = 8080;
```

### 2. Commit `.env.example`, Not `.env`

```bash
# ✅ Good
git add .env.example

# ❌ Bad
git add .env
```

### 3. Use Hot Reload

```bash
# ✅ Good: Auto-reload with Air
air

# ❌ Bad: Manual restart
go run ./cmd/laju-go  # Restart after every change
```

### 4. Run Tests Frequently

```bash
# Run tests after significant changes
go test ./...
```

### 5. Use Concurrent Script

```bash
# ✅ Good: One command for both servers
npm run dev:all

# ❌ Bad: Multiple terminals
# Terminal 1: npm run dev
# Terminal 2: air
```

## Next Steps

- [Production Deployment](production.md) - Deploy to production servers
- [Docker Deployment](docker.md) - Containerized deployment
- [Testing Guide](../guide/testing.md) - Comprehensive testing strategies
