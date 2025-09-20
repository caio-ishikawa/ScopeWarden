package models

import "fmt"

type Table string
type PortState string
type Protocol string

const (
	TargetTable      Table = "target"
	ScopeTable       Table = "scope"
	DomainTable      Table = "domain"
	PortTable        Table = "port"
	DaemonStatsTable Table = "daemon_stats"

	Open     PortState = "open"
	Filtered PortState = "filtered"
	Closed   PortState = "closed"

	UDP  Protocol = "udp"
	TCP  Protocol = "tcp"
	SCTP Protocol = "sctp"
)

type Module string

const (
	Gau      Module = "GAU"
	Waymore  Module = "WAYMORE"
	Telegram Module = "TELEGRAM"
)

type Resource interface {
	ResourceUUID() string
	ResourceName() string
}

type Target struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
}

func (t *Target) ResourceUUID() string {
	return t.UUID
}

func (t *Target) ResourceName() string {
	return t.Name
}

type Domain struct {
	UUID        string `json:"uuid"`
	TargetUUID  string `json:"target_uuid"`
	URL         string `json:"url"`
	IPAddress   string `json:"ip_address"`
	StatusCode  int    `json:"status_code"`
	LastUpdated string `json:"last_updated"`
}

func (d *Domain) ResourceUUID() string {
	return d.TargetUUID
}

// TODO: Implement
func (d *Domain) ResourceName() string {
	return d.TargetUUID
}

type Scope struct {
	UUID             string `json:"uuid"`
	TargetUUID       string `json:"target_uuid"`
	URL              string `json:"url"`
	AcceptSubdomains bool   `json:"accept_subdomains"`
	FirstRun         bool   `json:"first_run"`
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
