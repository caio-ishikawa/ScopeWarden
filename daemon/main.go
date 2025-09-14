package main

import (
	//"github.com/caio-ishikawa/target-tracker/daemon/modules"
	//"github.com/caio-ishikawa/target-tracker/shared/models"
	//"github.com/google/uuid"
	"log"
	//"time"
)

func main() {
	app, err := NewDaemon()
	if err != nil {
		log.Fatal(err)
	}

	// targetUUID := uuid.NewString()
	// target := models.Target{
	// 	UUID: targetUUID,
	// 	Name: "NASA",
	// }
	// if err := app.db.InsertTarget(target); err != nil {
	// 	log.Fatal(err)
	// }

	// s := models.Scope{
	// 	UUID:       uuid.NewString(),
	// 	TargetUUID: targetUUID,
	// 	URL:        "nasa.gov",
	// 	FirstRun:   true,
	// }

	// if err := app.db.InsertScope(s); err != nil {
	// 	log.Fatal(err)
	// }

	go app.RunDaemon()
	app.api.StartAPI()
}
