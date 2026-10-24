//go:build !windows

package main

// markOfTheWeb is a no-op off Windows (there are no Zone.Identifier streams).
func markOfTheWeb(path string) error { return nil }

// hideFile and unhideFile are no-ops off Windows (the temp files' leading dot hides them).
func hideFile(path string)   {}
func unhideFile(path string) {}
