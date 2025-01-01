package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"jarvis-agent/configs/config"

	"github.com/0xrawsec/golang-utils/log"
)

type APIClientPlugin struct {
	config     *config.Config
	stop       chan struct{}
	httpClient *http.Client
}

// New is the plugin constructor required by the plugin system
func New() config.Plugin {
	return &APIClientPlugin{
		stop: make(chan struct{}),
	}
}

func (p *APIClientPlugin) Name() string {
	return "APIClient"
}

func (p *APIClientPlugin) Init(cfg *config.Config) error {
	p.config = cfg
	p.httpClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // Note: Configure based on your needs
			},
		},
	}
	return nil
}

func (p *APIClientPlugin) Start(wg *sync.WaitGroup) {
	for _, api := range p.config.Webapi {
		wg.Add(1)
		go func(apiConfig config.WebAPI) {
			defer wg.Done()

			// Set default timeout if not specified
			timeout := time.Duration(30)
			if apiConfig.Timeout > 0 {
				timeout = time.Duration(apiConfig.Timeout)
			}

			for {
				select {
				case <-p.stop:
					return
				default:
					if err := p.executeRequest(apiConfig); err != nil {
						log.Errorf("API request failed for %s: %v", apiConfig.Name, err)
					}
					time.Sleep(timeout * time.Second)
				}
			}
		}(api)
	}
}

func (p *APIClientPlugin) Stop() {
	close(p.stop)
}

func (p *APIClientPlugin) executeRequest(api config.WebAPI) error {
	// Prepare request body
	var bodyReader io.Reader
	if api.Body != nil {
		bodyBytes, err := json.Marshal(api.Body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %v", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	// Create request
	req, err := http.NewRequest(strings.ToUpper(api.Method), api.Endpoint, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	// Add headers
	for _, header := range api.Headers {
		req.Header.Add(header.Key, header.Value)
	}

	// Add authentication
	switch strings.ToLower(api.AuthType) {
	case "basic":
		req.SetBasicAuth(api.Username, api.Password)
	case "apikey":
		req.Header.Add("X-API-Key", api.APIKey)
	case "bearer":
		req.Header.Add("Authorization", "Bearer "+api.APIKey)
	}

	// Add query parameters
	if len(api.Query) > 0 {
		q := req.URL.Query()
		for key, value := range api.Query {
			q.Add(key, value)
		}
		req.URL.RawQuery = q.Encode()
	}

	// Execute request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %v", err)
	}

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	log.Infof("API request successful for %s: %s", api.Name, string(body))
	return nil
}
