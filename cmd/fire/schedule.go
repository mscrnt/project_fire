package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/mscrnt/project_fire/pkg/db"
	"github.com/mscrnt/project_fire/pkg/plugin"
	_ "github.com/mscrnt/project_fire/pkg/plugin/cpu"    // Register CPU plugin
	_ "github.com/mscrnt/project_fire/pkg/plugin/memory" // Register Memory plugin
	"github.com/mscrnt/project_fire/pkg/schedule"
	"github.com/spf13/cobra"
)

func scheduleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schedule",
		Short: "Manage test schedules",
		Long:  "Create, manage, and run scheduled tests",
	}

	cmd.AddCommand(scheduleAddCmd())
	cmd.AddCommand(scheduleListCmd())
	cmd.AddCommand(scheduleRemoveCmd())
	cmd.AddCommand(scheduleEnableCmd())
	cmd.AddCommand(scheduleDisableCmd())
	cmd.AddCommand(scheduleStartCmd())
	cmd.AddCommand(scheduleShowCmd())

	return cmd
}

func scheduleAddCmd() *cobra.Command {
	var (
		name        string
		description string
		cronExpr    string
		pluginName  string
		config      map[string]string
		enabled     bool
	)

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a new schedule",
		Long: `Add a new test schedule with cron-style timing.

Cron expression format:
  ┌───────────── minute (0 - 59)
  │ ┌───────────── hour (0 - 23)
  │ │ ┌───────────── day of month (1 - 31)
  │ │ │ ┌───────────── month (1 - 12)
  │ │ │ │ ┌───────────── day of week (0 - 6) (Sunday to Saturday)
  │ │ │ │ │
  * * * * *

Examples:
  # Run CPU test every hour
  bench schedule add --name "Hourly CPU" --cron "0 * * * *" --plugin cpu

  # Run memory test daily at 2 AM
  bench schedule add --name "Daily Memory" --cron "0 2 * * *" --plugin memory --config size_mb=2048

  # Run stress test every Monday at 3:30 AM
  bench schedule add --name "Weekly Stress" --cron "30 3 * * 1" --plugin cpu --config threads=8`,
		RunE: func(_ *cobra.Command, _ []string) error {
			// Validate inputs
			if name == "" {
				return fmt.Errorf("schedule name is required")
			}
			if cronExpr == "" {
				return fmt.Errorf("cron expression is required")
			}
			if pluginName == "" {
				return fmt.Errorf("plugin name is required")
			}

			// Verify plugin exists
			if _, err := plugin.Get(pluginName); err != nil {
				return fmt.Errorf("plugin %s not found", pluginName)
			}

			// Open database
			dbPath := getDBPath()
			database, err := db.Open(dbPath)
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer func() { _ = database.Close() }()

			// Create schedule store
			store := schedule.NewStore(database)

			// Prepare parameters
			params := make(db.JSONData)
			for k, v := range config {
				// Try to parse as number
				if n, err := json.Number(v).Int64(); err == nil {
					params[k] = int(n)
				} else if f, err := json.Number(v).Float64(); err == nil {
					params[k] = f
				} else if v == "true" || v == "false" {
					params[k] = v == "true"
				} else {
					params[k] = v
				}
			}

			// Create schedule
			sched := &schedule.Schedule{
				Name:        name,
				Description: description,
				CronExpr:    cronExpr,
				Plugin:      pluginName,
				Params:      params,
				Enabled:     enabled,
			}

			if err := store.Create(sched); err != nil {
				return fmt.Errorf("failed to create schedule: %w", err)
			}

			fmt.Printf("Created schedule '%s' (ID: %d)\n", sched.Name, sched.ID)
			fmt.Printf("Cron: %s\n", sched.CronExpr)
			fmt.Printf("Plugin: %s\n", sched.Plugin)
			if sched.NextRunTime != nil {
				fmt.Printf("Next run: %s\n", sched.NextRunTime.Format("2006-01-02 15:04:05"))
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "Schedule name (required)")
	cmd.Flags().StringVarP(&description, "desc", "d", "", "Schedule description")
	cmd.Flags().StringVar(&cronExpr, "cron", "", "Cron expression (required)")
	cmd.Flags().StringVarP(&pluginName, "plugin", "p", "", "Plugin to run (required)")
	cmd.Flags().StringToStringVarP(&config, "config", "c", map[string]string{}, "Plugin configuration")
	cmd.Flags().BoolVar(&enabled, "enabled", true, "Enable schedule immediately")

	if err := cmd.MarkFlagRequired("name"); err != nil {
		// Log the error but don't fail - this is a development-time check
		fmt.Fprintf(os.Stderr, "Warning: failed to mark flag 'name' as required: %v\n", err)
	}
	if err := cmd.MarkFlagRequired("cron"); err != nil {
		// Log the error but don't fail - this is a development-time check
		fmt.Fprintf(os.Stderr, "Warning: failed to mark flag 'cron' as required: %v\n", err)
	}
	if err := cmd.MarkFlagRequired("plugin"); err != nil {
		// Log the error but don't fail - this is a development-time check
		fmt.Fprintf(os.Stderr, "Warning: failed to mark flag 'plugin' as required: %v\n", err)
	}

	return cmd
}

func scheduleListCmd() *cobra.Command {
	var (
		all      bool
		disabled bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List schedules",
		Long: `List all configured schedules.

Examples:
  # List enabled schedules
  bench schedule list

  # List all schedules
  bench schedule list --all`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Open database
			dbPath := getDBPath()
			database, err := db.Open(dbPath)
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer func() { _ = database.Close() }()

			// Create schedule store
			store := schedule.NewStore(database)

			// Build filter
			filter := schedule.Filter{}
			if !all && !disabled {
				enabled := true
				filter.Enabled = &enabled
			} else if disabled {
				enabled := false
				filter.Enabled = &enabled
			}

			// List schedules
			schedules, err := store.List(filter)
			if err != nil {
				return fmt.Errorf("failed to list schedules: %w", err)
			}

			if len(schedules) == 0 {
				fmt.Println("No schedules found")
				return nil
			}

			// Display schedules
			fmt.Printf("%-4s %-20s %-15s %-20s %-8s %-20s\n",
				"ID", "Name", "Plugin", "Cron", "Enabled", "Next Run")
			fmt.Println(strings.Repeat("-", 90))

			for _, sched := range schedules {
				nextRun := "N/A"
				if sched.NextRunTime != nil {
					if sched.IsOverdue() {
						nextRun = fmt.Sprintf("%s (overdue)", sched.NextRunTime.Format("2006-01-02 15:04"))
					} else {
						nextRun = sched.NextRunTime.Format("2006-01-02 15:04")
					}
				}

				fmt.Printf("%-4d %-20s %-15s %-20s %-8v %-20s\n",
					sched.ID,
					truncate(sched.Name, 20),
					sched.Plugin,
					sched.CronExpr,
					sched.Enabled,
					nextRun,
				)
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&all, "all", "a", false, "Show all schedules")
	cmd.Flags().BoolVar(&disabled, "disabled", false, "Show only disabled schedules")

	return cmd
}

func scheduleRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove [id|name]",
		Short: "Remove a schedule",
		Long: `Remove a schedule by ID or name.

Examples:
  bench schedule remove 1
  bench schedule remove "Hourly CPU Test"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Open database
			dbPath := getDBPath()
			database, err := db.Open(dbPath)
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer func() { _ = database.Close() }()

			// Create schedule store
			store := schedule.NewStore(database)

			// Try to parse as ID first
			var sched *schedule.Schedule
			if id, err := parseInt64(args[0]); err == nil {
				sched, err = store.Get(id)
				if err != nil {
					return fmt.Errorf("schedule with ID %d not found", id)
				}
			} else {
				// Try by name
				sched, err = store.GetByName(args[0])
				if err != nil {
					return fmt.Errorf("schedule '%s' not found", args[0])
				}
			}

			// Confirm deletion
			fmt.Printf("Delete schedule '%s' (ID: %d)? [y/N] ", sched.Name, sched.ID)
			var confirm string
			if _, err := fmt.Scanln(&confirm); err != nil {
				// Treat any error as a "no" response
				confirm = "n"
			}
			if !strings.EqualFold(confirm, "y") {
				fmt.Println("Cancelled")
				return nil
			}

			// Delete schedule
			if err := store.Delete(sched.ID); err != nil {
				return fmt.Errorf("failed to delete schedule: %w", err)
			}

			fmt.Printf("Deleted schedule '%s'\n", sched.Name)
			return nil
		},
	}

	return cmd
}

func scheduleEnableCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "enable [id|name]",
		Short: "Enable a schedule",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return toggleSchedule(args[0], true)
		},
	}
	return cmd
}

func scheduleDisableCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "disable [id|name]",
		Short: "Disable a schedule",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return toggleSchedule(args[0], false)
		},
	}
	return cmd
}

func toggleSchedule(identifier string, enable bool) error {
	// Open database
	dbPath := getDBPath()
	database, err := db.Open(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer func() { _ = database.Close() }()

	// Create schedule store
	store := schedule.NewStore(database)

	// Find schedule
	var sched *schedule.Schedule
	if id, err := parseInt64(identifier); err == nil {
		sched, err = store.Get(id)
		if err != nil {
			return fmt.Errorf("schedule with ID %d not found", id)
		}
	} else {
		sched, err = store.GetByName(identifier)
		if err != nil {
			return fmt.Errorf("schedule '%s' not found", identifier)
		}
	}

	// Toggle state
	if enable {
		if err := store.Enable(sched.ID); err != nil {
			return fmt.Errorf("failed to enable schedule: %w", err)
		}
		fmt.Printf("Enabled schedule '%s'\n", sched.Name)
	} else {
		if err := store.Disable(sched.ID); err != nil {
			return fmt.Errorf("failed to disable schedule: %w", err)
		}
		fmt.Printf("Disabled schedule '%s'\n", sched.Name)
	}

	return nil
}

func scheduleStartCmd() *cobra.Command {
	var (
		checkInterval time.Duration
		logFile       string
	)

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the scheduler daemon",
		Long: `Start the scheduler daemon to run tests automatically.

The scheduler will:
- Load all enabled schedules
- Execute tests according to their cron expressions
- Save results to the database
- Continue running until interrupted

Examples:
  # Start scheduler in foreground
  bench schedule start

  # Start with custom check interval
  bench schedule start --check-interval 30s

  # Start with log file
  bench schedule start --log scheduler.log`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Setup logging
			logger := log.New(os.Stdout, "[scheduler] ", log.LstdFlags)
			if logFile != "" {
				f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666)
				if err != nil {
					return fmt.Errorf("failed to open log file: %w", err)
				}
				defer func() { _ = f.Close() }()
				logger = log.New(f, "[scheduler] ", log.LstdFlags)
			}

			// Open database
			dbPath := getDBPath()
			database, err := db.Open(dbPath)
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer func() { _ = database.Close() }()

			// Create and start runner
			runner := schedule.NewRunner(database, logger)
			if err := runner.Start(); err != nil {
				return fmt.Errorf("failed to start scheduler: %w", err)
			}

			// Setup signal handling
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

			// Run check for overdue schedules periodically
			ticker := time.NewTicker(checkInterval)
			defer ticker.Stop()

			fmt.Println("Scheduler started. Press Ctrl+C to stop.")
			logger.Println("Scheduler daemon started")

			// Main loop
			for {
				select {
				case <-sigChan:
					logger.Println("Received shutdown signal")
					runner.Stop()
					return nil

				case <-ticker.C:
					if err := runner.CheckDue(); err != nil {
						logger.Printf("Error checking due schedules: %v", err)
					}
				}
			}
		},
	}

	cmd.Flags().DurationVar(&checkInterval, "check-interval", 60*time.Second, "Interval to check for overdue schedules")
	cmd.Flags().StringVar(&logFile, "log", "", "Log file path (default: stdout)")

	return cmd
}

func scheduleShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show [id|name]",
		Short: "Show schedule details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Open database
			dbPath := getDBPath()
			database, err := db.Open(dbPath)
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer func() { _ = database.Close() }()

			// Create schedule store
			store := schedule.NewStore(database)

			// Find schedule
			var sched *schedule.Schedule
			if id, err := parseInt64(args[0]); err == nil {
				sched, err = store.Get(id)
				if err != nil {
					return fmt.Errorf("schedule with ID %d not found", id)
				}
			} else {
				sched, err = store.GetByName(args[0])
				if err != nil {
					return fmt.Errorf("schedule '%s' not found", args[0])
				}
			}

			// Display details
			fmt.Printf("Schedule: %s (ID: %d)\n", sched.Name, sched.ID)
			if sched.Description != "" {
				fmt.Printf("Description: %s\n", sched.Description)
			}
			fmt.Printf("Plugin: %s\n", sched.Plugin)
			fmt.Printf("Cron Expression: %s\n", sched.CronExpr)
			fmt.Printf("Enabled: %v\n", sched.Enabled)
			fmt.Printf("Created: %s\n", sched.CreatedAt.Format("2006-01-02 15:04:05"))
			fmt.Printf("Updated: %s\n", sched.UpdatedAt.Format("2006-01-02 15:04:05"))

			if sched.LastRunTime != nil {
				fmt.Printf("\nLast Run: %s\n", sched.LastRunTime.Format("2006-01-02 15:04:05"))
				if sched.LastRunID != nil {
					fmt.Printf("Last Run ID: %d\n", *sched.LastRunID)
				}
			} else {
				fmt.Printf("\nLast Run: Never\n")
			}

			if sched.NextRunTime != nil {
				fmt.Printf("Next Run: %s", sched.NextRunTime.Format("2006-01-02 15:04:05"))
				if sched.IsOverdue() {
					fmt.Printf(" (OVERDUE)")
				}
				fmt.Println()
			}

			if len(sched.Params) > 0 {
				fmt.Printf("\nParameters:\n")
				for k, v := range sched.Params {
					fmt.Printf("  %s: %v\n", k, v)
				}
			}

			return nil
		},
	}

	return cmd
}

// Helper functions
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}

func parseInt64(s string) (int64, error) {
	var id int64
	_, err := fmt.Sscanf(s, "%d", &id)
	return id, err
}
