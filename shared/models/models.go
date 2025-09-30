package models

import "fmt"

type Table string
type PortState string
type Protocol string
type DomainSortBy string

const ()

const (
	TargetTable      Table = "target"
	ScopeTable       Table = "scope"
	DomainTable      Table = "domain"
	PortTable        Table = "port"
	BruteforcedTable Table = "bruteforced"
	DaemonStatsTable Table = "daemon_stats"

	Open     PortState = "open"
	Filtered PortState = "filtered"
	Closed   PortState = "closed"

	UDP  Protocol = "udp"
	TCP  Protocol = "tcp"
	SCTP Protocol = "sctp"

	SortPorts       DomainSortBy = "count_ports"
	SortBruteForced DomainSortBy = "count_bruteforced"
	SortNone        DomainSortBy = ""
)

type Module string

const (
	Telegram Module = "TELEGRAM"
)

// Represents tables in which scans can insert to (currently domain & target)
type TargetTables interface {
	GetUUID() string
	GetNotificationName() string
}

type Target struct {
	UUID    string `json:"uuid"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

func (t Target) GetUUID() string {
	return t.UUID
}

func (t Target) GetNotificationName() string {
	return t.Name
}

type Domain struct {
	UUID        string `json:"uuid"`
	TargetUUID  string `json:"target_uuid"`
	ScanUUID    string `json:"scan_uuid"`
	URL         string `json:"url"`
	IPAddress   string `json:"ip_address"`
	StatusCode  int    `json:"status_code"`
	FirstRun    bool   `json:"first_run"`
	LastUpdated string `json:"last_updated"`
}

func (d Domain) GetUUID() string {
	return d.UUID
}

func (d Domain) GetNotificationName() string {
	return d.URL
}

type Scope struct {
	UUID       string `json:"uuid"`
	TargetUUID string `json:"target_uuid"`
	URL        string `json:"url"`
	FirstRun   bool   `json:"first_run"`
}

type BruteForced struct {
	UUID        string `json:"uuid"`
	DomainUUID  string `json:"domain_uuid"`
	Path        string `json:"path"`
	FirstRun    bool   `json:"first_run"`
	LastUpdated string `json:"last_updated"`
}

type Port struct {
	UUID        string    `json:"uuid"`
	DomainUUID  string    `json:"domain_uuid"`
	Port        int       `json:"port"`
	Protocol    Protocol  `json:"protocol"`
	State       PortState `json:"state"`
	LastUpdated string    `json:"last_updated"`
}

func (p *Port) FormatPortResult() string {
	return fmt.Sprintf("%v %s", p.Port, p.State)
}

func ParseProtocolString(protoStr string) (Protocol, error) {
	switch protoStr {
	case "udp":
		return UDP, nil
	case "tcp":
		return TCP, nil
	case "sctp":
		return SCTP, nil
	default:
		return UDP, fmt.Errorf("Could not convert %s to Protocol", protoStr)
	}
}

func ParsePortState(portStateStr string) (PortState, error) {
	switch portStateStr {
	case "open":
		return Open, nil
	case "closed":
		return Closed, nil
	case "filtered":
		return Filtered, nil
	default:
		return Closed, fmt.Errorf("Could not convert %s to PortState", portStateStr)
	}
}
