package api

import (
	"encoding/json"
	"fmt"
	"github.com/caio-ishikawa/target-tracker/daemon/store"
	"github.com/caio-ishikawa/target-tracker/shared/models"
	"github.com/google/uuid"
	"log"
	"net/http"
	"strconv"
	"time"
)

type API struct {
	db store.Database
}

func NewAPI() (API, error) {
	db, err := store.Init()
	if err != nil {
		return API{}, fmt.Errorf("Failed to start DB client: %w", err)
	}

	return API{
		db: db,
	}, nil
}

func (a API) getDomains(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query()

	targetUUID := query.Get("target_uuid")
	if targetUUID == "" {
		log.Println("[API] No target uuid or domain url")
		http.Error(w, "No target UUID", http.StatusBadRequest)
		return
	}

	var limit int
	var offset int

	limitQuery := query.Get("limit")
	limit, err := strconv.Atoi(limitQuery)
	if err != nil {
		log.Println(err)
		http.Error(w, fmt.Sprintf("Invalid limit value %s", limitQuery), http.StatusBadRequest)
		return
	}
	offsetQuery := query.Get("offset")
	offset, err = strconv.Atoi(offsetQuery)
	if err != nil {
		log.Println(err)
		http.Error(w, fmt.Sprintf("Invalid offset value %s", offsetQuery), http.StatusBadRequest)
		return
	}

	domains, err := a.db.GetDomainsByTarget(limit, offset, targetUUID)
	resStruct := models.DomainListResponse{
		Domains: domains,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resStruct); err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
		return
	}
}

func (a API) getTargetByName(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query()

	targetName := query.Get("name")
	if targetName == "" {
		http.Error(w, "No target name", http.StatusBadRequest)
		return
	}

	target, err := a.db.GetTargetByName(targetName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get target with name %s", targetName), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(target); err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
		return
	}
}

func (a API) getScopes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query()

	targetUUID := query.Get("target_name")
	if targetUUID == "" {
		http.Error(w, "No target name", http.StatusBadRequest)
		return
	}

	scopes, err := a.db.GetAllScopes()
	if err != nil {
		http.Error(w, "Failed to get all scopes", http.StatusInternalServerError)
		return
	}

	resStruct := models.ScopeListResponse{
		Scopes: scopes,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resStruct); err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
		return
	}
}

func (a API) insertScope(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.InsertScopeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	target, err := a.db.GetTargetByName(req.TargetName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not get target by name %s", req.TargetName), http.StatusBadRequest)
		return
	}

	scopeUUID := uuid.NewString()

	scope := models.Scope{
		UUID:             scopeUUID,
		TargetUUID:       target.UUID,
		URL:              req.URL,
		AcceptSubdomains: req.AcceptSubdomains,
		FirstRun:         true,
	}

	if err = a.db.InsertScope(scope); err != nil {
		http.Error(w, "Failed to insert scope", http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusCreated)
}

func (a API) insertTarget(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.InsertTargetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Println(err.Error())
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	targetUUID := uuid.NewString()

	target := models.Target{
		UUID: targetUUID,
		Name: req.Name,
	}

	if err := a.db.InsertTarget(target); err != nil {
		log.Println(err.Error())
		http.Error(w, "Failed to insert target", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (a API) getPortsByDomain(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query()

	domainURL := query.Get("domain_url")
	if domainURL == "" {
		log.Println("no domain url")
		http.Error(w, "No domain url", http.StatusBadRequest)
		return
	}

	domain, err := a.db.GetDomainByURL(domainURL)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, fmt.Sprintf("Could not get domain by URL: %s", err.Error()), http.StatusInternalServerError)
	}

	if domain == nil {
		http.Error(w, fmt.Sprintf("Could not find domain by URL %s", domainURL), http.StatusNotFound)
	}

	scopes, err := a.db.GetPortByDomain(domain.UUID)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, fmt.Sprintf("Failed to get all ports for domain %s", domain.UUID), http.StatusInternalServerError)
		return
	}

	resStruct := models.PortListResponse{
		Ports: scopes,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resStruct); err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
		return
	}
}

func (a API) getBruteForcedPathsByDomain(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query()

	domainURL := query.Get("domain_url")
	if domainURL == "" {
		log.Println("no domain url")
		http.Error(w, "No domain url", http.StatusBadRequest)
		return
	}

	domain, err := a.db.GetDomainByURL(domainURL)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, fmt.Sprintf("Could not get domain by URL: %s", err.Error()), http.StatusInternalServerError)
	}

	if domain == nil {
		errMsg := fmt.Sprintf("Could not find domain by URL %s", domainURL)
		log.Println(errMsg)
		http.Error(w, errMsg, http.StatusNotFound)
	}

	bruteForcedPaths, err := a.db.GetBruteForcedByDomain(domain.UUID)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, fmt.Sprintf("Failed to get all ports for domain %s", domain.UUID), http.StatusInternalServerError)
		return
	}

	resStruct := models.BruteForcedListResponse{
		BruteForcedPaths: bruteForcedPaths,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resStruct); err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
		return
	}
}

func (a API) getStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats, err := a.db.GetStats()
	if err != nil {
		log.Println(err)
		http.Error(w, "Failed to get all scopes", http.StatusInternalServerError)
		return
	}

	scanTime := time.Since(stats.ScanBegin)

	var lastScanEnded *string
	if stats.LastScanEnded != nil {
		s := *stats.LastScanEnded
		l := s.String()
		lastScanEnded = &l
	}

	statsRes := models.StatsResponse{
		TotalFoundURLs:  stats.TotalFoundURLs,
		TotalNewURLs:    stats.TotalNewURLs,
		TotalFoundPorts: stats.TotalFoundPorts,
		TotalNewPorts:   stats.TotalNewPorts,
		ScanTime:        scanTime.String(),
		ScanBegin:       stats.ScanBegin.String(),
		LastScanEnded:   lastScanEnded,
		IsRunning:       stats.IsRunning,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(statsRes); err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
		return
	}
}

func (a API) StartAPI() error {
	http.HandleFunc("/domains", a.getDomains)
	http.HandleFunc("/scopes", a.getScopes)
	http.HandleFunc("/insert_scope", a.insertScope)
	http.HandleFunc("/insert_target", a.insertTarget)
	http.HandleFunc("/target", a.getTargetByName)
	http.HandleFunc("/stats", a.getStats)
	http.HandleFunc("/ports", a.getPortsByDomain)
	http.HandleFunc("/bruteforced", a.getBruteForcedPathsByDomain)

	log.Println("API listening on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		return fmt.Errorf("Error server API on port :8080, %w", err)
	}

	return nil
}
