//go:build !windows

package systray

import _ "embed"

// trayIconData contains PNG bytes for tray implementations that accept PNG.
//go:embed icon.png
var trayIconData []byte

