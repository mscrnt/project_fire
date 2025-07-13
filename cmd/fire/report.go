package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mscrnt/project_fire/pkg/db"
	"github.com/mscrnt/project_fire/pkg/report"
	"github.com/spf13/cobra"
)

func reportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "report",
		Short: "Generate test reports",
		Long:  "Generate HTML and PDF reports from test results",
	}

	cmd.AddCommand(reportGenerateCmd())
	cmd.AddCommand(reportListCmd())

	return cmd
}

func reportGenerateCmd() *cobra.Command {
	var (
		format    string
		output    string
		runID     int64
		latest    bool
		plugin    string
		landscape bool
		pageSize  string
	)

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate a report",
		Long: `Generate an HTML or PDF report from test results.

Examples:
  # Generate HTML report for latest run
  bench report generate --latest

  # Generate PDF report for specific run
  bench report generate --run 42 --format pdf --output report.pdf

  # Generate report for latest CPU test
  bench report generate --latest --plugin cpu

  # Generate landscape PDF with custom page size
  bench report generate --run 10 --format pdf --landscape --page-size A4`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Validate inputs
			if !latest && runID == 0 {
				return fmt.Errorf("either --latest or --run must be specified")
			}

			if format != "html" && format != "pdf" {
				return fmt.Errorf("format must be either 'html' or 'pdf'")
			}

			// Open database
			dbPath := getDBPath()
			database, err := db.Open(dbPath)
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer func() { _ = database.Close() }()

			// Find run ID
			if latest {
				runs, err := database.ListRuns(db.RunFilter{
					Plugin: plugin,
					Limit:  1,
				})
				if err != nil {
					return fmt.Errorf("failed to list runs: %w", err)
				}
				if len(runs) == 0 {
					return fmt.Errorf("no runs found")
				}
				runID = runs[0].ID
			}

			// Verify run exists
			run, err := database.GetRun(runID)
			if err != nil {
				return fmt.Errorf("run %d not found", runID)
			}

			// Create report generator
			generator := report.NewGenerator(database)

			// Generate output filename if not specified
			if output == "" {
				timestamp := time.Now().Format("20060102_150405")
				output = fmt.Sprintf("fire_report_%d_%s.%s", runID, timestamp, format)
			}

			// Generate report
			switch format {
			case "html":
				html, err := generator.GenerateHTML(runID)
				if err != nil {
					return fmt.Errorf("failed to generate HTML report: %w", err)
				}

				// Write to file
				if err := os.WriteFile(output, []byte(html), 0600); err != nil {
					return fmt.Errorf("failed to write HTML file: %w", err)
				}

			case "pdf":
				// Prepare PDF options
				options := report.DefaultPDFOptions()
				options.Landscape = landscape

				// Parse page size
				if pageSize != "" {
					switch strings.ToUpper(pageSize) {
					case "A4":
						options.PaperWidth = 8.27
						options.PaperHeight = 11.69
					case "A3":
						options.PaperWidth = 11.69
						options.PaperHeight = 16.54
					case "LETTER":
						// Default is already Letter
					case "LEGAL":
						options.PaperWidth = 8.5
						options.PaperHeight = 14.0
					default:
						return fmt.Errorf("unsupported page size: %s", pageSize)
					}
				}

				// Generate PDF
				if err := generator.GeneratePDF(runID, output, options); err != nil {
					return fmt.Errorf("failed to generate PDF report: %w", err)
				}
			}

			// Get absolute path for display
			absPath, _ := filepath.Abs(output)

			fmt.Printf("Generated %s report for run #%d\n", strings.ToUpper(format), runID)
			fmt.Printf("Plugin: %s\n", run.Plugin)
			fmt.Printf("Date: %s\n", run.StartTime.Format("2006-01-02 15:04:05"))
			fmt.Printf("Status: %s\n", formatStatus(run.Success))
			fmt.Printf("Output: %s\n", absPath)

			return nil
		},
	}

	cmd.Flags().StringVarP(&format, "format", "f", "html", "Output format (html or pdf)")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file path")
	cmd.Flags().Int64Var(&runID, "run", 0, "Run ID to generate report for")
	cmd.Flags().BoolVar(&latest, "latest", false, "Use latest run")
	cmd.Flags().StringVarP(&plugin, "plugin", "p", "", "Filter by plugin when using --latest")
	cmd.Flags().BoolVar(&landscape, "landscape", false, "Generate PDF in landscape mode")
	cmd.Flags().StringVar(&pageSize, "page-size", "LETTER", "PDF page size (A3, A4, LETTER, LEGAL)")

	return cmd
}

func reportListCmd() *cobra.Command {
	var (
		plugin  string
		success bool
		failed  bool
		limit   int
		since   string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available runs for reporting",
		Long: `List test runs that can be used to generate reports.

Examples:
  # List all runs
  bench report list

  # List successful runs only
  bench report list --success

  # List failed runs for CPU plugin
  bench report list --plugin cpu --failed

  # List runs from last 24 hours
  bench report list --since 24h`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Open database
			dbPath := getDBPath()
			database, err := db.Open(dbPath)
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer func() { _ = database.Close() }()

			// Build filter
			filter := db.RunFilter{
				Plugin: plugin,
				Limit:  limit,
			}

			// Handle success/failed flags
			if cmd.Flags().Changed("success") {
				val := true
				filter.Success = &val
			} else if cmd.Flags().Changed("failed") {
				val := false
				filter.Success = &val
			}

			// Parse since duration
			if since != "" {
				duration, err := parseDuration(since)
				if err != nil {
					return fmt.Errorf("invalid duration: %w", err)
				}
				sinceTime := time.Now().Add(-duration)
				filter.StartTime = &sinceTime
			}

			// List runs
			runs, err := database.ListRuns(filter)
			if err != nil {
				return fmt.Errorf("failed to list runs: %w", err)
			}

			if len(runs) == 0 {
				fmt.Println("No runs found")
				return nil
			}

			// Display runs
			fmt.Printf("%-6s %-15s %-20s %-20s %-8s %-10s\n",
				"ID", "Plugin", "Start Time", "End Time", "Status", "Duration")
			fmt.Println(strings.Repeat("-", 85))

			for _, run := range runs {
				endTime := "Running"
				duration := "N/A"
				if run.EndTime != nil {
					endTime = run.EndTime.Format("2006-01-02 15:04:05")
					duration = formatDuration(run.EndTime.Sub(run.StartTime))
				}

				fmt.Printf("%-6d %-15s %-20s %-20s %-8s %-10s\n",
					run.ID,
					run.Plugin,
					run.StartTime.Format("2006-01-02 15:04:05"),
					endTime,
					formatStatus(run.Success),
					duration,
				)
			}

			fmt.Printf("\nTotal: %d runs\n", len(runs))

			return nil
		},
	}

	cmd.Flags().StringVarP(&plugin, "plugin", "p", "", "Filter by plugin")
	cmd.Flags().BoolVar(&success, "success", false, "Show only successful runs")
	cmd.Flags().BoolVar(&failed, "failed", false, "Show only failed runs")
	cmd.Flags().IntVar(&limit, "limit", 50, "Maximum number of runs to show")
	cmd.Flags().StringVar(&since, "since", "", "Show runs since duration (e.g., 24h, 7d)")

	return cmd
}

// Helper functions
func formatStatus(success bool) string {
	if success {
		return "PASSED"
	}
	return "FAILED"
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	return fmt.Sprintf("%.1fh", d.Hours())
}

func parseDuration(s string) (time.Duration, error) {
	// Handle simple formats like "24h", "7d"
	if strings.HasSuffix(s, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(s, "d"))
		if err != nil {
			return 0, err
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}

	// Try standard duration parsing
	return time.ParseDuration(s)
}
