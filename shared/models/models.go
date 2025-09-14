package models

type PortState string

const (
	Open     PortState = "open"
	Filtered PortState = "filtered"
	Closed   PortState = "closed"
)

type Module string

const (
	Gau      Module = "GAU"
	Waymore  Module = "WAYMORE"
	Telegram Module = "TELEGRAM"
)

type Target struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
}

type Scope struct {
	UUID             string `json:"uuid"`
	TargetUUID       string `json:"target_uuid"`
	URL              string `json:"url"`
	AcceptSubdomains bool   `json:"accept_subdomains"`
	FirstRun         bool   `json:"first_run"`
}

type Domain struct {
	UUID        string `json:"uuid"`
	TargetUUID  string `json:"target_uuid"`
	URL         string `json:"url"`
	IPAddress   string `json:"ip_address"`
	QueryParams string `json:"query_params"`
	StatusCode  int    `json:"status_code"`
	LastUpdated string `json:"last_updated"`
}

type Port struct {
	UUID        string    `json:"uuid"`
	DomainUUID  string    `json:"domain_uuid"`
	Port        int       `json:"port"`
	State       PortState `json:"state"`
	LastUpdated string    `json:"last_updated"`
}
