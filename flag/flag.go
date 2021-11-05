package flag

import (
	"github.com/spf13/viper"
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

// ReportTyposAsErrors returns option "--report-typos-as-errors".
func ReportTyposAsErrors() bool {
	return viper.GetBool("check-po--report-typos-as-errors") ||
		viper.GetBool("check-commits--report-typos-as-errors") ||
		viper.GetBool("check--report-typos-as-errors")
}

// IgnoreTypos returns option "--ignore-typos".
func IgnoreTypos() bool {
	return viper.GetBool("check-po--ignore-typos") ||
		viper.GetBool("check-commits--ignore-typos") ||
		viper.GetBool("check--ignore-typos")
}

// Core returns option "--core".
func Core() bool {
	return viper.GetBool("check--core") || viper.GetBool("check-po--core")
}

// NoSpecialGettextVersions returns option "--no-gettext-back-compatible".
func NoSpecialGettextVersions() bool {
	return GitHubActionEvent() != "" || viper.GetBool("no-gettext-back-compatible")
}
