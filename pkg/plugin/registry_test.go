package plugin

import (
	"context"
	"testing"
	"time"
)

// mockPlugin is a test implementation of TestPlugin
type mockPlugin struct {
	name        string
	description string
	runFunc     func(ctx context.Context, params Params) (Result, error)
}

func (m *mockPlugin) Name() string {
	return m.name
}

func (m *mockPlugin) Description() string {
	return m.description
}

func (m *mockPlugin) Run(ctx context.Context, params Params) (Result, error) {
	if m.runFunc != nil {
		return m.runFunc(ctx, params)
	}
	return Result{Success: true}, nil
}

func (m *mockPlugin) ValidateParams(params Params) error {
	return nil
}

func (m *mockPlugin) DefaultParams() Params {
	return Params{
		Duration: 60 * time.Second,
		Threads:  1,
		Config:   make(map[string]interface{}),
	}
}

func TestRegistry(t *testing.T) {
	// Create a new registry for testing
	registry := NewRegistry()

	// Test registering a plugin
	plugin1 := &mockPlugin{name: "test1", description: "Test plugin 1"}
	err := registry.Register(plugin1)
	if err != nil {
		t.Fatalf("Failed to register plugin: %v", err)
	}

	// Test registering duplicate plugin
	err = registry.Register(plugin1)
	if err == nil {
		t.Fatal("Expected error when registering duplicate plugin")
	}

	// Test registering nil plugin
	err = registry.Register(nil)
	if err == nil {
		t.Fatal("Expected error when registering nil plugin")
	}

	// Test registering plugin with empty name
	plugin2 := &mockPlugin{name: "", description: "No name"}
	err = registry.Register(plugin2)
	if err == nil {
		t.Fatal("Expected error when registering plugin with empty name")
	}

	// Test getting plugin
	got, err := registry.Get("test1")
	if err != nil {
		t.Fatalf("Failed to get plugin: %v", err)
	}
	if got.Name() != "test1" {
		t.Errorf("Got wrong plugin: expected test1, got %s", got.Name())
	}

	// Test getting non-existent plugin
	_, err = registry.Get("nonexistent")
	if err == nil {
		t.Fatal("Expected error when getting non-existent plugin")
	}

	// Test listing plugins
	plugin3 := &mockPlugin{name: "test3", description: "Test plugin 3"}
	registry.Register(plugin3)

	list := registry.List()
	if len(list) != 2 {
		t.Errorf("Expected 2 plugins, got %d", len(list))
	}

	// Verify list is sorted
	if list[0] != "test1" || list[1] != "test3" {
		t.Errorf("List not sorted correctly: %v", list)
	}

	// Test GetAll
	all := registry.GetAll()
	if len(all) != 2 {
		t.Errorf("Expected 2 plugins from GetAll, got %d", len(all))
	}

	// Test Clear
	registry.Clear()
	list = registry.List()
	if len(list) != 0 {
		t.Errorf("Expected 0 plugins after Clear, got %d", len(list))
	}
}

func TestGlobalRegistry(t *testing.T) {
	// Save current state
	originalPlugins := globalRegistry.plugins
	defer func() {
		globalRegistry.plugins = originalPlugins
	}()

	// Clear global registry
	globalRegistry.Clear()

	// Test global Register
	plugin := &mockPlugin{name: "global-test", description: "Global test plugin"}
	err := Register(plugin)
	if err != nil {
		t.Fatalf("Failed to register plugin globally: %v", err)
	}

	// Test global Get
	got, err := Get("global-test")
	if err != nil {
		t.Fatalf("Failed to get global plugin: %v", err)
	}
	if got.Name() != "global-test" {
		t.Errorf("Got wrong plugin: expected global-test, got %s", got.Name())
	}

	// Test global List
	list := List()
	if len(list) != 1 || list[0] != "global-test" {
		t.Errorf("Unexpected global list: %v", list)
	}
}

func TestPluginInfo(t *testing.T) {
	registry := NewRegistry()

	// Register a basic plugin
	plugin1 := &mockPlugin{name: "basic", description: "Basic plugin"}
	registry.Register(plugin1)

	// Get plugin info
	infos := registry.GetPluginInfo()
	if len(infos) != 1 {
		t.Fatalf("Expected 1 plugin info, got %d", len(infos))
	}

	info := infos[0]
	if info.Name != "basic" {
		t.Errorf("Expected name 'basic', got %s", info.Name)
	}
	if info.Description != "Basic plugin" {
		t.Errorf("Expected description 'Basic plugin', got %s", info.Description)
	}
}
