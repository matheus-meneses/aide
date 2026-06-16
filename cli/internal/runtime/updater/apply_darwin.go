//go:build darwin

package updater

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// applyAppUpdate updates the macOS desktop app. The download/verify work runs
// now (with progress), then a detached helper waits for this process to exit
// before swapping the bundle and relaunching — the caller must quit afterwards.
func applyAppUpdate(ctx context.Context, method Method, rel Release, prog Progress) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if resolved, rErr := filepath.EvalSymlinks(exe); rErr == nil {
		exe = resolved
	}
	appPath := appBundlePath(exe)
	if appPath == "" {
		return fmt.Errorf("could not locate the Aide.app bundle")
	}

	if method == MethodHomebrewCask {
		return spawnCaskHelper(appPath, prog)
	}
	return spawnDMGHelper(ctx, appPath, rel, prog)
}

// spawnDMGHelper downloads and verifies the release DMG, stages the new bundle,
// and launches a detached helper that swaps it in once the app quits.
func spawnDMGHelper(_ context.Context, appPath string, rel Release, prog Progress) error {
	ver := strings.TrimPrefix(rel.Tag, "v")
	asset := fmt.Sprintf("Aide-%s.dmg", ver)
	url := downloadURL(rel.Tag, asset)

	work, err := os.MkdirTemp("", "aide-dmg-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(work)

	dmg := filepath.Join(work, asset)
	prog.emit("Downloading %s…", rel.Tag)
	if err := DownloadToPath(url, dmg); err != nil {
		return fmt.Errorf("downloading update: %w", err)
	}

	prog.emit("Verifying download…")
	want, err := fetchSHA256(url)
	if err != nil {
		return fmt.Errorf("fetching checksum: %w", err)
	}
	if err := verifyFileSHA256(dmg, want); err != nil {
		return err
	}

	prog.emit("Preparing update…")
	mount := filepath.Join(work, "mnt")
	if err := os.MkdirAll(mount, 0o755); err != nil {
		return err
	}
	if out, err := exec.Command("hdiutil", "attach", "-nobrowse", "-noverify", "-mountpoint", mount, dmg).CombinedOutput(); err != nil {
		return fmt.Errorf("mounting dmg: %w: %s", err, strings.TrimSpace(string(out)))
	}

	stageParent, err := os.MkdirTemp("", "aide-stage-*")
	if err != nil {
		_ = exec.Command("hdiutil", "detach", "-force", mount).Run()
		return err
	}
	stagedApp := filepath.Join(stageParent, "Aide.app")
	//nolint:gosec // paths are internal temp/mount locations, not user input
	cpErr := exec.Command("cp", "-R", filepath.Join(mount, "Aide.app"), stagedApp).Run()
	_ = exec.Command("hdiutil", "detach", "-force", mount).Run()
	if cpErr != nil {
		os.RemoveAll(stageParent)
		return fmt.Errorf("staging update: %w", cpErr)
	}

	script := dmgSwapScript(os.Getpid(), appPath, stagedApp, stageParent)
	if err := launchHelper(stageParent, script); err != nil {
		os.RemoveAll(stageParent)
		return err
	}

	prog.emit("Update ready. Restarting to finish…")
	return nil
}

// spawnCaskHelper launches a detached helper that runs `brew upgrade --cask`
// after the app quits, then relaunches it. brew can't replace a running app,
// so the work happens post-exit.
func spawnCaskHelper(appPath string, prog Progress) error {
	brew := brewBin()
	if brew == "" {
		return fmt.Errorf("homebrew not found")
	}
	stageParent, err := os.MkdirTemp("", "aide-cask-*")
	if err != nil {
		return err
	}
	script := caskSwapScript(os.Getpid(), brew, appPath, stageParent)
	if err := launchHelper(stageParent, script); err != nil {
		os.RemoveAll(stageParent)
		return err
	}
	prog.emit("Update ready. Restarting to finish…")
	return nil
}

func dmgSwapScript(pid int, appPath, stagedApp, stageParent string) string {
	return fmt.Sprintf(`#!/bin/bash
APP=%q
STAGED=%q
STAGE_PARENT=%q
PARENT_DIR="$(dirname "$APP")"

while kill -0 %d 2>/dev/null; do sleep 0.2; done

if [ -w "$PARENT_DIR" ]; then
  rm -rf "$APP"
  cp -R "$STAGED" "$APP"
  /usr/bin/xattr -dr com.apple.quarantine "$APP" 2>/dev/null || true
  /usr/bin/codesign --force --deep --sign - "$APP" 2>/dev/null || true
else
  CMD="rm -rf '$APP'; cp -R '$STAGED' '$APP'; /usr/bin/xattr -dr com.apple.quarantine '$APP'; /usr/bin/codesign --force --deep --sign - '$APP'"
  /usr/bin/osascript -e "do shell script \"$CMD\" with administrator privileges"
fi

rm -rf "$STAGE_PARENT"
/usr/bin/open "$APP"
`, appPath, stagedApp, stageParent, pid)
}

func caskSwapScript(pid int, brew, appPath, stageParent string) string {
	return fmt.Sprintf(`#!/bin/bash
BREW=%q
APP=%q
STAGE_PARENT=%q

while kill -0 %d 2>/dev/null; do sleep 0.2; done

"$BREW" update >/dev/null 2>&1 || true
"$BREW" tap matheus-meneses/aide >/dev/null 2>&1 || true
"$BREW" upgrade --cask aide >/dev/null 2>&1 || true

rm -rf "$STAGE_PARENT"
/usr/bin/open "$APP"
`, brew, appPath, stageParent, pid)
}

// launchHelper writes the script next to stageParent and starts it in a new
// session so it survives this process exiting.
func launchHelper(stageParent, script string) error {
	scriptPath := filepath.Join(filepath.Dir(stageParent), "aide-update-"+filepath.Base(stageParent)+".sh")
	if err := os.WriteFile(scriptPath, []byte(script), 0o600); err != nil {
		return err
	}
	cmd := exec.Command("/bin/bash", scriptPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	return cmd.Start()
}
