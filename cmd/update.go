package cmd

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

	"github.com/spf13/cobra"
)

type ghRelease struct {
	TagName string `json:"tag_name"`
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update queryit to the latest release",
	RunE: func(cmd *cobra.Command, args []string) error {
		const repo = "ShammiAnand/queryit"
		const binary = "queryit"

		fmt.Println("checking for updates...")

		rel, err := fetchLatestRelease(repo)
		if err != nil {
			return fmt.Errorf("fetch latest release: %w", err)
		}

		if rel.TagName == version {
			fmt.Printf("already at latest version (%s)\n", version)
			return nil
		}

		platform := runtime.GOOS + "_" + runtime.GOARCH
		url := fmt.Sprintf(
			"https://github.com/%s/releases/download/%s/%s_%s.tar.gz",
			repo, rel.TagName, binary, platform,
		)

		fmt.Printf("downloading %s (%s)...\n", rel.TagName, platform)

		bin, err := downloadAndExtract(url, binary, platform)
		if err != nil {
			return err
		}
		defer os.Remove(bin)

		selfPath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("locate current binary: %w", err)
		}
		selfPath, err = filepath.EvalSymlinks(selfPath)
		if err != nil {
			return fmt.Errorf("resolve symlink: %w", err)
		}

		if err := replaceBinary(selfPath, bin); err != nil {
			return err
		}

		fmt.Printf("updated to %s\n", rel.TagName)
		return nil
	},
}

func fetchLatestRelease(repo string) (*ghRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned %s", resp.Status)
	}

	var rel ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}
	if rel.TagName == "" {
		return nil, fmt.Errorf("no tag_name in response")
	}
	return &rel, nil
}

func downloadAndExtract(url, binary, platform string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed: HTTP %s\nCheck https://github.com/ShammiAnand/queryit/releases for available builds", resp.Status)
	}

	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return "", fmt.Errorf("decompress: %w", err)
	}
	defer gz.Close()

	want := binary + "_" + platform
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("read archive: %w", err)
		}

		name := filepath.Base(hdr.Name)
		if name != want && name != binary && !strings.HasPrefix(name, binary) {
			continue
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}

		tmp, err := os.CreateTemp("", "queryit-update-*")
		if err != nil {
			return "", err
		}
		if _, err := io.Copy(tmp, tr); err != nil {
			tmp.Close()
			os.Remove(tmp.Name())
			return "", fmt.Errorf("extract: %w", err)
		}
		tmp.Close()
		if err := os.Chmod(tmp.Name(), 0755); err != nil {
			os.Remove(tmp.Name())
			return "", err
		}
		return tmp.Name(), nil
	}

	return "", fmt.Errorf("binary not found in archive")
}

func replaceBinary(dst, src string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dir := filepath.Dir(dst)
	tmp, err := os.CreateTemp(dir, ".queryit-update-*")
	if err != nil {
		return fmt.Errorf("cannot write to %s (try running with sudo)", dir)
	}
	tmpPath := tmp.Name()

	if _, err := io.Copy(tmp, srcFile); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	tmp.Close()

	if err := os.Chmod(tmpPath, 0755); err != nil {
		os.Remove(tmpPath)
		return err
	}

	if err := os.Rename(tmpPath, dst); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("replace binary: %w (try running with sudo)", err)
	}
	return nil
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
