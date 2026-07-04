//go:build !windows

package main

// hideConsole 在非 Windows 平台為空實作
func hideConsole() {}

// showConsole 在非 Windows 平台為空實作
func showConsole() {}
