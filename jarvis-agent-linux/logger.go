package main

import (
	"fmt"
	"sync"
)

type LogMessage struct {
	Source  string
	Level   string
	Message string
}

type Logger struct {
	logChan chan LogMessage
	wg      *sync.WaitGroup
}

func NewLogger(wg *sync.WaitGroup) *Logger {
	logger := &Logger{
		logChan: make(chan LogMessage, 1000), // Buffer size of 1000
		wg:      wg,
	}

	// Start the logging goroutine
	wg.Add(1)
	go logger.processLogs()

	return logger
}

func (l *Logger) processLogs() {
	defer l.wg.Done()
	for msg := range l.logChan {
		// You can modify the output format here
		fmt.Printf("[%s][%s] %s\n", msg.Source, msg.Level, msg.Message)
	}
}

func (l *Logger) Log(source, level, message string) {
	select {
	case l.logChan <- LogMessage{Source: source, Level: level, Message: message}:
	default:
		// Channel is full, print directly to avoid blocking
		fmt.Printf("[%s][%s] %s (buffer full)\n", source, level, message)
	}
}

func (l *Logger) Close() {
	close(l.logChan)
}
