// Package dict provides dictionary to fix typos.
package dict

import (
	"regexp"
	"strings"
)

// KeepGitConfigVariableMinLetters is the minimum letter count (a-zA-Z) for a
// KeepWordsPattern match that looks like a git config variable (only letters
// and '.', and containing '.'). Shorter tokens such as "p.e" (Spanish/Portuguese
// "por ejemplo") are ignored to avoid false-positive typo reports.
const KeepGitConfigVariableMinLetters = 6

// KeepWordsPattern defines words we want to keep for check.
var KeepWordsPattern = regexp.MustCompile(`(` +
	`\${[a-zA-Z0-9_]+}` + // match shell variables: ${n}, ...
	`|` +
	`\$[a-zA-Z0-9_]+` + // match shell variables: $PATH, ...
	`|` +
	`\b[a-zA-Z.]+\.[a-zA-Z]+\b` + // match git config variables: color.ui, ... (short ones filtered by IsShortGitConfigLike)
	`|` +
	`\b[a-zA-Z0-9_]+_[a-zA-Z0-9]+\b` + // match variable names: var_name, ...
	`|` +
	`\bgit-[a-z-]+` + // match git commands: git-log, ...
	`|` +
	`\bgit [a-z]+-[a-z-]+` + // match git commands: git bisect--helper, ...
	`|` +
	`\b[a-z-]+--[a-z-]+` + // match helper commands: bisect--helper, ...
	`|` +
	// match git options: --option, --option=param1,param2 ..., --[no-]signed
	`--(\[[a-z-]+\])?[a-zA-Z0-9-]+(=[a-zA-Z0-9,<\.>*]+)?` +
	`|` +
	`%%\(.*?\)` + // match %(fieldname) in format argument of git-for-each-ref, ...
	`|` +
	`\brefs/[a-zA-Z0-9{}<>_.,/-]*` + // match refspec like: refs/remotes/<name>/HEAD, refs/{heads,tags}/...
	`)`)

// IsShortGitConfigLike reports whether s looks like a git config variable
// token (letters and '.' only, containing at least one '.') but has fewer than
// KeepGitConfigVariableMinLetters letters. Such tokens are not treated as
// keep-words for typo checking.
func IsShortGitConfigLike(s string) bool {
	if !strings.Contains(s, ".") {
		return false
	}
	letters := 0
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z':
			letters++
		case r == '.':
			// allowed in config-like tokens
		default:
			return false
		}
	}
	return letters < KeepGitConfigVariableMinLetters
}

// GlobalSkipPatterns defines words we want to ignore for check globally.
var GlobalSkipPatterns = []struct {
	Pattern *regexp.Regexp
	Replace string
}{
	{
		Pattern: regexp.MustCompile(`\b(` +
			"git-directories" +
			`|` +
			`e\.g\.?` +
			`|` +
			`i\.e\.?` +
			`)\b`),
		Replace: "...",
	},
	{
		// <variable_name>
		Pattern: regexp.MustCompile(`<[^>]+>`),
		Replace: "<...>",
	},
	{
		// [variable_name]
		Pattern: regexp.MustCompile(`\[[^]]+\]`),
		Replace: "[...]",
	},
	{
		// Complex placeholders in fprintf, such as: %2$.*1$s
		Pattern: regexp.MustCompile(`%[0-9]+\$\.\*[0-9]+\$`),
		Replace: "%.*",
	},
	{
		// Simple placeholders in fprintf, such as: %2$s, %3$d, %1$0.1f
		Pattern: regexp.MustCompile(`%[0-9]+\$`),
		Replace: "%",
	},
}
