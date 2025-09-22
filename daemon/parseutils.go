package main

import (
	"fmt"
	"github.com/caio-ishikawa/scopewarden/shared/models"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	wappalyzer "github.com/projectdiscovery/wappalyzergo"
)

type ResponseDetails struct {
	successful   bool
	technologies []string
	statusCode   int
}

type ParsedPortData struct {
	portNum      int
	portProtocol models.Protocol
	portState    models.PortState
}

func parseURL(urlStr string) (string, error) {
	parsed, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}

	var baseURL string
	if parsed.Scheme == "https" || parsed.Scheme == "http" {
		baseURL = fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host)
	} else if parsed.Scheme == "" {
		baseURL = parsed.Path
	}

	return baseURL, nil
}

func getResDetails(httpClient http.Client, url string) (ResponseDetails, error) {
	res, err := httpClient.Get(url)
	if err != nil {
		return ResponseDetails{successful: false}, nil
	}
	defer res.Body.Close()

	var statusCode int
	if res == nil {
		statusCode = 0
	} else {
		statusCode = res.StatusCode
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return ResponseDetails{}, fmt.Errorf("Failed to read response body for %s: %w", url, err)
	}

	wappalyzerClient, err := wappalyzer.New()
	if err != nil {
		return ResponseDetails{}, fmt.Errorf("Failed to start wappalyzer client: %w", err)
	}

	technologies := make([]string, 0)
	fingerprints := wappalyzerClient.Fingerprint(res.Header, data)
	for fingerprintKey := range fingerprints {
		technologies = append(technologies, strings.ToLower(fingerprintKey))
	}

	return ResponseDetails{
		successful:   true,
		technologies: technologies,
		statusCode:   statusCode,
	}, nil
}

func parsePortScanLine(line string) (ParsedPortData, error) {
	var port int
	var proto models.Protocol
	var state models.PortState

	split := strings.Fields(line)
	for i, s := range split {
		// Get port & protocol
		if i == 0 {
			portNum, protocol, err := parsePortScanPortProtocol(s)
			if err != nil {
				return ParsedPortData{}, err
			}

			port = portNum
			proto = protocol
		}

		// Get port state
		if i == 1 {
			portState, err := parsePortState(s)
			if err != nil {
				return ParsedPortData{}, err
			}

			state = portState
		}
	}

	return ParsedPortData{
		portNum:      port,
		portProtocol: proto,
		portState:    state,
	}, nil
}

func parsePortScanPortProtocol(toParse string) (int, models.Protocol, error) {
	portProtoSplit := strings.Split(toParse, "/")
	if len(portProtoSplit) != 2 {
		return 0, "", fmt.Errorf("Failed to parse port and protocol from port scan result %s", toParse)
	}

	// Get port number
	portInt, err := strconv.Atoi(strings.TrimSpace(portProtoSplit[0]))
	if err != nil {
		return 0, "", fmt.Errorf("Failed to parse port number %s", portProtoSplit[0])
	}

	port := portInt

	var proto models.Protocol

	// Get port protocol
	switch strings.TrimSpace(portProtoSplit[1]) {
	case string(models.TCP):
		proto = models.TCP
	case string(models.UDP):
		proto = models.UDP
	case string(models.SCTP):
		proto = models.SCTP
	default:
		return 0, "", fmt.Errorf("Failed to parse port protocol %s", portProtoSplit[1])
	}

	return port, proto, nil
}

func parsePortState(toParse string) (models.PortState, error) {
	var state models.PortState

	switch strings.TrimSpace(toParse) {
	case string(models.Open):
		state = models.Open
	case string(models.Filtered):
		state = models.Filtered
	case string(models.Closed):
		state = models.Closed
	default:
		return "", fmt.Errorf("Failed to parse port state %s", toParse)
	}

	return state, nil
}
