package main

import (
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"strings"
)

var (
	funcIgnoreKeywords = []string{"constexpr", "inline"}
)

var (
	funcOneLineRE            = regexp.MustCompile(`((?:\w+ +)*[\w:*&<>]+\s+(?:const\s+)?[*&\w]+\s*\((?:\s*(?:const\s*)?[\w:*&<>]+\s+(?:const\s+)?[&*\w]+\s*(?:,\s*(?:const\s*)?[\w:*&<>]+\s+(?:const\s+)?[&*\w]+\s*)*)?\)(?:\s*const)?)\s*{(?:[^\n]+)}\n`)
	funcRE                   = regexp.MustCompile(`\n((?:\w+ +)*[\w:*&<>]+\s+(?:const\s+)?[&*\w]+\s*\((?:\s*(?:const\s*)?[\w:*&<>]+\s+(?:const\s+)?[&*\w]+\s*(?:,\s*(?:const\s*)?[\w:*&<>]+\s+(?:const\s+)?[&*\w]+\s*)*)?\)(?:\s*const)?)\s*{(?:(?:\n {1,}[^\n]+)|\n *)*\n}\n`)
	func4RE                  = regexp.MustCompile(`\n( {4}(?:\w+ +)*[\w:*&<>]+\s+(?:const\s+)?[&*\w]+\s*\((?:\s*(?:const\s*)?[\w:*&<>]+\s+(?:const\s+)?[&*\w]+\s*(?:,\s*(?:const\s*)?[\w:*&<>]+\s+(?:const\s+)?[&*\w]+\s*)*)?\)(?:\s*const)?)\s*{(?:(?:\n {5,}[^\n]+)|\n *)*\n {4}}\n`)
	func8RE                  = regexp.MustCompile(`\n( {8}(?:\w+ +)*[\w:*&<>]+\s+(?:const\s+)?[&*\w]+\s*\((?:\s*(?:const\s*)?[\w:*&<>]+\s+(?:const\s+)?[&*\w]+\s*(?:,\s*(?:const\s*)?[\w:*&<>]+\s+(?:const\s+)?[&*\w]+\s*)*)?\)(?:\s*const)?)\s*{(?:(?:\n {9,}[^\n]+)|\n *)*\n {8}}\n`)
	varRE                    = regexp.MustCompile(`\n[^\s][^\n]+[^=<>]=[^=<>][^\n]+;`)
	doubleNewlineRE          = regexp.MustCompile(`(\r?\n)(\r?\n)(\r?\n)+`)
	emptyLineBlockBeginRE    = regexp.MustCompile(`{\n\n`)
	includeGuardUnderscoreRE = regexp.MustCompile(`[._-]+`)
)

func generateHeaders(filenames []string) {
	for _, name := range filenames {
		if !strings.HasSuffix(name, ".cpp") {
			fmt.Fprintf(os.Stderr, "skipping file %s: not a C++ file\n", name)
			continue
		}

		generateHeader(name)
	}
}

func generateHeader(name string) {
	hppName := name[:len(name)-3] + "hpp"
	selfInclude := regexp.MustCompile(fmt.Sprintf(`\s*#include ".*%s"\n`, path.Base(hppName)))
	includeGuard := strings.ToUpper(includeGuardUnderscoreRE.ReplaceAllString(path.Base(hppName), "_"))

	cppF, err := os.Open(name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open file %s: %v\n", name, err)
		return
	}
	defer cppF.Close()

	cpp, err := io.ReadAll(cppF)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read file %s: %v\n", name, err)
		return
	}

	hpp := string(cpp)
	hpp = funcOneLineRE.ReplaceAllStringFunc(hpp, func(s string) string {
		sm := funcOneLineRE.FindSubmatch([]byte(s))
		funcHead := string(sm[1])
		for _, kw := range funcIgnoreKeywords {
			if strings.Contains(funcHead, kw) {
				return s
			}
		}
		return fmt.Sprintf("%s;\n", funcHead)
	})
	hpp = funcRE.ReplaceAllStringFunc(hpp, func(s string) string {
		sm := funcRE.FindSubmatch([]byte(s))
		funcHead := string(sm[1])
		for _, kw := range funcIgnoreKeywords {
			if strings.Contains(funcHead, kw) {
				return s
			}
		}
		return fmt.Sprintf("\n%s;\n", funcHead)
	})
	hpp = func4RE.ReplaceAllStringFunc(hpp, func(s string) string {
		sm := func4RE.FindSubmatch([]byte(s))
		funcHead := string(sm[1])
		for _, kw := range funcIgnoreKeywords {
			if strings.Contains(funcHead, kw) {
				return s
			}
		}
		return fmt.Sprintf("\n%s;\n", funcHead)
	})
	hpp = func8RE.ReplaceAllStringFunc(hpp, func(s string) string {
		sm := func8RE.FindSubmatch([]byte(s))
		funcHead := string(sm[1])
		for _, kw := range funcIgnoreKeywords {
			if strings.Contains(funcHead, kw) {
				return s
			}
		}
		return fmt.Sprintf("\n%s;\n", funcHead)
	})
	hpp = varRE.ReplaceAllString(hpp, "")
	hpp = selfInclude.ReplaceAllString(hpp, "")
	hpp = fmt.Sprintf("#ifndef %s\n#define %s\n\n%s\n\n#endif  // %s", includeGuard, includeGuard, hpp, includeGuard)
	hpp = doubleNewlineRE.ReplaceAllString(hpp, "\n\n")
	hpp = emptyLineBlockBeginRE.ReplaceAllString(hpp, "{\n")

	hppF, err := os.Create(hppName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create file %s: %v\n", hppName, err)
		return
	}
	defer hppF.Close()

	if _, err := hppF.Write([]byte(hpp)); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write file %s: %v\n", hppName, err)
		return
	}

	fmt.Printf("generated %s\n", hppName)
}
