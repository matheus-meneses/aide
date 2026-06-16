package updater

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
)

// fetchSHA256 downloads the published "<assetURL>.sha256" sidecar and returns
// the expected lowercase hex digest (the first whitespace-separated field, as
// produced by sha256sum/shasum -a 256).
func fetchSHA256(assetURL string) (string, error) {
	resp, err := httpClient.Get(assetURL + ".sha256")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("HTTP %d fetching checksum", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return "", err
	}
	fields := strings.Fields(string(body))
	if len(fields) == 0 {
		return "", fmt.Errorf("empty checksum file")
	}
	return strings.ToLower(fields[0]), nil
}

// verifyFileSHA256 hashes the file at path and fails if it does not match want.
func verifyFileSHA256(path, want string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	got := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(got, want) {
		return fmt.Errorf("checksum mismatch: downloaded %s, expected %s", got, want)
	}
	return nil
}
