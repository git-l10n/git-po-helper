package util

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/git-l10n/git-po-helper/flag"
	"github.com/git-l10n/git-po-helper/repository"
)

func checkPoFilterFormat(poFile string) ([]string, bool) {
	var errs []string
	if flag.ReportFileLocations() == flag.ReportIssueNone || flag.AllowObsoleteEntries() {
		return nil, true
	}

	if !Exist(poFile) {
		errs = append(errs, fmt.Sprintf("cannot open %s: file does not exist", poFile))
		return errs, false
	}

	if !repository.Opened() {
		return []string{"Not in a git repository. Skipping filter attribute check for file locations."}, true
	}

	workDir := repository.WorkDir()
	absPath, err := filepath.Abs(poFile)
	if err != nil {
		errs = append(errs, fmt.Sprintf("cannot resolve path %s: %s", poFile, err))
		return errs, false
	}
	relPath, err := filepath.Rel(workDir, absPath)
	if err != nil || strings.HasPrefix(relPath, "..") {
		// File is outside repo (e.g. temp file from check-commits); skip filter check
		return nil, true
	}
	relPath = filepath.ToSlash(relPath)

	// Query git check-attr filter <path>
	cmd := exec.Command("git", "-C", workDir, "check-attr", "filter", relPath)
	cmd.Stderr = nil
	out, err := cmd.Output()
	if err != nil {
		errs = append(errs, fmt.Sprintf("git check-attr failed for %s: %s", poFile, err))
		return errs, false
	}

	// Parse: "path: filter: value"
	line := strings.TrimSpace(string(out))
	parts := strings.SplitN(line, ": ", 3)
	if len(parts) < 3 {
		errs = append(errs, fmt.Sprintf("unexpected git check-attr output: %s", line))
		return errs, false
	}
	filterValue := strings.TrimSpace(parts[2])

	if filterValue == "unspecified" || filterValue == "unset" || filterValue == "" {
		errs = append(errs,
			"No filter attribute set for XX.po. This will introduce location newlines into the",
			"repository and cause repository bloat.",
			"",
			"Please configure the filter attribute for XX.po, for example:",
			"",
			"    .gitattributes: *.po filter=gettext-no-location",
			"",
			"See:",
			"",
			"    https://lore.kernel.org/git/20220504124121.12683-1-worldhello.net@gmail.com/",
		)
		return errs, flag.ReportFileLocations() == flag.ReportIssueWarn
	}

	// Determine filter clean command: try git config filter.<name>.clean first
	var cmdArgs []string
	cleanOut, err := exec.Command("git", "-C", workDir, "config", "filter."+filterValue+".clean").Output()
	if err != nil || len(bytes.TrimSpace(cleanOut)) == 0 {
		errs = append(errs,
			fmt.Sprintf("File %s has filter %q set, but the filter clean command is not configured.", poFile, filterValue),
			fmt.Sprintf("Run 'git config filter.%s.clean <command>' to set the filter so that location lines in the file can be filtered out.", filterValue),
		)
	} else {
		cmdArgs = strings.Fields(string(bytes.TrimSpace(cleanOut)))
		if len(cmdArgs) == 0 {
			errs = append(errs, fmt.Sprintf("File %s has filter %q set, but the filter clean command is empty.", poFile, filterValue))
		}
	}
	// Determine filter clean command: use known mappings or git config filter.<name>.clean
	if len(cmdArgs) == 0 {
		switch filterValue {
		case "gettext-no-location":
			cmdArgs = []string{"msgcat", "--no-location", "-"}
		case "gettext-no-line-number":
			cmdArgs = []string{"msgcat", "--add-location=file", "-"}
		default:
			errs = append(errs, fmt.Sprintf("File %s has filter %q set, but the filter clean command is empty.", poFile, filterValue))
			return errs, false
		}
	}

	exe, err := exec.LookPath(cmdArgs[0])
	if err != nil {
		errs = append(errs, fmt.Sprintf("%s not found; cannot verify file format (filter %s)", cmdArgs[0], filterValue))
		return errs, false
	}

	original, err := os.ReadFile(poFile)
	if err != nil {
		errs = append(errs, fmt.Sprintf("cannot read %s: %s", poFile, err))
		return errs, false
	}

	cmd = exec.Command(exe, cmdArgs[1:]...)
	cmd.Stdin = bytes.NewReader(original)
	cmd.Stderr = nil
	formatted, err := cmd.Output()
	if err != nil {
		errs = append(errs, fmt.Sprintf("filter %s clean command failed for %s: %s", filterValue, poFile, err))
		return errs, false
	}

	if bytes.Equal(original, formatted) {
		return nil, true
	}

	// Content differs: produce diff
	origTmp, err := os.CreateTemp("", "git-po-helper-orig-*.po")
	if err != nil {
		errs = append(errs, fmt.Sprintf("cannot create temp file: %s", err))
		errs = append(errs, "File content does not match expected format (filter "+filterValue+")")
		return errs, flag.ReportFileLocations() == flag.ReportIssueWarn
	}
	_, _ = origTmp.Write(original)
	_ = origTmp.Close()
	defer os.Remove(origTmp.Name())

	formTmp, err := os.CreateTemp("", "git-po-helper-formatted-*.po")
	if err != nil {
		errs = append(errs, fmt.Sprintf("cannot create temp file: %s", err))
		errs = append(errs, "File content does not match expected format (filter "+filterValue+")")
		return errs, flag.ReportFileLocations() == flag.ReportIssueWarn
	}
	_, _ = formTmp.Write(formatted)
	_ = formTmp.Close()
	defer os.Remove(formTmp.Name())

	diffCmd := exec.Command("diff", "-u", origTmp.Name(), formTmp.Name())
	diffOut, _ := diffCmd.Output()

	filterCmd := strings.Join(cmdArgs, " ")
	errs = append(errs,
		fmt.Sprintf("Filter command for po file: %s", filterCmd),
		"File content differs before and after filtering, please run the",
		"filter on the file and commit again.",
		"",
		"Diff (before vs filtered):",
		"",
	)
	// Indent diff lines for consistency
	for _, line := range strings.Split(strings.TrimSuffix(string(diffOut), "\n"), "\n") {
		errs = append(errs, "    "+line)
	}

	return errs, flag.ReportFileLocations() == flag.ReportIssueWarn
}
