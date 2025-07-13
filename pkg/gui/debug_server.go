package gui

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"time"
)

// DebugServer provides a backdoor HTTP server for debugging
type DebugServer struct {
	gui       *FireGUI
	port      int
	callbacks map[string]func()
}

// NewDebugServer creates a new debug server
func NewDebugServer(port int) *DebugServer {
	return &DebugServer{
		port:      port,
		callbacks: make(map[string]func()),
	}
}

// SetGUI sets the GUI instance
func (ds *DebugServer) SetGUI(gui *FireGUI) {
	ds.gui = gui
}

// RegisterCallback registers a callback function
func (ds *DebugServer) RegisterCallback(name string, fn func()) {
	ds.callbacks[name] = fn
}

// Start starts the debug server
func (ds *DebugServer) Start() {
	ds.run()
}

// StartDebugServer starts a debug HTTP server on the specified port
func StartDebugServer(gui *FireGUI, port int) {
	ds := &DebugServer{
		gui:  gui,
		port: port,
	}

	go ds.run()
	fmt.Printf("DEBUG: Debug server started on http://localhost:%d\n", port)
}

func (ds *DebugServer) run() {
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	// Memory stats endpoint
	mux.HandleFunc("/debug/memory", func(w http.ResponseWriter, r *http.Request) {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"alloc_mb":       m.Alloc / 1024 / 1024,
			"total_alloc_mb": m.TotalAlloc / 1024 / 1024,
			"sys_mb":         m.Sys / 1024 / 1024,
			"num_gc":         m.NumGC,
			"goroutines":     runtime.NumGoroutine(),
		})
	})

	// Goroutines endpoint
	mux.HandleFunc("/debug/goroutines", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		buf := make([]byte, 1<<20) // 1MB buffer
		n := runtime.Stack(buf, true)
		w.Write(buf[:n])
	})

	// GUI state endpoint
	mux.HandleFunc("/debug/gui", func(w http.ResponseWriter, r *http.Request) {
		state := map[string]interface{}{
			"window_visible": false,
			"dashboard":      ds.gui.dashboard != nil,
		}

		if ds.gui.window != nil {
			state["window_visible"] = ds.gui.window.Canvas() != nil
			state["window_title"] = ds.gui.window.Title()
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(state)
	})

	// Dashboard state endpoint
	mux.HandleFunc("/debug/dashboard", func(w http.ResponseWriter, r *http.Request) {
		if ds.gui.dashboard == nil {
			http.Error(w, "Dashboard not initialized", 404)
			return
		}

		state := map[string]interface{}{
			"content_exists":   ds.gui.dashboard.content != nil,
			"running":          ds.gui.dashboard.running,
			"cpu_summary":      ds.gui.dashboard.cpuSummary != nil,
			"memory_summary":   ds.gui.dashboard.memorySummary != nil,
			"gpu_summary":      ds.gui.dashboard.gpuSummary != nil,
			"component_list":   ds.gui.dashboard.componentList != nil,
			"selected_index":   ds.gui.dashboard.selectedIndex,
			"components_count": len(ds.gui.dashboard.components),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(state)
	})

	// Force update endpoint
	mux.HandleFunc("/debug/update", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", 405)
			return
		}

		if ds.gui.dashboard != nil {
			go ds.gui.dashboard.updateMetrics()
			w.Write([]byte("Update triggered\n"))
		} else {
			http.Error(w, "Dashboard not initialized", 500)
		}
	})

	addr := fmt.Sprintf("localhost:%d", ds.port)
	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		fmt.Printf("DEBUG: Debug server error: %v\n", err)
	}
}
