package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/caio-ishikawa/scopewarden/shared/models"
	"net/http"
	"net/url"
	"strconv"

	"github.com/charmbracelet/bubbles/table"
)

// Gets domains and creates map for domain to associated rows (ports & bruteforced)
func (c *CLI) GetDomainRows(substr *string) ([]table.Row, error) {
	var search string
	if substr != nil {
		search = *substr
	}

	res, err := GetDomainsByTarget(c.targetUUID, c.domainOffset, c.sortBy, search, c.tableLimit)
	if err != nil {
		return nil, fmt.Errorf("Failed to get domains rows: %w", err)
	}

	output := make([]table.Row, 0)
	for _, domain := range res.Domains {
		var row PerDomainRow

		domainRow := table.Row{
			strconv.Itoa(domain.StatusCode),
			strconv.Itoa(domain.PortCount),
			strconv.Itoa(domain.BruteForcedCount),
			domain.URL,
		}

		for _, port := range domain.Ports {
			row.Port = append(
				row.Port,
				table.Row{
					strconv.Itoa(port.Port),
					string(port.Protocol),
					string(port.State),
				},
			)
		}

		for _, bruteForced := range domain.BruteForced {
			row.BruteForced = append(
				row.BruteForced,
				table.Row{
					bruteForced.Path,
				},
			)
		}

		output = append(output, domainRow)
		c.domainMap[domain.URL] = row
	}

	return output, nil
}

func (c *CLI) GetPortRows(ports []models.Port) ([]table.Row, error) {
	var rows []table.Row
	for _, port := range ports {
		portStr := strconv.Itoa(port.Port)
		if portStr == "" {
			return nil, fmt.Errorf("Failed to get rows for port: Could not convert port number to string %v", port.Port)
		}

		rows = append(rows, table.Row{portStr, string(port.Protocol), string(port.State)})
	}

	return rows, nil
}

func (c *CLI) GetBruteForcedRows(assets []models.BruteForced) ([]table.Row, error) {
	var rows []table.Row
	for _, asset := range assets {
		rows = append(rows, table.Row{asset.Path})
	}

	return rows, nil
}

func GetDomainsByTarget(target string, offset int, sortBy models.DomainSortBy, substr string, limit int) (models.DomainListResponse, error) {
	url := fmt.Sprintf("%s/domains?target_uuid=%s&limit=%v&offset=%v&sort_by=%s&url=%s", apiURL, target, limit, offset, sortBy, substr)
	res, err := http.Get(url)
	if err != nil {
		return models.DomainListResponse{}, fmt.Errorf("Could not get domains for target %s: %w", target, err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return models.DomainListResponse{}, parseError(res)
	}

	var ret models.DomainListResponse
	if err = json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return models.DomainListResponse{}, fmt.Errorf("Failed to decode API response: %w", err)
	}

	return ret, nil
}

func GetPortsByDomain(domainURL string) ([]models.Port, error) {
	param := url.Values{}
	param.Add("domain_url", domainURL)
	url := fmt.Sprintf("%s/ports?%s", apiURL, param.Encode())
	res, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("Could not get domains for domain %s: %w", domainURL, err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, parseError(res)
	}

	var ret models.PortListResponse
	if err = json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, fmt.Errorf("Failed to decode API response: %w", err)
	}

	return ret.Ports, nil
}

func GetBruteForcedByDomain(domainURL string, offset, limit int) ([]models.BruteForced, error) {
	param := url.Values{}
	param.Add("domain_url", domainURL)

	url := fmt.Sprintf("%s/bruteforced?%s&limit=%v&offset=%v", apiURL, param.Encode(), limit, offset)
	res, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("Could not get domains for domain %s: %w", domainURL, err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, parseError(res)
	}

	var ret models.BruteForcedListResponse
	if err = json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, fmt.Errorf("Failed to decode API response: %w", err)
	}

	return ret.BruteForcedPaths, nil
}

func GetTargetByName(target string) (models.Target, error) {
	res, err := http.Get(fmt.Sprintf("%s/target?name=%s", apiURL, target))
	if err != nil {
		return models.Target{}, fmt.Errorf("Could not get domains for target %s: %w", target, err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return models.Target{}, parseError(res)
	}

	var ret models.Target
	if err = json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return models.Target{}, fmt.Errorf("Failed to decode API response: %w", err)
	}

	return ret, nil
}

func GetStats() ([]models.StatsResponse, error) {
	res, err := http.Get(fmt.Sprintf("%s/stats", apiURL))
	if err != nil {
		return nil, fmt.Errorf("Could not get stats: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, parseError(res)
	}

	var ret []models.StatsResponse
	if err = json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, fmt.Errorf("Failed to decode API response: %w", err)
	}

	return ret, nil
}

func InsertScope(scopes ScopeInsert) error {
	for _, scopeURL := range scopes.ScopeURLs {
		reqBody := models.InsertScopeRequest{
			TargetName: scopes.TargetName,
			URL:        scopeURL,
		}

		body, err := json.Marshal(&reqBody)
		if err != nil {
			return fmt.Errorf("Failed to marshal scope request body: %w", err)
		}

		res, err := http.Post(fmt.Sprintf("%s/insert_scope", apiURL), "application/json", bytes.NewBuffer(body))
		if err != nil {
			return fmt.Errorf("Could not get stats: %w", err)
		}

		if res.StatusCode != http.StatusCreated {
			return parseError(res)
		}
	}

	return nil
}

func EnableDisableTarget(scopeName string, enabled bool) error {
	req, err := http.NewRequest(http.MethodPatch, fmt.Sprintf("%s/update_target?name=%s&enable_disable=%v", apiURL, scopeName, enabled), nil)
	if err != nil {
		return fmt.Errorf("Could update scope: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	//res, err := http.
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("Could not update scope: %w", err)
	}

	if res.StatusCode != http.StatusNoContent {
		return parseError(res)
	}

	return nil
}

func InsertTarget(target string) error {
	reqBody := models.InsertTargetRequest{
		Name: target,
	}

	body, err := json.Marshal(&reqBody)
	if err != nil {
		return fmt.Errorf("Failed to marshal scope request body: %w", err)
	}

	res, err := http.Post(fmt.Sprintf("%s/insert_target", apiURL), "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("Could not insert target: %w", err)
	}

	if res.StatusCode != http.StatusCreated {
		return parseError(res)
	}

	return nil
}

func parseError(res *http.Response) error {
	var msg models.ErrorResponse
	if err := json.NewDecoder(res.Body).Decode(&msg); err != nil {
		return fmt.Errorf("Failed to parse error response")
	}

	return fmt.Errorf("Error: %s ", msg.Message)
}
