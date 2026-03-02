package util

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/git-l10n/git-po-helper/flag"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// UpdatePotFile creates or update pot file. If the returned
// pot filename is not empty, it's caller's duty to remove it.
func UpdatePotFile() (string, bool) {
	var (
		opt = flag.GetPotFileFlag()
	)

	// We can disable this check using "--pot-file=no".
	if opt == flag.PotFileFlagNone {
		return "", true
	}

	// Try to download pot file.
	if opt == flag.PotFileFlagDownload {
		showProgress := flag.GitHubActionEvent() == ""
		tmpfile, err := os.CreateTemp("", "git.pot-*")
		if err != nil {
			log.Error(err)
			return "", false
		}
		tmpfile.Close()
		potFile := tmpfile.Name()
		showHorizontalLine()
		log.Infof("downloading pot file from %s", PotFileURL)
		if err := httpDownload(PotFileURL, potFile, showProgress); err != nil {
			os.Remove(potFile)
			potFile = ""
			for _, msg := range []string{
				fmt.Sprintf("fail to download latest pot file from %s.", PotFileURL),
				"",
				fmt.Sprintf("\t%s", err),
				"",
				"you can use option '--pot-file=build' to build the pot file from",
				"the source instead of downloading",
			} {
				log.Error(msg)
			}
			return "", false
		}
		return potFile, true
	}

	// Try to use the specific pot file in location.
	if opt == flag.PotFileFlagLocation {
		potFile := flag.GetPotFileLocation()
		if !Exist(potFile) {
			showHorizontalLine()
			for _, msg := range []string{
				fmt.Sprintf("pot file '%s' does not exist", potFile),
				"",
				"you can use option '--pot-file=download' to download pot file from",
				"the l10n coordinator's repository,",
				"or use option '--pot-file=build' to build the pot file from the source",
				"instead of downloading",
			} {
				log.Error(msg)
			}

			return "", false
		}
		return "", true
	}

	// Try to build pot file from source.
	if opt == flag.PotFileFlagUpdate {
		cmd := exec.Command("make", "pot")
		showHorizontalLine()
		log.Info("update pot file by running: make pot")
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			for _, msg := range []string{
				"fail to build the pot file from source",
				"",
				fmt.Sprintf("\t%s", err),
				"",
				"you can use option '--pot-file=download' to download pot file from",
				"the l10n coordinator's repository",
			} {
				log.Error(msg)
			}
			return "", false
		}
		return "", true
	}

	// Unknown option.
	log.Errorf("bad '--pot-file' option: %s", viper.GetString("pot-file"))
	return "", false
}

// CmdUpdate implements update sub command.
func CmdUpdate(fileName string) bool {
	var (
		cmd               *exec.Cmd
		msgCatCmd         *exec.Cmd
		locale            string
		localeFullName    string
		err               error
		poFile            string
		tmpFile           string
		cmdArgs           []string
		poTemplate        string
		ok                bool
		optNoLocation     = viper.GetBool("no-location")
		optNoFileLocation = viper.GetBool("no-file-location")
		output            []byte
	)

	locale = strings.TrimSuffix(filepath.Base(fileName), ".po")
	if localeFullName, err = GetPrettyLocaleName(locale); err != nil {
		log.Errorf("fail to get locale name: %s", err)
		return false
	}
	poFile = filepath.Join(PoDir, locale+".po")
	tmpFile = poFile + ".tmp"
	defer func() {
		os.Remove(tmpFile)
	}()

	// Update pot file.
	if poTemplate, ok = UpdatePotFile(); !ok {
		return false
	}
	if poTemplate == "" {
		poTemplate = filepath.Join(PoDir, GitPot)
	} else {
		defer os.Remove(poTemplate)
	}

	if !Exist(poTemplate) {
		log.Errorf(`fail to update "%s", pot file does not exist`, poFile)
		return false
	}
	if !Exist(poFile) {
		log.Errorf(`fail to update "%s", does not exist`, poFile)
		return false
	}

	cmdArgs = []string{"msgmerge"}
	if optNoFileLocation {
		cmdArgs = append(cmdArgs, "--no-location")
	} else {
		cmdArgs = append(cmdArgs, "--add-location=file")
	}
	cmdArgs = append(cmdArgs,
		"-o", "-", // Save output to stdout
		poFile,
		poTemplate,
	)
	log.Infof(`run msgmerge for "%s": %s`, localeFullName, strings.Join(cmdArgs, " "))
	cmd = exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.Stderr = os.Stderr

	if optNoLocation {
		msgCatCmdArgs := []string{"msgcat", "--add-location=file", "-"}
		log.Infof(`run msgcat for "%s": %s`, localeFullName, strings.Join(msgCatCmdArgs, " "))
		msgCatCmd = exec.Command(msgCatCmdArgs[0], msgCatCmdArgs[1:]...)
		msgCatCmd.Stdin, err = cmd.StdoutPipe()
		if err != nil {
			log.Errorf("fail to create pipe: %v\n", err)
			return false
		}
		if err := cmd.Start(); err != nil {
			log.Errorf(`fail to start msgmerge: %s`, err)
			return false
		}
		output, err = msgCatCmd.Output()
		if err != nil {
			log.Errorf(`fail to read output for "%s": %s`, poFile, err)
			return false
		}
	} else {
		output, err = cmd.Output()
		if err != nil {
			log.Errorf(`fail to read output for "%s": %s`, poFile, err)
			return false
		}
	}

	if err := os.WriteFile(tmpFile, output, 0644); err != nil {
		log.Errorf(`fail to write to "%s": %s`, tmpFile, err)
		return false
	}
	os.Rename(tmpFile, poFile)

	if optNoLocation {
		if err := cmd.Wait(); err != nil {
			log.Errorf(`wait failed: %s`, err)
			return false
		}
	}

	viper.Set("check--report-file-locations", "none")
	return CheckPoFile(locale, poFile)
}
