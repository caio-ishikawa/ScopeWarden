package main

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/caio-ishikawa/scopewarden/daemon/api"
	"github.com/caio-ishikawa/scopewarden/daemon/modules"
	"github.com/caio-ishikawa/scopewarden/daemon/store"
	"github.com/caio-ishikawa/scopewarden/shared/models"
	"github.com/google/uuid"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

type ConcurrencySettings struct {
	domainSemaphore     chan struct{}
	bruteForceSemaphore chan struct{}
	bruteForceWg        *sync.WaitGroup
}

type Daemon struct {
	db                      store.Database
	api                     api.API
	concurrencySettings     ConcurrencySettings
	currentlyProcessingURLs sync.Map
	telegram                modules.TelegramClient
	stats                   models.DaemonStats
	config                  modules.DaemonConfig
}

func NewDaemon() (Daemon, error) {
	db, err := store.Init()
	if err != nil {
		return Daemon{}, fmt.Errorf("Failed to start DB client: %w", err)
	}

	api, err := api.NewAPI()
	if err != nil {
		return Daemon{}, fmt.Errorf("Failed to create API: %w", err)
	}

	config, err := modules.NewDaemonConfig()
	if err != nil {
		return Daemon{}, fmt.Errorf("Failed to create Daemon: %w", err)
	}

	var telegram modules.TelegramClient
	if config.Global.Notify {
		telegram, err = modules.NewTelegramClient()
		if err != nil {
			return Daemon{}, fmt.Errorf("Failed to start Telegram client: %w", err)
		}
	}

	maxConcurrentDomainProcessing := 10
	maxConcurrentBruteForces := 5
	if config.Global.Intensity == modules.Aggressive {
		maxConcurrentDomainProcessing = 20
		maxConcurrentBruteForces = 10
	}

	var bruteForceWg sync.WaitGroup
	concurrencySettings := ConcurrencySettings{
		domainSemaphore:     make(chan struct{}, maxConcurrentDomainProcessing),
		bruteForceSemaphore: make(chan struct{}, maxConcurrentBruteForces),
		bruteForceWg:        &bruteForceWg,
	}

	return Daemon{
		db:                  db,
		api:                 api,
		concurrencySettings: concurrencySettings,
		telegram:            telegram,
		config:              config,
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

func (a *Daemon) RunDaemon() {
	for {
		// Update config before every scan
		config, err := modules.NewDaemonConfig()
		if err != nil {
			log.Printf("Failed to update daemon config: %s", err.Error())
			time.Sleep(10)
		}
		a.config = config

		// We should get stats from the db
		// Do not scan unless it's been enough time since the last scan ended or if the last scan took longer than the schedule
		if a.stats.LastScanEnded != nil &&
			(time.Since(*a.stats.LastScanEnded) < time.Duration(a.config.Global.Schedule)*time.Hour ||
				time.Since(a.stats.ScanBegin) < time.Duration(a.config.Global.Schedule)*time.Hour) {
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

		// Set initial stats
		a.stats.UUID = uuid.NewString()
		a.stats.ScanBegin = time.Now()
		a.stats.IsRunning = true
		// Insert first scan stats
		if err := a.db.InsertDaemonStats(a.stats); err != nil {
			log.Fatalf("Failed to insert initial daemon stats: %s", err.Error())
		}

		log.Printf("Starting scan %s", a.stats.UUID)

		// Set stats for current scan
		if err := a.db.UpdateDaemonStats(a.stats); err != nil {
			log.Printf("%s", err.Error())
		}

		// Run scope scans
		a.scanScopes(scopes)
		// Wait for brute force scans to end
		a.concurrencySettings.bruteForceWg.Wait()

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

func (a *Daemon) scanScopes(scopes []models.Scope) {
	for _, scope := range scopes {
		target, err := a.db.GetTarget(scope.TargetUUID)
		if err != nil {
			log.Printf("Failed to get target with UUID %s", scope.TargetUUID)
			continue
		}

		if target == nil {
			log.Printf("Failed to scan scope: Could not find target with UUID %s", scope.TargetUUID)
			continue
		}

		// Set up output channel & run modules
		outputChan := make(chan modules.ToolOutput, 1000)
		go a.ConsumeRealTime(models.DomainTable, outputChan, *target, scope)

		for _, tool := range a.config.Tools {
			log.Printf("Running tool %s", tool.ID)

			if err := modules.RunModule(tool, scope.URL, outputChan); err != nil {
				log.Printf("Failed to run module %s: %s", tool.ID, err.Error())
			}

			log.Printf("Finished scanning scope %s with tool %s", scope.URL, tool.ID)
		}

		// Close output channel once all scans have finished
		close(outputChan)
		log.Printf("Finished scanning scope %s", scope.URL)

		// Set scope's first_run to false after intial scan
		if scope.FirstRun == true {
			newScope := scope
			newScope.FirstRun = false
			if err := a.db.UpdateScope(newScope); err != nil {
				log.Printf("Failed to update scope after scan: %s", err.Error())
			}
		}
	}
}

// Updates current scan time and returns daemon stats
func (a *Daemon) Stats() models.DaemonStats {
	if a.stats.IsRunning {
		a.stats.ScanTime = time.Since(a.stats.ScanBegin)
	}

	return a.stats
}

// Consumes real time output of a tool. Handles both URL outputs and the output of the brute force attempts
func (a *Daemon) ConsumeRealTime(table models.Table, inputChan chan modules.ToolOutput, target models.TargetTables, scope models.Scope) {
	for input := range inputChan {
		switch table {
		case models.DomainTable:
			a.concurrencySettings.domainSemaphore <- struct{}{}

			httpClient := http.Client{
				Timeout: 5 * time.Second,
			}

			go a.processURLOutput(httpClient, input, scope, target)
		case models.BruteforcedTable:
			if err := a.processBruteForceResults(input, target, scope); err != nil {
				log.Println(err.Error())
			}
		default:
			log.Printf("Failed to consume output: Invalid table %s", table)
		}
	}
}

// Process URL output for a tool (parses, inserts/updates DB, notifies)
func (a *Daemon) processURLOutput(httpClient http.Client, input modules.ToolOutput, scope models.Scope, target models.TargetTables) {
	defer func() { <-a.concurrencySettings.domainSemaphore }()

	baseURL, err := parseURL(input.Output)
	if err != nil {
		// Log and ignore invalid URLs
		log.Printf("Failed to parse URL: %s - SKIPPING", err.Error())
		return
	}

	// Update daemon URL stat
	a.stats.TotalFoundURLs += 1
	if err := a.db.UpdateDaemonStats(a.stats); err != nil {
		log.Printf("Failed to update stats %s: %s", a.stats.UUID, err.Error())
		return
	}

	// Return early if URL is currently being processed by another routine
	_, loaded := a.currentlyProcessingURLs.Load(baseURL)
	if loaded {
		return
	}

	// Store URL being processed in sync.Map
	a.currentlyProcessingURLs.Store(baseURL, struct{}{})

	// Make sure to delete URL from sync.Map before returning
	defer a.currentlyProcessingURLs.Delete(baseURL)

	// Check if domain exists early in the processing
	existingDomain, err := a.db.GetDomainByURL(baseURL)
	if err != nil {
		log.Printf("Failed to get domain: %s", err.Error())
		return
	}

	// Return early if domain was already processed in this scan
	if existingDomain != nil && existingDomain.ScanUUID == a.stats.UUID {
		return
	}

	foundDomain := models.Domain{
		TargetUUID: target.GetUUID(),
		ScanUUID:   a.stats.UUID,
		URL:        baseURL,
	}

	// Make GET request and get fingerprinted technologies
	responseDetails, err := getResDetails(httpClient, baseURL)
	foundDomain.StatusCode = responseDetails.statusCode
	if err != nil {
		// Insert domain where request was unsuccessful, so that next iterations can exit early if it already processed the domain in this scan
		if existingDomain == nil {
			foundDomain.UUID = uuid.NewString()
			if err := a.db.InsertDomainRecord(foundDomain); err != nil {
				log.Printf("Failed to insert non-working domain %s: %s", baseURL, err.Error())
				return
			}

			return
		}

		foundDomain.UUID = existingDomain.UUID
		if err := a.db.UpdateDomainRecord(foundDomain); err != nil {
			log.Printf("Failed to update non-working domain %s: %s", baseURL, err.Error())
		}

		log.Printf("Failed to get response: %s", err.Error())

		// Return early if request was unsuccessful
		return
	}

	// Set found domain's status code
	foundDomain.StatusCode = responseDetails.statusCode

	log.Printf("Processing URL %s", baseURL)

	if len(responseDetails.technologies) > 0 {
		log.Printf("Found fingerprints for %s: %s", baseURL, strings.Join(responseDetails.technologies, ", "))
	}

	notification := models.Notification{
		TargetName: target.GetNotificationName(),
		Type:       models.URLUpdate,
		Content:    baseURL,
	}

	if existingDomain == nil {
		foundDomain.UUID = uuid.NewString()
		if err := a.insertNewFoundDomain(foundDomain, notification, scope.FirstRun); err != nil {
			log.Printf("Failed to insert new domain: %s", err.Error())
			return
		}

		a.portScan(input.Tool, foundDomain, scope, target)
		a.bruteForce(input.Tool, foundDomain, scope, responseDetails.technologies)

		return
	}

	// Update and notify if staus code has changed since last run
	if existingDomain.StatusCode != responseDetails.statusCode {
		foundDomain.UUID = existingDomain.UUID
		foundDomain.LastUpdated = time.Now().String()

		if err := a.updateExistingDomain(foundDomain, *existingDomain, notification); err != nil {
			log.Printf("Failed to update existing domain: %s", err.Error())
			return
		}

		a.portScan(input.Tool, foundDomain, scope, target)
		a.bruteForce(input.Tool, foundDomain, scope, responseDetails.technologies)
	}

	return
}

func (a *Daemon) insertNewFoundDomain(newDomain models.Domain, notification models.Notification, firstRun bool) error {
	a.stats.TotalNewURLs += 1
	if err := a.db.UpdateDaemonStats(a.stats); err != nil {
		return fmt.Errorf("Failed to process url %s: %w", newDomain.URL, err)
	}

	if err := a.db.InsertDomainRecord(newDomain); err != nil {
		return fmt.Errorf("Failed to process url %s: %w", newDomain.URL, err)
	}

	// Notify only if this is not the first run on the scope
	if !firstRun && a.config.Global.Notify {
		if err := a.telegram.SendMessage(notification); err != nil {
			return fmt.Errorf("Failed to process url %s: %w", newDomain.URL, err)
		}
	}

	return nil
}

func (a *Daemon) updateExistingDomain(newDomain models.Domain, existingDomain models.Domain, notification models.Notification) error {
	if err := a.db.UpdateDomainRecord(newDomain); err != nil {
		return fmt.Errorf("Failed to process url %s: %w", newDomain.URL, err)
	}

	if existingDomain.StatusCode == 0 && newDomain.StatusCode != 0 {
		a.stats.TotalNewURLs += 1
		if err := a.db.UpdateDaemonStats(a.stats); err != nil {
			log.Printf("%s", err.Error())
		}

		if a.config.Global.Notify {
			if err := a.telegram.SendMessage(notification); err != nil {
				return fmt.Errorf("Failed to process url %s: %w", newDomain.URL, err)
			}
		}
	}

	return nil
}

// Run port scan synchronously and process results for a specific domain
func (a *Daemon) portScan(tool modules.Tool, domain models.Domain, scope models.Scope, target models.TargetTables) {
	if tool.PortScanConfig.Run {
		// TODO: Account for overrides
		portScanRes, err := modules.RunPortScan(tool, domain, scope)
		if err != nil {
			log.Printf("Failed to run port scan for domain %s: %s", domain.URL, err.Error())
		}

		if err := a.processPortScan(portScanRes, domain, scope.FirstRun, target); err != nil {
			log.Printf("Failed to process port scan result: %s", err.Error())
		}
	}
}

// Process port scan output for (parses, inserts/updates DB, notifies)
func (a *Daemon) processPortScan(scanRes []byte, domain models.Domain, firstRun bool, target models.TargetTables) error {
	resBuf := bytes.NewBuffer(scanRes)
	scanner := bufio.NewScanner(resBuf)
	re, err := regexp.Compile(modules.PortRegex)
	if err != nil {
		return fmt.Errorf("Failed to compile regex for port scan: %w", err)
	}

	for scanner.Scan() {
		if match := re.MatchString(scanner.Text()); !match {
			continue
		}

		a.stats.TotalFoundPorts += 1

		portData, err := parsePortScanLine(scanner.Text())
		if err != nil {
			return err
		}

		foundPort := models.Port{
			DomainUUID:  domain.UUID,
			Port:        portData.portNum,
			Protocol:    portData.portProtocol,
			State:       portData.portState,
			LastUpdated: time.Now().String(),
		}

		notification := models.Notification{
			TargetName: target.GetNotificationName(),
			Type:       models.URLUpdate,
			Content:    foundPort.FormatPortResult(),
		}

		existingPort, err := a.db.GetPortByNumberAndDomain(foundPort.Port, foundPort.DomainUUID)
		if err != nil {
			return fmt.Errorf("Failed to process port scan results: %w", err)
		}

		if existingPort != nil {
			// Ignore update if port state is the same as last scan
			if foundPort.State == existingPort.State {
				continue
			}

			a.stats.TotalNewPorts += 1

			// Update port changes & notify
			foundPort.UUID = existingPort.UUID
			if err := a.db.UpdatePort(foundPort); err != nil {
				return fmt.Errorf("Failed to update port for domain %s", domain.URL)
			}

			// Notify
			if a.config.Global.Notify {
				if err = a.telegram.SendMessage(notification); err != nil {
					log.Printf("%s", err.Error())
				}
			}

			continue
		}

		a.stats.TotalNewPorts += 1

		// Insert new port scan result
		foundPort.UUID = uuid.NewString()
		if err := a.db.InsertPort(foundPort); err != nil {
			return fmt.Errorf("Failed to insert new port for domain %s", domain.URL)
		}

		// Notify if not first run
		if !firstRun && a.config.Global.Notify {
			if err = a.telegram.SendMessage(notification); err != nil {
				log.Printf("%s", err.Error())
			}
		}

		if err := a.db.UpdateDaemonStats(a.stats); err != nil {
			log.Printf("%s", err.Error())
		}

	}

	return nil
}

// Runs brute force command asynchronously
func (a *Daemon) bruteForce(
	tool modules.Tool, domain models.Domain, scope models.Scope, technologies []string,
) {
	// TODO: Account for overrides
	if tool.BruteForceConfig.Run {
		a.concurrencySettings.bruteForceWg.Add(1)

		outputChan := make(chan modules.ToolOutput, 1000)

		go a.ConsumeRealTime(models.BruteforcedTable, outputChan, domain, scope)
		go modules.RunBruteForce(
			a.concurrencySettings.bruteForceWg,
			a.concurrencySettings.bruteForceSemaphore,
			tool,
			domain,
			scope,
			technologies,
			outputChan,
		)
	}
}

func (a *Daemon) processBruteForceResults(input modules.ToolOutput, domain models.TargetTables, scope models.Scope) error {
	log.Printf("Processing bruteforced result %s", input.Output)

	newBruteForced := models.BruteForced{
		DomainUUID:  domain.GetUUID(),
		Path:        input.Output,
		FirstRun:    scope.FirstRun,
		LastUpdated: time.Now().String(),
	}

	notification := models.Notification{
		TargetName: domain.GetNotificationName(),
		Type:       models.URLUpdate,
		Content:    input.Output,
	}

	existingBruteForced, err := a.db.GetBruteForcedByPath(input.Output, domain.GetUUID())
	if err != nil {
		return fmt.Errorf("Failed to get existing bruteforced path: %w", err)
	}

	if existingBruteForced == nil {
		newBruteForced.UUID = uuid.NewString()
		if err := a.db.InsertBruteForced(newBruteForced); err != nil {
			return fmt.Errorf("Failed to process bruteforced path: %w", err)
		}

		if !scope.FirstRun && a.config.Global.Notify {
			if err := a.telegram.SendMessage(notification); err != nil {
				log.Printf("Failed to notify brute force result: %s", err.Error())
			}
		}

		return nil
	}

	newBruteForced.UUID = existingBruteForced.UUID
	if err := a.db.UpdateBruteForced(newBruteForced); err != nil {
		return fmt.Errorf("Failed to process bruteforced path: %w", err)
	}

	if a.config.Global.Notify {
		if err := a.telegram.SendMessage(notification); err != nil {
			log.Printf("Failed to notify brute force result: %s", err.Error())
		}
	}

	return nil
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
