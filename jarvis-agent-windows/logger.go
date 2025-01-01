package main

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

// BaseMessage represents the common structure for all messages
type BaseMessage struct {
	Timestamp time.Time              `json:"timestamp"`
	AgentID   string                 `json:"agent_id"`
	AgentName string                 `json:"agent_name"`
	Source    string                 `json:"source"`
	Type      string                 `json:"type"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// CreateBaseMessage creates a new base message with common fields
func CreateBaseMessage(config *Config, source, msgType, level, message string) BaseMessage {
	return BaseMessage{
		Timestamp: time.Now(),
		AgentID:   config.JarvisAgent.ID,
		AgentName: config.JarvisAgent.Name,
		Source:    source,
		Type:      msgType,
		Level:     level,
		Message:   message,
		Data:      make(map[string]interface{}),
	}
}

// SendLogMessage sends a log message to the API
func SendLogMessage(apiClient *APIClient, config *Config, msg BaseMessage) {
	if !config.APILogRemote.Enabled {
		return
	}

	responseCh := apiClient.SendRequest(msg, config.APILogRemote)
	response := <-responseCh
	if response.Error != nil {
		log.Printf("[API][ERROR] Failed to send event: %v", response.Error)
	}
}

type LogMessage struct {
	Source  string
	Level   string
	Message string
}

type Logger struct {
	logChan      chan LogMessage
	syslogChan   chan LogMessage
	wg           *sync.WaitGroup
	syslogClient *SyslogClient
}

func NewLogger(wg *sync.WaitGroup, config Config) *Logger {
	logger := &Logger{
		logChan:    make(chan LogMessage, 1000),
		syslogChan: make(chan LogMessage, 1000),
		wg:         wg,
	}

	if config.Syslog.Enabled {
		syslogClient, err := NewSyslogClient(config.Syslog)
		if err != nil {
			fmt.Printf("Failed to initialize syslog client: %v\n", err)
		} else {
			logger.syslogClient = syslogClient
			wg.Add(1)
			go logger.processSyslog()
		}
	}

	wg.Add(1)
	go logger.processLogs()

	return logger
}

func (l *Logger) processSyslog() {
	defer l.wg.Done()
	hostname, _ := os.Hostname()

	for msg := range l.syslogChan {
		if l.syslogClient != nil {
			priority := getPriority(msg.Level)
			err := l.syslogClient.Write(priority, time.Now(), hostname, msg.Source, msg.Message)
			if err != nil {
				fmt.Printf("Failed to write to syslog: %v\n", err)
			}
		}
	}
}

func getPriority(level string) int {
	switch level {
	case "ERROR":
		return 3 // Error severity
	case "WARN":
		return 4 // Warning severity
	case "INFO":
		return 6 // Informational severity
	case "DEBUG":
		return 7 // Debug severity
	default:
		return 6 // Default to informational
	}
}

func (l *Logger) processLogs() {
	defer l.wg.Done()
	for msg := range l.logChan {
		// You can modify the output format here
		fmt.Printf("[%s][%s] %s\n", msg.Source, msg.Level, msg.Message)
	}
}

func (l *Logger) Log(source, level, message string) {
	logMsg := LogMessage{
		Source:  source,
		Level:   level,
		Message: message,
	}

	select {
	case l.logChan <- logMsg:
	default:
		fmt.Printf("[%s][%s] %s (buffer full)\n", source, level, message)
	}

	if l.syslogClient != nil {
		select {
		case l.syslogChan <- logMsg:
		default:
			// Syslog buffer full, drop message
		}
	}
}

func (l *Logger) Close() {
	close(l.logChan)
	if l.syslogClient != nil {
		close(l.syslogChan)
		l.syslogClient.Close()
	}
}
