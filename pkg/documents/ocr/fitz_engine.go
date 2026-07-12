package ocr

import (
	"bytes"
	"fmt"
	"image/png"
	"log"
	"path/filepath"
	"strings"

	"github.com/gen2brain/go-fitz"
)

type FitzEngine struct{}

func NewFitzEngine() FitzEngine {
	return FitzEngine{}
}

func (FitzEngine) SupportedFormats() []string {
	return []string{".pdf"}
}

func (FitzEngine) ExtractText(filePath string) (string, error) {
	doc, err := fitz.New(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open PDF: %w", err)
	}
	defer doc.Close()

	var fullText strings.Builder
	numPages := doc.NumPage()

	log.Printf("Extracting vector PDF: %s (%d pages)", filepath.Base(filePath), numPages)

	for i := 0; i < numPages; i++ {
		text, err := doc.Text(i)
		if err != nil {
			log.Printf("Page %d extraction failed: %v (continuing)", i+1, err)
			continue
		}
		trimmedText := strings.TrimSpace(text)
		if len(trimmedText) > 0 {
			if i > 0 {
				fullText.WriteString("\n\n--- PAGE " + fmt.Sprintf("%d", i+1) + " ---\n\n")
			}
			fullText.WriteString(trimmedText)
		}
	}

	result := strings.TrimSpace(fullText.String())
	log.Printf("Vector PDF extraction complete: %d chars from %d pages", len(result), numPages)
	return result, nil
}

func (e FitzEngine) ExtractPages(filePath string) ([]PageContent, error) {
	doc, err := fitz.New(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open PDF: %w", err)
	}
	defer doc.Close()

	pages := make([]PageContent, 0, doc.NumPage())
	for i := 0; i < doc.NumPage(); i++ {
		text, err := doc.Text(i)
		if err != nil {
			continue
		}
		pages = append(pages, PageContent{
			PageNumber: i + 1,
			Text:       strings.TrimSpace(text),
		})
	}
	return pages, nil
}

func (FitzEngine) IsVectorPDF(filePath string) bool {
	if !strings.HasSuffix(strings.ToLower(filePath), ".pdf") {
		return false
	}

	doc, err := fitz.New(filePath)
	if err != nil {
		return false
	}
	defer doc.Close()

	numPages := doc.NumPage()
	if numPages == 0 {
		return false
	}

	maxPagesToCheck := numPages
	if maxPagesToCheck > 20 {
		maxPagesToCheck = 20
	}

	pagesWithText := 0
	totalTextLength := 0

	for i := 0; i < maxPagesToCheck; i++ {
		text, err := doc.Text(i)
		if err != nil {
			continue
		}
		trimmedText := strings.TrimSpace(text)
		totalTextLength += len(trimmedText)
		if len(trimmedText) > 50 {
			pagesWithText++
		}
	}

	textRatio := float64(pagesWithText) / float64(maxPagesToCheck)
	avgTextPerPage := float64(totalTextLength) / float64(maxPagesToCheck)
	isVector := textRatio >= 0.70 || avgTextPerPage > 200

	log.Printf("PDF analysis: %d/%d pages with text (%.0f%%), avg %.0f chars/page -> vector=%v",
		pagesWithText, maxPagesToCheck, textRatio*100, avgTextPerPage, isVector)

	return isVector
}

func (FitzEngine) RenderPagePNG(filePath string, pageIndex int) ([]byte, error) {
	doc, err := fitz.New(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open PDF: %w", err)
	}
	defer doc.Close()

	if pageIndex < 0 || pageIndex >= doc.NumPage() {
		return nil, fmt.Errorf("page index %d out of range", pageIndex)
	}

	img, err := doc.Image(pageIndex)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (FitzEngine) PageCount(filePath string) (int, error) {
	doc, err := fitz.New(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to open PDF: %w", err)
	}
	defer doc.Close()
	return doc.NumPage(), nil
}
