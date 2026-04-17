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

// checkPoFilterFormat verifies PO bytes match the repository's clean filter (e.g. msgcat).
// contentPath is the file whose bytes are read and filtered (often a temp checkout path).
// When repoAttrRelPath is non-empty, it must be a path relative to the repository root
// (e.g. "po/zh_CN.po") used only for git check-attr and user-facing messages; this allows
// checking content outside the worktree while still applying .gitattributes for the real path.
// When repoAttrRelPath is empty, the path for attributes is derived from contentPath under the worktree.
//
// attrSourceCommit, when non-empty, must be a revision (commit, tag, etc.) whose tree is used to
// resolve attributes: the command becomes "git check-attr --source=<rev> filter <path>".
// That matches how attributes apply at that revision and is important for bare repositories
// created with partial clone (promisor): .gitattributes may not be present locally until Git
// fetches missing blobs; --source ties attribute lookup to the commit under inspection so Git
// can materialize .gitattributes from the object store / remote as needed. When empty,
// attributes are resolved the usual way (working tree / index), which is appropriate for
// in-worktree checks (e.g. check-po on local files).
func checkPoFilterFormat(contentPath, repoAttrRelPath, attrSourceCommit string) ([]string, bool) {
	var errs []string
	if flag.NoCheckFilter() {
		return nil, true
	}
	if flag.ReportFileLocations() == flag.ReportIssueNone || flag.AllowObsoleteEntries() {
		return nil, true
	}

	if !Exist(contentPath) {
		errs = append(errs, fmt.Sprintf("cannot open %s: file does not exist", contentPath))
		return errs, false
	}

	if !repository.Opened() {
		return []string{"Not in a git repository. Skipping filter attribute check for file locations."}, true
	}

	workDir := repository.WorkDir()
	displayPath := contentPath
	if repoAttrRelPath != "" {
		displayPath = repoAttrRelPath
	}

	var relPath string
	if repoAttrRelPath != "" {
		if filepath.IsAbs(repoAttrRelPath) {
			errs = append(errs, fmt.Sprintf("filter attr path must be relative to repository: %s", repoAttrRelPath))
			return errs, false
		}
		clean := filepath.Clean(filepath.FromSlash(filepath.ToSlash(repoAttrRelPath)))
		joined := filepath.Join(workDir, clean)
		absJoined, err := filepath.Abs(joined)
		if err != nil {
			errs = append(errs, fmt.Sprintf("cannot resolve filter attr path %q: %s", repoAttrRelPath, err))
			return errs, false
		}
		absWork, err := filepath.Abs(workDir)
		if err != nil {
			errs = append(errs, fmt.Sprintf("cannot resolve work tree: %s", err))
			return errs, false
		}
		rel, err := filepath.Rel(absWork, absJoined)
		if err != nil || strings.HasPrefix(rel, "..") {
			errs = append(errs, fmt.Sprintf("filter attr path escapes repository: %s", repoAttrRelPath))
			return errs, false
		}
		relPath = filepath.ToSlash(rel)
	} else {
		absPath, err := filepath.Abs(contentPath)
		if err != nil {
			errs = append(errs, fmt.Sprintf("cannot resolve path %s: %s", contentPath, err))
			return errs, false
		}
		relPath, err = filepath.Rel(workDir, absPath)
		if err != nil || strings.HasPrefix(relPath, "..") {
			// File is outside repo and no logical path given; skip filter check
			return nil, true
		}
		relPath = filepath.ToSlash(relPath)
	}

	// Query git check-attr [--source=<rev>] filter <path>
	checkAttrArgs := []string{"-C", workDir, "check-attr"}
	if attrSourceCommit != "" {
		checkAttrArgs = append(checkAttrArgs, "--source="+attrSourceCommit)
	}
	checkAttrArgs = append(checkAttrArgs, "filter", relPath)
	cmd := exec.Command("git", checkAttrArgs...)
	cmd.Stderr = nil
	out, err := cmd.Output()
	if err != nil {
		errs = append(errs, fmt.Sprintf("git check-attr failed for %s: %s", displayPath, err))
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
			fmt.Sprintf("File %s has filter %q set, but the filter clean command is not configured.", displayPath, filterValue),
			fmt.Sprintf("Run 'git config filter.%s.clean <command>' to set the filter so that location lines in the file can be filtered out.", filterValue),
		)
	} else {
		cmdArgs = strings.Fields(string(bytes.TrimSpace(cleanOut)))
		if len(cmdArgs) == 0 {
			errs = append(errs, fmt.Sprintf("File %s has filter %q set, but the filter clean command is empty.", displayPath, filterValue))
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
			errs = append(errs, fmt.Sprintf("File %s has filter %q set, but the filter clean command is empty.", displayPath, filterValue))
			return errs, false
		}
	}

	exe, err := exec.LookPath(cmdArgs[0])
	if err != nil {
		errs = append(errs, fmt.Sprintf("%s not found; cannot verify file format (filter %s)", cmdArgs[0], filterValue))
		return errs, false
	}

	original, err := os.ReadFile(contentPath)
	if err != nil {
		errs = append(errs, fmt.Sprintf("cannot read %s: %s", contentPath, err))
		return errs, false
	}

	cmd = exec.Command(exe, cmdArgs[1:]...)
	cmd.Stdin = bytes.NewReader(original)
	cmd.Stderr = nil
	formatted, err := cmd.Output()
	if err != nil {
		errs = append(errs, fmt.Sprintf("filter %s clean command failed for %s: %s", filterValue, displayPath, err))
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
		"PO file does not match expected filter output.",
		"",
		"This repository uses a Git filter driver to automatically strip location",
		fmt.Sprintf("comments from PO files on commit (filter: %s).", filterCmd),
		"The file being committed still contains location comments, which will",
		"cause the file to appear modified for other users who have the filter",
		"driver configured.",
		"",
		"Please do one of the following:",
		"  - Set up the Git filter driver as described in po/README.md, or",
		"  - Run the filter manually before committing:",
		fmt.Sprintf("      %s <XX.po >tmp.po && mv tmp.po XX.po", filterCmd),
		"",
		"Diff (before vs filtered):",
		"",
	)
	const maxDiffLines = 10
	diffLines := strings.Split(strings.TrimSuffix(string(diffOut), "\n"), "\n")
	for i, line := range diffLines {
		if i >= maxDiffLines {
			errs = append(errs, "    ... ...")
			break
		}
		errs = append(errs, "    "+line)
	}

	return errs, flag.ReportFileLocations() == flag.ReportIssueWarn
}
