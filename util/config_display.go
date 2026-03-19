package util

import (
	"fmt"
	"os"

	"github.com/git-l10n/git-po-helper/config"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// fullConfigDisplay holds agent config and merged projects for YAML output.
type fullConfigDisplay struct {
	config.AgentConfig `yaml:",inline"`
	Projects           map[string]config.PotProjectEntry `yaml:"projects,omitempty"`
}

// projectPotConfigToEntry converts ProjectPotConfig to PotProjectEntry for display.
func projectPotConfigToEntry(c *ProjectPotConfig) config.PotProjectEntry {
	e := config.PotProjectEntry{
		DownloadURL:       c.DownloadURL,
		BuildDirRel:       c.BuildDirRel,
		PotFilenameRel:    c.PotFilenameRel,
		MinGettextVersion: c.MinGettextVersion,
	}
	if len(c.BuildCmd) > 0 {
		e.BuildCmd = make([]string, len(c.BuildCmd))
		copy(e.BuildCmd, c.BuildCmd)
	}
	if c.DefaultAction != DefaultPotActionUndefined {
		e.DefaultAction = c.DefaultAction.String()
	}
	return e
}

// CmdShowConfig displays the full configuration (agent + projects) in YAML format.
// Used by the top-level git-po-helper config command.
func CmdShowConfig() error {
	cfg, err := LoadAgentConfigForCmd()
	if err != nil {
		log.Errorf("failed to load configuration: %v", err)
		return err
	}

	projects := getMergedProjectPotConfigs()
	projectsMap := make(map[string]config.PotProjectEntry)
	for _, c := range projects {
		projectsMap[c.ProjectName] = projectPotConfigToEntry(c)
	}

	display := fullConfigDisplay{
		AgentConfig: *cfg,
		Projects:    projectsMap,
	}

	yamlData, err := yaml.Marshal(&display)
	if err != nil {
		log.Errorf("failed to marshal configuration to YAML: %v", err)
		return fmt.Errorf("failed to marshal configuration to YAML: %w", err)
	}

	fmt.Println("# Full configuration (agent + projects)")
	fmt.Println("# Merged from:")
	fmt.Println("# - User home directory: ~/.git-po-helper.yaml (lower priority)")
	fmt.Println("# - Repository root: <repo-root>/.git-po-helper.yaml (higher priority)")
	fmt.Println("# - Or --config <path> when specified")
	fmt.Println()
	os.Stdout.Write(yamlData)

	return nil
}
