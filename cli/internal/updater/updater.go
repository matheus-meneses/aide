package updater

import (
	"aide/cli/internal/xdg"
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
	throttleFile          = ".last_version_check"
	throttleWindow        = 12 * time.Hour
)

func releaseBaseURL() string {
	if v := os.Getenv("AIDE_RELEASE_URL"); v != "" {
		return v
	}
	return defaultReleaseBaseURL
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

	latest, err := fetchLatestVersion()
	if err != nil {
		return
	}

	markChecked()

	if latest != "" && latest != currentVersion {
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

func fetchLatestVersion() (string, error) {
	resp, err := httpClient.Get(releaseBaseURL() + "/VERSION")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(body)), nil
}

func printUpdateBanner(current, latest string) {
	installURL := "https://raw.githubusercontent.com/matheus-meneses/aide/main/install.sh"
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
				fmt.Printf("\r      %.1f%% (%d / %d MB)", pct, written/(1024*1024), size/(1024*1024))
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}
	fmt.Println()
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
