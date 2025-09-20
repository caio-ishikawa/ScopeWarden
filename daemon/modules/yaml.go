package modules

import (
	"fmt"
	"github.com/caio-ishikawa/target-tracker/shared/models"
	"gopkg.in/yaml.v3"
	"net/url"
	"os"
	"strings"
)

type OutputType string

const (
	RealTimeOutput OutputType = "realtime"
	FileOutput     OutputType = "file"

	ScopePlaceholder = "<scope>"
	PortScanTool     = "nmap"
)

type GlobalConfig struct {
	// How frequently the scan runs (in hours)
	Schedule int `yaml:"schedule"`
}

type OutputParserConfig struct {
	Type  OutputType `yaml:"type"`
	File  string     `yaml:"file"`
	Regex string     `yaml:"regex"`
}

type PortScanConfig struct {
	Run   bool     `yaml:"run"`
	Ports []string `yaml:"ports"`
}

type Tool struct {
	ID             string             `yaml:"id"`
	Cmd            string             `yaml:"command"`
	Table          models.Table       `yaml:"table"`
	VerboseLogging bool               `yaml:"verbose"`
	PortScanConfig PortScanConfig     `yaml:"port_scan"`
	ParserConfig   OutputParserConfig `yaml:"parser"`
}

func (t *Tool) GeneratePortScanCmd(target string) (CommandExecution, error) {
	output := CommandExecution{
		command: PortScanTool,
	}

	parsedTarget, err := url.Parse(target)
	if err != nil {
		return CommandExecution{}, fmt.Errorf("Failed to parse URL for port scan: %w", err)
	}

	scanTarget := parsedTarget.Host

	args := make([]string, 0)
	if len(t.PortScanConfig.Ports) == 0 {
		args = []string{scanTarget, "-Pn", "-T3"}
	} else {
		args = []string{scanTarget, "-Pn", "-T3", "-p", strings.Join(t.PortScanConfig.Ports, ",")}
	}

	output.args = args

	return output, nil
}

type DaemonConfig struct {
	Global GlobalConfig `yaml:"global"`
	Tools  []Tool       `yaml:"tools"`
}

func NewDaemonConfig() (DaemonConfig, error) {
	configFilePath := os.Getenv("TARGET_TRACKER_CONFIG")
	if configFilePath == "" {
		return DaemonConfig{}, fmt.Errorf("Failed to read config yaml: config file path not set")
	}

	file, err := os.ReadFile(configFilePath)
	if err != nil {
		return DaemonConfig{}, fmt.Errorf("Failed to read config yaml: %w", err)
	}

	var modules DaemonConfig
	if err := yaml.Unmarshal(file, &modules); err != nil {
		return DaemonConfig{}, fmt.Errorf("Failed to parse config yaml: %w", err)
	}

	// Validate tools config
	for _, tool := range modules.Tools {
		if tool.Cmd == "" {
			return DaemonConfig{}, fmt.Errorf("Failed to parse tool %s command: empty command", tool.ID)
		}

		if tool.ParserConfig.Type == FileOutput && tool.ParserConfig.File == "" {
			return DaemonConfig{}, fmt.Errorf("Invalid config yaml: Tool %s has parser type 'file' but no output file set", tool.ID)
		}

		if tool.ParserConfig.Regex == "" {
			return DaemonConfig{}, fmt.Errorf("Invalid config yaml: Empty regex for tool %s", tool.ID)
		}
	}

	return modules, nil
}
