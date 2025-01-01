package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"time"
)

type SyslogClient struct {
	protocol  string
	address   string
	conn      net.Conn
	tlsConfig *tls.Config
}

func NewSyslogClient(config Syslog) (*SyslogClient, error) {
	client := &SyslogClient{
		protocol: config.Protocol,
		address:  config.Server,
	}

	if config.Protocol == "tcp+tls" {
		client.tlsConfig = &tls.Config{
			InsecureSkipVerify: config.SkipVerify,
		}
	}

	if err := client.connect(); err != nil {
		return nil, err
	}

	return client, nil
}

func (c *SyslogClient) connect() error {
	var err error
	switch c.protocol {
	case "tcp", "tcp+tls":
		if c.protocol == "tcp+tls" {
			c.conn, err = tls.Dial("tcp", c.address, c.tlsConfig)
		} else {
			c.conn, err = net.Dial("tcp", c.address)
		}
	case "udp":
		c.conn, err = net.Dial("udp", c.address)
	default:
		return fmt.Errorf("unsupported protocol: %s", c.protocol)
	}
	return err
}

func (c *SyslogClient) Write(priority int, timestamp time.Time, hostname, tag, content string) error {
	if c.conn == nil {
		if err := c.connect(); err != nil {
			return err
		}
	}

	msg := fmt.Sprintf("<%d>%s %s %s: %s\n",
		priority,
		timestamp.Format(time.RFC3339),
		hostname,
		tag,
		content,
	)

	_, err := c.conn.Write([]byte(msg))
	if err != nil {
		c.conn.Close()
		c.conn = nil
		return err
	}

	return nil
}

func (c *SyslogClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
