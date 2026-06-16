//go:build !darwin

package runner

// SystemTrustBundle is a no-op outside macOS: on Linux the sandbox can read the
// system trust files directly, so OpenSSL/truststore resolve trust without help.
func SystemTrustBundle() string { return "" }
