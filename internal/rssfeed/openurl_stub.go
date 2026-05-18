//go:build !linux && !windows

package rssfeed

import "errors"

// OpenURL returns an error on unsupported platforms.
func OpenURL(url string) error {
	return errors.New("open-url not supported on this platform")
}
