package gettext

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
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
	Version0_14 = "0.14"
	VersionAny  = "any"
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
	line, err := bytes.NewBuffer(output).ReadString('\n')
	if err != nil {
		return "", err
	}

	if strings.Contains(line, " 0.14") || strings.Contains(line, " 0.15") {
		return Version0_14, nil
	}
	return VersionAny, nil
}

func findGettext() {
	execPath, err := exec.LookPath("gettext")
	if err == nil {
		if version, err := gettextVersion(execPath); err == nil {
			switch version {
			case Version0_14:
				GettextAppMap[Version0_14] = gettextApp{
					Path: filepath.Dir(execPath),
				}
			default:
				GettextAppMap[VersionAny] = gettextApp{
					Path: filepath.Dir(execPath),
				}
			}
		}
	}

	if flag.NoSpecialGettextVersions() {
		return
	}

	doSearch := false
	for version := range GettextAppHints {
		if _, ok := GettextAppMap[version]; !ok {
			doSearch = true
		}
	}
	if !doSearch {
		return
	}

	for _, rootDir := range []string{
		"/opt/gettext",
		"/usr/local/Cellar/gettext",
	} {
		filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
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
					case VersionAny:
						fallthrough
					default:
						if _, ok := GettextAppMap[VersionAny]; !ok {
							GettextAppMap[VersionAny] = gettextApp{Path: filepath.Dir(execPath)}
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

var (
	installStatus = map[string]bool{}
)

// InstallGettext installs special gettext versions in temporary directory.
func InstallGettext(version string) error {
	var (
		gettextURL    string
		gettextSrcDir string
	)

	// Already installed
	if _, ok := GettextAppMap[version]; ok {
		return nil
	}
	// No such version
	if _, ok := GettextAppHints[version]; !ok {
		return fmt.Errorf("don't know how to install gettext version: %s", version)
	}
	// Only try to install once.
	if _, ok := installStatus[version]; ok {
		return nil
	}
	defer func() {
		installStatus[version] = false
	}()

	// Create temporary directory
	dirName, err := ioutil.TempDir("", "gettext")
	if err != nil {
		return err
	}

	// Download gettext
	targetFile := "gettext.tar.gz"
	out, err := os.Create(filepath.Join(dirName, targetFile))
	if err != nil {
		return err
	}
	defer out.Close()

	log.Infof("downloading gettext %s", version)
	switch version {
	case Version0_14:
		gettextURL = "https://ftp.gnu.org/gnu/gettext/gettext-0.14.6.tar.gz"
		gettextSrcDir = filepath.Join(dirName, "gettext-0.14.6")
	default:
		return fmt.Errorf("unknown gettext version: %s", version)
	}

	resp, err := http.Get(gettextURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}
	out.Close()

	// Extrace gettext tar.gz
	log.Infof("extracting gettext %s", version)
	cmd := exec.Command("tar", "-xzf", targetFile)
	cmd.Dir = dirName
	if err = cmd.Run(); err != nil {
		return err
	}

	// Build and install gettext in temporary directory.
	log.Infoln("running ./configure")
	cmd = exec.Command("sh", "./configure", "--prefix="+dirName)
	cmd.Dir = gettextSrcDir
	if err = cmd.Run(); err != nil {
		return err
	}

	log.Infoln("running: make")
	cmd = exec.Command("make")
	cmd.Dir = gettextSrcDir
	if err = cmd.Run(); err != nil {
		return err
	}

	log.Infoln("running: make install")
	cmd = exec.Command("make", "install")
	cmd.Dir = gettextSrcDir
	if err = cmd.Run(); err != nil {
		return err
	}

	// Add gettext to PATH
	GettextAppMap[version] = gettextApp{
		Path: filepath.Join(dirName, "bin"),
		// Cleanup temporary directory
		Defer: func() {
			log.Printf("removing temporary installed gettext at %s", dirName)
			os.RemoveAll(dirName)
		},
	}
	installStatus[version] = true
	return nil
}

func init() {
	findGettext()
}
