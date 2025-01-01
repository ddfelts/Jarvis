package main

import (
	"fmt"
	"jarvis-agent/configs/config"
	"sync"
	"time"

	"github.com/0xrawsec/golang-utils/log"
)

type ServiceMonitorPlugin struct {
	config    *config.Config
	stop      chan struct{}
	services  []string
	sleepTime int
}

// New is the plugin constructor required by the plugin system
func New() config.Plugin {
	return &ServiceMonitorPlugin{
		stop: make(chan struct{}),
	}
}

func (p *ServiceMonitorPlugin) Name() string {
	return "ServiceMonitor"
}

func (p *ServiceMonitorPlugin) Init(cfg *config.Config) error {
	p.config = cfg
	p.services = cfg.Service.Service
	p.sleepTime = cfg.Service.Sleeptime
	if p.sleepTime == 0 {
		p.sleepTime = 30 // default sleep time in seconds
	}
	return nil
}

func (p *ServiceMonitorPlugin) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-p.stop:
				return
			default:
				p.monitorServices()
				time.Sleep(time.Duration(p.sleepTime) * time.Second)
			}
		}
	}()
}

func (p *ServiceMonitorPlugin) Stop() {
	close(p.stop)
}

func (p *ServiceMonitorPlugin) monitorServices() {
	for _, service := range p.services {
		status, err := getServiceStatus(service)
		if err != nil {
			log.Errorf("Failed to get status for service %s: %v", service, err)
			continue
		}
		log.Infof("Service %s status: %s", service, status)
	}
}

func getServiceStatus(serviceName string) (string, error) {
	// Implementation of getServiceStatus from your existing code
	// Copy your existing service status checking logic here
	return "", fmt.Errorf("not implemented")
}
