package modules

import (
	"bufio"
	"fmt"
	"github.com/caio-ishikawa/target-tracker/models"
	"log"
	"os/exec"
	"regexp"
	"sync"
)

func RunGau(scope models.Scope, outputChan chan string) {
	cmd := exec.Command("gau", scope.URL)

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
