package main

import (
	"fmt"
	"headshot/internal/update"
	"os"
)

func main() {
	if len(os.Args) == 2 {
		switch os.Args[1] {
		case "watch":
			wd, _ := os.Getwd()
			watch(wd)
		case "build":
			buildRoot, _ = os.Getwd()
			ignoreMatcher = getIgnoreMatcher()
			fullBuild()
		case "update":
			if err := update.Update(false); err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}
		default:
			generateHeaders(os.Args[1:])
		}
	} else {
		generateHeaders(os.Args[1:])
	}
}

