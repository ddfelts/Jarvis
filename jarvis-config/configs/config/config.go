package config

import (
	"time"
)

type Config struct {
	Default     Default     `yaml:"default"`
	Service     Service     `yaml:"service_monitor"`
	Syslog      Syslog      `yaml:"syslog"`
	Windowslogs Windowslogs `yaml:"windowslogs"`
	Filelogs    []string    `yaml:"filelogs"`
	Webapi      []WebAPI    `yaml:"webapi"`
	Plugins     Plugins     `yaml:"plugins"`
}

type Windowslogs struct {
	Forward  bool     `yaml:"forward"`
	Channels []string `yaml:"channels"`
}

type Syslog struct {
	Tag          string `yaml:"tag"`
	SyslogServer string `yaml:"syslog_server"`
}

type Service struct {
	Service   []string `yaml:"service"`
	Sleeptime int      `yaml:"sleeptime"`
}

type Default struct {
	Syslog         bool   `yaml:"syslog"`
	APIURL         string `yaml:"api_url"`
	APIKey         string `yaml:"api_key"`
	CPU            bool   `yaml:"cpu"`
	Memory         bool   `yaml:"memory"`
	Disk           bool   `yaml:"disk"`
	Network        bool   `yaml:"network"`
	ServiceMonitor bool   `yaml:"service_monitor"`
	Windowslogs    bool   `yaml:"windowslogs"`
	Filelogs       bool   `yaml:"filelogs"`
	Webapi         bool   `yaml:"webapi"`
}

type WebAPI struct {
	Name       string            `yaml:"name"`
	Endpoint   string            `yaml:"endpoint"`
	Method     string            `yaml:"method"`
	Headers    []Header          `yaml:"headers"`
	Query      map[string]string `yaml:"query"`
	Body       interface{}       `yaml:"body"`
	AuthType   string            `yaml:"auth_type"`
	Username   string            `yaml:"username"`
	Password   string            `yaml:"password"`
	APIKey     string            `yaml:"api_key"`
	Timeout    int               `yaml:"timeout"`
	SkipVerify bool              `yaml:"skip_verify"`
}

type Header struct {
	Key   string `yaml:"key"`
	Value string `yaml:"value"`
}

type SystemMetrics struct {
	Timestamp   time.Time      `json:"timestamp"`
	CPU         CPUMetrics     `json:"cpu"`
	Memory      MemoryMetrics  `json:"memory"`
	Network     NetworkMetrics `json:"network"`
	Temperature []TempMetrics  `json:"temperature"`
}

type CPUMetrics struct {
	UsagePercent []float64 `json:"usage_percent"`
	Count        int       `json:"count"`
}

type MemoryMetrics struct {
	Total        uint64  `json:"total"`
	Used         uint64  `json:"used"`
	Free         uint64  `json:"free"`
	UsagePercent float64 `json:"usage_percent"`
}

type NetworkMetrics struct {
	BytesSent   uint64 `json:"bytes_sent"`
	BytesRecv   uint64 `json:"bytes_recv"`
	PacketsSent uint64 `json:"packets_sent"`
	PacketsRecv uint64 `json:"packets_recv"`
}

type TempMetrics struct {
	SensorKey string  `json:"sensor_key"`
	Temp      float64 `json:"temperature"`
}

type Plugins struct {
	Directory string                 `yaml:"directory"`
	Enabled   []string               `yaml:"enabled"`
	Config    map[string]interface{} `yaml:"config"`
}

type Logger interface {
	Log(source, level, message string)
}
