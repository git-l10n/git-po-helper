package util

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// UpdatePotFile creates or update pot file. If the returned
// pot filename is not empty, it's caller's duty to remove it.
func UpdatePotFile(projectName string, poFile string) (string, bool) {
	cfg := GetProjectPotConfig(projectName, poFile)
	action := cfg.GetEffectiveAction()

	// We can disable this check using "--pot-file=no".
	if action == DefaultPotActionNo {
		path := cfg.GetActualPotFile()
		if path != "" {
			path = cfg.potFilename
		}
		return path, true
	}

	path, err := cfg.AcquirePotFile(projectName, poFile)
	if err != nil {
		log.Error(err)
		return "", false
	}
	return path, true
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
	localeFullName, localeErrs := GetPrettyLocaleName(locale)
	if len(localeErrs) > 0 {
		for _, e := range localeErrs {
			log.Errorf("fail to get locale name: %s", e)
		}
		return false
	}
	poFile = filepath.Join(PoDir, locale+".po")
	tmpFile = poFile + ".tmp"
	defer func() {
		os.Remove(tmpFile)
	}()

	// Load PO to get ProjectName for UpdatePotFile.
	projectName := ""
	if data, err := os.ReadFile(poFile); err == nil {
		if po, err := ParsePoEntries(data); err == nil {
			projectName = po.GetProject()
		}
	}

	// Update pot file.
	if poTemplate, ok = UpdatePotFile(projectName, poFile); !ok {
		return false
	}
	if poTemplate == "" {
		poTemplate = filepath.Join(PoDir, GitPot)
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
	if err := os.Rename(tmpFile, poFile); err != nil {
		log.Errorf(`fail to rename "%s" to "%s": %s`, tmpFile, poFile, err)
		return false
	}

	if optNoLocation {
		if err := cmd.Wait(); err != nil {
			log.Errorf(`wait failed: %s`, err)
			return false
		}
	}

	viper.Set("check--report-file-locations", "none")
	viper.Set("check--allow-obsolete", true)
	return CheckPoFile(locale, poFile)
}
