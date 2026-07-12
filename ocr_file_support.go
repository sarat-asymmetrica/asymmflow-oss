package main

import "strings"

var ocrSupportedExtensions = []string{
	".pdf",
	".xlsx",
	".xls",
	".docx",
	".rtf",
	".msg",
	".eml",
	".png",
	".jpg",
	".jpeg",
	".bmp",
	".tiff",
	".tif",
	".webp",
}

var ocrWatcherExtensions = []string{
	".msg",
	".xml",
	".xlsx",
	".xls",
	".pdf",
	".docx",
	".rtf",
	".eml",
	".png",
	".jpg",
	".jpeg",
	".bmp",
	".tiff",
	".tif",
	".webp",
}

func supportedOCRFileExtensions() []string {
	return append([]string(nil), ocrSupportedExtensions...)
}

func supportedOCRWatcherExtensions() []string {
	return append([]string(nil), ocrWatcherExtensions...)
}

func buildGlobPattern(exts []string) string {
	patterns := make([]string, 0, len(exts))
	for _, ext := range exts {
		patterns = append(patterns, "*"+ext)
	}
	return strings.Join(patterns, ";")
}

func supportedOCRFileDialogPattern() string {
	return buildGlobPattern(supportedOCRFileExtensions())
}

func supportedOCRImagePattern() string {
	return buildGlobPattern([]string{".png", ".jpg", ".jpeg", ".bmp", ".tiff", ".tif", ".webp"})
}

func supportedOCROfficePattern() string {
	return buildGlobPattern([]string{".docx", ".xlsx", ".xls", ".rtf"})
}

func supportedOCREmailPattern() string {
	return buildGlobPattern([]string{".msg", ".eml"})
}
