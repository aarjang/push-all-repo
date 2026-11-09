package main

import (
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aarjang/push-all-repo/internal/model"
)

// version is set by the release build via -ldflags "-X main.version=…".
var version = "dev"

const usageText = `pushall [flags] [PATH]

Push every Git repository under a folder (or one repository), one at a time,
and print a report of what was pushed and what was not.

With no flags the tool is interactive. Flags pre-answer prompts.

Flags:
  --mode folder|single           what PATH is (default: prompt)
  --dirty ask|commit|push-only|skip
                                 what to do with uncommitted changes (default ask)
  --message TEXT                 commit message for --dirty commit
  --create-repo ask|yes|no       offer to create a GitHub repo when no remote (default ask)
  --visibility private|public    visibility for created repos (default private)
  --no-review                    skip the review/exclude step
  --report PATH                  save the Markdown report to PATH ("-" = never save)
  --timeout SECONDS              per-push timeout (default 120)
  --strict                       user-skipped repos count as failures
  --no-color                     plain output (also implied by NO_COLOR or non-TTY)
  --verbose                      echo every git/gh command to stderr
  --version                      print version and exit
  --help                         show this help

Exit codes: 0 all pushed or up to date · 1 something not pushed · 2 usage or path error · 130 interrupted
`

// parseFlags turns args into Options. done=true means the caller should exit
// with code (help or version printed, or a usage error reported to stderr).
func parseFlags(args []string, stdout, stderr io.Writer) (opts model.Options, code int, done bool) {
	opts = model.DefaultOptions()
	fs := flag.NewFlagSet("pushall", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var mode, dirty, createRepo, visibility string
	var timeout int
	var showVersion, showHelp bool
	fs.StringVar(&mode, "mode", "", "")
	fs.StringVar(&dirty, "dirty", string(model.DirtyAsk), "")
	fs.StringVar(&opts.Message, "message", "", "")
	fs.StringVar(&createRepo, "create-repo", string(model.CreateRepoAsk), "")
	fs.StringVar(&visibility, "visibility", "private", "")
	fs.BoolVar(&opts.NoReview, "no-review", false, "")
	fs.StringVar(&opts.ReportPath, "report", "", "")
	fs.IntVar(&timeout, "timeout", int(model.DefaultTimeout/time.Second), "")
	fs.BoolVar(&opts.Strict, "strict", false, "")
	fs.BoolVar(&opts.NoColor, "no-color", false, "")
	fs.BoolVar(&opts.Verbose, "verbose", false, "")
	fs.BoolVar(&showVersion, "version", false, "")
	fs.BoolVar(&showHelp, "help", false, "")
	fs.BoolVar(&showHelp, "h", false, "")

	// Allow flags before and after PATH: re-parse whatever follows each
	// positional argument.
	var positional []string
	rest := args
	for {
		if err := fs.Parse(rest); err != nil {
			fmt.Fprintf(stderr, "pushall: %v\n\n%s", err, usageText)
			return opts, 2, true
		}
		if fs.NArg() == 0 {
			break
		}
		positional = append(positional, fs.Arg(0))
		rest = fs.Args()[1:]
	}
	if showHelp {
		fmt.Fprint(stdout, usageText)
		return opts, 0, true
	}
	if showVersion {
		fmt.Fprintf(stdout, "pushall %s\n", version)
		return opts, 0, true
	}
	usage := func(msg string) (model.Options, int, bool) {
		fmt.Fprintf(stderr, "pushall: %s\n\n%s", msg, usageText)
		return opts, 2, true
	}

	switch strings.ToLower(mode) {
	case "":
		opts.Mode = model.ModeUnset
	case "folder":
		opts.Mode = model.FolderOfProjects
	case "single":
		opts.Mode = model.SingleProject
	default:
		return usage(fmt.Sprintf("invalid --mode %q (folder|single)", mode))
	}
	switch model.DirtyPolicy(strings.ToLower(dirty)) {
	case model.DirtyAsk, model.DirtyCommit, model.DirtyPushOnly, model.DirtySkip:
		opts.Dirty = model.DirtyPolicy(strings.ToLower(dirty))
	default:
		return usage(fmt.Sprintf("invalid --dirty %q (ask|commit|push-only|skip)", dirty))
	}
	switch model.CreateRepoPolicy(strings.ToLower(createRepo)) {
	case model.CreateRepoAsk, model.CreateRepoYes, model.CreateRepoNo:
		opts.CreateRepo = model.CreateRepoPolicy(strings.ToLower(createRepo))
	default:
		return usage(fmt.Sprintf("invalid --create-repo %q (ask|yes|no)", createRepo))
	}
	switch strings.ToLower(visibility) {
	case "private":
		opts.Visibility = model.Private
	case "public":
		opts.Visibility = model.Public
	default:
		return usage(fmt.Sprintf("invalid --visibility %q (private|public)", visibility))
	}
	if timeout <= 0 {
		return usage("--timeout must be a positive number of seconds")
	}
	opts.Timeout = time.Duration(timeout) * time.Second

	switch len(positional) {
	case 0:
	case 1:
		opts.Path = positional[0]
	default:
		return usage("expected at most one PATH argument")
	}
	return opts, 0, false
}
