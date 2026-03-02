// Package flag provides viper flags.
package flag

import (
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const (
	ReportIssueNone = iota
	ReportIssueWarn
	ReportIssueError
)

// Verbose returns option "--verbose".
func Verbose() int {
	return viper.GetInt("verbose")
}

// Quiet returns option "--quiet".
func Quiet() int {
	return viper.GetInt("quiet")
}

// Force returns option "--force".
func Force() bool {
	return viper.GetBool("check--force") || viper.GetBool("check-commits--force")
}

// GitHubActionEvent returns option "--github-action-event".
func GitHubActionEvent() string {
	return viper.GetString("github-action-event")
}

// NoGPG returns option "--no-gpg".
func NoGPG() bool {
	return GitHubActionEvent() != "" || viper.GetBool("check--no-gpg") || viper.GetBool("check-commits--no-gpg")
}

// ReportTypos returns way to display typos (none, warn, error).
func ReportTypos() int {
	var value = ""

	if GitHubActionEvent() != "" {
		return ReportIssueWarn
	}
	if v := viper.GetString("check--report-typos"); v != "" {
		value = v
	} else if v := viper.GetString("check-po--report-typos"); v != "" {
		value = v
	} else if v := viper.GetString("check-commits--report-typos"); v != "" {
		value = v
	}
	switch value {
	case "none":
		return ReportIssueNone
	case "warn":
		return ReportIssueWarn
	case "error":
		fallthrough
	default:
		return ReportIssueError
	}
}

// ReportFileLocations returns way to display typos (none, warn, error).
func ReportFileLocations() int {
	var value = ""

	if GitHubActionEvent() != "" {
		return ReportIssueError
	}
	if v := viper.GetString("check--report-file-locations"); v != "" {
		value = v
	} else if v := viper.GetString("check-po--report-file-locations"); v != "" {
		value = v
	} else if v := viper.GetString("check-commits--report-file-locations"); v != "" {
		value = v
	}
	switch value {
	case "none":
		return ReportIssueNone
	case "warn":
		return ReportIssueWarn
	case "error":
		fallthrough
	default:
		return ReportIssueError
	}
}

const (
	PotFileFlagNone = iota
	PotFileFlagLocation
	PotFileFlagUpdate
	PotFileFlagDownload
)

// getPotFileOpt returns the --pot-file value (defined on root command).
func getPotFileOpt() string {
	if viper.IsSet("pot-file") {
		return viper.GetString("pot-file")
	}
	return "auto"
}

// GetPotFileLocation returns option "--pot-file".
func GetPotFileLocation() string {
	value := getPotFileOpt()

	switch GetPotFileFlag() {
	case PotFileFlagNone:
		log.Fatalf("unknown location for opt: %s", value)
	case PotFileFlagUpdate:
		value = "po/git.pot"
	case PotFileFlagLocation:
		fallthrough
	default:
	}
	return value
}

// GetPotFileFlag returns option "--pot-file".
func GetPotFileFlag() int {
	var (
		ret int
		opt = strings.ToLower(getPotFileOpt())
	)

	if opt == "" {
		opt = "auto"
	}

	// Handle "auto" value
	if opt == "auto" {
		if GitHubActionEvent() != "" {
			opt = "download"
		} else {
			opt = "build"
		}
	}

	switch opt {
	case "no", "false":
		ret = PotFileFlagNone
	case "build", "make", "update":
		ret = PotFileFlagUpdate
	case "download":
		ret = PotFileFlagDownload
	default:
		if strings.Contains(opt, "/") {
			ret = PotFileFlagLocation
		} else {
			log.Fatalf("unknown value for --pot-file: %s", opt)
		}
	}
	return ret
}

// Core returns option "--core".
func Core() bool {
	return viper.GetBool("check--core") || viper.GetBool("check-po--core")
}

// NoSpecialGettextVersions returns option "--no-special-gettext-versions".
func NoSpecialGettextVersions() bool {
	return viper.GetBool("no-special-gettext-versions")
}

// SetGettextUseMultipleVersions sets option "gettext-use-multiple-versions".
func SetGettextUseMultipleVersions(value bool) {
	viper.Set("gettext-use-multiple-versions", value)
}

// GettextUseMultipleVersions returns option "gettext-use-multiple-versions".
func GettextUseMultipleVersions() bool {
	return viper.GetBool("gettext-use-multiple-versions")
}

// AgentConfigFile returns option "--config" (custom agent config file path).
// If non-empty, agent config is loaded only from this file.
func AgentConfigFile() string {
	return viper.GetString("config")
}
