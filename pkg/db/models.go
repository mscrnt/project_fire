package db

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// Run represents a test execution record
type Run struct {
	ID        int64      `json:"id"`
	Plugin    string     `json:"plugin"`
	Params    JSONData   `json:"params"`
	StartTime time.Time  `json:"start_time"`
	EndTime   *time.Time `json:"end_time"`
	ExitCode  int        `json:"exit_code"`
	Success   bool       `json:"success"`
	Error     string     `json:"error,omitempty"`
	Stdout    string     `json:"stdout,omitempty"`
	Stderr    string     `json:"stderr,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// Result represents a metric result from a test run
type Result struct {
	ID        int64     `json:"id"`
	RunID     int64     `json:"run_id"`
	Metric    string    `json:"metric"`
	Value     float64   `json:"value"`
	Unit      string    `json:"unit"`
	CreatedAt time.Time `json:"created_at"`
}

// JSONData is a custom type for storing JSON in SQLite
type JSONData map[string]interface{}

// Value implements the driver.Valuer interface
func (j JSONData) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements the sql.Scanner interface
func (j *JSONData) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	var data []byte
	switch v := value.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("cannot scan type %T into JSONData", value)
	}

	return json.Unmarshal(data, j)
}

// RunStatus represents the status of a test run
type RunStatus string

const (
	RunStatusPending  RunStatus = "pending"
	RunStatusRunning  RunStatus = "running"
	RunStatusComplete RunStatus = "complete"
	RunStatusFailed   RunStatus = "failed"
)

// GetStatus returns the status of a run
func (r *Run) GetStatus() RunStatus {
	if r.EndTime == nil {
		if r.StartTime.IsZero() {
			return RunStatusPending
		}
		return RunStatusRunning
	}

	if r.Success {
		return RunStatusComplete
	}
	return RunStatusFailed
}

// Duration returns the duration of the run
func (r *Run) Duration() time.Duration {
	if r.EndTime == nil {
		return 0
	}
	return r.EndTime.Sub(r.StartTime)
}

// RunFilter represents filters for querying runs
type RunFilter struct {
	Plugin    string
	StartTime *time.Time
	EndTime   *time.Time
	Success   *bool
	Limit     int
	Offset    int
}

// ResultFilter represents filters for querying results
type ResultFilter struct {
	RunID  *int64
	Metric string
	Limit  int
	Offset int
}

// ExportFormat represents the format for exporting data
type ExportFormat string

const (
	ExportFormatCSV  ExportFormat = "csv"
	ExportFormatJSON ExportFormat = "json"
)
