package main

import (
	"flag"
	"fmt"

	"github.com/codekaizen-github/wordpress-plugin-registry-oras/internal/andrew"
	"github.com/codekaizen-github/wordpress-plugin-registry-oras/server"
)

func main() {
	// Define command line flags
	versionFlag := flag.Bool("version", false, "Print version information and exit")
	flag.Parse()

	// If version flag is set, print version info and exit
	if *versionFlag {
		fmt.Printf("WordPress Plugin Registry ORAS v%s\n", server.Version)
		fmt.Printf("Commit: %s\n", server.Commit)
		fmt.Printf("Built: %s\n", server.Date)
		return
	}

	fmt.Println(andrew.Thing())

	// Initialize and start the server
	server.Initialize()
}
