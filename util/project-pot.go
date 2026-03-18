package util

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/git-l10n/git-po-helper/flag"
	log "github.com/sirupsen/logrus"
)

// DefaultPotAction is the default --pot-file when "auto".
type DefaultPotAction int

const (
	DefaultPotActionUndefined DefaultPotAction = iota
	DefaultPotActionNo
	DefaultPotActionAuto
	DefaultPotActionBuild
	DefaultPotActionDownload
	DefaultPotActionUseIfExist
)

func (a DefaultPotAction) String() string {
	switch a {
	case DefaultPotActionUndefined:
		return "undefined"
	case DefaultPotActionNo:
		return "no"
	case DefaultPotActionAuto:
		return "auto"
	case DefaultPotActionBuild:
		return "build"
	case DefaultPotActionDownload:
		return "download"
	case DefaultPotActionUseIfExist:
		return "use_if_exist"
	default:
		return "no"
	}
}

// ProjectPotConfig holds project-specific POT file defaults.
type ProjectPotConfig struct {
	// ProjectName is the project identifier for matching (e.g. "Git").
	ProjectName string
	// DownloadURL is the URL for downloading the POT file.
	DownloadURL string
	// BuildCmd is the command and args to build POT from source (e.g. []string{"make", "-j<job>", "pot"}).
	// Use <job> in an arg for job count substitution (replaced with runtime.NumCPU()).
	BuildCmd []string
	// BuildDirRel is relative to po dir, used to infer BuildDir (e.g. "../").
	BuildDirRel string
	// PotFilenameRel is the POT filename in po dir, used to infer PotFilename (e.g. "git.pot").
	PotFilenameRel string
	// DefaultAction is the default --pot-file when "auto".
	// Use GetEffectiveAction() to get the resolved value.
	DefaultAction DefaultPotAction

	// actualPotFile is the path from --pot-file=path when user specifies an explicit file.
	actualPotFile string
	// buildDir is the working directory for executing build cmd (inferred from poFile).
	// E.g. repo root when poFile is "po/zh_CN.po".
	buildDir string
	// potFilename is the POT file path relative to working directory (inferred from poFile).
	// E.g. "po/git.pot" when buildDir is repo root.
	potFilename string
	// effectiveAction is the resolved --pot-file action, set by Init.
	effectiveAction DefaultPotAction
}

var projectPotConfigs = []*ProjectPotConfig{
	{
		ProjectName:    "Git",
		DownloadURL:    "https://github.com/git-l10n/pot-changes/raw/pot/master/po/git.pot",
		BuildCmd:       []string{"make", "-j", "<job>", "pot"},
		BuildDirRel:    "../",
		PotFilenameRel: "git.pot",
		DefaultAction:  DefaultPotActionAuto,
	},
}

// defaultProjectPotConfig is the config for unknown projects.
var defaultProjectPotConfig = &ProjectPotConfig{
	DefaultAction: DefaultPotActionNo,
}

// Init initializes buildDir, potFilename and effectiveAction from poFile and flags.
func (c *ProjectPotConfig) Init(poFile string) {
	if poFile != "" {
		poDir := filepath.Dir(poFile)
		if c.BuildDirRel != "" {
			buildDir := filepath.Join(poDir, c.BuildDirRel)
			if abs, err := filepath.Abs(buildDir); err == nil {
				c.buildDir = abs
			}
		}
		if c.PotFilenameRel != "" {
			potFilename := filepath.Join(poDir, c.PotFilenameRel)
			if abs, err := filepath.Abs(potFilename); err == nil {
				c.potFilename = abs
			}
		}
	}
	opt := ""
	if flag.IsPotFileSet() {
		opt = strings.ToLower(flag.GetPotFileRaw())
	} else {
		opt = c.DefaultAction.String()
	}
	if opt == "" {
		opt = "auto"
	}
	switch opt {
	case "auto":
		if flag.GitHubActionEvent() != "" {
			c.effectiveAction = DefaultPotActionDownload
		} else {
			c.effectiveAction = DefaultPotActionBuild
		}
	case "no", "false":
		c.effectiveAction = DefaultPotActionNo
	case "build", "make", "update":
		c.effectiveAction = DefaultPotActionBuild
	case "download":
		c.effectiveAction = DefaultPotActionDownload
	case "use_if_exist":
		c.effectiveAction = DefaultPotActionUseIfExist
	default:
		if strings.Contains(opt, "/") || strings.HasSuffix(opt, ".pot") {
			c.actualPotFile = flag.GetPotFileRaw()
			c.effectiveAction = DefaultPotActionUseIfExist
		} else {
			log.Warnf("unknown --pot-file option: %s, using default action: %s",
				opt, c.DefaultAction.String())
			c.effectiveAction = DefaultPotActionNo
		}
	}
}

// SetActualPotFile sets the actual POT file path (from --pot-file=path) for use_if_exist.
// The path is global (from --pot-file flag), so it is stored on all configs.
func (c *ProjectPotConfig) SetActualPotFile(path string) {
	c.actualPotFile = path
}

// GetActualPotFile returns the actual POT file path.
func (c *ProjectPotConfig) GetActualPotFile() string {
	return c.actualPotFile
}

// GetEffectiveAction returns the effective action, set by Init.
// For configs that have not been initialized (e.g. defaultProjectPotConfig),
// computes and returns the action on the fly.
func (c *ProjectPotConfig) GetEffectiveAction() DefaultPotAction {
	if c.effectiveAction != DefaultPotActionUndefined {
		return c.effectiveAction
	}
	c.Init("")
	return c.effectiveAction
}

// BuildPotFile builds POT from source using BuildDir and BuildCmd.
// If actualPotFile is already set and exists, returns it without executing the command.
// On success, sets actualPotFile and returns the path.
func (c *ProjectPotConfig) BuildPotFile() (string, error) {
	actualPath := c.GetActualPotFile()
	if actualPath != "" {
		if Exist(actualPath) {
			return actualPath, nil
		} else {
			return "", fmt.Errorf("pot file %s does not exist", actualPath)
		}
	}
	if len(c.BuildCmd) == 0 {
		return "", errors.New("build command is not configured")
	}
	buildDir := c.buildDir
	if buildDir == "" {
		return "", errors.New("build directory is not set")
	}
	args := make([]string, len(c.BuildCmd))
	for i, a := range c.BuildCmd {
		if strings.Contains(a, "<job>") {
			args[i] = strings.ReplaceAll(a, "<job>", fmt.Sprintf("%d", runtime.NumCPU()))
		} else {
			args[i] = a
		}
	}
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = buildDir
	cmd.Stderr = os.Stderr
	reportSectionStart(log.InfoLevel, "Build pot template")
	log.Infof("update pot file by running: %s (in %s)", strings.Join(args, " "), buildDir)
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
		return "", fmt.Errorf("fail to build pot file: %w", err)
	}
	potPath := c.potFilename
	if !Exist(potPath) {
		return "", fmt.Errorf("pot file %s does not exist after build", potPath)
	}
	c.SetActualPotFile(potPath)
	return potPath, nil
}

// DownloadPotFile downloads the POT file from DownloadURL to a temp file.
// Returns the path to the downloaded file.
func (c *ProjectPotConfig) DownloadPotFile() (string, error) {
	url := c.DownloadURL
	if url == "" {
		return "", fmt.Errorf("download URL is not set for project: %s", c.ProjectName)
	}
	tmpfile, err := os.CreateTemp("", "*.pot")
	if err != nil {
		return "", err
	}
	tmpfile.Close()
	path := tmpfile.Name()
	showProgress := flag.GitHubActionEvent() == ""
	reportSectionStart(log.InfoLevel, "Download pot template")
	log.Infof("downloading pot file from %s", url)
	if err := httpDownload(url, path, showProgress); err != nil {
		os.Remove(path)
		return "", fmt.Errorf("fail to download: %w", err)
	}
	return path, nil
}

// AcquirePotFile acquires the POT file by building, downloading, or using existing.
// On success, sets actualPotFile and returns the path. On error, returns the error.
func (c *ProjectPotConfig) AcquirePotFile(projectName string, poFile string) (string, error) {
	action := c.GetEffectiveAction()
	if path := c.GetActualPotFile(); path != "" {
		if !Exist(path) {
			return "", fmt.Errorf("pot file %s does not exist", path)
		}
		return path, nil
	}
	var path string
	switch action {
	case DefaultPotActionDownload:
		var err error
		path, err = c.DownloadPotFile()
		if err != nil {
			return "", err
		}
	case DefaultPotActionBuild:
		var err error
		path, err = c.BuildPotFile()
		if err != nil {
			return "", err
		}
	case DefaultPotActionNo:
		return "", nil
	case DefaultPotActionUseIfExist:
		path = c.potFilename
		if !Exist(path) {
			return "", nil
		}
	default:
		return "", fmt.Errorf("unknown pot action: %s", action.String())
	}
	c.SetActualPotFile(path)
	return path, nil
}

// GetProjectPotConfig returns project config for projectName (case-insensitive match on ProjectName).
// When poFile is non-empty, infers BuildDir and PotFilename relative to current directory.
// Unknown projects get _default_ config (DefaultActionNo, empty other fields).
func GetProjectPotConfig(projectName string, poFile string) *ProjectPotConfig {
	var cfg *ProjectPotConfig
	for _, c := range projectPotConfigs {
		if strings.EqualFold(c.ProjectName, projectName) {
			cfg = c
			break
		}
	}
	if cfg == nil {
		cfg = defaultProjectPotConfig
	}
	cfg.Init(poFile)
	return cfg
}
