package models

type DomainWithCount struct {
	UUID             string        `json:"uuid"`
	TargetUUID       string        `json:"target_uuid"`
	ScanUUID         string        `json:"scan_uuid"`
	URL              string        `json:"url"`
	IPAddress        string        `json:"ip_address"`
	StatusCode       int           `json:"status_code"`
	FirstRun         bool          `json:"first_run"`
	LastUpdated      string        `json:"last_updated"`
	PortCount        int           `json:"port_count"`
	BruteForcedCount int           `json:"brute_forced_count"`
	BruteForced      []BruteForced `json:"brute_forced"`
	Ports            []Port        `json:"ports"`
}

type DomainListResponse struct {
	Domains []DomainWithCount `json:"domains"`
	Total   int               `json:"total"`
	Page    int               `json:"page"`
}

type ScopeListResponse struct {
	Scopes []Scope `json:"scope"`
}

type PortListResponse struct {
	Ports []Port `json:"port"`
}

type BruteForcedListResponse struct {
	BruteForcedPaths []BruteForced `json:"bruteforced"`
}

type InsertScopeRequest struct {
	TargetName string `json:"target_name"`
	URL        string `json:"url"`
}

type InsertTargetRequest struct {
	Name string `json:"name"`
}

type StatsResponse struct {
	// Represents the total number of found URLs in this current scan
	TotalFoundURLs int `json:"total_found_urls"`
	// Represents all new URLs (includes URLs that already existed but were down previously)
	TotalNewURLs int `json:"total_new_urls"`
	// Represents the total number of port found in this current scan
	TotalFoundPorts int `json:"total_found_ports"`
	// Represents all new port founds in this current scan (includes ports that were closed/filtered and are now open)
	TotalNewPorts int `json:"total_new_ports"`
	// Represents how long the current scan has been running
	ScanTime string `json:"scan_time"`
	// Represents the time the current scan began
	ScanBegin string `json:"scan_begin"`
	// Represents the time the current scan began
	LastScanEnded *string `json:"last_scan_ended,omitempty"`
	// Represents whether or not the scan is running currently
	IsRunning bool `json:"is_running"`
}

type ErrorResponse struct {
	Message string `json:"message"`
}
