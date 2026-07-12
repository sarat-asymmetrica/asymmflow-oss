package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"
)

func main() {
	inputPath := flag.String("input", "", "Path to the markdown checklist")
	outputPath := flag.String("output", "", "Path to the generated PDF")
	flag.Parse()

	if strings.TrimSpace(*inputPath) == "" || strings.TrimSpace(*outputPath) == "" {
		fmt.Fprintln(os.Stderr, "usage: go run ./cmd/generate_deployment_checklist_pdf -input <markdown> -output <pdf>")
		os.Exit(2)
	}

	lines, err := readLines(*inputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read input: %v\n", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(filepath.Dir(*outputPath), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create output dir: %v\n", err)
		os.Exit(1)
	}

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetTitle("AsymmFlow Deployment Sign-Off Checklist", false)
	pdf.SetAuthor("AsymmFlow", false)
	pdf.SetCreator("AsymmFlow checklist generator", false)
	pdf.SetMargins(16, 18, 16)
	pdf.SetAutoPageBreak(true, 16)
	pdf.AliasNbPages("")
	pdf.SetHeaderFuncMode(func() {
		pdf.SetY(8)
		pdf.SetFont("Helvetica", "B", 12)
		pdf.SetTextColor(30, 30, 30)
		pdf.CellFormat(0, 8, "AsymmFlow Deployment Sign-Off Checklist", "", 0, "L", false, 0, "")
		pdf.SetFont("Helvetica", "", 9)
		pdf.SetTextColor(110, 110, 110)
		pdf.CellFormat(0, 8, time.Now().Format("2 January 2006"), "", 0, "R", false, 0, "")
		pdf.Ln(10)
		pdf.SetDrawColor(220, 220, 220)
		pdf.Line(16, pdf.GetY(), 194, pdf.GetY())
		pdf.Ln(4)
	}, true)
	pdf.SetFooterFunc(func() {
		pdf.SetY(-10)
		pdf.SetFont("Helvetica", "", 8)
		pdf.SetTextColor(130, 130, 130)
		pdf.CellFormat(0, 6, fmt.Sprintf("Page %d/{nb}", pdf.PageNo()), "", 0, "C", false, 0, "")
	})
	pdf.AddPage()

	for _, line := range lines {
		renderMarkdownLine(pdf, line)
	}

	if err := pdf.OutputFileAndClose(*outputPath); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write pdf: %v\n", err)
		os.Exit(1)
	}
}

func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func renderMarkdownLine(pdf *gofpdf.Fpdf, raw string) {
	line := strings.TrimRight(raw, " \t")
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		pdf.Ln(2)
		return
	}

	switch {
	case strings.HasPrefix(trimmed, "# "):
		pdf.Ln(2)
		pdf.SetFont("Helvetica", "B", 17)
		pdf.SetTextColor(20, 20, 20)
		pdf.MultiCell(0, 8, strings.TrimSpace(strings.TrimPrefix(trimmed, "# ")), "", "L", false)
		pdf.Ln(1)
	case strings.HasPrefix(trimmed, "## "):
		pdf.Ln(1)
		pdf.SetFont("Helvetica", "B", 13)
		pdf.SetTextColor(25, 25, 25)
		pdf.MultiCell(0, 7, strings.TrimSpace(strings.TrimPrefix(trimmed, "## ")), "", "L", false)
		pdf.Ln(0.5)
	case strings.HasPrefix(trimmed, "- [ ] "):
		pdf.SetFont("Helvetica", "", 10)
		pdf.SetTextColor(35, 35, 35)
		pdf.MultiCell(0, 5.6, "[] "+strings.TrimSpace(strings.TrimPrefix(trimmed, "- [ ] ")), "", "L", false)
	case strings.HasPrefix(trimmed, "- [x] "), strings.HasPrefix(trimmed, "- [X] "):
		pdf.SetFont("Helvetica", "", 10)
		pdf.SetTextColor(35, 35, 35)
		text := strings.TrimPrefix(strings.TrimPrefix(trimmed, "- [x] "), "- [X] ")
		pdf.MultiCell(0, 5.6, "[x] "+strings.TrimSpace(text), "", "L", false)
	case strings.HasPrefix(trimmed, "- "):
		pdf.SetFont("Helvetica", "", 10)
		pdf.SetTextColor(45, 45, 45)
		pdf.MultiCell(0, 5.4, "• "+strings.TrimSpace(strings.TrimPrefix(trimmed, "- ")), "", "L", false)
	default:
		pdf.SetFont("Helvetica", "", 10)
		pdf.SetTextColor(55, 55, 55)
		pdf.MultiCell(0, 5.4, trimmed, "", "L", false)
	}
}
