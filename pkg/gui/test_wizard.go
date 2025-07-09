package gui

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/mscrnt/project_fire/pkg/db"
	"github.com/mscrnt/project_fire/pkg/plugin"
)

// TestWizard represents the test configuration wizard
type TestWizard struct {
	content fyne.CanvasObject
	dbPath  string

	// Wizard state
	currentStep int

	// Step 1: Plugin selection
	pluginSelect   *widget.Select
	selectedPlugin string

	// Step 2: Parameters
	paramForm *widget.Form
	params    map[string]interface{}

	// Step 3: Review and run
	summaryLabel *widget.Label
	runButton    *widget.Button
	logEntry     *widget.Entry

	// Navigation
	backButton *widget.Button
	nextButton *widget.Button

	// Running test
	cancelFunc context.CancelFunc
	running    bool
}

// NewTestWizard creates a new test wizard
func NewTestWizard(dbPath string) *TestWizard {
	w := &TestWizard{
		dbPath: dbPath,
		params: make(map[string]interface{}),
	}
	w.build()
	return w
}

// build creates the wizard UI
func (w *TestWizard) build() {
	// Create steps
	step1 := w.createStep1()
	step2 := w.createStep2()
	step3 := w.createStep3()

	// Stack for steps
	steps := container.NewMax(step1, step2, step3)

	// Navigation buttons
	w.backButton = widget.NewButton("Back", w.previousStep)
	w.backButton.Disable()

	w.nextButton = widget.NewButton("Next", w.nextStep)

	navigation := container.NewHBox(
		layout.NewSpacer(),
		w.backButton,
		w.nextButton,
	)

	// Main content
	w.content = container.NewBorder(
		nil, navigation, nil, nil,
		steps,
	)

	// Show first step
	w.showStep(0)
}

// Content returns the wizard content
func (w *TestWizard) Content() fyne.CanvasObject {
	return w.content
}

// createStep1 creates the plugin selection step
func (w *TestWizard) createStep1() fyne.CanvasObject {
	// Get available plugins
	plugins := plugin.List()
	pluginNames := make([]string, len(plugins))
	for i, p := range plugins {
		pluginNames[i] = p
	}

	// Plugin selector
	w.pluginSelect = widget.NewSelect(pluginNames, func(selected string) {
		w.selectedPlugin = selected
		w.nextButton.Enable()
	})
	w.pluginSelect.PlaceHolder = "Select a test plugin..."

	// Plugin descriptions
	descriptions := container.NewVBox(
		widget.NewCard("CPU Stress Test", "", widget.NewLabel(
			"Stress test CPU with configurable thread count and operations")),
		widget.NewCard("Memory Test", "", widget.NewLabel(
			"Test memory allocation and access patterns")),
	)

	return container.NewBorder(
		widget.NewLabelWithStyle("Step 1: Select Test Plugin", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		nil, nil, nil,
		container.NewVBox(
			widget.NewLabel("Choose the type of test to run:"),
			w.pluginSelect,
			widget.NewSeparator(),
			descriptions,
		),
	)
}

// createStep2 creates the parameter configuration step
func (w *TestWizard) createStep2() fyne.CanvasObject {
	w.paramForm = widget.NewForm()

	return container.NewBorder(
		widget.NewLabelWithStyle("Step 2: Configure Parameters", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		nil, nil, nil,
		container.NewScroll(w.paramForm),
	)
}

// createStep3 creates the review and run step
func (w *TestWizard) createStep3() fyne.CanvasObject {
	w.summaryLabel = widget.NewLabel("Test Summary")
	w.summaryLabel.Wrapping = fyne.TextWrapWord

	w.runButton = widget.NewButton("Run Test", w.runTest)
	w.runButton.Importance = widget.HighImportance

	w.logEntry = widget.NewMultiLineEntry()
	w.logEntry.Disable()

	logScroll := container.NewScroll(w.logEntry)
	logScroll.SetMinSize(fyne.NewSize(600, 300))

	return container.NewBorder(
		widget.NewLabelWithStyle("Step 3: Review and Run", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		nil, nil, nil,
		container.NewVBox(
			w.summaryLabel,
			widget.NewSeparator(),
			w.runButton,
			widget.NewLabel("Test Output:"),
			logScroll,
		),
	)
}

// Navigation methods
func (w *TestWizard) showStep(step int) {
	w.currentStep = step

	// Update navigation buttons
	if step == 0 {
		w.backButton.Disable()
		w.nextButton.Enable()
		w.nextButton.SetText("Next")
	} else if step == 1 {
		w.backButton.Enable()
		w.nextButton.Enable()
		w.nextButton.SetText("Next")
		w.updateParameterForm()
	} else if step == 2 {
		w.backButton.Enable()
		w.nextButton.Disable()
		w.updateSummary()
	}
}

// previousStep goes to the previous step
func (w *TestWizard) previousStep() {
	if w.currentStep > 0 {
		w.showStep(w.currentStep - 1)
	}
}

// nextStep goes to the next step
func (w *TestWizard) nextStep() {
	if w.currentStep < 2 {
		// Save current step data
		if w.currentStep == 1 {
			w.saveParameters()
		}
		w.showStep(w.currentStep + 1)
	}
}

// updateParameterForm updates the parameter form for the selected plugin
func (w *TestWizard) updateParameterForm() {
	w.paramForm.Items = nil

	// Get plugin to get default parameters
	p, err := plugin.Get(w.selectedPlugin)
	if err != nil {
		return
	}

	defaultParams := p.DefaultParams()

	// Add duration field
	durationEntry := widget.NewEntry()
	durationEntry.SetText(fmt.Sprintf("%.0f", defaultParams.Duration.Seconds()))
	w.paramForm.Append("Duration (seconds)", durationEntry)

	// Add plugin-specific fields
	switch w.selectedPlugin {
	case "cpu":
		threadsEntry := widget.NewEntry()
		if threads, ok := defaultParams.Config["threads"].(int); ok {
			threadsEntry.SetText(strconv.Itoa(threads))
		}
		w.paramForm.Append("Threads", threadsEntry)

	case "memory":
		sizeEntry := widget.NewEntry()
		if size, ok := defaultParams.Config["size_mb"].(int); ok {
			sizeEntry.SetText(strconv.Itoa(size))
		}
		w.paramForm.Append("Size (MB)", sizeEntry)
	}

	w.paramForm.Refresh()
}

// saveParameters saves the form parameters
func (w *TestWizard) saveParameters() {
	w.params = make(map[string]interface{})

	// Extract values from form
	for _, item := range w.paramForm.Items {
		if entry, ok := item.Widget.(*widget.Entry); ok {
			value := entry.Text
			label := item.Text

			switch label {
			case "Duration (seconds)":
				if duration, err := strconv.ParseFloat(value, 64); err == nil {
					w.params["duration"] = duration
				}
			case "Threads":
				if threads, err := strconv.Atoi(value); err == nil {
					w.params["threads"] = threads
				}
			case "Size (MB)":
				if size, err := strconv.Atoi(value); err == nil {
					w.params["size_mb"] = size
				}
			}
		}
	}
}

// updateSummary updates the test summary
func (w *TestWizard) updateSummary() {
	summary := fmt.Sprintf("Test Configuration:\n\n")
	summary += fmt.Sprintf("Plugin: %s\n", w.selectedPlugin)
	summary += fmt.Sprintf("Duration: %.0f seconds\n", w.params["duration"])

	switch w.selectedPlugin {
	case "cpu":
		if threads, ok := w.params["threads"].(int); ok {
			summary += fmt.Sprintf("Threads: %d\n", threads)
		}
	case "memory":
		if size, ok := w.params["size_mb"].(int); ok {
			summary += fmt.Sprintf("Size: %d MB\n", size)
		}
	}

	w.summaryLabel.SetText(summary)
}

// runTest runs the configured test
func (w *TestWizard) runTest() {
	if w.running {
		// Cancel running test
		if w.cancelFunc != nil {
			w.cancelFunc()
		}
		return
	}

	w.running = true
	w.runButton.SetText("Cancel")
	w.logEntry.SetText("Starting test...\n")
	w.backButton.Disable()

	// Create context with cancel
	ctx, cancel := context.WithCancel(context.Background())
	w.cancelFunc = cancel

	// Run test in goroutine
	go func() {
		defer func() {
			w.running = false
			w.runButton.SetText("Run Test")
			w.backButton.Enable()
			w.cancelFunc = nil
		}()

		// Get plugin
		p, err := plugin.Get(w.selectedPlugin)
		if err != nil {
			w.appendLog(fmt.Sprintf("Error: %v\n", err))
			return
		}

		// Prepare parameters
		params := p.DefaultParams()
		if duration, ok := w.params["duration"].(float64); ok {
			params.Duration = time.Duration(duration) * time.Second
		}

		// Apply plugin-specific parameters
		for k, v := range w.params {
			if k != "duration" {
				params.Config[k] = v
			}
		}

		// Open database
		database, err := db.Open(w.dbPath)
		if err != nil {
			w.appendLog(fmt.Sprintf("Database error: %v\n", err))
			return
		}
		defer database.Close()

		// Create run record
		run, err := database.CreateRun(w.selectedPlugin, params.Config)
		if err != nil {
			w.appendLog(fmt.Sprintf("Failed to create run: %v\n", err))
			return
		}

		w.appendLog(fmt.Sprintf("Created run ID: %d\n", run.ID))

		// Run the test
		result, err := p.Run(ctx, params)
		if err != nil {
			w.appendLog(fmt.Sprintf("Test error: %v\n", err))
			run.Success = false
			run.Error = err.Error()
		} else {
			run.Success = result.Success
			run.Stdout = result.Stdout
			run.Stderr = result.Stderr

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

				if err := database.CreateResults(run.ID, result.Metrics, units); err != nil {
					w.appendLog(fmt.Sprintf("Failed to save metrics: %v\n", err))
				}
			}
		}

		// Update run record
		endTime := time.Now()
		run.EndTime = &endTime
		if err := database.UpdateRun(run); err != nil {
			w.appendLog(fmt.Sprintf("Failed to update run: %v\n", err))
		}

		// Display results
		w.appendLog("\nTest completed!\n")
		w.appendLog(fmt.Sprintf("Success: %v\n", run.Success))
		w.appendLog(fmt.Sprintf("Duration: %s\n", run.Duration()))

		if result.Stdout != "" {
			w.appendLog("\nOutput:\n" + result.Stdout)
		}

		if len(result.Metrics) > 0 {
			w.appendLog("\nMetrics:\n")
			for name, value := range result.Metrics {
				w.appendLog(fmt.Sprintf("  %s: %.2f\n", name, value))
			}
		}
	}()
}

// appendLog appends text to the log
func (w *TestWizard) appendLog(text string) {
	current := w.logEntry.Text
	w.logEntry.SetText(current + text)
	w.logEntry.CursorRow = len(w.logEntry.Text)
}
