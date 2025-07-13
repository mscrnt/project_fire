// Package report provides report generation functionality for test results.
package report

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// PDFOptions contains options for PDF generation
type PDFOptions struct {
	Landscape           bool
	PrintBackground     bool
	PreferCSSPageSize   bool
	PaperWidth          float64
	PaperHeight         float64
	MarginTop           float64
	MarginBottom        float64
	MarginLeft          float64
	MarginRight         float64
	HeaderTemplate      string
	FooterTemplate      string
	DisplayHeaderFooter bool
}

// DefaultPDFOptions returns default PDF options
func DefaultPDFOptions() PDFOptions {
	return PDFOptions{
		Landscape:           false,
		PrintBackground:     true,
		PreferCSSPageSize:   false,
		PaperWidth:          8.5,  // Letter width in inches
		PaperHeight:         11.0, // Letter height in inches
		MarginTop:           0.4,
		MarginBottom:        0.4,
		MarginLeft:          0.4,
		MarginRight:         0.4,
		DisplayHeaderFooter: false,
	}
}

// GeneratePDF generates a PDF report for a run
func (g *Generator) GeneratePDF(runID int64, outputPath string, options *PDFOptions) error {
	// Generate HTML first
	html, err := g.GenerateHTML(runID)
	if err != nil {
		return fmt.Errorf("failed to generate HTML: %w", err)
	}

	// Create temporary HTML file
	tmpFile, err := os.CreateTemp("", "fire-report-*.html")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	// Write HTML to temp file
	if _, err := tmpFile.WriteString(html); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("failed to write HTML: %w", err)
	}
	_ = tmpFile.Close()

	// Convert to PDF
	return htmlToPDF(tmpFile.Name(), outputPath, options)
}

// htmlToPDF converts an HTML file to PDF using chromedp
func htmlToPDF(htmlPath, pdfPath string, options *PDFOptions) error {
	// Create context
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	// Set timeout
	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Read HTML file
	htmlContent, err := os.ReadFile(htmlPath) // #nosec G304 -- htmlPath is a generated report file path
	if err != nil {
		return fmt.Errorf("failed to read HTML file: %w", err)
	}

	// Navigate to data URL with HTML content
	dataURL := "data:text/html;charset=utf-8," + string(htmlContent)

	// Generate PDF
	var pdfData []byte
	if err := chromedp.Run(ctx,
		chromedp.Navigate(dataURL),
		chromedp.WaitReady("body"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			params := page.PrintToPDF()

			// Apply options
			params = params.
				WithLandscape(options.Landscape).
				WithPrintBackground(options.PrintBackground).
				WithPreferCSSPageSize(options.PreferCSSPageSize).
				WithPaperWidth(options.PaperWidth).
				WithPaperHeight(options.PaperHeight).
				WithMarginTop(options.MarginTop).
				WithMarginBottom(options.MarginBottom).
				WithMarginLeft(options.MarginLeft).
				WithMarginRight(options.MarginRight).
				WithDisplayHeaderFooter(options.DisplayHeaderFooter)

			if options.HeaderTemplate != "" {
				params = params.WithHeaderTemplate(options.HeaderTemplate)
			}
			if options.FooterTemplate != "" {
				params = params.WithFooterTemplate(options.FooterTemplate)
			}

			var err error
			pdfData, _, err = params.Do(ctx)
			return err
		}),
	); err != nil {
		return fmt.Errorf("failed to generate PDF: %w", err)
	}

	// Write PDF to file
	if err := os.WriteFile(pdfPath, pdfData, 0o600); err != nil {
		return fmt.Errorf("failed to write PDF: %w", err)
	}

	return nil
}

// QuickPDF generates a PDF with default options
func (g *Generator) QuickPDF(runID int64, outputPath string) error {
	options := DefaultPDFOptions()
	return g.GeneratePDF(runID, outputPath, &options)
}
