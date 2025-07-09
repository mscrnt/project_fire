package db

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
)

// ExportCSV exports results to CSV format
func (db *DB) ExportCSV(w io.Writer, runID int64) error {
	// Get run information
	run, err := db.GetRun(runID)
	if err != nil {
		return fmt.Errorf("failed to get run: %w", err)
	}
	
	// Get results
	results, err := db.GetResults(runID)
	if err != nil {
		return fmt.Errorf("failed to get results: %w", err)
	}
	
	// Create CSV writer
	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()
	
	// Write headers
	headers := []string{
		"Run ID", "Plugin", "Start Time", "End Time", "Duration (s)",
		"Success", "Exit Code", "Metric", "Value", "Unit",
	}
	if err := csvWriter.Write(headers); err != nil {
		return fmt.Errorf("failed to write headers: %w", err)
	}
	
	// Calculate duration
	duration := float64(0)
	if run.EndTime != nil {
		duration = run.EndTime.Sub(run.StartTime).Seconds()
	}
	
	// Write results
	for _, result := range results {
		row := []string{
			strconv.FormatInt(run.ID, 10),
			run.Plugin,
			run.StartTime.Format("2006-01-02 15:04:05"),
			"",
			fmt.Sprintf("%.3f", duration),
			strconv.FormatBool(run.Success),
			strconv.Itoa(run.ExitCode),
			result.Metric,
			fmt.Sprintf("%.6f", result.Value),
			result.Unit,
		}
		
		if run.EndTime != nil {
			row[3] = run.EndTime.Format("2006-01-02 15:04:05")
		}
		
		if err := csvWriter.Write(row); err != nil {
			return fmt.Errorf("failed to write row: %w", err)
		}
	}
	
	return nil
}

// ExportJSON exports results to JSON format
func (db *DB) ExportJSON(w io.Writer, runID int64) error {
	// Get run information
	run, err := db.GetRun(runID)
	if err != nil {
		return fmt.Errorf("failed to get run: %w", err)
	}
	
	// Get results
	results, err := db.GetResults(runID)
	if err != nil {
		return fmt.Errorf("failed to get results: %w", err)
	}
	
	// Create export structure
	export := struct {
		Run     *Run      `json:"run"`
		Results []*Result `json:"results"`
	}{
		Run:     run,
		Results: results,
	}
	
	// Encode to JSON
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(export); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}
	
	return nil
}

// ExportAllCSV exports all runs and results to CSV format
func (db *DB) ExportAllCSV(w io.Writer) error {
	// Get all runs
	runs, err := db.ListRuns(RunFilter{})
	if err != nil {
		return fmt.Errorf("failed to list runs: %w", err)
	}
	
	// Create CSV writer
	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()
	
	// Write headers
	headers := []string{
		"Run ID", "Plugin", "Start Time", "End Time", "Duration (s)",
		"Success", "Exit Code", "Metric", "Value", "Unit",
	}
	if err := csvWriter.Write(headers); err != nil {
		return fmt.Errorf("failed to write headers: %w", err)
	}
	
	// Write results for each run
	for _, run := range runs {
		results, err := db.GetResults(run.ID)
		if err != nil {
			return fmt.Errorf("failed to get results for run %d: %w", run.ID, err)
		}
		
		// Calculate duration
		duration := float64(0)
		if run.EndTime != nil {
			duration = run.EndTime.Sub(run.StartTime).Seconds()
		}
		
		// Write results
		for _, result := range results {
			row := []string{
				strconv.FormatInt(run.ID, 10),
				run.Plugin,
				run.StartTime.Format("2006-01-02 15:04:05"),
				"",
				fmt.Sprintf("%.3f", duration),
				strconv.FormatBool(run.Success),
				strconv.Itoa(run.ExitCode),
				result.Metric,
				fmt.Sprintf("%.6f", result.Value),
				result.Unit,
			}
			
			if run.EndTime != nil {
				row[3] = run.EndTime.Format("2006-01-02 15:04:05")
			}
			
			if err := csvWriter.Write(row); err != nil {
				return fmt.Errorf("failed to write row: %w", err)
			}
		}
	}
	
	return nil
}