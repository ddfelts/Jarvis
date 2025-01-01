package main

import (
	"fmt"
	"jarvis-agent/configs/config"
	"time"

	"github.com/0xrawsec/golang-utils/log"
	"github.com/RackSec/srslog"
)

type SyslogClient struct {
	writer *srslog.Writer
	tag    string
}

func NewSyslogClient(config *config.Config) (*SyslogClient, error) {
	if !config.Default.Syslog {
		return nil, fmt.Errorf("syslog is not enabled in config")
	}

	writer, err := srslog.Dial(
		"udp",
		config.Syslog.SyslogServer,
		srslog.LOG_INFO|srslog.LOG_DAEMON,
		config.Syslog.Tag,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to syslog server: %v", err)
	}

	return &SyslogClient{
		writer: writer,
		tag:    config.Syslog.Tag,
	}, nil
}

func (s *SyslogClient) Close() error {
	if s.writer != nil {
		return s.writer.Close()
	}
	return nil
}

func (s *SyslogClient) SendLog(message string) error {
	if s.writer == nil {
		return fmt.Errorf("syslog writer is not initialized")
	}
	return s.writer.Info(message)
}

func startSyslogForwarder(config *config.Config, messages chan string) {
	client, err := NewSyslogClient(config)
	if err != nil {
		log.Errorf("Failed to create syslog client: %v", err)
		return
	}
	defer client.Close()

	for message := range messages {
		if err := client.SendLog(message); err != nil {
			log.Errorf("Failed to send log to syslog: %v", err)
			// Try to reconnect
			if client, err = NewSyslogClient(config); err != nil {
				log.Errorf("Failed to reconnect to syslog: %v", err)
				time.Sleep(5 * time.Second)
			}
		}
	}
}
