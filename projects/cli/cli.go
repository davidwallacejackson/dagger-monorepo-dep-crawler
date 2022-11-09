package main

import (
	"fmt"
	"os"

	"github.com/davidwallacejackson/dagger-monorepo-dep-crawler/lib"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: greet <name>")
		os.Exit(1)
	}

	fmt.Println(lib.Greet(os.Args[1]))
}
