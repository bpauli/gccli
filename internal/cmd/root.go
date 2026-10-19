package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/alecthomas/kong"

	"github.com/bpauli/gccli/internal/config"
	"github.com/bpauli/gccli/internal/errfmt"
	"github.com/bpauli/gccli/internal/outfmt"
	"github.com/bpauli/gccli/internal/ui"
	"github.com/bpauli/gccli/internal/units"
)

// Globals holds runtime state injected into command Run methods.
type Globals struct {
	Context context.Context
	UI      *ui.UI
	Stdout  io.Writer
	Stderr  io.Writer
	Account string
	Parser  *kong.Kong
	Units   units.Preference
}

// CLI is the root Kong struct.
type CLI struct {
	RootFlags

	Version kong.VersionFlag `help:"Print version information." name:"version"`

	kong.Plugins
}

var commands = []Node{
	authGroup,
	activitiesGroup,
	activityGroup,
	workoutsGroup,
	healthGroup,
	bodyGroup,
	devicesGroup,
	gearGroup,
	goalsGroup,
	badgesGroup,
	coursesGroup,
	challengesGroup,
	recordsLeaf,
	profileGroup,
	hydrationGroup,
	nutritionGroup,
	trainingGroup,
	eventsGroup,
	wellnessGroup,
	exercisesGroup,
	notificationsGroup,
	socialGroup,
	Leaf{
		Name: "reload", Help: "Request data reload for a date.",
		Spec: argsOf(&struct {
			Date string `arg:"" optional:"" help:"Date to reload (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
		}{}),
		Endpoint: "wellness.request-reload",
		Params:   Params{"date": In("Date")},
		Verb:     "request reload", Output: JSON{},
		LateDates: []string{"Date"},
	},
	Custom{Name: "completion", Help: "Generate shell completion script.", Cmd: &CompletionCmd{}},
}

// Execute runs the CLI with the given arguments and version info.
// It returns the process exit code.
func Execute(args []string, version, commit, date string) int {
	return execute(args, version, commit, date, os.Stdout, os.Stderr)
}

func execute(args []string, version, commit, date string, stdout, stderr io.Writer) int {
	versionStr := fmt.Sprintf("gccli %s (%s) built %s", version, commit, date)

	if errs := checkTable(commands); len(errs) > 0 {
		for _, err := range errs {
			_, _ = fmt.Fprintf(stderr, "gccli: command table: %s\n", err)
		}
		return 1
	}

	cli := CLI{Plugins: attach(commands)}
	parser, err := kong.New(&cli,
		kong.Name("gccli"),
		kong.Description("Garmin Connect CLI"),
		kong.UsageOnError(),
		kong.Writers(stdout, stderr),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
		}),
		kong.Help(colorHelpPrinter),
		kong.Vars{
			"version": versionStr,
			"units":   strings.Join(units.PreferenceNames(), ","),
		},
	)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "gccli: %s\n", err)
		return 1
	}

	// Override Exit to avoid os.Exit during --help and --version.
	parser.Exit = func(code int) {
		panic(exitSignal{code: code})
	}

	return run(parser, &cli, args, stdout, stderr)
}

func run(parser *kong.Kong, cli *CLI, args []string, stdout, stderr io.Writer) (code int) {
	// Catch exit signals from Kong's --help and --version handlers.
	defer func() {
		if r := recover(); r != nil {
			if sig, ok := r.(exitSignal); ok {
				code = sig.code
				return
			}
			panic(r)
		}
	}()

	kongCtx, err := parser.Parse(args)
	if err != nil {
		parser.Stdout = stderr
		parser.Exit = func(int) { panic(exitSignal{code: kindUsage.exit}) }
		parser.FatalIfErrorf(err)
		return kindUsage.exit // unreachable: FatalIfErrorf panics via exitSignal
	}

	mode := cli.outputMode()

	// Determine color mode from flag and env var.
	colorMode := cli.Color
	if cm := config.ColorMode(); cm != "" && colorMode == "auto" {
		colorMode = cm
	}

	// Build context with output mode and UI.
	ctx := context.Background()
	ctx = outfmt.NewContext(ctx, mode)
	u := ui.NewWithWriter(stderr, colorMode)
	ctx = ui.NewContext(ctx, u)

	cfg, err := config.Read()
	if err != nil {
		cfg = &config.File{}
	}

	// Resolve account: flag/env → config file default.
	account := cli.Account
	if account == "" {
		account = cfg.Account()
	}

	// Inject runtime globals into the command.
	g := &Globals{
		Context: ctx,
		UI:      u,
		Stdout:  stdout,
		Stderr:  stderr,
		Account: account,
		Parser:  parser,
		Units:   unitsPreference(cli.Units, cfg, u),
	}

	if err := kongCtx.Run(g); err != nil {
		if mode == outfmt.JSON {
			writeJSONError(g.Stderr, err)
		} else {
			u.Error(fmt.Errorf("%s", errfmt.Format(err)))
		}
		return exitCode(err)
	}

	return 0
}

func unitsPreference(flag string, cfg *config.File, u *ui.UI) units.Preference {
	if p, err := units.ParsePreference(flag); err == nil {
		if _, pinned := p.Pinned(); pinned {
			return p
		}
	}
	p, err := units.ParsePreference(cfg.Units)
	if err != nil {
		u.Warnf("config: ignoring invalid \"units\" value: %v", err)
		return units.Auto
	}
	return p
}
