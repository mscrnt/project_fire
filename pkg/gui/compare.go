package gui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/mscrnt/project_fire/pkg/db"
)

// Compare represents the run comparison view
type Compare struct {
	content fyne.CanvasObject
	dbPath  string

	// UI elements
	run1Select  *widget.Select
	run2Select  *widget.Select
	compareBtn  *widget.Button
	resultLabel *widget.Label

	// Data
	runs        []*db.Run
	run1Results []*db.Result
	run2Results []*db.Result
}

// NewCompare creates a new compare view
func NewCompare(dbPath string) *Compare {
	c := &Compare{
		dbPath: dbPath,
	}
	c.build()
	return c
}

// build creates the compare UI
func (c *Compare) build() {
	// Run selectors
	c.run1Select = widget.NewSelect([]string{}, func(value string) {
		c.compareBtn.Enable()
	})
	c.run1Select.PlaceHolder = "Select first run..."

	c.run2Select = widget.NewSelect([]string{}, func(value string) {
		c.compareBtn.Enable()
	})
	c.run2Select.PlaceHolder = "Select second run..."

	c.compareBtn = widget.NewButton("Compare", c.compareRuns)
	c.compareBtn.Disable()
	c.compareBtn.Importance = widget.HighImportance

	selectionCard := widget.NewCard("Select Runs to Compare", "",
		container.NewVBox(
			container.NewGridWithColumns(2,
				container.NewVBox(
					widget.NewLabel("Run 1:"),
					c.run1Select,
				),
				container.NewVBox(
					widget.NewLabel("Run 2:"),
					c.run2Select,
				),
			),
			c.compareBtn,
		),
	)

	// Results area
	c.resultLabel = widget.NewLabel("Select two runs to compare their metrics")
	c.resultLabel.Wrapping = fyne.TextWrapWord

	resultScroll := container.NewScroll(c.resultLabel)

	// Layout
	c.content = container.NewBorder(
		selectionCard, nil, nil, nil,
		widget.NewCard("Comparison Results", "", resultScroll),
	)

	// Load runs
	c.loadRuns()
}

// Content returns the compare content
func (c *Compare) Content() fyne.CanvasObject {
	return c.content
}

// Refresh refreshes the compare view
func (c *Compare) Refresh() {
	c.loadRuns()
}

// loadRuns loads available runs
func (c *Compare) loadRuns() {
	database, err := db.Open(c.dbPath)
	if err != nil {
		return
	}
	defer database.Close()

	// Load successful runs only
	success := true
	runs, err := database.ListRuns(db.RunFilter{
		Success: &success,
		Limit:   100,
	})
	if err != nil {
		return
	}

	c.runs = runs

	// Update selectors
	options := make([]string, len(runs))
	for i, run := range runs {
		options[i] = fmt.Sprintf("#%d - %s (%s)",
			run.ID,
			run.Plugin,
			run.StartTime.Format("2006-01-02 15:04"))
	}

	c.run1Select.Options = options
	c.run2Select.Options = options
	c.run1Select.Refresh()
	c.run2Select.Refresh()
}

// compareRuns compares the selected runs
func (c *Compare) compareRuns() {
	// Get selected run IDs
	idx1 := c.run1Select.SelectedIndex()
	idx2 := c.run2Select.SelectedIndex()

	if idx1 < 0 || idx2 < 0 || idx1 >= len(c.runs) || idx2 >= len(c.runs) {
		return
	}

	run1 := c.runs[idx1]
	run2 := c.runs[idx2]

	// Load results
	database, err := db.Open(c.dbPath)
	if err != nil {
		c.resultLabel.SetText("Error: Failed to open database")
		return
	}
	defer database.Close()

	results1, err := database.GetResults(run1.ID)
	if err != nil {
		c.resultLabel.SetText("Error: Failed to load results for run 1")
		return
	}

	results2, err := database.GetResults(run2.ID)
	if err != nil {
		c.resultLabel.SetText("Error: Failed to load results for run 2")
		return
	}

	// Compare results
	comparison := fmt.Sprintf("Comparison Results\n\n")
	comparison += fmt.Sprintf("Run 1: #%d (%s) - %s\n", run1.ID, run1.Plugin, run1.StartTime.Format("2006-01-02 15:04"))
	comparison += fmt.Sprintf("Run 2: #%d (%s) - %s\n\n", run2.ID, run2.Plugin, run2.StartTime.Format("2006-01-02 15:04"))

	// Compare durations
	if run1.EndTime != nil && run2.EndTime != nil {
		dur1 := run1.Duration()
		dur2 := run2.Duration()
		diff := dur2 - dur1
		comparison += fmt.Sprintf("Duration:\n")
		comparison += fmt.Sprintf("  Run 1: %s\n", formatDuration(dur1))
		comparison += fmt.Sprintf("  Run 2: %s\n", formatDuration(dur2))
		comparison += fmt.Sprintf("  Difference: %s (%.1f%%)\n\n",
			formatDuration(diff),
			(float64(diff)/float64(dur1))*100)
	}

	// Compare metrics
	metrics1 := make(map[string]*db.Result)
	for _, r := range results1 {
		metrics1[r.Metric] = r
	}

	metrics2 := make(map[string]*db.Result)
	for _, r := range results2 {
		metrics2[r.Metric] = r
	}

	comparison += "Metrics Comparison:\n"

	// Find common metrics
	for name, r1 := range metrics1 {
		if r2, ok := metrics2[name]; ok {
			diff := r2.Value - r1.Value
			pctChange := (diff / r1.Value) * 100

			comparison += fmt.Sprintf("\n%s:\n", name)
			comparison += fmt.Sprintf("  Run 1: %.2f %s\n", r1.Value, r1.Unit)
			comparison += fmt.Sprintf("  Run 2: %.2f %s\n", r2.Value, r2.Unit)
			comparison += fmt.Sprintf("  Change: %.2f (%.1f%%)\n", diff, pctChange)

			if pctChange > 0 {
				comparison += "  ↑ Improved\n"
			} else if pctChange < 0 {
				comparison += "  ↓ Degraded\n"
			} else {
				comparison += "  = No change\n"
			}
		}
	}

	// Find unique metrics
	for name := range metrics1 {
		if _, ok := metrics2[name]; !ok {
			comparison += fmt.Sprintf("\n%s: Only in Run 1\n", name)
		}
	}

	for name := range metrics2 {
		if _, ok := metrics1[name]; !ok {
			comparison += fmt.Sprintf("\n%s: Only in Run 2\n", name)
		}
	}

	c.resultLabel.SetText(comparison)
}
