#!/bin/bash

# Script to create a new FIRE plugin

set -e

# Check if plugin name is provided
if [ $# -eq 0 ]; then
    echo "Usage: $0 <plugin_name> [category]"
    echo "Categories: cpu, memory, disk, gpu, network"
    exit 1
fi

PLUGIN_NAME=$1
PLUGIN_CATEGORY=${2:-"cpu"}
PLUGIN_NAME_LOWER=$(echo "$PLUGIN_NAME" | tr '[:upper:]' '[:lower:]')
PLUGIN_NAME_UPPER=$(echo "${PLUGIN_NAME:0:1}" | tr '[:lower:]' '[:upper:]')${PLUGIN_NAME:1}

# Create plugin directory
PLUGIN_DIR="pkg/plugin/$PLUGIN_NAME_LOWER"
mkdir -p "$PLUGIN_DIR"

# Create main plugin file
cat > "$PLUGIN_DIR/${PLUGIN_NAME_LOWER}.go" << EOF
package $PLUGIN_NAME_LOWER

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mscrnt/project_fire/pkg/plugin"
)

func init() {
	plugin.Register("$PLUGIN_NAME_LOWER", &${PLUGIN_NAME_UPPER}Plugin{})
}

// ${PLUGIN_NAME_UPPER}Plugin implements the $PLUGIN_NAME_LOWER test plugin
type ${PLUGIN_NAME_UPPER}Plugin struct{}

// ${PLUGIN_NAME_UPPER}Params defines the parameters for the $PLUGIN_NAME_LOWER test
type ${PLUGIN_NAME_UPPER}Params struct {
	Duration time.Duration \`json:"duration"\`
	// Add more parameters as needed
}

// ${PLUGIN_NAME_UPPER}Result defines the result structure for the $PLUGIN_NAME_LOWER test
type ${PLUGIN_NAME_UPPER}Result struct {
	Score     float64 \`json:"score"\`
	StartTime time.Time \`json:"start_time"\`
	EndTime   time.Time \`json:"end_time"\`
	// Add more result fields as needed
}

// Info returns metadata about the plugin
func (p *${PLUGIN_NAME_UPPER}Plugin) Info() plugin.TestInfo {
	return plugin.TestInfo{
		Name:          "$PLUGIN_NAME_LOWER",
		Description:   "TODO: Add description for $PLUGIN_NAME test",
		Category:      "$PLUGIN_CATEGORY",
		RequiresAdmin: false,
	}
}

// DefaultParams returns default parameters for the test
func (p *${PLUGIN_NAME_UPPER}Plugin) DefaultParams() interface{} {
	return ${PLUGIN_NAME_UPPER}Params{
		Duration: 30 * time.Second,
	}
}

// ValidateParams validates the test parameters
func (p *${PLUGIN_NAME_UPPER}Plugin) ValidateParams(params interface{}) error {
	cp, ok := params.(${PLUGIN_NAME_UPPER}Params)
	if !ok {
		return fmt.Errorf("invalid parameter type")
	}
	
	if cp.Duration < time.Second {
		return fmt.Errorf("duration must be at least 1 second")
	}
	
	return nil
}

// Run executes the $PLUGIN_NAME_LOWER test
func (p *${PLUGIN_NAME_UPPER}Plugin) Run(ctx context.Context, params interface{}, progress plugin.ProgressCallback) (interface{}, error) {
	cp := params.(${PLUGIN_NAME_UPPER}Params)
	
	// Report initial progress
	if progress != nil {
		progress(0, "Starting $PLUGIN_NAME_LOWER test...")
	}
	
	startTime := time.Now()
	
	// TODO: Implement your test logic here
	// Example progress reporting:
	steps := int(cp.Duration.Seconds())
	for i := 0; i < steps; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Second):
			if progress != nil {
				percent := float64(i+1) / float64(steps) * 100
				progress(int(percent), fmt.Sprintf("Running $PLUGIN_NAME_LOWER test... %d/%d", i+1, steps))
			}
		}
	}
	
	endTime := time.Now()
	
	result := ${PLUGIN_NAME_UPPER}Result{
		Score:     100.0, // TODO: Calculate actual score
		StartTime: startTime,
		EndTime:   endTime,
	}
	
	// Report completion
	if progress != nil {
		progress(100, "$PLUGIN_NAME_LOWER test completed")
	}
	
	return result, nil
}

// ParseResults parses raw JSON results
func (p *${PLUGIN_NAME_UPPER}Plugin) ParseResults(data []byte) (interface{}, error) {
	var result ${PLUGIN_NAME_UPPER}Result
	err := json.Unmarshal(data, &result)
	return result, err
}

// FormatResults formats results for display
func (p *${PLUGIN_NAME_UPPER}Plugin) FormatResults(results interface{}) string {
	r, ok := results.(${PLUGIN_NAME_UPPER}Result)
	if !ok {
		return "Invalid result format"
	}
	
	duration := r.EndTime.Sub(r.StartTime)
	return fmt.Sprintf("Score: %.2f (Duration: %s)", r.Score, duration.Round(time.Second))
}
EOF

# Create test file
cat > "$PLUGIN_DIR/${PLUGIN_NAME_LOWER}_test.go" << EOF
package $PLUGIN_NAME_LOWER

import (
	"context"
	"testing"
	"time"
)

func Test${PLUGIN_NAME_UPPER}Plugin_Info(t *testing.T) {
	p := &${PLUGIN_NAME_UPPER}Plugin{}
	info := p.Info()
	
	if info.Name != "$PLUGIN_NAME_LOWER" {
		t.Errorf("expected name '$PLUGIN_NAME_LOWER', got %s", info.Name)
	}
	
	if info.Category != "$PLUGIN_CATEGORY" {
		t.Errorf("expected category '$PLUGIN_CATEGORY', got %s", info.Category)
	}
}

func Test${PLUGIN_NAME_UPPER}Plugin_ValidateParams(t *testing.T) {
	p := &${PLUGIN_NAME_UPPER}Plugin{}
	
	tests := []struct {
		name    string
		params  interface{}
		wantErr bool
	}{
		{
			name: "valid params",
			params: ${PLUGIN_NAME_UPPER}Params{
				Duration: 10 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "duration too short",
			params: ${PLUGIN_NAME_UPPER}Params{
				Duration: 500 * time.Millisecond,
			},
			wantErr: true,
		},
		{
			name:    "wrong type",
			params:  "invalid",
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := p.ValidateParams(tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateParams() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test${PLUGIN_NAME_UPPER}Plugin_Run(t *testing.T) {
	p := &${PLUGIN_NAME_UPPER}Plugin{}
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	params := ${PLUGIN_NAME_UPPER}Params{
		Duration: 2 * time.Second,
	}
	
	progressCalled := false
	progress := func(percent int, message string) {
		progressCalled = true
		t.Logf("Progress: %d%% - %s", percent, message)
	}
	
	result, err := p.Run(ctx, params, progress)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	
	if !progressCalled {
		t.Error("progress callback was not called")
	}
	
	r, ok := result.(${PLUGIN_NAME_UPPER}Result)
	if !ok {
		t.Fatalf("result is not of type ${PLUGIN_NAME_UPPER}Result")
	}
	
	if r.Score == 0 {
		t.Error("score should not be zero")
	}
}
EOF

echo "✅ Created new plugin: $PLUGIN_NAME_LOWER"
echo "📁 Location: $PLUGIN_DIR"
echo ""
echo "Next steps:"
echo "1. Edit $PLUGIN_DIR/${PLUGIN_NAME_LOWER}.go to implement your test logic"
echo "2. Update the plugin description and parameters"
echo "3. Run tests: go test -v ./$PLUGIN_DIR/..."
echo "4. Register in documentation"