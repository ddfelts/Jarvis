package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

func collectMetrics(config Config, apiClient *APIClient) (*SystemMetrics, error) {

	metrics := &SystemMetrics{
		Timestamp: time.Now(),
	}

	baseMsg := CreateBaseMessage(&config, "system_monitor", "metrics", "info", "System metrics collected")

	// CPU metrics
	if config.SystemMonitor.CPU {
		cpuPercent, err := cpu.Percent(time.Second, true)
		if err != nil {
			return nil, fmt.Errorf("error getting CPU metrics: %v", err)
		}
		count, err := cpu.Counts(true)
		if err != nil {
			return nil, fmt.Errorf("error getting CPU count: %v", err)
		}
		metrics.CPU = CPUMetrics{
			UsagePercent: cpuPercent,
			Count:        count,
		}
		baseMsg.Data["cpu"] = metrics.CPU
	}

	// Memory metrics
	if config.SystemMonitor.Memory {
		vmStat, err := mem.VirtualMemory()
		if err != nil {
			return nil, fmt.Errorf("error getting memory metrics: %v", err)
		}
		metrics.Memory = MemoryMetrics{
			Total:        vmStat.Total,
			Used:         vmStat.Used,
			Free:         vmStat.Free,
			UsagePercent: vmStat.UsedPercent,
		}
		baseMsg.Data["memory"] = metrics.Memory
	}
	// Network metrics
	if config.SystemMonitor.Network {
		netIO, err := net.IOCounters(false)
		if err != nil {
			return nil, fmt.Errorf("error getting network metrics: %v", err)
		}
		if len(netIO) > 0 {
			metrics.Network = NetworkMetrics{
				BytesSent:   netIO[0].BytesSent,
				BytesRecv:   netIO[0].BytesRecv,
				PacketsSent: netIO[0].PacketsSent,
				PacketsRecv: netIO[0].PacketsRecv,
			}
		}
	}

	// Temperature metrics
	if config.SystemMonitor.Temperature {
		temps, err := host.SensorsTemperatures()
		if err != nil {
			logger.Log("Metrics", "WARN", fmt.Sprintf("Failed to get temperature metrics: %v", err))
		} else {
			metrics.Temperature = make([]TempMetrics, 0, len(temps))
			for _, temp := range temps {
				if temp.Temperature > 0 { // Filter out invalid readings
					metrics.Temperature = append(metrics.Temperature, TempMetrics{
						SensorKey: temp.SensorKey,
						Temp:      temp.Temperature,
					})
				}
			}
		}
	}

	SendLogMessage(apiClient, &config, baseMsg)

	return metrics, nil
}

func monitorSystem(ctx context.Context, config Config, wg *sync.WaitGroup) {
	apiClient := NewAPIClient(3, wg)
	defer apiClient.Close()

	ticker := time.NewTicker(time.Duration(config.SystemMonitor.SleepTime) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("System monitor shutting down...")
			return
		case <-ticker.C:
			metrics, err := collectMetrics(config, apiClient)
			if err != nil {
				logger.Log("Metrics", "ERROR", fmt.Sprintf("Failed to collect metrics: %v", err))
				continue
			}

			jsonMetrics, err := json.Marshal(metrics)
			if err != nil {
				logger.Log("Metrics", "ERROR", fmt.Sprintf("Failed to marshal metrics: %v", err))
				continue
			}
			logger.Log("Metrics", "INFO", string(jsonMetrics))

			if config.SystemMonitor.APILogRemote {
				responseCh := apiClient.SendRequest(metrics, config.APILogRemote)
				select {
				case response := <-responseCh:
					if response.Error != nil {
						logger.Log("API", "ERROR", response.Error.Error())
					}
				case <-ctx.Done():
					return
				case <-time.After(5 * time.Second):
					logger.Log("API", "ERROR", "Timeout waiting for API response")
					return
				}
			}
		}
	}
}
