// Package dict provides dictionary to fix typos.
package dict

import "regexp"

// KeepWordsPattern defines words we want to keep for check.
var KeepWordsPattern = regexp.MustCompile(`(` +
	`\${[a-zA-Z0-9_]+}` + // match shell variables: ${n}, ...
	`|` +
	`\$[a-zA-Z0-9_]+` + // match shell variables: $PATH, ...
	`|` +
	`\b[a-zA-Z.]+\.[a-zA-Z]+\b` + // match git config variables: color.ui, ...
	`|` +
	`\b[a-zA-Z0-9_]+_[a-zA-Z0-9]+\b` + // match variable names: var_name, ...
	`|` +
	`\bgit-[a-z-]+` + // match git commands: git-log, ...
	`|` +
	`\bgit [a-z]+-[a-z-]+` + // match git commands: git bisect--helper, ...
	`|` +
	`\b[a-z-]+--[a-z-]+` + // match helper commands: bisect--helper, ...
	`|` +
	`--[a-zA-Z0-9-=]+` + // match git options: --option, --option=value, ...
	`|` +
	`%%\(.*?\)` + // match %(fieldname) in format argument of git-for-each-ref, ...
	`|` +
	`\brefs/[a-zA-Z0-9{}<>_.,/-]+` + // match refspec like: refs/remotes/<name>/HEAD, refs/{heads,tags}/...
	`)`)

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
