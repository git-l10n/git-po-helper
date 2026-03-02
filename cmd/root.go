// Package cmd provides CLI implementations.
package cmd

import (
	"fmt"

	"github.com/git-l10n/git-po-helper/flag"
	"github.com/git-l10n/git-po-helper/repository"
	"github.com/git-l10n/git-po-helper/util"
	"github.com/git-l10n/git-po-helper/version"
	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = rootCommand{}

// errorWithUsage marks an error that should display command usage.
type errorWithUsage struct{ msg string }

func (e errorWithUsage) Error() string { return e.msg }

// NewErrorWithUsage creates an error that should display usage (e.g. argument/flag errors).
func NewErrorWithUsage(a ...interface{}) error {
	return errorWithUsage{msg: fmt.Sprintln(a...)}
}

// NewErrorWithUsageF creates an error that should display usage.
func NewErrorWithUsageF(format string, a ...interface{}) error {
	return errorWithUsage{msg: fmt.Sprintf(format, a...)}
}

// NewStandardError creates an error that should not display usage.
func NewStandardError(a ...interface{}) error {
	return fmt.Errorf("%s", fmt.Sprint(a...))
}

// NewStandardErrorF creates an error that should not display usage.
func NewStandardErrorF(format string, a ...interface{}) error {
	return fmt.Errorf(format, a...)
}

// IsErrorWithUsage returns true if the error should display command usage.
func IsErrorWithUsage(err error) bool {
	_, ok := err.(errorWithUsage)
	return ok
}

// Response wraps error for subcommand, and is returned from cmd package.
type Response struct {
	// Err contains error returned from the subcommand executed.
	Err error

	// Cmd contains the command object.
	Cmd *cobra.Command
}

type rootCommand struct {
	cmd *cobra.Command
}

func (v *rootCommand) initLog() {
	f := new(log.TextFormatter)
	f.DisableTimestamp = true
	f.DisableLevelTruncation = true
	if flag.GitHubActionEvent() != "" {
		f.ForceColors = true
	}
	log.SetFormatter(f)
	verbose := flag.Verbose()
	quiet := flag.Quiet()
	if verbose == 1 {
		log.SetLevel(log.DebugLevel)
	} else if verbose > 1 {
		log.SetLevel(log.TraceLevel)
	} else if quiet == 1 {
		log.SetLevel(log.WarnLevel)
	} else if quiet > 1 {
		log.SetLevel(log.ErrorLevel)
	}
}

func (v *rootCommand) initRepository() {
	repository.OpenRepository("")
}

func (v *rootCommand) preCheck() {
	if err := util.CheckPrereq(); err != nil {
		log.Fatal(err)
	}
}

// Command represents the base command when called without any subcommands
func (v *rootCommand) Command() *cobra.Command {
	if v.cmd != nil {
		return v.cmd
	}

	v.cmd = &cobra.Command{
		Use:   "git-po-helper",
		Short: "Helper for git l10n",
		// Let main.go handle error output; do not show usage on every error
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return v.Execute(args)
		},
	}
	v.cmd.Version = version.Version
	v.cmd.SetVersionTemplate(`{{with .Name}}{{printf "%s " .}}{{end}}{{printf "version %s" .Version}}
`)
	v.cmd.PersistentFlags().Bool("dryrun",
		false,
		"dryrun mode")
	v.cmd.PersistentFlags().CountP("quiet",
		"q",
		"quiet mode")
	v.cmd.PersistentFlags().CountP("verbose",
		"v",
		"verbose mode")
	v.cmd.PersistentFlags().String("github-action-event",
		"",
		"github-action event name")
	v.cmd.PersistentFlags().Bool("no-special-gettext-versions",
		false,
		"no check using gettext 0.14 for back compatible")
	v.cmd.PersistentFlags().String("pot-file",
		"auto",
		"way to get latest pot file: 'auto', 'download', 'build', 'no' or filename such as po/git.pot")
	v.cmd.PersistentFlags().String("config",
		"",
		"load agent configuration from this file (overrides ~/.git-po-helper.yaml and repo git-po-helper.yaml)")
	_ = v.cmd.PersistentFlags().MarkHidden("dryrun")
	_ = v.cmd.PersistentFlags().MarkHidden("no-special-gettext-versions")
	_ = v.cmd.PersistentFlags().MarkHidden("github-action-event")

	_ = viper.BindPFlag(
		"dryrun",
		v.cmd.PersistentFlags().Lookup("dryrun"))
	_ = viper.BindPFlag(
		"quiet",
		v.cmd.PersistentFlags().Lookup("quiet"))
	_ = viper.BindPFlag(
		"verbose",
		v.cmd.PersistentFlags().Lookup("verbose"))
	_ = viper.BindPFlag(
		"no-special-gettext-versions",
		v.cmd.PersistentFlags().Lookup("no-special-gettext-versions"))
	_ = viper.BindPFlag(
		"github-action-event",
		v.cmd.PersistentFlags().Lookup("github-action-event"))
	_ = viper.BindPFlag(
		"pot-file",
		v.cmd.PersistentFlags().Lookup("pot-file"))
	_ = viper.BindPFlag(
		"config",
		v.cmd.PersistentFlags().Lookup("config"))

	return v.cmd
}

func (v rootCommand) Execute(args []string) error {
	return NewErrorWithUsage("run 'git-po-helper -h' for help")
}

func (v *rootCommand) AddCommand(cmds ...*cobra.Command) {
	v.Command().AddCommand(cmds...)
}

// potFileVisibleCommands lists commands that use --pot-file; the flag is shown only for them.
var potFileVisibleCommands = map[string]bool{
	"check": true, "check-po": true, "check-commits": true,
	"check-pot": true, "init": true, "update": true,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() Response {
	var (
		resp Response
	)

	// Hide --pot-file for commands that do not use it (must run before ExecuteC).
	hidePotFileForCommands()

	// Ensure all commands use SilenceErrors so main.go handles error output.
	setSilenceErrorsRecursive(rootCmd.Command())

	c, err := rootCmd.Command().ExecuteC()
	resp.Err = err
	resp.Cmd = c
	return resp
}

func init() {
	cobra.OnInitialize(rootCmd.initLog)
	cobra.OnInitialize(rootCmd.initRepository)
	cobra.OnInitialize(rootCmd.preCheck)
}

// setSilenceErrorsRecursive sets SilenceErrors on c and all its descendants.
func setSilenceErrorsRecursive(c *cobra.Command) {
	c.SilenceErrors = true
	for _, child := range c.Commands() {
		setSilenceErrorsRecursive(child)
	}
}

// hidePotFileForCommands sets a custom help func for commands that do not use
// --pot-file, so the flag is hidden only when showing help for those commands.
// Commands in potFileVisibleCommands get the default help to avoid inheriting
// the hiding behavior from root.
func hidePotFileForCommands() {
	root := rootCmd.Command()
	defaultHelp := root.HelpFunc() // capture before modifying root
	var visit func(*cobra.Command)
	visit = func(c *cobra.Command) {
		if potFileVisibleCommands[c.Name()] {
			c.SetHelpFunc(defaultHelp)
		} else {
			markPotFileHiddenForHelp(c)
		}
		for _, child := range c.Commands() {
			visit(child)
		}
	}
	visit(root)
}

// markPotFileHiddenForHelp sets a help func that hides --pot-file before rendering.
func markPotFileHiddenForHelp(c *cobra.Command) {
	var baseHelp func(*cobra.Command, []string)
	if c.Parent() != nil {
		baseHelp = c.Parent().HelpFunc()
	} else {
		baseHelp = c.HelpFunc()
	}
	c.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		// For root, use PersistentFlags(); for children, use InheritedFlags()
		f := cmd.InheritedFlags().Lookup("pot-file")
		if f == nil {
			f = cmd.PersistentFlags().Lookup("pot-file")
		}
		if f != nil {
			f.Hidden = true
		}
		baseHelp(cmd, args)
	})
}
