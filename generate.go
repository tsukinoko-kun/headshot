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

const (
	startInterface = `#ifndef %s
#define %s
#define HEADSHOT_INTERFACE

`
	start = `#ifndef %s
#define %s

`

	endInterface = `

#undef HEADSHOT_INTERFACE
#endif  // %s
`
	end = `

#endif  // %s
`
)

var (
	funcOneLineRE            = regexp.MustCompile(`((?:\w+ +)*[\w:*&<>()]+\s+(?:const\s+)?[*&\w]+\s*\((?:\s*(?:const\s*)?[\w:*&<>()]+\s+(?:const\s+)?[&*\w]+\s*(?:,\s*(?:const\s*)?[\w:*&<>()]+\s+(?:const\s+)?[&*\w]+\s*)*)?\)(?:\s*const)?)\s*{(?:[^\n]+)}\n`)
	funcRE                   = regexp.MustCompile(`\n((?:\w+ +)*[\w:*&<>()]+\s+(?:const\s+)?[&*\w]+\s*\((?:\s*(?:const\s*)?[\w:*&<>()]+\s+(?:const\s+)?[&*\w]+\s*(?:,\s*(?:const\s*)?[\w:*&<>()]+\s+(?:const\s+)?[&*\w]+\s*)*)?\)(?:\s*const)?)\s*{(?:(?:\n {1,}[^\n]+)|\n *)*\n}\n`)
	func4RE                  = regexp.MustCompile(`\n( {4}(?:\w+ +)*[\w:*&<>()]+\s+(?:const\s+)?[&*\w]+\s*\((?:\s*(?:const\s*)?[\w:*&<>()]+\s+(?:const\s+)?[&*\w]+\s*(?:,\s*(?:const\s*)?[\w:*&<>()]+\s+(?:const\s+)?[&*\w]+\s*)*)?\)(?:\s*const)?)\s*{(?:(?:\n {5,}[^\n]+)|\n *)*\n {4}}\n`)
	func8RE                  = regexp.MustCompile(`\n( {8}(?:\w+ +)*[\w:*&<>()]+\s+(?:const\s+)?[&*\w]+\s*\((?:\s*(?:const\s*)?[\w:*&<>()]+\s+(?:const\s+)?[&*\w]+\s*(?:,\s*(?:const\s*)?[\w:*&<>()]+\s+(?:const\s+)?[&*\w]+\s*)*)?\)(?:\s*const)?)\s*{(?:(?:\n {9,}[^\n]+)|\n *)*\n {8}}\n`)
	varRE                    = regexp.MustCompile(`\n(?:\w+\s+)*[\w:*&<>]+\s+\w+\s*(?:=[^;]+)?;`)
	clayRE                   = regexp.MustCompile(`#define\s+CLAY_IMPLEMENTATION`)
	interfaceRE              = regexp.MustCompile(`\bHEADSHOT_INTERFACE\b`)
	includeRE                = regexp.MustCompile(`#include\s+([<"]([^>"]+)[>"])(\s*)`)
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
	name = unixPath(name)
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
	hpp = clayRE.ReplaceAllString(hpp, "")
	hpp = includeRE.ReplaceAllStringFunc(hpp, func(s string) string {
		sm := includeRE.FindStringSubmatch(s)
		lib := sm[2]
		if strings.HasSuffix(lib, "SDL_main.h") {
			return ""
		}
		if strings.HasSuffix(lib, ".cpp") {
			return fmt.Sprintf(
				"#include %c%s.hpp%c%s",
				sm[1][0],            // " or <
				lib[:len(lib)-4],    // include file without .cpp extension
				sm[1][len(sm[1])-1], // " or >
				sm[3],               // trailing whitespace after include
			)
		}
		return s
	})
	hpp = selfInclude.ReplaceAllString(hpp, "")
	if interfaceRE.MatchString(hpp) {
		hpp = fmt.Sprintf(startInterface, includeGuard, includeGuard) +
			hpp +
			fmt.Sprintf(endInterface, includeGuard)
		hpp = interfaceRE.ReplaceAllString(hpp, "HEADSHOT_INTERFACE_"+includeGuard)
	} else {
		hpp = fmt.Sprintf(start, includeGuard, includeGuard) +
			hpp +
			fmt.Sprintf(end, includeGuard)
	}
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
