package util

import (
	"github.com/spf13/viper"
)

// FlagVerbose returns option "--verbose".
func FlagVerbose() int {
	return viper.GetInt("verbose")
}

// FlagVerbose returns option "--quiet".
func FlagQuiet() int {
	return viper.GetInt("quiet")
}

// FlagForce returns option "--force".
func FlagForce() bool {
	return viper.GetBool("check--force") || viper.GetBool("check-commits--force")
}

// FlagGitHubAction returns option "--github-action".
func FlagGitHubAction() bool {
	if viper.GetString("github-action-event") != "" {
		return true
	}
	return viper.GetBool("github-action")
}

// FlagGitHubActionEvent returns option "--github-action-event".
func FlagGitHubActionEvent() string {
	return viper.GetString("github-action-event")
}

// FlagNoGPG returns option "--no-gpg".
func FlagNoGPG() bool {
	return FlagGitHubAction() || viper.GetBool("check--no-gpg") || viper.GetBool("check-commits--no-gpg")
}

// FlagReportTyposAsErrors returns option "--report-typos-as-errors".
func FlagReportTyposAsErrors() bool {
	return viper.GetBool("check-po--report-typos-as-errors") ||
		viper.GetBool("check-commits--report-typos-as-errors") ||
		viper.GetBool("check--report-typos-as-errors")
}

// FlagIgnoreTypos returns option "--ignore-typos".
func FlagIgnoreTypos() bool {
	return viper.GetBool("check-po--ignore-typos") ||
		viper.GetBool("check-commits--ignore-typos") ||
		viper.GetBool("check--ignore-typos")
}

// FlagCore returns option "--core".
func FlagCore() bool {
	return viper.GetBool("check--core") || viper.GetBool("check-po--core")
}

// FlagNoGettext14 returns option "--no-gettext-back-compatible".
func FlagNoGettext14() bool {
	return FlagGitHubAction() || viper.GetBool("no-gettext-back-compatible")
}
