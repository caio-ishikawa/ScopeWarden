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
	UUID string
	Name string
}

type Scope struct {
	UUID       string
	TargetUUID string
	URL        string
	FirstRun   bool
}

type Domain struct {
	UUID        string
	TargetUUID  string
	URL         string
	IPAddress   string
	QueryParams string
	StatusCode  int
	LastUpdated string
}

type Port struct {
	UUID        string
	DomainUUID  string
	Port        int
	State       PortState
	LastUpdated string
}
