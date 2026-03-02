package cmd

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const groupAnnotationKey = "group"

// flagUsagesByGroup formats local flags by their "group" annotation.
// Flags with the same group are printed under a section header.
// Flags without a group annotation are printed under "Other options".
// Falls back to default FlagUsages if no flags have group annotations.
func flagUsagesByGroup(cmd *cobra.Command) string {
	fs := cmd.LocalFlags()
	if fs == nil || !cmd.HasAvailableLocalFlags() {
		return ""
	}

	// Collect flags by group, preserving first-seen order of groups
	var groupOrder []string
	groups := make(map[string][]*pflag.Flag)
	hasAnyGroup := false

	fs.VisitAll(func(flag *pflag.Flag) {
		if flag.Hidden {
			return
		}
		group := "Other options"
		if flag.Annotations != nil {
			if g, ok := flag.Annotations[groupAnnotationKey]; ok && len(g) > 0 {
				group = g[0]
				hasAnyGroup = true
			}
		}
		// Cobra adds --help after command creation; put it in General options
		if group == "Other options" && flag.Name == "help" {
			group = "General options"
		}
		if _, seen := groups[group]; !seen {
			groupOrder = append(groupOrder, group)
		}
		groups[group] = append(groups[group], flag)
	})

	if !hasAnyGroup {
		return fs.FlagUsages()
	}

	var buf bytes.Buffer
	for _, group := range groupOrder {
		flags := groups[group]
		fmt.Fprintf(&buf, "\n%s:\n", group)
		formatFlags(&buf, flags)
	}
	return strings.TrimPrefix(buf.String(), "\n")
}

func formatFlags(buf *bytes.Buffer, flags []*pflag.Flag) {
	lines := make([]string, 0, len(flags))
	maxlen := 0

	for _, flag := range flags {
		line := ""
		if flag.Shorthand != "" && flag.ShorthandDeprecated == "" {
			line = fmt.Sprintf("  -%s, --%s", flag.Shorthand, flag.Name)
		} else {
			line = fmt.Sprintf("      --%s", flag.Name)
		}

		varname, usage := pflag.UnquoteUsage(flag)
		if varname != "" {
			line += " " + varname
		}
		if flag.NoOptDefVal != "" {
			switch flag.Value.Type() {
			case "string":
				line += fmt.Sprintf("[=\"%s\"]", flag.NoOptDefVal)
			case "bool", "boolfunc":
				if flag.NoOptDefVal != "true" {
					line += fmt.Sprintf("[=%s]", flag.NoOptDefVal)
				}
			case "count":
				if flag.NoOptDefVal != "+1" {
					line += fmt.Sprintf("[=%s]", flag.NoOptDefVal)
				}
			default:
				line += fmt.Sprintf("[=%s]", flag.NoOptDefVal)
			}
		}

		line += "\x00"
		if len(line) > maxlen {
			maxlen = len(line)
		}

		line += usage
		if !isZeroValue(flag) {
			if flag.Value.Type() == "string" {
				line += fmt.Sprintf(" (default %q)", flag.DefValue)
			} else {
				line += fmt.Sprintf(" (default %s)", flag.DefValue)
			}
		}
		if flag.Deprecated != "" {
			line += fmt.Sprintf(" (DEPRECATED: %s)", flag.Deprecated)
		}

		lines = append(lines, line)
	}

	for _, line := range lines {
		sidx := strings.Index(line, "\x00")
		spacing := strings.Repeat(" ", maxlen-sidx)
		fmt.Fprintln(buf, line[:sidx], spacing, line[sidx+1:])
	}
}

func isZeroValue(flag *pflag.Flag) bool {
	switch flag.DefValue {
	case "false", "", "0", "<nil>", "[]":
		return true
	}
	if flag.Value.Type() == "bool" || flag.Value.Type() == "boolfunc" {
		return flag.DefValue == "false" || flag.DefValue == ""
	}
	return false
}

func init() {
	cobra.AddTemplateFunc("flagUsagesByGroup", flagUsagesByGroup)
}
