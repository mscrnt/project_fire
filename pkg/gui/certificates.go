package gui

import (
	"fmt"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
	"github.com/mscrnt/project_fire/pkg/cert"
	"github.com/mscrnt/project_fire/pkg/db"
)

// Certificates represents the certificate management view
type Certificates struct {
	content fyne.CanvasObject
	dbPath  string
	window  fyne.Window
	
	// UI elements
	runSelect   *widget.Select
	issueBtn    *widget.Button
	verifyBtn   *widget.Button
	statusLabel *widget.Label
	
	// Data
	runs []*db.Run
}

// NewCertificates creates a new certificates view
func NewCertificates(dbPath string) *Certificates {
	c := &Certificates{
		dbPath: dbPath,
	}
	c.build()
	return c
}

// build creates the certificates UI
func (c *Certificates) build() {
	// Certificate issuing section
	c.runSelect = widget.NewSelect([]string{}, func(value string) {
		c.issueBtn.Enable()
	})
	c.runSelect.PlaceHolder = "Select a test run..."
	
	c.issueBtn = widget.NewButton("Issue Certificate", c.issueCertificate)
	c.issueBtn.Disable()
	c.issueBtn.Importance = widget.HighImportance
	
	issueCard := widget.NewCard("Issue Certificate", "Generate a certificate for test results",
		container.NewVBox(
			widget.NewLabel("Select a successful test run:"),
			c.runSelect,
			c.issueBtn,
		),
	)
	
	// Certificate verification section
	c.verifyBtn = widget.NewButton("Verify Certificate...", c.verifyCertificate)
	
	verifyCard := widget.NewCard("Verify Certificate", "Verify an existing certificate",
		container.NewVBox(
			widget.NewLabel("Check the authenticity of a test certificate:"),
			c.verifyBtn,
		),
	)
	
	// Status section
	c.statusLabel = widget.NewLabel("Ready")
	statusCard := widget.NewCard("Status", "", c.statusLabel)
	
	// CA information
	caPath := c.getCAPath()
	caInfo := fmt.Sprintf("CA Location: %s", caPath)
	if _, err := os.Stat(filepath.Join(caPath, "ca.crt")); os.IsNotExist(err) {
		caInfo += "\n\n⚠️ CA not initialized. Run 'bench cert init' to create CA."
	} else {
		caInfo += "\n\n✓ CA is initialized and ready."
	}
	
	caCard := widget.NewCard("Certificate Authority", "", 
		widget.NewLabel(caInfo),
	)
	
	// Layout
	c.content = container.NewVBox(
		issueCard,
		verifyCard,
		caCard,
		statusCard,
	)
	
	// Load runs
	c.loadRuns()
}

// Content returns the certificates content
func (c *Certificates) Content() fyne.CanvasObject {
	return c.content
}

// SetWindow sets the parent window
func (c *Certificates) SetWindow(w fyne.Window) {
	c.window = w
}

// loadRuns loads successful runs
func (c *Certificates) loadRuns() {
	database, err := db.Open(c.dbPath)
	if err != nil {
		c.statusLabel.SetText("Error: Failed to open database")
		return
	}
	defer database.Close()
	
	// Load successful runs only
	success := true
	runs, err := database.ListRuns(db.RunFilter{
		Success: &success,
		Limit:   50,
	})
	if err != nil {
		c.statusLabel.SetText("Error: Failed to load runs")
		return
	}
	
	c.runs = runs
	
	// Update selector
	options := make([]string, len(runs))
	for i, run := range runs {
		options[i] = fmt.Sprintf("#%d - %s (%s)", 
			run.ID, 
			run.Plugin,
			run.StartTime.Format("2006-01-02 15:04"))
	}
	
	c.runSelect.Options = options
	c.runSelect.Refresh()
}

// issueCertificate issues a certificate for the selected run
func (c *Certificates) issueCertificate() {
	idx := c.runSelect.SelectedIndex()
	if idx < 0 || idx >= len(c.runs) {
		return
	}
	
	run := c.runs[idx]
	
	// Get save location
	saveDialog := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
		if err != nil {
			c.statusLabel.SetText(fmt.Sprintf("Error: %v", err))
			return
		}
		if writer == nil {
			return
		}
		defer writer.Close()
		
		// Issue certificate
		c.statusLabel.SetText("Issuing certificate...")
		
		// Load CA
		caPath := c.getCAPath()
		issuer, err := cert.LoadCA(
			filepath.Join(caPath, "ca.crt"),
			filepath.Join(caPath, "ca.key"),
		)
		if err != nil {
			c.statusLabel.SetText(fmt.Sprintf("Error: Failed to load CA - %v", err))
			return
		}
		
		// Load results
		database, err := db.Open(c.dbPath)
		if err != nil {
			c.statusLabel.SetText("Error: Failed to open database")
			return
		}
		defer database.Close()
		
		results, err := database.GetResults(run.ID)
		if err != nil {
			c.statusLabel.SetText("Error: Failed to load results")
			return
		}
		
		// Issue certificate
		certificate, err := issuer.IssueCertificate(run, results)
		if err != nil {
			c.statusLabel.SetText(fmt.Sprintf("Error: Failed to issue certificate - %v", err))
			return
		}
		
		// Write certificate
		pem := certificate.SavePEM()
		if _, err := writer.Write([]byte(pem)); err != nil {
			c.statusLabel.SetText(fmt.Sprintf("Error: Failed to save certificate - %v", err))
			return
		}
		
		c.statusLabel.SetText(fmt.Sprintf("Certificate issued successfully for run #%d", run.ID))
		
	}, c.getWindow())
	
	saveDialog.SetFileName(fmt.Sprintf("fire_cert_%d.pem", run.ID))
	saveDialog.SetFilter(storage.NewExtensionFileFilter([]string{".pem", ".crt"}))
	saveDialog.Show()
}

// verifyCertificate verifies a certificate file
func (c *Certificates) verifyCertificate() {
	openDialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			c.statusLabel.SetText(fmt.Sprintf("Error: %v", err))
			return
		}
		if reader == nil {
			return
		}
		defer reader.Close()
		
		// Create temporary file
		tmpFile, err := os.CreateTemp("", "cert-*.pem")
		if err != nil {
			c.statusLabel.SetText(fmt.Sprintf("Error: %v", err))
			return
		}
		defer os.Remove(tmpFile.Name())
		
		// Copy certificate to temp file
		data := make([]byte, 4096)
		for {
			n, err := reader.Read(data)
			if n > 0 {
				tmpFile.Write(data[:n])
			}
			if err != nil {
				break
			}
		}
		tmpFile.Close()
		
		// Verify certificate
		caPath := c.getCAPath()
		result, err := cert.VerifyCertificateFile(
			tmpFile.Name(),
			filepath.Join(caPath, "ca.crt"),
		)
		if err != nil {
			c.statusLabel.SetText(fmt.Sprintf("Error: %v", err))
			return
		}
		
		// Show results
		resultText := cert.FormatVerifyResult(result)
		
		resultDialog := dialog.NewCustom("Certificate Verification", "Close",
			container.NewScroll(widget.NewLabel(resultText)),
			c.getWindow(),
		)
		resultDialog.Resize(fyne.NewSize(500, 400))
		resultDialog.Show()
		
		if result.Valid {
			c.statusLabel.SetText("Certificate is valid ✓")
		} else {
			c.statusLabel.SetText("Certificate is invalid ✗")
		}
		
	}, c.getWindow())
	
	openDialog.SetFilter(storage.NewExtensionFileFilter([]string{".pem", ".crt", ".cer"}))
	openDialog.Show()
}

// getCAPath returns the CA directory path
func (c *Certificates) getCAPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, ".fire", "ca")
}

// getWindow returns the parent window
func (c *Certificates) getWindow() fyne.Window {
	if c.window != nil {
		return c.window
	}
	// Fallback to current app window
	if app := fyne.CurrentApp(); app != nil {
		if windows := app.Driver().AllWindows(); len(windows) > 0 {
			return windows[0]
		}
	}
	return nil
}