package main

import (
	"fmt"
	"jarvis-agent/configs/config"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/0xrawsec/golang-utils/log"
	"github.com/0xrawsec/golang-win32/win32/wevtapi"
	"gopkg.in/yaml.v3"
)

const (
	ExitSuccess = 0
	ExitFailure = 1
)

func loadConfig(filename string) (config.Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return config.Config{}, fmt.Errorf("error reading config file: %v", err)
	}

	var cfg config.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return config.Config{}, fmt.Errorf("error parsing config file: %v", err)
	}

	return cfg, nil
}

// Global variables
var (
	// initialized
	eventProvider = wevtapi.NewPullEventProvider()
	logger        *Logger
)

func terminate() {
	os.Exit(ExitFailure)
}

func main() {
	config, err := loadConfig("config.yaml")
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		terminate()
	}
	wg := sync.WaitGroup{}

	// Initialize logger
	logger = NewLogger(&wg)
	defer logger.Close()

	// Initialize plugin manager
	pluginManager := NewPluginManager(&config, &wg)
	if err := pluginManager.LoadPluginsFromDir(config.Plugins.Directory); err != nil {
		log.Errorf("Failed to load plugins: %v", err)
	}

	// Signal handler
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		pluginManager.StopAll()
		eventProvider.Stop()
		terminate()
	}()

	// Start plugins
	pluginManager.StartAll()

	//Windows logs
	if config.Default.Windowslogs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			windowslogs(config)
		}()
	}

	wg.Wait()
}
