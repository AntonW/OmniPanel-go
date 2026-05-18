//go:build linux

package rssfeed

import "os/exec"

// OpenURL opens the given URL in the default browser on Linux.
func OpenURL(url string) error {
	return exec.Command("xdg-open", url).Start()
}
