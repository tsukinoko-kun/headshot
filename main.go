package main

import "os"

func main() {
	if len(os.Args) == 2 && os.Args[1] == "watch" {
		wd, _ := os.Getwd()
		watch(wd)
	} else {
		generateHeaders(os.Args[1:])
	}
}
