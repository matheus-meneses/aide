package updater

import "testing"

func TestIsNewer(t *testing.T) {
	cases := []struct {
		latest, current string
		want            bool
	}{
		{"v0.2.0", "v0.1.0", true},
		{"v0.1.0", "v0.2.0", false},
		{"v0.2.0", "v0.2.0", false},
		{"v0.2.0-rc.9", "v0.2.0-rc.8", true},
		{"v0.2.0-rc.8", "v0.2.0-rc.9", false},
		{"v0.2.0-rc.10", "v0.2.0-rc.9", true},
		{"v0.2.0-rc.8", "v0.2.0-rc.8", false},
		{"v0.2.0", "v0.2.0-rc.9", true},
		{"v0.2.0-rc.9", "v0.2.0", false},
		{"v0.2.0-rc.1", "v0.1.0", true},
		{"v0.1.0", "v0.2.0-rc.1", false},
		{"", "v0.1.0", false},
	}
	for _, c := range cases {
		if got := IsNewer(c.latest, c.current); got != c.want {
			t.Errorf("IsNewer(%q, %q) = %v, want %v", c.latest, c.current, got, c.want)
		}
	}
}
