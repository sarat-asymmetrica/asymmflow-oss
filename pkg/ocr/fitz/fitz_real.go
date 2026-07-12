// Real go-fitz implementation using MuPDF bindings
// Installed via: go get github.com/gen2brain/go-fitz
package fitz

import (
	"fmt"
	"image"
	"time"

	gofitz "github.com/gen2brain/go-fitz"
)

// RealDocument wraps go-fitz document
type RealDocument struct {
	doc      *gofitz.Document
	path     string
	numPages int
}

// OpenDocument opens a document using go-fitz
func OpenDocument(path string) (*RealDocument, error) {
	doc, err := gofitz.New(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open document: %w", err)
	}

	return &RealDocument{
		doc:      doc,
		path:     path,
		numPages: doc.NumPage(),
	}, nil
}

// Close closes the document
func (d *RealDocument) Close() error {
	return d.doc.Close()
}

// NumPages returns the number of pages
func (d *RealDocument) NumPages() int {
	return d.numPages
}

// ExtractText extracts text from a specific page
func (d *RealDocument) ExtractText(pageNum int) (string, error) {
	if pageNum < 0 || pageNum >= d.numPages {
		return "", fmt.Errorf("page %d out of range (0-%d)", pageNum, d.numPages-1)
	}
	return d.doc.Text(pageNum)
}

// ExtractAllText extracts text from all pages
func (d *RealDocument) ExtractAllText() (string, error) {
	var allText string
	for i := 0; i < d.numPages; i++ {
		text, err := d.doc.Text(i)
		if err != nil {
			return allText, fmt.Errorf("failed to extract page %d: %w", i, err)
		}
		allText += text + "\n"
	}
	return allText, nil
}

// ExtractImage extracts a page as an image (for OCR)
func (d *RealDocument) ExtractImage(pageNum int) (image.Image, error) {
	if pageNum < 0 || pageNum >= d.numPages {
		return nil, fmt.Errorf("page %d out of range (0-%d)", pageNum, d.numPages-1)
	}
	return d.doc.Image(pageNum)
}

// ExtractHTML extracts page as HTML
func (d *RealDocument) ExtractHTML(pageNum int) (string, error) {
	if pageNum < 0 || pageNum >= d.numPages {
		return "", fmt.Errorf("page %d out of range (0-%d)", pageNum, d.numPages-1)
	}
	return d.doc.HTML(pageNum, true)
}

// ExtractPDFReal extracts text from PDF using real go-fitz
func ExtractPDFReal(filepath string) (*ExtractionResult, error) {
	start := time.Now()

	doc, err := OpenDocument(filepath)
	if err != nil {
		return &ExtractionResult{
			Success: false,
			Error:   err,
			Method:  "gofitz_error",
		}, err
	}
	defer doc.Close()

	// Extract all text
	text, err := doc.ExtractAllText()
	if err != nil {
		return &ExtractionResult{
			Success: false,
			Error:   err,
			Method:  "gofitz_error",
		}, err
	}

	// Determine if vector or scanned
	isVector := len(text) > 50
	method := "vector_pdf"
	needsOCR := false

	if !isVector {
		method = "scanned_pdf"
		needsOCR = true
	}

	result := &ExtractionResult{
		Success:         true,
		Text:            text,
		Method:          method,
		Pages:           doc.NumPages(),
		Characters:      len(text),
		Duration:        time.Since(start),
		NeedsOCR:        needsOCR,
		DigitalRoot:     DigitalRoot(len(text)),
		ComplexityClass: MirzakhaniComplexity(doc.NumPages(), len(text)/max(doc.NumPages(), 1)),
	}

	// If scanned, extract images for OCR
	if needsOCR {
		images := make([]image.Image, 0, doc.NumPages())
		for i := 0; i < doc.NumPages(); i++ {
			img, err := doc.ExtractImage(i)
			if err == nil {
				images = append(images, img)
			}
		}
		result.Images = images
	}

	return result, nil
}
