package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/mscrnt/project_fire/internal/version"
	"github.com/mscrnt/project_fire/pkg/telemetry"
	"github.com/spf13/cobra"
)

var (
	// Build variables set by ldflags
	buildVersion string
	buildCommit  string
	buildTime    string

	// Telemetry flags
	telemetryEnabled  bool
	telemetryEndpoint string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "bench",
		Short: "F.I.R.E. - Full Intensity Rigorous Evaluation",
		Long: `F.I.R.E. is a comprehensive PC test bench for burn-in tests, 
endurance stress testing, and benchmark analysis.`,
		Version: version.GetVersion(buildVersion, buildCommit, buildTime),
		PersistentPreRun: func(_ *cobra.Command, _ []string) {
			// Set app version for telemetry
			telemetry.SetAppVersion(version.GetVersion(buildVersion, buildCommit, buildTime))

			// Initialize telemetry based on flags
			telemetry.Initialize(telemetryEndpoint, "", telemetryEnabled)

			// Set up panic handler
			defer func() {
				if rec := recover(); rec != nil {
					stack := make([]byte, 32<<10)
					n := runtime.Stack(stack, false)
					telemetry.RecordPanic(rec, stack[:n])
					telemetry.Shutdown()
					panic(rec) // Re-panic to maintain default behavior
				}
			}()
		},
		PersistentPostRun: func(_ *cobra.Command, _ []string) {
			// Ensure telemetry is flushed on normal exit
			telemetry.Shutdown()
		},
	}

	// Add telemetry flags
	rootCmd.PersistentFlags().BoolVar(&telemetryEnabled, "telemetry", true, "Enable anonymous telemetry for hardware compatibility")
	rootCmd.PersistentFlags().StringVar(&telemetryEndpoint, "telemetry-endpoint", "", "Custom telemetry endpoint (default: https://firelogs.mscrnt.com/logs)")

	// Add commands
	rootCmd.AddCommand(versionCmd())
	rootCmd.AddCommand(createTestCmd())
	rootCmd.AddCommand(agentCmd())
	rootCmd.AddCommand(exportCmd())
	rootCmd.AddCommand(listCmd())
	rootCmd.AddCommand(showCmd())
	rootCmd.AddCommand(scheduleCmd())
	rootCmd.AddCommand(reportCmd())
	rootCmd.AddCommand(certCmd())
	rootCmd.AddCommand(guiCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Println(version.GetDetailedVersion(buildVersion, buildCommit, buildTime))
		},
	}
}
