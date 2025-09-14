package main

import (
	//"github.com/caio-ishikawa/target-tracker/daemon/modules"
	"github.com/caio-ishikawa/target-tracker/shared/models"
	"github.com/google/uuid"
	"log"
	//"time"
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

	go app.RunDaemon()
	app.api.StartAPI()

	//for {
	//	// Avoid running scan before timeout
	//	// if time.Since(app.stats.LastScanEnded) < time.Duration(app.config.ScanTimeout)*time.Hour {
	//	// 	continue
	//	// }

	//	if app.stats.IsRunning {
	//		log.Println("Previous scan ran for longer than scan timeout - CONSIDER ADJUSTING TIMEOUT")
	//		continue
	//	}

	//	// Set stats for current scan
	//	log.Println("Starting scan")
	//	app.stats.ScanBegin = time.Now()
	//	app.stats.IsRunning = true
	//	if err := app.db.UpdateDaemonStats(app.stats); err != nil {
	//		log.Printf("%s", err.Error())
	//	}

	//	// Start of actual daemon
	//	scopes, err := app.db.GetAllScopes()
	//	if err != nil {
	//		log.Fatal(err)
	//	}

	//	for _, scope := range scopes {

	//		// Start notification process
	//		notificationChannel := make(chan models.Notification, 1000)
	//		go app.Notify(notificationChannel)

	//		// Run GAU & Waymore and consume output in real-time
	//		gauChan := make(chan string, 1000)
	//		waymoreChan := make(chan string, 1000)
	//		go app.ConsumeRealTime(gauChan, target, scope.FirstRun, models.Gau)
	//		modules.RunModule(models.Gau, scope, gauChan)
	//		modules.RunModule(models.Waymore, scope, waymoreChan)

	//		// TODO: Run nmap

	//		// Set scope's first_run to false after intial scan
	//		if scope.FirstRun == true {
	//			newScope := scope
	//			newScope.FirstRun = false
	//			app.db.UpdateScope(newScope)
	//		}
	//	}

	//	// Reset stats when scan ends
	//	log.Printf("Scan ended - duration: %s", app.stats.ScanTime.String())
	//	app.stats.LastScanEnded = time.Now()
	//	app.stats.ScanTime = time.Now().Sub(time.Now())
	//	app.stats.IsRunning = false
	//	if err := app.db.UpdateDaemonStats(app.stats); err != nil {
	//		log.Printf("%s", err.Error())
	//	}
	//}

}
