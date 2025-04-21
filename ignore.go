package main

import (
	"bufio"
	"os"
	"strings"

	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
)

func getIgnoreMatcher() gitignore.Matcher {
	var patterns []gitignore.Pattern
	if f, err := os.Open(".helmet"); err == nil {
		defer f.Close()
		scanner := bufio.NewScanner(f)

		// Read and print lines
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if !strings.HasPrefix(line, "#") {
				patterns = append(patterns, gitignore.ParsePattern(line, nil))
			}
		}
	}
	return gitignore.NewMatcher(patterns)
}
