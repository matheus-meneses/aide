package updater

import (
	"context"
	"fmt"
	"os"
)

// AutoCheck runs a throttled (12h) update check. When autoApply is true and the
// installation can be upgraded in place from the CLI (script or Homebrew
// formula), it applies the update and prints progress to stderr; otherwise it
// prints the upgrade banner. All network/update errors are swallowed so this
// never disrupts the command the user actually ran.
func AutoCheck(version string, autoApply bool) {
	if version == "" || version == "dev" {
		return
	}
	if !shouldCheck() {
		return
	}
	rel, err := LatestUpgrade(version)
	if err != nil {
		return
	}
	markChecked()
	if !IsNewer(rel.Tag, version) {
		return
	}

	if !autoApply {
		printUpdateBanner(version, rel.Tag)
		return
	}

	method := DetectMethod(version)
	// App methods need the GUI process to relaunch via a detached helper, which
	// makes no sense from a one-shot CLI command — fall back to the banner.
	if method != MethodScript && method != MethodHomebrewFormula {
		printUpdateBanner(version, rel.Tag)
		return
	}

	fmt.Fprintf(os.Stderr, "\nUpdating aide %s -> %s…\n", version, rel.Tag)
	prog := Progress(func(line string) { fmt.Fprintf(os.Stderr, "  %s\n", line) })
	if _, err := Apply(context.Background(), version, method, prog); err != nil {
		fmt.Fprintf(os.Stderr, "  update failed: %v\n", err)
	}
}
