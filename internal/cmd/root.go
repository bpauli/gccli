package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/alecthomas/kong"

	"github.com/bpauli/gccli/internal/config"
	"github.com/bpauli/gccli/internal/errfmt"
	"github.com/bpauli/gccli/internal/outfmt"
	"github.com/bpauli/gccli/internal/ui"
)

// Globals holds runtime state injected into command Run methods.
type Globals struct {
	Context context.Context
	UI      *ui.UI
	Account string
}

// CLI is the top-level command structure parsed by Kong.
type CLI struct {
	RootFlags

	Version kong.VersionFlag `help:"Print version information." name:"version"`

	Auth       AuthCmd       `cmd:"" help:"Manage authentication."`
	Activities ActivitiesCmd `cmd:"" help:"List and search activities."`
	Activity   ActivityCmd   `cmd:"" help:"View activity details."`
	Workouts   WorkoutsCmd   `cmd:"" help:"Manage workouts."`
	Health     HealthCmd     `cmd:"" help:"View health data."`
	Body       BodyCmd       `cmd:"" help:"Body composition and weight data."`
	Devices    DevicesCmd    `cmd:"" help:"Manage devices."`
	Gear       GearCmd       `cmd:"" help:"Manage gear."`
	Goals      GoalsCmd      `cmd:"" help:"View goals."`
	Badges     BadgesCmd     `cmd:"" help:"View badges."`
	Challenges ChallengesCmd `cmd:"" help:"View challenges."`
	Records    RecordsCmd    `cmd:"" help:"View personal records."`
	Profile    ProfileCmd    `cmd:"" help:"View user profile."`
	Hydration  HydrationCmd  `cmd:"" help:"Track hydration."`
	Training   TrainingCmd   `cmd:"" help:"View training plans."`
	Wellness   WellnessCmd   `cmd:"" help:"View wellness data."`
	Reload     ReloadCmd     `cmd:"" help:"Request data reload for a date."`
}

// Execute runs the CLI with the given arguments and version info.
// It returns the process exit code.
func Execute(args []string, version, commit, date string) int {
	versionStr := fmt.Sprintf("gccli %s (%s) built %s", version, commit, date)

	var cli CLI

	parser, err := kong.New(&cli,
		kong.Name("gccli"),
		kong.Description("Garmin Connect CLI"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
		}),
		kong.Help(colorHelpPrinter),
		kong.Vars{"version": versionStr},
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gccli: %s\n", err)
		return 1
	}

	// Override Exit to avoid os.Exit during --help and --version.
	parser.Exit = func(code int) {
		panic(exitSignal{code: code})
	}

	return run(parser, &cli, args)
}

func run(parser *kong.Kong, cli *CLI, args []string) (code int) {
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
		parser.FatalIfErrorf(err)
		return 1 // unreachable: FatalIfErrorf panics via exitSignal
	}

	// Determine output mode from flags and env vars.
	mode := outfmt.Table
	if cli.JSON || config.IsJSON() {
		mode = outfmt.JSON
	} else if cli.Plain || config.IsPlain() {
		mode = outfmt.Plain
	}

	// Determine color mode from flag and env var.
	colorMode := cli.Color
	if cm := config.ColorMode(); cm != "" && colorMode == "auto" {
		colorMode = cm
	}

	// Build context with output mode and UI.
	ctx := context.Background()
	ctx = outfmt.NewContext(ctx, mode)
	u := ui.New(colorMode)
	ctx = ui.NewContext(ctx, u)

	// Resolve account: flag/env → config file default.
	account := cli.Account
	if account == "" {
		if cfg, err := config.Read(); err == nil {
			account = cfg.Account()
		}
	}

	// Inject runtime globals into the command.
	g := &Globals{
		Context: ctx,
		UI:      u,
		Account: account,
	}

	if err := kongCtx.Run(g); err != nil {
		u.Error(fmt.Errorf("%s", errfmt.Format(err)))
		return exitCode(err)
	}

	return 0
}
