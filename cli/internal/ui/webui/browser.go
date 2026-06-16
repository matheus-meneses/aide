package webui

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

func openBrowser(url string) {
	switch runtime.GOOS {
	case "darwin":
		if focusExistingChromeTab(url) {
			return
		}
		exec.Command("open", url).Start() //nolint:errcheck // fire-and-forget browser open
	case "linux":
		exec.Command("xdg-open", url).Start() //nolint:errcheck // fire-and-forget browser open
	}
}

func focusExistingChromeTab(url string) bool {
	match := strings.TrimPrefix(strings.TrimPrefix(url, "https://"), "http://")
	script := fmt.Sprintf(`tell application "System Events"
	if not (exists process "Google Chrome") then return "notrunning"
end tell
tell application "Google Chrome"
	repeat with theWin in windows
		set tabCount to count of tabs of theWin
		repeat with j from 1 to tabCount
			if (URL of tab j of theWin) contains "%s" then
				set active tab index of theWin to j
				set index of theWin to 1
				activate
				return "found"
			end if
		end repeat
	end repeat
end tell
return "notfound"`, match)

	out, err := exec.Command("osascript", "-e", script).Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "found"
}
