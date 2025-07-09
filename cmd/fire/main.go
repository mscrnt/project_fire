package main

import (
	"fmt"
	"os"

	"github.com/mscrnt/project_fire/internal/version"
	"github.com/spf13/cobra"
)

var (
	// Build variables set by ldflags
	buildVersion string
	buildCommit  string
	buildTime    string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "bench",
		Short: "F.I.R.E. - Full Intensity Rigorous Evaluation",
		Long: `F.I.R.E. is a comprehensive PC test bench for burn-in tests, 
endurance stress testing, and benchmark analysis.`,
		Version: version.GetVersion(buildVersion, buildCommit, buildTime),
	}

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
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(version.GetDetailedVersion(buildVersion, buildCommit, buildTime))
		},
	}
}

