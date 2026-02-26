package util

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/git-l10n/git-po-helper/flag"
	"github.com/qiniu/iconv"
	log "github.com/sirupsen/logrus"
)

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
	currentTS := time.Now().UTC().Unix()
	if ts > currentTS {
		// Allow 15 minutes' drift for github actions
		if flag.GitHubActionEvent() == "" || ts-currentTS > 900 {
			return fmt.Errorf("date is in the future, %s from now",
				getDuration(ts-currentTS))
		}
	} else if currentTS-ts > 3600*24*180 /* a half year earlier */ {
		log.Warnf("commit %s: too old commit date (%s earlier). Please check your system clock!",
			v.CommitID(), getDuration(currentTS-ts))
	}
	return nil
}

func (v *commitLog) checkAuthorCommitter() bool {
	var (
		re                = regexp.MustCompile(`^(\S.+ <\S+@.+\.\S+>) ([0-9]+)( ([+-][0-9]+))?$`)
		m                 []string
		value             string
		author, committer string
		err               error
		errs              []string
		warns             []string
	)

	defer func() {
		if len(warns) > 0 {
			reportResultMessages(warns, "", log.WarnLevel)
		}
		if len(errs) > 0 {
			reportResultMessages(errs, "", log.ErrorLevel)
		}
	}()

	if _, ok := v.Meta["author"]; !ok {
		errs = append(errs,
			fmt.Sprintf("commit %s: cannot find author field in commit",
				v.CommitID()))
		return false
	}
	if _, ok := v.Meta["committer"]; !ok {
		errs = append(errs,
			fmt.Sprintf("commit %s: cannot find committer field in commit",
				v.CommitID()))
		return false
	}

	value = v.Meta["author"].(string)
	m = re.FindStringSubmatch(value)
	if len(m) == 0 {
		errs = append(errs,
			fmt.Sprintf("commit %s: bad format for author field: %s",
				v.CommitID(), value))
	} else {
		author = m[1]
		if err = v.checkCommitDate(m[2]); err != nil {
			errs = append(errs,
				fmt.Sprintf("commit %s: bad author date: %s",
					v.CommitID(), err))
		}
	}

	value = v.Meta["committer"].(string)
	m = re.FindStringSubmatch(value)
	if len(m) == 0 {
		errs = append(errs,
			fmt.Sprintf("commit %s: bad format for committer field: %s",
				v.CommitID(), value))
	} else {
		committer = m[1]
		if err = v.checkCommitDate(m[2]); err != nil {
			errs = append(errs,
				fmt.Sprintf("commit %s: bad committer date: %s", v.CommitID(), err))
		}
	}
	if author != committer {
		warns = append(warns,
			fmt.Sprintf("commit %s: author (%s) and committer (%s) are different",
				v.CommitID(), author, committer))
	}

	return len(errs) == 0
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
		nr      = len(v.Msg)
		subject string
		width   int
		errs    []string
		warns   []string
	)

	defer func() {
		if len(warns) > 0 {
			reportResultMessages(warns, "", log.WarnLevel)
		}
		if len(errs) > 0 {
			reportResultMessages(errs, "", log.ErrorLevel)
		}
	}()

	if nr == 0 {
		errs = append(errs,
			fmt.Sprintf("commit %s: do not have any commit message",
				v.CommitID()))
		return false
	} else if nr > 1 && v.Msg[1] != "" {
		errs = append(errs,
			fmt.Sprintf("commit %s: no blank line between subject and body of commit message",
				v.CommitID()))
	}

	subject = v.Msg[0]
	width = len(subject)

	if v.isMergeCommit() {
		if !strings.HasPrefix(subject, "Merge ") {
			errs = append(errs,
				fmt.Sprintf(`commit %s: merge commit does not have prefix "Merge" in subject`,
					v.CommitID()))
		}
	} else if !strings.HasPrefix(subject, commitSubjectPrefix+" ") {
		errs = append(errs,
			fmt.Sprintf(`commit %s: subject ("%s") does not have prefix "%s"`,
				v.CommitID(),
				abbrevMsg(subject),
				commitSubjectPrefix))
	}

	if width > subjectWidthHardLimit {
		errs = append(errs,
			fmt.Sprintf(`commit %s: subject ("%s") is too long: %d > %d`,
				v.CommitID(),
				abbrevMsg(subject),
				width,
				subjectWidthHardLimit))
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
			warns = append(warns,
				fmt.Sprintf(`commit %s: subject length %d > %d, about %d%% commits have a subject less than %d characters`,
					v.CommitID(),
					width,
					info.Width,
					info.Percent,
					info.Width))
			break
		}
	}
	if width == 0 {
		errs = append(errs,
			fmt.Sprintf(`commit %s: subject is empty`,
				v.CommitID()))
		return false
	}

	if subject[width-1] == '.' {
		errs = append(errs,
			fmt.Sprintf("commit %s: subject should not end with period",
				v.CommitID()))
	}

	for _, c := range subject {
		if c > unicode.MaxASCII || !unicode.IsPrint(c) {
			errs = append(errs,
				fmt.Sprintf(`commit %s: subject has non-ascii character "%c"`,
					v.CommitID(), c))
			break
		}
	}

	return len(errs) == 0
}

func (v *commitLog) checkBody() bool {
	var (
		nr        = len(v.Msg)
		width     int
		bodyStart int
		bodyEnd   int
		sigStart  int
		errs      []string
		warns     []string
	)

	defer func() {
		if len(warns) > 0 {
			reportResultMessages(warns, "", log.WarnLevel)
		}
		if len(errs) > 0 {
			reportResultMessages(errs, "", log.ErrorLevel)
		}
	}()

	if nr == 0 {
		// Already checked this case when checking subject.
		return false
	}

	if nr == 1 {
		if v.isMergeCommit() {
			errs = append(errs,
				fmt.Sprintf("commit %s: empty body of the commit message, set merge.log=true",
					v.CommitID()))
		} else {
			errs = append(errs,
				fmt.Sprintf("commit %s: empty body of the commit message, no s-o-b signature",
					v.CommitID()))
		}
		return false
	}
	if v.Msg[nr-1] == "" {
		errs = append(errs,
			fmt.Sprintf("commit %s: empty line at the end of the commit message",
				v.CommitID()))
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
			errs = append(errs,
				fmt.Sprintf("commit %s: too many empty lines found at line #%d",
					v.CommitID(),
					idx))
			return false
		}
	}

	// For a merge commit, do not check s-o-b signature and width of body.
	if v.isMergeCommit() {
		// no news is good news
		return len(errs) == 0
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
		errs = append(errs,
			fmt.Sprintf(`commit %s: cannot find "%s" signature`,
				v.CommitID(),
				sobPrefix))
	}

	// Scan width of lines.
	for i := bodyStart; i < bodyEnd; i++ {
		width = len(v.Msg[i])
		if width > bodyWidthHardLimit {
			errs = append(errs,
				fmt.Sprintf(`commit %s: line #%d ("%s") is too long: %d > %d`,
					v.CommitID(),
					i+1,
					abbrevMsg(v.Msg[i]),
					width,
					bodyWidthHardLimit))
		}
	}

	// Make sure all signatures are in format "key: value".
	if hasSobPrefix {
		for i := sigStart; i < nr; i++ {
			if !strings.Contains(v.Msg[i], ": ") {
				errs = append(errs,
					fmt.Sprintf(`commit %s: no colon in signature at line #%d: "%s"`,
						v.CommitID(),
						i+1,
						abbrevMsg(v.Msg[i])))
				break
			}
		}
	}

	return len(errs) == 0
}

func (v *commitLog) checkGpg() bool {
	var (
		errs []string
	)

	defer func() {
		if len(errs) > 0 {
			reportResultMessages(errs, "", log.ErrorLevel)
		}
	}()

	if flag.NoGPG() {
		return true
	}
	if v.hasGpgSig() {
		cmd := exec.Command("git",
			"verify-commit",
			v.CommitID())
		if err := cmd.Run(); err != nil {
			errs = append(errs,
				fmt.Sprintf("commit %s: cannot verify gpg-sig: %s",
					v.CommitID(), err))
		}
	}

	return len(errs) == 0
}

func sameEncoding(enc1, enc2 string) bool {
	enc1 = strings.Replace(strings.ToLower(enc1), "-", "", -1)
	enc2 = strings.Replace(strings.ToLower(enc2), "-", "", -1)
	return enc1 == enc2
}

func (v *commitLog) checkEncoding() bool {
	var (
		err      error
		out      = make([]byte, 1024)
		useIconv = true
		cd       iconv.Iconv
		errs     []string
		warns    []string
	)

	defer func() {
		if len(warns) > 0 {
			reportResultMessages(warns, "", log.WarnLevel)
		}
		if len(errs) > 0 {
			reportResultMessages(errs, "", log.ErrorLevel)
		}
	}()

	if sameEncoding(defaultEncoding, v.Encoding()) {
		useIconv = false
	} else {
		cd, err = iconv.Open(defaultEncoding, v.Encoding())
		if err != nil {
			errs = append(errs, fmt.Sprintf("iconv.Open failed: %s", err))
			return false
		}
		defer cd.Close()
	}

	doEncodingCheck := func(list ...string) {
		var (
			err error
		)
		for _, line := range list {
			if useIconv {
				lineWidth := len(line)
				nLeft := lineWidth
				for nLeft > 0 {
					_, nLeft, err = cd.Do([]byte(line[lineWidth-nLeft:]), nLeft, out)
					if err != nil {
						errs = append(errs,
							fmt.Sprintf(`commit %s: bad %s characters in: "%s"`,
								v.CommitID(), v.Encoding(), line),
							fmt.Sprintf("\t%s", err),
						)
						break
					}
				}
			} else {
				if !utf8.ValidString(line) {
					errs = append(errs,
						fmt.Sprintf(`commit %s: bad UTF-8 characters in: "%s"`,
							v.CommitID(), line))
				}
			}
		}
	}

	// Check author, committer
	doEncodingCheck(v.Meta["author"].(string), v.Meta["committer"].(string))

	// Check commit log
	doEncodingCheck(v.Msg...)

	return len(errs) == 0
}

func checkCommitLog(commit string) bool {
	var (
		ok        = true
		commitLog = newCommitLog(commit)
	)
	cmd := exec.Command("git",
		"cat-file",
		"commit",
		commit)
	stdout, err := cmd.StdoutPipe()
	if err == nil {
		err = cmd.Start()
	}
	if err != nil {
		log.Errorf("Fail to get commit log of %s", commit)
		return false
	}
	if !commitLog.Parse(stdout) {
		ok = false
	}
	if err = cmd.Wait(); err != nil {
		log.Errorf("Fail to get commit log of %s", commit)
		ok = false
	}

	ok = commitLog.checkAuthorCommitter() && ok
	ok = commitLog.checkSubject() && ok
	ok = commitLog.checkBody() && ok
	ok = commitLog.checkEncoding() && ok
	ok = commitLog.checkGpg() && ok

	return ok
}
