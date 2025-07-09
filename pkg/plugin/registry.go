package plugin

import (
	"fmt"
	"sort"
	"sync"
)

// Registry manages all available test plugins
type Registry struct {
	mu      sync.RWMutex
	plugins map[string]TestPlugin
}

// globalRegistry is the default plugin registry
var globalRegistry = &Registry{
	plugins: make(map[string]TestPlugin),
}

// Register adds a plugin to the global registry
func Register(plugin TestPlugin) error {
	return globalRegistry.Register(plugin)
}

// Get retrieves a plugin from the global registry
func Get(name string) (TestPlugin, error) {
	return globalRegistry.Get(name)
}

// List returns all registered plugin names
func List() []string {
	return globalRegistry.List()
}

// Register adds a plugin to the registry
func (r *Registry) Register(plugin TestPlugin) error {
	if plugin == nil {
		return fmt.Errorf("plugin cannot be nil")
	}
	
	name := plugin.Name()
	if name == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}
	
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if _, exists := r.plugins[name]; exists {
		return fmt.Errorf("plugin %q already registered", name)
	}
	
	r.plugins[name] = plugin
	return nil
}

// Get retrieves a plugin by name
func (r *Registry) Get(name string) (TestPlugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	plugin, exists := r.plugins[name]
	if !exists {
		return nil, fmt.Errorf("plugin %q not found", name)
	}
	
	return plugin, nil
}

// List returns all registered plugin names in sorted order
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	names := make([]string, 0, len(r.plugins))
	for name := range r.plugins {
		names = append(names, name)
	}
	
	sort.Strings(names)
	return names
}

// GetAll returns all registered plugins
func (r *Registry) GetAll() map[string]TestPlugin {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	// Create a copy to avoid data races
	plugins := make(map[string]TestPlugin, len(r.plugins))
	for name, plugin := range r.plugins {
		plugins[name] = plugin
	}
	
	return plugins
}

// Clear removes all plugins from the registry
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.plugins = make(map[string]TestPlugin)
}

// NewRegistry creates a new plugin registry
func NewRegistry() *Registry {
	return &Registry{
		plugins: make(map[string]TestPlugin),
	}
}

// GetPluginInfo returns detailed information about all registered plugins
func GetPluginInfo() []PluginInfo {
	return globalRegistry.GetPluginInfo()
}

// GetPluginInfo returns detailed information about all plugins
func (r *Registry) GetPluginInfo() []PluginInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	var infos []PluginInfo
	for _, plugin := range r.plugins {
		info := PluginInfo{
			Name:        plugin.Name(),
			Description: plugin.Description(),
		}
		
		// Try to get additional info if the plugin implements an extended interface
		if extPlugin, ok := plugin.(interface{ Info() PluginInfo }); ok {
			info = extPlugin.Info()
		}
		
		infos = append(infos, info)
	}
	
	// Sort by name for consistent output
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Name < infos[j].Name
	})
	
	return infos
}