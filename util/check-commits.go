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
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/git-l10n/git-po-helper/flag"
	"github.com/git-l10n/git-po-helper/repository"
	"github.com/mattn/go-isatty"
	"github.com/qiniu/iconv"
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

const (
	checkResultBreak = 1 << iota
	checkResultError
)

type commitLog struct {
	// Meta holds header of a raw commit
	Meta map[string]interface{}
	// Msg holds commit message of a raw commit
	Msg []string
	// oid is commit ID for this commit
	oid string
}

func newCommitLog(oid string) commitLog {
	commitLog := commitLog{oid: oid}
	commitLog.Meta = make(map[string]interface{})
	commitLog.Msg = []string{}
	return commitLog
}

// Encoding is encoding for this commit log
func (v *commitLog) Encoding() string {
	if e, ok := v.Meta["encoding"]; ok {
		return e.(string)
	}
	return defaultEncoding
}

// CommitID is commit-id for this commit log
func (v *commitLog) CommitID() string {
	if len(v.oid) > 7 {
		return v.oid[:7]
	}
	return v.oid
}

func (v *commitLog) isMergeCommit() bool {
	if parents, ok := v.Meta["parent"]; ok {
		return len(parents.([]string)) > 1
	}
	return false
}

func (v *commitLog) hasGpgSig() bool {
	if val, ok := v.Meta["gpgsig"]; ok {
		return val.(bool)
	} else if val, ok := v.Meta["gpgsig-sha256"]; ok {
		return val.(bool)
	}
	return false
}

// Parse reads and parse raw commit object
func (v *commitLog) Parse(r io.Reader) bool {
	var (
		ret    = true
		isMeta = true
	)

	reader := bufio.NewReader(r)
	for {
		line, err := reader.ReadString('\n')
		if line == "" {
			break
		}
		line = strings.TrimRight(line, "\n")
		if isMeta && line == "" {
			isMeta = false
			continue
		}
		if isMeta {
			kv := strings.SplitN(line, " ", 2)
			if len(kv) != 2 {
				log.Errorf("commit %s: cannot parse commit HEADER: %s", v.CommitID(), line)
				ret = false
			}
			switch kv[0] {
			case "author", "committer", "encoding", "tree":
				if _, ok := v.Meta[kv[0]]; ok {
					log.Errorf("commit %s: duplicate header: %s", v.CommitID(), line)
					ret = false
				} else {
					v.Meta[kv[0]] = kv[1]
				}
			case "parent":
				if _, ok := v.Meta[kv[0]]; !ok {
					v.Meta[kv[0]] = []string{}
				}
				v.Meta[kv[0]] = append(v.Meta[kv[0]].([]string), kv[1])
			case "gpgsig", "gpgsig-sha256", "mergetag":
				if _, ok := v.Meta[kv[0]]; ok {
					log.Errorf("commit %s: duplicate header: %s", v.CommitID(), line)
					ret = false
					break
				}
				v.Meta[kv[0]] = true
				for {
					peek, err := reader.Peek(1)
					if err == nil {
						if peek[0] == ' ' {
							// Consume one line
							_, err = reader.ReadString('\n')
						} else {
							// Next header
							break
						}
					}
					if err != nil {
						log.Errorf(`commit %s: header "%s" is too short, early EOF: %s`,
							v.CommitID(), kv[0], err)
						ret = false
						break
					}
				}
			default:
				log.Errorf("commit %s: unknown commit header: %s", v.CommitID(), line)
				ret = false
			}
		} else {
			v.Msg = append(v.Msg, line)
		}
		if err != nil {
			break
		}
	}
	return ret
}

func getDuration(s int64) string {
	seconds := fmt.Sprintf("%ds", s)
	d, err := time.ParseDuration(seconds)
	if err != nil {
		log.Errorf("fail to parse duration: %s: %s", seconds, err)
		return seconds
	}
	return d.String()
}

func (v *commitLog) checkCommitDate(date string) error {
	// Timestamp of a commit is in UTC
	ts, err := strconv.ParseInt(date, 10, 64)
	if err != nil {
		return fmt.Errorf("bad timestamp: %s", date)
	}
	currentTs := time.Now().UTC().Unix()
	if ts > currentTs {
		return fmt.Errorf("date is in the future, %s from now",
			getDuration(ts-currentTs))
	} else if currentTs-ts > 3600*24*180 /* a half year earlier */ {
		log.Warnf("commit %s: too old commit date (%s earlier). Please check your system clock!",
			v.CommitID(), getDuration(currentTs-ts))
	}
	return nil
}

func (v *commitLog) checkAuthorCommitter() bool {
	var (
		ret               = true
		re                = regexp.MustCompile(`^(.+ <.+@.+\..+>) ([0-9]+)( ([+-][0-9]+))?$`)
		m                 []string
		value             string
		author, committer string
		err               error
	)

	if _, ok := v.Meta["author"]; !ok {
		log.Errorf("commit %s: cannot find author field in commit", v.CommitID())
		return false
	}
	if _, ok := v.Meta["committer"]; !ok {
		log.Errorf("commit %s: cannot find committer field in commit", v.CommitID())
		return false
	}

	value = v.Meta["author"].(string)
	m = re.FindStringSubmatch(value)
	if len(m) == 0 {
		log.Errorf("commit %s: bad format for author field: %s", v.CommitID(), value)
		ret = false
	} else {
		author = m[1]
		if err = v.checkCommitDate(m[2]); err != nil {
			log.Errorf("commit %s: bad author date: %s", v.CommitID(), err)
			ret = false
		}
	}

	value = v.Meta["committer"].(string)
	m = re.FindStringSubmatch(value)
	if len(m) == 0 {
		log.Errorf("commit %s: bad format for committer field: %s", v.CommitID(), value)
		ret = false
	} else {
		committer = m[1]
		if err = v.checkCommitDate(m[2]); err != nil {
			log.Errorf("commit %s: bad committer date: %s", v.CommitID(), err)
			ret = false
		}
	}
	if author != committer {
		log.Warnf("commit %s: author (%s) and committer (%s) are different",
			v.CommitID(), author, committer)
	}

	return ret
}

func abbrevMsg(line string) string {
	var (
		pos   = 0
		begin = 0
		width = len(line)
	)

	for ; pos < width; pos++ {
	inner:
		switch line[pos] {
		case ' ':
			fallthrough
		case '(':
			fallthrough
		case '"':
			break inner
		default:
			begin++
			continue
		}
		if begin > 0 && pos > 7 {
			break
		}
	}
	if pos == width {
		return line
	}
	return line[:pos] + " ..."
}

func (v *commitLog) checkSubject() bool {
	var (
		ret     = true
		nr      = len(v.Msg)
		subject string
		width   int
	)

	if nr > 1 {
		if v.Msg[1] != "" {
			log.Errorf("commit %s: no blank line between subject and body of commit message", v.CommitID())
			ret = false
		}
	} else if nr == 0 {
		log.Errorf("commit %s: do not have any commit message", v.CommitID())
		return false
	}

	subject = v.Msg[0]
	width = len(subject)

	if v.isMergeCommit() {
		if !strings.HasPrefix(subject, "Merge ") {
			log.Errorf(`commit %s: merge commit does not have prefix "Merge" in subject`,
				v.CommitID())
			ret = false
		}
	} else if !strings.HasPrefix(subject, commitSubjectPrefix+" ") {
		log.Errorf(`commit %s: subject ("%s") does not have prefix "%s"`,
			v.CommitID(),
			abbrevMsg(subject),
			commitSubjectPrefix)
		ret = false
	}

	if width > subjectWidthHardLimit {
		log.Errorf(`commit %s: subject ("%s") is too long: %d > %d`,
			v.CommitID(),
			abbrevMsg(subject),
			width,
			subjectWidthHardLimit)
		ret = false
	}
	for _, info := range []struct {
		Width   int
		Percent int
	}{
		{72, 98},
		{64, 90},
		{50, 63},
	} {
		if width > info.Width {
			log.Warnf(`commit %s: subject length %d > %d, about %d%% commits have a subject less than %d characters`,
				v.CommitID(),
				width,
				info.Width,
				info.Percent,
				info.Width)
			break
		}
	}
	if width == 0 {
		log.Errorf(`commit %s: subject is empty`, v.CommitID())
		return false
	}

	if subject[width-1] == '.' {
		log.Errorf("commit %s: subject should not end with period", v.CommitID())
		ret = false
	}

	for _, c := range subject {
		if c > unicode.MaxASCII || !unicode.IsPrint(c) {
			log.Errorf(`commit %s: subject has non-ascii character "%c"`, v.CommitID(), c)
			ret = false
			break
		}
	}

	return ret
}

func (v *commitLog) checkBody() bool {
	var (
		ret       = true
		nr        = len(v.Msg)
		width     int
		bodyStart int
		bodyEnd   int
		sigStart  int
	)

	if nr == 0 {
		// Already checked this case when checking subject.
		return false
	}
	if nr == 1 {
		if v.isMergeCommit() {
			log.Errorf("commit %s: empty body of the commit message, set merge.log=true",
				v.CommitID())
		} else {
			log.Errorf("commit %s: empty body of the commit message, no s-o-b signature",
				v.CommitID())
		}
		return false
	}
	if v.Msg[nr-1] == "" {
		log.Errorf("commit %s: empty line at the end of the commit message", v.CommitID())
		return false
	}
	emptyLines := 0
	for idx, line := range v.Msg {
		if line == "" {
			emptyLines++
		} else {
			emptyLines = 0
		}
		if emptyLines > 1 {
			log.Errorf("commit %s: too many empty lines found at line #%d",
				v.CommitID(),
				idx)
			return false
		}
	}

	// For a merge commit, do not check s-o-b signature and width of body.
	if v.isMergeCommit() {
		return ret
	}

	if v.Msg[1] != "" {
		// Error about no empty line between subject and body has been reported
		// when checking subject of commit log.
		bodyStart = 1
	} else {
		bodyStart = 2
	}

	// Signature is at the last part of the body, and has an empty line before it.
	sigStart = bodyStart
	for i := bodyStart; i < nr; i++ {
		if len(v.Msg[i]) == 0 {
			sigStart = i + 1
		}
	}

	// Check if has a s-o-b signature
	hasSobPrefix := false
	for i := sigStart; i < nr; i++ {
		if strings.HasPrefix(v.Msg[i], sobPrefix+" ") {
			hasSobPrefix = true
			break
		}
	}
	if hasSobPrefix {
		// Signature may have a email address longer than 80 characters, ignore them.
		bodyEnd = sigStart
	} else {
		// No signature, so needs to scan width of lines to end of the body.
		bodyEnd = nr
	}
	if !hasSobPrefix {
		log.Errorf(`commit %s: cannot find "%s" signature`,
			v.CommitID(),
			sobPrefix)
		ret = false
	}

	// Scan width of lines.
	for i := bodyStart; i < bodyEnd; i++ {
		width = len(v.Msg[i])
		if width > bodyWidthHardLimit {
			log.Errorf(`commit %s: line #%d ("%s") is too long: %d > %d`,
				v.CommitID(),
				i+1,
				abbrevMsg(v.Msg[i]),
				width,
				bodyWidthHardLimit)
			ret = false
		}
	}

	// Make sure all signatures are in format "key: value".
	if hasSobPrefix {
		for i := sigStart; i < nr; i++ {
			if !strings.Contains(v.Msg[i], ": ") {
				log.Errorf(`commit %s: no colon in signature at line #%d: "%s"`,
					v.CommitID(),
					i+1,
					abbrevMsg(v.Msg[i]))
				ret = false
				break
			}
		}
	}

	return ret
}

func (v *commitLog) checkGpg() bool {
	var ret = true

	if flag.NoGPG() {
		return ret
	}
	if v.hasGpgSig() {
		cmd := exec.Command("git",
			"verify-commit",
			v.CommitID())
		if err := cmd.Run(); err != nil {
			log.Errorf("commit %s: cannot verify gpg-sig: %s", v.CommitID(), err)
			ret = false
		}
	}
	return ret
}

func sameEncoding(enc1, enc2 string) bool {
	enc1 = strings.Replace(strings.ToLower(enc1), "-", "", -1)
	enc2 = strings.Replace(strings.ToLower(enc2), "-", "", -1)
	return enc1 == enc2
}

func (v *commitLog) checkEncoding() bool {
	var (
		ret      = true
		err      error
		out      = make([]byte, 1024)
		useIconv = true
		cd       iconv.Iconv
	)

	if sameEncoding(defaultEncoding, v.Encoding()) {
		useIconv = false
	} else {
		cd, err = iconv.Open(defaultEncoding, v.Encoding())
		if err != nil {
			log.Errorf("iconv.Open failed: %s", err)
			return false
		}
		defer cd.Close()
	}

	doEncodingCheck := func(list ...string) bool {
		var (
			err    error
			retVal = true
		)
		for _, line := range list {
			if useIconv {
				lineWidth := len(line)
				nLeft := lineWidth
				for nLeft > 0 {
					_, nLeft, err = cd.Do([]byte(line[lineWidth-nLeft:]), nLeft, out)
					if err != nil {
						log.Errorf(`commit %s: bad %s characters in: "%s"`,
							v.CommitID(), v.Encoding(), line)
						log.Errorf("\t%s", err)
						retVal = false
						break
					}
				}
			} else {
				if !utf8.ValidString(line) {
					log.Errorf(`commit %s: bad UTF-8 characters in: "%s"`,
						v.CommitID(), line)
					retVal = false
				}
			}
		}
		return retVal
	}

	// Check author, committer
	if !doEncodingCheck(v.Meta["author"].(string), v.Meta["committer"].(string)) {
		ret = false
	}

	// Check commit log
	if !doEncodingCheck(v.Msg...) {
		ret = false
	}

	return ret
}

func checkCommitLog(commit string) bool {
	var (
		ret       = true
		commitLog = newCommitLog(commit)
	)
	cmd := exec.Command("git",
		"cat-file",
		"commit",
		commit)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Errorf("Fail to get commit log of %s", commit)
		return false
	}
	if err = cmd.Start(); err != nil {
		log.Errorf("Fail to get commit log of %s", commit)
		return false
	}
	if !commitLog.Parse(stdout) {
		ret = false
	}
	if err = cmd.Wait(); err != nil {
		log.Errorf("Fail to get commit log of %s", commit)
		ret = false
	}

	if !commitLog.checkAuthorCommitter() {
		ret = false
	}
	if !commitLog.checkSubject() {
		ret = false
	}
	if !commitLog.checkBody() {
		ret = false
	}
	if !commitLog.checkEncoding() {
		ret = false
	}
	if !commitLog.checkGpg() {
		ret = false
	}

	return ret
}

func checkCommitChanges(commit string) int {
	var (
		err            error
		invalidChanges = []string{}
		verifyChanges  = []string{}
		ret            int
	)

	cmd := exec.Command("git",
		"diff-tree",
		"-r",
		commit)
	stdout, err := cmd.StdoutPipe()
	if err == nil {
		err = cmd.Start()
	}
	if err != nil {
		log.Errorf("commit %s: fail to run git-diff-tree: %s", AbbrevCommit(commit), err)
		return checkResultError | checkResultBreak
	}
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if idx := strings.Index(line, "\t"); idx >= 0 {
			fileName := line[idx+1:]
			if !strings.HasPrefix(fileName, PoDir+"/") {
				invalidChanges = append(invalidChanges, line[idx+1:])
			} else if fileName == "po/TEAMS" {
				verifyChanges = append(verifyChanges, fileName)
			} else if strings.HasSuffix(fileName, ".po") {
				verifyChanges = append(verifyChanges, fileName)
			}
		}
	}
	if err = cmd.Wait(); err != nil {
		log.Errorf("commit %s: fail to run git-diff-tree: %s", AbbrevCommit(commit), err)
		return checkResultError | checkResultBreak
	}
	if len(invalidChanges) > 0 {
		msg := bytes.NewBuffer(nil)
		msg.WriteString(fmt.Sprintf("commit %s: found changes beyond \"%s/\" directory:\n",
			AbbrevCommit(commit), PoDir))
		for _, change := range invalidChanges {
			msg.WriteString("\t\t")
			msg.WriteString(change)
			msg.WriteString("\n")
		}
		if len(verifyChanges) == 0 && flag.GitHubActionEvent() != "" {
			switch flag.GitHubActionEvent() {
			case "push":
				log.Warn(msg)
				log.Warnf(`commit %s: break because this commit is not for git-l10n`,
					AbbrevCommit(commit))
				// Ignore this error for push event.
				return checkResultBreak
			case "pull_request":
				fallthrough
			case "pull_request_target":
				fallthrough
			default:
				log.Error(msg)
				log.Errorf(`commit %s: break because this commit is not for git-l10n`,
					AbbrevCommit(commit))
				return checkResultError | checkResultBreak
			}
		}
		log.Error(msg)
		ret |= checkResultError
	}
	for _, fileName := range verifyChanges {
		tmpFile := FileRevision{
			Revision: commit,
			File:     fileName,
		}
		if err := checkoutTmpfile(&tmpFile); err != nil || tmpFile.Tmpfile == "" {
			log.Errorf("commit %s: fail to checkout %s of revision %s: %s",
				AbbrevCommit(commit), tmpFile.File, tmpFile.Revision, err)
			ret |= checkResultError
			continue
		}
		if fileName == "po/TEAMS" {
			if _, errors := ParseTeams(tmpFile.Tmpfile); len(errors) > 0 {
				for _, error := range errors {
					log.Errorf("commit %s: %s", AbbrevCommit(commit), error)
				}
				ret |= checkResultError
			}
		} else {
			locale := strings.TrimSuffix(filepath.Base(fileName), ".po")
			prompt := fmt.Sprintf("[%s@%s]",
				filepath.Join(PoDir, locale+".po"),
				AbbrevCommit(commit))
			if !CheckPoFileWithPrompt(locale, tmpFile.Tmpfile, prompt) {
				ret |= checkResultError
			}
		}
		os.Remove(tmpFile.Tmpfile)
	}
	return ret
}

// CheckCommit will run various checks for the given commit
func CheckCommit(commit string) int {
	var (
		ret int
	)

	ret = checkCommitChanges(commit)
	if (ret & checkResultBreak) != 0 {
		return ret
	}
	if !checkCommitLog(commit) {
		ret |= checkResultError
	}
	return ret
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
	log.Infof("successfully fetched %d missing blob(s) in a batch from partial clone",
		len(blobList))
	scanner = bufio.NewScanner(bytes.NewReader(out))
	if err != nil {
		for scanner.Scan() {
			log.Error(scanner.Text())
		}
		return err
	}
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

	pass := 0
	fail := 0
	for i := 0; i < nr; i++ {
		res := CheckCommit(commits[i])
		if (res & checkResultError) != 0 {
			fail++
		} else if res == 0 {
			pass++
		}
		if (res & checkResultBreak) != 0 {
			break
		}
	}
	if nr > 0 {
		if nr > pass+fail {
			log.Infof("checking commits: %d passed, %d failed, %d skipped.", pass, fail, nr-pass-fail)
		} else if fail != 0 {
			log.Infof("checking commits: %d passed, %d failed.", pass, fail)
		} else {
			log.Infof("checking commits: %d passed.", pass)
		}
	} else {
		log.Infoln("no commit checked.")
	}

	return fail == 0
}
