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

	rel, err := LatestUpgrade(currentVersion)
	if err != nil {
		return
	}

	markChecked()

	if IsNewer(rel.Tag, currentVersion) {
		printUpdateBanner(currentVersion, rel.Tag)
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
	return fetchRelease("https://api.github.com/repos/" + repoSlug() + "/releases/latest")
}

// LatestUpgrade returns the most relevant release to offer for the given
// current version. Stable builds only consider stable releases; prerelease
// builds (e.g. v0.2.0-rc.8) also consider newer prereleases, so an rc can move
// to a newer rc without waiting for a stable cut.
func LatestUpgrade(current string) (Release, error) {
	if strings.Contains(strings.TrimSpace(current), "-") {
		return latestIncludingPre()
	}
	return LatestRelease()
}

func latestIncludingPre() (Release, error) {
	url := "https://api.github.com/repos/" + repoSlug() + "/releases?per_page=30"
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

	var items []struct {
		TagName string `json:"tag_name"`
		Body    string `json:"body"`
		HTMLURL string `json:"html_url"`
		Draft   bool   `json:"draft"`
	}
	if err := json.Unmarshal(body, &items); err != nil {
		return Release{}, err
	}

	var best Release
	for _, it := range items {
		tag := strings.TrimSpace(it.TagName)
		if it.Draft || parseSemver(tag) == nil {
			continue
		}
		if best.Tag == "" || compareVersions(tag, best.Tag) > 0 {
			best = Release{
				Tag:   tag,
				Notes: strings.TrimSpace(it.Body),
				URL:   strings.TrimSpace(it.HTMLURL),
			}
		}
	}
	if best.Tag == "" {
		return Release{}, fmt.Errorf("no releases found")
	}
	return best, nil
}

// ReleaseByTag fetches a specific release (including prereleases) by its tag.
// A missing leading "v" is added so callers can pass a bare version string.
func ReleaseByTag(tag string) (Release, error) {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return Release{}, fmt.Errorf("empty tag")
	}
	if !strings.HasPrefix(tag, "v") {
		tag = "v" + tag
	}
	return fetchRelease("https://api.github.com/repos/" + repoSlug() + "/releases/tags/" + tag)
}

func fetchRelease(url string) (Release, error) {
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

// IsNewer reports whether release tag `latest` is a newer version than
// `current`, honoring semver prerelease precedence: v0.2.0-rc.9 is newer than
// v0.2.0-rc.8, and a final v0.2.0 is newer than any v0.2.0-rc.N.
func IsNewer(latest, current string) bool {
	return compareVersions(latest, current) > 0
}

// compareVersions returns >0 if a is newer than b, <0 if older, 0 if equal.
func compareVersions(a, b string) int {
	an := parseSemver(a)
	bn := parseSemver(b)
	if an == nil || bn == nil {
		switch {
		case a == b:
			return 0
		case a > b:
			return 1
		default:
			return -1
		}
	}
	for i := 0; i < 3; i++ {
		if an[i] != bn[i] {
			if an[i] > bn[i] {
				return 1
			}
			return -1
		}
	}
	ap := prereleasePart(a)
	bp := prereleasePart(b)
	switch {
	case ap == "" && bp == "":
		return 0
	case ap == "":
		return 1
	case bp == "":
		return -1
	default:
		return comparePrerelease(ap, bp)
	}
}

// prereleasePart returns the prerelease identifiers of a version (the bit after
// "-"), with any build metadata ("+...") stripped. Empty for final releases.
func prereleasePart(v string) string {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	if i := strings.IndexByte(v, '+'); i >= 0 {
		v = v[:i]
	}
	i := strings.IndexByte(v, '-')
	if i < 0 {
		return ""
	}
	return v[i+1:]
}

// comparePrerelease compares dot-separated prerelease identifiers per semver:
// numeric identifiers compare numerically, numeric ranks below alphanumeric,
// and a longer identifier set wins when all preceding fields are equal.
func comparePrerelease(a, b string) int {
	as := strings.Split(a, ".")
	bs := strings.Split(b, ".")
	for i := 0; i < len(as) || i < len(bs); i++ {
		if i >= len(as) {
			return -1
		}
		if i >= len(bs) {
			return 1
		}
		ai, aerr := strconv.Atoi(as[i])
		bi, berr := strconv.Atoi(bs[i])
		switch {
		case aerr == nil && berr == nil:
			if ai != bi {
				if ai > bi {
					return 1
				}
				return -1
			}
		case aerr == nil:
			return -1
		case berr == nil:
			return 1
		default:
			if as[i] != bs[i] {
				if as[i] > bs[i] {
					return 1
				}
				return -1
			}
		}
	}
	return 0
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
