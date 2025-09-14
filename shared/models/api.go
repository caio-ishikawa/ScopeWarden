package models

type DomainListResponse struct {
	Domains []Domain `json:"domains"`
}

type ScopeListResponse struct {
	Scopes []Scope `json:"scope"`
}

type InsertScopeRequest struct {
	TargetName       string `json:"target_name"`
	URL              string `json:"url"`
	AcceptSubdomains bool   `json:"accept_subdomains"`
}

type InsertTargetRequest struct {
	Name string `json:"name"`
}
