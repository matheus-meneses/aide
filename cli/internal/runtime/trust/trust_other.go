//go:build !darwin

package trust

// SystemBundle is a no-op outside macOS: on Linux the sandbox can read the
// system trust files directly, so OpenSSL/truststore resolve trust without help.
func SystemBundle() string { return "" }
