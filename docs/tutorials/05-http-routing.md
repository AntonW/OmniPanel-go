# Chapter 5: HTTP Routing & REST API

## What This Package Does

The `routes` package sets up the HTTP server using the [Fiber](https://gofiber.io/) framework. It defines all URL endpoints, serves static files, and connects HTTP requests to the `AppState`.

## Fiber Framework Basics

Fiber is a web framework inspired by Express.js (Node.js). If you've seen Express, Fiber will feel familiar:

```go
app := fiber.New()
app.Get("/path", handler)
app.Post("/path", handler)
app.Use(middleware)
app.Listen(":3000")
```

## Building the Router

```go
// internal/routes/router.go
func NewRouter(s *state.AppState) *fiber.App {
    app := fiber.New()

    app.Use(func(c *fiber.Ctx) error {
        c.Locals("state", s)
        return c.Next()
    })
```

> **Concept: Middleware**
> Middleware is a function that runs before every request handler. `app.Use(...)` registers it globally. Here, we store `AppState` in the request context via `c.Locals()`, so every handler can access it.

> **Concept: `c.Locals`**
> `Locals` is a per-request key-value store. You put data in during middleware and retrieve it in handlers. This avoids passing `AppState` as a parameter to every handler.

## Route Registration

```go
    app.Get("/", serveStartPage)     // Start page (panel list, host controls, log)
    app.Get("/panel", servePanel)    // Panel client
    app.Get("/editor", serveEditorUI) // Panel editor
    app.Get("/login", serveLoginPage) // Login form (exempt from auth middleware)

    // CSS files are served without authentication so the login page
    // and unauthenticated pages can load styles.
    app.Static("/", staticDir, fiber.Static{
        Next: func(c *fiber.Ctx) bool {
            return !strings.HasSuffix(c.Path(), ".css")
        },
    })

    app.Use(auth.Middleware(cfg.AuthToken))

    // Static files (JS, CSS, assets served directly from static/ directory)
    // Disable browser caching for JS/CSS so changes are always picked up
    noCacheStaticMiddleware := func(c *fiber.Ctx) error {
        if strings.HasSuffix(c.Path(), ".js") || strings.HasSuffix(c.Path(), ".css") {
            c.Set("Cache-Control", "no-cache, no-store, must-revalidate")
        }
        return c.Next()
    }
    app.Use(noCacheStaticMiddleware)
    app.Static("/", s.StaticDir)

    app.Get("/ws", ws.New(func(c *ws.Conn) {
        handleWS(c, s)
    }))
    app.Get("/api/blocks", getBlocks)
    app.Get("/api/panels", getPanels)
    app.Post("/api/panel/save", savePanel)
    app.Get("/api/panel/load", loadPanel)
    app.Get("/api/panel/content", getPanelContent)
    app.Delete("/api/panel/delete", deletePanel)
    app.Post("/api/joystick-count", setJoystickCount)
    app.Get("/api/config", getConfig)
    app.Post("/api/data/push", pushData)
```

Routes are organized into three groups:
- **UI pages**: `/`, `/panel`, `/editor` (friendly URLs for HTML pages)
- **Static files**: `app.Static("/", s.StaticDir)` serves all JS, CSS, and block assets directly from the `static/` directory
- **WebSocket**: `/ws` (wrapped with `ws.New()` for WebSocket upgrade)
- **REST API**: `/api/*` endpoints for data operations

> **Concept: `app.Static`**
> Fiber's `app.Static(prefix, root)` maps a URL prefix to a filesystem directory. A request to `/client/client.js` serves `static/client/client.js`. Content-Type is set automatically based on file extension. This eliminates the need for individual handler functions for each JS/CSS file.

> **Concept: Cache-busting middleware**
> Browsers aggressively cache `.js` and `.css` files. During development this means code changes won't appear until you do a hard refresh (Ctrl+Shift+R). The `noCacheStaticMiddleware` runs before `app.Static` and checks if the request path ends with `.js` or `.css`. If so, it sets `Cache-Control: no-cache, no-store, must-revalidate` — telling the browser to always fetch a fresh copy. Images, fonts, and other assets are unaffected and cache normally.

> **Key Pattern: Targeted middleware**
> Instead of disabling caching for all static files (which would hurt performance for images/fonts), the middleware uses `strings.HasSuffix` to target only the file types that change frequently. This is a common pattern: inspect the request path and apply headers conditionally.

> **Key Pattern: Selective static serving**
> Fiber's `fiber.Static` struct accepts a `Next` function that determines whether to skip the static handler. By returning `true` for non-CSS paths, CSS files are served immediately without hitting auth middleware, while all other static files fall through to the authenticated `app.Static("/", s.StaticDir)` below.
>
> ```go
> app.Static("/", staticDir, fiber.Static{
>     Next: func(c *fiber.Ctx) bool {
>         return !strings.HasSuffix(c.Path(), ".css") // skip non-CSS
>     },
> })
> ```

## Serving Static Files

```go
    blocksPath := filepath.Join(s.UserPath, "blocks")
    if _, err := os.Stat(blocksPath); err == nil {
        app.Static("/blocks", blocksPath)
    }

    assetsPath := filepath.Join(s.UserPath, "assets")
    if _, err := os.Stat(assetsPath); err == nil {
        app.Static("/assets", assetsPath)
    }
```

`app.Static` maps a URL prefix to a filesystem directory. A request to `/blocks/button.html` serves `user/blocks/button.html`. The `os.Stat` check prevents errors if these directories don't exist yet.

## The Generic File Server

```go
func serveFile(c *fiber.Ctx, path string, contentType string) error {
    data, err := os.ReadFile(path)
    if err != nil {
        slog.Error("Failed to serve file", "path", path, "error", err)
        return c.Status(fiber.StatusNotFound).SendString("Not found")
    }
    c.Set("Content-Type", contentType)
    return c.Send(data)
}
```

All UI handlers use this helper. It reads the file from disk, sets the correct `Content-Type` header, and sends the bytes. If the file doesn't exist, it returns a 404.

> **Key Pattern: Centralized file serving**
> Instead of scattering `os.ReadFile` across handlers, a single helper ensures consistent error handling and content-type setting.

## UI Handlers (panel.go, editor.go)

These files contain handlers only for HTML pages. JS and CSS assets are served directly by `app.Static`:

```go
// internal/routes/panel.go
func serveStartPage(c *fiber.Ctx) error {
    s := c.Locals("state").(*state.AppState)
    path := filepath.Join(s.StaticDir, "index.html")
    return serveFile(c, path, "text/html; charset=utf-8")
}

func servePanel(c *fiber.Ctx) error {
    s := c.Locals("state").(*state.AppState)
    path := filepath.Join(s.StaticDir, "client", "index.html")
    return serveFile(c, path, "text/html; charset=utf-8")
}

// internal/routes/editor.go
func serveEditorUI(c *fiber.Ctx) error {
    s := c.Locals("state").(*state.AppState)
    path := filepath.Join(s.StaticDir, "editor", "editor.html")
    return serveFile(c, path, "text/html; charset=utf-8")
}
```

HTML references use paths that match the `static/` directory structure:
- `/client/index.html` → `<script src="/client/client.js">`
- `/index.html` → inline JavaScript for host controls, panel loading, and WebSocket logging
- `/editor/editor.html` → `<link href="/editor/editor-style.css">` and `<script src="/editor/editor-renderer.js">`

> **Concept: Type assertion**
> `c.Locals("state")` returns `interface{}` (Go's "any" type). `.(*state.AppState)` asserts that the value is actually an `*AppState`. If it's not, the program panics. Since we put it there ourselves in middleware, this is safe.

## REST API Handlers

### Get Blocks (Directory Tree)

```go
// internal/routes/editor.go
func getBlocks(c *fiber.Ctx) error {
    s := c.Locals("state").(*state.AppState)
    blocksPath := filepath.Join(s.UserPath, "blocks")
    tree := buildDirectoryTree(blocksPath)
    if tree == nil {
        tree = []BlockNode{}
    }
    return c.JSON(tree)
}
```

Returns a JSON array of block files and folders. Used by the editor's sidebar.

### Recursive Directory Tree

```go
type BlockNode struct {
    Name     string      `json:"name"`
    Type     string      `json:"type"`
    Path     string      `json:"path"`
    Children []BlockNode `json:"children,omitempty"`
}

func buildDirectoryTree(path string) []BlockNode {
    entries, err := os.ReadDir(path)
    if err != nil {
        return nil
    }

    sort.Slice(entries, func(i, j int) bool {
        return entries[i].Name() < entries[j].Name()
    })

    var result []BlockNode
    for _, entry := range entries {
        fullPath := filepath.Join(path, entry.Name())
        if entry.IsDir() {
            children := buildDirectoryTree(fullPath)  // recursion!
            result = append(result, BlockNode{
                Name:     entry.Name(),
                Type:     "folder",
                Path:     fullPath,
                Children: children,
            })
        } else if strings.HasSuffix(entry.Name(), ".html") {
            result = append(result, BlockNode{
                Name: entry.Name(),
                Type: "file",
                Path: fullPath,
            })
        }
    }
    return result
}
```

> **Concept: Recursion**
> `buildDirectoryTree` calls itself for each subdirectory. This naturally produces a nested tree structure. The base case is when a directory has no subdirectories — the recursive call returns an empty slice.

> **Concept: `os.ReadDir`**
> Returns `[]os.DirEntry`, which is more efficient than the older `os.ReadDir` (which returned `[]os.FileInfo`). `DirEntry` has `Name()`, `IsDir()`, and `Type()` methods without needing to stat each file.

### Get Panels List

```go
func getPanels(c *fiber.Ctx) error {
    s := c.Locals("state").(*state.AppState)
    panelsPath := filepath.Join(s.UserPath, "panels")

    allPanels := listPanels(panelsPath)

    return c.JSON(map[string]any{
        "allPanels": allPanels,
    })
}

func listPanels(panelsPath string) []string {
    entries, err := os.ReadDir(panelsPath)
    if err != nil {
        return nil
    }
    var panels []string
    for _, entry := range entries {
        name := entry.Name()
        if strings.HasSuffix(name, ".json") {
            panels = append(panels, strings.TrimSuffix(name, ".json"))
        }
    }
    sort.Strings(panels)
    return panels
}
```

Lists all `.json` files in `user/panels/`, strips the extension, and returns them sorted.

### Save Panel

```go
func savePanel(c *fiber.Ctx) error {
    s := c.Locals("state").(*state.AppState)
    panelsPath := filepath.Join(s.UserPath, "panels")

    var data map[string]any
    if err := c.BodyParser(&data); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(map[string]any{"success": false, "error": err.Error()})
    }

    fileName, _ := data["fileName"].(string)
    if fileName == "" {
        fileName = "new_panel"
    }

    content := data["content"]
    filePath := filepath.Join(panelsPath, fileName+".json")

    jsonData, err := json.MarshalIndent(content, "", "    ")
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(map[string]any{"success": false, "error": err.Error()})
    }

    if err := writeFile(filePath, jsonData); err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(map[string]any{"success": false, "error": err.Error()})
    }

    return c.JSON(map[string]any{"success": true, "path": filePath})
}
```

> **Concept: `c.BodyParser`**
> Fiber's `BodyParser` automatically parses the request body based on `Content-Type`. For JSON requests, it unmarshals into the provided pointer.

### Set Joystick Count

```go
func setJoystickCount(c *fiber.Ctx) error {
    s := c.Locals("state").(*state.AppState)
    var data map[string]any
    if err := c.BodyParser(&data); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(...)
    }
    countFloat, _ := data["count"].(float64)
    count := uint8(countFloat)
    s.UpdateJoystickCount(count)
    return c.JSON(map[string]any{"success": true, "count": count})
}
```

> **Concept: JSON numbers are float64**
> When Go unmarshals JSON into `map[string]any`, all numbers become `float64` (because JSON doesn't distinguish integer from float). We cast to `uint8` for the actual value.

### Push Data (REST API)

```go
func pushData(c *fiber.Ctx) error {
    s := c.Locals("state").(*state.AppState)
    var data map[string]any
    if err := c.BodyParser(&data); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(...)
    }
    key, _ := data["key"].(string)
    if key == "" {
        return c.JSON(map[string]any{"success": false, "error": "Missing key"})
    }
    value := data["value"]
    unit, _ := data["unit"].(string)
    source, _ := data["source"].(string)

    if source != "" {
        s.DataBus.SetSource(key, value, unit, source)
    } else {
        s.DataBus.Set(key, value, unit)
    }
    return c.JSON(map[string]any{"success": true, "key": key})
}
```

Allows external systems to push metrics into the DataBus via HTTP POST. These metrics then get broadcast to all connected WebSocket clients.

## Key Takeaways

- Fiber provides Express.js-style routing for Go
- Middleware + `c.Locals` shares `AppState` across all handlers
- `app.Static` serves entire directories with automatic Content-Type detection
- A targeted middleware sets `Cache-Control: no-cache, no-store, must-revalidate` for `.js`/`.css` files so changes are always picked up
- HTML pages use friendly URLs (`/`, `/panel`, `/editor`) with custom handlers
- The start page (`/`) combines panel list, editor link, host controls, and live connection log in a single page
- JS/CSS assets are served directly via `app.Static` — no individual handlers needed
- HTML references match the `static/` directory structure (e.g., `/client/client.js`)
- Recursive directory traversal builds nested JSON trees
- `c.BodyParser` handles JSON request body parsing automatically
- JSON numbers unmarshal as `float64` in `map[string]any`

[← Back: Chapter 4](04-databus.md) · [Next: Chapter 6 →](06-websocket.md)
