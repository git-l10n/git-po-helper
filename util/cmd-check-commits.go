package util

import (
	"bufio"
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

	"github.com/mattn/go-isatty"
	"github.com/qiniu/iconv"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const (
	defaultMaxCommits     = 100
	subjectWidthHardLimit = 62
	subjectWidthSoftLimit = 50
	bodyWidthHardLimit    = 72
	commitSubjectPrefix   = "l10n:"
	sobPrefix             = "Signed-off-by:"
	defaultEncoding       = "utf-8"
)

var (
	shouldCheckTeams = false
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

func (v *commitLog) hasMergeTag() bool {
	if val, ok := v.Meta["mergetag"]; ok {
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
					if err != nil {
						log.Errorf(`commit %s: header "%s" is too short, early EOF: %s`,
							v.CommitID(), kv[0], err)
						ret = false
						break
					}
					if peek[0] == ' ' {
						// Consume one line
						reader.ReadString('\n')
					} else {
						// Next header
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

func (v *commitLog) checkCommitDate(date string, timeZone string) error {
	ts, err := strconv.ParseInt(date, 10, 64)
	if err != nil {
		return fmt.Errorf("bad timestamp: %s", date)
	}
	if len(timeZone) > 0 {
		tz, err := strconv.ParseInt(timeZone[1:], 10, 64)
		if err != nil {
			return fmt.Errorf("bad timezone: %s", timeZone)
		}
		tz = tz * 36 /* tz * 3600 / 100 */
		if timeZone[0] == '+' {
			ts -= tz
		} else {
			ts += tz
		}
	}
	currentTs := time.Now().UTC().Unix()
	if ts > currentTs {
		return fmt.Errorf("date is in the future, %d seconds from now", ts-currentTs)
	}
	log.Debugf("commit %s: ts is : %d, currentTs is : %d", v.CommitID(), ts, currentTs)
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
		if err = v.checkCommitDate(m[2], m[4]); err != nil {
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
		if err = v.checkCommitDate(m[2], m[4]); err != nil {
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
		log.Errorf(`commit %s: do not have prefix "%s" in subject`,
			v.CommitID(), commitSubjectPrefix)
		ret = false
	}

	if width > subjectWidthHardLimit {
		log.Errorf(`commit %s: subject is too long (%d > %d)`,
			v.CommitID(), width, subjectWidthHardLimit)
		ret = false
	} else if width > subjectWidthSoftLimit {
		log.Warnf(`commit %s: subject is too long (%d > %d)`,
			v.CommitID(), width, subjectWidthSoftLimit)
	} else if width == 0 {
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
		sigStart  = 0
	)

	if nr == 0 {
		return false
	} else if nr > 1 {
		if v.Msg[1] != "" {
			bodyStart = 1
		} else if nr == 2 {
			log.Errorf("commit %s: empty body of commit message", v.CommitID())
			return false
		} else {
			bodyStart = 2
		}

		for i := bodyStart; i < nr; i++ {
			width = len(v.Msg[i])
			if width > bodyWidthHardLimit {
				log.Errorf(`commit %s: commit log message is too long (%d > %d)`,
					v.CommitID(), width, bodyWidthHardLimit)
				ret = false
			} else if width == 0 {
				sigStart = i
			}
		}
	}

	// For a merge commit, do not check s-o-b signature
	if v.isMergeCommit() {
		return ret
	}

	hasSobPrefix := false
	if sigStart == 0 {
		sigStart = nr - 1
	}
	for i := sigStart; i < nr; i++ {
		if strings.HasPrefix(v.Msg[i], sobPrefix+" ") {
			hasSobPrefix = true
			break
		}
	}
	if !hasSobPrefix {
		log.Errorf(`commit %s: cannot find "%s" signature`,
			v.CommitID(), sobPrefix)
		ret = false
	}
	return ret
}

func (v *commitLog) checkGpg() bool {
	var ret = true

	if viper.GetBool("check--no-gpg") || viper.GetBool("check-commits--no-gpg") {
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

func checkCommitChanges(commit string) bool {
	var (
		err        error
		badChanges = []string{}
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
		return false
	}
	reader := bufio.NewReader(stdout)
	for {
		line, err := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if idx := strings.Index(line, "\t"); idx >= 0 {
			fileName := line[idx+1:]
			if !strings.HasPrefix(fileName, PoDir+"/") {
				badChanges = append(badChanges, line[idx+1:])
			}
			if fileName == "po/TEAMS" {
				shouldCheckTeams = true
			}
		}
		if err != nil {
			break
		}
	}
	if err = cmd.Wait(); err != nil {
		log.Errorf("commit %s: fail to run git-diff-tree: %s", AbbrevCommit(commit), err)
		return false
	}
	if len(badChanges) > 0 {
		log.Errorf(`commit %s: found changes beyond "%s/" directory`,
			AbbrevCommit(commit), PoDir)
		for _, change := range badChanges {
			log.Errorf("\t\t%s", change)
		}
		return false
	}
	return true
}

// CheckCommit will run various checks for the given commit
func CheckCommit(commit string) bool {
	var ret = true

	if !checkCommitChanges(commit) {
		ret = false
	}
	if !checkCommitLog(commit) {
		ret = false
	}

	if shouldCheckTeams {
		ParseTeams(filepath.Join("po", "TEAMS"))

	}

	return ret
}

// CmdCheckCommits implements check-commits sub command.
func CmdCheckCommits(args ...string) bool {
	var (
		ret     = true
		commits = []string{}
		cmdArgs = []string{
			"git",
			"rev-list",
		}
		maxCommits int64
		err        error
	)

	maxCommits, err = strconv.ParseInt(os.Getenv("MAX_COMMITS"), 10, 32)
	if err != nil {
		maxCommits = defaultMaxCommits
	}
	if len(args) > 0 {
		cmdArgs = append(cmdArgs, args...)
	} else {
		cmdArgs = append(cmdArgs, "HEAD@{u}..HEAD")
	}
	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.Dir = GitRootDir
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Errorf("Fail to run git-rev-list: %s", err)
		return false
	}
	if err = cmd.Start(); err != nil {
		log.Errorf("Fail to run git-rev-list: %s", err)
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
	if len(commits) > int(maxCommits) &&
		(!viper.GetBool("check--force") && !viper.GetBool("check-commits--force")) {
		if isatty.IsTerminal(os.Stdin.Fd()) && isatty.IsTerminal(os.Stdout.Fd()) {
			answer := GetUserInput(fmt.Sprintf("too many commits to check (%d > %d), continue to run? (y/N)",
				len(commits), maxCommits),
				"no")
			if !AnswerIsTrue(answer) {
				return false
			}
		} else {
			log.Errorf("too many commits to check (%d > %d), check args or use option --force",
				len(commits), maxCommits)
			return false
		}
	}
	for _, commit := range commits {
		if !CheckCommit(commit) {
			ret = false
		}
	}
	return ret
}
