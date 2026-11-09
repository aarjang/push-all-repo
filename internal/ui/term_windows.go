//go:build windows

package ui

import "golang.org/x/sys/windows"

// consoleIsUTF8 reports whether the console output code page is UTF-8 (65001).
func consoleIsUTF8() bool {
	return windows.GetConsoleOutputCP() == 65001
}
