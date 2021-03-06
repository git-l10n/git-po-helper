// Package gettext provides utils for gettext.
package gettext

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/git-l10n/git-po-helper/flag"
	log "github.com/sirupsen/logrus"
)

// gettextApp defines the gettext application.
type gettextApp struct {
	Path  string
	Defer func()
}

// Versions of gettext available in the system.
const (
	Version0_14    = "0.14"
	VersionDefault = "default"
)

var (
	// GettextAppMap is a map of gettext versions to gettext apps.
	GettextAppMap = map[string]gettextApp{}
	// GettextAppHints is a map of hints for special gettext versions.
	GettextAppHints = map[string]string{
		Version0_14: "Need gettext 0.14 for some checks, see:\n    https://lore.kernel.org/git/874l8rwrh2.fsf@evledraar.gmail.com/",
	}

	showHintsCount = 0
)

// Program is the name of the program.
func (app gettextApp) Program(name string) string {
	return filepath.Join(app.Path, name)
}

// ShowHints shows hints for missing gettext versions.
func ShowHints() {
	if showHintsCount == 0 {
		showHintsCount++
		for version, msg := range GettextAppHints {
			if _, ok := GettextAppMap[version]; !ok {
				for _, line := range strings.Split(msg, "\n") {
					log.Warnln(line)
				}
			}
		}
	}
}

func gettextVersion(execPath string) (string, error) {
	cmd := exec.Command(execPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	version, err := bytes.NewBuffer(output).ReadString('\n')
	if err != nil {
		return "", err
	}
	version = strings.TrimSpace(version)
	items := strings.Split(version, " ")
	version = items[len(items)-1]
	if strings.HasPrefix(version, "0.14") || strings.HasPrefix(version, "0.15") {
		return Version0_14, nil
	}
	return version, nil
}

// FindGettext finds different versions of gettext installed.
func FindGettext() {
	execPath, err := exec.LookPath("gettext")
	if err == nil {
		if version, err := gettextVersion(execPath); err == nil {
			switch version {
			case Version0_14:
				GettextAppMap[Version0_14] = gettextApp{Path: filepath.Dir(execPath)}
			default:
				GettextAppMap[VersionDefault] = gettextApp{Path: filepath.Dir(execPath)}
			}
		}
	}

	if flag.NoSpecialGettextVersions() {
		return
	}

	for _, rootDir := range []string{
		"/opt/gettext",
		"/usr/local/Cellar/gettext",
	} {
		_ = filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
			if info == nil {
				return filepath.SkipDir
			}
			if !info.IsDir() {
				return nil
			}
			execPath = filepath.Join(path, "bin", "gettext")
			if fi, err := os.Stat(execPath); err == nil && fi.Mode().IsRegular() {
				if version, err := gettextVersion(execPath); err == nil {
					switch version {
					case Version0_14:
						if _, ok := GettextAppMap[Version0_14]; !ok {
							GettextAppMap[Version0_14] = gettextApp{Path: filepath.Dir(execPath)}
						}
					default:
						if _, ok := GettextAppMap[version]; !ok {
							GettextAppMap[version] = gettextApp{Path: filepath.Dir(execPath)}
						}
					}
				}
			}
			if path == rootDir {
				return nil
			}
			return filepath.SkipDir
		})
	}
}

func init() {
	cmd := exec.Command("git", "config", "--bool", "gettext.useMultipleVersions")
	out, _ := cmd.Output()
	if strings.TrimSpace(string(out)) == "true" {
		flag.SetGettextUseMultipleVersions(true)
	} else {
		flag.SetGettextUseMultipleVersions(false)
	}
}
