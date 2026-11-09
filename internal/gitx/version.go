package gitx

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"time"
)

// MinMajor and MinMinor are the oldest supported git version.
const (
	MinMajor = 2
	MinMinor = 20
)

var versionRe = regexp.MustCompile(`git version (\d+)\.(\d+)`)

// ParseVersion extracts major and minor from `git --version` output.
func ParseVersion(out string) (major, minor int, err error) {
	m := versionRe.FindStringSubmatch(out)
	if m == nil {
		return 0, 0, fmt.Errorf("unrecognised git version output %q", out)
	}
	major, _ = strconv.Atoi(m[1])
	minor, _ = strconv.Atoi(m[2])
	return major, minor, nil
}

// Available checks that git runs and is at least MinMajor.MinMinor.
func Available(ctx context.Context, r Runner) (string, error) {
	out, _, err := r.Run(ctx, "", 15*time.Second, "--version")
	if err != nil {
		return "", fmt.Errorf("git is not installed or not on PATH: %w", err)
	}
	major, minor, err := ParseVersion(out)
	if err != nil {
		return "", err
	}
	if major < MinMajor || (major == MinMajor && minor < MinMinor) {
		return "", fmt.Errorf("git %d.%d is too old; %d.%d or newer is required", major, minor, MinMajor, MinMinor)
	}
	return fmt.Sprintf("%d.%d", major, minor), nil
}
