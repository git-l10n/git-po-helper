// Package cmd provides CLI implementations.
package cmd

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/git-l10n/git-po-helper/flag"
	"github.com/git-l10n/git-po-helper/repository"
	"github.com/git-l10n/git-po-helper/util"
	"github.com/git-l10n/git-po-helper/version"
	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	rootCmd    = rootCommand{}
	errExecute = errors.New("fail to execute")
)

// commandError is an error used to signal different error situations in command handling.
type commandError struct {
	s         string
	userError bool
}

func (c commandError) Error() string {
	return c.s
}

func (c commandError) isUserError() bool {
	return c.userError
}

func newUserError(a ...interface{}) commandError {
	return commandError{s: fmt.Sprintln(a...), userError: true}
}

func newUserErrorF(format string, a ...interface{}) commandError {
	return commandError{s: fmt.Sprintf(format, a...), userError: true}
}

// Catch some of the obvious user errors from Cobra.
// We don't want to show the usage message for every error.
// The below may be to generic. Time will show.
var userErrorRegexp = regexp.MustCompile("argument|flag|shorthand")

func isUserError(err error) bool {
	if cErr, ok := err.(commandError); ok && cErr.isUserError() {
		return true
	}

	return userErrorRegexp.MatchString(err.Error())
}

// Response wraps error for subcommand, and is returned from cmd package.
type Response struct {
	// Err contains error returned from the subcommand executed.
	Err error

	// Cmd contains the command object.
	Cmd *cobra.Command
}

// IsUserError indicates it is a user fault, and should display the command
// usage in addition to displaying the error itself.
func (r Response) IsUserError() bool {
	return r.Err != nil && isUserError(r.Err)
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
		// Do not want to show usage on every error
		SilenceUsage: true,
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
		"download",
		"way to get latest pot file: 'download', 'build', 'no' or filename such as po/git.pot")
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

	return v.cmd
}

func (v rootCommand) Execute(args []string) error {
	return newUserError("run 'git-po-helper -h' for help")
}

func (v *rootCommand) AddCommand(cmds ...*cobra.Command) {
	v.Command().AddCommand(cmds...)
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() Response {
	var (
		resp Response
	)

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
