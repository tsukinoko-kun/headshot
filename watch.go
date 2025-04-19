package main

import (
	"fmt"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
)

var (
	debouncerMut   = sync.Mutex{}
	debouncerTimer *time.Timer
	buildRoot      string
)

var ignoreList = []string{".ccls-cache", "build", "out", "dist", "CMakeFiles", "vcpkg_installed", ".cache", "bin", "lib", "obj", "nbproject", ".vs", ".vscode", ".zed", ".idea", ".fleet"}

func watch(path string) {
	buildRoot = path

	fullBuild()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Printf("failed to create fs watcher: %v\n", err)
		os.Exit(1)
	}
	defer watcher.Close()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	watchAllDirs := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		splitPath := strings.Split(path, string(filepath.Separator))
		for _, ignoreStr := range ignoreList {
			if slices.Contains(splitPath, ignoreStr) {
				return filepath.SkipDir
			}
		}
		if d.IsDir() {
			_ = watcher.Add(path)
		}
		return nil
	}

	go devWatcher(watcher, watchAllDirs)

	_ = filepath.WalkDir(path, watchAllDirs)

	<-sigs
}

func debouncedFullBuild() {
	debouncerMut.Lock()
	defer debouncerMut.Unlock()

	if debouncerTimer != nil {
		debouncerTimer.Stop()
	}

	debouncerTimer = time.NewTimer(500 * time.Millisecond)
	select {
	case <-debouncerTimer.C:
		fullBuild()
	}
}

func fullBuild() {
	_ = filepath.WalkDir(buildRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		splitPath := strings.Split(path, string(filepath.Separator))
		for _, ignoreStr := range ignoreList {
			if slices.Contains(splitPath, ignoreStr) {
				return filepath.SkipDir
			}
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".cpp") {
			return nil
		}

		generateHeader(path)
		return nil
	})
}

func devWatcher(watcher *fsnotify.Watcher, watchAllDirs func(string, fs.DirEntry, error) error) {
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			fileInfo, err := os.Stat(event.Name)
			if err == nil && fileInfo.IsDir() {
				_ = filepath.WalkDir(event.Name, watchAllDirs)
			}
			if event.Has(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) && !strings.HasSuffix(event.Name, ".hpp") {
				debouncedFullBuild()
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			fmt.Println("error:", err)
		}
	}
}
