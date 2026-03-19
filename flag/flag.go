// Package flag provides viper flags.
package flag

import (
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
// When not set, also checks GITHUB_ACTIONS env (set by GitHub Actions to "true"),
// so --pot-file defaults to "download" when running in CI without explicit flag.
func GitHubActionEvent() string {
	if v := viper.GetString("github-action-event"); v != "" {
		return v
	}
	return ""
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

// AllowObsoleteEntries returns true when obsolete entries should be allowed
// (e.g. after msgmerge in update flow, which creates obsolete entries by design).
func AllowObsoleteEntries() bool {
	return viper.GetBool("check--allow-obsolete")
}

// ReportFileLocations returns way to display typos (none, warn, error).
func ReportFileLocations() int {
	var value = ""

	if v := viper.GetString("check--report-file-locations"); v != "" {
		value = v
	} else if v := viper.GetString("check-po--report-file-locations"); v != "" {
		value = v
	} else if v := viper.GetString("check-commits--report-file-locations"); v != "" {
		value = v
	}
	if value == "" && GitHubActionEvent() != "" {
		return ReportIssueError
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

// IsPotFileSet returns true when --pot-file was explicitly set by user.
func IsPotFileSet() bool {
	return viper.IsSet("pot-file")
}

// GetPotFileRaw returns the raw --pot-file value.
func GetPotFileRaw() string {
	return viper.GetString("pot-file")
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

// GetConfigFilePath returns option "--config" (custom agent config file path).
// If non-empty, agent config is loaded only from this file.
func GetConfigFilePath() string {
	return viper.GetString("config")
}
