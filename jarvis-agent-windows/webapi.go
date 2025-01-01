package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type WebAPIMonitor struct {
	config *Config
	client *APIClient
	logger *Logger
}

type WebAPIResponse struct {
	Timestamp time.Time              `json:"timestamp"`
	Name      string                 `json:"name"`
	Subject   string                 `json:"subject"`
	Status    int                    `json:"status"`
	Duration  float64                `json:"duration"`
	Data      map[string]interface{} `json:"data"`
}

func NewWebAPIMonitor(config *Config, client *APIClient, logger *Logger) *WebAPIMonitor {
	return &WebAPIMonitor{
		config: config,
		client: client,
		logger: logger,
	}
}

func (w *WebAPIMonitor) Start(ctx context.Context) {
	// Create tickers for enabled APIs only
	type apiTicker struct {
		api    *WebAPI
		ticker *time.Ticker
	}

	var tickers []apiTicker
	for i, api := range w.config.Webapi {
		if api.Enabled { // Only create tickers for enabled APIs
			tickers = append(tickers, apiTicker{
				api:    &w.config.Webapi[i], // Use address from original slice to avoid copy
				ticker: time.NewTicker(time.Duration(api.SleepTime) * time.Second),
			})
		}
	}

	if len(tickers) == 0 {
		w.logger.Log("WebAPI", "INFO", "No enabled APIs to monitor")
		return
	}

	// Clean up all tickers when done
	defer func() {
		for _, t := range tickers {
			t.ticker.Stop()
		}
	}()

	// Monitor all APIs in a single select loop
	for {
		select {
		case <-ctx.Done():
			w.logger.Log("WebAPI", "INFO", "Stopping all API monitors...")
			return
		default:
			for _, t := range tickers {
				select {
				case <-t.ticker.C:
					w.checkEndpoint(t.api)
				default:
					// Don't block if ticker hasn't fired
				}
			}
			// Small sleep to prevent tight loop
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (w *WebAPIMonitor) checkEndpoint(api *WebAPI) {
	startTime := time.Now()

	// Create request
	var body io.Reader
	if api.Body != nil {
		jsonBody, err := json.Marshal(api.Body)
		if err != nil {
			w.handleError(api, fmt.Errorf("error marshaling body: %v", err))
			return
		}
		body = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(api.Method, api.Endpoint, body)
	if err != nil {
		w.handleError(api, fmt.Errorf("error creating request: %v", err))
		return
	}

	// Add headers
	for _, header := range api.Headers {
		req.Header.Set(header.Key, header.Value)
	}

	// Add auth if configured
	if api.AuthType == "apikey" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", api.APIKey))
	} else if api.AuthType == "basic" {
		req.SetBasicAuth(api.Username, api.Password)
	}

	// Execute request
	client := createHTTPClient(APILogRemote{
		APITimeout:    api.Timeout,
		APISkipVerify: api.SkipVerify,
	})

	resp, err := client.Do(req)
	if err != nil {
		w.handleError(api, fmt.Errorf("request failed: %v", err))
		return
	}
	defer resp.Body.Close()

	// Parse response
	duration := time.Since(startTime).Seconds()
	responseData := make(map[string]interface{})

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		w.handleError(api, fmt.Errorf("error reading response: %v", err))
		return
	}

	if len(bodyBytes) > 0 {
		if err := json.Unmarshal(bodyBytes, &responseData); err != nil {
			responseData["raw_response"] = string(bodyBytes)
		}
	}

	// Create response object
	apiResponse := WebAPIResponse{
		Timestamp: time.Now(),
		Name:      api.Name,
		Subject:   api.Subject,
		Status:    resp.StatusCode,
		Duration:  duration,
		Data:      responseData,
	}

	// Send to configured outputs
	w.sendToOutputs(api, apiResponse)
}

func (w *WebAPIMonitor) sendToOutputs(api *WebAPI, response WebAPIResponse) {
	// Send to API if configured
	if api.APILogRemote {
		baseMsg := CreateBaseMessage(w.config, "webapi", "api_check", "info",
			fmt.Sprintf("API check for %s completed", api.Name))
		baseMsg.Data = map[string]interface{}{
			"response": response,
		}
		SendLogMessage(w.client, w.config, baseMsg)
	}

	// Send to syslog if configured
	if w.config.Syslog.Enabled {
		jsonResponse, _ := json.Marshal(response)
		w.logger.Log("WebAPI", "INFO", string(jsonResponse))
	}

	// Always log status
	level := "INFO"
	if response.Status >= 400 {
		level = "ERROR"
	}
	w.logger.Log("WebAPI", level,
		fmt.Sprintf("%s [%d] - %.2fs", api.Name, response.Status, response.Duration))
}

func (w *WebAPIMonitor) handleError(api *WebAPI, err error) {
	errResponse := WebAPIResponse{
		Timestamp: time.Now(),
		Name:      api.Name,
		Subject:   api.Subject,
		Status:    -1,
		Data: map[string]interface{}{
			"error": err.Error(),
		},
	}
	w.sendToOutputs(api, errResponse)
}
