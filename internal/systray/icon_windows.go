//go:build windows

package systray

import _ "embed"

// trayIconData contains Windows ICO bytes for the tray icon.
//go:embed icon.ico
var trayIconData []byte

