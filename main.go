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
		if err := a.ParseURLOutput(httpClient, input, firstRun, tool, target); err != nil {
			log.Println(err.Error())
		}
	}
}

func (a App) ParseURLOutput(httpClient http.Client, input string, firstRun bool, tool models.Module, target models.Target) error {
	parsed, err := url.Parse(input)
	if err != nil {
		return fmt.Errorf("[%s] Failed to parse URL %s - SKIPPING", tool, input)
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
		return fmt.Errorf("[%s] %w - SKIPPING", tool, err)
	}

	notification := models.Notification{
		TargetName: target.Name,
		Type:       models.URLUpdate,
		Content:    baseURL,
	}

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
			return fmt.Errorf("[%s] %q - SKIPPING", tool, err)
		}

		// Notify only if this is not the first run on the scope
		if !firstRun {
			log.Println("SHOULD NOTIFY")
			if err = a.telegram.SendMessage(notification); err != nil {
				log.Printf("[%s] %s", tool, err.Error())
			}
		}

		return nil
	}

	// Update and notify if staus code has changed since last run
	if domain.StatusCode != res.StatusCode || domain.StatusCode == 0 {
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
			return fmt.Errorf("[%s] %w - SKIPPING", tool, err)
		}

		log.Println("SHOULD NOTIFY")
		if err = a.telegram.SendMessage(notification); err != nil {
			return fmt.Errorf("[%s] %w", tool, err)
		}
	}

	return nil
}
