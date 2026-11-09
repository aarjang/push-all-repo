//go:build !windows

package ui

import (
	"os"
	"strings"
)

// consoleIsUTF8 guesses from the locale variables.
func consoleIsUTF8() bool {
	for _, k := range []string{"LC_ALL", "LC_CTYPE", "LANG"} {
		if v := os.Getenv(k); v != "" {
			v = strings.ToLower(v)
			return strings.Contains(v, "utf-8") || strings.Contains(v, "utf8")
		}
	}
	return false
}
