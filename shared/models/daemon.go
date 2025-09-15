package models

import (
	"time"
)

type DaemonStats struct {
	// Represents the total number of found URLs in this current scan
	TotalFoundURLs int `json:"total_found_urls"`
	// Represents all new URLs (includes URLs that already existed but were down previously)
	TotalNewURLs int `json:"total_new_urls"`
	// Represents the total number of port found in this current scan
	TotalFoundPorts int `json:"total_found_ports"`
	// Represents all new port founds in this current scan (includes ports that were closed/filtered and are now open)
	TotalNewPorts int `json:"total_new_ports"`
	// Represents how long the current scan has been running
	ScanTime time.Duration `json:"scan_time"`
	// Represents the time the current scan began
	ScanBegin time.Time `json:"scan_begin"`
	// Represents the time the last scan ended
	LastScanEnded *time.Time `json:"last_scan_ended,omitempty"`
	// Represents whether or not the scan is running currently
	IsRunning bool `json:"is_running"`
}
