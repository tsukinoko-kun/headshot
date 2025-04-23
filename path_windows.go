//go:build windows

package main

import (
	"os"
	"path"
	"path/filepath"
	"strings"
)

func unixPath(systemPath string) string {
	if filepath.IsAbs(systemPath) {
		wd, _ := os.Getwd()
		systemPath, _ = filepath.Rel(wd, systemPath)
	}
	return path.Clean(strings.ReplaceAll(systemPath, "\\", "/"))
}
