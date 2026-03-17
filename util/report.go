// Package util provides report and message utilities.
package util

import (
	"fmt"
	"os"
	"strings"

	"github.com/mattn/go-isatty"
	log "github.com/sirupsen/logrus"
)

const (
	reportIndent        = "    "
	reportLevelWidth    = 7  // WARNING
	reportPromptWidth   = 10 // [zh_CN.po]
	reportMsgSep        = "  "
	reportBannerRuleLen = 36
)

// Continuation style for multi-line report messages.
const (
	// ReportContinuationPadding: continuation lines use space padding so only the message aligns.
	ReportContinuationPadding = 0
	// ReportContinuationRepeat: continuation lines repeat level and prompt on every line.
	ReportContinuationRepeat = 1
)

// ReportContinuationStyle selects how continuation lines are rendered (default: Padding).
var ReportContinuationStyle = ReportContinuationRepeat

var (
	colorReset  = ""
	colorInfo   = ""
	colorWarn   = ""
	colorError  = ""
	colorPrompt = ""
	colorBold   = ""
)

func refreshReportColors() {
	if isatty.IsTerminal(os.Stderr.Fd()) {
		colorReset = "\033[0m"
		colorInfo = "\033[36m"
		colorWarn = "\033[33m"
		colorError = "\033[31m"
		colorPrompt = "\033[35m"
		colorBold = "\033[1m"
	} else {
		colorReset = ""
		colorInfo = ""
		colorWarn = ""
		colorError = ""
		colorPrompt = ""
		colorBold = ""
	}
}

func levelColor(l log.Level) string {
	switch l {
	case log.InfoLevel:
		return colorInfo
	case log.WarnLevel:
		return colorWarn
	case log.ErrorLevel, log.FatalLevel, log.PanicLevel:
		return colorError
	case SectionLevelNotice:
		return colorInfo // same as info; or use distinct color
	default:
		return colorWarn
	}
}

func levelLabelPadded(l log.Level) string {
	var s string
	switch l {
	case log.InfoLevel:
		s = "INFO"
	case log.WarnLevel:
		s = "WARNING"
	case log.ErrorLevel, log.FatalLevel, log.PanicLevel:
		s = "ERROR"
	default:
		s = "WARNING"
	}
	if len(s) < reportLevelWidth {
		s += strings.Repeat(" ", reportLevelWidth-len(s))
	}
	return s
}

// SectionLevelNotice Section level for banner (in addition to log.Level). Notice is between Info and Warn.
const SectionLevelNotice = log.Level(6)

// Section icon options (pick one per level; current choice in sectionIcon):
//
//	Info:   ℹ (U+2139)  ● ◆ ℹ️ ⓘ
//	Warn:   ⚠ (U+26A0)  ▲ ⚡
//	Error:  ✖ (U+2716)  ✗ ❌ ■
//	Notice: ※ (U+203B)  ◉ 🔔 📢
func sectionIcon(l log.Level) string {
	switch l {
	case log.InfoLevel:
		return "ℹ️"
	case log.WarnLevel:
		return "⚠️"
	case log.ErrorLevel, log.FatalLevel, log.PanicLevel:
		return "❌"
	case SectionLevelNotice:
		return "🔔"
	default:
		return "💡"
	}
}

func formatPromptField(prompt string) string {
	if prompt == "" {
		return strings.Repeat(" ", reportPromptWidth)
	}
	if len(prompt) > reportPromptWidth {
		if reportPromptWidth <= 1 {
			return "…"
		}
		return prompt[:reportPromptWidth-1] + "…"
	}
	return prompt + strings.Repeat(" ", reportPromptWidth-len(prompt))
}

func messageColumnPad() int {
	return reportLevelWidth + len(reportMsgSep) + reportPromptWidth + len(reportMsgSep)
}

// forceFullRow: always print LEVEL and prompt (e.g. blank lines); ignores continuation padding.
func writeReportLine(level log.Level, prompt, line string, firstLine bool, forceFullRow bool) {
	lc := levelColor(level)
	pc := colorPrompt
	rs := colorReset
	levelStr := levelLabelPadded(level)
	promptFmt := formatPromptField(prompt)

	usePadding := !forceFullRow && ReportContinuationStyle == ReportContinuationPadding && !firstLine

	var b strings.Builder
	b.WriteString(reportIndent)
	if usePadding {
		b.WriteString(strings.Repeat(" ", messageColumnPad()))
		b.WriteString(line)
	} else {
		b.WriteString(lc)
		b.WriteString(colorBold)
		b.WriteString(levelStr)
		b.WriteString(rs)
		b.WriteString(" ")
		b.WriteString(pc)
		b.WriteString(promptFmt)
		b.WriteString(rs)
		b.WriteString(reportMsgSep)
		b.WriteString(line)
	}
	fmt.Fprintln(os.Stderr, b.String())
}

// reportSectionStart prints a banner (icon + title) with no indent. Title empty uses a neutral rule.
func reportSectionStart(level log.Level, title string) {
	refreshReportColors()
	icon := sectionIcon(level)
	lc := levelColor(level)
	rs := colorReset
	title = strings.TrimSpace(title)
	if title == "" {
		title = strings.Repeat("·", reportBannerRuleLen)
	}
	fmt.Fprintf(os.Stderr, "%s%s%s%s %s\n", lc, colorBold, icon, rs, title)
}

// ReportSection prints a titled section for one or more message lines (errs variadic, last).
// Order: section title, ok, success level when ok, prompt, lines.
// If ok is false, lines use ERROR; if ok, successLevel is INFO or WARN (else treated as WARN).
// Example: ReportSection("Locale name", false, log.InfoLevel, prompt, err.Error()).
func ReportSection(sectionTitle string, ok bool, successLevel log.Level, prompt string, errs ...string) {
	sl := successLevel
	if sl != log.InfoLevel && sl != log.WarnLevel {
		sl = log.WarnLevel
	}
	level := log.ErrorLevel
	if ok {
		level = sl
	}
	reportResultMessages(sectionTitle, level, prompt, errs, true)
}

func reportResultMessages(sectionTitle string, level log.Level, prompt string, errs []string, withBanner bool) {
	if len(errs) == 0 {
		return
	}
	refreshReportColors()
	if withBanner {
		reportSectionStart(level, sectionTitle)
	}

	firstLine := true
	for _, err := range errs {
		if err == "" {
			writeReportLine(level, prompt, "", true, true)
			firstLine = true
			continue
		}
		lines := strings.Split(err, "\n")
		for _, line := range lines {
			if line == "" {
				writeReportLine(level, prompt, "", true, true)
				firstLine = true
				continue
			}
			writeReportLine(level, prompt, line, firstLine, false)
			firstLine = false
		}
	}
}
