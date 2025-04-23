package update

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

type GithubRelease struct {
	TagName string `json:"tag_name"`
}

func Update(force bool) error {
	// 1. Get the latest release JSON from GitHub
	releaseURL := "https://api.github.com/repos/tsukinoko-kun/headshot/releases/latest"
	resp, err := http.Get(releaseURL)
	if err != nil {
		return fmt.Errorf("failed to get latest release info: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var release GithubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return fmt.Errorf("failed to decode release info: %w", err)
	}

	// Assume tag is like "v1.0.1"; remove the "v" for filename formatting.
	version := release.TagName

	if semVer(version).IsNewerThan(semVer(Version)) {
		fmt.Printf("current version: %s\nlatest version: %s\n", Version, version)
	} else {
		fmt.Printf("current version %s is up to date\n", Version)
		if force {
			fmt.Println("forcing update")
		} else {
			return nil
		}
	}

	if len(version) > 0 && version[0] == 'v' {
		version = version[1:]
	}

	targetOS := runtime.GOOS
	targetArch := runtime.GOARCH

	// Determine expected binary name.
	binaryName := "headshot"
	if targetOS == "windows" {
		binaryName += ".exe"
	}

	// 2. Build download URL.
	downloadURL := fmt.Sprintf("https://github.com/tsukinoko-kun/headshot/releases/download/%s/%s_%s_%s_%s.tar.gz",
		release.TagName, "headshot", version, targetOS, targetArch)
	log.Printf("Downloading update from: %s", downloadURL)

	// 3. Download the tar.gz file.
	resp, err = http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download update: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return fmt.Errorf("failed to download update, status code: %d", resp.StatusCode)
	}
	// Create a temporary directory for the update.
	tmpDir, err := os.MkdirTemp("", "headshot-update")
	if err != nil {
		resp.Body.Close()
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	// Save downloaded archive.
	tarFilePath := filepath.Join(tmpDir, "update.tar.gz")
	tarFile, err := os.Create(tarFilePath)
	if err != nil {
		resp.Body.Close()
		return fmt.Errorf("failed to create tar file: %w", err)
	}
	_, err = io.Copy(tarFile, resp.Body)
	tarFile.Close()
	resp.Body.Close()
	if err != nil {
		return fmt.Errorf("failed to save tar file: %w", err)
	}

	// 4. Extract the binary file from the archive.
	newBinaryPath, err := extractBinary(tarFilePath, binaryName, tmpDir)
	if err != nil {
		return fmt.Errorf("failed to extract binary: %w", err)
	}
	log.Printf("Extracted new binary to: %s", newBinaryPath)

	// 5. Get the current executable path.
	currentPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get current executable path: %w", err)
	}
	currentPath, err = filepath.EvalSymlinks(currentPath)
	if err != nil {
		return fmt.Errorf("failed to evaluate symlinks: %w", err)
	}
	log.Printf("Current executable: %s", currentPath)

	// 6. Replace or swap the binary.
	if runtime.GOOS == "windows" {
		// Windows locks the executable so we must schedule a replacement.
		if err := updateOnWindows(currentPath, newBinaryPath); err != nil {
			return fmt.Errorf("windows update failed: %w", err)
		}
		log.Println("Update scheduled on Windows. Exiting...")
		os.Exit(0)
	} else {
		// On Unix (Linux/macOS), renaming the running executable works.
		if err := os.Rename(newBinaryPath, currentPath); err != nil {
			return fmt.Errorf("failed to replace binary: %w", err)
		}
		log.Println("Binary updated successfully")
	}

	return nil
}

func extractBinary(tarFilePath, binaryName, destDir string) (string, error) {
	file, err := os.Open(tarFilePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	gz, err := gzip.NewReader(file)
	if err != nil {
		return "", err
	}
	defer gz.Close()

	tarReader := tar.NewReader(gz)
	var newBinaryPath string
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}
		// Look for the expected binary file.
		if filepath.Base(header.Name) != binaryName {
			continue
		}

		newBinaryPath = filepath.Join(destDir, binaryName)
		outFile, err := os.Create(newBinaryPath)
		if err != nil {
			return "", err
		}
		if _, err := io.Copy(outFile, tarReader); err != nil {
			outFile.Close()
			return "", err
		}
		outFile.Close()
		// Make executable on Unix.
		if runtime.GOOS != "windows" {
			_ = os.Chmod(newBinaryPath, 0777)
		}
		break
	}
	if newBinaryPath == "" {
		return "", errors.New("binary not found in tar archive")
	}
	return newBinaryPath, nil
}

func updateOnWindows(currentPath, newBinaryPath string) error {
	// Create a temporary batch file that will:
	//   - Wait (via ping as a delay) until the current executable is no longer locked.
	//   - Move the new binary over the current one.
	//   - Restart the application.
	//   - Remove itself.
	batFile, err := os.CreateTemp("", "update-*.bat")
	if err != nil {
		return err
	}
	batPath := batFile.Name()
	script := fmt.Sprintf(`@echo off
echo Waiting for the application to exit...
ping 127.0.0.1 -n 5 > nul
move /Y "%s" "%s"
start "" "%s"
del "%%~f0"
exit
`, newBinaryPath, currentPath, currentPath)
	if _, err := batFile.WriteString(script); err != nil {
		batFile.Close()
		return err
	}
	batFile.Close()

	// Launch the batch file in a detached process.
	cmd := exec.Command("cmd", "/C", "start", "", batPath)
	return cmd.Start()
}
