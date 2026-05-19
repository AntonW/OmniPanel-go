//go:build !windows

package speech

func ensureVoskWindowsRuntime(userPath string, runtimeURL string) error {
	return nil
}
