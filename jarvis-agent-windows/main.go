package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"
)

var logger *Logger

func main() {
	// Parse command line flags
	configPath := flag.String("config", "config.yaml", "Path to config file")
	flag.Parse()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up logging with rotation (10MB max size, keep 5 files)
	logFile, err := setupLogging("jarvis-agent.log", 10*1024*1024, 5)
	if err != nil {
		fmt.Printf("Failed to set up logging: %v\n", err)
		return
	}
	defer logFile.Close()
	log.SetOutput(logFile)
	log.SetFlags(log.Ldate | log.Ltime | log.LUTC)

	log.Printf("Starting Jarvis Agent...")

	// Initialize components
	wg := &sync.WaitGroup{}
	wg.Add(1) // Add for the main goroutine

	// Load configuration
	config, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger
	logger = NewLogger(wg, config)
	defer logger.Close()

	// Start components based on configuration
	if config.ServiceMonitor.Enabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			monitorServices(ctx, config.ServiceMonitor.Services, config.ServiceMonitor.SleepTime, wg, config)
		}()
	}

	if config.SystemMonitor.Enabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			monitorSystem(ctx, config, wg)
		}()
	}

	if config.WebMonitor.Enabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			monitorWebsites(ctx, config, wg)
		}()
	}

	if len(config.Webapi) > 0 {
		wg.Add(1)
		apiClient := NewAPIClient(3, wg)
		webMonitor := NewWebAPIMonitor(&config, apiClient, logger)

		go func() {
			defer wg.Done()
			webMonitor.Start(ctx)
		}()
	}

	if config.Windowslogs.Enabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			monitorWindowsLogs(ctx, config, wg)
		}()
	}

	// Handle shutdown signal in a separate goroutine
	go func() {
		sig := <-sigChan
		log.Printf("Received signal: %v", sig)
		log.Printf("Initiating shutdown...")
		cancel() // Cancel context first

		// Set a timeout for graceful shutdown
		shutdownTimer := time.NewTimer(10 * time.Second)
		shutdownDone := make(chan struct{})

		go func() {
			wg.Wait()
			close(shutdownDone)
		}()

		select {
		case <-shutdownDone:
			log.Printf("Clean shutdown completed")
		case <-shutdownTimer.C:
			log.Printf("Shutdown timed out, forcing exit")
			os.Exit(1)
		}
	}()

	// Wait for all components
	wg.Wait()
	log.Printf("Program terminated")
}

func loadConfig(filename string) (Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return Config{}, fmt.Errorf("error reading config file: %v", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("error parsing config file: %v", err)
	}

	return cfg, nil
}

func setupLogging(filename string, maxSize int64, keepFiles int) (*os.File, error) {
	// Check if file exists and get its size
	info, err := os.Stat(filename)
	if err == nil && info.Size() > maxSize {
		// Rotate existing log files
		for i := keepFiles - 1; i > 0; i-- {
			oldName := fmt.Sprintf("%s.%d", filename, i)
			newName := fmt.Sprintf("%s.%d", filename, i+1)
			if _, err := os.Stat(oldName); err == nil {
				os.Rename(oldName, newName)
			}
		}
		// Rename current log file
		os.Rename(filename, filename+".1")
	}

	// Open/create new log file
	return os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
}
