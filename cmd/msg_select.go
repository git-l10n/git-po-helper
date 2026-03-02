package cmd

import (
	"bytes"
	"io"
	"os"

	"github.com/git-l10n/git-po-helper/util"
	"github.com/spf13/cobra"
)

type msgSelectCommand struct {
	cmd *cobra.Command
	O   struct {
		Range        string
		NoHeader     bool
		Output       string
		JSON         bool
		Translated   bool
		Untranslated bool
		Fuzzy        bool
		WithObsolete bool
		NoObsolete   bool
		OnlySame     bool
		OnlyObsolete bool
		UnsetFuzzy   bool
		ClearFuzzy   bool
	}
}

func (v *msgSelectCommand) Command() *cobra.Command {
	if v.cmd != nil {
		return v.cmd
	}

	v.cmd = &cobra.Command{
		Use:   "msg-select <po-file>",
		Short: "Extract entries from PO/POT file by index range",
		Long: `Extract entries from a PO/POT file or a gettext JSON file by entry number range.
Input can be either a PO/POT file or a JSON file (same schema as produced by --json).
Use -o <file> to write to a file (avoids stderr mixing when redirecting stdout).
Use --json to output a single JSON object (header_comment, header_meta, entries) instead of PO text.
See docs/design/msg-select-json-output.md for the gettext JSON schema (GettextJSON/GettextEntry in util/gettext_json.go).

Entry 0 is the header entry; it is included when content entries are selected
(use --no-header to omit; for JSON output the file header is always included).
Entry numbers 1, 2, 3, ... refer to the first, second, third content entries.
If no content entries match the range, PO output is empty; JSON output has entries: [].

By default, all entries are selected (translated, same, untranslated, fuzzy, obsolete).
Use --translated, --untranslated, --fuzzy to filter by state (OR relationship).
Use --no-obsolete to exclude obsolete entries; --with-obsolete to include (default).
Use --only-same or --only-obsolete for a single state (mutually exclusive with above).

Range format (--range): comma-separated numbers or ranges, e.g. "3,5,9-13".
Omit --range to select all entries. Range applies to the filtered list.
  - Single numbers: 3, 5 (extract entries 3 and 5)
  - Ranges: 9-13 (extract entries 9 through 13 inclusive)
  - -N: entries 1 through N (e.g. -5 for first 5 entries)
  - N-: entries N through last (e.g. 50- for entries 50 to end)
  - Combined: 3,5,9-13 (extract entries 3, 5, 9, 10, 11, 12, 13)

Examples:
  git-po-helper msg-select --range "1-10" po/zh_CN.po
  git-po-helper msg-select --no-obsolete po/zh_CN.po
  git-po-helper msg-select --only-obsolete po/zh_CN.po
  git-po-helper msg-select --translated --range "-5" -o batch.po po/zh_CN.po`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return v.Execute(args)
		},
	}

	fs := v.cmd.Flags()
	fs.SortFlags = false

	// General options
	fs.StringVar(&v.O.Range, "range", "", "entry range to extract (e.g. 3,5,9-13); omit to select all")
	fs.BoolVar(&v.O.NoHeader, "no-header", false, "omit header entry from output")
	fs.BoolVar(&v.O.JSON, "json", false, "output JSON instead of PO text")
	fs.StringVarP(&v.O.Output, "output", "o", "",
		"write output to file (use - for stdout); empty output overwrites file")
	fs.SetAnnotation("range", "group", []string{"General options"})
	fs.SetAnnotation("no-header", "group", []string{"General options"})
	fs.SetAnnotation("json", "group", []string{"General options"})
	fs.SetAnnotation("output", "group", []string{"General options"})

	// State filter: translated, untranslated, fuzzy (OR when combined)
	fs.BoolVar(&v.O.Translated, "translated", false, "select translated entries (msgstr not empty, not fuzzy)")
	fs.BoolVar(&v.O.Untranslated, "untranslated", false, "select untranslated entries (msgstr empty)")
	fs.BoolVar(&v.O.Fuzzy, "fuzzy", false, "select fuzzy entries")
	fs.SetAnnotation("translated", "group", []string{"State filter"})
	fs.SetAnnotation("untranslated", "group", []string{"State filter"})
	fs.SetAnnotation("fuzzy", "group", []string{"State filter"})

	// Obsolete handling: include or exclude
	fs.BoolVar(&v.O.WithObsolete, "with-obsolete", false, "include obsolete entries (default)")
	fs.BoolVar(&v.O.NoObsolete, "no-obsolete", false, "exclude obsolete entries")
	fs.SetAnnotation("with-obsolete", "group", []string{"Obsolete handling"})
	fs.SetAnnotation("no-obsolete", "group", []string{"Obsolete handling"})

	// Single-state filter: mutually exclusive with state filter above
	fs.BoolVar(&v.O.OnlySame, "only-same", false, "only entries where msgstr equals msgid")
	fs.BoolVar(&v.O.OnlyObsolete, "only-obsolete", false, "only obsolete entries")
	fs.SetAnnotation("only-same", "group", []string{"Single-state filter"})
	fs.SetAnnotation("only-obsolete", "group", []string{"Single-state filter"})

	// Fuzzy handling
	fs.BoolVar(&v.O.UnsetFuzzy, "unset-fuzzy", false,
		"remove fuzzy marker from fuzzy entries in output (keep translations)")
	fs.BoolVar(&v.O.ClearFuzzy, "clear-fuzzy", false,
		"remove fuzzy marker and clear msgstr for fuzzy entries (msgid/msgid_plural preserved)")
	fs.SetAnnotation("unset-fuzzy", "group", []string{"Fuzzy handling"})
	fs.SetAnnotation("clear-fuzzy", "group", []string{"Fuzzy handling"})

	// Custom usage template with grouped flags
	v.cmd.SetUsageTemplate(`Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{flagUsagesByGroup . | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`)

	return v.cmd
}

func (v msgSelectCommand) Execute(args []string) error {
	if len(args) != 1 {
		return NewErrorWithUsage("msg-select requires exactly one argument: <po-file>")
	}
	filter, err := v.buildFilter()
	if err != nil {
		return err
	}

	poFile := args[0]
	var w io.Writer = os.Stdout
	if v.O.Output != "" && v.O.Output != "-" {
		f, err := os.Create(v.O.Output)
		if err != nil {
			return NewStandardErrorF("failed to create output file %s: %v", v.O.Output, err)
		}
		defer f.Close()
		w = f
	}
	if v.O.UnsetFuzzy && v.O.ClearFuzzy {
		return NewErrorWithUsage("--unset-fuzzy and --clear-fuzzy are mutually exclusive")
	}
	// Load → Filter → Save: ReadFileToGettextJSON auto-detects PO vs JSON
	peek, err := os.ReadFile(poFile)
	if err != nil {
		return NewStandardErrorF("failed to read %s: %v", poFile, err)
	}
	if len(peek) > 512 {
		peek = peek[:512]
	}
	trimmed := bytes.TrimLeft(peek, " \t\r\n")
	inputWasPO := len(trimmed) == 0 || trimmed[0] != '{'
	if err := util.MsgSelectFromFile(poFile, v.O.Range, w, v.O.JSON, v.O.NoHeader, inputWasPO,
		v.O.UnsetFuzzy, v.O.ClearFuzzy, filter); err != nil {
		return NewStandardErrorF("%v", err)
	}
	return nil
}

func (v msgSelectCommand) buildFilter() (*util.EntryStateFilter, error) {
	// Mutually exclusive: --only-same and --only-obsolete
	if v.O.OnlySame && v.O.OnlyObsolete {
		return nil, NewErrorWithUsage("--only-same and --only-obsolete are mutually exclusive")
	}
	// --only-same/--only-obsolete are mutually exclusive with --translated, --untranslated, --fuzzy
	if v.O.OnlySame && (v.O.Translated || v.O.Untranslated || v.O.Fuzzy) {
		return nil, NewErrorWithUsage("--only-same is mutually exclusive with --translated, --untranslated, --fuzzy")
	}
	if v.O.OnlyObsolete && (v.O.Translated || v.O.Untranslated || v.O.Fuzzy) {
		return nil, NewErrorWithUsage("--only-obsolete is mutually exclusive with --translated, --untranslated, --fuzzy")
	}
	// Default: include obsolete. --no-obsolete excludes.
	f := util.EntryStateFilter{
		Translated:   v.O.Translated,
		Untranslated: v.O.Untranslated,
		Fuzzy:        v.O.Fuzzy,
		WithObsolete: !v.O.NoObsolete,
		NoObsolete:   v.O.NoObsolete,
		OnlySame:     v.O.OnlySame,
		OnlyObsolete: v.O.OnlyObsolete,
	}
	return &f, nil
}

var msgSelectCmd = msgSelectCommand{}

func init() {
	rootCmd.AddCommand(msgSelectCmd.Command())
}
