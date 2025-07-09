package schedule

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/mscrnt/project_fire/pkg/db"
	"github.com/robfig/cron/v3"
)

// Store handles schedule persistence
type Store struct {
	db *db.DB
}

// NewStore creates a new schedule store
func NewStore(database *db.DB) *Store {
	return &Store{db: database}
}

// Create creates a new schedule
func (s *Store) Create(schedule *Schedule) error {
	// Validate cron expression
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	cronSchedule, err := parser.Parse(schedule.CronExpr)
	if err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	// Calculate next run time
	now := time.Now()
	nextRun := cronSchedule.Next(now)
	schedule.NextRunTime = &nextRun
	schedule.CreatedAt = now
	schedule.UpdatedAt = now

	result, err := s.db.Conn().Exec(
		`INSERT INTO schedules (name, description, cron_expr, plugin, params, enabled, next_run_time, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		schedule.Name, schedule.Description, schedule.CronExpr, schedule.Plugin,
		schedule.Params, schedule.Enabled, schedule.NextRunTime,
		schedule.CreatedAt, schedule.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create schedule: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	schedule.ID = id
	return nil
}

// Get retrieves a schedule by ID
func (s *Store) Get(id int64) (*Schedule, error) {
	schedule := &Schedule{}
	err := s.db.Conn().QueryRow(
		`SELECT id, name, description, cron_expr, plugin, params, enabled,
		 last_run_id, last_run_time, next_run_time, created_at, updated_at
		 FROM schedules WHERE id = ?`,
		id,
	).Scan(
		&schedule.ID, &schedule.Name, &schedule.Description,
		&schedule.CronExpr, &schedule.Plugin, &schedule.Params,
		&schedule.Enabled, &schedule.LastRunID, &schedule.LastRunTime,
		&schedule.NextRunTime, &schedule.CreatedAt, &schedule.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("schedule not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get schedule: %w", err)
	}
	return schedule, nil
}

// GetByName retrieves a schedule by name
func (s *Store) GetByName(name string) (*Schedule, error) {
	schedule := &Schedule{}
	err := s.db.Conn().QueryRow(
		`SELECT id, name, description, cron_expr, plugin, params, enabled,
		 last_run_id, last_run_time, next_run_time, created_at, updated_at
		 FROM schedules WHERE name = ?`,
		name,
	).Scan(
		&schedule.ID, &schedule.Name, &schedule.Description,
		&schedule.CronExpr, &schedule.Plugin, &schedule.Params,
		&schedule.Enabled, &schedule.LastRunID, &schedule.LastRunTime,
		&schedule.NextRunTime, &schedule.CreatedAt, &schedule.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("schedule not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get schedule: %w", err)
	}
	return schedule, nil
}

// List retrieves schedules based on filters
func (s *Store) List(filter ScheduleFilter) ([]*Schedule, error) {
	query := `SELECT id, name, description, cron_expr, plugin, params, enabled,
	          last_run_id, last_run_time, next_run_time, created_at, updated_at
	          FROM schedules WHERE 1=1`
	args := []interface{}{}

	if filter.Plugin != "" {
		query += " AND plugin = ?"
		args = append(args, filter.Plugin)
	}

	if filter.Enabled != nil {
		query += " AND enabled = ?"
		args = append(args, *filter.Enabled)
	}

	query += " ORDER BY name"

	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)

		if filter.Offset > 0 {
			query += " OFFSET ?"
			args = append(args, filter.Offset)
		}
	}

	rows, err := s.db.Conn().Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list schedules: %w", err)
	}
	defer rows.Close()

	var schedules []*Schedule
	for rows.Next() {
		schedule := &Schedule{}
		err := rows.Scan(
			&schedule.ID, &schedule.Name, &schedule.Description,
			&schedule.CronExpr, &schedule.Plugin, &schedule.Params,
			&schedule.Enabled, &schedule.LastRunID, &schedule.LastRunTime,
			&schedule.NextRunTime, &schedule.CreatedAt, &schedule.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan schedule: %w", err)
		}
		schedules = append(schedules, schedule)
	}

	return schedules, nil
}

// Update updates a schedule
func (s *Store) Update(schedule *Schedule) error {
	// Validate cron expression if changed
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	cronSchedule, err := parser.Parse(schedule.CronExpr)
	if err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	// Recalculate next run time
	now := time.Now()
	nextRun := cronSchedule.Next(now)
	schedule.NextRunTime = &nextRun
	schedule.UpdatedAt = now

	_, err = s.db.Conn().Exec(
		`UPDATE schedules SET name = ?, description = ?, cron_expr = ?, plugin = ?,
		 params = ?, enabled = ?, next_run_time = ?, updated_at = ?
		 WHERE id = ?`,
		schedule.Name, schedule.Description, schedule.CronExpr, schedule.Plugin,
		schedule.Params, schedule.Enabled, schedule.NextRunTime, schedule.UpdatedAt,
		schedule.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update schedule: %w", err)
	}
	return nil
}

// UpdateLastRun updates the last run information for a schedule
func (s *Store) UpdateLastRun(scheduleID int64, runID int64) error {
	// Get schedule to recalculate next run time
	schedule, err := s.Get(scheduleID)
	if err != nil {
		return err
	}

	// Parse cron expression
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	cronSchedule, err := parser.Parse(schedule.CronExpr)
	if err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	// Update last run and calculate next run
	now := time.Now()
	nextRun := cronSchedule.Next(now)

	_, err = s.db.Conn().Exec(
		`UPDATE schedules SET last_run_id = ?, last_run_time = ?, next_run_time = ?
		 WHERE id = ?`,
		runID, now, nextRun, scheduleID,
	)
	if err != nil {
		return fmt.Errorf("failed to update last run: %w", err)
	}
	return nil
}

// Enable enables a schedule
func (s *Store) Enable(id int64) error {
	// Get schedule to recalculate next run time
	schedule, err := s.Get(id)
	if err != nil {
		return err
	}

	// Parse cron expression
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	cronSchedule, err := parser.Parse(schedule.CronExpr)
	if err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	// Calculate next run from now
	now := time.Now()
	nextRun := cronSchedule.Next(now)

	_, err = s.db.Conn().Exec(
		`UPDATE schedules SET enabled = 1, next_run_time = ? WHERE id = ?`,
		nextRun, id,
	)
	if err != nil {
		return fmt.Errorf("failed to enable schedule: %w", err)
	}
	return nil
}

// Disable disables a schedule
func (s *Store) Disable(id int64) error {
	_, err := s.db.Conn().Exec(
		`UPDATE schedules SET enabled = 0 WHERE id = ?`,
		id,
	)
	if err != nil {
		return fmt.Errorf("failed to disable schedule: %w", err)
	}
	return nil
}

// Delete deletes a schedule
func (s *Store) Delete(id int64) error {
	_, err := s.db.Conn().Exec(
		`DELETE FROM schedules WHERE id = ?`,
		id,
	)
	if err != nil {
		return fmt.Errorf("failed to delete schedule: %w", err)
	}
	return nil
}

// GetDue returns all schedules that are due to run
func (s *Store) GetDue() ([]*Schedule, error) {
	now := time.Now()
	rows, err := s.db.Conn().Query(
		`SELECT id, name, description, cron_expr, plugin, params, enabled,
		 last_run_id, last_run_time, next_run_time, created_at, updated_at
		 FROM schedules 
		 WHERE enabled = 1 AND (next_run_time IS NULL OR next_run_time <= ?)
		 ORDER BY next_run_time`,
		now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get due schedules: %w", err)
	}
	defer rows.Close()

	var schedules []*Schedule
	for rows.Next() {
		schedule := &Schedule{}
		err := rows.Scan(
			&schedule.ID, &schedule.Name, &schedule.Description,
			&schedule.CronExpr, &schedule.Plugin, &schedule.Params,
			&schedule.Enabled, &schedule.LastRunID, &schedule.LastRunTime,
			&schedule.NextRunTime, &schedule.CreatedAt, &schedule.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan schedule: %w", err)
		}
		schedules = append(schedules, schedule)
	}

	return schedules, nil
}