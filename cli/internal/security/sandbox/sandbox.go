package sandbox

// Policy describes the sandbox restrictions applied to a plugin subprocess.
//
// Network is intentionally coarse: the underlying OS sandboxes (sandbox-exec,
// bwrap) cannot filter by host, so a non-empty Network enables all network
// access while an empty Network denies it. The host list documents intent and
// gates the all-or-nothing toggle; it is not a per-host allowlist.
type Policy struct {
	Name    string
	Dir     string
	Browser bool
	Network []string
	Reads   []string
	Writes  []string
}
