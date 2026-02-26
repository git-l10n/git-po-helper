package util

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/git-l10n/git-po-helper/flag"
	"github.com/git-l10n/git-po-helper/repository"
	"github.com/mattn/go-isatty"
	log "github.com/sirupsen/logrus"
)

const (
	defaultMaxCommits     = 100
	subjectWidthHardLimit = 72
	bodyWidthHardLimit    = 72
	commitSubjectPrefix   = "l10n:"
	sobPrefix             = "Signed-off-by:"
	defaultEncoding       = "utf-8"
)

func getCommitChanges(commit string) ([]string, bool) {
	var changes []string

	cmd := exec.Command("git",
		"diff-tree",
		"-r",
		"-z",
		"--no-renames",
		"--diff-filter=ACM",
		"--name-only",
		commit)
	stdout, err := cmd.StdoutPipe()
	if err == nil {
		err = cmd.Start()
	}
	if err != nil {
		log.Errorf("commit %s: fail to run git-diff-tree: %s", AbbrevCommit(commit), err)
		return nil, false
	}
	buffer, err := io.ReadAll(stdout)
	if err != nil {
		return nil, false
	}
	for i, file := range strings.Split(string(buffer), "\000") {
		if i == 0 || file == "" {
			continue
		}
		changes = append(changes, file)
	}
	if err = cmd.Wait(); err != nil {
		log.Errorf("commit %s: fail to run git-diff-tree: %s", AbbrevCommit(commit), err)
		return nil, false
	}
	return changes, true
}

func checkCommitChanges(commit string, notL10nChanges, l10nChanges []string) (ok, brk bool) {
	var (
		errs  []string
		warns []string
	)

	// commit is OK, if no error is found.
	ok = true
	// If brk is true, will stop parsing other commits.
	brk = false

	defer func() {
		if len(warns) > 0 {
			reportResultMessages(warns, "", log.WarnLevel)
		}
		if len(errs) > 0 {
			ok = false
			reportResultMessages(errs, "", log.ErrorLevel)
		}
	}()

	if len(notL10nChanges) > 0 {
		msg := bytes.NewBuffer(nil)
		msg.WriteString(fmt.Sprintf("commit %s: found changes beyond \"%s/\" directory:\n",
			AbbrevCommit(commit), PoDir))
		for _, change := range notL10nChanges {
			msg.WriteString("\t\t")
			msg.WriteString(change)
			msg.WriteString("\n")
		}
		if len(l10nChanges) == 0 && flag.GitHubActionEvent() != "" {
			brk = true // not l10n commit, and stop parsing other commits.
			switch flag.GitHubActionEvent() {
			case "push":
				warns = append(warns,
					msg.String(),
					fmt.Sprintf(`commit %s: break because this commit is not for git-l10n`,
						AbbrevCommit(commit)))
			case "pull_request":
				fallthrough
			case "pull_request_target":
				fallthrough
			default:
				errs = append(errs,
					msg.String(),
					fmt.Sprintf(`commit %s: break because this commit is not for git-l10n`,
						AbbrevCommit(commit)))
			}
			return
		}
		errs = append(errs, msg.String())
	}

	for _, fileName := range l10nChanges {
		tmpFile := FileRevision{
			Revision: commit,
			File:     fileName,
		}
		if err := CheckoutTmpfile(&tmpFile); err != nil || tmpFile.Tmpfile == "" {
			errs = append(errs,
				fmt.Sprintf("commit %s: fail to checkout %s of revision %s: %s",
					AbbrevCommit(commit), tmpFile.File, tmpFile.Revision, err))
			continue
		}
		if fileName == "po/TEAMS" {
			if _, errors := ParseTeams(tmpFile.Tmpfile); len(errors) > 0 {
				for _, error := range errors {
					errs = append(errs,
						fmt.Sprintf("commit %s: %s",
							AbbrevCommit(commit), error))
				}
			}
		} else {
			locale := strings.TrimSuffix(filepath.Base(fileName), ".po")
			prompt := fmt.Sprintf("[%s@%s]",
				filepath.Join(PoDir, locale+".po"),
				AbbrevCommit(commit))
			if !CheckPoFileWithPrompt(locale, tmpFile.Tmpfile, prompt) {
				// Error errs in CheckPoFileWithPrompt() have been output already,
				// mark ok as false
				ok = false
			}
		}
		os.Remove(tmpFile.Tmpfile)
	}
	return
}

func fetchBlobsInPartialClone(args []string) error {
	var (
		maxCommits int
		blobList   []string
		cmd        *exec.Cmd
		scanner    *bufio.Scanner
		out        []byte
	)

	// Check if repo is partial clone
	if !repository.Config().GetBool("remote.origin.promisor", false) {
		return nil
	}

	cmdArgs := []string{
		"git",
		"rev-list",
		"--objects",
		"--missing=print",
	}

	if max, err := strconv.ParseInt(os.Getenv("MAX_COMMITS"), 10, 32); err == nil {
		maxCommits = int(max)
	} else {
		maxCommits = defaultMaxCommits
	}
	cmdArgs = append(cmdArgs, fmt.Sprintf("--max-count=%d", maxCommits))

	if len(args) > 0 {
		re := regexp.MustCompile(`^(0{40,}\.\.)`)
		for _, arg := range args {
			if re.MatchString(arg) {
				arg = re.ReplaceAllString(arg, "")
			}
			cmdArgs = append(cmdArgs, arg)
		}
	} else {
		cmdArgs = append(cmdArgs, "HEAD@{u}..HEAD")
	}
	cmdArgs = append(cmdArgs, "--", "po/")
	cmd = exec.Command(cmdArgs[0], cmdArgs[1:]...)
	stdout, err := cmd.StdoutPipe()
	if err == nil {
		err = cmd.Start()
	}
	if err != nil {
		return err
	}

	scanner = bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "?") {
			blobList = append(blobList, line[1:])
		}
	}
	if err = cmd.Wait(); err != nil {
		return err
	}
	if len(blobList) == 0 {
		log.Infof("no missing blobs of po/* in partial clone")
		return nil
	}

	cmd = exec.Command("git",
		"-c", "fetch.negotiationAlgorithm=noop",
		"fetch", "origin",
		"--no-tags",
		"--no-write-fetch-head",
		"--recurse-submodules=no",
		"--filter=blob:none",
		"--stdin",
	)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	go func() {
		defer stdin.Close()
		for _, blob := range blobList {
			if _, err := io.WriteString(stdin, blob); err != nil {
				log.Fatalf("fail to write blob id to git-fetch: %s", err)
			}
			if _, err := io.WriteString(stdin, "\n"); err != nil {
				log.Fatalf("fail to write blob id to git-fetch: %s", err)
			}
		}
	}()

	out, err = cmd.CombinedOutput()
	if err != nil {
		return err
	}
	log.Debugf("successfully fetched %d missing blob(s) in a batch from partial clone",
		len(blobList))
	scanner = bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		log.Info(scanner.Text())
	}
	return nil
}

// CmdCheckCommits implements check-commits sub command.
func CmdCheckCommits(args ...string) bool {
	var (
		commits = []string{}
		cmdArgs = []string{
			"git",
			"rev-list",
		}
		maxCommits int
		err        error
	)

	if max, err := strconv.ParseInt(os.Getenv("MAX_COMMITS"), 10, 32); err == nil {
		maxCommits = int(max)
	} else {
		maxCommits = defaultMaxCommits
	}
	if len(args) > 0 {
		re := regexp.MustCompile(`^(0{40,}\.\.)`)
		for _, arg := range args {
			if re.MatchString(arg) {
				arg = re.ReplaceAllString(arg, "")
			}
			cmdArgs = append(cmdArgs, arg)
		}
	} else {
		cmdArgs = append(cmdArgs, "HEAD@{u}..HEAD")
	}
	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.Dir = repository.GitDir()
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Errorf("fail to run git-rev-list: %s", err)
		return false
	}
	if err = cmd.Start(); err != nil {
		log.Errorf("fail to run git-rev-list: %s", err)
		return false
	}
	reader := bufio.NewReader(stdout)
	for {
		line, err := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if line != "" {
			commits = append(commits, line)
		}
		if err != nil {
			break
		}
	}
	err = cmd.Wait()
	if err != nil {
		log.Errorf("fail to run git-rev-list: %s", err)
		return false
	}
	nr := len(commits)
	if nr == 0 {
		log.Infoln("no commit checked.")
		return true
	}
	if nr > maxCommits {
		if flag.Force() {
			nr = maxCommits
		} else if !isatty.IsTerminal(os.Stdin.Fd()) || !isatty.IsTerminal(os.Stdout.Fd()) {
			log.Warnf("too many commits to check (%d > %d), check args or use option --force",
				len(commits), maxCommits)
			nr = maxCommits
		} else {
			answer := GetUserInput(fmt.Sprintf("too many commits to check (%d > %d), continue to run? (y/N)",
				len(commits), maxCommits),
				"no")
			if !AnswerIsTrue(answer) {
				return false
			}
		}
	}

	// Fetch missing objects ("po/*") in partial clone
	if err = fetchBlobsInPartialClone(args); err != nil {
		log.Warnf("fail to fetch missing blob in batch from partial clone: %s", err)
	}

	return checkCommits(commits[0:nr]...)
}

func checkCommits(commits ...string) bool {
	var (
		pass      = 0
		fail      = 0
		nr        = len(commits)
		tipCommit = commits[0]
		poMaps    = make(map[string]bool)
	)

	for i := 0; i < nr; i++ {
		var (
			commit         = commits[i]
			changes        []string
			notL10nChanges []string
			l10nChanges    []string
			ok             = true
			brk            = false
		)

		changes, ok = getCommitChanges(commit)
		if !ok {
			break
		}
		for _, change := range changes {
			if !strings.HasPrefix(change, PoDir+"/") && change != ".github/workflows/l10n.yml" {
				notL10nChanges = append(notL10nChanges, change)
			} else if change == "po/TEAMS" {
				l10nChanges = append(l10nChanges, change)
			} else if strings.HasSuffix(change, ".po") {
				l10nChanges = append(l10nChanges, change)
				poMaps[change] = true
			}
		}

		ok, brk = checkCommitChanges(commit, notL10nChanges, l10nChanges)

		if !brk {
			ok = checkCommitLog(commit) && ok
		}
		if brk {
			if !ok {
				fail++
			}
			break
		} else {
			if ok {
				pass++
			} else {
				fail++
			}
		}
	}

	if nr > pass+fail {
		log.Infof("checking commits: %d passed, %d failed, %d skipped.", pass, fail, nr-pass-fail)
	} else if fail != 0 {
		log.Infof("checking commits: %d passed, %d failed.", pass, fail)
	} else {
		log.Infof("checking commits: %d passed.", pass)
	}

	// We can disable this check using "--pot-file=no".
	if flag.GetPotFileFlag() != flag.PotFileFlagNone {
		poFiles := []string{}
		for file := range poMaps {
			poFiles = append(poFiles, file)
		}
		if ok := CheckUnfinishedPoFiles(tipCommit, poFiles); !ok {
			return false
		}
	}

	return fail == 0
}
