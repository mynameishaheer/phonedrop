//go:build !windows

package main

import "fmt"

// PhoneDrop is a Windows app; on other systems only the portable parts build (for the tests).
func main() { fmt.Println("PhoneDrop runs on Windows.") }
