package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

type WebStatus struct {
	URL        string        `json:"url"`
	Status     string        `json:"status"`
	StatusCode int           `json:"status_code"`
	Time       time.Time     `json:"timestamp"`
	Response   time.Duration `json:"response_time"`
}

func monitorWebsites(ctx context.Context, config Config, wg *sync.WaitGroup) {
	apiClient := NewAPIClient(3, wg)
	defer apiClient.Close()

	client := &http.Client{
		Timeout: time.Duration(config.WebMonitor.Timeout) * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: config.WebMonitor.SkipVerify,
			},
		},
	}

	ticker := time.NewTicker(time.Duration(config.WebMonitor.SleepTime) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("Web monitor shutting down...")
			return
		case <-ticker.C:
			// Use a WaitGroup to track all website checks
			checkWg := sync.WaitGroup{}
			for _, site := range config.WebMonitor.URLs {
				checkWg.Add(1)
				go func(url string) {
					defer checkWg.Done()
					select {
					case <-ctx.Done():
						return
					default:
						checkWebsite(ctx, url, client, apiClient, config)
					}
				}(site)
			}
			// Wait for all checks to complete or context to be cancelled
			done := make(chan struct{})
			go func() {
				checkWg.Wait()
				close(done)
			}()
			select {
			case <-done:
				// All checks completed
			case <-ctx.Done():
				return
			}
		}
	}
}

func checkWebsite(ctx context.Context, url string, client *http.Client, apiClient *APIClient, config Config) {
	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		logger.Log("WebMonitor", "ERROR", fmt.Sprintf("Failed to create request for %s: %v", url, err))
		return
	}

	resp, err := client.Do(req)
	elapsed := time.Since(start)

	status := WebStatus{
		URL:      url,
		Time:     time.Now(),
		Response: elapsed,
	}

	if err != nil {
		status.Status = "Down"
		status.StatusCode = 0
		logger.Log("WebMonitor", "ERROR", fmt.Sprintf("Failed to connect to %s: %v", url, err))
	} else {
		defer resp.Body.Close()
		status.Status = "Up"
		status.StatusCode = resp.StatusCode
		if resp.StatusCode >= 400 {
			status.Status = "Error"
			logger.Log("WebMonitor", "WARN", fmt.Sprintf("%s returned status code %d", url, resp.StatusCode))
		} else {
			logger.Log("WebMonitor", "INFO", fmt.Sprintf("%s is up (response time: %v)", url, elapsed))
		}
	}

	jsonStatus, err := json.Marshal(status)
	if err != nil {
		logger.Log("WebMonitor", "ERROR", fmt.Sprintf("Failed to marshal status: %v", err))
		return
	}
	logger.Log("WebMonitor", "INFO", string(jsonStatus))

	if config.WebMonitor.APILogRemote {
		responseCh := apiClient.SendRequest(status, config.APILogRemote)
		select {
		case response := <-responseCh:
			if response.Error != nil {
				logger.Log("API", "ERROR", fmt.Sprintf("Failed to send status for %s: %v", url, response.Error))
			}
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Second):
			logger.Log("API", "ERROR", "Timeout waiting for API response")
			return
		}
	}
}
