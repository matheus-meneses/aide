package updater

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// ErrUpToDate is returned by Apply when no newer release is available.
var ErrUpToDate = errors.New("already up to date")

// Progress receives human-readable status lines emitted while an update runs.
type Progress func(line string)

func (p Progress) emit(format string, args ...any) {
	if p != nil {
		p(fmt.Sprintf(format, args...))
	}
}

// Result describes the outcome of Apply.
type Result struct {
	// Version is the release tag that was installed.
	Version string
	// RestartNow is true when the update was staged by a detached helper that
	// is waiting for this process to exit (the macOS app flow); the caller
	// must quit so the helper can swap the bundle and relaunch.
	RestartNow bool
}

// Apply performs an in-place update appropriate to how aide was installed.
// currentVersion is the running build version; method comes from DetectMethod.
// Progress lines are reported via prog (may be nil).
func Apply(ctx context.Context, currentVersion string, method Method, prog Progress) (Result, error) {
	rel, err := LatestUpgrade(currentVersion)
	if err != nil {
		return Result{}, fmt.Errorf("checking latest version: %w", err)
	}
	if !IsNewer(rel.Tag, currentVersion) {
		return Result{}, ErrUpToDate
	}

	switch method {
	case MethodScript:
		if err := applyScript(ctx, rel, prog); err != nil {
			return Result{}, err
		}
		return Result{Version: rel.Tag}, nil
	case MethodHomebrewFormula:
		if err := applyBrewFormula(ctx, prog); err != nil {
			return Result{}, err
		}
		return Result{Version: rel.Tag}, nil
	case MethodHomebrewCask, MethodManualApp:
		if err := applyAppUpdate(ctx, method, rel, prog); err != nil {
			return Result{}, err
		}
		return Result{Version: rel.Tag, RestartNow: true}, nil
	default:
		return Result{}, fmt.Errorf("automatic update is not supported for this installation")
	}
}

// downloadURL builds the release asset URL for a specific tag. AIDE_RELEASE_URL
// overrides the base (used in tests / mirrors).
func downloadURL(tag, asset string) string {
	if v := os.Getenv("AIDE_RELEASE_URL"); v != "" {
		return strings.TrimRight(v, "/") + "/" + asset
	}
	tag = strings.TrimSpace(tag)
	if !strings.HasPrefix(tag, "v") {
		tag = "v" + tag
	}
	return "https://github.com/" + repoSlug() + "/releases/download/" + tag + "/" + asset
}

// applyScript replaces a binary that was installed by the install script (or any
// standalone copy aide can write to) with the latest release binary.
func applyScript(_ context.Context, rel Release, prog Progress) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if resolved, rErr := filepath.EvalSymlinks(exe); rErr == nil {
		exe = resolved
	}

	asset := fmt.Sprintf("aide_%s_%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		asset += ".exe"
	}
	url := downloadURL(rel.Tag, asset)

	prog.emit("Downloading %s…", rel.Tag)
	tmp := exe + ".new"
	if err := DownloadToPath(url, tmp); err != nil {
		return fmt.Errorf("downloading update: %w", err)
	}
	defer os.Remove(tmp)

	prog.emit("Verifying download…")
	want, err := fetchSHA256(url)
	if err != nil {
		return fmt.Errorf("fetching checksum: %w", err)
	}
	if err := verifyFileSHA256(tmp, want); err != nil {
		return err
	}

	if err := os.Chmod(tmp, 0o755); err != nil {
		return err
	}

	prog.emit("Installing…")
	if err := os.Rename(tmp, exe); err != nil {
		return fmt.Errorf("replacing binary: %w", err)
	}

	prog.emit("Updated to %s. Restart aide to use the new version.", rel.Tag)
	return nil
}

// applyBrewFormula upgrades a Homebrew-managed CLI. brew owns the binary and
// runs unprivileged against its user-owned prefix, so this never overwrites
// files ourselves and never needs admin.
func applyBrewFormula(ctx context.Context, prog Progress) error {
	brew := brewBin()
	if brew == "" {
		return fmt.Errorf("homebrew not found")
	}
	prog.emit("Updating Homebrew…")
	if err := runStreaming(ctx, prog, brew, "update"); err != nil {
		prog.emit("brew update warning: %v", err)
	}
	_ = runStreaming(ctx, prog, brew, "tap", "matheus-meneses/aide")
	prog.emit("Upgrading aide…")
	if err := runStreaming(ctx, prog, brew, "upgrade", "aide"); err != nil {
		return fmt.Errorf("brew upgrade failed: %w", err)
	}
	prog.emit("Updated. Restart aide to use the new version.")
	return nil
}

// runStreaming runs a command, streaming each line of combined output to prog.
func runStreaming(ctx context.Context, prog Progress, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = brewExecEnv(name)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	var wg sync.WaitGroup
	scan := func(r io.Reader) {
		defer wg.Done()
		sc := bufio.NewScanner(r)
		sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for sc.Scan() {
			if line := strings.TrimSpace(sc.Text()); line != "" {
				prog.emit("%s", line)
			}
		}
	}
	wg.Add(2)
	go scan(stdout)
	go scan(stderr)
	wg.Wait()

	return cmd.Wait()
}

// brewExecEnv ensures the brew bin directory is on PATH so brew can find its
// own helpers when invoked from a GUI app with a minimal environment.
func brewExecEnv(name string) []string {
	env := os.Environ()
	dir := filepath.Dir(name)
	for i, kv := range env {
		if strings.HasPrefix(kv, "PATH=") {
			env[i] = "PATH=" + dir + string(os.PathListSeparator) + strings.TrimPrefix(kv, "PATH=")
			return env
		}
	}
	return append(env, "PATH="+dir)
}
