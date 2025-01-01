package main

import (
	"fmt"
	"time"

	"github.com/0xrawsec/golang-utils/log"
	"golang.org/x/sys/windows/svc/mgr"
)

func checkService(m *mgr.Mgr, serviceName string) {
	s, err := m.OpenService(serviceName)
	if err != nil {
		logger.Log("ServiceMonitor", "ERROR", fmt.Sprintf("Service %s not found: %v", serviceName, err))
		return
	}
	defer s.Close()

	status, err := s.Query()
	if err != nil {
		logger.Log("ServiceMonitor", "ERROR", fmt.Sprintf("Failed to query service %s: %v", serviceName, err))
		return
	}

	switch status.State {
	case 0: // Stopped
		logger.Log("ServiceMonitor", "WARN", fmt.Sprintf("Service %s is stopped", serviceName))
	case 1: // Running
		logger.Log("ServiceMonitor", "INFO", fmt.Sprintf("Service %s is running", serviceName))
		return
	case 7: // Paused
		logger.Log("ServiceMonitor", "INFO", fmt.Sprintf("Service %s is paused", serviceName))
	case 2: // StartPending
		logger.Log("ServiceMonitor", "INFO", fmt.Sprintf("Service %s is starting", serviceName))
	case 3: // StopPending
		logger.Log("ServiceMonitor", "INFO", fmt.Sprintf("Service %s is stopping", serviceName))
	default:
		logger.Log("ServiceMonitor", "INFO", fmt.Sprintf("Service %s state changed to: %d", serviceName, status.State))
	}
}

func monitorServices(services []string, sleeptime int) {
	m, err := mgr.Connect()
	if err != nil {
		log.Errorf("Failed to connect to service manager: %v", err)
		return
	}
	defer m.Disconnect()

	for {
		for _, serviceName := range services {
			checkService(m, serviceName)
		}
		time.Sleep(time.Duration(sleeptime) * time.Second)
	}
}
