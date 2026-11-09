package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/aarjang/push-all-repo/internal/github"
	"github.com/aarjang/push-all-repo/internal/gitx"
	"github.com/aarjang/push-all-repo/internal/run"
	"github.com/aarjang/push-all-repo/internal/ui"
)

func main() {
	os.Exit(realMain(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func realMain(args []string, stdin, stdout, stderr *os.File) int {
	opts, code, done := parseFlags(args, stdout, stderr)
	if done {
		return code
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		stop() // a second Ctrl+C now terminates immediately
	}()

	interactive := ui.IsTTY(stdin) && ui.IsTTY(stdout)
	style := ui.NewStyle(opts.NoColor || !ui.IsTTY(stdout))

	var verbose io.Writer
	if opts.Verbose {
		verbose = stderr
	}
	gitRunner := gitx.NewExecRunner(verbose)
	if _, err := gitx.Available(ctx, gitRunner); err != nil {
		fmt.Fprintf(stderr, "pushall: %v\n", err)
		return run.ExitUsage
	}

	var prompter ui.Prompter = ui.NonInteractivePrompter{}
	if interactive {
		prompter = ui.NewHuhPrompter(style, stdout)
	}
	cwd, _ := os.Getwd()

	_, exit := run.Run(ctx, opts, run.Deps{
		Git:      gitx.New(gitRunner),
		GitHub:   github.NewGhClient(verbose),
		Prompter: prompter,
		Progress: &ui.Progress{W: stdout, Style: style},
		Out:      stdout,
		Err:      stderr,
		Style:    style,
		Version:  version,
		Cwd:      cwd,
	})
	return exit
}
