package cmd

import (
	"github.com/git-l10n/git-po-helper/util"
	"github.com/spf13/cobra"
)

func newAgentRunParseLogCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "parse-log [log-file]",
		Short: "Parse agent JSONL log file and display formatted output",
		Long: `Parse a Claude or Qwen/Gemini agent JSONL log file (one JSON object per line).
Auto-detects format and displays with type-specific icons:
- 🤔 thinking content
- 🔧 tool_use content (tool name and input)
- 🤖 text content
- 💬 user/tool_result (raw size)

If no log file is specified, defaults to /tmp/claude.log.jsonl.

Examples:
  git-po-helper agent-run parse-log
  git-po-helper agent-run parse-log /tmp/claude.log.jsonl
  git-po-helper agent-run parse-log /tmp/qwen.log.jsonl`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 1 {
				return NewErrorWithUsage("parse-log expects at most one argument: log-file")
			}
			logFile := "/tmp/claude.log.jsonl"
			if len(args) > 0 {
				logFile = args[0]
			}
			if err := util.CmdAgentRunParseLog(logFile); err != nil {
				return NewStandardErrorF("%v", err)
			}
			return nil
		},
	}

	return cmd
}
