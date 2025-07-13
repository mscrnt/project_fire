package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

// DB wraps the SQL database connection
type DB struct {
	conn *sql.DB
	path string
}

// Open creates or opens a SQLite database
func Open(path string) (*DB, error) {
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database connection
	conn, err := sql.Open("sqlite3", path+"?_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := conn.Ping(); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db := &DB{
		conn: conn,
		path: path,
	}

	// Run migrations
	if err := db.Migrate(); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return db, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// Conn returns the underlying database connection
func (db *DB) Conn() *sql.DB {
	return db.conn
}

// Path returns the database file path
func (db *DB) Path() string {
	return db.path
}

// Migrate creates or updates the database schema
func (db *DB) Migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS runs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		plugin TEXT NOT NULL,
		params TEXT,
		start_time DATETIME NOT NULL,
		end_time DATETIME,
		exit_code INTEGER DEFAULT 0,
		success BOOLEAN DEFAULT 0,
		error TEXT,
		stdout TEXT,
		stderr TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS results (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		run_id INTEGER NOT NULL,
		metric TEXT NOT NULL,
		value REAL NOT NULL,
		unit TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (run_id) REFERENCES runs(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS schedules (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		description TEXT,
		cron_expr TEXT NOT NULL,
		plugin TEXT NOT NULL,
		params TEXT,
		enabled BOOLEAN DEFAULT 1,
		last_run_id INTEGER,
		last_run_time DATETIME,
		next_run_time DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (last_run_id) REFERENCES runs(id) ON DELETE SET NULL
	);

	CREATE INDEX IF NOT EXISTS idx_runs_plugin ON runs(plugin);
	CREATE INDEX IF NOT EXISTS idx_runs_start_time ON runs(start_time);
	CREATE INDEX IF NOT EXISTS idx_runs_success ON runs(success);
	CREATE INDEX IF NOT EXISTS idx_results_run_id ON results(run_id);
	CREATE INDEX IF NOT EXISTS idx_results_metric ON results(metric);
	CREATE INDEX IF NOT EXISTS idx_schedules_enabled ON schedules(enabled);
	CREATE INDEX IF NOT EXISTS idx_schedules_next_run ON schedules(next_run_time);
	
	-- Trigger to update updated_at timestamp
	CREATE TRIGGER IF NOT EXISTS update_runs_timestamp 
	AFTER UPDATE ON runs
	BEGIN
		UPDATE runs SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
	END;

	CREATE TRIGGER IF NOT EXISTS update_schedules_timestamp 
	AFTER UPDATE ON schedules
	BEGIN
		UPDATE schedules SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
	END;
	`

	_, err := db.conn.Exec(schema)
	return err
}

// CreateRun creates a new test run record
func (db *DB) CreateRun(plugin string, params JSONData) (*Run, error) {
	run := &Run{
		Plugin:    plugin,
		Params:    params,
		StartTime: time.Now(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	result, err := db.conn.Exec(
		`INSERT INTO runs (plugin, params, start_time, created_at, updated_at) 
		 VALUES (?, ?, ?, ?, ?)`,
		run.Plugin, run.Params, run.StartTime, run.CreatedAt, run.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create run: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	run.ID = id
	return run, nil
}

// UpdateRun updates a test run record
func (db *DB) UpdateRun(run *Run) error {
	_, err := db.conn.Exec(
		`UPDATE runs SET 
		 end_time = ?, exit_code = ?, success = ?, error = ?, 
		 stdout = ?, stderr = ?, updated_at = ?
		 WHERE id = ?`,
		run.EndTime, run.ExitCode, run.Success, run.Error,
		run.Stdout, run.Stderr, time.Now(), run.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update run: %w", err)
	}
	return nil
}

// GetRun retrieves a run by ID
func (db *DB) GetRun(id int64) (*Run, error) {
	run := &Run{}
	err := db.conn.QueryRow(
		`SELECT id, plugin, params, start_time, end_time, exit_code, 
		 success, error, stdout, stderr, created_at, updated_at
		 FROM runs WHERE id = ?`,
		id,
	).Scan(
		&run.ID, &run.Plugin, &run.Params, &run.StartTime, &run.EndTime,
		&run.ExitCode, &run.Success, &run.Error, &run.Stdout, &run.Stderr,
		&run.CreatedAt, &run.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("run not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get run: %w", err)
	}
	return run, nil
}

// ListRuns retrieves runs based on filters
func (db *DB) ListRuns(filter RunFilter) ([]*Run, error) {
	query := `SELECT id, plugin, params, start_time, end_time, exit_code, 
	          success, error, stdout, stderr, created_at, updated_at
	          FROM runs WHERE 1=1`
	args := []interface{}{}

	if filter.Plugin != "" {
		query += " AND plugin = ?"
		args = append(args, filter.Plugin)
	}

	if filter.StartTime != nil {
		query += " AND start_time >= ?"
		args = append(args, filter.StartTime)
	}

	if filter.EndTime != nil {
		query += " AND start_time <= ?"
		args = append(args, filter.EndTime)
	}

	if filter.Success != nil {
		query += " AND success = ?"
		args = append(args, filter.Success)
	}

	query += " ORDER BY start_time DESC"

	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)

		if filter.Offset > 0 {
			query += " OFFSET ?"
			args = append(args, filter.Offset)
		}
	}

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list runs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var runs []*Run
	for rows.Next() {
		run := &Run{}
		err := rows.Scan(
			&run.ID, &run.Plugin, &run.Params, &run.StartTime, &run.EndTime,
			&run.ExitCode, &run.Success, &run.Error, &run.Stdout, &run.Stderr,
			&run.CreatedAt, &run.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan run: %w", err)
		}
		runs = append(runs, run)
	}

	return runs, nil
}

// CreateResult creates a new result record
func (db *DB) CreateResult(runID int64, metric string, value float64, unit string) error {
	_, err := db.conn.Exec(
		`INSERT INTO results (run_id, metric, value, unit) VALUES (?, ?, ?, ?)`,
		runID, metric, value, unit,
	)
	if err != nil {
		return fmt.Errorf("failed to create result: %w", err)
	}
	return nil
}

// CreateResults creates multiple result records in a transaction
func (db *DB) CreateResults(runID int64, metrics map[string]float64, units map[string]string) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		// Only rollback if we haven't committed
		_ = tx.Rollback()
	}()

	stmt, err := tx.Prepare(
		`INSERT INTO results (run_id, metric, value, unit) VALUES (?, ?, ?, ?)`,
	)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	for metric, value := range metrics {
		unit := units[metric]
		if _, err := stmt.Exec(runID, metric, value, unit); err != nil {
			return fmt.Errorf("failed to insert result %s: %w", metric, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetResults retrieves results for a run
func (db *DB) GetResults(runID int64) ([]*Result, error) {
	rows, err := db.conn.Query(
		`SELECT id, run_id, metric, value, unit, created_at
		 FROM results WHERE run_id = ? ORDER BY metric`,
		runID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get results: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var results []*Result
	for rows.Next() {
		result := &Result{}
		err := rows.Scan(
			&result.ID, &result.RunID, &result.Metric,
			&result.Value, &result.Unit, &result.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan result: %w", err)
		}
		results = append(results, result)
	}

	return results, nil
}

// ListResults retrieves results based on filters
func (db *DB) ListResults(filter ResultFilter) ([]*Result, error) {
	query := `SELECT id, run_id, metric, value, unit, created_at
	          FROM results WHERE 1=1`
	args := []interface{}{}

	if filter.RunID != nil {
		query += " AND run_id = ?"
		args = append(args, *filter.RunID)
	}

	if filter.Metric != "" {
		query += " AND metric = ?"
		args = append(args, filter.Metric)
	}

	query += " ORDER BY created_at DESC"

	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)

		if filter.Offset > 0 {
			query += " OFFSET ?"
			args = append(args, filter.Offset)
		}
	}

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list results: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var results []*Result
	for rows.Next() {
		result := &Result{}
		err := rows.Scan(
			&result.ID, &result.RunID, &result.Metric,
			&result.Value, &result.Unit, &result.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan result: %w", err)
		}
		results = append(results, result)
	}

	return results, nil
}
