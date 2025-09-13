package main

import (
	"github.com/caio-ishikawa/target-tracker/daemon/modules"
	"github.com/caio-ishikawa/target-tracker/shared/models"
	"github.com/google/uuid"
	"log"
)

func main() {
	app, err := NewDaemon()
	if err != nil {
		log.Fatal(err)
	}

	targetUUID := uuid.NewString()
	target := models.Target{
		UUID: targetUUID,
		Name: "NASA",
	}
	if err := app.db.InsertTarget(target); err != nil {
		log.Fatal(err)
	}

	s := models.Scope{
		UUID:       uuid.NewString(),
		TargetUUID: targetUUID,
		URL:        "nasa.gov",
		FirstRun:   true,
	}

	if err := app.db.InsertScope(s); err != nil {
		log.Fatal(err)
	}

	scopes, err := app.db.GetAllScopes()
	if err != nil {
		log.Fatal(err)
	}

	for _, scope := range scopes {

		// Start notification process
		notificationChannel := make(chan models.Notification, 1000)
		go app.Notify(notificationChannel)

		// Run GAU & Waymore and consume output in real-time
		gauChan := make(chan string, 1000)
		waymoreChan := make(chan string, 1000)
		go app.ConsumeRealTime(gauChan, target, scope.FirstRun, models.Gau)
		modules.RunModule(models.Gau, scope, gauChan)
		modules.RunModule(models.Waymore, scope, waymoreChan)

		// TODO: Run nmap

		// Set scope's first_run to false after intial scan
		if scope.FirstRun == true {
			newScope := scope
			newScope.FirstRun = false
			app.db.UpdateScope(newScope)
		}
	}

}
