package main

import (
	"fmt"
	"github.com/caio-ishikawa/target-tracker/models"
	"github.com/caio-ishikawa/target-tracker/modules"
	"github.com/caio-ishikawa/target-tracker/store"
	"github.com/google/uuid"
	"log"
	"net/http"
	"net/url"
	"time"
)

type App struct {
	db       store.Database
	telegram modules.TelegramClient
}

func main() {
	db, err := store.Init()
	if err != nil {
		panic(err)
	}

	telegram, err := modules.NewTelegramClient()
	if err != nil {
		panic(err)
	}

	app := App{
		db:       db,
		telegram: telegram,
	}

	targetUUID := uuid.NewString()
	target := models.Target{
		UUID: targetUUID,
		Name: "NASA",
	}
	if err := app.db.InsertTarget(target); err != nil {
		fmt.Println(err.Error())
		panic("woops")
	}

	scope := models.Scope{
		UUID:       uuid.NewString(),
		TargetUUID: targetUUID,
		URL:        "nasa.gov",
		FirstRun:   true,
	}

	// TODO: Get all scopes to run the modules with

	// TODO:
	// Start notification process
	//notificationChannel := make(chan models.Notification, 1000)
	//go app.Notify(notificationChannel)

	// Start real-time consumer process
	gauChan := make(chan string, 1000)
	// TODO: this probably needs some kind of wait group to avoid running a both waymore and gau at the same time
	// and getting blocked by some of the sources (and nmap should only run after both)
	go app.ConsumeRealTime(gauChan, target, scope.FirstRun, models.Gau)

	// Run GAU
	modules.RunGau(scope, gauChan)
	// TODO: Run Waymore
	// TODO: Run nmap
}

func (a App) Notify(notifyChan chan models.Notification) {
	for input := range notifyChan {
		if err := a.telegram.SendMessage(input); err != nil {
			log.Printf("[TELEGRAM] Failed to send message via Telegram client: %s", err.Error())
		}
	}
}

// Consume real-time output of command
func (a App) ConsumeRealTime(inputChan chan string, target models.Target, firstRun bool, tool models.Module) {
	httpClient := http.Client{
		Timeout: 5 * time.Second,
	}

	for input := range inputChan {
		parsed, err := url.Parse(input)
		if err != nil {
			log.Printf("[%s] Failed to parse URL %s - SKIPPING", tool, input)
			continue
		}

		res, err := httpClient.Get(input)
		if err != nil {
			log.Printf("[%s] Failed to make request to URL %s: %s", tool, input, err.Error())
		}

		var statusCode int
		if res == nil {
			statusCode = 0
		} else {
			statusCode = res.StatusCode
		}

		var baseURL string
		if parsed.Scheme == "https" || parsed.Scheme == "http" {
			baseURL = fmt.Sprintf("%s://%s%s", parsed.Scheme, parsed.Host, parsed.Path)
		} else if parsed.Scheme == "" {
			baseURL = parsed.Path
		}

		domain, err := a.db.GetDomainByURL(baseURL)
		if err != nil {
			log.Printf("[%s] %s - SKIPPING", tool, err.Error())
			continue
		}

		// TODO:
		// notification := models.Notification{
		// 	TargetName: target.Name,
		// 	Type:       models.URLUpdate,
		// 	Content:    baseURL,
		// }

		if domain == nil {
			toInsert := models.Domain{
				UUID:        uuid.NewString(),
				TargetUUID:  target.UUID,
				URL:         baseURL,
				IPAddress:   "", // TODO: This needs to be updated when running the port scanner
				QueryParams: parsed.RawQuery,
				StatusCode:  statusCode,
			}
			if err := a.db.InsertDomainRecord(toInsert); err != nil {
				log.Printf("[%s] %s - SKIPPING", tool, err)
				continue
			}

			// Notify only if this is not the first run on the scope
			if !firstRun {
				log.Println("SHOULD NOTIFY")
				// TODO:
				// if err = a.telegram.SendMessage(notification); err != nil {
				// 	log.Printf("[%s] %s", tool, err.Error())
				// }
			}

			continue
		}

		toInsert := models.Domain{
			UUID:        domain.UUID,
			TargetUUID:  target.UUID,
			URL:         baseURL,
			IPAddress:   domain.IPAddress,
			QueryParams: parsed.RawQuery,
			StatusCode:  statusCode,
			LastUpdated: time.Now().String(),
		}
		if err := a.db.UpdateDomainRecord(toInsert); err != nil {
			log.Printf("[%s] %s - SKIPPING", tool, err.Error())
			continue
		}

		// Notify if old domain became active
		if domain.StatusCode != res.StatusCode || domain.StatusCode == 0 {
			log.Println("SHOULD NOTIFY")
			// TODO:
			// if err = a.telegram.SendMessage(notification); err != nil {
			// 	log.Printf("[%s] %s", tool, err.Error())
			// }
		}
	}
}

func (a App) ConsumeFromOutputFile(fileName string) {
}
