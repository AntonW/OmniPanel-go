// Package systray provides a cross-platform system tray icon for OmniPanel-go.
// It offers a context menu with fullscreen toggle and exit application options.
// On Linux, the tray is only initialized if a display server (X11 or Wayland)
// is detected via the DISPLAY or WAYLAND_DISPLAY environment variables.
package systray

import (
	_ "embed"
	"os"
	"sync"

	"fyne.io/systray"
)

//go:embed icon.png
var iconData []byte

// Tray manages the system tray icon and its menu items.
type Tray struct {
	mu             sync.Mutex
	fullscreen     bool
	fullscreenItem *systray.MenuItem
	exitItem       *systray.MenuItem
	onFullscreen   func()
	onExit         func()
	quitChan       chan struct{}
}

// IsHeadless returns true if no display server is detected (Linux only).
func IsHeadless() bool {
	display := os.Getenv("DISPLAY")
	wayland := os.Getenv("WAYLAND_DISPLAY")
	return display == "" && wayland == ""
}

// New creates a new system tray instance.
// onFullscreen is called when the fullscreen menu item is clicked.
// onExit is called when the exit menu item is clicked.
func New(onFullscreen, onExit func()) *Tray {
	return &Tray{
		onFullscreen: onFullscreen,
		onExit:       onExit,
		quitChan:     make(chan struct{}),
	}
}

// Run starts the system tray main loop. This call blocks until Quit() is called.
func (t *Tray) Run() {
	systray.Run(t.onReady, t.onExit)
	<-t.quitChan
}

// Quit signals the tray to exit.
func (t *Tray) Quit() {
	systray.Quit()
	close(t.quitChan)
}

// onReady sets up the tray icon, title, tooltip, and menu items.
// It registers click handlers for the fullscreen toggle and exit items.
// The fullscreen item uses a static "Toggle Fullscreen" label — the internal
// state tracks whether fullscreen is active, but the menu text does not change.
func (t *Tray) onReady() {
	systray.SetIcon(iconData)
	systray.SetTitle("OmniPanel")
	systray.SetTooltip("OmniPanel-go")

	t.fullscreenItem = systray.AddMenuItem("Toggle Fullscreen", "Toggle fullscreen on all clients")
	t.exitItem = systray.AddMenuItem("Exit Application", "Shut down OmniPanel-go")

	go func() {
		for {
			select {
			case <-t.fullscreenItem.ClickedCh:
				t.toggleFullscreen()
			case <-t.exitItem.ClickedCh:
				t.onExit()
				return
			}
		}
	}()
}

// toggleFullscreen flips the internal fullscreen state and calls the onFullscreen callback.
// The menu item title and tooltip remain static ("Toggle Fullscreen") — the caller
// (main.go) is responsible for broadcasting enter-fullscreen or exit-fullscreen to clients.
func (t *Tray) toggleFullscreen() {
	t.mu.Lock()
	t.fullscreen = !t.fullscreen
	t.fullscreenItem.SetTitle("Toggle fullscreen")
	t.fullscreenItem.SetTooltip("Toggle fullscreen on all clients")

	t.mu.Unlock()

	if t.onFullscreen != nil {
		t.onFullscreen()
	}
}
