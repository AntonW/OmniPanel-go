//go:build windows

package rssfeed

import "os/exec"

// OpenURL opens the given URL in the default browser on Windows.
func OpenURL(url string) error {
	return exec.Command("cmd", "/c", "start", url).Start()
}
