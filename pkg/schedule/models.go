package schedule

import (
	"time"

	"github.com/mscrnt/project_fire/pkg/db"
)

// Schedule represents a scheduled test configuration
type Schedule struct {
	ID          int64       `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	CronExpr    string      `json:"cron_expr"`
	Plugin      string      `json:"plugin"`
	Params      db.JSONData `json:"params"`
	Enabled     bool        `json:"enabled"`
	LastRunID   *int64      `json:"last_run_id"`
	LastRunTime *time.Time  `json:"last_run_time"`
	NextRunTime *time.Time  `json:"next_run_time"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

// ScheduleFilter represents filters for querying schedules
type ScheduleFilter struct {
	Plugin  string
	Enabled *bool
	Limit   int
	Offset  int
}

// IsOverdue returns true if the schedule is overdue for execution
func (s *Schedule) IsOverdue() bool {
	if !s.Enabled || s.NextRunTime == nil {
		return false
	}
	return time.Now().After(*s.NextRunTime)
}

// ShouldRun returns true if the schedule should run now
func (s *Schedule) ShouldRun() bool {
	if !s.Enabled {
		return false
	}

	// If never run, should run
	if s.LastRunTime == nil {
		return true
	}

	// If next run time is set and passed, should run
	if s.NextRunTime != nil && time.Now().After(*s.NextRunTime) {
		return true
	}

	return false
}
