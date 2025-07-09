package report

import (
	"bytes"
	"fmt"
	"html/template"
	"time"

	"github.com/mscrnt/project_fire/pkg/db"
)

// ReportData contains all data needed for report generation
type ReportData struct {
	Run          *db.Run
	Results      []*db.Result
	Plugin       string
	GeneratedAt  time.Time
	SystemInfo   SystemInfo
	MetricGroups []MetricGroup
}

// SystemInfo contains system information
type SystemInfo struct {
	Hostname     string
	OS           string
	Architecture string
	CPUModel     string
	CPUCores     int
	TotalMemory  string
}

// MetricGroup groups related metrics together
type MetricGroup struct {
	Name    string
	Metrics []MetricDisplay
}

// MetricDisplay represents a metric for display
type MetricDisplay struct {
	Name  string
	Value string
	Unit  string
	Raw   float64
}

// Generator creates reports from test data
type Generator struct {
	database *db.DB
}

// NewGenerator creates a new report generator
func NewGenerator(database *db.DB) *Generator {
	return &Generator{
		database: database,
	}
}

// GenerateHTML generates an HTML report for a run
func (g *Generator) GenerateHTML(runID int64) (string, error) {
	// Load data
	data, err := g.loadReportData(runID)
	if err != nil {
		return "", err
	}

	// Load template
	tmpl, err := g.loadHTMLTemplate()
	if err != nil {
		return "", err
	}

	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// loadReportData loads all data needed for a report
func (g *Generator) loadReportData(runID int64) (*ReportData, error) {
	// Get run
	run, err := g.database.GetRun(runID)
	if err != nil {
		return nil, fmt.Errorf("failed to get run: %w", err)
	}

	// Get results
	results, err := g.database.GetResults(runID)
	if err != nil {
		return nil, fmt.Errorf("failed to get results: %w", err)
	}

	// Prepare data
	data := &ReportData{
		Run:         run,
		Results:     results,
		Plugin:      run.Plugin,
		GeneratedAt: time.Now(),
		SystemInfo:  g.getSystemInfo(),
	}

	// Group metrics
	data.MetricGroups = g.groupMetrics(results)

	return data, nil
}

// getSystemInfo collects system information
func (g *Generator) getSystemInfo() SystemInfo {
	// This is a simplified version - in production you'd use gopsutil
	return SystemInfo{
		Hostname:     "localhost",
		OS:           "Linux",
		Architecture: "x86_64",
		CPUModel:     "Unknown",
		CPUCores:     4,
		TotalMemory:  "16 GB",
	}
}

// groupMetrics groups metrics by category
func (g *Generator) groupMetrics(results []*db.Result) []MetricGroup {
	// Simple grouping - in production you'd have more sophisticated logic
	groups := make(map[string][]MetricDisplay)

	for _, result := range results {
		group := "General"

		// Determine group based on metric name
		if contains(result.Metric, []string{"cpu", "operations", "bogo"}) {
			group = "CPU Performance"
		} else if contains(result.Metric, []string{"memory", "alloc", "heap"}) {
			group = "Memory Performance"
		} else if contains(result.Metric, []string{"disk", "io", "throughput"}) {
			group = "Disk Performance"
		}

		display := MetricDisplay{
			Name:  formatMetricName(result.Metric),
			Value: formatValue(result.Value, result.Unit),
			Unit:  result.Unit,
			Raw:   result.Value,
		}

		groups[group] = append(groups[group], display)
	}

	// Convert to slice
	var metricGroups []MetricGroup
	for name, metrics := range groups {
		metricGroups = append(metricGroups, MetricGroup{
			Name:    name,
			Metrics: metrics,
		})
	}

	return metricGroups
}

// loadHTMLTemplate loads the HTML report template
func (g *Generator) loadHTMLTemplate() (*template.Template, error) {
	// Define template functions
	funcMap := template.FuncMap{
		"formatTime": func(t time.Time) string {
			return t.Format("2006-01-02 15:04:05")
		},
		"formatDuration": func(d time.Duration) string {
			return fmt.Sprintf("%.2f seconds", d.Seconds())
		},
		"statusClass": func(success bool) string {
			if success {
				return "success"
			}
			return "failure"
		},
		"statusText": func(success bool) string {
			if success {
				return "PASSED"
			}
			return "FAILED"
		},
	}

	// Parse template
	tmpl := template.New("report").Funcs(funcMap)
	tmpl, err := tmpl.Parse(htmlTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	return tmpl, nil
}

// Helper functions
func contains(s string, substrs []string) bool {
	for _, substr := range substrs {
		if len(s) >= len(substr) && s[:len(substr)] == substr {
			return true
		}
	}
	return false
}

func formatMetricName(name string) string {
	// Convert snake_case to Title Case
	result := ""
	capitalize := true
	for _, ch := range name {
		if ch == '_' {
			result += " "
			capitalize = true
		} else if capitalize {
			result += string(ch - 32) // Convert to uppercase
			capitalize = false
		} else {
			result += string(ch)
		}
	}
	return result
}

func formatValue(value float64, unit string) string {
	if unit == "%" {
		return fmt.Sprintf("%.1f", value)
	} else if unit == "MB/s" || unit == "ops/s" {
		return fmt.Sprintf("%.2f", value)
	} else if value >= 1000000 {
		return fmt.Sprintf("%.2fM", value/1000000)
	} else if value >= 1000 {
		return fmt.Sprintf("%.2fK", value/1000)
	}
	return fmt.Sprintf("%.2f", value)
}

// htmlTemplate is the default HTML report template
const htmlTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>F.I.R.E. Test Report - Run #{{.Run.ID}}</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
            background-color: #f5f5f5;
        }
        .container {
            background-color: white;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            padding: 30px;
        }
        h1, h2, h3 {
            color: #2c3e50;
        }
        .header {
            border-bottom: 3px solid #FF6B35;
            padding-bottom: 20px;
            margin-bottom: 30px;
        }
        .status {
            display: inline-block;
            padding: 5px 15px;
            border-radius: 4px;
            font-weight: bold;
            text-transform: uppercase;
        }
        .status.success {
            background-color: #10B981;
            color: white;
        }
        .status.failure {
            background-color: #EF4444;
            color: white;
        }
        .info-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 20px;
            margin: 20px 0;
        }
        .info-card {
            background-color: #f8f9fa;
            padding: 15px;
            border-radius: 4px;
            border-left: 4px solid #FF6B35;
        }
        .info-card h3 {
            margin: 0 0 10px 0;
            color: #666;
            font-size: 0.9em;
            text-transform: uppercase;
        }
        .info-card p {
            margin: 0;
            font-size: 1.1em;
            font-weight: 500;
        }
        .metrics-section {
            margin: 30px 0;
        }
        .metric-group {
            margin-bottom: 25px;
        }
        .metric-group h3 {
            background-color: #f0f0f0;
            padding: 10px;
            margin: 0 0 15px 0;
            border-radius: 4px;
        }
        .metrics-table {
            width: 100%;
            border-collapse: collapse;
        }
        .metrics-table th,
        .metrics-table td {
            padding: 10px;
            text-align: left;
            border-bottom: 1px solid #e0e0e0;
        }
        .metrics-table th {
            background-color: #f8f9fa;
            font-weight: 600;
            color: #666;
        }
        .metrics-table tr:last-child td {
            border-bottom: none;
        }
        .footer {
            margin-top: 40px;
            padding-top: 20px;
            border-top: 1px solid #e0e0e0;
            text-align: center;
            color: #666;
            font-size: 0.9em;
        }
        .error-section {
            background-color: #FEE;
            border: 1px solid #FCC;
            border-radius: 4px;
            padding: 15px;
            margin: 20px 0;
        }
        .error-section h3 {
            color: #C00;
            margin-top: 0;
        }
        pre {
            background-color: #f4f4f4;
            padding: 10px;
            border-radius: 4px;
            overflow-x: auto;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>F.I.R.E. Test Report</h1>
            <p>Run ID: #{{.Run.ID}} | Plugin: {{.Plugin}} | 
               Status: <span class="status {{statusClass .Run.Success}}">{{statusText .Run.Success}}</span>
            </p>
        </div>

        <div class="info-grid">
            <div class="info-card">
                <h3>Start Time</h3>
                <p>{{formatTime .Run.StartTime}}</p>
            </div>
            <div class="info-card">
                <h3>End Time</h3>
                <p>{{if .Run.EndTime}}{{formatTime .Run.EndTime}}{{else}}Still Running{{end}}</p>
            </div>
            <div class="info-card">
                <h3>Duration</h3>
                <p>{{if .Run.EndTime}}{{formatDuration .Run.Duration}}{{else}}N/A{{end}}</p>
            </div>
            <div class="info-card">
                <h3>Exit Code</h3>
                <p>{{.Run.ExitCode}}</p>
            </div>
        </div>

        {{if .Run.Error}}
        <div class="error-section">
            <h3>Error Details</h3>
            <pre>{{.Run.Error}}</pre>
        </div>
        {{end}}

        {{if .Run.Params}}
        <div class="metrics-section">
            <h2>Test Parameters</h2>
            <table class="metrics-table">
                <thead>
                    <tr>
                        <th>Parameter</th>
                        <th>Value</th>
                    </tr>
                </thead>
                <tbody>
                    {{range $key, $value := .Run.Params}}
                    <tr>
                        <td>{{$key}}</td>
                        <td>{{$value}}</td>
                    </tr>
                    {{end}}
                </tbody>
            </table>
        </div>
        {{end}}

        <div class="metrics-section">
            <h2>Test Results</h2>
            {{range .MetricGroups}}
            <div class="metric-group">
                <h3>{{.Name}}</h3>
                <table class="metrics-table">
                    <thead>
                        <tr>
                            <th>Metric</th>
                            <th>Value</th>
                            <th>Unit</th>
                        </tr>
                    </thead>
                    <tbody>
                        {{range .Metrics}}
                        <tr>
                            <td>{{.Name}}</td>
                            <td>{{.Value}}</td>
                            <td>{{.Unit}}</td>
                        </tr>
                        {{end}}
                    </tbody>
                </table>
            </div>
            {{end}}
        </div>

        <div class="footer">
            <p>Generated by F.I.R.E. on {{formatTime .GeneratedAt}}</p>
            <p>Full Intensity Rigorous Evaluation</p>
        </div>
    </div>
</body>
</html>
`
