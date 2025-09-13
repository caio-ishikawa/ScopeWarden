package modules

import (
	"fmt"
	"os/exec"

	"github.com/caio-ishikawa/target-tracker/shared/models"
)

func RunWaymore(scope models.Scope) (string, error) {
	outputFile := fmt.Sprintf("/tmp/waymore-%s", scope.UUID)
	cmd := exec.Command("waymore", "-i", scope.URL, "-oU", outputFile)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("Failed to run Waymore: %w", err)
	}

	return outputFile, nil
}
