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
		cmd             *exec.Cmd
		locale          string
		localeFullName  string
		poFile          string
		tmpFile         string
		cmdArgs         []string
		poTemplate      string
		ok              bool
		optNoLineNumber = viper.GetBool("no-line-number")
		optNoLocation   = viper.GetBool("no-location")
	)

	locale = strings.TrimSuffix(filepath.Base(fileName), ".po")
	localeFullName = FormatLocaleName(locale)
	localeErrs := ValidateLocale(locale)
	// ISO / locale validation errors are logged but do not stop the update when
	// FormatLocaleName still yields a display name (warn-only). If the locale
	// cannot be interpreted at all (empty display name), abort.
	if len(localeErrs) > 0 {
		for _, e := range localeErrs {
			log.Errorf("%s", e)
		}
		if localeFullName == "" {
			return false
		}
	}
	poFile = filepath.Clean(fileName)
	if rel := filepath.ToSlash(poFile); !strings.HasPrefix(rel, PoDir+"/") {
		poFile = filepath.Join(PoDir, locale+".po")
	}
	tmpFile = poFile + ".tmp"
	defer func() {
		os.Remove(tmpFile)
	}()

	// Load PO to get ProjectName for UpdatePotFile.
	if !Exist(poFile) {
		log.Errorf(`fail to update "%s", does not exist`, poFile)
		return false
	}
	projectName := ""
	data, err := os.ReadFile(poFile)
	if err != nil {
		log.Errorf("fail to read %s: %s", poFile, err)
		return false
	}
	if po, err := ParsePoEntries(data); err == nil {
		projectName = po.GetProject()
	} else {
		log.Errorf("fail to parse %s: %s", poFile, err)
		return false
	}

	if projectName == "" {
		log.Errorf("fail to get project name from %s", poFile)
		return false
	}

	// Update pot file.
	if poTemplate, ok = UpdatePotFile(projectName, poFile); !ok {
		return false
	}
	if poTemplate == "" {
		log.Errorf("fail to update %s, unknown pot file", poFile)
		return false
	}

	if !Exist(poTemplate) {
		log.Errorf(`fail to update "%s", pot file "%s" does not exist`, poFile, poTemplate)
		return false
	}

	cmdArgs = []string{"msgmerge"}
	if optNoLocation {
		cmdArgs = append(cmdArgs, "--no-location")
	} else if optNoLineNumber {
		cmdArgs = append(cmdArgs, "--add-location=file")
	}
	cmdArgs = append(cmdArgs,
		"-o", tmpFile,
		poFile,
		poTemplate,
	)
	log.Infof(`run msgmerge for "%s": %s`, localeFullName, strings.Join(cmdArgs, " "))
	cmd = exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Errorf(`msgmerge failed for "%s": %s`, poFile, err)
		return false
	}

	if err := os.Rename(tmpFile, poFile); err != nil {
		log.Errorf(`fail to rename "%s" to "%s": %s`, tmpFile, poFile, err)
		return false
	}

	prevReportLoc := viper.GetString("check--report-file-locations")
	prevAllowObsolete := viper.GetBool("check--allow-obsolete")
	defer func() {
		viper.Set("check--report-file-locations", prevReportLoc)
		viper.Set("check--allow-obsolete", prevAllowObsolete)
	}()
	viper.Set("check--report-file-locations", "none")
	viper.Set("check--allow-obsolete", true)
	return CheckPoFile(locale, poFile, true)
}
