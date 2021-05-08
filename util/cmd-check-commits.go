package util

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mattn/go-isatty"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const (
	defaultMaxCommits = 100
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

func (v *commitLog) CommitID() string {
	if len(v.oid) > 7 {
		return v.oid[:7]
	}
	return v.oid
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
							v.CommitID(), err)
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

// Display show contents of commit
func (v *commitLog) Display() {
	for key, val := range v.Meta {
		switch key {
		case "author", "committer", "encoding", "tree":
			fmt.Printf("%s %s\n", key, val.(string))
		case "parent":
			for _, line := range val.([]string) {
				fmt.Printf("%s %s\n", key, line)
			}
		case "gpgsig", "gpgsig-sha256", "mergetag":
			fmt.Printf("%s ...\n", key)
			fmt.Println(" ...")
			fmt.Println(" ...")
		}
	}
	fmt.Println("")
	for _, line := range v.Msg {
		fmt.Println(line)
	}
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
			if !strings.HasPrefix(line[idx+1:], PoDir+"/") {
				badChanges = append(badChanges, line[idx+1:])
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
			log.Errorf("\t%s", change)
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
	cmdArgs = append(cmdArgs, args...)
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
	if len(commits) > int(maxCommits) && !viper.GetBool("force") {
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
