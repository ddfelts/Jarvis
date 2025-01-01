package main

import (
	"time"
)

type Config struct {
	ServiceMonitor ServiceMonitor `yaml:"service_monitor"`
	Syslog         Syslog         `yaml:"syslog"`
	Windowslogs    Windowslogs    `yaml:"windowslogs"`
	Filelogs       []string       `yaml:"filelogs"`
	Webapi         []WebAPI       `yaml:"webapi"`
	SystemMonitor  SystemMonitor  `yaml:"system_monitor"`
	APILogRemote   APILogRemote   `yaml:"apilogremote"`
	WebMonitor     WebMonitor     `yaml:"web_monitor"`
	JarvisAgent    JarvisAgent    `yaml:"jarvisagent"`
}

type JarvisAgent struct {
	Name string `yaml:"name"`
	ID   string `yaml:"id"`
}

type Windowslogs struct {
	Enabled      bool     `yaml:"enabled"`
	APILogRemote bool     `yaml:"apilogremote,omitempty"`
	Channels     []string `yaml:"channels"`
}

type Syslog struct {
	Enabled    bool   `yaml:"enabled"`
	Protocol   string `yaml:"protocol"` // "udp", "tcp", or "tcp+tls"
	Server     string `yaml:"server"`
	Tag        string `yaml:"tag"`
	SkipVerify bool   `yaml:"skip_verify"`
}

type ServiceMonitor struct {
	Enabled      bool     `yaml:"enabled"`
	APILogRemote bool     `yaml:"apilogremote,omitempty"`
	SleepTime    int      `yaml:"sleeptime"`
	Services     []string `yaml:"service"`
}

type APILogRemote struct {
	Enabled       bool        `yaml:"enabled"`
	APIURL        string      `yaml:"api_url"`
	APIKey        string      `yaml:"api_key,omitempty"`
	APIMethod     string      `yaml:"api_method"`
	APITimeout    int         `yaml:"api_timeout"`
	APISkipVerify bool        `yaml:"api_skip_verify,omitempty"`
	APIAuthType   string      `yaml:"api_auth_type,omitempty"`
	APIUsername   string      `yaml:"api_username,omitempty"`
	APIPassword   string      `yaml:"api_password,omitempty"`
	APIHeaders    []APIHeader `yaml:"api_headers,omitempty"`
	APIBody       []string    `yaml:"api_body,omitempty"`
	APIQuery      []string    `yaml:"api_query,omitempty"`
}

type SystemMonitor struct {
	Enabled      bool `yaml:"enabled"`
	APILogRemote bool `yaml:"apilogremote,omitempty"`
	SleepTime    int  `yaml:"sleeptime"`
	CPU          bool `yaml:"cpu"`
	Memory       bool `yaml:"memory"`
	Disk         bool `yaml:"disk"`
	Network      bool `yaml:"network"`
	Temperature  bool `yaml:"temperature"`
}

type Enabled struct {
	Enabled bool `yaml:"enabled"`
}

type WebAPI struct {
	Enabled      bool              `yaml:"enabled"`
	Name         string            `yaml:"name"`
	Subject      string            `yaml:"subject"`
	Endpoint     string            `yaml:"endpoint"`
	Method       string            `yaml:"method"`
	Headers      []APIHeader       `yaml:"headers"`
	Query        map[string]string `yaml:"query"`
	Body         interface{}       `yaml:"body"`
	AuthType     string            `yaml:"auth_type"`
	Username     string            `yaml:"username"`
	Password     string            `yaml:"password"`
	APIKey       string            `yaml:"api_key"`
	Timeout      int               `yaml:"timeout"`
	SkipVerify   bool              `yaml:"skip_verify"`
	SleepTime    int               `yaml:"sleeptime"`
	APILogRemote bool              `yaml:"apilogremote"`
}

type APIHeader struct {
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

type WebMonitor struct {
	APILogRemote bool     `yaml:"apilogremote,omitempty"`
	Enabled      bool     `yaml:"enabled"`
	SleepTime    int      `yaml:"sleeptime"`
	Timeout      int      `yaml:"timeout"`
	SkipVerify   bool     `yaml:"skip_verify"`
	URLs         []string `yaml:"urls"`
}
