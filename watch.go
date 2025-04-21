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
	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
)

var (
	debouncerMut    = sync.Mutex{}
	debouncerTimers = map[string]*time.Timer{}
	buildRoot       string
)

var (
	ignoreMatcher gitignore.Matcher = nil
	ignoreList                      = []string{".ccls-cache", "build", "out", "dist", "vendored", "vendor", "CMakeFiles", "vcpkg_installed", ".cache", "bin", "lib", "obj", "nbproject", ".vs", ".vscode", ".zed", ".idea", ".fleet", ".git"}
)

func toGitPath(path string) []string {
	if filepath.IsAbs(path) {
		relPath, err := filepath.Rel(buildRoot, path)
		if err == nil {
			path = relPath
		}
	}
	return strings.Split(path, string(filepath.Separator))
}

func watch(path string) {
	buildRoot = path
	ignoreMatcher = getIgnoreMatcher()

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
		if !d.IsDir() {
			return nil
		}
		gitPath := toGitPath(path)
		for _, ignoreStr := range ignoreList {
			if slices.Contains(gitPath, ignoreStr) {
				return filepath.SkipDir
			}
		}
		if ignoreMatcher != nil && ignoreMatcher.Match(gitPath, true) {
			return filepath.SkipDir
		}
		_ = watcher.Add(path)
		return nil
	}

	go devWatcher(watcher, watchAllDirs)

	_ = filepath.WalkDir(path, watchAllDirs)

	<-sigs
}

func debouncedBuild(name string) {
	debouncerMut.Lock()

	debouncerTimer, found := debouncerTimers[name]
	if found {
		if debouncerTimer != nil {
			debouncerTimer.Stop()
		}
	}

	debouncerTimer = time.NewTimer(500 * time.Millisecond)
	debouncerTimers[name] = debouncerTimer

	debouncerMut.Unlock()

	_, ok := <-debouncerTimer.C
	debouncerMut.Lock()
	if ok {
		generateHeader(name)
	}
	debouncerMut.Unlock()
}

func fullBuild() {
	_ = filepath.WalkDir(buildRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		isDir := d.IsDir()
		gitPath := toGitPath(path)
		if isDir {
			for _, ignoreStr := range ignoreList {
				if slices.Contains(gitPath, ignoreStr) {
					return filepath.SkipDir
				}
			}
		}
		if ignoreMatcher.Match(gitPath, isDir) {
			if isDir {
				return filepath.SkipDir
			} else {
				return nil
			}
		}
		if isDir {
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
	ignoreFileName := filepath.Join(buildRoot, ".helmet")
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			fileInfo, err := os.Stat(event.Name)
			if err == nil && fileInfo.IsDir() {
				_ = filepath.WalkDir(event.Name, watchAllDirs)
			} else if event.Has(fsnotify.Write | fsnotify.Create | fsnotify.Remove | fsnotify.Rename) {
				if strings.HasSuffix(event.Name, ".cpp") {
					gitPath := toGitPath(event.Name)
					if !ignoreMatcher.Match(gitPath, false) {
						go debouncedBuild(event.Name)
					}
				} else if event.Name == ignoreFileName {
					ignoreMatcher = getIgnoreMatcher()
				}
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			fmt.Println("error:", err)
		}
	}
}
