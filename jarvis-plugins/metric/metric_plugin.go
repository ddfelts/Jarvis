package main

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"jarvis-agent/configs/config"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

type MetricPlugin struct {
	config *config.Config
	logger config.Logger
	stop   chan struct{}
}

func New() interface{} {
	return &MetricPlugin{
		stop: make(chan struct{}),
	}
}

func (p *MetricPlugin) Name() string {
	return "MetricPlugin"
}

func (p *MetricPlugin) Init(config *config.Config) error {
	p.config = config
	return nil
}

func (p *MetricPlugin) collectMetrics() (*config.SystemMetrics, error) {
	metrics := &config.SystemMetrics{
		Timestamp: time.Now(),
	}

	if p.config.Default.CPU {
		cpuPercent, err := cpu.Percent(time.Second, true)
		if err != nil {
			return nil, fmt.Errorf("error getting CPU metrics: %v", err)
		}
		count, err := cpu.Counts(true)
		if err != nil {
			return nil, fmt.Errorf("error getting CPU count: %v", err)
		}
		metrics.CPU = config.CPUMetrics{
			UsagePercent: cpuPercent,
			Count:        count,
		}
	}

	if p.config.Default.Memory {
		vmStat, err := mem.VirtualMemory()
		if err != nil {
			return nil, fmt.Errorf("error getting memory metrics: %v", err)
		}
		metrics.Memory = config.MemoryMetrics{
			Total:        vmStat.Total,
			Used:         vmStat.Used,
			Free:         vmStat.Free,
			UsagePercent: vmStat.UsedPercent,
		}
	}

	if p.config.Default.Network {
		netIO, err := net.IOCounters(false)
		if err != nil {
			return nil, fmt.Errorf("error getting network metrics: %v", err)
		}
		if len(netIO) > 0 {
			metrics.Network = config.NetworkMetrics{
				BytesSent:   netIO[0].BytesSent,
				BytesRecv:   netIO[0].BytesRecv,
				PacketsSent: netIO[0].PacketsSent,
				PacketsRecv: netIO[0].PacketsRecv,
			}
		}
	}
	return metrics, nil
}

func (p *MetricPlugin) Start(wg *sync.WaitGroup) {
	if !p.config.Default.CPU && !p.config.Default.Memory && !p.config.Default.Network {
		return
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				metrics, err := p.collectMetrics()
				if err != nil {
					p.logger.Log("MetricPlugin", "ERROR", fmt.Sprintf("Failed to collect metrics: %v", err))
					continue
				}

				jsonMetrics, err := json.Marshal(metrics)
				if err != nil {
					p.logger.Log("MetricPlugin", "ERROR", fmt.Sprintf("Failed to marshal metrics: %v", err))
					continue
				}

				p.logger.Log("MetricPlugin", "INFO", string(jsonMetrics))
			case <-p.stop:
				return
			}
		}
	}()
}

func (p *MetricPlugin) Stop() {
	close(p.stop)
}
