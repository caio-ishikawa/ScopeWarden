package main

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/caio-ishikawa/target-tracker/daemon/api"
	"github.com/caio-ishikawa/target-tracker/daemon/models"
	"github.com/caio-ishikawa/target-tracker/daemon/modules"
	"github.com/caio-ishikawa/target-tracker/daemon/store"
	"github.com/google/uuid"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Daemon struct {
	db       store.Database
	api      api.API
	telegram modules.TelegramClient
	stats    models.DaemonStats
	config   modules.DaemonConfig
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

	config, err := modules.NewDaemonConfig()
	if err != nil {
		return Daemon{}, fmt.Errorf("Failed to create Daemon: %w", err)
	}

	return Daemon{
		db:       db,
		api:      api,
		telegram: telegram,
		config:   config,
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
		// Avoid running scan before timeout
		if a.stats.LastScanEnded != nil {
			if time.Since(*a.stats.LastScanEnded) < time.Duration(a.config.Global.Schedule)*time.Hour {
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
			defer close(outputChan)
			go a.ConsumeRealTime(outputChan, target, scope.FirstRun)
			for _, module := range a.config.Tools {
				log.Printf("Running tool %s", module.ID)

				if err := modules.RunModule(module, scope.URL, outputChan); err != nil {
					log.Printf("Failed to run module %s: %s", module.ID, err.Error())
				}
			}

			// Set scope's first_run to false after intial scan
			if scope.FirstRun == true {
				newScope := scope
				newScope.FirstRun = false
				a.db.UpdateScope(newScope)
			}

			// Get all domains for the target after running domain scan
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

// Updates current scan time and returns daemon stats
func (a *Daemon) Stats() models.DaemonStats {
	a.stats.ScanTime = time.Since(a.stats.ScanBegin)
	return a.stats
}

// Consume real-time output of a tool
func (a *Daemon) ConsumeRealTime(inputChan chan modules.ToolOutput, resource models.Resource, firstRun bool) {
	httpClient := http.Client{
		Timeout: 5 * time.Second,
	}

	// Generic input processing in case more tables are added later
	for input := range inputChan {
		switch input.Tool.Table {
		case models.DomainTable:
			if err := a.processURLOutput(httpClient, input, firstRun, resource); err != nil {
				log.Println(err.Error())
			}
		default:
			log.Printf("Failed to consume data for tool %s: Unknown table: %s", input.Tool.ID, input.Tool.Table)
		}
	}
}

// Process URL output for a tool (parses, inserts/updates DB, notifies)
func (a *Daemon) processURLOutput(httpClient http.Client, input modules.ToolOutput, firstRun bool, resource models.Resource) error {
	log.Printf("Processing URL %s", input.Output)

	parsed, err := url.Parse(input.Output)
	if err != nil {
		return fmt.Errorf("Failed to parse URL %s - SKIPPING", input.Output)
	}

	// Only process successful requests to avoid noise in DB
	res, err := httpClient.Get(input.Output)
	if err != nil {
		log.Printf("Failed to make request to URL %s: %s", input.Output, err.Error())
		return nil
	}

	// Increment found URLs
	a.stats.TotalFoundURLs += 1
	if err := a.db.UpdateDaemonStats(a.stats); err != nil {
		return fmt.Errorf("Failed to process url %s: %w", input.Output, err)
	}

	var statusCode int
	if res == nil {
		statusCode = 0
	} else {
		statusCode = res.StatusCode
	}

	var baseURL string
	if parsed.Scheme == "https" || parsed.Scheme == "http" {
		baseURL = fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host)
	} else if parsed.Scheme == "" {
		baseURL = parsed.Path
	}

	domain, err := a.db.GetDomainByURL(baseURL)
	if err != nil {
		return fmt.Errorf("Failed to process URL: %w", err)
	}

	notification := models.Notification{
		TargetName: resource.ResourceName(),
		Type:       models.URLUpdate,
		Content:    baseURL,
	}

	if domain == nil {
		a.stats.TotalNewURLs += 1
		if err := a.db.UpdateDaemonStats(a.stats); err != nil {
			return fmt.Errorf("Failed to process url %s: %w", input.Output, err)
		}

		toInsert := models.Domain{
			UUID:       uuid.NewString(),
			TargetUUID: resource.ResourceUUID(),
			URL:        baseURL,
			IPAddress:  "", // TODO: This needs to be updated when running the port scanner
			StatusCode: statusCode,
		}

		if err := a.db.InsertDomainRecord(toInsert); err != nil {
			return fmt.Errorf("Failed to process url %s: %w", input.Output, err)
		}

		// Notify only if this is not the first run on the scope
		if !firstRun {
			if err = a.telegram.SendMessage(notification); err != nil {
				return fmt.Errorf("Failed to process url %s: %w", input.Output, err)
			}
		}

		a.portScan(input.Tool, toInsert, firstRun, resource)

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
			TargetUUID:  resource.ResourceUUID(),
			URL:         baseURL,
			IPAddress:   domain.IPAddress,
			StatusCode:  statusCode,
			LastUpdated: time.Now().String(),
		}

		if err := a.db.UpdateDomainRecord(toInsert); err != nil {
			return fmt.Errorf("Failed to process url %s: %w", input.Output, err)
		}

		if err = a.telegram.SendMessage(notification); err != nil {
			return fmt.Errorf("Failed to process url %s: %w", input.Output, err)
		}

		a.portScan(input.Tool, toInsert, firstRun, resource)
	}

	return nil
}

// Run port scan synchronously and process results for a specific domain
func (a *Daemon) portScan(tool modules.Tool, domain models.Domain, firstRun bool, resource models.Resource) {
	if tool.PortScanConfig.Run {
		portScanRes, err := modules.RunPortScan(tool, domain, firstRun)
		if err != nil {
			log.Printf("Failed to run port scan for domain %s: %s", domain.URL, err.Error())
		}

		if err := a.processPortScan(portScanRes, domain, firstRun, resource); err != nil {
			log.Printf("Failed to process port scan result: %s", err.Error())
		}
	}
}

// Process port scan output for (parses, inserts/updates DB, notifies)
func (a *Daemon) processPortScan(scanRes []byte, domain models.Domain, firstRun bool, resource models.Resource) error {
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

		log.Printf("Processing port %s for domain %s", scanner.Text(), domain.URL)

		var port int
		var proto models.Protocol
		var state models.PortState

		split := strings.Fields(scanner.Text())
		for i, s := range split {
			// Get port & protocol
			if i == 0 {
				portProtoSplit := strings.Split(s, "/")
				if len(portProtoSplit) != 2 {
					return fmt.Errorf("Failed to parse port and protocol from port scan result %s", s)
				}

				// Get port number
				portInt, err := strconv.Atoi(strings.TrimSpace(portProtoSplit[0]))
				if err != nil {
					return fmt.Errorf("Failed to parse port number %s", portProtoSplit[0])
				}
				port = portInt

				// Get port protocol
				switch strings.TrimSpace(portProtoSplit[1]) {
				case string(models.TCP):
					proto = models.TCP
				case string(models.UDP):
					proto = models.UDP
				case string(models.SCTP):
					proto = models.SCTP
				default:
					return fmt.Errorf("Failed to parse port protocol %s", portProtoSplit[1])
				}
			}

			// Get port state
			if i == 1 {
				switch strings.TrimSpace(s) {
				case string(models.Open):
					state = models.Open
				case string(models.Filtered):
					state = models.Filtered
				case string(models.Closed):
					state = models.Closed
				default:
					return fmt.Errorf("Failed to parse port state %s", s)
				}
			}
		}

		foundPort := models.Port{
			DomainUUID:  domain.UUID,
			Port:        port,
			Protocol:    proto,
			State:       state,
			LastUpdated: time.Now().String(),
		}

		notification := models.Notification{
			TargetName: resource.ResourceName(),
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

			// Update port changes & notify
			foundPort.UUID = existingPort.UUID
			if err := a.db.UpdatePort(foundPort); err != nil {
				return fmt.Errorf("Failed to update port for domain %s", domain.URL)
			}

			// Notify
			if err = a.telegram.SendMessage(notification); err != nil {
				log.Printf("%s", err.Error())
			}

			continue
		}

		// Insert new port scan result
		foundPort.UUID = uuid.NewString()
		if err := a.db.InsertPort(foundPort); err != nil {
			return fmt.Errorf("Failed to insert new port for domain %s", domain.URL)
		}

		// Notify if not first run
		if !firstRun {
			if err = a.telegram.SendMessage(notification); err != nil {
				log.Printf("%s", err.Error())
			}
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
