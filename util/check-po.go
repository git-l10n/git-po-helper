package util

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/git-l10n/git-po-helper/flag"
	log "github.com/sirupsen/logrus"
)

// checkPoMetaEscapeChars reads meta line by line and reports abnormal newline sequences.
// Abnormal: literal "\\n" (backslash+n) in decoded meta, which may indicate double-escape or corruption.
func checkPoMetaEscapeChars(po *GettextPO) ([]string, bool) {
	var errs []string
	for i, line := range po.Meta() {
		if strings.Contains(line, `\n`) {
			errs = append(errs, fmt.Sprintf("header meta line %d contains literal \\\\n (abnormal; use real newline or proper PO escape): %q", i+1, line))
		}
	}
	return errs, len(errs) == 0
}

// locationLineNumPattern matches a reference that contains a line number (e.g. file.c:116 or file.c:116,5).
var locationLineNumPattern = regexp.MustCompile(`:\d+`)

// entryDescWithLine returns a short description for an entry (index, msgid, optional line from parsing).
// Format: "entry <N>@L<line> (msgid %q)" when line > 0, else "entry <N> (msgid %q)" so the line number is easy to spot and grep.
func entryDescWithLine(entryIndex int, msgid string, startLineNo int) string {
	if startLineNo > 0 {
		return fmt.Sprintf("entry %d@L%d (msgid %q)", entryIndex, startLineNo, msgid)
	}
	return fmt.Sprintf("entry %d (msgid %q)", entryIndex, msgid)
}

// checkPoLocationCommentsNoLineNumbers scans each entry's comments for location comments (#:)
// and reports any that contain line numbers. Per Git l10n convention, location comments should
// not include line numbers (use --add-location=file or --no-location).
func checkPoLocationCommentsNoLineNumbers(po *GettextPO) ([]string, bool) {
	var errs []string

	for i, e := range po.Entries {
		msgid := e.MsgID
		if len(msgid) > 30 {
			msgid = msgid[:27] + "..."
		}
		entryDesc := entryDescWithLine(i+1, msgid, e.EntryLocation)
		for _, c := range e.Comments {
			trimmed := strings.TrimSpace(c)
			if !strings.HasPrefix(trimmed, "#:") {
				continue
			}
			content := strings.TrimPrefix(trimmed, "#:")
			content = strings.TrimSpace(content)
			for _, ref := range strings.Fields(content) {
				if locationLineNumPattern.MatchString(ref) {
					errs = append(errs, fmt.Sprintf("%s: location comment contains line number (use file-only or remove): %q", entryDesc, ref))
					return errs, false
				}
			}
		}
	}

	return errs, true
}

// gettextVersionAtLeast returns true if minVersion >= required (e.g. "0.15" >= "0.15").
// Invalid or empty minVersion is treated as "0" and thus not at least any required.
func gettextVersionAtLeast(minVersion, required string) bool {
	maj, min := parseGettextVersion(minVersion)
	rMaj, rMin := parseGettextVersion(required)
	if maj < rMaj {
		return false
	}
	if maj > rMaj {
		return true
	}
	return min >= rMin
}

func parseGettextVersion(s string) (major, minor int) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, 0
	}
	parts := strings.SplitN(s, ".", 2)
	major, _ = strconv.Atoi(parts[0])
	if len(parts) > 1 {
		minor, _ = strconv.Atoi(parts[1])
	}
	return major, minor
}

// checkPoCompatibility reports gettext version compatibility issues based on minGettextVersion:
// - 0.15+: msgctxt, previous-msgctxt (#| msgctxt / #| msgid), obsolete msgctxt (#~ msgctxt)
// - 0.16+: obsolete-previous-msgid (#~| msgid), obsolete-previous-msgctxt (#~| msgctxt), obsolete-previous-plural (#~| msgid_plural)
// If minGettextVersion is empty, returns no errors (caller skips the check).
func checkPoCompatibility(po *GettextPO, minGettextVersion string) ([]string, bool) {
	if minGettextVersion == "" {
		return nil, true
	}
	need015 := gettextVersionAtLeast(minGettextVersion, "0.15")
	need016 := gettextVersionAtLeast(minGettextVersion, "0.16")

	var errs []string
	for i, e := range po.Entries {
		msgid := e.MsgID
		if len(msgid) > 30 {
			msgid = msgid[:27] + "..."
		}
		entryDesc := entryDescWithLine(i+1, msgid, e.EntryLocation)

		if !need015 {
			if e.MsgCtxt != nil {
				if e.Obsolete {
					errs = append(errs, fmt.Sprintf("%s: #~ msgctxt (obsolete with context) not supported by gettext below 0.15", entryDesc))
				} else {
					errs = append(errs, fmt.Sprintf("%s: msgctxt not supported by gettext below 0.15", entryDesc))
				}
				continue
			}
			if !e.IsObsolete() && e.HasPreviousMsgctxt() {
				errs = append(errs, fmt.Sprintf("%s: previous msgctxt (#|) not supported by gettext below 0.15", entryDesc))
				continue
			}
		}

		if !need016 {
			if e.IsObsolete() && e.HasPreviousMsgid() {
				errs = append(errs, fmt.Sprintf("%s: #~| msgid (obsolete previous) not supported by gettext below 0.16", entryDesc))
				continue
			}
			if e.IsObsolete() && e.HasPreviousMsgctxt() {
				errs = append(errs, fmt.Sprintf("%s: #~| msgctxt (obsolete previous) not supported by gettext below 0.16", entryDesc))
				continue
			}
			if e.IsObsolete() && e.HasPreviousMsgidPlural() {
				errs = append(errs, fmt.Sprintf("%s: #~| msgid_plural (obsolete previous) not supported by gettext below 0.16", entryDesc))
				continue
			}
		}
	}
	if len(errs) > 0 {
		return errs, false
	}
	return nil, true
}

// checkPoNoObsoleteEntries reports error if any entry has Obsolete=true.
func checkPoNoObsoleteEntries(po *GettextPO) ([]string, bool) {
	var count int
	for _, e := range po.Entries {
		if e.Obsolete {
			count++
		}
	}
	if count > 0 {
		return []string{fmt.Sprintf("you have %d obsolete entries, please remove them", count)}, false
	}
	return nil, true
}

// CheckPoFile checks syntax of "po/xx.po".
// When compareWithPot is true, also checks incomplete translations against the POT template.
func CheckPoFile(locale, poFile string, compareWithPot bool) bool {
	return CheckPoFileWithPrompt(locale, poFile, compareWithPot, "")
}

// CheckPoFileWithPrompt checks syntax of "po/xx.po", and use specific prompt.
// When compareWithPot is true, also checks incomplete translations against the POT template
// (subject to --pot-file; use "no" to skip acquisition inside CheckWithPoFile).
func CheckPoFileWithPrompt(locale, poFile string, compareWithPot bool, prompt string) bool {
	var (
		ret  = true
		ok   bool
		errs []string
	)

	if prompt == "" {
		prompt = fmt.Sprintf("[%s]", locale+".po")
	}

	if !Exist(poFile) {
		log.Errorf(`%s\tfail to check "%s", does not exist`, prompt, poFile)
		return false
	}

	// Run msgfmt to check syntax of a .po file
	errs, ok = checkPoWithMsgfmt(poFile)
	ReportSection("Syntax check with msgfmt", ok, log.InfoLevel, prompt, errs...)
	ret = ret && ok

	// Get pretty locale name, and validate locale name.
	locale = strings.TrimSuffix(filepath.Base(locale), ".po")
	localeErrs := ValidateLocale(locale)
	if len(localeErrs) > 0 {
		msgs := make([]string, 0, len(localeErrs))
		for _, e := range localeErrs {
			msgs = append(msgs, e.Error())
		}
		ReportSection("Locale name", false, log.InfoLevel, prompt, msgs...)
		ret = false
	}

	poData, err := os.ReadFile(poFile)
	if err != nil {
		log.Errorf(`%s\tfail to read %q: %v`, prompt, poFile, err)
		return false
	}
	po, err := ParsePoEntries(poData)
	if err != nil {
		log.Errorf(`%s\tfail to parse %q: %v`, prompt, poFile, err)
		return false
	}

	// Check header meta for abnormal newline sequences (e.g. literal \n).
	errs, ok = checkPoMetaEscapeChars(po)
	ReportSection("Syntax of PO header meta lines", ok, log.InfoLevel, prompt, errs...)
	ret = ret && ok

	// Check that location comments (#:) do not contain line numbers.
	if flag.ReportFileLocations() != flag.ReportIssueNone {
		errs, ok = checkPoLocationCommentsNoLineNumbers(po)
		ok = ok || flag.ReportFileLocations() == flag.ReportIssueWarn
		ReportSection("Location comments (#:)", ok, log.InfoLevel, prompt, errs...)
		ret = ret && ok
	}

	// Compatibility checks (only when project sets MinGettextVersion): 0.15+ msgctxt/#|/#~ msgctxt, 0.16+ #~|.
	projectName := po.GetProject()
	cfg := GetProjectPotConfig(projectName, poFile)
	if cfg.MinGettextVersion != "" {
		errs, ok = checkPoCompatibility(po, cfg.MinGettextVersion)
		ReportSection("gettext compatibility", ok, log.InfoLevel, prompt, errs...)
		ret = ret && ok
	}

	// No obsolete entries allowed (unless AllowObsoleteEntries, e.g. in update flow).
	if !flag.AllowObsoleteEntries() {
		errs, ok = checkPoNoObsoleteEntries(po)
		ReportSection("Obsolete #~ entries", ok, log.InfoLevel, prompt, errs...)
		ret = ret && ok
	}

	// Format check: use driver return from git-check-attr to format PO file
	errs, ok = checkPoFilterFormat(poFile)
	ReportSection("PO filter (.gitattributes)", ok, log.InfoLevel, prompt, errs...)
	ret = ret && ok

	// Check possible typos in a .po file.
	errs, ok = checkTyposInPo(locale, po)
	ReportSection("msgid/msgstr pattern check", ok, log.WarnLevel, prompt, errs...)
	ret = ret && ok

	// Check that Project-Id-Version defines a project name.
	if projectName == "" {
		ReportSection("Project name", false, log.InfoLevel, prompt,
			"project name is not defined in PO file")
		ret = false
	}

	// Check incomplete translations against POT (can be disabled with "--pot-file=no" inside CheckWithPoFile).
	if compareWithPot {
		if !CheckWithPotFile("HEAD", projectName, poFile) {
			ret = false
		}
	}

	return ret
}

// CmdCheckPo implements check-po sub command.
// Args must be non-empty. Each arg is either a directory (scan for *.po in it, no recursion)
// or a file (.po or .pot). All found/listed .po files are checked; .pot files are checked
// for CamelCase config variables when Project-Id-Version indicates Git.
func CmdCheckPo(args ...string) bool {
	ret := true

	if len(args) == 0 {
		log.Errorf("no arguments given; specify .po/.pot files or directories containing them")
		return false
	}

	type checkItem struct{ locale, poFile string }
	var toCheck []checkItem
	var potFilesToCheck []string

	for _, arg := range args {
		info, err := os.Stat(arg)
		if err != nil {
			log.Errorf("cannot access %q: %v", arg, err)
			ret = false
			continue
		}
		if info.IsDir() {
			entries, err := os.ReadDir(arg)
			if err != nil {
				log.Errorf("cannot read directory %q: %v", arg, err)
				ret = false
				continue
			}
			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				if filepath.Ext(e.Name()) != ".po" {
					continue
				}
				poFile := filepath.Join(arg, e.Name())
				locale := strings.TrimSuffix(e.Name(), ".po")
				toCheck = append(toCheck, checkItem{locale: locale, poFile: poFile})
			}
			continue
		}
		// file
		ext := filepath.Ext(arg)
		if ext == ".pot" {
			potFilesToCheck = append(potFilesToCheck, arg)
			continue
		}
		if ext != ".po" {
			log.Errorf("not a .po or .pot file: %q", arg)
			ret = false
			continue
		}
		poFile := arg
		locale := strings.TrimSuffix(filepath.Base(arg), ".po")
		toCheck = append(toCheck, checkItem{locale: locale, poFile: poFile})
	}

	if len(toCheck) == 0 && len(potFilesToCheck) == 0 {
		log.Errorf("no .po or .pot files to check (specify .po/.pot files or directories containing them)")
		return false
	}

	for _, item := range toCheck {
		if !CheckPoFile(item.locale, item.poFile, true) {
			ret = false
		}
		if flag.Core() {
			if !CheckCorePoFile(item.locale, item.poFile) {
				ret = false
			}
		}
	}

	for _, potFile := range potFilesToCheck {
		if err := CheckGitPotFile(potFile); err != nil {
			log.Errorf("%v", err)
			ret = false
		}
	}
	return ret
}
