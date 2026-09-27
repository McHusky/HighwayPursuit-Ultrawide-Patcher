//go:build !windows

package main

import "fmt"

func showInfo(title, text string)  { fmt.Printf("[%s]\n%s\n", title, text) }
func showError(title, text string) { fmt.Printf("[%s ERROR]\n%s\n", title, text) }
func askYesNo(title, text string) bool {
	fmt.Printf("[%s]\n%s\n(auto-yes on non-Windows test build)\n", title, text)
	return true
}
