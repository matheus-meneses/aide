//go:build darwin

package runner

import (
	"aide/cli/internal/xdg"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const systemTrustTTL = 6 * time.Hour

// SystemTrustBundle exports the macOS trust store (system roots + admin and
// user keychains) into a cached PEM file and returns its path. Plugins run
// inside a sandbox that denies the Mach access macOS needs to evaluate trust
// natively, so we hand them a file-based CA bundle instead, which OpenSSL can
// read without leaving the sandbox. Returns "" if the export fails.
func SystemTrustBundle() string {
	path := filepath.Join(xdg.AideHome(), "cache", "system-trust.pem")
	keychains := []string{
		"/System/Library/Keychains/SystemRootCertificates.keychain",
		"/Library/Keychains/System.keychain",
		filepath.Join(os.Getenv("HOME"), "Library", "Keychains", "login.keychain-db"),
	}

	if trustCacheFresh(path, keychains) {
		return path
	}

	var pem []byte
	for _, kc := range keychains {
		if _, err := os.Stat(kc); err != nil {
			continue
		}
		out, err := exec.Command("/usr/bin/security", "find-certificate", "-a", "-p", kc).Output()
		if err != nil || len(out) == 0 {
			continue
		}
		pem = append(pem, out...)
	}
	if len(pem) == 0 {
		return ""
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return ""
	}
	if err := os.WriteFile(path, pem, 0o600); err != nil {
		return ""
	}
	return path
}

// trustCacheFresh reports whether the cached bundle can be reused: it must
// exist, be non-empty, sit within the TTL backstop, and be newer than every
// keychain. Any keychain modified after the cache (a CA added/removed) forces a
// regeneration on the next run, so trust changes are reflected promptly.
func trustCacheFresh(path string, keychains []string) bool {
	fi, err := os.Stat(path)
	if err != nil || fi.Size() == 0 {
		return false
	}
	if time.Since(fi.ModTime()) >= systemTrustTTL {
		return false
	}
	for _, kc := range keychains {
		if kfi, err := os.Stat(kc); err == nil && kfi.ModTime().After(fi.ModTime()) {
			return false
		}
	}
	return true
}
