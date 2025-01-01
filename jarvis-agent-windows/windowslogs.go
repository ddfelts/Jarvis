package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/0xrawsec/golang-evtx/evtx"
	"github.com/0xrawsec/golang-win32/win32/wevtapi"
)

type WindowsEvent struct {
	Channel   string                 `json:"channel"`
	EventData map[string]interface{} `json:"event_data,omitempty"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Computer  string                 `json:"computer"`
}

type EventProvider struct {
	provider *wevtapi.PullEventProvider
}

func NewEventProvider() *EventProvider {
	provider := wevtapi.NewPullEventProvider()
	return &EventProvider{provider: provider}
}

func XMLEventToGoEvtxMap(xe *wevtapi.XMLEvent) (*evtx.GoEvtxMap, error) {
	ge := make(evtx.GoEvtxMap)
	bytes, err := json.Marshal(xe.ToJSONEvent())
	if err != nil {
		return &ge, err
	}
	err = json.Unmarshal(bytes, &ge)
	if err != nil {
		return &ge, err
	}
	return &ge, nil
}

func monitorWindowsLogs(ctx context.Context, config Config, wg *sync.WaitGroup) {
	defer wg.Done()

	apiClient := NewAPIClient(3, wg)
	channelWg := &sync.WaitGroup{}

	eventProvider := NewEventProvider()

	// Create a channel to signal monitors to stop
	done := make(chan bool)

	// Start channel monitors
	for _, channel := range config.Windowslogs.Channels {
		channelWg.Add(1)
		go func(ch string) {
			defer channelWg.Done()
			monitorChannel(ch, eventProvider, apiClient, config, done)
		}(channel)
	}

	// Wait for context cancellation
	<-ctx.Done()
	log.Printf("Stopping Windows log monitors...")
	close(done)
	channelWg.Wait()
	apiClient.Close()
}

func monitorChannel(channel string, ep *EventProvider, apiClient *APIClient, config Config, stopChan <-chan bool) {
	logger.Log("WindowsLogs", "INFO", fmt.Sprintf("Starting monitoring of channel: %s", channel))

	events := ep.provider.FetchEvents([]string{channel}, wevtapi.EvtSubscribeToFutureEvents)

	for {
		select {
		case <-stopChan:
			logger.Log("WindowsLogs", "INFO", fmt.Sprintf("Stopping monitor for channel: %s", channel))
			return
		case xe := <-events:
			if xe == nil {
				continue
			}
			// Process event with timeout
			done := make(chan bool)
			go func() {
				if err := processEvent(xe, channel, apiClient, config, stopChan); err != nil {
					logger.Log("WindowsLogs", "ERROR", fmt.Sprintf("Failed to process event: %v", err))
				}
				close(done)
			}()

			select {
			case <-done:
				// Event processed successfully
			case <-stopChan:
				logger.Log("WindowsLogs", "INFO", fmt.Sprintf("Stopping monitor for channel: %s", channel))
				return
			}
		}
	}
}

func processEvent(evt *wevtapi.XMLEvent, channel string, apiClient *APIClient, config Config, stopChan <-chan bool) error {
	jsonData, err := json.Marshal(evt.ToJSONEvent())
	if err != nil {
		return fmt.Errorf("failed to marshal event: %v", err)
	}

	var eventData map[string]interface{}
	if err := json.Unmarshal(jsonData, &eventData); err != nil {
		return fmt.Errorf("failed to unmarshal event: %v", err)
	}

	winEvent := &WindowsEvent{
		Channel:   channel,
		EventData: eventData,
		Level:     evt.System.Level,
		Computer:  evt.System.Computer,
	}

	// Convert to JSON
	jsonEvent, err := json.Marshal(winEvent)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %v", err)
	}

	// Log locally
	logger.Log("WindowsLogs", "INFO", string(jsonEvent))

	// Forward to API if enabled
	if config.Windowslogs.APILogRemote {
		baseMsg := CreateBaseMessage(&config, "windows_logs", "event", getLevelString(winEvent.Level), winEvent.Computer)
		baseMsg.Data["event"] = winEvent
		responseCh := apiClient.SendRequest(baseMsg, config.APILogRemote)
		select {
		case response := <-responseCh:
			if response.Error != nil {
				logger.Log("API", "ERROR", fmt.Sprintf("Failed to send event: %v", response.Error))
			}
		case <-stopChan:
			return nil
		case <-time.After(5 * time.Second):
			logger.Log("API", "ERROR", "Timeout waiting for API response")
			return nil
		}
	}

	// Forward to Syslog if enabled
	if config.Syslog.Enabled {
		logger.Log("Syslog", getLevelString(winEvent.Level), string(jsonEvent))
	}

	return nil
}

func getLevelString(level string) string {
	switch level {
	case "1":
		return "Critical"
	case "2":
		return "Error"
	case "3":
		return "Warning"
	case "4":
		return "Information"
	case "5":
		return "Verbose"
	default:
		return "Unknown"
	}
}
