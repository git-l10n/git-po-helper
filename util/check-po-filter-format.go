package util

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/git-l10n/git-po-helper/flag"
	"github.com/git-l10n/git-po-helper/repository"
)

// checkPoFilterFormat checks git attributes for the filter driver and matches PO #: comments
// to that policy: gettext-no-line-number allows #: lines but refs must be file-only (no line
// numbers; see checkPoLocationCommentsNoLineNumbers); gettext-no-location (and unsupported
// drivers, which fall back to gettext-no-location) require no #: lines (checkPoLocationCommentsAbsent).
// If no filter is set, this function appends the missing-attribute guidance but does not return
// early so the caller's "Location comments (#:)" section can still run.
// It does not run msgcat or compare normalized bytes, and does not read filter.<driver>.clean.
// When repoAttrRelPath is non-empty, it must be a path relative to the repository root
// (e.g. "po/zh_CN.po") used for git check-attr (unless filterAttribute is injected), and for
// user-facing messages; this allows
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
//
// filterAttribute, when non-empty after trimming, is used as the Git filter driver name
// instead of running "git check-attr filter <path>" (for example in tests without
// .gitattributes). When empty, the filter is read from git check-attr as usual.
// Callers that honor --no-check-filter must not invoke this function when that flag is set
// (see check-po.go).
func checkPoFilterFormat(contentPath, repoAttrRelPath, attrSourceCommit, filterAttribute string) ([]string, bool) {
	var errs []string

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

	filterValue := strings.TrimSpace(filterAttribute)
	if filterValue == "" {
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
		filterValue = strings.TrimSpace(parts[2])
	}

	missingFilter := filterValue == "unspecified" || filterValue == "unset" || filterValue == ""
	if missingFilter {
		errs = append(errs,
			"No Git `filter` attribute is set for *.po files on this path.",
			"",
			"The filter attribute describes how Git should normalize #: location comments on each",
			"PO entry when you commit. Those comments change often as source files move; committing",
			"their churn produces noisy diffs and inflates the repository.",
			"",
			"Setting filter=gettext-no-location or filter=gettext-no-line-number in .gitattributes",
			"tells git-po-helper which location style you intend, so it can flag bad #: lines in",
			"the PO (for example references that still include line numbers).",
			"",
			"Please configure the filter for XX.po, for example:",
			"",
			"    .gitattributes: *.po filter=gettext-no-location",
			"",
			"See:",
			"",
			"    https://lore.kernel.org/git/20220504124121.12683-1-worldhello.net@gmail.com/",
		)
	}

	effectiveFilter := filterValue
	if !missingFilter && filterValue != "gettext-no-location" && filterValue != "gettext-no-line-number" {
		if len(errs) > 0 {
			errs = append(errs, "")
		}
		errs = append(errs, fmt.Sprintf(
			"Unsupported filter attribute %q for %s; "+
				`using "gettext-no-location" rules as fallback (PO must not contain #: location comments). `+
				`Prefer filter=gettext-no-location or filter=gettext-no-line-number in .gitattributes.`,
			filterValue, displayPath))
		effectiveFilter = "gettext-no-location"
	}

	poData, err := os.ReadFile(contentPath)
	if err != nil {
		if len(errs) > 0 {
			errs = append(errs, "")
		}
		errs = append(errs, fmt.Sprintf("cannot read %s: %s", displayPath, err))
		return errs, false
	}
	po, err := ParsePoEntries(poData)
	if err != nil {
		if len(errs) > 0 {
			errs = append(errs, "")
		}
		errs = append(errs, fmt.Sprintf("cannot parse %s: %s", displayPath, err))
		return errs, false
	}
	if effectiveFilter == "gettext-no-line-number" {
		locErrs, locOk := checkPoLocationCommentsNoLineNumbers(po)
		if !locOk {
			if len(errs) > 0 {
				errs = append(errs, "")
			}
			errs = append(errs, locErrs...)
		}
	} else {
		locErrs, locOk := checkPoLocationCommentsAbsent(po)
		if !locOk {
			if len(errs) > 0 {
				errs = append(errs, "")
			}
			errs = append(errs, locErrs...)
		}
	}

	hasErrs := len(errs) > 0
	filterOk := !hasErrs
	if hasErrs && flag.ReportFileLocations() == flag.ReportIssueWarn {
		filterOk = true
	}
	return errs, filterOk
}
