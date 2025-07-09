package schedule

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/mscrnt/project_fire/pkg/db"
	"github.com/mscrnt/project_fire/pkg/plugin"
	"github.com/robfig/cron/v3"
)

// Runner manages scheduled test executions
type Runner struct {
	cron     *cron.Cron
	store    *Store
	database *db.DB
	jobs     map[int64]cron.EntryID
	mu       sync.RWMutex
	logger   *log.Logger
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewRunner creates a new schedule runner
func NewRunner(database *db.DB, logger *log.Logger) *Runner {
	if logger == nil {
		logger = log.Default()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Runner{
		cron:     cron.New(cron.WithParser(cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow))),
		store:    NewStore(database),
		database: database,
		jobs:     make(map[int64]cron.EntryID),
		logger:   logger,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start starts the scheduler
func (r *Runner) Start() error {
	r.logger.Println("Starting scheduler...")

	// Load all enabled schedules
	enabled := true
	schedules, err := r.store.List(ScheduleFilter{Enabled: &enabled})
	if err != nil {
		return fmt.Errorf("failed to load schedules: %w", err)
	}

	// Register each schedule
	for _, schedule := range schedules {
		if err := r.registerSchedule(schedule); err != nil {
			r.logger.Printf("Failed to register schedule %s: %v", schedule.Name, err)
		}
	}

	// Start cron scheduler
	r.cron.Start()

	r.logger.Printf("Scheduler started with %d active schedules", len(r.jobs))
	return nil
}

// Stop stops the scheduler
func (r *Runner) Stop() {
	r.logger.Println("Stopping scheduler...")
	
	// Cancel context
	r.cancel()
	
	// Stop cron scheduler
	ctx := r.cron.Stop()
	
	// Wait for running jobs to complete
	select {
	case <-ctx.Done():
		r.logger.Println("All jobs completed")
	case <-time.After(5 * time.Minute):
		r.logger.Println("Timeout waiting for jobs to complete")
	}

	r.logger.Println("Scheduler stopped")
}

// RegisterSchedule adds a schedule to the runner
func (r *Runner) RegisterSchedule(scheduleID int64) error {
	schedule, err := r.store.Get(scheduleID)
	if err != nil {
		return err
	}

	return r.registerSchedule(schedule)
}

// UnregisterSchedule removes a schedule from the runner
func (r *Runner) UnregisterSchedule(scheduleID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if entryID, exists := r.jobs[scheduleID]; exists {
		r.cron.Remove(entryID)
		delete(r.jobs, scheduleID)
		r.logger.Printf("Unregistered schedule ID %d", scheduleID)
	}

	return nil
}

// RefreshSchedule updates a schedule in the runner
func (r *Runner) RefreshSchedule(scheduleID int64) error {
	// Unregister existing job
	if err := r.UnregisterSchedule(scheduleID); err != nil {
		return err
	}

	// Re-register if enabled
	schedule, err := r.store.Get(scheduleID)
	if err != nil {
		return err
	}

	if schedule.Enabled {
		return r.registerSchedule(schedule)
	}

	return nil
}

// registerSchedule registers a schedule with the cron scheduler
func (r *Runner) registerSchedule(schedule *Schedule) error {
	if !schedule.Enabled {
		return nil
	}

	// Create job function
	job := r.createJob(schedule)

	// Add to cron
	entryID, err := r.cron.AddFunc(schedule.CronExpr, job)
	if err != nil {
		return fmt.Errorf("failed to add cron job: %w", err)
	}

	// Track job
	r.mu.Lock()
	r.jobs[schedule.ID] = entryID
	r.mu.Unlock()

	r.logger.Printf("Registered schedule '%s' (ID: %d) with cron expression: %s",
		schedule.Name, schedule.ID, schedule.CronExpr)

	return nil
}

// createJob creates a job function for a schedule
func (r *Runner) createJob(schedule *Schedule) func() {
	return func() {
		// Check context
		select {
		case <-r.ctx.Done():
			return
		default:
		}

		r.logger.Printf("Executing scheduled job: %s", schedule.Name)
		
		// Run in goroutine to not block scheduler
		go func() {
			if err := r.executeSchedule(schedule); err != nil {
				r.logger.Printf("Failed to execute schedule %s: %v", schedule.Name, err)
			}
		}()
	}
}

// executeSchedule executes a scheduled test
func (r *Runner) executeSchedule(schedule *Schedule) error {
	// Recover from panics
	defer func() {
		if p := recover(); p != nil {
			r.logger.Printf("Panic in schedule %s: %v", schedule.Name, p)
		}
	}()

	// Get plugin
	p, err := plugin.Get(schedule.Plugin)
	if err != nil {
		return fmt.Errorf("plugin not found: %w", err)
	}

	// Prepare parameters
	params := p.DefaultParams()
	
	// Apply saved parameters
	if schedule.Params != nil {
		// Convert JSONData to plugin.Params config
		if params.Config == nil {
			params.Config = make(map[string]interface{})
		}
		for k, v := range schedule.Params {
			params.Config[k] = v
		}
	}

	// Create run record
	run, err := r.database.CreateRun(schedule.Plugin, schedule.Params)
	if err != nil {
		return fmt.Errorf("failed to create run record: %w", err)
	}

	r.logger.Printf("Started run %d for schedule %s", run.ID, schedule.Name)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(r.ctx, params.Duration+30*time.Second)
	defer cancel()

	// Run the test
	startTime := time.Now()
	result, err := p.Run(ctx, params)
	endTime := time.Now()

	// Update run record
	run.EndTime = &endTime
	run.Success = result.Success
	run.Error = result.Error
	run.Stdout = result.Stdout
	run.Stderr = result.Stderr
	if err != nil {
		run.ExitCode = 1
		if run.Error == "" {
			run.Error = err.Error()
		}
	}

	if err := r.database.UpdateRun(run); err != nil {
		r.logger.Printf("Failed to update run record: %v", err)
	}

	// Save metrics
	if len(result.Metrics) > 0 {
		units := make(map[string]string)
		// Try to get units from plugin info
		if infoPlugin, ok := p.(interface{ Info() plugin.PluginInfo }); ok {
			info := infoPlugin.Info()
			for _, metric := range info.Metrics {
				units[metric.Name] = metric.Unit
			}
		}

		if err := r.database.CreateResults(run.ID, result.Metrics, units); err != nil {
			r.logger.Printf("Failed to save metrics: %v", err)
		}
	}

	// Update schedule's last run info
	if err := r.store.UpdateLastRun(schedule.ID, run.ID); err != nil {
		r.logger.Printf("Failed to update schedule last run: %v", err)
	}

	r.logger.Printf("Completed run %d for schedule %s (success: %v, duration: %s)",
		run.ID, schedule.Name, result.Success, endTime.Sub(startTime))

	return nil
}

// CheckDue runs any overdue schedules immediately
func (r *Runner) CheckDue() error {
	schedules, err := r.store.GetDue()
	if err != nil {
		return fmt.Errorf("failed to get due schedules: %w", err)
	}

	for _, schedule := range schedules {
		r.logger.Printf("Running overdue schedule: %s", schedule.Name)
		go func(s *Schedule) {
			if err := r.executeSchedule(s); err != nil {
				r.logger.Printf("Failed to execute overdue schedule %s: %v", s.Name, err)
			}
		}(schedule)
	}

	return nil
}

// ListJobs returns information about all scheduled jobs
func (r *Runner) ListJobs() []cron.Entry {
	return r.cron.Entries()
}