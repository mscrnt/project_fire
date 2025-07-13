package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/mscrnt/project_fire/pkg/db"
	"github.com/spf13/cobra"
)

var (
	exportRunID  int64
	exportOutput string
	exportAll    bool
)

func exportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export test results",
		Long:  "Export test results in various formats",
	}

	cmd.AddCommand(exportCSVCmd())
	cmd.AddCommand(exportJSONCmd())

	return cmd
}

func exportCSVCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "csv",
		Short: "Export results to CSV format",
		Long: `Export test results to CSV format.

Examples:
  # Export specific run to file
  bench export csv --run 42 --out results.csv

  # Export specific run to stdout
  bench export csv --run 42

  # Export all runs
  bench export csv --all --out all-results.csv`,
		RunE: runExportCSV,
	}

	cmd.Flags().Int64Var(&exportRunID, "run", 0, "Run ID to export")
	cmd.Flags().StringVarP(&exportOutput, "out", "o", "", "Output file (default: stdout)")
	cmd.Flags().BoolVar(&exportAll, "all", false, "Export all runs")

	return cmd
}

func exportJSONCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "json",
		Short: "Export results to JSON format",
		Long: `Export test results to JSON format.

Examples:
  # Export specific run to file
  bench export json --run 42 --out results.json

  # Export specific run to stdout
  bench export json --run 42`,
		RunE: runExportJSON,
	}

	cmd.Flags().Int64Var(&exportRunID, "run", 0, "Run ID to export")
	cmd.Flags().StringVarP(&exportOutput, "out", "o", "", "Output file (default: stdout)")

	return cmd
}

func runExportCSV(_ *cobra.Command, _ []string) error {
	// Validate flags
	if !exportAll && exportRunID == 0 {
		return fmt.Errorf("either --run or --all must be specified")
	}

	// Open database
	dbPath := getDBPath()
	database, err := db.Open(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer func() { _ = database.Close() }()

	// Prepare output writer
	var out *os.File
	if exportOutput == "" {
		out = os.Stdout
	} else {
		out, err = os.Create(exportOutput) // #nosec G304 -- exportOutput is a user-specified output file path from command line flag
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer func() { _ = out.Close() }()
	}

	// Export data
	if exportAll {
		if err := database.ExportAllCSV(out); err != nil {
			return fmt.Errorf("failed to export CSV: %w", err)
		}
		if exportOutput != "" {
			fmt.Printf("Exported all runs to %s\n", exportOutput)
		}
	} else {
		// Check if run exists
		if _, err := database.GetRun(exportRunID); err != nil {
			return fmt.Errorf("run %d not found", exportRunID)
		}

		if err := database.ExportCSV(out, exportRunID); err != nil {
			return fmt.Errorf("failed to export CSV: %w", err)
		}
		if exportOutput != "" {
			fmt.Printf("Exported run %d to %s\n", exportRunID, exportOutput)
		}
	}

	return nil
}

func runExportJSON(_ *cobra.Command, _ []string) error {
	// Validate flags
	if exportRunID == 0 {
		return fmt.Errorf("--run must be specified")
	}

	// Open database
	dbPath := getDBPath()
	database, err := db.Open(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer func() { _ = database.Close() }()

	// Check if run exists
	if _, err := database.GetRun(exportRunID); err != nil {
		return fmt.Errorf("run %d not found", exportRunID)
	}

	// Prepare output writer
	var out *os.File
	if exportOutput == "" {
		out = os.Stdout
	} else {
		out, err = os.Create(exportOutput) // #nosec G304 -- exportOutput is a user-specified output file path from command line flag
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer func() { _ = out.Close() }()
	}

	// Export data
	if err := database.ExportJSON(out, exportRunID); err != nil {
		return fmt.Errorf("failed to export JSON: %w", err)
	}

	if exportOutput != "" {
		fmt.Printf("Exported run %d to %s\n", exportRunID, exportOutput)
	}

	return nil
}

// Helper command to list runs
func listCmd() *cobra.Command {
	var (
		listPlugin  string
		listLimit   int
		listSuccess bool
		listFailed  bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List test runs",
		Long: `List test runs from the database.

Examples:
  # List all runs
  bench list

  # List only CPU test runs
  bench list --plugin cpu

  # List only failed runs
  bench list --failed

  # List last 10 runs
  bench list --limit 10`,
		RunE: func(_ *cobra.Command, _ []string) error {
			// Open database
			dbPath := getDBPath()
			database, err := db.Open(dbPath)
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer func() { _ = database.Close() }()

			// Build filter
			filter := db.RunFilter{
				Plugin: listPlugin,
				Limit:  listLimit,
			}

			if listSuccess && !listFailed {
				success := true
				filter.Success = &success
			} else if listFailed && !listSuccess {
				success := false
				filter.Success = &success
			}

			// Get runs
			runs, err := database.ListRuns(filter)
			if err != nil {
				return fmt.Errorf("failed to list runs: %w", err)
			}

			if len(runs) == 0 {
				fmt.Println("No runs found")
				return nil
			}

			// Display runs
			fmt.Printf("%-6s %-15s %-20s %-20s %-10s %-8s\n",
				"ID", "Plugin", "Start Time", "End Time", "Duration", "Status")
			fmt.Println(strings.Repeat("-", 80))

			for _, run := range runs {
				endTime := "running"
				duration := "-"
				status := "running"

				if run.EndTime != nil {
					endTime = run.EndTime.Format("2006-01-02 15:04:05")
					duration = fmt.Sprintf("%.1fs", run.Duration().Seconds())
					if run.Success {
						status = "success"
					} else {
						status = "failed"
					}
				}

				fmt.Printf("%-6d %-15s %-20s %-20s %-10s %-8s\n",
					run.ID,
					run.Plugin,
					run.StartTime.Format("2006-01-02 15:04:05"),
					endTime,
					duration,
					status,
				)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&listPlugin, "plugin", "p", "", "Filter by plugin name")
	cmd.Flags().IntVarP(&listLimit, "limit", "n", 50, "Maximum number of runs to show")
	cmd.Flags().BoolVar(&listSuccess, "success", false, "Show only successful runs")
	cmd.Flags().BoolVar(&listFailed, "failed", false, "Show only failed runs")

	return cmd
}

// Helper command to show run details
func showCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show [run-id]",
		Short: "Show detailed run information",
		Long: `Show detailed information about a specific test run.

Examples:
  # Show run details
  bench show 42

  # Show run with full output
  bench show 42 -v`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse run ID
			runID, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid run ID: %s", args[0])
			}

			// Open database
			dbPath := getDBPath()
			database, err := db.Open(dbPath)
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer func() { _ = database.Close() }()

			// Get run
			run, err := database.GetRun(runID)
			if err != nil {
				return fmt.Errorf("run %d not found", runID)
			}

			// Get results
			results, err := database.GetResults(runID)
			if err != nil {
				return fmt.Errorf("failed to get results: %w", err)
			}

			// Display run information
			fmt.Printf("Run ID: %d\n", run.ID)
			fmt.Printf("Plugin: %s\n", run.Plugin)
			fmt.Printf("Start Time: %s\n", run.StartTime.Format("2006-01-02 15:04:05"))

			if run.EndTime != nil {
				fmt.Printf("End Time: %s\n", run.EndTime.Format("2006-01-02 15:04:05"))
				fmt.Printf("Duration: %.2f seconds\n", run.Duration().Seconds())
			} else {
				fmt.Printf("End Time: (still running)\n")
			}

			fmt.Printf("Success: %v\n", run.Success)
			fmt.Printf("Exit Code: %d\n", run.ExitCode)

			if run.Error != "" {
				fmt.Printf("Error: %s\n", run.Error)
			}

			// Display parameters
			if len(run.Params) > 0 {
				fmt.Printf("\nParameters:\n")
				for k, v := range run.Params {
					fmt.Printf("  %s: %v\n", k, v)
				}
			}

			// Display results
			if len(results) > 0 {
				fmt.Printf("\nResults:\n")
				for _, result := range results {
					if result.Unit != "" {
						fmt.Printf("  %s: %.6f %s\n", result.Metric, result.Value, result.Unit)
					} else {
						fmt.Printf("  %s: %.6f\n", result.Metric, result.Value)
					}
				}
			}

			// Display output if verbose
			verbose, _ := cmd.Flags().GetBool("verbose")
			if verbose {
				if run.Stdout != "" {
					fmt.Printf("\nStandard Output:\n%s\n", run.Stdout)
				}
				if run.Stderr != "" {
					fmt.Printf("\nStandard Error:\n%s\n", run.Stderr)
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolP("verbose", "v", false, "Show full output")

	return cmd
}
