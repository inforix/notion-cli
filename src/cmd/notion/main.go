package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "-h", "--help", "help":
			fmt.Println("notion CLI (work in progress)")
			fmt.Println("\nUsage:\n  notion <command> [flags]\n")
			fmt.Println("Examples:\n  notion pages export <page_id>\n")
			return
		}
	}

	fmt.Println("notion CLI (work in progress)")
	fmt.Println("Run 'notion --help' for usage.")
}
