package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime/debug"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"github.com/swtormy-agents/linkwarden-cli/internal/linkwarden"
)

const (
	exitRuntime = 1
	exitUsage   = 2
	exitAuth    = 3
	exitPolicy  = 4
)

type exitError struct {
	code int
	kind string
	err  error
	api  *linkwarden.APIError
}

func (e *exitError) Error() string { return e.err.Error() }

func usageErr(format string, a ...any) error {
	return &exitError{code: exitUsage, kind: "usage", err: fmt.Errorf(format, a...)}
}

func runtimeErr(err error) error {
	return &exitError{code: exitRuntime, kind: "runtime", err: err}
}

func authErr(err error) error {
	return &exitError{code: exitAuth, kind: "auth", err: err}
}

func policyErr(msg string) error {
	return &exitError{code: exitPolicy, kind: "policy", err: errors.New(msg)}
}

var Version = "dev"
var Commit = ""

func versionString() string {
	if Version != "dev" {
		out := Version
		if Commit != "" {
			c := Commit
			if len(c) > 7 {
				c = c[:7]
			}
			out += "+" + c
		}
		return out
	}
	if bi, ok := debug.ReadBuildInfo(); ok {
		rev, dirty := "", ""
		for _, s := range bi.Settings {
			switch s.Key {
			case "vcs.revision":
				rev = s.Value
			case "vcs.modified":
				if s.Value == "true" {
					dirty = "+dirty"
				}
			}
		}
		if len(rev) > 7 {
			rev = rev[:7]
		}
		if rev != "" {
			return "dev-" + rev + dirty
		}
	}
	return Version
}

type app struct {
	in     io.Reader
	out    io.Writer
	errOut io.Writer
	getenv func(string) string
	isTTY  func() bool

	cfg        linkwarden.Config
	dryRun     bool
	allowWrite bool
	jq         string
	pretty     bool
	baseURL    string
	timeout    time.Duration

	allowWriteWarnOnce sync.Once
}

func Execute(args []string, stdin io.Reader, stdout, stderr io.Writer, getenv func(string) string) int {
	if getenv == nil {
		getenv = os.Getenv
	}
	a := &app{
		in: stdin, out: stdout, errOut: stderr, getenv: getenv,
		isTTY: defaultIsTTY, timeout: 60 * time.Second,
	}
	root := a.rootCmd()
	root.SetArgs(args)
	root.SetOut(stdout)
	root.SetErr(stderr)
	if err := root.Execute(); err != nil {
		return a.reportError(err)
	}
	return 0
}

func (a *app) reportError(err error) int {
	ee, ok := err.(*exitError)
	if !ok {
		ee = &exitError{code: exitUsage, kind: "usage", err: err}
	}
	detail := map[string]any{"type": ee.kind, "message": a.redact(ee.err.Error())}
	if ee.api != nil {
		detail["status"] = ee.api.Status
		if ee.api.Message != "" {
			detail["message"] = a.redact(ee.api.Message)
		}
	}
	b, _ := marshal(struct {
		OK    bool           `json:"ok"`
		Error map[string]any `json:"error"`
	}{false, detail})
	fmt.Fprintln(a.errOut, string(b))
	return ee.code
}

func (a *app) redact(s string) string {
	return (&linkwarden.Client{Token: a.cfg.Token}).Redact(s)
}

func (a *app) loadConfig() error {
	cfg, err := linkwarden.LoadConfig(a.getenv)
	if err != nil {
		return usageErr("%s", err.Error())
	}
	if a.baseURL != "" {
		cfg.URL = a.baseURL
	}
	if a.allowWrite {
		cfg.AllowWrite = true
		cfg.AllowWriteSource = "--allow-write"
	}
	if cfg.URL != "" {
		n, insecure, err := linkwarden.ValidateBaseURL(cfg.URL)
		if err != nil {
			return usageErr("%s", err.Error())
		}
		cfg.URL = n
		if insecure {
			fmt.Fprintln(a.errOut, "linkwarden: warning: LINKWARDEN_URL uses http://; prefer https://")
		}
	}
	a.cfg = cfg
	return nil
}

func (a *app) writesAllowed() bool {
	return a.cfg.AllowWrite
}

func (a *app) client() (*linkwarden.Client, error) {
	if err := a.loadConfig(); err != nil {
		return nil, err
	}
	if a.cfg.URL == "" && !a.dryRun {
		return nil, usageErr("LINKWARDEN_URL is not set (export it or set url in config.json)")
	}
	if a.cfg.Token == "" && !a.dryRun {
		return nil, authErr(errors.New("LINKWARDEN_TOKEN is not set (session JWT or config.json with chmod 600)"))
	}
	return &linkwarden.Client{
		Token:   a.cfg.Token,
		BaseURL: a.cfg.URL,
		HTTP:    &http.Client{Timeout: a.timeout},
	}, nil
}

func (a *app) do(ctx context.Context, r linkwarden.Request) error {
	if a.dryRun {
		if err := a.loadConfig(); err != nil {
			return err
		}
		if a.cfg.URL == "" {
			return usageErr("LINKWARDEN_URL is not set")
		}
		return a.emitValue(linkwarden.DryRun(a.cfg.URL, r))
	}
	c, err := a.client()
	if err != nil {
		return err
	}
	raw, err := c.Do(ctx, r)
	if err != nil {
		return a.wrapErr(err)
	}
	return a.emit(raw)
}

func (a *app) wrapErr(err error) error {
	var api *linkwarden.APIError
	if errors.As(err, &api) {
		code, kind := exitRuntime, "api"
		if api.IsAuth() {
			code, kind = exitAuth, "auth"
		}
		return &exitError{code: code, kind: kind, err: api, api: api}
	}
	return runtimeErr(err)
}

func (a *app) requireWrite(action string) error {
	if err := a.loadConfig(); err != nil {
		return err
	}
	if a.writesAllowed() {
		a.warnAllowWriteSource()
		return nil
	}
	return policyErr(fmt.Sprintf("%s is blocked while allow_write is off: set LINKWARDEN_ALLOW_WRITE=true, allow_write in config.json, or pass --allow-write", action))
}

func (a *app) rootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "linkwarden",
		Short:         "Linkwarden bookmark API from the shell",
		Long:          rootLong,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	root.CompletionOptions.DisableDefaultCmd = true
	pf := root.PersistentFlags()
	pf.BoolVar(&a.dryRun, "dry-run", false, "print the HTTP request instead of sending it")
	pf.BoolVar(&a.allowWrite, "allow-write", false, "opt in to this write command (also LINKWARDEN_ALLOW_WRITE or config allow_write)")
	pf.StringVar(&a.jq, "jq", "", "filter JSON output with a jq expression")
	pf.BoolVar(&a.pretty, "pretty", false, "indent JSON output")
	pf.StringVar(&a.baseURL, "url", "", "Linkwarden instance URL (default $LINKWARDEN_URL)")
	pf.DurationVar(&a.timeout, "http-timeout", 60*time.Second, "HTTP timeout per request")

	root.AddCommand(
		a.versionCmd(),
		a.linksCmd(),
		a.collectionsCmd(),
		a.tagsCmd(),
		a.skillCmd(),
	)
	return root
}

const rootLong = `linkwarden calls the Linkwarden REST API (https://docs.linkwarden.app/api). JSON on stdout, errors as JSON on stderr.

Auth
  export LINKWARDEN_URL=https://linkwarden.example
  export LINKWARDEN_TOKEN=...           Authorization: Bearer header (never printed; masked as <TOKEN>)

Safety
  Create/update/delete link commands require allow_write. Default is read-only.
  Non-interactive: export LINKWARDEN_ALLOW_WRITE=true or "allow_write": true in config.json

Exit codes: 0 ok, 1 API/runtime, 2 usage, 3 auth, 4 blocked by allow_write policy`

func (a *app) versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version and commit as JSON",
		RunE: func(cmd *cobra.Command, args []string) error {
			commit := Commit
			if commit == "" {
				if bi, ok := debug.ReadBuildInfo(); ok {
					for _, s := range bi.Settings {
						if s.Key == "vcs.revision" {
							commit = s.Value
							break
						}
					}
				}
			}
			return a.emitValue(map[string]string{
				"version": versionString(),
				"commit":  commit,
			})
		},
	}
}

func (a *app) warnAllowWriteSource() {
	a.allowWriteWarnOnce.Do(func() {
		if !a.cfg.AllowWrite {
			return
		}
		var msg string
		switch a.cfg.AllowWriteSource {
		case "LINKWARDEN_ALLOW_WRITE":
			msg = "linkwarden: allow_write enabled via LINKWARDEN_ALLOW_WRITE (see README)"
		case "allow_write file":
			msg = "linkwarden: allow_write enabled via allow_write file in config directory (see README)"
		case "--allow-write":
			msg = "linkwarden: allow_write enabled via --allow-write flag for this invocation"
		case "default":
			return
		default:
			if a.cfg.AllowWriteSource != "" {
				msg = fmt.Sprintf("linkwarden: allow_write enabled via %q (see README)", a.cfg.AllowWriteSource)
			}
		}
		if msg != "" {
			fmt.Fprintln(a.errOut, msg)
		}
	})
}

func defaultIsTTY() bool {
	f, err := os.Stderr.Stat()
	if err != nil {
		return false
	}
	return (f.Mode() & os.ModeCharDevice) != 0
}
