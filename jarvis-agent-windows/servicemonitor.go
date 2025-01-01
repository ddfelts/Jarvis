package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"golang.org/x/sys/windows/svc/mgr"
)

type ServiceStatus struct {
	Name   string    `json:"name"`
	Status string    `json:"status"`
	Time   time.Time `json:"timestamp"`
}

func monitorServices(ctx context.Context, services []string, sleepTime int, wg *sync.WaitGroup, config Config) {
	apiClient := NewAPIClient(3, wg)
	defer apiClient.Close()

	ticker := time.NewTicker(time.Duration(sleepTime) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("Service monitor shutting down...")
			return
		case <-ticker.C:
			for _, service := range services {
				select {
				case <-ctx.Done():
					return
				default:
					checkService(ctx, service, apiClient, config)
				}
			}
		}
	}
}

func getServiceStateString(state uint32) string {
	switch state {
	case 0x4: // SERVICE_RUNNING
		return "Running"
	case 0x1: // SERVICE_STOPPED
		return "Stopped"
	case 0x2: // SERVICE_START_PENDING
		return "Starting"
	case 0x3: // SERVICE_STOP_PENDING
		return "Stopping"
	case 0x5: // SERVICE_PAUSE_PENDING
		return "Pausing"
	case 0x6: // SERVICE_PAUSED
		return "Paused"
	case 0x7: // SERVICE_CONTINUE_PENDING
		return "Resuming"
	default:
		return "Unknown"
	}
}

func checkService(ctx context.Context, serviceName string, apiClient *APIClient, config Config) {
	manager, err := mgr.Connect()
	if err != nil {
		logger.Log("Service", "ERROR", fmt.Sprintf("Failed to connect to service manager: %v", err))
		return
	}
	defer manager.Disconnect()

	service, err := manager.OpenService(serviceName)
	if err != nil {
		logger.Log("Service", "ERROR", fmt.Sprintf("Failed to open service %s: %v", serviceName, err))
		return
	}
	defer service.Close()

	status, err := service.Query()
	if err != nil {
		logger.Log("Service", "ERROR", fmt.Sprintf("Failed to query service %s: %v", serviceName, err))
		return
	}

	serviceStatus := ServiceStatus{
		Name:   serviceName,
		Status: getServiceStateString(uint32(status.State)),
		Time:   time.Now(),
	}

	jsonStatus, err := json.Marshal(serviceStatus)
	if err != nil {
		logger.Log("Service", "ERROR", fmt.Sprintf("Failed to marshal service status: %v", err))
		return
	}

	logger.Log("Service", "INFO", string(jsonStatus))

	if config.ServiceMonitor.APILogRemote {
		responseCh := apiClient.SendRequest(serviceStatus, config.APILogRemote)
		select {
		case response := <-responseCh:
			if response.Error != nil {
				logger.Log("API", "ERROR", fmt.Sprintf("Failed to send status for %s: %v", serviceName, response.Error))
			}
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Second):
			logger.Log("API", "ERROR", "Timeout waiting for API response")
			return
		}
	}
}
