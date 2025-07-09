package gui

import (
	"fmt"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/mscrnt/project_fire/pkg/db"
)

// History represents the test history view
type History struct {
	content fyne.CanvasObject
	dbPath  string

	// UI elements
	table *widget.Table
	runs  []*db.Run

	// Filters
	pluginFilter *widget.Select
	limitFilter  *widget.Select
}

// NewHistory creates a new history view
func NewHistory(dbPath string) *History {
	h := &History{
		dbPath: dbPath,
		runs:   make([]*db.Run, 0),
	}
	h.build()
	return h
}

// build creates the history UI
func (h *History) build() {
	// Create filters
	h.pluginFilter = widget.NewSelect([]string{"All", "cpu", "memory", "disk", "gpu"}, func(value string) {
		h.loadRuns()
	})
	h.pluginFilter.SetSelected("All")

	h.limitFilter = widget.NewSelect([]string{"50", "100", "250", "500"}, func(value string) {
		h.loadRuns()
	})
	h.limitFilter.SetSelected("50")

	filterBar := container.NewHBox(
		widget.NewLabel("Plugin:"),
		h.pluginFilter,
		widget.NewLabel("Limit:"),
		h.limitFilter,
		widget.NewButton("Refresh", h.Refresh),
	)

	// Create table
	h.table = widget.NewTable(
		func() (int, int) {
			return len(h.runs) + 1, 7 // +1 for header, 7 columns
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(i widget.TableCellID, o fyne.CanvasObject) {
			label := o.(*widget.Label)

			if i.Row == 0 {
				// Header row
				headers := []string{"ID", "Plugin", "Start Time", "Duration", "Status", "Exit Code", "Actions"}
				label.SetText(headers[i.Col])
				label.TextStyle = fyne.TextStyle{Bold: true}
			} else {
				// Data row
				run := h.runs[i.Row-1]
				switch i.Col {
				case 0:
					label.SetText(strconv.FormatInt(run.ID, 10))
				case 1:
					label.SetText(run.Plugin)
				case 2:
					label.SetText(run.StartTime.Format("2006-01-02 15:04:05"))
				case 3:
					if run.EndTime != nil {
						label.SetText(formatDuration(run.Duration()))
					} else {
						label.SetText("Running...")
					}
				case 4:
					if run.Success {
						label.SetText("✓ Passed")
					} else {
						label.SetText("✗ Failed")
					}
				case 5:
					label.SetText(strconv.Itoa(run.ExitCode))
				case 6:
					label.SetText("View")
				}
			}
		},
	)

	// Set column widths
	h.table.SetColumnWidth(0, 50)  // ID
	h.table.SetColumnWidth(1, 100) // Plugin
	h.table.SetColumnWidth(2, 150) // Start Time
	h.table.SetColumnWidth(3, 100) // Duration
	h.table.SetColumnWidth(4, 100) // Status
	h.table.SetColumnWidth(5, 80)  // Exit Code
	h.table.SetColumnWidth(6, 100) // Actions

	// Handle row selection
	h.table.OnSelected = func(id widget.TableCellID) {
		if id.Row > 0 && id.Col == 6 { // Actions column
			h.viewRunDetails(h.runs[id.Row-1])
		}
	}

	// Layout
	h.content = container.NewBorder(
		filterBar, nil, nil, nil,
		h.table,
	)

	// Load initial data
	h.loadRuns()
}

// Content returns the history content
func (h *History) Content() fyne.CanvasObject {
	return h.content
}

// Refresh refreshes the history
func (h *History) Refresh() {
	h.loadRuns()
}

// loadRuns loads runs from the database
func (h *History) loadRuns() {
	database, err := db.Open(h.dbPath)
	if err != nil {
		return
	}
	defer database.Close()

	// Build filter
	filter := db.RunFilter{}

	if h.pluginFilter.Selected != "All" {
		filter.Plugin = h.pluginFilter.Selected
	}

	if limit, err := strconv.Atoi(h.limitFilter.Selected); err == nil {
		filter.Limit = limit
	}

	// Load runs
	runs, err := database.ListRuns(filter)
	if err != nil {
		return
	}

	h.runs = runs
	h.table.Refresh()
}

// viewRunDetails shows detailed view of a run
func (h *History) viewRunDetails(run *db.Run) {
	// Load results
	database, err := db.Open(h.dbPath)
	if err != nil {
		return
	}
	defer database.Close()

	results, err := database.GetResults(run.ID)
	if err != nil {
		return
	}

	// Create detail view
	content := container.NewVBox(
		widget.NewLabelWithStyle(fmt.Sprintf("Run #%d Details", run.ID), fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		widget.NewLabel(fmt.Sprintf("Plugin: %s", run.Plugin)),
		widget.NewLabel(fmt.Sprintf("Start Time: %s", run.StartTime.Format("2006-01-02 15:04:05"))),
	)

	if run.EndTime != nil {
		content.Add(widget.NewLabel(fmt.Sprintf("End Time: %s", run.EndTime.Format("2006-01-02 15:04:05"))))
		content.Add(widget.NewLabel(fmt.Sprintf("Duration: %s", formatDuration(run.Duration()))))
	}

	content.Add(widget.NewLabel(fmt.Sprintf("Success: %v", run.Success)))
	content.Add(widget.NewLabel(fmt.Sprintf("Exit Code: %d", run.ExitCode)))

	if run.Error != "" {
		content.Add(widget.NewSeparator())
		content.Add(widget.NewLabel("Error:"))
		errorEntry := widget.NewMultiLineEntry()
		errorEntry.SetText(run.Error)
		errorEntry.Disable()
		content.Add(errorEntry)
	}

	// Add metrics
	if len(results) > 0 {
		content.Add(widget.NewSeparator())
		content.Add(widget.NewLabel("Metrics:"))

		metricsStr := ""
		for _, result := range results {
			metricsStr += fmt.Sprintf("%s: %.2f %s\n", result.Metric, result.Value, result.Unit)
		}

		metricsEntry := widget.NewMultiLineEntry()
		metricsEntry.SetText(metricsStr)
		metricsEntry.Disable()
		content.Add(metricsEntry)
	}

	// Show in dialog
	dialog := widget.NewCard("Run Details", "", container.NewScroll(content))
	dialog.Resize(fyne.NewSize(600, 500))

	popup := widget.NewModalPopUp(dialog, fyne.CurrentApp().Driver().AllWindows()[0].Canvas())
	popup.Show()
}
