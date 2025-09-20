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

	TargetPlaceholder   = "<target>"
	WordlistPlaceholder = "<wordlist>"
	PortScanTool        = "nmap"
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

type BruteForceCondition struct {
	Technology string `yaml:"technology"`
	Wordlist   string `yaml:"wordlist"`
}

type BruteForceConfig struct {
	Run        bool                  `yaml:"run"`
	Cmd        string                `yaml:"command"`
	Regex      string                `yaml:"regex"`
	Conditions []BruteForceCondition `yaml:"conditions"`
}

type PortScanConfig struct {
	Run   bool     `yaml:"run"`
	Ports []string `yaml:"ports"`
}

type Tool struct {
	ID               string             `yaml:"id"`
	Cmd              string             `yaml:"command"`
	Table            models.Table       `yaml:"table"`
	TargetTable      models.Table       `yaml:"target_table"`
	VerboseLogging   bool               `yaml:"verbose"`
	PortScanConfig   PortScanConfig     `yaml:"port_scan"`
	ParserConfig     OutputParserConfig `yaml:"parser"`
	BruteForceConfig BruteForceConfig   `yaml:"brute_force"`
}

func GeneratePortScanCmd(ports []string, target string) (CommandExecution, error) {
	output := CommandExecution{
		command: PortScanTool,
	}

	parsedTarget, err := url.Parse(target)
	if err != nil {
		return CommandExecution{}, fmt.Errorf("Failed to parse URL for port scan: %w", err)
	}

	scanTarget := parsedTarget.Host

	args := make([]string, 0)
	if len(ports) == 0 {
		args = []string{scanTarget, "-Pn", "-T3"}
	} else {
		args = []string{scanTarget, "-Pn", "-T3", "-p", strings.Join(ports, ",")}
	}

	output.args = args

	return output, nil
}

func GenerateBruteForceCmd(bruteForceConfig BruteForceConfig, target string, technology string) (*CommandExecution, error) {
	var output CommandExecution
	foundTechWordlist := false
	args := make([]string, 0)

	split := strings.Fields(bruteForceConfig.Cmd)
	for i, s := range split {
		if i == 0 {
			output.command = s
			continue
		}
		if s == TargetPlaceholder {
			args = append(args, target)
			continue
		}
		if s == WordlistPlaceholder {
			for _, conditions := range bruteForceConfig.Conditions {
				if conditions.Technology == technology {
					args = append(args, conditions.Wordlist)
					foundTechWordlist = true
					break
				}
			}

			// Return nil if no wordlist is found for technology
			if !foundTechWordlist {
				return nil, nil
			}

			continue
		}

		args = append(args, s)
	}

	output.args = args

	return &output, nil
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

		if tool.BruteForceConfig.Run {

		}
	}

	return modules, nil
}
