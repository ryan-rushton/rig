package updater

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type release struct {
	TagName string `json:"tag_name"`
}

// LatestRelease fetches the latest release tag from GitHub.
func LatestRelease() (string, error) {
	resp, err := http.Get("https://api.github.com/repos/ryan-rushton/rig/releases/latest")
	if err != nil {
		return "", fmt.Errorf("fetching latest release: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github API returned status %d", resp.StatusCode)
	}

	var r release
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return "", fmt.Errorf("decoding release response: %w", err)
	}

	if r.TagName == "" {
		return "", fmt.Errorf("empty tag_name in release response")
	}

	return r.TagName, nil
}

// IsNewer returns true if latest is newer than current.
// Returns false if current is "dev".
func IsNewer(current, latest string) bool {
	if current == "dev" {
		return false
	}
	return normalizeVersion(latest) > normalizeVersion(current)
}

// normalizeVersion pads each dot-separated segment to 4 digits for
// lexicographic comparison (e.g. "2025.1.3" → "2025.0001.0003").
func normalizeVersion(v string) string {
	v = strings.TrimPrefix(v, "v")
	parts := strings.Split(v, ".")
	for i, p := range parts {
		parts[i] = fmt.Sprintf("%04s", p)
	}
	return strings.Join(parts, ".")
}

// DownloadAndReplace downloads the release tarball for the given tag and
// replaces the current executable with the new binary.
func DownloadAndReplace(tag string) error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding executable path: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("resolving symlinks: %w", err)
	}

	goarch := runtime.GOARCH
	goos := runtime.GOOS

	// GoReleaser uses these naming conventions.
	osName := goos
	archName := goarch
	switch goarch {
	case "amd64":
		archName = "x86_64"
	case "arm64":
		archName = "arm64"
	}
	switch goos {
	case "darwin":
		osName = "Darwin"
	case "linux":
		osName = "Linux"
	}

	fileName := fmt.Sprintf("rig_%s_%s.tar.gz", osName, archName)
	url := fmt.Sprintf(
		"https://github.com/ryan-rushton/rig/releases/download/%s/%s",
		tag, fileName,
	)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("downloading release: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	binary, err := extractBinary(resp.Body)
	if err != nil {
		return fmt.Errorf("extracting binary: %w", err)
	}

	// Write to a temp file in the same directory, then atomically rename.
	dir := filepath.Dir(execPath)
	tmp, err := os.CreateTemp(dir, "rig-update-*")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmp.Name()

	cleanup := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}

	if _, err := tmp.Write(binary); err != nil {
		cleanup()
		return fmt.Errorf("writing temp file: %w", err)
	}
	if err := tmp.Chmod(0o755); err != nil {
		cleanup()
		return fmt.Errorf("setting permissions: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("closing temp file: %w", err)
	}

	if err := os.Rename(tmpPath, execPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("replacing executable: %w", err)
	}

	return nil
}

// extractBinary reads a tar.gz stream and returns the contents of the "rig" binary.
func extractBinary(r io.Reader) ([]byte, error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("creating gzip reader: %w", err)
	}
	defer func() { _ = gz.Close() }()

	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading tar: %w", err)
		}

		if filepath.Base(header.Name) == "rig" && header.Typeflag == tar.TypeReg {
			data, err := io.ReadAll(tr)
			if err != nil {
				return nil, fmt.Errorf("reading binary from tar: %w", err)
			}
			return data, nil
		}
	}

	return nil, fmt.Errorf("rig binary not found in archive")
}
