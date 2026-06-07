package updater

import (
	"crypto/tls"
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
	NexusBaseURL   = "https://nexus.sharedservices.local/repository/aide"
	throttleFile   = ".last_version_check"
	throttleWindow = 12 * time.Hour
)

var httpClient = &http.Client{
	Timeout: 3 * time.Second,
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		Proxy:           http.ProxyFromEnvironment,
	},
}

func aideHome() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".aide")
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
	os.MkdirAll(filepath.Dir(path), 0o755)
	os.WriteFile(path, []byte(strconv.FormatInt(time.Now().Unix(), 10)), 0o644)
}

func fetchLatestVersion() (string, error) {
	resp, err := httpClient.Get(NexusBaseURL + "/VERSION")
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
	fmt.Fprintf(os.Stderr, "\n╭─────────────────────────────────────────────────────╮\n")
	fmt.Fprintf(os.Stderr, "│  A new version of aide is available: %-15s │\n", latest)
	fmt.Fprintf(os.Stderr, "│  Current: %-42s│\n", current)
	fmt.Fprintf(os.Stderr, "│                                                     │\n")
	fmt.Fprintf(os.Stderr, "│  curl -fsSL %s/install.sh | bash  │\n", NexusBaseURL)
	fmt.Fprintf(os.Stderr, "╰─────────────────────────────────────────────────────╯\n\n")
}

func DownloadFile(url string, dest *os.File, showProgress bool) error {
	resp, err := httpClient.Get(url)
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
	os.MkdirAll(filepath.Dir(destPath), 0o755)
	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()
	return DownloadFile(url, f, false)
}
