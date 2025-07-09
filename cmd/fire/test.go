package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/mscrnt/project_fire/pkg/db"
	"github.com/mscrnt/project_fire/pkg/plugin"
	_ "github.com/mscrnt/project_fire/pkg/plugin/cpu"    // Register CPU plugin
	_ "github.com/mscrnt/project_fire/pkg/plugin/memory" // Register Memory plugin
	"github.com/spf13/cobra"
)

var (
	testPlugin   string
	testDuration time.Duration
	testThreads  int
	testConfig   map[string]string
	testDryRun   bool
	testList     bool
)

func createTestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test [plugin]",
		Short: "Run a system test",
		Long: `Execute various system tests including CPU, memory, disk, and GPU stress tests.

Examples:
  # List available plugins
  bench test --list

  # Run CPU stress test for 60 seconds
  bench test cpu --duration 60s

  # Run memory test with 2GB allocation
  bench test memory --config size_mb=2048

  # Dry run to see what would be executed
  bench test cpu --dry-run`,
		Args: cobra.MaximumNArgs(1),
		RunE: runTest,
	}

	cmd.Flags().StringVarP(&testPlugin, "plugin", "p", "", "Plugin to run (if not specified as argument)")
	cmd.Flags().DurationVarP(&testDuration, "duration", "d", 60*time.Second, "Test duration")
	cmd.Flags().IntVarP(&testThreads, "threads", "t", 0, "Number of threads (0 = auto)")
	cmd.Flags().StringToStringVarP(&testConfig, "config", "c", map[string]string{}, "Plugin configuration (key=value)")
	cmd.Flags().BoolVar(&testDryRun, "dry-run", false, "Show what would be executed without running")
	cmd.Flags().BoolVarP(&testList, "list", "l", false, "List available plugins")

	return cmd
}

func runTest(cmd *cobra.Command, args []string) error {
	// Handle list flag
	if testList {
		return listPlugins()
	}

	// Get plugin name
	pluginName := testPlugin
	if len(args) > 0 {
		pluginName = args[0]
	}

	if pluginName == "" {
		return fmt.Errorf("plugin name required")
	}

	// Get plugin from registry
	p, err := plugin.Get(pluginName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintf(os.Stderr, "\nAvailable plugins:\n")
		listPlugins()
		return err
	}

	// Prepare parameters
	params := p.DefaultParams()
	params.Duration = testDuration
	if testThreads > 0 {
		params.Threads = testThreads
	}

	// Apply config overrides
	if params.Config == nil {
		params.Config = make(map[string]interface{})
	}
	for k, v := range testConfig {
		// Try to parse as number
		if n, err := json.Number(v).Int64(); err == nil {
			params.Config[k] = int(n)
		} else if f, err := json.Number(v).Float64(); err == nil {
			params.Config[k] = f
		} else if v == "true" || v == "false" {
			params.Config[k] = v == "true"
		} else {
			params.Config[k] = v
		}
	}

	// Validate parameters
	if err := p.ValidateParams(params); err != nil {
		return fmt.Errorf("invalid parameters: %w", err)
	}

	// Dry run mode
	if testDryRun {
		fmt.Printf("Would run plugin: %s\n", p.Name())
		fmt.Printf("Description: %s\n", p.Description())
		fmt.Printf("Duration: %s\n", params.Duration)
		fmt.Printf("Threads: %d\n", params.Threads)
		fmt.Printf("Config:\n")
		for k, v := range params.Config {
			fmt.Printf("  %s: %v\n", k, v)
		}
		return nil
	}

	// Open database
	dbPath := getDBPath()
	database, err := db.Open(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	// Create run record
	run, err := database.CreateRun(pluginName, db.JSONData(params.Config))
	if err != nil {
		return fmt.Errorf("failed to create run record: %w", err)
	}

	fmt.Printf("Starting test: %s (run ID: %d)\n", p.Name(), run.ID)
	fmt.Printf("Duration: %s, Threads: %d\n", params.Duration, params.Threads)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), params.Duration+30*time.Second)
	defer cancel()

	// Run the test
	startTime := time.Now()
	result, err := p.Run(ctx, params)
	endTime := time.Now()

	// Update run record
	run.EndTime = &endTime
	run.Success = result.Success
	run.Error = result.Error
	run.Stdout = result.Stdout
	run.Stderr = result.Stderr
	if err != nil {
		run.ExitCode = 1
		if run.Error == "" {
			run.Error = err.Error()
		}
	}

	if err := database.UpdateRun(run); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to update run record: %v\n", err)
	}

	// Save metrics to database
	unitsMap := make(map[string]string)
	if len(result.Metrics) > 0 {
		// Try to get units from plugin info
		if infoPlugin, ok := p.(interface{ Info() plugin.PluginInfo }); ok {
			info := infoPlugin.Info()
			for _, metric := range info.Metrics {
				unitsMap[metric.Name] = metric.Unit
			}
		}

		if err := database.CreateResults(run.ID, result.Metrics, unitsMap); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to save metrics: %v\n", err)
		}
	}

	// Display results
	fmt.Printf("\nTest completed in %s\n", endTime.Sub(startTime))
	fmt.Printf("Success: %v\n", result.Success)

	if result.Error != "" {
		fmt.Printf("Error: %s\n", result.Error)
	}

	if len(result.Metrics) > 0 {
		fmt.Printf("\nMetrics:\n")
		for name, value := range result.Metrics {
			unit := ""
			if u, ok := unitsMap[name]; ok {
				unit = u
			}
			if unit != "" {
				fmt.Printf("  %s: %.2f %s\n", name, value, unit)
			} else {
				fmt.Printf("  %s: %.2f\n", name, value)
			}
		}
	}

	if len(result.Details) > 0 {
		fmt.Printf("\nDetails:\n")
		for k, v := range result.Details {
			fmt.Printf("  %s: %v\n", k, v)
		}
	}

	if err != nil {
		return err
	}

	return nil
}

func listPlugins() error {
	plugins := plugin.List()

	if len(plugins) == 0 {
		fmt.Println("No plugins registered")
		return nil
	}

	fmt.Println("Available plugins:")
	for _, name := range plugins {
		p, err := plugin.Get(name)
		if err != nil {
			continue
		}
		fmt.Printf("  %-15s %s\n", name, p.Description())
	}

	return nil
}
