package api

import (
	"encoding/json"
	"fmt"
	"github.com/caio-ishikawa/target-tracker/shared/models"
	"github.com/caio-ishikawa/target-tracker/shared/store"
	"github.com/google/uuid"
	"log"
	"net/http"
	"strconv"
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
	}

	query := r.URL.Query()

	targetUUID := query.Get("target_uuid")
	if targetUUID == "" {
		http.Error(w, "No target UUID", http.StatusBadRequest)
		return
	}

	var limit int
	var offset int

	limitQuery := query.Get("limit")
	limit, err := strconv.Atoi(limitQuery)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid limit value %s", limitQuery), http.StatusBadRequest)
		return
	}
	offsetQuery := query.Get("offset")
	offset, err = strconv.Atoi(offsetQuery)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid offset value %s", offsetQuery), http.StatusBadRequest)
		return
	}

	domains, err := a.db.GetDomainsPerTarget(limit, offset, targetUUID)
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

func (a API) getScopes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}

	query := r.URL.Query()

	targetUUID := query.Get("target_name")
	if targetUUID == "" {
		http.Error(w, "No target name", http.StatusBadRequest)
	}

	scopes, err := a.db.GetAllScopes()
	if err != nil {
		http.Error(w, "Failed to get all scopes", http.StatusInternalServerError)
	}

	resStruct := models.ScopeListResponse{
		Scopes: scopes,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resStruct); err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
	}
}

func (a API) insertScope(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}

	var req models.InsertScopeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
	}
	defer r.Body.Close()

	target, err := a.db.GetTargetByName(req.TargetName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not get target by name %s", req.TargetName), http.StatusBadRequest)
	}

	scopeUUID := uuid.NewString()

	scope := models.Scope{
		UUID:       scopeUUID,
		TargetUUID: target.UUID,
		URL:        req.URL,
		FirstRun:   true,
	}

	if err = a.db.InsertScope(scope); err != nil {
		http.Error(w, "Failed to insert scope", http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusCreated)
}

func (a API) StartAPI() error {
	http.HandleFunc("/domains", a.getDomains)
	http.HandleFunc("/scopes", a.getScopes)
	http.HandleFunc("/insert_scope", a.insertScope)

	log.Println("API listening on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		return fmt.Errorf("Error server API on port :8080, %w", err)
	}

	return nil
}
