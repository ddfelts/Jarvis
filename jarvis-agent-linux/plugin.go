package main

import (
	"fmt"
	"jarvis-agent/configs/config"
	"path/filepath"
	"plugin"
	"sync"

	"github.com/0xrawsec/golang-utils/log"
)

// Plugin interface that all plugins must implement
type Plugin interface {
	Name() string
	Init(config *config.Config) error
	Start(wg *sync.WaitGroup)
	Stop()
}

// PluginManager handles plugin loading and lifecycle
type PluginManager struct {
	plugins map[string]Plugin
	config  *config.Config
	wg      *sync.WaitGroup
}

func NewPluginManager(config *config.Config, wg *sync.WaitGroup) *PluginManager {
	return &PluginManager{
		plugins: make(map[string]Plugin),
		config:  config,
		wg:      wg,
	}
}

// LoadPlugin loads a plugin from a .so file
func (pm *PluginManager) LoadPlugin(path string) error {
	// Load the plugin
	p, err := plugin.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open plugin %s: %v", path, err)
	}

	// Look up the "New" function
	newFunc, err := p.Lookup("New")
	if err != nil {
		return fmt.Errorf("failed to find New function in plugin %s: %v", path, err)
	}

	// Cast and call the function
	pluginConstructor, ok := newFunc.(func() Plugin)
	if !ok {
		return fmt.Errorf("plugin %s 'New' function has wrong signature", path)
	}

	plugin := pluginConstructor()
	if plugin == nil {
		return fmt.Errorf("plugin %s constructor returned nil", path)
	}

	// Initialize the plugin
	if err := plugin.Init(pm.config); err != nil {
		return fmt.Errorf("failed to initialize plugin %s: %v", path, err)
	}

	pm.plugins[plugin.Name()] = plugin
	return nil
}

// LoadPluginsFromDir loads all .so files from a directory
func (pm *PluginManager) LoadPluginsFromDir(dir string) error {
	files, err := filepath.Glob(filepath.Join(dir, "*.so"))
	if err != nil {
		return fmt.Errorf("failed to read plugin directory: %v", err)
	}

	for _, file := range files {
		if err := pm.LoadPlugin(file); err != nil {
			log.Warnf("Failed to load plugin %s: %v", file, err)
			continue
		}
	}
	return nil
}

// StartAll starts all loaded plugins
func (pm *PluginManager) StartAll() {
	for name, p := range pm.plugins {
		log.Infof("Starting plugin: %s", name)
		p.Start(pm.wg)
	}
}

// StopAll stops all plugins
func (pm *PluginManager) StopAll() {
	for name, p := range pm.plugins {
		log.Infof("Stopping plugin: %s", name)
		p.Stop()
	}
}
