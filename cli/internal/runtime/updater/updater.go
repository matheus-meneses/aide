package updater

import (
	"aide/cli/internal/platform/xdg"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	defaultReleaseBaseURL = "https://github.com/matheus-meneses/aide/releases/latest/download"
	defaultRepoSlug       = "matheus-meneses/aide"
	throttleFile          = ".last_version_check"
	throttleWindow        = 12 * time.Hour
)

func releaseBaseURL() string {
	if v := os.Getenv("AIDE_RELEASE_URL"); v != "" {
		return v
	}
	return defaultReleaseBaseURL
}

func repoSlug() string {
	if v := os.Getenv("AIDE_REPO"); v != "" {
		return v
	}
	return defaultRepoSlug
}

func InstallURL() string {
	return releaseBaseURL() + "/install.sh"
}

var httpClient = &http.Client{
	Timeout: 5 * time.Second,
	Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	},
}

func aideHome() string {
	return xdg.AideHome()
}

func CheckOnce(currentVersion string) {
	if currentVersion == "dev" {
		return
	}

	if !shouldCheck() {
		return
	}

	latest, err := LatestVersion()
	if err != nil {
		return
	}

	markChecked()

	if IsNewer(latest, currentVersion) {
		printUpdateBanner(currentVersion, latest)
	}
}

func shouldCheck() bool {
	path := filepath.Join(aideHome(), throttleFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return true
	}
	ts, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return true
	}
	return time.Since(time.Unix(ts, 0)) >= throttleWindow
}

func markChecked() {
	path := filepath.Join(aideHome(), throttleFile)
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	_ = os.WriteFile(path, []byte(strconv.FormatInt(time.Now().Unix(), 10)), 0o600)
}

// Release describes the latest published release and its notes.
type Release struct {
	Tag   string
	Notes string
	URL   string
}

// LatestRelease fetches the latest (non-prerelease) release from GitHub,
// including the tag, release-notes markdown, and the HTML release URL.
func LatestRelease() (Release, error) {
	url := "https://api.github.com/repos/" + repoSlug() + "/releases/latest"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return Release{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "aide-updater")

	resp, err := httpClient.Do(req)
	if err != nil {
		return Release{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Release{}, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Release{}, err
	}

	var payload struct {
		TagName string `json:"tag_name"`
		Body    string `json:"body"`
		HTMLURL string `json:"html_url"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return Release{}, err
	}
	return Release{
		Tag:   strings.TrimSpace(payload.TagName),
		Notes: strings.TrimSpace(payload.Body),
		URL:   strings.TrimSpace(payload.HTMLURL),
	}, nil
}

// LatestVersion returns just the tag of the latest release.
func LatestVersion() (string, error) {
	rel, err := LatestRelease()
	if err != nil {
		return "", err
	}
	return rel.Tag, nil
}

func IsNewer(latest, current string) bool {
	if latest == "" || latest == current {
		return false
	}
	lv := parseSemver(latest)
	cv := parseSemver(current)
	if lv == nil || cv == nil {
		return latest != current
	}
	for i := 0; i < 3; i++ {
		if lv[i] != cv[i] {
			return lv[i] > cv[i]
		}
	}
	return false
}

func parseSemver(v string) []int {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ".")
	out := make([]int, 3)
	for i := 0; i < 3 && i < len(parts); i++ {
		n, err := strconv.Atoi(parts[i])
		if err != nil {
			return nil
		}
		out[i] = n
	}
	return out
}

func printUpdateBanner(current, latest string) {
	installURL := InstallURL()
	fmt.Fprintf(os.Stderr, "\n╭──────────────────────────────────────────────────────────────╮\n")
	fmt.Fprintf(os.Stderr, "│  A new version of aide is available: %-15s        │\n", latest)
	fmt.Fprintf(os.Stderr, "│  Current: %-52s│\n", current)
	fmt.Fprintf(os.Stderr, "│                                                              │\n")
	fmt.Fprintf(os.Stderr, "│  curl -fsSL %s | bash  │\n", installURL)
	fmt.Fprintf(os.Stderr, "╰──────────────────────────────────────────────────────────────╯\n\n")
}

func DownloadFile(url string, dest *os.File, showProgress bool) error {
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
		},
	}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}

	if !showProgress {
		_, err = io.Copy(dest, resp.Body)
		return err
	}

	size := resp.ContentLength
	written := int64(0)
	buf := make([]byte, 32*1024)

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, wErr := dest.Write(buf[:n]); wErr != nil {
				return wErr
			}
			written += int64(n)
			if size > 0 {
				pct := float64(written) / float64(size) * 100
				fmt.Fprintf(os.Stdout, "\r      %.1f%% (%d / %d MB)", pct, written/(1024*1024), size/(1024*1024))
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}
	fmt.Fprintln(os.Stdout)
	return nil
}

func DownloadToPath(url, destPath string) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return fmt.Errorf("creating parent dirs: %w", err)
	}
	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()
	return DownloadFile(url, f, false)
}
