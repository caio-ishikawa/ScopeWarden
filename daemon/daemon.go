package main

import (
	"fmt"
	"github.com/caio-ishikawa/target-tracker/daemon/api"
	"github.com/caio-ishikawa/target-tracker/daemon/modules"
	"github.com/caio-ishikawa/target-tracker/shared/models"
	"github.com/caio-ishikawa/target-tracker/shared/store"
	"github.com/google/uuid"
	"log"
	"net/http"
	"net/url"
	"time"
)

type DaemonConfig struct {
	// Represents the amount of time to wait before running the next scan in hours
	ScanTimeout int
}

func NewDaemonConfig(scanTimeout int) DaemonConfig {
	return DaemonConfig{
		ScanTimeout: scanTimeout,
	}
}

type Daemon struct {
	db       store.Database
	api      api.API
	telegram modules.TelegramClient
	stats    models.DaemonStats
	config   DaemonConfig
}

func NewDaemon() (Daemon, error) {
	db, err := store.Init()
	if err != nil {
		return Daemon{}, fmt.Errorf("Failed to start DB client: %w", err)
	}

	telegram, err := modules.NewTelegramClient()
	if err != nil {
		return Daemon{}, fmt.Errorf("Failed to start Telegram client: %w", err)
	}

	api, err := api.NewAPI()
	if err != nil {
		return Daemon{}, fmt.Errorf("Failed to create API: %w", err)
	}

	return Daemon{
		db:       db,
		api:      api,
		telegram: telegram,
		config:   NewDaemonConfig(12),
		stats: models.DaemonStats{
			TotalFoundURLs:  0,
			TotalNewURLs:    0,
			TotalFoundPorts: 0,
			TotalNewPorts:   0,
			ScanBegin:       time.Now(),
			ScanTime:        time.Duration(0),
			LastScanEnded:   nil,
			IsRunning:       false,
		},
	}, nil
}

func (a *Daemon) Stats() models.DaemonStats {
	a.stats.ScanTime = time.Since(a.stats.ScanBegin)
	return a.stats
}

func (a *Daemon) Notify(notifyChan chan models.Notification) {
	for input := range notifyChan {
		if err := a.telegram.SendMessage(input); err != nil {
			log.Printf("[TELEGRAM] Failed to send message via Telegram client: %s", err.Error())
		}
	}
}

// Consume real-time output of command
func (a *Daemon) ConsumeRealTime(inputChan chan string, target models.Target, firstRun bool, tool models.Module) {
	httpClient := http.Client{
		Timeout: 5 * time.Second,
	}

	for input := range inputChan {
		if err := a.parseURLOutput(httpClient, input, firstRun, tool, target); err != nil {
			log.Println(err.Error())
		}
	}
}

func (a *Daemon) parseURLOutput(httpClient http.Client, input string, firstRun bool, tool models.Module, target models.Target) error {
	parsed, err := url.Parse(input)
	if err != nil {
		return fmt.Errorf("[%s] Failed to parse URL %s - SKIPPING", tool, input)
	}

	// Only process successful requests to avoid noise in DB
	res, err := httpClient.Get(input)
	if err != nil {
		log.Printf("[%s] Failed to make request to URL %s: %s - SKIPPING", tool, input, err.Error())
		return nil
	}

	// Increment found URLs
	a.stats.TotalFoundURLs += 1
	if err := a.db.UpdateDaemonStats(a.stats); err != nil {
		log.Printf("%s", err.Error())
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
		a.stats.TotalNewURLs += 1
		if err := a.db.UpdateDaemonStats(a.stats); err != nil {
			log.Printf("%s", err.Error())
		}

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
			if err = a.telegram.SendMessage(notification); err != nil {
				log.Printf("[%s] %s", tool, err.Error())
			}
		}

		return nil
	}

	// Update and notify if staus code has changed since last run
	if domain.StatusCode != res.StatusCode || domain.StatusCode == 0 {
		a.stats.TotalNewURLs += 1
		if err := a.db.UpdateDaemonStats(a.stats); err != nil {
			log.Printf("%s", err.Error())
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
			return fmt.Errorf("[%s] %w - SKIPPING", tool, err)
		}

		if err = a.telegram.SendMessage(notification); err != nil {
			return fmt.Errorf("[%s] %w", tool, err)
		}
	}

	return nil
}

func (a *Daemon) RunDaemon() {
	if err := a.db.InsertDaemonStats(a.stats); err != nil {
		log.Fatalf("Failed to insert initial daemon stats")
	}
	for {
		// Avoid running scan before timeout
		if a.stats.LastScanEnded != nil {
			if time.Since(*a.stats.LastScanEnded) < time.Duration(a.config.ScanTimeout)*time.Hour {
				continue
			}
		}

		if a.stats.IsRunning {
			log.Println("Previous scan ran for longer than scan timeout - CONSIDER ADJUSTING TIMEOUT")
			continue
		}

		// Start of actual daemon
		scopes, err := a.db.GetAllScopes()
		if err != nil {
			log.Printf("Failed to get all scopes: %s", err.Error())
			time.Sleep(10 * time.Second)
			continue
		}

		if len(scopes) == 0 {
			log.Println("No scopes found - continuing")
			time.Sleep(10 * time.Second)
			continue
		}

		// Set stats for current scan
		log.Println("Starting scan")
		a.stats.ScanBegin = time.Now()
		a.stats.IsRunning = true
		if err := a.db.UpdateDaemonStats(a.stats); err != nil {
			log.Printf("%s", err.Error())
		}

		for _, scope := range scopes {
			targetPtr, err := a.db.GetTarget(scope.TargetUUID)
			if err != nil {
				log.Printf("Failed to get target with UUID %s", scope.TargetUUID)
				continue
			}

			if targetPtr == nil {
				log.Printf("Could not find target with UUID %s", scope.TargetUUID)
				continue
			}

			// Dereference target
			target := *targetPtr

			// Start notification process
			notificationChannel := make(chan models.Notification, 1000)
			go a.Notify(notificationChannel)

			// Run GAU & Waymore and consume output in real-time
			gauChan := make(chan string, 1000)
			waymoreChan := make(chan string, 1000)
			go a.ConsumeRealTime(gauChan, target, scope.FirstRun, models.Gau)
			modules.RunModule(models.Gau, scope, gauChan)
			modules.RunModule(models.Waymore, scope, waymoreChan)

			// TODO: Run nmap

			// Set scope's first_run to false after intial scan
			if scope.FirstRun == true {
				newScope := scope
				newScope.FirstRun = false
				a.db.UpdateScope(newScope)
			}
		}

		// Reset stats when scan ends
		log.Printf("Scan ended - duration: %s", a.stats.ScanTime.String())
		now := time.Now()
		a.stats.ScanTime = time.Duration(0)
		a.stats.LastScanEnded = &now
		a.stats.IsRunning = false
		if err := a.db.UpdateDaemonStats(a.stats); err != nil {
			log.Printf("%s", err.Error())
		}
	}
}

func (a *Daemon) TestTelegram() {
	testNotification := models.Notification{
		TargetName: "TEST",
		Type:       "TEST",
		Content:    "TEST",
	}
	if err := a.telegram.SendMessage(testNotification); err != nil {
		log.Printf("[%s] %s", models.Telegram, err.Error())
	}
}
