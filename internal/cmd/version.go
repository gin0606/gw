package cmd

import "fmt"

// Version prints the version string to stdout.
func Version(version string) {
	fmt.Printf("gw version %s\n", version)
}
