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
	funcRE                   = regexp.MustCompile(`\s*{\n(\s+[^\n]+\n)+}`)
	varRE                    = regexp.MustCompile(`\s*=[^;]+`)
	doubleNewlineRE          = regexp.MustCompile(`(\r?\n)(\r?\n)(\r?\n)+`)
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

	hpp := funcRE.ReplaceAllString(string(cpp), ";")
	hpp = varRE.ReplaceAllString(hpp, "")
	hpp = selfInclude.ReplaceAllString(hpp, "")
	hpp = fmt.Sprintf("#ifndef %s\n#define %s\n\n%s\n\n#endif // %s", includeGuard, includeGuard, hpp, includeGuard)
	hpp = doubleNewlineRE.ReplaceAllString(hpp, "\n\n")

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
