package util

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
)

type FileRevision struct {
	Revision string
	File     string
	Tmpfile  string
}

// Example output:
//     git.pot:NNN: this message is used but not defined in /tmp/git.po.XXXX
//     /tmp/git.po.XXXX:NNN: warning: this message is not used
var (
	reNewEntry = regexp.MustCompile(`:([0-9]*): this message is used but not defined in`)
	reDelEntry = regexp.MustCompile(`:([0-9]*): warning: this message is not used`)
)

func checkoutTmpfile(f *FileRevision) error {
	if f.Revision == "" {
		return nil
	}
	tmpfile, err := os.CreateTemp("", "*--"+filepath.Base(f.File))
	if err != nil {
		return fmt.Errorf("fail to create tmpfile: %s", err)
	}
	cmd := exec.Command("git",
		"show",
		f.Revision+":"+f.File)
	cmd.Stderr = os.Stderr
	out, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf(`get StdoutPipe failed: %s`, err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("fail to start git-show command: %s", err)
	}
	if _, err := io.Copy(tmpfile, out); err != nil {
		return fmt.Errorf("fail to write tmpfile: %s", err)
	}
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("fail to wait git-show command: %s", err)
	}
	if err := tmpfile.Close(); err != nil {
		return fmt.Errorf("fail to close tmpfile: %s", err)
	}
	f.Tmpfile = tmpfile.Name()
	log.Debugf(`creating "%s" file using command: %s`, f.Tmpfile, cmd.String())
	return nil
}

func DiffFileRevision(src, dest FileRevision) bool {
	var (
		srcFile  string
		destFile string
	)
	if err := checkoutTmpfile(&src); err != nil {
		log.Errorf("fail to checkout %s of revision %s: %s", src.File, src.Revision, err)
	}
	if err := checkoutTmpfile(&dest); err != nil {
		log.Errorf("fail to checkout %s of revision %s: %s", dest.File, dest.Revision, err)
	}
	if src.Tmpfile != "" {
		srcFile = src.Tmpfile
		defer func() {
			os.Remove(src.Tmpfile)
			src.Tmpfile = ""
		}()
	} else {
		srcFile = src.File
	}
	if dest.Tmpfile != "" {
		destFile = dest.Tmpfile
		defer func() {
			os.Remove(dest.Tmpfile)
			dest.Tmpfile = ""
		}()
	} else {
		destFile = dest.File
	}
	return DiffFiles(srcFile, destFile)
}

func DiffFiles(src string, dest string) bool {
	var (
		add int32
		del int32
	)
	if !Exist(src) {
		log.Fatalf(`file "%s" not exist`, src)
	}
	if !Exist(dest) {
		log.Fatalf(`file "%s" not exist`, dest)
	}

	cmd := exec.Command("msgcmp",
		"-N",
		"--use-untranslated",
		src,
		dest)

	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "LANGUAGE=C")
	out, err := cmd.StderrPipe()
	if err != nil {
		log.Fatalf("fail to run msgcmp: %s", err)
	}
	log.Debugf("running diff command: %s", cmd.String())
	if err := cmd.Start(); err != nil {
		log.Fatalf("fail to start msgcmp: %s", err)
	}
	reader := bufio.NewReader(out)
	for {
		line, err := reader.ReadString('\n')
		if line == "" {
			break
		}
		if reNewEntry.MatchString(line) {
			add++
		} else if reDelEntry.MatchString(line) {
			del++
		}

		if err != nil {
			break
		}
	}
	cmd.Wait()
	diffStat := ""
	if add != 0 {
		diffStat = fmt.Sprintf("%d new", add)
	}
	if del != 0 {
		if diffStat != "" {
			diffStat += ", "
		}
		diffStat += fmt.Sprintf("%d removed", del)
	}
	fmt.Fprintf(os.Stderr, "# Diff between %s and %s\n",
		filepath.Base(src), filepath.Base(dest))
	if diffStat == "" {
		fmt.Fprintf(os.Stderr, "\tNothing changed.\n")
	}

	if filepath.Base(dest) == GitPot {
		gitDescribe := ""
		if out, err := exec.Command("git", "describe", "--always").Output(); err == nil {
			gitDescribe = strings.TrimSpace(string(out))
		}
		fmt.Fprintf(os.Stderr, "l10n: git.pot: vN.N.N round N (%s)\n", diffStat)
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintf(os.Stderr, "Generate po/git.pot from (%s) for git vN.N.N l10n round N.\n",
			gitDescribe)
	} else {
		fmt.Fprintf(os.Stderr, "\t%s\n", diffStat)
	}
	return true
}
