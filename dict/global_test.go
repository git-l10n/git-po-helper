package dict

import (
	"sort"
	"testing"
)

type keepWordsTest struct {
	Input   string
	Matches []string
}

var keepWordsTests = []keepWordsTest{
	{
		Input:   "Cannot $action: You have unstaged changes.",
		Matches: []string{"$action"},
	},
	{
		Input:   "usage: $dashless $USAGE",
		Matches: []string{"$USAGE", "$dashless"},
	},
	{
		Input: "fetch normally indicates which branches had a forced update,\n" +
			"but that check has been disabled; to re-enable, use '--show-forced-updates'\n" +
			"flag or run 'git config fetch.showForcedUpdates true'",
		Matches: []string{"--show-forced-updates", "fetch.showForcedUpdates"},
	},
	{
		Input:   "unable to create lazy_name thread: %s",
		Matches: []string{"lazy_name"},
	},
	{
		Input:   "do not run git-update-server-info",
		Matches: []string{"git-update-server-info"},
	},
	{
		Input:   "git cat-file (-t | -s) [--allow-unknown-type] <object>",
		Matches: []string{"--allow-unknown-type", "git cat-file"},
	},
	{
		Input:   "also apply the patch (use with --stat/--summary/--check)",
		Matches: []string{"--check", "--stat", "--summary"},
	},
	{
		Input:   "--negotiate-only needs one or more --negotiation-tip=*",
		Matches: []string{"--negotiate-only", "--negotiation-tip=*"},
	},
	{
		Input:   "git maintenance run [--auto] [--[no-]quiet] [--task=<task>] [--schedule]",
		Matches: []string{"--[no-]quiet", "--auto", "--schedule", "--task=<task>"},
	},
	{
		Input:   "synonym for --dirstat=files,param1,param2...",
		Matches: []string{"--dirstat=files,param1,param2..."},
	},
	{
		Input:   "synonym for --dirstat=files,<...>,<...>...",
		Matches: []string{"--dirstat=files,<...>,<...>..."},
	},
	{
		Input:   "expected format: %%(color:<color>)",
		Matches: []string{"%%(color:<color>)"},
	},
	{
		Input:   "expected format: %%(align:<width>,<position>)",
		Matches: []string{"%%(align:<width>,<position>)"},
	},
	{
		Input:   "starting with \"refs/\".",
		Matches: []string{"refs/"},
	},
	{
		Input:   "delete refs/remotes/<name>/HEAD",
		Matches: []string{"refs/remotes/<name>/HEAD"},
	},
	{
		Input:   "is a ref in \"refs/{heads,tags}/\"",
		Matches: []string{"refs/{heads,tags}/"},
	},
}

func TestKeepWordsPattern(t *testing.T) {
	for _, tc := range keepWordsTests {
		m := KeepWordsPattern.FindAllStringSubmatch(tc.Input, -1)
		matches := []string{}
		for i := range m {
			matches = append(matches, m[i][0])
		}
		sort.Strings(matches)
		if len(tc.Matches) != len(matches) {
			t.Errorf("Failed to match: different length: %d != %d, expect: %v, actual: %v",
				len(tc.Matches),
				len(matches),
				tc.Matches,
				matches)
		} else {
			for i := range tc.Matches {
				if tc.Matches[i] != matches[i] {
					t.Errorf("Failed to match. expect: %v, actual: %v",
						tc.Matches,
						matches)
					break
				}
			}
		}
	}
}
