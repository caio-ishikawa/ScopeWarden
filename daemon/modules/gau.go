package modules

import (
	"bufio"
	"fmt"
	"github.com/caio-ishikawa/target-tracker/shared/models"
	"log"
	"os/exec"
	"regexp"
	"sync"
)

type commandExecution struct {
	command string
	args    []string
}

func RunModule(module models.Module, scope models.Scope, outputChan chan string) error {
	var command string
	var args []string

	switch module {
	case models.Gau:
		command = "gau"
		args = []string{scope.URL}
	case models.Waymore:
		command = "waymore"
		args = []string{"-i", scope.URL}
		if !scope.AcceptSubdomains {
			args = append(args, "--no-subs")
		}
	default:
		return fmt.Errorf("Unknown module: %s", module)
	}

	commandExecution := commandExecution{
		command: command,
		args:    args,
	}

	runCmd(commandExecution, outputChan)

	return nil
}

func runCmd(command commandExecution, outputChan chan string) {
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
		re := regexp.MustCompile(`^(https?:\/\/)?([a-zA-Z0-9.-]+\.[a-zA-Z]{2,})(:\d+)?(\/[^\r\n]*)?$`)
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			output := scanner.Text()
			isURL := re.MatchString(output)
			if !isURL {
				log.Printf("[%s] Output %s did not match URL - SKIPPING", models.Gau, output)
				continue
			}
			outputChan <- output

			continue
		}
	}()

	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			fmt.Println("[STDERR]", scanner.Text())
		}
	}()
	if err := cmd.Wait(); err != nil {
		log.Fatal(err)
	}
	wg.Wait()

	close(outputChan)
}
