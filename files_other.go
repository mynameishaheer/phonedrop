//go:build !windows

package main

// markOfTheWeb is a no-op off Windows (there are no Zone.Identifier streams).
func markOfTheWeb(path string) error { return nil }
