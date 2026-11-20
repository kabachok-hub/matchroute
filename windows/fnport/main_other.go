//go:build !windows

package main

import "fmt"

// The program is for Windows; on other systems only the tests of its core run.
func main() {
	fmt.Println("MatchRoute for Windows: build with GOOS=windows")
}
