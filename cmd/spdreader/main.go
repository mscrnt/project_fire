//go:build windows
// +build windows

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mscrnt/project_fire/pkg/spdreader"
)

var (
	listFlag    = flag.Bool("list", false, "List all memory modules in a table")
	verboseFlag = flag.Bool("v", false, "Enable verbose logging")
	helpFlag    = flag.Bool("help", false, "Show help")
)

func main() {
	flag.Parse()

	if *helpFlag {
		printHelp()
		os.Exit(0)
	}

	// Set up logging
	if !*verboseFlag {
		log.SetOutput(os.Stderr)
		log.SetFlags(0)
	}

	// Set up signal handling for cleanup
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Create SPD reader instance
	reader, err := spdreader.New()
	if err != nil {
		log.Fatalf("Failed to initialize SPD reader: %v", err)
	}
	defer reader.Close()

	// Handle interrupt signal
	go func() {
		<-sigChan
		fmt.Fprintln(os.Stderr, "\nInterrupted, cleaning up...")
		reader.Close()
		os.Exit(1)
	}()

	// Read all modules
	modules, err := reader.ReadAllModules()
	if err != nil {
		log.Fatalf("Failed to read memory modules: %v", err)
	}

	if len(modules) == 0 {
		fmt.Println("No memory modules detected")
		os.Exit(0)
	}

	// Display results
	if *listFlag {
		printModulesTable(modules)
	} else {
		printModulesJSON(modules)
	}
}

func printHelp() {
	fmt.Println("SPD Reader - Read and parse SPD data from memory modules")
	fmt.Println()
	fmt.Println("Usage: spdreader [options]")
	fmt.Println()
	fmt.Println("Options:")
	flag.PrintDefaults()
	fmt.Println()
	fmt.Println("This tool requires Administrator privileges to access SMBus.")
}

func printModulesTable(modules []spdreader.SPDModule) {
	fmt.Printf("%-6s %-8s %-10s %-8s %-6s %-8s %-20s %-16s %-10s\n",
		"Slot", "Type", "Speed", "Size", "Ranks", "Width", "Manufacturer", "Part Number", "Serial")
	fmt.Println(string(make([]byte, 110)))

	for _, m := range modules {
		fmt.Printf("%-6d %-8s %-10s %-8s %-6d %-8s %-20s %-16s %-10s\n",
			m.Slot,
			m.Type,
			fmt.Sprintf("%d MT/s", m.DataRateMTs),
			fmt.Sprintf("%.0f GB", m.CapacityGB),
			m.Ranks,
			fmt.Sprintf("x%d", m.DataWidth),
			m.JEDECManufacturer,
			m.PartNumber,
			m.Serial)
	}
}

func printModulesJSON(modules []spdreader.SPDModule) {
	// Simple JSON output for programmatic use
	fmt.Println("[")
	for i, m := range modules {
		fmt.Printf("  {\n")
		fmt.Printf("    \"slot\": %d,\n", m.Slot)
		fmt.Printf("    \"type\": \"%s\",\n", m.Type)
		fmt.Printf("    \"dataRate\": %d,\n", m.DataRateMTs)
		fmt.Printf("    \"pcRate\": %d,\n", m.PCRate)
		fmt.Printf("    \"capacity\": %.1f,\n", m.CapacityGB)
		fmt.Printf("    \"ranks\": %d,\n", m.Ranks)
		fmt.Printf("    \"dataWidth\": %d,\n", m.DataWidth)
		fmt.Printf("    \"manufacturer\": \"%s\",\n", m.JEDECManufacturer)
		fmt.Printf("    \"partNumber\": \"%s\",\n", m.PartNumber)
		fmt.Printf("    \"serial\": \"%s\"\n", m.Serial)
		if i < len(modules)-1 {
			fmt.Printf("  },\n")
		} else {
			fmt.Printf("  }\n")
		}
	}
	fmt.Println("]")
}