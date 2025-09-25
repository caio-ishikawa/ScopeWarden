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
	db                  store.Database
	api                 api.API
	currentScanUUID     string
	concurrencySettings ConcurrencySettings
	telegram            modules.TelegramClient
	stats               models.DaemonStats
	config              modules.DaemonConfig
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
		maxConcurrentDomainProcessing = 30
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
		currentScanUUID:     uuid.NewString(),
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
	if err := a.db.InsertDaemonStats(a.stats); err != nil {
		log.Fatalf("Failed to insert initial daemon stats")
	}
	for {
		// Update config before every scan
		config, err := modules.NewDaemonConfig()
		if err != nil {
			log.Printf("Failed to update daemon config: %s", err.Error())
			time.Sleep(10)
		}
		a.config = config

		// Do not scan unless it's been enough time since the last scan ended or if the last scan took longer than the schedule
		if a.stats.LastScanEnded != nil &&
			(time.Since(*a.stats.LastScanEnded) < time.Duration(a.config.Global.Schedule)*time.Hour ||
				time.Since(a.stats.ScanBegin) < time.Duration(a.config.Global.Schedule)*time.Hour) {
			continue
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

		// Set new scan UUID
		a.currentScanUUID = uuid.NewString()
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
		go a.ConsumeRealTime(models.DomainTable, outputChan, *target, scope.FirstRun)
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
	a.stats.ScanTime = time.Since(a.stats.ScanBegin)
	return a.stats
}

// Consumes real time output of a tool. Handles both URL outputs and the output of the brute force attempts. Since brute force attempts
// can take some time, they are run concurrently and are limited to 5 processes. This function waits for the processing of the brute force
// scans to finish before returning.
func (a *Daemon) ConsumeRealTime(table models.Table, inputChan chan modules.ToolOutput, target models.TargetTables, firstRun bool) {
	for input := range inputChan {
		switch table {
		case models.DomainTable:
			a.concurrencySettings.domainSemaphore <- struct{}{}

			httpClient := http.Client{
				Timeout: 5 * time.Second,
			}

			go a.processURLOutput(httpClient, input, firstRun, target)
		case models.BruteforcedTable:
			if err := a.processBruteForceResults(input, target, firstRun); err != nil {
				log.Println(err.Error())
			}
		default:
			log.Printf("Failed to consume output: Invalid table %s", table)
		}
	}

	// Attepmt to delete all unsuccessful domains found from previous scan
	if err := a.db.DeleteUnsuccessfulDomains(); err != nil {
		log.Printf("Failed to delete unsuccessful domains: %s", err.Error())
	}
}

// Process URL output for a tool (parses, inserts/updates DB, notifies)
func (a *Daemon) processURLOutput(httpClient http.Client, input modules.ToolOutput, firstRun bool, target models.TargetTables) {
	defer func() { <-a.concurrencySettings.domainSemaphore }()

	baseURL, err := parseURL(input.Output)
	if err != nil {
		// Log and ignore invalid URLs
		log.Printf("Failed to parse URL: %s - SKIPPING", err.Error())
		return
	}

	a.stats.TotalFoundURLs += 1
	if err := a.db.UpdateDaemonStats(a.stats); err != nil {
		log.Printf("Failed to update stats: %s", err.Error())
		return
	}

	// Check if domain exists early in the processing
	existingDomain, err := a.db.GetDomainByURL(baseURL)
	if err != nil {
		log.Printf("Failed to get domain: %s", err.Error())
		return
	}

	// Return early if domain was already processed in this scan
	if existingDomain != nil && existingDomain.ScanUUID == a.currentScanUUID {
		if existingDomain.StatusCode == 0 {
			log.Printf("Already processed invalid domain %s - SKIPPING", input.Output)
			return
		}

		log.Printf("Already processed URL %s - SKIPPING", baseURL)
		return
	}

	foundDomain := models.Domain{
		TargetUUID: target.GetUUID(),
		ScanUUID:   a.currentScanUUID,
		URL:        baseURL,
	}

	// Make GET request and get fingerprinted technologies
	responseDetails, err := getResDetails(httpClient, baseURL)
	if err != nil {
		log.Printf("Failed to get response for domain %s: %s", baseURL, err.Error())

		// Insert domain where request was unsuccessful, so that next iterations can exit early if it already processed the domain in this scan
		foundDomain.UUID = uuid.NewString()
		if err := a.db.InsertDomainRecord(foundDomain); err != nil {
			log.Printf("Failed to insert non-working domain %s: %s", baseURL, err.Error())
			return
		}

		log.Printf("Failed to make request to domain %s - SKIPPING", baseURL)

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
		if err := a.insertNewFoundDomain(foundDomain, notification, firstRun); err != nil {
			log.Printf("Failed to insert new domain: %s", err.Error())
			return
		}

		a.portScan(input.Tool, foundDomain, firstRun, target)
		a.bruteForce(input.Tool, foundDomain, firstRun, responseDetails.technologies)

		return
	}

	// Update and notify if staus code has changed since last run
	if existingDomain.StatusCode != responseDetails.statusCode || existingDomain.StatusCode == 0 {
		foundDomain.UUID = existingDomain.UUID
		foundDomain.LastUpdated = time.Now().String()

		if err := a.updateExistingDomain(foundDomain, notification); err != nil {
			log.Printf("Failed to update existing domain: %s", err.Error())
			return
		}

		a.portScan(input.Tool, foundDomain, firstRun, target)
		a.bruteForce(input.Tool, foundDomain, firstRun, responseDetails.technologies)
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

func (a *Daemon) updateExistingDomain(newDomain models.Domain, notification models.Notification) error {
	a.stats.TotalNewURLs += 1
	if err := a.db.UpdateDaemonStats(a.stats); err != nil {
		log.Printf("%s", err.Error())
	}

	if err := a.db.UpdateDomainRecord(newDomain); err != nil {
		return fmt.Errorf("Failed to process url %s: %w", newDomain.URL, err)
	}

	if a.config.Global.Notify {
		if err := a.telegram.SendMessage(notification); err != nil {
			return fmt.Errorf("Failed to process url %s: %w", newDomain.URL, err)
		}
	}

	return nil
}

// Run port scan synchronously and process results for a specific domain
func (a *Daemon) portScan(tool modules.Tool, domain models.Domain, firstRun bool, target models.TargetTables) {
	if tool.PortScanConfig.Run {
		portScanRes, err := modules.RunPortScan(tool, domain, firstRun)
		if err != nil {
			log.Printf("Failed to run port scan for domain %s: %s", domain.URL, err.Error())
		}

		if err := a.processPortScan(portScanRes, domain, firstRun, target); err != nil {
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

		log.Printf("Processing port %s for domain %s", scanner.Text(), domain.URL)

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
	tool modules.Tool, domain models.Domain, firstRun bool, technologies []string,
) {
	if tool.BruteForceConfig.Run {
		a.concurrencySettings.bruteForceWg.Add(1)

		outputChan := make(chan modules.ToolOutput, 1000)

		go a.ConsumeRealTime(models.BruteforcedTable, outputChan, domain, firstRun)
		go modules.RunBruteForce(
			a.concurrencySettings.bruteForceWg,
			a.concurrencySettings.bruteForceSemaphore,
			tool,
			domain,
			firstRun,
			technologies,
			outputChan,
		)
	}
}

func (a *Daemon) processBruteForceResults(input modules.ToolOutput, domain models.TargetTables, firstRun bool) error {
	log.Printf("Processing bruteforced result %s", input.Output)

	newBruteForced := models.BruteForced{
		DomainUUID:  domain.GetUUID(),
		Path:        input.Output,
		FirstRun:    firstRun,
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

		if !firstRun && a.config.Global.Notify {
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
