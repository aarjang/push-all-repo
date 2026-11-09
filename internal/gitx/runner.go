package gitx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// Runner executes one external command. Every process the tool spawns goes
// through a Runner so the environment hardening lives in one place.
type Runner interface {
	Run(ctx context.Context, dir string, timeout time.Duration, args ...string) (stdout, stderr string, err error)
}

// ExitError is returned when the command ran but failed.
type ExitError struct {
	Code      int
	Stderr    string
	TimedOut  bool
	Cancelled bool
}

func (e *ExitError) Error() string {
	switch {
	case e.TimedOut:
		return "command timed out"
	case e.Cancelled:
		return "command cancelled"
	}
	line := LastLine(e.Stderr)
	if line == "" {
		return fmt.Sprintf("exit status %d", e.Code)
	}
	return fmt.Sprintf("exit status %d: %s", e.Code, line)
}

// LastLine returns the last non-empty line of s, trimmed.
func LastLine(s string) string {
	lines := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if l := strings.TrimSpace(lines[i]); l != "" {
			return l
		}
	}
	return ""
}

// ExecRunner runs real processes.
type ExecRunner struct {
	// Program is the executable to run; defaults to "git" or $PUSHALL_GIT.
	Program string
	// Verbose, when non-nil, receives one line per command.
	Verbose io.Writer
	env     []string
}

// NewExecRunner returns a runner for git, honouring the PUSHALL_GIT override.
func NewExecRunner(verbose io.Writer) *ExecRunner {
	prog := os.Getenv("PUSHALL_GIT")
	if prog == "" {
		prog = "git"
	}
	return &ExecRunner{Program: prog, Verbose: verbose, env: HardenedEnv(os.Environ())}
}

// NewProgramRunner returns a runner for another executable (e.g. gh) with the
// same hardened environment.
func NewProgramRunner(program string, verbose io.Writer) *ExecRunner {
	return &ExecRunner{Program: program, Verbose: verbose, env: HardenedEnv(os.Environ())}
}

// HardenedEnv returns base plus the variables that stop git from ever
// prompting (contracts/git-commands.md "Runner environment").
func HardenedEnv(base []string) []string {
	has := func(key string) bool {
		for _, kv := range base {
			if strings.HasPrefix(kv, key+"=") {
				return true
			}
		}
		return false
	}
	env := make([]string, 0, len(base)+8)
	for _, kv := range base {
		k := strings.SplitN(kv, "=", 2)[0]
		switch k {
		case "GIT_TERMINAL_PROMPT", "LC_ALL", "GIT_OPTIONAL_LOCKS", "GCM_INTERACTIVE":
			continue // overridden below
		}
		env = append(env, kv)
	}
	env = append(env,
		"GIT_TERMINAL_PROMPT=0",
		"LC_ALL=C",
		"GIT_OPTIONAL_LOCKS=0",
		"GCM_INTERACTIVE=never",
	)
	if !has("GIT_ASKPASS") && !has("SSH_ASKPASS") {
		if p := noopAskpass(); p != "" {
			env = append(env, "GIT_ASKPASS="+p)
		}
	}
	if !has("GIT_SSH_COMMAND") && !has("GIT_SSH") {
		env = append(env, "GIT_SSH_COMMAND=ssh -o BatchMode=yes")
	}
	return env
}

// noopAskpass returns a program that prints nothing and exits successfully,
// so a credential prompt yields an empty answer instead of blocking.
func noopAskpass() string {
	if runtime.GOOS == "windows" {
		return "" // GIT_TERMINAL_PROMPT=0 and GCM_INTERACTIVE=never suffice
	}
	for _, p := range []string{"/usr/bin/true", "/bin/true"} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	if p, err := exec.LookPath("true"); err == nil {
		return p
	}
	return ""
}

// Run executes Program with args in dir. A timeout <= 0 means none.
func (r *ExecRunner) Run(ctx context.Context, dir string, timeout time.Duration, args ...string) (string, string, error) {
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	cmd := exec.CommandContext(ctx, r.Program, args...)
	cmd.Dir = dir
	cmd.Env = r.env
	cmd.Stdin = nil
	// After the context ends the process is killed; do not wait forever for
	// grandchildren (hooks, ssh) that still hold our pipes.
	cmd.WaitDelay = time.Second
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	start := time.Now()
	runErr := cmd.Run()
	stdout, stderr := out.String(), errb.String()

	var exitErr error
	if runErr != nil {
		ee := &ExitError{Code: -1, Stderr: stderr}
		var xe *exec.ExitError
		if errors.As(runErr, &xe) {
			ee.Code = xe.ExitCode()
		}
		if ctx.Err() != nil {
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				ee.TimedOut = true
			} else {
				ee.Cancelled = true
			}
		} else if ee.Code == -1 {
			// could not start (program missing etc.)
			exitErr = fmt.Errorf("run %s: %w", r.Program, runErr)
		}
		if exitErr == nil {
			exitErr = ee
		}
	}
	if r.Verbose != nil {
		code := 0
		var ee *ExitError
		if errors.As(exitErr, &ee) {
			code = ee.Code
		} else if exitErr != nil {
			code = -1
		}
		fmt.Fprintf(r.Verbose, "$ %s %s -> exit %d (%s)\n", r.Program, strings.Join(args, " "), code, time.Since(start).Round(time.Millisecond))
	}
	return stdout, stderr, exitErr
}
