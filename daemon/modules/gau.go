package modules

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/caio-ishikawa/target-tracker/shared/models"
	"log"
	"os/exec"
	"regexp"
	"strings"
	"sync"
)

const (
	PortRegex = `^(\d+)\/(tcp|udp|sctp)\s+(open|closed|filtered|unfiltered|open\|filtered|closed\|filtered)\b.*$`
)

type ToolOutput struct {
	Output string
	Tool   Tool
}

type CommandExecution struct {
	command string
	args    []string
}

func RunModule(tool Tool, targetURL string, outputChan chan ToolOutput) error {
	execution, err := parseModuleCommand(tool, targetURL)
	if err != nil {
		return err
	}

	switch tool.ParserConfig.Type {
	case RealTimeOutput:
		runCmdAsync(tool, execution, outputChan)
	case FileOutput:
		// TODO: file output
	default:
		return fmt.Errorf("Failed to process parser type: %s", tool.ParserConfig.Type)
	}

	return nil
}

func parseModuleCommand(module Tool, targetURL string) (CommandExecution, error) {
	split := strings.Split(module.Cmd, " ")
	if len(split) == 0 {
		return CommandExecution{}, fmt.Errorf("Failed to parse tool %s command: could not detect <scope>", module.ID)
	}

	var output CommandExecution
	args := make([]string, 0)
	detectedScopePlaceholder := false
	for i, s := range split {
		if i == 0 {
			output.command = s
			continue
		}

		if s == ScopePlaceholder {
			args = append(args, targetURL)
			detectedScopePlaceholder = true
			continue
		}

		args = append(args, s)
	}

	output.args = args

	if !detectedScopePlaceholder {
		return CommandExecution{}, fmt.Errorf("Failed to parse tool %s command: could not detect <scope>", module.ID)
	}

	return output, nil
}

func runCmdAsync(tool Tool, command CommandExecution, outputChan chan ToolOutput) {
	cmd := exec.Command(command.command, command.args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}

	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		re := regexp.MustCompile(tool.ParserConfig.Regex)
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			output := scanner.Text()
			isURL := re.MatchString(output)
			if !isURL {
				log.Printf("[%s] Output %s did not match URL - SKIPPING", models.Gau, output)
				continue
			}

			toolOutput := ToolOutput{
				Tool:   tool,
				Output: output,
			}

			outputChan <- toolOutput

			continue
		}
	}()

	if tool.VerboseLogging {
		go func() {
			defer wg.Done()
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				fmt.Println("[STDERR]", scanner.Text())
			}
		}()
	}

	if err := cmd.Wait(); err != nil {
		log.Fatal(err)
	}
	wg.Wait()

	close(outputChan)
}

func RunPortScan(tool Tool, domain models.Domain, firstRun bool) ([]byte, error) {
	log.Printf("Running port scan for %s", domain.URL)

	commandExecution, err := tool.GeneratePortScanCmd(domain.URL)
	if err != nil {
		return nil, fmt.Errorf("Failed to generate port scan command: %w", err)
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.Command(commandExecution.command, commandExecution.args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("Failed to run port scan: %w", err)
	}

	if tool.VerboseLogging {
		log.Printf("[STDERR] %s", stderr.String())
	}

	return stdout.Bytes(), nil
}
